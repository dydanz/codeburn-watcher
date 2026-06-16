package analytics_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

func makeEvent(session, model string, activity shared.Activity, input, output, cacheRead int, isError bool) shared.StoredEvent {
	return shared.StoredEvent{
		Source:          shared.SourceClaudeCode,
		SessionID:       session,
		EventKey:        fmt.Sprintf("%s-%s-%d", session, model, time.Now().UnixNano()),
		Model:           model,
		Activity:        activity,
		TS:              time.Now().UTC().Format(time.RFC3339),
		InputTokens:     input,
		OutputTokens:    output,
		CacheReadTokens: cacheRead,
		IsError:         isError,
	}
}

func TestComputeEmpty(t *testing.T) {
	engine := analytics.NewMetricsEngine(analytics.NewPricingService())
	m := engine.Compute(nil)
	if m.Events != 0 || m.Sessions != 0 || m.SpendTokens != 0 {
		t.Errorf("empty input should produce zero metrics, got Events=%d Sessions=%d Spend=%d",
			m.Events, m.Sessions, m.SpendTokens)
	}
}

func TestComputeRatiosInBounds(t *testing.T) {
	pricing := analytics.NewPricingService()
	engine := analytics.NewMetricsEngine(pricing)

	events := []shared.StoredEvent{
		makeEvent("s1", "claude-sonnet-4", shared.ActivityCoding, 1000, 500, 200, false),
		makeEvent("s1", "claude-sonnet-4", shared.ActivityThinking, 800, 300, 100, false),
		makeEvent("s2", "claude-haiku-4", shared.ActivityExploration, 500, 200, 50, false),
		makeEvent("s2", "claude-haiku-4", shared.ActivityCoding, 600, 300, 80, true),
		makeEvent("s2", "claude-haiku-4", shared.ActivityCoding, 400, 200, 60, false),
	}

	m := engine.Compute(events)

	ratios := map[string]float64{
		"CacheHitRatio":     m.CacheHitRatio,
		"ReworkRatio":       m.ReworkRatio,
		"ThinkToCodeRatio":  m.ThinkToCodeRatio,
		"ContextBloatShare": m.ContextBloatShare,
		"ColdRestartShare":  m.ColdRestartShare,
		"PremiumWasteShare": m.PremiumWasteShare,
		"RetryShare":        m.RetryShare,
	}

	for name, ratio := range ratios {
		if ratio < 0 || ratio > 1 {
			t.Errorf("%s = %f, want in [0, 1]", name, ratio)
		}
	}
}

func TestComputeSessionCount(t *testing.T) {
	engine := analytics.NewMetricsEngine(analytics.NewPricingService())
	events := []shared.StoredEvent{
		makeEvent("session-a", "claude-sonnet-4", shared.ActivityCoding, 100, 50, 0, false),
		makeEvent("session-a", "claude-sonnet-4", shared.ActivityCoding, 100, 50, 0, false),
		makeEvent("session-b", "claude-sonnet-4", shared.ActivityCoding, 100, 50, 0, false),
	}
	m := engine.Compute(events)
	if m.Sessions != 2 {
		t.Errorf("Sessions = %d, want 2", m.Sessions)
	}
	if m.Events != 3 {
		t.Errorf("Events = %d, want 3", m.Events)
	}
}

func TestComputeActivityBreakdownsPresent(t *testing.T) {
	engine := analytics.NewMetricsEngine(analytics.NewPricingService())
	events := []shared.StoredEvent{
		makeEvent("s1", "claude-sonnet-4", shared.ActivityCoding, 100, 50, 0, false),
	}
	m := engine.Compute(events)
	for _, a := range shared.Activities {
		if _, ok := m.ByActivity[a]; !ok {
			t.Errorf("ByActivity missing activity %q", a)
		}
	}
}

func TestPricingServiceKnownModel(t *testing.T) {
	p := analytics.NewPricingService()
	cost := p.CostOf("claude-sonnet-4-6", 1_000_000, 1_000_000, 0, 0)
	if cost.Estimated {
		t.Error("claude-sonnet-4 should have a known price")
	}
	// $3/M input + $15/M output = $18
	if cost.Usd < 17 || cost.Usd > 19 {
		t.Errorf("CostOf claude-sonnet-4 1M/1M = $%.2f, want ~$18", cost.Usd)
	}
}

func TestPricingServiceUnknownModel(t *testing.T) {
	p := analytics.NewPricingService()
	cost := p.CostOf("some-unknown-model-xyz", 1000, 500, 0, 0)
	if !cost.Estimated {
		t.Error("unknown model should return Estimated=true")
	}
}
