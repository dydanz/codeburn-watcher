package reconciliation_test

import (
	"testing"

	"github.com/dydanz/codeburn-watcher/internal/analytics"
	"github.com/dydanz/codeburn-watcher/internal/reconciliation"
)

func TestReconcileOk(t *testing.T) {
	svc := reconciliation.ReconciliationService{}
	window := analytics.QueryWindow{Days: 7}
	local := map[string]int{"claude-sonnet-4-5": 1000}
	api := []reconciliation.ProviderUsage{
		{Model: "claude-sonnet-4-5", InputTokens: 900, OutputTokens: 100},
	}
	result := svc.Compare("anthropic", window, local, api)
	if result.Breach {
		t.Fatal("expected no breach: local == api")
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
	if result.Rows[0].Verdict != reconciliation.VerdictOk {
		t.Errorf("expected VerdictOk, got %s", result.Rows[0].Verdict)
	}
}

func TestReconcileBreachAtTolerance(t *testing.T) {
	svc := reconciliation.ReconciliationService{}
	window := analytics.QueryWindow{Days: 7}
	// local = 1051 > 1000 * 1.05 = 1050 → breach
	local := map[string]int{"gpt-4o": 1051}
	api := []reconciliation.ProviderUsage{
		{Model: "gpt-4o", InputTokens: 800, OutputTokens: 200},
	}
	result := svc.Compare("openai", window, local, api)
	if !result.Breach {
		t.Fatal("expected breach: local exceeds api * tolerance")
	}
	if result.Rows[0].Verdict != reconciliation.VerdictLocalExceedsAPI {
		t.Errorf("expected VerdictLocalExceedsAPI, got %s", result.Rows[0].Verdict)
	}
}

func TestReconcileJustBelowTolerance(t *testing.T) {
	svc := reconciliation.ReconciliationService{}
	window := analytics.QueryWindow{Days: 7}
	// local = 1050 == 1000 * 1.05 → NOT a breach (boundary is exclusive)
	local := map[string]int{"gpt-4o": 1050}
	api := []reconciliation.ProviderUsage{
		{Model: "gpt-4o", InputTokens: 800, OutputTokens: 200},
	}
	result := svc.Compare("openai", window, local, api)
	if result.Breach {
		t.Fatal("expected no breach: local == api * tolerance (exclusive boundary)")
	}
}

func TestReconcileLocalOnly(t *testing.T) {
	svc := reconciliation.ReconciliationService{}
	window := analytics.QueryWindow{Days: 7}
	local := map[string]int{"gemini-2.0-flash": 500}
	api := []reconciliation.ProviderUsage{} // no API data for this model
	result := svc.Compare("google", window, local, api)
	if result.Breach {
		t.Fatal("local-only should not trigger breach")
	}
	if result.Rows[0].Verdict != reconciliation.VerdictLocalOnly {
		t.Errorf("expected VerdictLocalOnly, got %s", result.Rows[0].Verdict)
	}
}

func TestProviderUsageTotalTokens(t *testing.T) {
	u := reconciliation.ProviderUsage{
		InputTokens:         100,
		OutputTokens:        200,
		CacheReadTokens:     50,
		CacheCreationTokens: 25,
	}
	if got := u.TotalTokens(); got != 375 {
		t.Errorf("TotalTokens = %d, want 375", got)
	}
}
