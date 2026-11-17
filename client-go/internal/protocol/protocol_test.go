package protocol

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestFlagsAndStrip(t *testing.T) {
	a := strings.Join([]string{"s", "chat1", "13", "C:/work", "n;c;agent=CoDeX;allow=WebSearch,Bash(git:*);model=gpt-6;reasoning=high", "Chat", "Character: Warrior", "hello" + FieldSep + "world"}, FieldSep)
	jobs := JobsFromStrip(13, a)
	if len(jobs) != 1 {
		t.Fatal(jobs)
	}
	j := jobs[0]
	if j.Text != "hello"+FieldSep+"world" || j.ContextText == nil || *j.ContextText != "Character: Warrior" || !j.NewSession || j.Agent != "codex" || len(j.Allow) != 2 || j.Model != "gpt-6" || j.Reasoning != "high" {
		t.Fatalf("bad parsed job: %#v", j)
	}
	old := JobsFromStrip(9, strings.Join([]string{"s", "c", "4", "here", "n", "message"}, FieldSep)+RecordSep+strings.Join([]string{"s", "here", "", "legacy"}, FieldSep))
	if len(old) != 2 || old[0].ID != 4 || old[1].ID != 9 || old[1].Text != "legacy" {
		t.Fatalf("legacy: %#v", old)
	}
	short := JobsFromStrip(1, strings.Join([]string{"s", "c", "2", "", "c", "X", "text only"}, FieldSep))
	if short[0].ContextText != nil {
		t.Fatal("context falsely parsed")
	}
}
func makeFrames(text, session string, id int) []string {
	rec := strings.Join([]string{session, "chat1", fmt.Sprint(id), "C:/work", "c;agent=codex", "Items", "context", text}, FieldSep)
	raw := []byte(rec)
	var parts []string
	for i := 0; i < len(raw); i += 1100 {
		end := i + 1100
		if end > len(raw) {
			end = len(raw)
		}
		parts = append(parts, hex.EncodeToString(raw[i:end]))
	}
	out := make([]string, len(parts))
	for i, v := range parts {
		out[i] = strings.Join([]string{"F1", session, fmt.Sprint(id), fmt.Sprint(i + 1), fmt.Sprint(len(parts)), v}, FieldSep)
	}
	return out
}
func TestAssemblerMultipart(t *testing.T) {
	text := strings.Repeat("ação 🧙 ", 3000)
	parts := makeFrames(text, "session1", 8)
	a := NewAssembler()
	for i := len(parts) - 1; i >= 1; i-- {
		if _, ok := a.Add(parts[i]); ok {
			t.Fatalf("fragment delivered too early: %d", i)
		}
		if _, ok := a.Add(parts[i]); ok {
			t.Fatalf("duplicate delivered: %d", i)
		}
	}
	payload, ok := a.Add(parts[0])
	if !ok {
		t.Fatal("missing assembled payload")
	}
	jobs := JobsFromStrip(8, payload)
	if len(jobs) != 1 || jobs[0].Text != text || *jobs[0].ContextText != "context" {
		t.Fatalf("bad reconstruction")
	}
	if p, ok := a.Add("legacy"); !ok || p != "legacy" {
		t.Fatal("legacy stripped")
	}
}
func TestAssemblerLimitsAndIdentity(t *testing.T) {
	a := NewAssembler()
	for _, s := range []string{"F1\x1fs\x1f1\x1f0\x1f2\x1faa", "F1\x1fs\x1f1\x1f1\x1f10000\x1faa", "F1\x1fs\x1f1\x1f1\x1f2\x1fzz"} {
		if _, ok := a.Add(s); ok {
			t.Fatal("accepted invalid part:", s)
		}
	}
	parts := makeFrames(strings.Repeat("a", 7000), "s", 8)
	base := time.Unix(1, 0)
	if _, ok := a.AddAt(parts[0], base); ok {
		t.Fatal("incomplete")
	}
	for _, p := range parts[1:] {
		if _, ok := a.AddAt(p, base.Add(121*time.Second)); ok {
			t.Fatal("expired fragment accepted")
		}
	}
	if _, ok := a.AddAt(parts[0], base.Add(121*time.Second)); !ok {
		t.Fatal("new complete message missing")
	}
	forged := makeFrames(strings.Repeat("a", 7000), "s", 9)
	for _, p := range forged {
		p = strings.Replace(p, "F1\x1fs\x1f", "F1\x1fforged\x1f", 1)
		if _, ok := a.Add(p); ok {
			t.Fatal("accepted forged identity")
		}
	}
}
func TestParseOutbox(t *testing.T) {
	hexs := func(s string) string { return hex.EncodeToString([]byte(s)) }
	src := `AzerMindDB = { ["outbox"] = { ["id"] = 7, ["session"] = "abc", ["chat"] = "c1", ["text"] = "` + hexs("héllo") + `", ["cwd"] = "` + hexs("realms") + `", ["ctx"] = "` + hexs("Character: Test") + `", ["allow"] = "` + hexs("WebSearch\x1fBash(git:*)") + `", ["agent"] = "CoDeX", ["newSession"] = true, } }`
	j, ok := ParseOutbox(src)
	if !ok || j.ID != 7 || j.Text != "héllo" || j.CWD != "realms" || j.Agent != "codex" || !j.NewSession || *j.ContextText != "Character: Test" || len(j.Allow) != 2 {
		t.Fatalf("bad outbox: %#v %v", j, ok)
	}
	if _, ok := ParseOutbox(`AzerMindDB = {}`); ok {
		t.Fatal("accepted empty outbox")
	}
}
func TestLuaAndContext(t *testing.T) {
	s := LuaStr("a\"b\\c\nd\re\x01")
	if s != `"a\"b\\c\nde\001"` {
		t.Fatalf("lua escape: %q", s)
	}
	t0 := time.Date(2026, 11, 16, 12, 4, 1, 123000000, time.UTC)
	body := LuaTable("AzerMind_SlotData", []Reply{{Chat: "c", ID: 7, Status: "done", Text: "hello", Summary: "short", Denied: []string{"WebSearch"}}}, LuaOptions{Now: t0, CWD: "C:/work", Agent: "codex", Agents: []string{"codex", "claude"}})
	for _, needle := range []string{"AzerMind_SlotData = {", "now = 1794830641", "ts = \"2026-11-16T12:04:01.123Z\"", "summary = \"short\"", "denied = { \"WebSearch\" }", "agentSettings = {", "aiDefaults = {"} {
		if !strings.Contains(body, needle) {
			t.Errorf("missing %q in:\n%s", needle, body)
		}
	}
	original := "Character: Hero\nQuest log: 1234 active\nSpellbook: " + strings.Repeat("á🧙", 4000)
	got := NormalizeGameContext(original)
	if !utf8.ValidString(got) || len([]byte(got)) > ContextLimit || !strings.HasSuffix(got, ContextPartial) || !strings.Contains(got, "Quest log: 1234 active") {
		t.Fatalf("bad context: %q", got[:min(250, len(got))])
	}
	if NormalizeGameContext("A\r\nB") != "A\nB" {
		t.Fatal("bad CRLF")
	}
	wav := SilentWAV()
	if len(wav) != 124 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatal("invalid WAV")
	}
}
