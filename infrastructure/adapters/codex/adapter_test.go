package codex_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/codex"
	"github.com/dydanz/codeburn-watcher/internal/collection"
)

func writeJSONL(t *testing.T, path string, lines []map[string]any) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, l := range lines {
		_ = enc.Encode(l)
	}
}

func TestCodexAdapter_DeltaTokens(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".codex")
	_ = os.MkdirAll(dir, 0755)

	writeJSONL(t, filepath.Join(dir, "sess.jsonl"), []map[string]any{
		{
			"id": "e1", "session_id": "s1", "model": "gpt-4o",
			"project": "proj", "timestamp_ms": 1700000000000,
			"total_input_tokens": 300, "total_output_tokens": 100,
		},
		{
			"id": "e2", "session_id": "s1", "model": "gpt-4o",
			"project": "proj", "timestamp_ms": 1700000001000,
			"total_input_tokens": 500, "total_output_tokens": 150,
		},
	})

	adapter := codex.CodexAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, result, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Source != "codex" {
		t.Errorf("Source = %q", result.Source)
	}
	// e1: 300 input, 100 output (delta from 0)
	// e2: 200 input delta, 50 output delta
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].InputTokens != 300 {
		t.Errorf("event[0] InputTokens = %d, want 300", events[0].InputTokens)
	}
	if events[1].InputTokens != 200 {
		t.Errorf("event[1] InputTokens = %d, want 200 (delta)", events[1].InputTokens)
	}
}

func TestCodexAdapter_SkipsZeroDelta(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".codex")
	_ = os.MkdirAll(dir, 0755)

	writeJSONL(t, filepath.Join(dir, "zero.jsonl"), []map[string]any{
		{
			"id": "e1", "session_id": "s1",
			"total_input_tokens": 0, "total_output_tokens": 0, "timestamp_ms": 1700000000000,
		},
	})

	adapter := codex.CodexAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events (zero delta), got %d", len(events))
	}
}

func TestCodexAdapter_MissingDir(t *testing.T) {
	adapter := codex.CodexAdapter{HomeDir: t.TempDir(), Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
