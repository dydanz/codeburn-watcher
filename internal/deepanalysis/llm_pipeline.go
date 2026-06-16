package deepanalysis

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// LlmPipeline builds privacy-safe payloads and dispatches to local LLM CLIs.
type LlmPipeline struct {
	Agents []LlmAgent // declaration order determines DetectAgent priority
}

// BuildPayload constructs the privacy-safe LlmPayload from domain objects.
// No raw event content (prompts, code, file paths) is included.
func (l LlmPipeline) BuildPayload(
	events []shared.StoredEvent,
	m analytics.Metrics,
	recs []insights.EnrichedRecommendation,
	days int,
) LlmPayload {
	// project rollup
	type projAccum struct {
		tokens   int
		costUsd  float64
		sessions map[string]struct{}
	}
	projMap := make(map[string]*projAccum)
	for _, e := range events {
		a := projMap[e.Project]
		if a == nil {
			a = &projAccum{sessions: make(map[string]struct{})}
			projMap[e.Project] = a
		}
		a.tokens += e.InputTokens + e.OutputTokens + e.CacheReadTokens + e.CacheCreationTokens
		a.sessions[e.SessionID] = struct{}{}
	}
	projects := make([]LlmProjectStat, 0, len(projMap))
	for name, a := range projMap {
		projects = append(projects, LlmProjectStat{
			Project:     name,
			SpendTokens: a.tokens,
			Sessions:    len(a.sessions),
		})
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].SpendTokens > projects[j].SpendTokens })

	// rule-based findings from recommendations
	keys := make([]insights.MetricKey, 0, len(recs))
	for _, r := range recs {
		keys = append(keys, r.Key)
	}

	return LlmPayload{
		WindowDays: days,
		Overall: LlmOverall{
			Events:           m.Events,
			Sessions:         m.Sessions,
			SpendTokens:      m.SpendTokens,
			CostUsd:          m.CostUsd,
			CacheHitRatio:    m.CacheHitRatio,
			ReworkRatio:      m.ReworkRatio,
			ThinkToCodeRatio: m.ThinkToCodeRatio,
		},
		Projects:          projects,
		RuleBasedFindings: keys,
	}
}

// BuildPrompt serialises the payload to JSON and wraps it in an analysis prompt.
func (l LlmPipeline) BuildPrompt(payload LlmPayload) string {
	data, _ := json.MarshalIndent(payload, "", "  ")
	return fmt.Sprintf(
		"You are an AI coding assistant advisor. Analyse the following aggregate token usage"+
			" metrics and provide 3-5 specific, actionable recommendations.\n\n"+
			"Privacy note: this payload contains only aggregate statistics — no code,"+
			" prompts, or personal data.\n\n```json\n%s\n```\n", data)
}

// DetectAgent returns the first agent whose command is found in PATH.
func (l LlmPipeline) DetectAgent() (LlmAgent, bool) {
	for _, a := range l.Agents {
		if _, err := exec.LookPath(a.Cmd); err == nil {
			return a, true
		}
	}
	return LlmAgent{}, false
}

// Run executes the LLM CLI with the prompt passed via stdin argument.
// Returns the exit code (0 = success).
func (l LlmPipeline) Run(prompt string, agent LlmAgent) int {
	args := append(agent.Args, prompt) //nolint:gocritic
	cmd := exec.Command(agent.Cmd, args...)
	cmd.Stdout = writerFunc(func(p []byte) (int, error) {
		fmt.Print(strings.TrimRight(string(p), ""))
		return len(p), nil
	})
	cmd.Stderr = cmd.Stdout
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
