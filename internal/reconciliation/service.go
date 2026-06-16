package reconciliation

import "github.com/dydanz/codeburn-watcher/internal/analytics"

// ReconciliationService compares local event totals against provider billing data.
type ReconciliationService struct{}

// Compare builds a ReconciliationResult.
// localTotals: map[model]→total local tokens.
// apiUsages: slice from the provider billing API.
func (ReconciliationService) Compare(
	provider string,
	window analytics.QueryWindow,
	localTotals map[string]int,
	apiUsages []ProviderUsage,
) ReconciliationResult {
	// index API data by model name
	apiByModel := make(map[string]int, len(apiUsages))
	for _, u := range apiUsages {
		apiByModel[u.Model] = u.TotalTokens()
	}

	rows := make([]ReconcileRow, 0, len(localTotals))
	breach := false

	for model, localTok := range localTotals {
		apiTok, hasAPI := apiByModel[model]
		var verdict ReconcileVerdict
		var coverage float64

		switch {
		case !hasAPI:
			verdict = VerdictLocalOnly
		case float64(localTok) > float64(apiTok)*ReconcileTolerance:
			verdict = VerdictLocalExceedsAPI
			breach = true
			coverage = safeDiv(localTok, apiTok)
		default:
			verdict = VerdictOk
			coverage = safeDiv(localTok, apiTok)
		}

		rows = append(rows, ReconcileRow{
			Model:       model,
			LocalTokens: localTok,
			ApiTokens:   apiTok,
			Coverage:    coverage,
			Verdict:     verdict,
		})
	}

	return ReconciliationResult{
		Provider: provider,
		Window:   window,
		Rows:     rows,
		Breach:   breach,
	}
}

func safeDiv(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}
