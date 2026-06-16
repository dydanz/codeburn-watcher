package insights

import "context"

// FollowThroughRepository persists and queries FollowThroughRecords.
// Implemented by infrastructure/sqlite.SqliteFollowThroughStore.
type FollowThroughRepository interface {
	LoadRecords(ctx context.Context) ([]FollowThroughRecord, error)
	UpsertRecords(ctx context.Context, records []FollowThroughRecord) error
}
