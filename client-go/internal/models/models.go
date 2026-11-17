// Package models exposes compatible model choices and local CLI/account catalogs.
package models

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/agents"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/config"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
)

var modelID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_./:@+\[\]-]{0,199}$`)
var efforts = map[string][]string{
	"codex":  {"none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra"},
	"claude": {"low", "medium", "high", "xhigh", "max"},
	"grok":   {"none", "minimal", "low", "medium", "high", "xhigh", "max"},
	"agy":    {"low", "medium", "high", "xhigh", "max"},
	"hermes": {"none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra"},
}

func list(name string, values ...string) []protocol.Model {
	rows := make([]protocol.Model, 0, len(values))
	for _, v := range values {
		rows = append(rows, protocol.Model{Model: v, Name: v})
	}
	return rows
}
func compatible(id string) []protocol.Model {
	switch id {
	case "claude":
		return list(id, "opus", "sonnet", "haiku", "claude-opus-5-5", "claude-sonnet-5-5", "claude-haiku-5-5", "claude-opus-4-8", "claude-sonnet-4-6")
	case "grok":
		return list(id, "grok-build", "grok-4.7")
	case "agy":
		return list(id, "gemini-4-argon-high", "gemini-4-argon-medium", "gemini-3.8-flash-high", "gemini-3.8-flash-medium")
	case "hermes":
		return list(id, "opus", "sonnet", "kimi")
	}
	return nil
}
func ValidID(s string) bool { return modelID.MatchString(s) }
func ValidEffort(id, effort string) bool {
	if effort == "" {
		return true
	}
	for _, e := range efforts[id] {
		if e == effort {
			return true
		}
	}
	return false
}
func Parse(output []byte) []protocol.Model {
	var rows []protocol.Model
	var data struct {
		Models []protocol.Model `json:"models"`
		Data   []protocol.Model `json:"data"`
	}
	if json.Unmarshal(output, &rows) != nil {
		if json.Unmarshal(output, &data) == nil {
			rows = data.Models
			if len(rows) == 0 {
				rows = data.Data
			}
		}
	}
	seen := map[string]bool{}
	result := make([]protocol.Model, 0, len(rows))
	for _, r := range rows {
		if ValidID(r.Model) && !seen[r.Model] {
			seen[r.Model] = true
			result = append(result, r)
			if len(result) >= 80 {
				break
			}
		}
	}
	return result
}
func CodexCache() []protocol.Model {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		h, _ := os.UserHomeDir()
		home = filepath.Join(h, ".codex")
	}
	b, err := os.ReadFile(filepath.Join(home, "models_cache.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Models []struct {
			Slug       string `json:"slug"`
			Visibility string `json:"visibility"`
			Levels     []struct {
				Effort string `json:"effort"`
			} `json:"supported_reasoning_levels"`
		} `json:"models"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return nil
	}
	result := []protocol.Model{}
	for _, m := range doc.Models {
		if !ValidID(m.Slug) || m.Visibility == "hide" || m.Slug == "codex-auto-review" {
			continue
		}
		r := protocol.Model{Model: m.Slug, Name: m.Slug, Efforts: []string{}}
		for _, x := range m.Levels {
			if ValidEffort("codex", x.Effort) {
				r.Efforts = append(r.Efforts, x.Effort)
			}
		}
		result = append(result, r)
		if len(result) >= 40 {
			break
		}
	}
	return result
}
func Snapshot(c config.Config) (map[string]protocol.AgentSetting, []protocol.Model) {
	codex := CodexCache()
	result := map[string]protocol.AgentSetting{}
	for _, id := range agents.Names {
		ac := c.Agents[id]
		models := append([]protocol.Model(nil), compatible(id)...)
		if id == "codex" {
			models = codex
		}
		names := map[string]bool{}
		full := []protocol.Model{}
		if ac.Model != "" && ValidID(ac.Model) {
			full = append(full, protocol.Model{Model: ac.Model, Name: ac.Model})
			names[ac.Model] = true
		}
		for _, m := range models {
			if !names[m.Model] {
				full = append(full, m)
				names[m.Model] = true
			}
		}
		for _, m := range ac.Models {
			if ValidID(m.Model) && !names[m.Model] {
				full = append(full, m)
				names[m.Model] = true
			}
		}
		status := "compatibility"
		if id == "codex" && len(codex) > 0 {
			status = "local"
		}
		result[id] = protocol.AgentSetting{Model: ac.Model, Reasoning: ac.Reasoning, Efforts: efforts[id], Models: full, CatalogStatus: status, CatalogSource: "Compatibility choices (account access not verified)"}
	}
	return result, codex
}
func Refresh(ctx context.Context, c config.Config) (map[string]protocol.AgentSetting, []protocol.Model) {
	settings, codex := Snapshot(c)
	for _, id := range []string{"grok", "agy"} {
		ac := c.Agents[id]
		cmd := ac.Path
		if cmd == "" {
			cmd = id
		}
		if _, err := exec.LookPath(cmd); err != nil {
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
		out, err := exec.CommandContext(probeCtx, cmd, "models").Output()
		cancel()
		if err != nil {
			continue
		}
		models := Parse(out)
		if len(models) == 0 {
			continue
		}
		s := settings[id]
		s.Models = models
		s.CatalogStatus = "discovered"
		s.CatalogSource = "Installed provider catalog"
		settings[id] = s
	}
	return settings, codex
}
func Validate(id, model, reasoning string, settings map[string]protocol.AgentSetting) error {
	if !agents.IsAgent(id) {
		return &ValidationError{"Invalid agent."}
	}
	if model != "" && !ValidID(model) {
		return &ValidationError{"Invalid model identifier."}
	}
	if !ValidEffort(id, reasoning) {
		return &ValidationError{"Unsupported reasoning effort."}
	}
	if reasoning != "" {
		s := settings[id]
		for _, m := range s.Models {
			if m.Model == model && m.EffortsKnown != nil && *m.EffortsKnown {
				found := false
				for _, e := range m.Efforts {
					if e == reasoning {
						found = true
					}
				}
				if !found {
					return &ValidationError{"Selected model does not support this effort."}
				}
			}
		}
	}
	return nil
}

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return strings.TrimSpace(e.Message) }
