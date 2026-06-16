package shared_test

import (
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

func TestActivitiesCompleteness(t *testing.T) {
	expected := map[shared.Activity]bool{
		shared.ActivityThinking:     false,
		shared.ActivityExploration:  false,
		shared.ActivityCoding:       false,
		shared.ActivityTesting:      false,
		shared.ActivityShipping:     false,
		shared.ActivityConversation: false,
	}
	for _, a := range shared.Activities {
		if _, ok := expected[a]; !ok {
			t.Errorf("Activities contains unknown activity %q", a)
		}
		expected[a] = true
	}
	for a, seen := range expected {
		if !seen {
			t.Errorf("Activities missing %q", a)
		}
	}
}

func TestSourceConstants(t *testing.T) {
	sources := []shared.Source{
		shared.SourceClaudeCode,
		shared.SourceGeminiCli,
		shared.SourceCodex,
		shared.SourceCopilot,
		shared.SourceCursor,
		shared.SourceAntigravity,
	}
	if len(sources) != 6 {
		t.Errorf("expected 6 sources, got %d", len(sources))
	}
}

func TestStoredEventZeroValue(t *testing.T) {
	var e shared.StoredEvent
	if e.IsError {
		t.Error("StoredEvent zero value should have IsError=false")
	}
	if e.Tools != nil {
		t.Error("StoredEvent zero value should have nil Tools")
	}
}
