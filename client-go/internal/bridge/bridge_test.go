package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/agents"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/config"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/transport"
)

func TestEndToEndMockAgent(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("POSIX mock shell fixture")
	}
	root := t.TempDir()
	addon := filepath.Join(root, "AddOns")
	bridgeDir := filepath.Join(root, "bridge")
	for _, dir := range []string{filepath.Join(addon, "AzerMind"), bridgeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(addon, "AzerMind", "AzerMind.toc"), []byte("Test"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{AddonDir: addon, InboxFile: filepath.Join(addon, "AzerMind", "Inbox.lua"), Agent: "claude", MaxParallel: 2, Slots: 2, ActMax: 3, PresenceMax: 3, Agents: map[string]agents.Config{}}
	cli := filepath.Join(root, "agent")
	script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '{\"type\":\"result\",\"session_id\":\"session-123\",\"result\":\"All good.\\nTL;DR: done\"}'\n"
	if err := os.WriteFile(cli, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	cfg.Agents["claude"] = agents.Config{Path: cli}
	if _, _, err := transport.InstallSlots(cfg.Transport()); err != nil {
		t.Fatal(err)
	}
	c, err := New(cfg, bridgeDir, root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := "Character: Mage"
	job := protocol.Job{ID: 13, Session: "addon-session", Chat: "chat1", Text: "test prompt", ContextText: &ctx}
	c.Submit(job)
	c.Wait()
	body, err := os.ReadFile(filepath.Join(addon, "AzerMind_S001", "Inbox.lua"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`status = "done"`, `session = "session-123"`, `summary = "done"`, `text = "All good.\nTL;DR: done"`} {
		if !strings.Contains(string(body), needle) {
			t.Errorf("missing %q in %s", needle, string(body))
		}
	}
	if !c.State.SeenJob(job) {
		t.Fatal("not marked handled")
	}
	if c.State.Sessions["chat:chat1"] != "session-123" {
		t.Fatal("session not stored")
	}
	c.Submit(job)
	c.Wait()
	if len(c.State.Handled[job.Session]) != 1 {
		t.Fatal("duplicate job")
	}
	saved, err := os.ReadFile(filepath.Join(bridgeDir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err = json.Unmarshal(saved, &state); err != nil {
		t.Fatal(err)
	}
	if state["context"] == nil {
		t.Fatal("context missing")
	}
}
func TestInvalidFolderProducesError(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{AddonDir: filepath.Join(root, "AddOns"), Agent: "claude", MaxParallel: 1, Slots: 1}
	c, err := New(cfg, root, filepath.Join(root, "absent"))
	if err != nil {
		t.Fatal(err)
	}
	j := protocol.Job{ID: 1, Session: "x", Chat: "1", Text: "hi"}
	c.Submit(j)
	c.Wait()
	if !c.State.SeenJob(j) {
		t.Fatal("missing completion")
	}
	b, err := os.ReadFile(filepath.Join(cfg.AddonDir, "AzerMind", "Inbox.lua"))
	if err != nil || !strings.Contains(string(b), "Folder does not exist") {
		t.Fatalf("missing error reply: %v %s", err, b)
	}
}
