package copilot

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

// copilotSession is one Copilot session file.
type copilotSession struct {
	RequestID string `json:"requestId"`
	Model     string `json:"model"`
	Project   string `json:"project"`
	Turns     []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
		TS      int64  `json:"ts"`
	} `json:"turns"`
}

// CopilotAdapter reads session JSON files from the VS Code Copilot extension storage.
// Uses estimateTokens (4 chars ≈ 1 token) since Copilot doesn't expose token counts.
type CopilotAdapter struct {
	HomeDir    string
	Classifier collection.ActivityClassifier
}

func (a CopilotAdapter) logDir() string {
	home := a.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	// VS Code extension storage on darwin/linux
	candidates := []string{
		filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "github.copilot-chat"),
		filepath.Join(home, ".config", "Code", "User", "globalStorage", "github.copilot-chat"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}

// estimateTokens estimates token count from text length (4 chars ≈ 1 token).
func estimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	return (len(s) + 3) / 4
}

// Collect reads session files from the Copilot extension storage.
func (a CopilotAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	result := collection.CollectResult{Source: shared.SourceCopilot}
	dir := a.logDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, result, nil
	}

	var events []collection.UsageEvent
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var session copilotSession
		if err := json.Unmarshal(data, &session); err != nil {
			return nil
		}

		sessionID := session.RequestID
		if sessionID == "" {
			sessionID = filepath.Base(path)
		}
		var inputTok, outputTok int
		var lastTS int64
		for _, t := range session.Turns {
			switch strings.ToLower(t.Role) {
			case "user":
				inputTok += estimateTokens(t.Content)
			case "assistant":
				outputTok += estimateTokens(t.Content)
			}
			if t.TS > lastTS {
				lastTS = t.TS
			}
		}
		result.Found++
		if inputTok+outputTok == 0 {
			return nil
		}
		ts := time.UnixMilli(lastTS).UTC().Format(time.RFC3339)
		if lastTS == 0 {
			ts = time.Now().UTC().Format(time.RFC3339)
		}
		events = append(events, collection.UsageEvent{
			Source:       shared.SourceCopilot,
			SessionID:    sessionID,
			EventKey:     sessionID,
			Project:      session.Project,
			Model:        session.Model,
			Activity:     a.Classifier.Classify(nil, session.Model),
			TS:           ts,
			InputTokens:  inputTok,
			OutputTokens: outputTok,
		})
		return nil
	})

	result.Inserted = len(events)
	return events, result, nil
}

func NewCopilotAdapter() *CopilotAdapter {
	return &CopilotAdapter{Classifier: collection.ActivityClassifier{}}
}

var _ collection.AgentAdapter = (*CopilotAdapter)(nil)
var _ = fmt.Sprintf
