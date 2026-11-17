package transport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
)

func TestInstallAndPublish(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "AzerMind"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AzerMind", "AzerMind.toc"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{AddonDir: dir, Slots: 2, ActMax: 3, PresenceMax: 60}
	made, kept, err := InstallSlots(cfg)
	if err != nil || made != 76 || kept != 0 {
		t.Fatalf("first install: %d %d %v", made, kept, err)
	}
	made, kept, err = InstallSlots(cfg)
	if err != nil || made != 0 || kept != 76 {
		t.Fatalf("second install: %d %d %v", made, kept, err)
	}
	p := NewPublisher(cfg, protocol.LuaOptions{Agent: "codex"})
	if err := p.Publish("s:c", protocol.Reply{Chat: "c", ID: 1, Status: "done", Text: "done"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "AzerMind", "Inbox.lua"), filepath.Join(dir, "AzerMind_S001", "Inbox.lua"), filepath.Join(dir, "AzerMind_S002", "Inbox.lua")} {
		b, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(b), `text = "done"`) {
			t.Fatalf("bad publication %q: %v %q", path, err, b)
		}
	}
	if err := p.Signal("ack", 3, true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "AzerMind", "ack", "001.wav"))
	if err != nil || len(b) != 124 {
		t.Fatalf("signal: %v %d", err, len(b))
	}
	if err := p.ResetBeats(1); err != nil {
		t.Fatal(err)
	}
	if err := p.Beat(1); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(dir, "AzerMind", "act", "001", "01.wav"))
	if err != nil || len(b) != 124 {
		t.Fatalf("beat: %v %d", err, len(b))
	}
	if err := p.PresenceBeat(); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(dir, "AzerMind", "presence", "0001.wav"))
	if err != nil || len(b) != 124 {
		t.Fatalf("presence: %v %d", err, len(b))
	}
}
