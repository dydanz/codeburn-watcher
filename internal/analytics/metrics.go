package analytics

import (
	"time"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// ActivityBreakdown holds per-activity token totals for a window.
type ActivityBreakdown struct {
	Activity shared.Activity
	Tokens   int
	Share    float64 // Tokens / total SpendTokens
	CostUsd  float64
}

// ModelBreakdown holds per-model token totals for a window.
type ModelBreakdown struct {
	Model   string
	Tokens  int
	Share   float64
	CostUsd float64
}

// ContextGrowth holds per-session context growth data.
type ContextGrowth struct {
	Ratio    float64 // late half avg / early half avg
	EarlyAvg float64
	LateAvg  float64
}

// Metrics is the complete computed snapshot for a QueryWindow.
// The central value object of the Analytics BC.
type Metrics struct {
	// Totals
	Events             int
	Sessions           int
	SpendTokens        int
	CostUsd            float64
	CostEstimated      bool
	CostUnpricedTokens int

	// Efficiency ratios — all in [0, 1]
	CacheHitRatio     float64
	ReworkRatio       float64
	ThinkToCodeRatio  float64
	ContextBloatShare float64
	ColdRestartShare  float64
	PremiumWasteShare float64
	RetryShare        float64

	// Breakdowns
	ByActivity map[shared.Activity]ActivityBreakdown
	ByModel    map[string]ModelBreakdown

	// Raw counts used by downstream BCs
	CacheReadTokens     int
	CacheCreationTokens int
	RetryTokens         int
	ColdRestartTokens   int
	BloatSessions       int
	TotalLongSessions   int
}

// MetricsSnapshot wraps Metrics with its identity (window + timestamp).
// Downstream BCs consume Metrics directly; the snapshot is the application-layer DTO.
type MetricsSnapshot struct {
	Window     QueryWindow
	ComputedAt time.Time
	Metrics    Metrics
}
