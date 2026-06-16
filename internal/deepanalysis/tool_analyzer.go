package deepanalysis

import (
	"sort"
	"strings"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

const topToolN = 15

// ToolAnalyzer computes per-tool error and retry statistics. Pure — no I/O.
type ToolAnalyzer struct{}

// ComputeToolStats aggregates error rates and retry tokens per tool across all events.
func (ToolAnalyzer) ComputeToolStats(events []shared.StoredEvent) []ToolStat {
	type accum struct {
		turns       int
		errorTurns  int
		retryTokens int
	}
	byTool := make(map[string]*accum)

	for _, e := range events {
		tok := e.InputTokens + e.OutputTokens + e.CacheReadTokens + e.CacheCreationTokens
		for _, raw := range e.Tools {
			tool := strings.ToLower(strings.TrimSpace(raw))
			if tool == "" {
				continue
			}
			a := byTool[tool]
			if a == nil {
				a = &accum{}
				byTool[tool] = a
			}
			a.turns++
			if e.IsError {
				a.errorTurns++
				a.retryTokens += tok
			}
		}
	}

	stats := make([]ToolStat, 0, len(byTool))
	for tool, a := range byTool {
		errorRate := 0.0
		if a.turns > 0 {
			errorRate = float64(a.errorTurns) / float64(a.turns)
		}
		stats = append(stats, ToolStat{
			Tool:        tool,
			Turns:       a.turns,
			ErrorTurns:  a.errorTurns,
			ErrorRate:   errorRate,
			RetryTokens: a.retryTokens,
		})
	}

	sort.Slice(stats, func(i, j int) bool { return stats[i].Turns > stats[j].Turns })
	if len(stats) > topToolN {
		stats = stats[:topToolN]
	}
	return stats
}
