package shared

// Source identifies which agent produced an event.
type Source string

const (
	SourceClaudeCode  Source = "claude-code"
	SourceGeminiCli   Source = "gemini-cli"
	SourceCodex       Source = "codex"
	SourceCopilot     Source = "copilot"
	SourceCursor      Source = "cursor"
	SourceAntigravity Source = "antigravity"
)

// Activity classifies what a turn was doing.
type Activity string

const (
	ActivityThinking     Activity = "thinking"
	ActivityExploration  Activity = "exploration"
	ActivityCoding       Activity = "coding"
	ActivityTesting      Activity = "testing"
	ActivityShipping     Activity = "shipping"
	ActivityConversation Activity = "conversation"
)

// Activities is the canonical ordered slice used for tie-breaking and iteration.
var Activities = []Activity{
	ActivityThinking,
	ActivityExploration,
	ActivityCoding,
	ActivityTesting,
	ActivityShipping,
	ActivityConversation,
}

// StoredEvent is the stable cross-BC event contract.
// Defined once in the shared kernel; consumed read-only by BCs 2, 3, 6, 7.
// Only BC1 (Collection) writes it. Any schema change requires coordinated
// updates across all consuming BCs and the SQLite DDL.
type StoredEvent struct {
	Source              Source
	SessionID           string
	EventKey            string
	Project             string
	Branch              string
	Model               string
	Tools               []string // decoded from JSON column
	Activity            Activity
	TS                  string // ISO8601 UTC, always ends in Z
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	IsError             bool
}
