package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dydanz/codeburn-watcher/internal/shared"
	sharedports "github.com/dydanz/codeburn-watcher/internal/shared/ports"
)

// SqliteEventStore implements both EventWriter and EventReader.
type SqliteEventStore struct {
	db *sql.DB
}

// NewEventStore wraps an open *sql.DB as an SqliteEventStore.
func NewEventStore(db *sql.DB) *SqliteEventStore {
	return &SqliteEventStore{db: db}
}

// WriteEvents persists a batch of raw events using INSERT OR IGNORE.
// Returns the count of newly inserted rows.
func (s *SqliteEventStore) WriteEvents(ctx context.Context, events []sharedports.RawUsageEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const q = `INSERT OR IGNORE INTO events
		(source, session_id, event_key, project, branch, model, tools, activity,
		 ts, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, is_error)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, e := range events {
		toolsJSON, _ := json.Marshal(e.Tools)
		isErr := 0
		if e.IsError {
			isErr = 1
		}
		res, err := stmt.ExecContext(ctx,
			e.Source, e.SessionID, e.EventKey, e.Project, e.Branch, e.Model,
			string(toolsJSON), e.Activity, e.TS,
			e.InputTokens, e.OutputTokens, e.CacheReadTokens, e.CacheCreationTokens,
			isErr,
		)
		if err != nil {
			return inserted, fmt.Errorf("insert event %s/%s: %w", e.Source, e.EventKey, err)
		}
		rows, _ := res.RowsAffected()
		inserted += int(rows)
	}
	return inserted, tx.Commit()
}

// LoadEvents returns events matching the filter, ordered by ts asc.
func (s *SqliteEventStore) LoadEvents(ctx context.Context, filter sharedports.QueryFilter) ([]shared.StoredEvent, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -filter.Days).Format(time.RFC3339)

	query := `SELECT source, session_id, event_key, project, branch, model, tools, activity,
		ts, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, is_error
		FROM events WHERE ts >= ?`
	args := []any{cutoff}
	if filter.ProjectFilter != "" {
		query += " AND project = ?"
		args = append(args, filter.ProjectFilter)
	}
	if filter.SourceFilter != "" {
		query += " AND source = ?"
		args = append(args, filter.SourceFilter)
	}
	query += " ORDER BY ts ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}
	defer rows.Close()

	var events []shared.StoredEvent
	for rows.Next() {
		var e shared.StoredEvent
		var toolsJSON string
		var isErr int
		if err := rows.Scan(
			&e.Source, &e.SessionID, &e.EventKey, &e.Project, &e.Branch,
			&e.Model, &toolsJSON, &e.Activity, &e.TS,
			&e.InputTokens, &e.OutputTokens, &e.CacheReadTokens, &e.CacheCreationTokens,
			&isErr,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		_ = json.Unmarshal([]byte(toolsJSON), &e.Tools)
		e.IsError = isErr != 0
		events = append(events, e)
	}
	return events, rows.Err()
}
