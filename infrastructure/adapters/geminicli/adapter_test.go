package geminicli_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dydanz/codeburn-watcher/infrastructure/adapters/geminicli"
	"github.com/dydanz/codeburn-watcher/internal/collection"
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, _ := json.Marshal(v)
	_ = os.WriteFile(path, data, 0644)
}

func TestGeminiCliAdapter_Shape1(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini")
	_ = os.MkdirAll(dir, 0755)

	writeJSON(t, filepath.Join(dir, "chat_2025-01.json"), []map[string]any{
		{
			"role":    "model",
			"project": "myproj",
			"metadata": map[string]any{
				"inputTokenCount":  200,
				"outputTokenCount": 80,
				"model":            "gemini-2.0-flash",
				"turn":             "turn_001",
			},
		},
	})

	adapter := geminicli.GeminiCliAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, result, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Source != "gemini-cli" {
		t.Errorf("Source = %q", result.Source)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.InputTokens != 200 {
		t.Errorf("InputTokens = %d, want 200", e.InputTokens)
	}
	if e.Model != "gemini-2.0-flash" {
		t.Errorf("Model = %q, want gemini-2.0-flash", e.Model)
	}
}

func TestGeminiCliAdapter_Shape2(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini")
	_ = os.MkdirAll(dir, 0755)

	writeJSON(t, filepath.Join(dir, "chat_2025-02.json"), []map[string]any{
		{
			"role": "model",
			"usageMetadata": map[string]any{
				"promptTokenCount":     150,
				"candidatesTokenCount": 60,
			},
		},
	})

	adapter := geminicli.GeminiCliAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].InputTokens != 150 {
		t.Errorf("InputTokens = %d, want 150", events[0].InputTokens)
	}
	if events[0].OutputTokens != 60 {
		t.Errorf("OutputTokens = %d, want 60", events[0].OutputTokens)
	}
}

func TestGeminiCliAdapter_MissingDir(t *testing.T) {
	adapter := geminicli.GeminiCliAdapter{HomeDir: t.TempDir(), Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestGeminiCliAdapter_SkipsUserRole(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini")
	_ = os.MkdirAll(dir, 0755)

	writeJSON(t, filepath.Join(dir, "chat_2025-03.json"), []map[string]any{
		{
			"role": "user",
			"metadata": map[string]any{
				"inputTokenCount":  100,
				"outputTokenCount": 50,
			},
		},
	})

	adapter := geminicli.GeminiCliAdapter{HomeDir: home, Classifier: collection.ActivityClassifier{}}
	events, _, err := adapter.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events (user role skipped), got %d", len(events))
	}
}
