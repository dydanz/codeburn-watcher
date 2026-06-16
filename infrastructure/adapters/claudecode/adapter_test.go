package claudecode_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/claudecode"
	"github.com/dydanz/codeburn-watcher/internal/collection"
)

func writeJSONL(t *testing.T, dir string, filename string, lines []map[string]any) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, l := range lines {
		_ = enc.Encode(l)
	}
}

func TestClaudeCodeAdapter_ParsesAssistantTurns(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "myproject")
	_ = os.MkdirAll(projDir, 0755)

	writeJSONL(t, projDir, "session.jsonl", []map[string]any{
		{
			"uuid":      "evt1",
			"sessionId": "sess1",
			"message": map[string]any{
				"role":  "user",
				"usage": map[string]any{"input_tokens": 50},
			},
		},
		{
			"uuid":      "evt2",
			"sessionId": "sess1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude-sonnet-4-5",
				"content": []map[string]any{
					{"type": "tool_use", "name": "Read"},
					{"type": "text", "text": "Here is the file"},
				},
				"usage": map[string]any{
					"input_tokens":              100,
					"output_tokens":             50,
					"cache_read_input_tokens":   20,
					"cache_creation_input_tokens": 5,
				},
			},
		},
	})

	adapter := claudecode.ClaudeCodeAdapter{
		HomeDir:    home,
		Classifier: collection.ActivityClassifier{},
	}
	events, result, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Source != "claude-code" {
		t.Errorf("Source = %q", result.Source)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (assistant only), got %d", len(events))
	}
	e := events[0]
	if e.EventKey != "evt2" {
		t.Errorf("EventKey = %q, want evt2", e.EventKey)
	}
	if e.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", e.InputTokens)
	}
	if e.CacheReadTokens != 20 {
		t.Errorf("CacheReadTokens = %d, want 20", e.CacheReadTokens)
	}
	if e.Model != "claude-sonnet-4-5" {
		t.Errorf("Model = %q", e.Model)
	}
}

func TestClaudeCodeAdapter_MissingDir(t *testing.T) {
	adapter := claudecode.ClaudeCodeAdapter{
		HomeDir:    t.TempDir(),
		Classifier: collection.ActivityClassifier{},
	}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("expected no error when dir missing, got: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
