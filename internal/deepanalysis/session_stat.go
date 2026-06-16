package deepanalysis

import (
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// SessionStat is the behavioral summary of one agent session.
type SessionStat struct {
	SessionID        string
	Project          string
	Source           shared.Source
	Turns            int
	SpendTokens      int
	CostUsd          float64
	ErrorTurns       int
	FixIterations    int     // testing→coding transitions
	AvgContextTokens int
	ContextGrowth    float64 // late/early context ratio; 0 if too short
	ColdRestartTurns int
	ColdRestartTokens int
	DurationMin      int
	Dominant         shared.Activity
}

// ToolStat summarizes error and retry behavior for one tool.
type ToolStat struct {
	Tool        string
	Turns       int
	ErrorTurns  int
	ErrorRate   float64
	RetryTokens int
}

// DeepAnalysis is the aggregated output of session + tool analysis.
type DeepAnalysis struct {
	ExpensiveSessions    []SessionStat // top 8 by SpendTokens desc
	FixLoopSessions      []SessionStat // FixIterations >= 2, top 8 by FixIterations desc
	ContextHeavySessions []SessionStat // Turns >= 10, top 8 by AvgContextTokens desc
	BloatTrendSessions   []SessionStat // ContextGrowth >= 2, top 8 by ContextGrowth desc
	ColdRestartSessions  []SessionStat // ColdRestartTurns > 0, top 8 by ColdRestartTokens desc
	ToolStats            []ToolStat    // top 15 by Turns desc
}

// LlmOverall is the privacy-safe global metrics summary for LLM payload.
type LlmOverall struct {
	Events          int
	Sessions        int
	SpendTokens     int
	CostUsd         float64
	CacheHitRatio   float64
	ReworkRatio     float64
	ThinkToCodeRatio float64
}

// LlmProjectStat is per-project rollup for LLM payload.
type LlmProjectStat struct {
	Project     string
	SpendTokens int
	CostUsd     float64
	Sessions    int
}

// LlmSessionSlim is the slim session representation safe to send to an LLM.
// No raw content — behavioral stats only.
type LlmSessionSlim struct {
	Turns             int
	SpendTokens       int
	CostUsd           float64
	ErrorTurns        int
	FixIterations     int
	ContextGrowth     float64
	ColdRestartTokens int
	DurationMin       int
	Dominant          shared.Activity
}

// LlmToolErrorRate is one tool's error stats for LLM payload.
type LlmToolErrorRate struct {
	Tool        string
	Turns       int
	ErrorRate   float64
	RetryTokens int
}

// LlmAgent describes one supported local LLM CLI.
type LlmAgent struct {
	Name string // "claude" | "gemini" | "codex"
	Cmd  string
	Args []string // args to pass before the prompt
}

// LlmPayload is the aggregates-only JSON object that crosses the PrivacyBoundary.
// Constructed from Metrics + DeepAnalysis; never contains raw event content.
type LlmPayload struct {
	WindowDays           int
	Overall              LlmOverall
	Projects             []LlmProjectStat
	ExpensiveSessions    []LlmSessionSlim
	FixLoopSessions      []LlmSessionSlim
	ContextHeavySessions []LlmSessionSlim
	BloatTrendSessions   []LlmSessionSlim
	ColdRestartSessions  []LlmSessionSlim
	ToolErrorRates       []LlmToolErrorRate
	RuleBasedFindings    []insights.MetricKey
}
