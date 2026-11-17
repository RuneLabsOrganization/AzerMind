package agents

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildArguments(t *testing.T) {
	for _, id := range Names {
		inv, err := Build(Request{Agent: id, CWD: "/tmp/work", Prompt: "hello", System: "system", SystemShort: "short", Config: Config{Model: "some-model", Reasoning: "high", PermissionMode: "acceptEdits", AllowedTools: []string{"WebSearch"}}})
		if err != nil {
			t.Fatal(err)
		}
		if inv.Command == "" || len(inv.Args) == 0 {
			t.Fatalf("empty invocation %s", id)
		}
		joined := strings.Join(inv.Args, " ")
		if !strings.Contains(joined, "some-model") {
			t.Errorf("no model for %s: %s", id, joined)
		}
		if id == "codex" && !strings.Contains(joined, "model_reasoning_effort") {
			t.Fatal("codex effort flag missing")
		}
		if id == "claude" && !strings.Contains(joined, "--append-system-prompt") {
			t.Fatal("claude system flag missing")
		}
		if id == "grok" && !strings.Contains(joined, "--prompt-file") {
			t.Fatal("grok file flag missing")
		}
		if id == "hermes" && !inv.StreamText {
			t.Fatal("hermes must stream text")
		}
	}
	if _, err := Build(Request{Agent: "unknown"}); err == nil {
		t.Fatal("unexpected unknown agent")
	}
}
func TestResumeAndPermissions(t *testing.T) {
	inv, err := Build(Request{Agent: "codex", CWD: "/tmp", Prompt: "hello", System: "FIRST", SystemShort: "RESUMED", Resume: "sess", Config: Config{PermissionMode: "default"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(inv.Stdin, "RESUMED") || strings.Contains(inv.Stdin, "FIRST") || !strings.Contains(strings.Join(inv.Args, " "), "resume sess") {
		t.Fatalf("resume flags %v", inv)
	}
	h, err := Build(Request{Agent: "hermes", CWD: "/tmp", Prompt: "hi", Config: Config{ExtraArgs: []string{"--yolo", "--foo"}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(h.Args, " "), "yolo") || !strings.Contains(strings.Join(h.Args, " "), "--foo") {
		t.Fatal(h.Args)
	}
	g, _ := Build(Request{Agent: "grok", CWD: "/tmp", Prompt: "hi", Config: Config{AllowedTools: []string{"Bash(git:*)"}, DeniedTools: []string{"WebFetch"}}})
	if !strings.Contains(strings.Join(g.Args, " "), "Bash(git *)") || !strings.Contains(strings.Join(g.Args, " "), "--deny WebFetch") {
		t.Fatal(g.Args)
	}
}
func TestAgentParsers(t *testing.T) {
	cases := []struct {
		name    string
		events  []string
		want    string
		session string
	}{
		{"claude", []string{`{"type":"assistant","message":{"content":[{"type":"text","text":"working"}]} ,"session_id":"sid"}`, `{"type":"result","result":"final","session_id":"sid"}`}, "final", "sid"},
		{"codex", []string{`{"type":"thread.started","thread_id":"thread"}`, `{"type":"item.completed","item":{"type":"agent_message","text":"codex final"}}`, `{"type":"turn.completed"}`}, "codex final", "thread"},
		{"grok", []string{`{"type":"text","data":"grok "}`, `{"type":"text","data":"final"}`, `{"type":"end","sessionId":"grok-id"}`}, "grok final", "grok-id"},
		{"agy", []string{`{"event":"init","conversation_id":"agy-id"}`, `{"event":"step_update","step_update":{"step_type":"agent_response","text_delta":"agy "}}`, `{"event":"result","result":{"status":"SUCCESS","response":"agy final"}}`}, "agy final", "agy-id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser(tc.name)
			var got Result
			var session string
			for _, line := range tc.events {
				e, err := p.Feed([]byte(line))
				if err != nil {
					t.Fatal(err)
				}
				if e.Session != "" {
					session = e.Session
				}
				if e.Done != nil {
					got = *e.Done
				}
			}
			if got.Text != tc.want || session != tc.session {
				t.Fatalf("got: %v, session %s", got, session)
			}
		})
	}
	h := NewParser("hermes")
	e := h.Finish("reply\n", "session_id: hermes-123", 0)
	if e.Session != "hermes-123" || e.Done.Text != "reply" {
		t.Fatal(e)
	}
}
func TestRunMockClaude(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("shell fixture is POSIX-only")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "mock-claude")
	payload := "#!/bin/sh\ncat >/dev/null\necho '{\"type\":\"assistant\",\"session_id\":\"abc\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"working\"}]}}'\necho '{\"type\":\"result\",\"session_id\":\"abc\",\"result\":\"TL;DR: pass\"}'\n"
	if err := os.WriteFile(script, []byte(payload), 0755); err != nil {
		t.Fatal(err)
	}
	var progress []string
	result, err := Run(context.Background(), Request{Agent: "claude", CWD: dir, Prompt: "hello", Config: Config{Path: script}, Timeout: 3 * time.Second}, func(s string) { progress = append(progress, s) })
	if err != nil || result.Text != "TL;DR: pass" || result.Session != "abc" || !reflect.DeepEqual(progress, []string{"working"}) {
		t.Fatalf("result=%#v progress=%#v err=%v", result, progress, err)
	}
}
