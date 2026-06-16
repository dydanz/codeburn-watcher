package antigravity

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

// STEP_TOOLS maps antigravity step types to tool names for activity classification.
var stepTools = map[string]string{
	"gen":    "Write",
	"step":   "Bash",
	"read":   "Read",
	"search": "WebSearch",
}

// antigravityLog is one step log file.
type antigravityLog struct {
	SessionID string `json:"session_id"`
	Project   string `json:"project"`
	Model     string `json:"model"`
	Steps     []struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		TimestampMs  int64  `json:"timestamp_ms"`
		InputTokens  int    `json:"input_tokens"`
		OutputTokens int    `json:"output_tokens"`
		IsError      bool   `json:"is_error"`
	} `json:"steps"`
}

// tsToIso converts a unix-millisecond timestamp to ISO8601 UTC.
func tsToIso(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

// AntigravityAdapter reads step-attribution logs from ~/.antigravity/.
type AntigravityAdapter struct {
	HomeDir    string
	Classifier collection.ActivityClassifier
}

func (a AntigravityAdapter) logDir() string {
	home := a.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".antigravity")
}

// Collect reads step JSON logs, attributing gen/step types to appropriate tools.
func (a AntigravityAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	result := collection.CollectResult{Source: shared.SourceAntigravity}
	dir := a.logDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, result, nil
	}

	var events []collection.UsageEvent
	seen := make(map[string]bool)

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var log antigravityLog
		if err := json.Unmarshal(data, &log); err != nil {
			return nil
		}
		for _, step := range log.Steps {
			result.Found++
			if seen[step.ID] {
				continue
			}
			if step.InputTokens+step.OutputTokens == 0 {
				continue
			}
			seen[step.ID] = true
			tools := []string{}
			if t, ok := stepTools[step.Type]; ok {
				tools = append(tools, t)
			}
			events = append(events, collection.UsageEvent{
				Source:       shared.SourceAntigravity,
				SessionID:    log.SessionID,
				EventKey:     step.ID,
				Project:      log.Project,
				Model:        log.Model,
				Tools:        tools,
				Activity:     a.Classifier.Classify(tools, log.Model),
				TS:           tsToIso(step.TimestampMs),
				InputTokens:  step.InputTokens,
				OutputTokens: step.OutputTokens,
				IsError:      step.IsError,
			})
		}
		return nil
	})

	result.Inserted = len(events)
	return events, result, nil
}

func NewAntigravityAdapter() *AntigravityAdapter {
	return &AntigravityAdapter{Classifier: collection.ActivityClassifier{}}
}

var _ collection.AgentAdapter = (*AntigravityAdapter)(nil)
var _ = fmt.Sprintf
