package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

var declinedRE = regexp.MustCompile(`(?i)(declined|rejected|forbidden|not allowed)`)

// logLine is one JSONL line in a Claude Code session log.
type logLine struct {
	UUID        string `json:"uuid"`
	ParentUUID  string `json:"parentUuid"`
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId"`
	CWD         string `json:"cwd"`
	Message     struct {
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content json.RawMessage `json:"content"`
		Usage   struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// contentBlock helps detect tool use in assistant messages.
type contentBlock struct {
	Type  string `json:"type"`
	Name  string `json:"name"` // for tool_use blocks
	Input struct {
		Command string `json:"command"`
	} `json:"input"`
}

// ClaudeCodeAdapter reads session logs from ~/.claude/projects/**/*.jsonl.
type ClaudeCodeAdapter struct {
	HomeDir   string // override for tests; defaults to os.UserHomeDir()
	Classifier collection.ActivityClassifier
}

func (a ClaudeCodeAdapter) logDir() (string, error) {
	home := a.HomeDir
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// Collect reads all JSONL session logs, extracts assistant turns with usage data.
func (a ClaudeCodeAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	result := collection.CollectResult{Source: shared.SourceClaudeCode}
	dir, err := a.logDir()
	if err != nil {
		return nil, result, nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, result, nil
	}

	var events []collection.UsageEvent
	seen := make(map[string]bool)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		evs, err := parseJSONL(path, a.Classifier)
		if err != nil {
			return nil // skip corrupt files
		}
		for _, e := range evs {
			result.Found++
			key := string(e.Source) + "/" + e.EventKey
			if !seen[key] {
				seen[key] = true
				events = append(events, e)
			}
		}
		return nil
	})
	result.Inserted = len(events)
	return events, result, err
}

func parseJSONL(path string, classifier collection.ActivityClassifier) ([]collection.UsageEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// derive project from directory basename
	project := filepath.Base(filepath.Dir(path))

	var events []collection.UsageEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	for scanner.Scan() {
		var line logLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Message.Role != "assistant" {
			continue
		}
		usage := line.Message.Usage
		total := usage.InputTokens + usage.OutputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
		if total == 0 {
			continue
		}

		tools, isErr := extractTools(line.Message.Content)

		ts := line.Timestamp
		if ts == "" {
			ts = time.Now().UTC().Format(time.RFC3339)
		}
		if !strings.HasSuffix(ts, "Z") && !strings.Contains(ts, "+") {
			ts += "Z"
		}

		events = append(events, collection.UsageEvent{
			Source:              shared.SourceClaudeCode,
			SessionID:          line.SessionID,
			EventKey:           line.UUID,
			Project:            project,
			Model:              line.Message.Model,
			Tools:              tools,
			Activity:           classifier.Classify(tools, line.Message.Model),
			TS:                 ts,
			InputTokens:        usage.InputTokens,
			OutputTokens:       usage.OutputTokens,
			CacheReadTokens:    usage.CacheReadInputTokens,
			CacheCreationTokens: usage.CacheCreationInputTokens,
			IsError:            isErr,
		})
	}
	return events, scanner.Err()
}

func extractTools(raw json.RawMessage) ([]string, bool) {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, false
	}
	toolSet := make(map[string]bool)
	isErr := false
	for _, b := range blocks {
		if b.Type == "tool_use" && b.Name != "" {
			toolSet[b.Name] = true
		}
		if b.Type == "text" {
			// string check for declined patterns done on raw text
		}
	}
	if declinedRE.Match(raw) {
		isErr = true
	}
	tools := make([]string, 0, len(toolSet))
	for t := range toolSet {
		tools = append(tools, t)
	}
	return tools, isErr
}

// NewClaudeCodeAdapter returns a ready-to-use adapter.
func NewClaudeCodeAdapter() *ClaudeCodeAdapter {
	return &ClaudeCodeAdapter{Classifier: collection.ActivityClassifier{}}
}

// ensure interface compliance
var _ collection.AgentAdapter = (*ClaudeCodeAdapter)(nil)

// helpers used by other adapters via shared utility
var _ = fmt.Sprintf // keep fmt import
