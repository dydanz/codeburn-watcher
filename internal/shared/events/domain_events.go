// Package events defines domain events for codeburn-watcher.
// Events are plain structs propagated via direct function return values —
// no event bus in the initial implementation.
package events

import (
	"time"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// UsageEventsCollected is emitted after a collection run for one source.
type UsageEventsCollected struct {
	Source     shared.Source
	Found      int
	Inserted   int
	OccurredAt time.Time
}

// MetricsComputed is emitted after the analytics engine runs.
// The Metrics field uses interface{} to avoid a circular import with the
// analytics package; callers type-assert to analytics.Metrics.
type MetricsComputed struct {
	WindowDays int
	OccurredAt time.Time
}

// RecommendationsEnriched is emitted when recommendations are ready.
type RecommendationsEnriched struct {
	Count      int
	OccurredAt time.Time
}

// ExportSigned is emitted when a signed export is produced.
type ExportSigned struct {
	Username    string
	Fingerprint string // hex fingerprint
	GeneratedAt string // ISO8601 UTC
	OccurredAt  time.Time
}

// ReconciliationBreached is emitted when local tokens exceed provider billed
// beyond the tolerance threshold.
type ReconciliationBreached struct {
	Provider   string
	BreachRows int
	OccurredAt time.Time
}
