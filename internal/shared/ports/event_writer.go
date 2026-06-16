package ports

import "context"

// EventWriter is the outbound port for persisting UsageEvents.
// Implemented by infrastructure/sqlite.SqliteEventStore.
// Defined in the shared kernel because BC1 (Collection) writes and
// infrastructure must implement it without importing BC1's domain package.
type EventWriter interface {
	// WriteEvents persists a batch of usage events.
	// Uses INSERT OR IGNORE semantics — duplicate (source, event_key) pairs
	// are silently skipped. Returns the count of newly inserted rows.
	WriteEvents(ctx context.Context, events []RawUsageEvent) (inserted int, err error)
}

// RawUsageEvent is the wire type passed across the EventWriter port boundary.
// It mirrors collection.UsageEvent field-for-field to avoid importing BC1
// from the shared kernel (which would violate the dependency direction).
type RawUsageEvent struct {
	Source              string
	SessionID           string
	EventKey            string
	Project             string
	Branch              string
	Model               string
	Tools               []string
	Activity            string
	TS                  string
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	IsError             bool
}
