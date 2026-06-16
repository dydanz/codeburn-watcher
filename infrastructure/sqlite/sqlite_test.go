package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dydanz/codeburn-watcher/infrastructure/sqlite"
	"github.com/dydanz/codeburn-watcher/internal/insights"
	sharedports "github.com/dydanz/codeburn-watcher/internal/shared/ports"
)

func openTestDB(t *testing.T) *sqlite.SqliteEventStore {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlite.NewEventStore(db)
}

func openTestFollowThrough(t *testing.T) *sqlite.SqliteFollowThroughStore {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlite.NewFollowThroughStore(db)
}

func TestWriteAndLoadEvents(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	events := []sharedports.RawUsageEvent{
		{
			Source: "claude-code", SessionID: "s1", EventKey: "e1",
			Project: "myproj", Activity: "coding",
			TS: time.Now().UTC().Format(time.RFC3339),
			InputTokens: 100, OutputTokens: 50,
		},
		{
			Source: "claude-code", SessionID: "s1", EventKey: "e2",
			Project: "myproj", Activity: "thinking",
			TS: time.Now().UTC().Format(time.RFC3339),
			InputTokens: 200, OutputTokens: 80,
		},
	}

	inserted, err := store.WriteEvents(ctx, events)
	if err != nil {
		t.Fatalf("WriteEvents: %v", err)
	}
	if inserted != 2 {
		t.Errorf("inserted = %d, want 2", inserted)
	}

	// duplicate insert should be ignored
	inserted2, err := store.WriteEvents(ctx, events[:1])
	if err != nil {
		t.Fatalf("WriteEvents duplicate: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("duplicate insert should return 0, got %d", inserted2)
	}

	loaded, err := store.LoadEvents(ctx, sharedports.QueryFilter{Days: 7})
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("loaded %d events, want 2", len(loaded))
	}
}

func TestLoadEvents_ProjectFilter(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	ts := time.Now().UTC().Format(time.RFC3339)

	events := []sharedports.RawUsageEvent{
		{Source: "claude-code", SessionID: "s1", EventKey: "e1", Project: "proj-a", TS: ts},
		{Source: "claude-code", SessionID: "s1", EventKey: "e2", Project: "proj-b", TS: ts},
	}
	_, _ = store.WriteEvents(ctx, events)

	loaded, err := store.LoadEvents(ctx, sharedports.QueryFilter{Days: 7, ProjectFilter: "proj-a"})
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Project != "proj-a" {
		t.Errorf("expected 1 proj-a event, got %v", loaded)
	}
}

func TestFollowThroughStore_UpsertAndLoad(t *testing.T) {
	store := openTestFollowThrough(t)
	ctx := context.Background()

	records := []insights.FollowThroughRecord{
		{
			Key:           insights.FindingCacheMiss,
			BaselineValue: 0.3,
			BaselineTS:    time.Now().UTC(),
			CurrentValue:  0.35,
			Status:        insights.StatusImproving,
		},
	}
	if err := store.UpsertRecords(ctx, records); err != nil {
		t.Fatalf("UpsertRecords: %v", err)
	}

	loaded, err := store.LoadRecords(ctx)
	if err != nil {
		t.Fatalf("LoadRecords: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 record, got %d", len(loaded))
	}
	if loaded[0].Key != insights.FindingCacheMiss {
		t.Errorf("Key = %s, want FindingCacheMiss", loaded[0].Key)
	}
	if loaded[0].Status != insights.StatusImproving {
		t.Errorf("Status = %s, want improving", loaded[0].Status)
	}
}
