package ports

import (
	"context"

	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// QueryFilter defines the scope for an event query.
// Mirrors analytics.QueryWindow field-for-field; defined here to avoid
// importing the analytics package from the shared kernel.
type QueryFilter struct {
	Days          int
	ProjectFilter string // "" = all projects
	SourceFilter  string // "" = all sources
}

// EventReader is the outbound port for querying stored events.
// Implemented by infrastructure/sqlite.SqliteEventStore.
// Consumed by BCs 2 (Analytics), 3 (Insights), 6 (Reconciliation), 7 (Deep Analysis).
type EventReader interface {
	// LoadEvents returns all stored events matching the filter window,
	// ordered by timestamp ascending.
	LoadEvents(ctx context.Context, filter QueryFilter) ([]shared.StoredEvent, error)
}
