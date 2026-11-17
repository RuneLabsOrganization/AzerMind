// Package transport writes the addon files used for no-reload replies and signals.
package transport

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
)

// Config mirrors the Node bridge's slots, actMax and presenceMax settings.
type Config struct {
	AddonDir    string
	InboxFile   string
	Slots       int
	ActMax      int
	PresenceMax int
	Interface   string
}

func (c Config) defaults() Config {
	if c.Slots <= 0 {
		c.Slots = 200
	}
	if c.ActMax <= 0 {
		c.ActMax = 60
	}
	if c.PresenceMax <= 0 {
		c.PresenceMax = 2000
	}
	if c.Interface == "" {
		c.Interface = "16001"
	}
	if c.InboxFile == "" {
		c.InboxFile = filepath.Join(c.AddonDir, "AzerMind", "Inbox.lua")
	}
	return c
}

func AtomicWrite(file string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(file), ".azermind-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}
func ensure(file string, data []byte) (bool, error) {
	if _, err := os.Stat(file); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err == nil, err
}

// InstallSlots is idempotent: it never replaces player files or existing slots.
func InstallSlots(in Config) (made, kept int, err error) {
	c := in.defaults()
	if _, e := os.Stat(filepath.Join(c.AddonDir, "AzerMind", "AzerMind.toc")); e != nil {
		return 0, 0, fmt.Errorf("AzerMind addon not found: %w", e)
	}
	add := func(path string, content []byte) error {
		created, e := ensure(path, content)
		if e != nil {
			return e
		}
		if created {
			made++
		} else {
			kept++
		}
		return nil
	}
	for i := 1; i <= c.Slots; i++ {
		num := protocol.Pad3(i)
		name := "AzerMind_S" + num
		dir := filepath.Join(c.AddonDir, name)
		toc := fmt.Sprintf("## Interface: %s\n## Title: AzerMind slot %s\n## Notes: Reply slot for AzerMind. Load-on-demand; leave it enabled.\n## LoadOnDemand: 1\n## Dependencies: AzerMind\n\nInbox.lua\n", c.Interface, num)
		if e := add(filepath.Join(dir, name+".toc"), []byte(toc)); e != nil {
			return made, kept, e
		}
		if e := add(filepath.Join(dir, "Inbox.lua"), []byte("AzerMind_SlotData = nil\n")); e != nil {
			return made, kept, e
		}
		for _, kind := range []string{"sig", "ack"} {
			if e := add(filepath.Join(c.AddonDir, "AzerMind", kind, num+".wav"), nil); e != nil {
				return made, kept, e
			}
		}
		for k := 1; k <= c.ActMax; k++ {
			if e := add(filepath.Join(c.AddonDir, "AzerMind", "act", num, fmt.Sprintf("%02d.wav", k)), nil); e != nil {
				return made, kept, e
			}
		}
	}
	for i := 1; i <= c.PresenceMax; i++ {
		if e := add(filepath.Join(c.AddonDir, "AzerMind", "presence", fmt.Sprintf("%04d.wav", i)), nil); e != nil {
			return made, kept, e
		}
	}
	if e := add(filepath.Join(c.AddonDir, "AzerMind", "ctl", "empty.wav"), nil); e != nil {
		return made, kept, e
	}
	if e := add(filepath.Join(c.AddonDir, "AzerMind", "ctl", "valid.wav"), protocol.SilentWAV()); e != nil {
		return made, kept, e
	}
	return made, kept, nil
}

type Publisher struct {
	Config   Config
	Mu       sync.Mutex
	Live     map[string]protocol.Reply
	Order    []string
	Presence int
	Beats    map[int]int
	Opts     protocol.LuaOptions
}

func NewPublisher(c Config, opt protocol.LuaOptions) *Publisher {
	return &Publisher{Config: c.defaults(), Live: make(map[string]protocol.Reply), Beats: make(map[int]int), Opts: opt}
}
func (p *Publisher) Publish(key string, reply protocol.Reply) error {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	if _, exists := p.Live[key]; !exists {
		p.Order = append(p.Order, key)
	}
	p.Live[key] = reply
	return p.publishLocked()
}
func (p *Publisher) PublishNow() error { p.Mu.Lock(); defer p.Mu.Unlock(); return p.publishLocked() }
func (p *Publisher) publishLocked() error {
	records := make([]protocol.Reply, 0, len(p.Order))
	for _, key := range p.Order {
		records = append(records, p.Live[key])
	}
	if len(records) > 30 {
		records = records[len(records)-30:]
	}
	// A single snapshot is copied to every slot: a game reload can consume any unused slot.
	if err := AtomicWrite(p.Config.InboxFile, []byte(protocol.LuaTable("AzerMind_Inbox", records, p.Opts))); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(p.Config.AddonDir, "AzerMind_S001", "Inbox.lua")); err != nil {
		return nil
	}
	body := []byte(protocol.LuaTable("AzerMind_SlotData", records, p.Opts))
	for i := 1; i <= p.Config.Slots; i++ {
		if err := AtomicWrite(filepath.Join(p.Config.AddonDir, "AzerMind_S"+protocol.Pad3(i), "Inbox.lua"), body); err != nil {
			return err
		}
	}
	return nil
}
func (p *Publisher) Signal(kind string, id int, on bool) error {
	if kind != "sig" && kind != "ack" {
		return fmt.Errorf("invalid signal %q", kind)
	}
	file := filepath.Join(p.Config.AddonDir, "AzerMind", kind, protocol.Pad3(protocol.SlotNumber(id, p.Config.Slots))+".wav")
	if on {
		return AtomicWrite(file, protocol.SilentWAV())
	}
	return AtomicWrite(file, nil)
}
func (p *Publisher) ResetBeats(id int) error {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	p.Beats[id] = 0
	for i := 1; i <= p.Config.ActMax; i++ {
		if err := AtomicWrite(filepath.Join(p.Config.AddonDir, "AzerMind", "act", protocol.Pad3(protocol.SlotNumber(id, p.Config.Slots)), fmt.Sprintf("%02d.wav", i)), nil); err != nil {
			return err
		}
	}
	return nil
}
func (p *Publisher) Beat(id int) error {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	p.Beats[id]++
	if p.Beats[id] > p.Config.ActMax {
		return nil
	}
	return AtomicWrite(filepath.Join(p.Config.AddonDir, "AzerMind", "act", protocol.Pad3(protocol.SlotNumber(id, p.Config.Slots)), fmt.Sprintf("%02d.wav", p.Beats[id])), protocol.SilentWAV())
}
func (p *Publisher) PresenceBeat() error {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	root := filepath.Join(p.Config.AddonDir, "AzerMind", "presence")
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	p.Presence = p.Presence%p.Config.PresenceMax + 1
	if err := AtomicWrite(filepath.Join(root, fmt.Sprintf("%04d.wav", p.Presence)), protocol.SilentWAV()); err != nil {
		return err
	}
	for i := 1; i <= 50; i++ {
		next := (p.Presence-1+i)%p.Config.PresenceMax + 1
		if next == p.Presence {
			continue
		}
		if err := AtomicWrite(filepath.Join(root, fmt.Sprintf("%04d.wav", next)), nil); err != nil {
			return err
		}
	}
	return nil
}
func (p *Publisher) RunPresence(stop <-chan struct{}, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	_ = p.PresenceBeat()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			_ = p.PresenceBeat()
		}
	}
}
