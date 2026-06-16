package reconciliation

import "github.com/dydanz/codeburn-watcher/internal/analytics"

// ReconcileTolerance is the 5% buffer before flagging a breach.
// Accounts for clock skew and daily-bucket boundary effects.
const ReconcileTolerance = 1.05

// ReconcileVerdict is the per-model outcome.
type ReconcileVerdict string

const (
	VerdictOk              ReconcileVerdict = "ok"
	VerdictLocalExceedsAPI ReconcileVerdict = "local-exceeds-api"
	VerdictLocalOnly       ReconcileVerdict = "local-only"
)

// ReconcileRow is one row in the per-model comparison table.
type ReconcileRow struct {
	Model       string
	LocalTokens int
	ApiTokens   int
	Coverage    float64 // LocalTokens / ApiTokens; 0 for local-only
	Verdict     ReconcileVerdict
}

// ReconciliationResult is the outcome of one reconcile run.
// Invariant: Breach == true iff any Row has Verdict == VerdictLocalExceedsAPI.
type ReconciliationResult struct {
	Provider string
	Window   analytics.QueryWindow
	Rows     []ReconcileRow
	Breach   bool
}

// HasBreach returns true when any model's local total exceeds the API total
// beyond the tolerance threshold.
func (r ReconciliationResult) HasBreach() bool { return r.Breach }
