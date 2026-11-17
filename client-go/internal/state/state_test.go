package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
)

func TestLegacyHandledAndResume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	data := `{"lastId":3,"handled":{"old":3,"recent":{"4":1}},"sessions":{"s:c":"legacy-session"},"sessionAgent":{}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.SeenJob(protocol.Job{Session: "old", ID: 2}) || !s.SeenJob(protocol.Job{Session: "recent", ID: 4}) {
		t.Fatalf("bad migration: %#v", s.Handled)
	}
	j := protocol.Job{Session: "s", Chat: "c", ID: 7}
	resume, err := s.GetSession(j, "codex", "/tmp", "same")
	if err != nil || resume != "legacy-session" {
		t.Fatalf("resume %q %v", resume, err)
	}
	if err := s.Mark(j); err != nil {
		t.Fatal(err)
	}
	if !s.SeenJob(j) {
		t.Fatal("not marked")
	}
}
