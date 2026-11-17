package models

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/agents"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/config"
)

func TestParseCatalogAndValidate(t *testing.T) {
	rows := Parse([]byte(`{"models":[{"model":"grok-4.7","name":"Grok"},{"model":"bad model"},{"model":"grok-4.7"}]}`))
	if len(rows) != 1 || rows[0].Model != "grok-4.7" {
		t.Fatal(rows)
	}
	c := config.Config{Agents: map[string]agents.Config{"claude": {Model: "custom-provider"}, "codex": {Model: "gpt-custom"}}}
	settings, _ := Snapshot(c)
	if len(settings["claude"].Models) == 0 || settings["claude"].Models[0].Model != "custom-provider" {
		t.Fatal(settings["claude"])
	}
	if err := Validate("codex", "gpt-custom", "high", settings); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		Agent  string
		Model  string
		Effort string
	}{{"unknown", "a", ""}, {"claude", "bad model", ""}, {"claude", "good", "ultra"}} {
		if err := Validate(tc.Agent, tc.Model, tc.Effort, settings); err == nil {
			t.Fatalf("accepted bad settings %#v", tc)
		}
	}
}
func TestCodexLocalCatalog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	b := []byte(`{"models":[{"slug":"gpt-6","visibility":"list","supported_reasoning_levels":[{"effort":"medium"},{"effort":"high"}]},{"slug":"codex-auto-review"},{"slug":"hidden","visibility":"hide"}]}`)
	if err := os.WriteFile(filepath.Join(dir, "models_cache.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	rows := CodexCache()
	if len(rows) != 1 || rows[0].Model != "gpt-6" || len(rows[0].Efforts) != 2 {
		t.Fatal(rows)
	}
}
