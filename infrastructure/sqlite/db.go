package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const ddl = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS events (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    source               TEXT    NOT NULL,
    session_id           TEXT    NOT NULL,
    event_key            TEXT    NOT NULL,
    project              TEXT    NOT NULL DEFAULT '',
    branch               TEXT    NOT NULL DEFAULT '',
    model                TEXT    NOT NULL DEFAULT '',
    tools                TEXT    NOT NULL DEFAULT '[]',
    activity             TEXT    NOT NULL DEFAULT '',
    ts                   TEXT    NOT NULL,
    input_tokens         INTEGER NOT NULL DEFAULT 0,
    output_tokens        INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens    INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    is_error             INTEGER NOT NULL DEFAULT 0,
    UNIQUE(source, event_key)
);

CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_project ON events(project);
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);

CREATE TABLE IF NOT EXISTS recommendations (
    metric_key     TEXT    PRIMARY KEY,
    baseline_value REAL    NOT NULL DEFAULT 0,
    baseline_ts    TEXT    NOT NULL DEFAULT '',
    current_value  REAL    NOT NULL DEFAULT 0,
    status         TEXT    NOT NULL DEFAULT 'open',
    updated_at     TEXT    NOT NULL DEFAULT ''
);
`

// Open opens (or creates) the SQLite database at path, runs DDL migration,
// and returns a ready-to-use *sql.DB.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open %s: %w", path, err)
	}
	if _, err := db.Exec(ddl); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite migrate: %w", err)
	}
	return db, nil
}
