package geminicli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// chatEntry supports two JSON shapes Gemini CLI has used across versions.
// Shape 1: {role, content, metadata: {inputTokenCount, outputTokenCount}, project}
// Shape 2: {role, parts, usageMetadata: {promptTokenCount, candidatesTokenCount}, project}
type chatEntry struct {
	Role    string `json:"role"`
	Project string `json:"project"`
	// Shape 1
	Metadata struct {
		InputTokenCount  int    `json:"inputTokenCount"`
		OutputTokenCount int    `json:"outputTokenCount"`
		Model            string `json:"model"`
		Turn             string `json:"turn"` // used as event key
	} `json:"metadata"`
	// Shape 2
	UsageMetadata struct {
		PromptTokenCount     int    `json:"promptTokenCount"`
		CandidatesTokenCount int    `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	// tool calls in parts
	Parts json.RawMessage `json:"parts"`
}

// GeminiCliAdapter reads chat JSON logs from ~/.gemini/.
type GeminiCliAdapter struct {
	HomeDir    string
	Classifier collection.ActivityClassifier
}

func (a GeminiCliAdapter) logDir() string {
	home := a.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".gemini")
}

// Collect walks ~/.gemini/**/*.json and parses model turns.
func (a GeminiCliAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	result := collection.CollectResult{Source: shared.SourceGeminiCli}
	dir := a.logDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, result, nil
	}

	var events []collection.UsageEvent
	seen := make(map[string]bool)
	idx := 0

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var entries []chatEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil
		}
		sessionID := strings.TrimSuffix(filepath.Base(path), ".json")
		for _, e := range entries {
			if strings.ToLower(e.Role) != "model" {
				continue
			}
			input, output := tokenCounts(e)
			if input+output == 0 {
				continue
			}
			result.Found++
			idx++
			key := sessionID + "/" + strconv.Itoa(idx)
			if seen[key] {
				continue
			}
			seen[key] = true
			ts := time.Now().UTC().Format(time.RFC3339)
			project := e.Project
			if project == "" {
				project = filepath.Base(filepath.Dir(path))
			}
			events = append(events, collection.UsageEvent{
				Source:       shared.SourceGeminiCli,
				SessionID:    sessionID,
				EventKey:     key,
				Project:      project,
				Model:        modelFromEntry(e),
				Activity:     a.Classifier.Classify(nil, ""),
				TS:           ts,
				InputTokens:  input,
				OutputTokens: output,
			})
		}
		return nil
	})

	result.Inserted = len(events)
	return events, result, nil
}

func tokenCounts(e chatEntry) (int, int) {
	if e.Metadata.InputTokenCount > 0 || e.Metadata.OutputTokenCount > 0 {
		return e.Metadata.InputTokenCount, e.Metadata.OutputTokenCount
	}
	return e.UsageMetadata.PromptTokenCount, e.UsageMetadata.CandidatesTokenCount
}

func modelFromEntry(e chatEntry) string {
	if e.Metadata.Model != "" {
		return e.Metadata.Model
	}
	return "gemini-2.0-flash"
}

func NewGeminiCliAdapter() *GeminiCliAdapter {
	return &GeminiCliAdapter{Classifier: collection.ActivityClassifier{}}
}

var _ collection.AgentAdapter = (*GeminiCliAdapter)(nil)
