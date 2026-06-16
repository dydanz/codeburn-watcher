package insights

import "github.com/dydanz/codeburn-watcher/internal/analytics"

// MetricKey identifies one of the 8 named findings.
type MetricKey string

const (
	FindingCacheMiss     MetricKey = "cache-miss"
	FindingRework        MetricKey = "rework"
	FindingThinkOvercode MetricKey = "think-overcode"
	FindingBloat         MetricKey = "bloat"
	FindingColdRestart   MetricKey = "cold-restart"
	FindingPremiumWaste  MetricKey = "premium-waste"
	FindingRetryLoop     MetricKey = "retry-loop"
	FindingHighCost      MetricKey = "high-cost"
)

// finding is an internal predicate + metadata pair.
type finding struct {
	Key       MetricKey
	Label     string
	predicate func(m analytics.Metrics) bool
}

// findings is the ordered list of all 8 findings and their activation thresholds.
var findings = []finding{
	{
		Key:   FindingCacheMiss,
		Label: "Low cache hit ratio",
		predicate: func(m analytics.Metrics) bool {
			return m.CacheHitRatio < 0.40
		},
	},
	{
		Key:   FindingRework,
		Label: "High rework ratio",
		predicate: func(m analytics.Metrics) bool {
			return m.ReworkRatio > 0.20
		},
	},
	{
		Key:   FindingThinkOvercode,
		Label: "Thinking exceeds coding",
		predicate: func(m analytics.Metrics) bool {
			return m.ThinkToCodeRatio > 0.50
		},
	},
	{
		Key:   FindingBloat,
		Label: "Context bloat in sessions",
		predicate: func(m analytics.Metrics) bool {
			return m.ContextBloatShare > 0.20
		},
	},
	{
		Key:   FindingColdRestart,
		Label: "Frequent cold restarts",
		predicate: func(m analytics.Metrics) bool {
			return m.ColdRestartShare > 0.10
		},
	},
	{
		Key:   FindingPremiumWaste,
		Label: "Premium model on low-value tasks",
		predicate: func(m analytics.Metrics) bool {
			return m.PremiumWasteShare > 0.25
		},
	},
	{
		Key:   FindingRetryLoop,
		Label: "High retry loop rate",
		predicate: func(m analytics.Metrics) bool {
			return m.RetryShare > 0.15
		},
	},
	{
		Key:   FindingHighCost,
		Label: "High overall cost",
		predicate: func(m analytics.Metrics) bool {
			return m.CostUsd > 50.0
		},
	},
}

// ActiveFindings returns the keys of all findings whose predicate is true.
func ActiveFindings(m analytics.Metrics) []MetricKey {
	var active []MetricKey
	for _, f := range findings {
		if f.predicate(m) {
			active = append(active, f.Key)
		}
	}
	return active
}
