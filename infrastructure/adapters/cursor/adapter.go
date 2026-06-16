package cursor

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/dydanz/codeburn-watcher/internal/collection"
	"github.com/dydanz/codeburn-watcher/internal/shared"
)

// CursorAdapter reads from Cursor's SQLite workspace databases.
// Uses the WAL-copy pattern to avoid conflicts with the live Cursor process.
type CursorAdapter struct {
	HomeDir    string
	Classifier collection.ActivityClassifier
}

func (a CursorAdapter) dbPaths() []string {
	home := a.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	candidates := []string{
		filepath.Join(home, "Library", "Application Support", "Cursor", "User", "workspaceStorage"),
		filepath.Join(home, ".config", "Cursor", "User", "workspaceStorage"),
	}
	var found []string
	for _, c := range candidates {
		_ = filepath.Walk(c, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Base(path) == "state.vscdb" {
				found = append(found, path)
			}
			return nil
		})
	}
	return found
}

// walCopy copies a SQLite DB and its WAL/SHM files to a temp directory.
func walCopy(src string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "cursor-wal-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	dst := filepath.Join(dir, "state.vscdb")
	if err := copyFile(src, dst); err != nil {
		cleanup()
		return "", nil, err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = copyFile(src+suffix, dst+suffix) // ignore missing WAL/SHM
	}
	return dst, cleanup, nil
}

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}

// Collect reads Cursor workspace DBs for AI composer/bubble usage.
func (a CursorAdapter) Collect(_ context.Context) ([]collection.UsageEvent, collection.CollectResult, error) {
	result := collection.CollectResult{Source: shared.SourceCursor}
	paths := a.dbPaths()
	if len(paths) == 0 {
		return nil, result, nil
	}

	var events []collection.UsageEvent
	seen := make(map[string]bool)

	for _, path := range paths {
		tmpDB, cleanup, err := walCopy(path)
		if err != nil {
			continue
		}
		defer cleanup()

		db, err := sql.Open("sqlite", tmpDB+"?mode=ro")
		if err != nil {
			continue
		}

		rows, err := db.Query(`SELECT value FROM ItemTable WHERE key LIKE 'aiService.%'`)
		if err != nil {
			_ = db.Close()
			continue
		}

		for rows.Next() {
			var value []byte
			if err := rows.Scan(&value); err != nil {
				continue
			}
			evs := parseCursorValue(value, a.Classifier)
			for _, e := range evs {
				result.Found++
				if !seen[e.EventKey] {
					seen[e.EventKey] = true
					events = append(events, e)
				}
			}
		}
		_ = rows.Close()
		_ = db.Close()
	}

	result.Inserted = len(events)
	return events, result, nil
}

// parseCursorValue attempts to extract token usage from Cursor's stored JSON blobs.
func parseCursorValue(data []byte, classifier collection.ActivityClassifier) []collection.UsageEvent {
	// Cursor stores usage in proprietary JSON; we parse what we can
	// A real implementation would use protowire for ComposerDoc format
	// For now: return empty (graceful degradation)
	_ = data
	_ = classifier
	return nil
}

func NewCursorAdapter() *CursorAdapter {
	return &CursorAdapter{Classifier: collection.ActivityClassifier{}}
}

var _ collection.AgentAdapter = (*CursorAdapter)(nil)
var _ = fmt.Sprintf
var _ = time.Now
