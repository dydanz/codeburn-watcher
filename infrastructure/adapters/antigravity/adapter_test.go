package antigravity_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/antigravity"
	"github.com/dydanz/codeburn-watcher/internal/collection"
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, _ := json.Marshal(v)
	_ = os.WriteFile(path, data, 0644)
}

func TestAntigravityAdapter_Basic(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".antigravity")
	_ = os.MkdirAll(dir, 0755)

	writeJSON(t, filepath.Join(dir, "run1.json"), map[string]any{
		"session_id": "sess1",
		"project":    "myproj",
		"model":      "gpt-4o",
		"steps": []map[string]any{
			{
				"id": "step1", "type": "gen",
				"timestamp_ms": 1700000000000,
				"input_tokens": 400, "output_tokens": 120,
			},
			{
				"id": "step2", "type": "step",
				"timestamp_ms": 1700000001000,
				"input_tokens": 0, "output_tokens": 0, // zero → skip
			},
		},
	})

	adapter := antigravity.AntigravityAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, result, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Source != "antigravity" {
		t.Errorf("Source = %q", result.Source)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (step2 zero-skipped), got %d", len(events))
	}
	if events[0].InputTokens != 400 {
		t.Errorf("InputTokens = %d, want 400", events[0].InputTokens)
	}
	if events[0].Project != "myproj" {
		t.Errorf("Project = %q, want myproj", events[0].Project)
	}
}

func TestAntigravityAdapter_Deduplication(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".antigravity")
	_ = os.MkdirAll(dir, 0755)

	step := map[string]any{
		"id": "dup1", "type": "gen",
		"timestamp_ms": 1700000000000,
		"input_tokens": 100, "output_tokens": 50,
	}
	writeJSON(t, filepath.Join(dir, "run_a.json"), map[string]any{
		"session_id": "s1", "steps": []map[string]any{step},
	})
	writeJSON(t, filepath.Join(dir, "run_b.json"), map[string]any{
		"session_id": "s1", "steps": []map[string]any{step}, // same step ID
	})

	adapter := antigravity.AntigravityAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event (deduped), got %d", len(events))
	}
}

func TestAntigravityAdapter_MissingDir(t *testing.T) {
	adapter := antigravity.AntigravityAdapter{HomeDir: t.TempDir(), Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
