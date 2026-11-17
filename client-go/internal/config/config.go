package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/agents"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/transport"
)

type Capture struct {
	Enabled        *bool  `json:"enabled"`
	ProcessName    string `json:"processName"`
	CellPx         int    `json:"cellPx"`
	CellsPerRow    int    `json:"cellsPerRow"`
	MaxRows        int    `json:"maxRows"`
	IntervalMs     int    `json:"intervalMs"`
	Python         string `json:"python"`
	WindowName     string `json:"windowName"`
	KeepComposited bool   `json:"keepComposited"`
}
type Config struct {
	AddonDir           string                   `json:"addonDir"`
	InboxFile          string                   `json:"inboxFile"`
	SavedVariablesFile string                   `json:"savedVariablesFile"`
	DefaultCWD         string                   `json:"defaultCwd"`
	TocInterface       string                   `json:"tocInterface"`
	Slots              int                      `json:"slots"`
	ActMax             int                      `json:"actMax"`
	PresenceMax        int                      `json:"presenceMax"`
	PresenceIntervalMs int                      `json:"presenceIntervalMs"`
	MaxParallel        int                      `json:"maxParallel"`
	Capture            Capture                  `json:"capture"`
	Agent              string                   `json:"agent"`
	Agents             map[string]agents.Config `json:"agents"`
	GameContext        *bool                    `json:"gameContext"`
	PrimerFile         string                   `json:"primerFile"`
	PollMs             int                      `json:"pollMs"`
	ProgressWriteMs    int                      `json:"progressWriteMs"`
	TimeoutMs          int                      `json:"timeoutMs"`
}

func Load(file string) (Config, error) {
	var c Config
	b, err := os.ReadFile(file)
	if err != nil {
		return c, fmt.Errorf("read %s: %w", file, err)
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.AddonDir == "" {
		return c, fmt.Errorf("addonDir is required in %s", file)
	}
	if c.Slots <= 0 {
		c.Slots = 200
	}
	if c.ActMax <= 0 {
		c.ActMax = 60
	}
	if c.PresenceMax <= 0 {
		c.PresenceMax = 2000
	}
	if c.PresenceIntervalMs <= 0 {
		c.PresenceIntervalMs = 30000
	}
	if c.MaxParallel <= 0 {
		c.MaxParallel = 3
	}
	if c.PollMs <= 0 {
		c.PollMs = 750
	}
	if c.ProgressWriteMs <= 0 {
		c.ProgressWriteMs = 3000
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = 1800000
	}
	if c.Agent == "" {
		c.Agent = "claude"
	}
	c.Agent = strings.ToLower(c.Agent)
	if !agents.IsAgent(c.Agent) {
		return c, fmt.Errorf("unsupported agent %q", c.Agent)
	}
	if c.Capture.ProcessName == "" {
		c.Capture.ProcessName = "Wow"
	}
	if c.Capture.CellPx <= 0 {
		c.Capture.CellPx = 4
	}
	if c.Capture.CellsPerRow <= 0 {
		c.Capture.CellsPerRow = 200
	}
	if c.Capture.MaxRows <= 0 {
		c.Capture.MaxRows = 96
	}
	if c.Capture.IntervalMs <= 0 {
		c.Capture.IntervalMs = 250
	}
	if c.Capture.Python == "" {
		c.Capture.Python = "python3"
	}
	if c.Agents == nil {
		c.Agents = map[string]agents.Config{}
	}
	if c.InboxFile == "" {
		c.InboxFile = filepath.Join(c.AddonDir, "AzerMind", "Inbox.lua")
	}
	if c.Capture.Enabled == nil {
		yes := true
		c.Capture.Enabled = &yes
	}
	return c, nil
}
func (c Config) Transport() transport.Config {
	return transport.Config{AddonDir: c.AddonDir, InboxFile: c.InboxFile, Slots: c.Slots, ActMax: c.ActMax, PresenceMax: c.PresenceMax, Interface: c.TocInterface}
}
