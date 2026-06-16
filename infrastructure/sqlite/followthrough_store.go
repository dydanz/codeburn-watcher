package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/insights"
)

// SqliteFollowThroughStore implements insights.FollowThroughRepository.
// Table: recommendations (metric_key, baseline_value, baseline_ts, current_value, status, updated_at)
// Uses INSERT OR REPLACE for upsert semantics.
type SqliteFollowThroughStore struct {
	db *sql.DB
}

// NewFollowThroughStore wraps an open *sql.DB.
func NewFollowThroughStore(db *sql.DB) *SqliteFollowThroughStore {
	return &SqliteFollowThroughStore{db: db}
}

// LoadRecords returns all follow-through records from the recommendations table.
func (s *SqliteFollowThroughStore) LoadRecords(ctx context.Context) ([]insights.FollowThroughRecord, error) {
	const q = `SELECT metric_key, baseline_value, baseline_ts, current_value, status FROM recommendations`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("load recommendations: %w", err)
	}
	defer rows.Close()

	var records []insights.FollowThroughRecord
	for rows.Next() {
		var r insights.FollowThroughRecord
		var key, tsStr, status string
		if err := rows.Scan(&key, &r.BaselineValue, &tsStr, &r.CurrentValue, &status); err != nil {
			return nil, fmt.Errorf("scan recommendation: %w", err)
		}
		r.Key = insights.MetricKey(key)
		r.Status = insights.FollowThroughStatus(status)
		t, _ := time.Parse(time.RFC3339, tsStr)
		r.BaselineTS = t
		records = append(records, r)
	}
	return records, rows.Err()
}

// UpsertRecords inserts or replaces follow-through records.
func (s *SqliteFollowThroughStore) UpsertRecords(ctx context.Context, records []insights.FollowThroughRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const q = `INSERT OR REPLACE INTO recommendations
		(metric_key, baseline_value, baseline_ts, current_value, status, updated_at)
		VALUES (?,?,?,?,?,?)`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, r := range records {
		_, err := stmt.ExecContext(ctx,
			string(r.Key),
			r.BaselineValue,
			r.BaselineTS.UTC().Format(time.RFC3339),
			r.CurrentValue,
			string(r.Status),
			now,
		)
		if err != nil {
			return fmt.Errorf("upsert recommendation %s: %w", r.Key, err)
		}
	}
	return tx.Commit()
}
