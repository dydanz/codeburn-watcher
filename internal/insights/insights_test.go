package insights_test

import (
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// metricsWithRatio builds a Metrics with specific ratio values for testing.
func metricsWithRatio(cacheHit, rework, premiumWaste, coldRestart, bloat, thinkToCode, retry float64, costUsd float64) analytics.Metrics {
	byActivity := map[shared.Activity]analytics.ActivityBreakdown{}
	for _, a := range shared.Activities {
		byActivity[a] = analytics.ActivityBreakdown{Activity: a}
	}
	return analytics.Metrics{
		Events: 100, Sessions: 10, SpendTokens: 10000, CostUsd: costUsd,
		CacheHitRatio:     cacheHit,
		ReworkRatio:       rework,
		PremiumWasteShare: premiumWaste,
		ColdRestartShare:  coldRestart,
		ContextBloatShare: bloat,
		ThinkToCodeRatio:  thinkToCode,
		RetryShare:        retry,
		ByActivity:        byActivity,
		ByModel:           map[string]analytics.ModelBreakdown{},
	}
}

func TestActiveFindings_AllFindings(t *testing.T) {
	// All thresholds breached
	m := metricsWithRatio(0.10, 0.30, 0.35, 0.20, 0.30, 0.65, 0.20, 100.0)
	active := insights.ActiveFindings(m)
	if len(active) != 8 {
		t.Errorf("expected 8 active findings, got %d: %v", len(active), active)
	}
}

func TestActiveFindings_None(t *testing.T) {
	// No thresholds breached
	m := metricsWithRatio(0.80, 0.05, 0.05, 0.02, 0.05, 0.20, 0.05, 10.0)
	active := insights.ActiveFindings(m)
	if len(active) != 0 {
		t.Errorf("expected 0 active findings, got %d: %v", len(active), active)
	}
}

func TestActiveFindings_CacheMissThreshold(t *testing.T) {
	// Just below threshold — should NOT trigger
	m := metricsWithRatio(0.40, 0, 0, 0, 0, 0, 0, 0)
	active := insights.ActiveFindings(m)
	for _, k := range active {
		if k == insights.FindingCacheMiss {
			t.Error("CacheMiss should not trigger at exactly 0.40")
		}
	}
	// Just below threshold — SHOULD trigger
	m2 := metricsWithRatio(0.39, 0, 0, 0, 0, 0, 0, 0)
	found := false
	for _, k := range insights.ActiveFindings(m2) {
		if k == insights.FindingCacheMiss {
			found = true
		}
	}
	if !found {
		t.Error("CacheMiss should trigger at 0.39")
	}
}

func TestPersonaAssigner_AllPersonas(t *testing.T) {
	pa := insights.PersonaAssigner{}
	seen := map[string]bool{}

	cases := []analytics.Metrics{
		metricsWithRatio(0.5, 0.30, 0, 0, 0, 0, 0, 0),    // Firefighter (rework > 0.25)
		metricsWithRatio(0.5, 0, 0, 0, 0, 0.70, 0, 0),    // Architect (thinkToCode > 0.60)
		metricsWithRatio(0.5, 0, 0, 0, 0, 0, 0, 0),        // Balanced (fallback)
	}
	for _, m := range cases {
		p := pa.Assign(m)
		seen[p.ID] = true
	}
	// At minimum these should be reachable
	if !seen["firefighter"] {
		t.Error("firefighter persona not reachable")
	}
	if !seen["architect"] {
		t.Error("architect persona not reachable")
	}
	if !seen["balanced"] {
		t.Error("balanced persona (fallback) not reachable")
	}
}

func TestPersonaAssigner_ReturnedPersonaHasLabel(t *testing.T) {
	pa := insights.PersonaAssigner{}
	m := metricsWithRatio(0.5, 0, 0, 0, 0, 0, 0, 0)
	p := pa.Assign(m)
	if p.Label == "" {
		t.Error("Persona should have a non-empty Label")
	}
	if p.ID == "" {
		t.Error("Persona should have a non-empty ID")
	}
}

func TestRecommendationEngineSortedBySavings(t *testing.T) {
	pricing := analytics.NewPricingService()
	engine := insights.NewRecommendationEngine(pricing)
	m := metricsWithRatio(0.10, 0.30, 0.35, 0.20, 0.30, 0.65, 0.20, 100.0)
	m.ByModel = map[string]analytics.ModelBreakdown{
		"claude-sonnet-4": {Model: "claude-sonnet-4", Tokens: 10000, CostUsd: 5.0},
	}

	recs, bill := engine.Enrich(nil, m, 30)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}
	// Verify priorities are sequential starting at 1
	for i, r := range recs {
		if r.Priority != i+1 {
			t.Errorf("rec[%d].Priority = %d, want %d", i, r.Priority, i+1)
		}
	}
	// Verify sorted descending by savings
	for i := 1; i < len(recs); i++ {
		if recs[i-1].Evidence.SavingsUsd < recs[i].Evidence.SavingsUsd {
			t.Errorf("recommendations not sorted: recs[%d].SavingsUsd=%f < recs[%d].SavingsUsd=%f",
				i-1, recs[i-1].Evidence.SavingsUsd, i, recs[i].Evidence.SavingsUsd)
		}
	}
	if bill.Sessions == 0 {
		t.Error("PotentialBill.Sessions should be non-zero")
	}
}
