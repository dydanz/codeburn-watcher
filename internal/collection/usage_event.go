package collection

import "github.com/dydanz/codeburn-watcher/internal/shared"

// UsageEvent is the normalized output of one adapter turn.
// Immutable. Constructed by adapter Collect functions.
type UsageEvent struct {
	Source              shared.Source
	SessionID           string
	EventKey            string   // adapter-specific unique key for deduplication
	Project             string   // directory basename only
	Branch              string   // "" if not determinable
	Model               string
	Tools               []string
	Activity            shared.Activity
	TS                  string // ISO8601 UTC with Z suffix
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	IsError             bool
}

// CollectResult summarises one collection run for a single source.
type CollectResult struct {
	Source   shared.Source
	Found    int // total events seen in log source
	Inserted int // new events written to store (Found - duplicates)
}

// DeduplicationKey is the unique identity for INSERT OR IGNORE.
// Enforced by a SQLite UNIQUE constraint on (source, event_key).
type DeduplicationKey struct {
	Source   shared.Source
	EventKey string
}
