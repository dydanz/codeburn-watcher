package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// codexEvent is one line from a Codex log file.
// Codex uses cumulative counters; we delta-diff between consecutive events.
type codexEvent struct {
	ID           string `json:"id"`
	Model        string `json:"model"`
	Project      string `json:"project"`
	SessionID    string `json:"session_id"`
	Timestamp    int64  `json:"timestamp_ms"`
	TotalInput   int    `json:"total_input_tokens"`
	TotalOutput  int    `json:"total_output_tokens"`
	Tools        []string `json:"tools"`
	IsError      bool   `json:"is_error"`
}

// CodexAdapter reads cumulative-counter JSONL logs from ~/.codex/.
type CodexAdapter struct {
	HomeDir    string
	Classifier collection.ActivityClassifier
}

func (a CodexAdapter) logDir() string {
	home := a.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".codex")
}

// Collect walks ~/.codex/**/*.jsonl and delta-diffs cumulative counters.
func (a CodexAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	result := collection.CollectResult{Source: shared.SourceCodex}
	dir := a.logDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, result, nil
	}

	// track last cumulative value per session
	lastInput := make(map[string]int)
	lastOutput := make(map[string]int)
	var events []collection.UsageEvent

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var e codexEvent
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				continue
			}
			result.Found++
			inputDelta := e.TotalInput - lastInput[e.SessionID]
			outputDelta := e.TotalOutput - lastOutput[e.SessionID]
			if inputDelta < 0 {
				inputDelta = e.TotalInput
			}
			if outputDelta < 0 {
				outputDelta = e.TotalOutput
			}
			lastInput[e.SessionID] = e.TotalInput
			lastOutput[e.SessionID] = e.TotalOutput
			if inputDelta+outputDelta == 0 {
				continue
			}
			ts := time.UnixMilli(e.Timestamp).UTC().Format(time.RFC3339)
			events = append(events, collection.UsageEvent{
				Source:       shared.SourceCodex,
				SessionID:    e.SessionID,
				EventKey:     e.ID,
				Project:      e.Project,
				Model:        e.Model,
				Tools:        e.Tools,
				Activity:     a.Classifier.Classify(e.Tools, e.Model),
				TS:           ts,
				InputTokens:  inputDelta,
				OutputTokens: outputDelta,
				IsError:      e.IsError,
			})
		}
		return nil
	})

	result.Inserted = len(events)
	return events, result, nil
}

func NewCodexAdapter() *CodexAdapter {
	return &CodexAdapter{Classifier: collection.ActivityClassifier{}}
}

var _ collection.AgentAdapter = (*CodexAdapter)(nil)
var _ = fmt.Sprintf // keep fmt
