package insights

import (
	"fmt"
	"sort"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// Target specifies the improvement goal for a finding.
type Target struct {
	Metric   MetricKey
	Goal     float64
	Message  string
	Personal bool // true when derived from developer's own top-quartile sessions
}

// Evidence quantifies why a finding is active.
type Evidence struct {
	CurrentValue float64
	GoalValue    float64
	SavingsUsd   float64
	MetricLabel  string
	SavingsLabel string
}

// EnrichedRecommendation is a confirmed finding with quantified evidence.
type EnrichedRecommendation struct {
	Key      MetricKey
	Label    string
	Priority int // 1 = highest savings
	Target   Target
	Evidence Evidence
}

// PotentialBill is the headline savings estimate across all active recommendations.
type PotentialBill struct {
	TotalSavingsUsd float64
	Sessions        int
	Days            int
}

// BlendedRates holds cost-per-token averaged across all models.
type BlendedRates struct {
	InputRate  float64
	OutputRate float64
}

// RecommendationEngine produces EnrichedRecommendations from Metrics + events.
type RecommendationEngine struct {
	Pricing analytics.PricingService
}

// NewRecommendationEngine returns a RecommendationEngine with the default pricing table.
func NewRecommendationEngine(pricing analytics.PricingService) RecommendationEngine {
	return RecommendationEngine{Pricing: pricing}
}

// Enrich evaluates all 8 findings against m and returns enriched recommendations
// for those that are active, sorted by estimated savings descending.
func (r RecommendationEngine) Enrich(
	events []shared.StoredEvent,
	m analytics.Metrics,
	days int,
) ([]EnrichedRecommendation, PotentialBill) {
	blended := r.blendedRates(m)
	active := ActiveFindings(m)

	var recs []EnrichedRecommendation
	for _, key := range active {
		rec := r.enrich(key, m, blended, days)
		recs = append(recs, rec)
	}

	// sort by savings descending, assign priority
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Evidence.SavingsUsd > recs[j].Evidence.SavingsUsd
	})
	var total float64
	for i := range recs {
		recs[i].Priority = i + 1
		total += recs[i].Evidence.SavingsUsd
	}

	bill := PotentialBill{
		TotalSavingsUsd: total,
		Sessions:        m.Sessions,
		Days:            days,
	}
	return recs, bill
}

func (r RecommendationEngine) blendedRates(m analytics.Metrics) BlendedRates {
	if m.SpendTokens == 0 {
		return BlendedRates{}
	}
	var totalInput, totalOutput int
	var totalCost float64
	for _, bd := range m.ByModel {
		totalInput += bd.Tokens / 2 // rough split
		totalOutput += bd.Tokens / 2
		totalCost += bd.CostUsd
	}
	if totalInput == 0 {
		return BlendedRates{}
	}
	rate := totalCost / float64(totalInput+totalOutput)
	return BlendedRates{InputRate: rate, OutputRate: rate}
}

// savingsUsd estimates monthly savings from resolving a finding.
func (r RecommendationEngine) savingsUsd(avoidableTokens int, blended BlendedRates, days int) float64 {
	if days == 0 {
		return 0
	}
	dailyRate := (blended.InputRate + blended.OutputRate) / 2
	monthlySavings := float64(avoidableTokens) * dailyRate * 30 / float64(days)
	return monthlySavings
}

func (r RecommendationEngine) enrich(key MetricKey, m analytics.Metrics, blended BlendedRates, days int) EnrichedRecommendation {
	switch key {
	case FindingCacheMiss:
		goal := 0.60
		avoidable := int(float64(m.SpendTokens) * (goal - m.CacheHitRatio))
		if avoidable < 0 {
			avoidable = 0
		}
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "Low cache hit ratio",
			Target: Target{Metric: key, Goal: goal,
				Message: "Aim for 60%+ cache hit ratio by keeping sessions alive longer"},
			Evidence: Evidence{
				CurrentValue: m.CacheHitRatio, GoalValue: goal,
				SavingsUsd: savings,
				MetricLabel: pct(m.CacheHitRatio), SavingsLabel: usd(savings),
			},
		}
	case FindingRework:
		goal := 0.10
		avoidable := int(float64(m.SpendTokens) * (m.ReworkRatio - goal))
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "High rework ratio",
			Target: Target{Metric: key, Goal: goal,
				Message: "Reduce error-recovery loops by breaking work into smaller steps"},
			Evidence: Evidence{
				CurrentValue: m.ReworkRatio, GoalValue: goal,
				SavingsUsd: savings,
				MetricLabel: pct(m.ReworkRatio), SavingsLabel: usd(savings),
			},
		}
	case FindingPremiumWaste:
		goal := 0.10
		avoidable := int(float64(m.SpendTokens) * (m.PremiumWasteShare - goal))
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "Premium model on low-value tasks",
			Target: Target{Metric: key, Goal: goal,
				Message: "Use Haiku or Flash for exploration and conversation turns"},
			Evidence: Evidence{
				CurrentValue: m.PremiumWasteShare, GoalValue: goal,
				SavingsUsd: savings,
				MetricLabel: pct(m.PremiumWasteShare), SavingsLabel: usd(savings),
			},
		}
	case FindingColdRestart:
		goal := 0.05
		avoidable := int(float64(m.ColdRestartTokens) * (1 - goal/m.ColdRestartShare))
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "Frequent cold restarts",
			Target: Target{Metric: key, Goal: goal,
				Message: "Keep sessions warm — resume within 5 minutes to avoid re-paying input tokens"},
			Evidence: Evidence{
				CurrentValue: m.ColdRestartShare, GoalValue: goal,
				SavingsUsd: savings,
				MetricLabel: pct(m.ColdRestartShare), SavingsLabel: usd(savings),
			},
		}
	case FindingBloat:
		avoidable := m.SpendTokens / 10 // rough estimate
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "Context bloat in sessions",
			Target: Target{Metric: key, Goal: 0.10,
				Message: "Start fresh sessions when context grows beyond 2× early size"},
			Evidence: Evidence{
				CurrentValue: m.ContextBloatShare, GoalValue: 0.10,
				SavingsUsd: savings,
				MetricLabel: pct(m.ContextBloatShare), SavingsLabel: usd(savings),
			},
		}
	case FindingThinkOvercode:
		avoidable := m.SpendTokens / 20
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "Thinking exceeds coding",
			Target: Target{Metric: key, Goal: 0.30,
				Message: "Use a cheaper model for planning; switch to Sonnet for implementation"},
			Evidence: Evidence{
				CurrentValue: m.ThinkToCodeRatio, GoalValue: 0.30,
				SavingsUsd: savings,
				MetricLabel: pct(m.ThinkToCodeRatio), SavingsLabel: usd(savings),
			},
		}
	case FindingRetryLoop:
		avoidable := m.RetryTokens / 2
		savings := r.savingsUsd(avoidable, blended, days)
		return EnrichedRecommendation{
			Key:   key,
			Label: "High retry loop rate",
			Target: Target{Metric: key, Goal: 0.05,
				Message: "Diagnose root cause of tool errors instead of letting the agent retry"},
			Evidence: Evidence{
				CurrentValue: m.RetryShare, GoalValue: 0.05,
				SavingsUsd: savings,
				MetricLabel: pct(m.RetryShare), SavingsLabel: usd(savings),
			},
		}
	case FindingHighCost:
		savings := m.CostUsd * 0.20 // assume 20% addressable
		return EnrichedRecommendation{
			Key:   key,
			Label: "High overall cost",
			Target: Target{Metric: key, Goal: 0,
				Message: "Review other findings to identify the highest-savings optimisations"},
			Evidence: Evidence{
				CurrentValue: m.CostUsd, GoalValue: 0,
				SavingsUsd: savings,
				MetricLabel: usd(m.CostUsd), SavingsLabel: usd(savings),
			},
		}
	default:
		return EnrichedRecommendation{Key: key, Label: string(key)}
	}
}

func pct(v float64) string {
	return fmt.Sprintf("%.1f%%", v*100)
}

func usd(v float64) string {
	return fmt.Sprintf("$%.2f/mo", v)
}
