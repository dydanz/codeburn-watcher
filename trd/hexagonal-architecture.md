# Hexagonal Architecture — Ports & Adapters

The domain is at the centre. Everything outside — file system, SQLite, HTTP,
OS scheduler, terminal output — communicates through port interfaces. The
domain never imports an infrastructure package.

```
            ┌─────────────────────────────────────────────────┐
            │                APPLICATION LAYER                 │
            │         (command/query handlers, DTOs)           │
            └──────────────────────┬──────────────────────────┘
                                   │
            ┌──────────────────────▼──────────────────────────┐
            │                  DOMAIN LAYER                    │
            │   BCs 1-7: aggregates, VOs, domain services      │
            │   Port interfaces defined here                   │
            └──────────────────────┬──────────────────────────┘
                                   │ implements
            ┌──────────────────────▼──────────────────────────┐
            │              INFRASTRUCTURE LAYER                │
            │                                                  │
            │  INBOUND adapters (drive the domain):           │
            │    - CLI commands (cobra)                        │
            │    - VS Code extension bridge (TS, unchanged)    │
            │                                                  │
            │  OUTBOUND adapters (driven by the domain):      │
            │    - SqliteEventStore                            │
            │    - Agent log adapters (6)                      │
            │    - TerminalRenderer, HtmlRenderer              │
            │    - FileKeystore                                │
            │    - HttpExportTransport / FilesystemTransport   │
            │    - AnthropicUsageAdapter / OpenAIUsageAdapter  │
            │    - LaunchdScheduler / CronScheduler            │
            └─────────────────────────────────────────────────┘
```

---

## Outbound Ports (defined in domain, implemented in infrastructure)

### Event Store (shared — BC1 writes, BC2/3/6/7 read)

```go
// Package: internal/shared/ports

// EventWriter — BC 1 outbound port
type EventWriter interface {
    WriteEvents(ctx context.Context, events []collection.UsageEvent) (inserted int, err error)
}

// EventReader — BC 2/3/6/7 outbound port
type EventReader interface {
    LoadEvents(ctx context.Context, window analytics.QueryWindow) ([]shared.StoredEvent, error)
}
```

**Implementation**: `infrastructure/sqlite/event_store.go`

```go
// SqliteEventStore implements both EventWriter and EventReader.
// Backed by a single SQLite database at ~/.token-monitor/token-monitor.sqlite.
// Uses modernc.org/sqlite (pure Go, no CGO).
type SqliteEventStore struct {
    db *sql.DB
}

// WriteEvents: INSERT OR IGNORE INTO events (...) VALUES (...)
// LoadEvents: SELECT * FROM events WHERE ts >= ? ORDER BY ts
//   Applies project/source filters when set.
//   Uses WAL mode for read concurrency (see WAL-copy pattern for adapters).
```

### Follow-Through Repository (BC 3)

```go
// Package: internal/insights/ports

type FollowThroughRepository interface {
    LoadRecords(ctx context.Context) ([]insights.FollowThroughRecord, error)
    UpsertRecords(ctx context.Context, records []insights.FollowThroughRecord) error
}
```

**Implementation**: `infrastructure/sqlite/followthrough_store.go`

```go
// SqliteFollowThroughStore implements FollowThroughRepository.
// Table: recommendations (metric_key, baseline_value, baseline_ts, current_value, status, updated_at)
// Uses INSERT OR REPLACE for upsert semantics (TRD §5 `syncFindings`).
type SqliteFollowThroughStore struct {
    db *sql.DB
}
```

### Keystore (BC 4)

```go
// Package: internal/trust/ports

type KeystorePort interface {
    EnsureKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error)
    LoadKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error)
    LoadKeyring(path string) (trust.Keyring, error)
}
```

**Implementation**: `infrastructure/fs/keystore.go`

```go
// FileKeystore implements KeystorePort.
// EnsureKeypair:
//   - If signing-key.pem exists at dir, load and return it.
//   - Otherwise generate ed25519.GenerateKey, marshal PKCS8 private + SPKI public,
//     write with os.OpenFile(0600) for private, 0644 for public.
// LoadKeypair: read and parse both PEM files.
// LoadKeyring: read JSON file, unmarshal map[string]string → Keyring.
type FileKeystore struct{}
```

### Export Transport (BC 5)

```go
// Package: internal/team/ports

type ExportTransportPort interface {
    Push(ctx context.Context, filename string, body []byte, destination string) error
    FetchConfig(ctx context.Context, url string) (string, error)
}
```

**Implementations**:

```go
// infrastructure/http/export_transport.go

// HttpExportTransport: POST multipart/form-data or raw JSON body to https:// destination.
// Uses &http.Client{Timeout: 30 * time.Second}.
type HttpExportTransport struct {
    Client *http.Client
}

// FilesystemExportTransport: copy file to a local path destination.
type FilesystemExportTransport struct{}

// CompositeExportTransport: dispatch based on whether destination starts with "http".
type CompositeExportTransport struct {
    HTTP HttpExportTransport
    FS   FilesystemExportTransport
}
```

### Deploy Config Port (BC 5)

```go
// Package: internal/team/ports

type DeployConfigPort interface {
    Load() (team.DeployConfig, error)
    Save(config team.DeployConfig) error
}
```

**Implementation**: `infrastructure/fs/deploy_config.go`

```go
// FileDeployConfig: reads/writes ~/.token-monitor/config.json.
// Path resolution uses os.UserHomeDir() + ".token-monitor/config.json".
type FileDeployConfig struct{}
```

### Scheduler Port (BC 5)

```go
// Package: internal/team/ports

type SchedulerPort interface {
    Install(binaryPath string, config team.DeployConfig) error
    Remove() error
}
```

**Implementations**:

```go
// infrastructure/scheduler/launchd.go (darwin)
// Generates plist XML from team.BuildLaunchdPlist (TRD §10).
// Writes to ~/Library/LaunchAgents/<label>.plist.
// Calls launchctl load / unload via os/exec.
type LaunchdScheduler struct{}

// infrastructure/scheduler/cron.go (linux)
// Appends # TOKEN_MONITOR marker line to crontab via `crontab -l` + `crontab -`.
// Removes by filtering the marker line.
type CronScheduler struct{}

// NewScheduler() SchedulerPort — returns the right impl based on runtime.GOOS.
func NewScheduler() SchedulerPort {
    switch runtime.GOOS {
    case "darwin": return LaunchdScheduler{}
    case "linux":  return CronScheduler{}
    default:       return unsupportedScheduler{}
    }
}
```

### Provider Usage Port (BC 6)

```go
// Package: internal/reconciliation/ports

type ProviderUsagePort interface {
    FetchUsage(ctx context.Context, baseURL, key string, startMs, endMs int64) ([]reconciliation.ProviderUsage, error)
}
```

**Implementations**:

```go
// infrastructure/providers/anthropic.go
// GET /v1/organizations/usage_report/messages
// Headers: x-api-key, anthropic-version: 2023-06-01
// Params: starting_at (ISO), ending_at (ISO), bucket_width=1d, limit=31, group_by[]=model
// Pagination: has_more + next_page
// Field mapping: uncached_input_tokens → Input, output_tokens → Output,
//   cache_read_input_tokens → CacheRead,
//   cache_creation.ephemeral_5m_input_tokens + ephemeral_1h_input_tokens → CacheCreation
type AnthropicUsageAdapter struct {
    Client *http.Client
}

// infrastructure/providers/openai.go
// GET /v1/organization/usage/completions
// Headers: authorization: Bearer <key>
// Params: start_time (unix-seconds), end_time (unix-seconds), bucket_width=1d,
//         group_by=model, limit=31
// Pagination: has_more + next_page
// Field mapping: input_tokens - input_cached_tokens → Input,
//   input_cached_tokens → CacheRead, output_tokens → Output
// (OpenAI input_tokens INCLUDES cached — must subtract)
type OpenAIUsageAdapter struct {
    Client *http.Client
}

// ProviderRegistry maps provider name → ProviderUsagePort.
// Used by the reconcile command to dispatch.
var ProviderRegistry = map[string]ProviderSpec{
    "anthropic": {KeyEnv: "ANTHROPIC_ADMIN_KEY", URLEnv: "TOKEN_MONITOR_ANTHROPIC_URL",
                  DefaultURL: "https://api.anthropic.com", Adapter: AnthropicUsageAdapter{}},
    "openai":    {KeyEnv: "OPENAI_ADMIN_KEY", URLEnv: "TOKEN_MONITOR_OPENAI_URL",
                  DefaultURL: "https://api.openai.com", Adapter: OpenAIUsageAdapter{}},
}
```

---

## Inbound Adapters (driving the domain)

### CLI (`cmd/token-monitor`)

The CLI is the primary inbound adapter. It parses flags, assembles domain
objects, calls application-layer handlers, and formats exit codes.

```go
// cmd/token-monitor/main.go
// Uses github.com/spf13/cobra for sub-command dispatch.
// Each sub-command maps to one Application Service handler.

// Sub-commands:
// collect   → application.CollectHandler
// report    → application.ReportHandler
// analyze   → application.AnalyzeHandler
// html      → application.HtmlHandler
// merge     → application.MergeHandler
// init      → application.InitHandler
// push      → application.PushHandler
// schedule  → application.ScheduleHandler
// reconcile → application.ReconcileHandler
// fingerprint → application.FingerprintHandler
```

### VS Code Extension Bridge

The extension bridge (`extension/src/bridge.ts`) remains TypeScript and
unchanged — it drives the domain via the CLI binary (inbound adapter to
the binary as a black box, not to Go packages directly). See TRD §12.

---

## Output Adapters (Presentation — not a BC)

These are infrastructure adapters that convert domain objects to I/O formats.
They implement no port interfaces; they are called directly by application
handlers.

### Terminal Renderer

```go
// infrastructure/render/terminal.go
// Implements the full RenderReport, RenderTrend, RenderEnrichedRecs,
// RenderTeamReport, RenderAnalysis, RenderReconcile functions from TRD §9 + §11.
// Depends on: github.com/mattn/go-runewidth for Unicode-width column padding.
// No port interface — application handlers call directly.
type TerminalRenderer struct{}

func (TerminalRenderer) RenderReport(m analytics.Metrics, recs []insights.EnrichedRecommendation,
    trends []insights.TrendRow, movers []insights.ProjectMover, days int) string
func (TerminalRenderer) RenderAnalysis(deep deepanalysis.DeepAnalysis, m analytics.Metrics,
    recs []insights.EnrichedRecommendation, days int) string
func (TerminalRenderer) RenderReconcile(provider string, result reconciliation.ReconciliationResult,
    days int) string
func (TerminalRenderer) RenderTeamReport(rollups []team.Rollup, days int) string
```

### HTML Renderer

```go
// infrastructure/render/html.go
// Self-contained dark-theme HTML dashboard.
// Uses custom esc() with numeric HTML entities (not html.EscapeString — TRD §13.5).
type HtmlRenderer struct{}

func (HtmlRenderer) RenderHtml(m analytics.Metrics, recs []insights.EnrichedRecommendation,
    trends []insights.TrendRow, movers []insights.ProjectMover, days int) string
func (HtmlRenderer) RenderTeamHtml(rollups []team.Rollup, days int) string
```

---

## Adapter selection per Agent

The six agent adapters plug into the Collection BC via the `AgentAdapter` port.
They are registered in a discovery list (not a map, to preserve detection order):

```go
// infrastructure/adapters/registry.go

var AgentAdapters = []collection.AgentAdapter{
    adapters.NewClaudeCodeAdapter(),
    adapters.NewGeminiCliAdapter(),
    adapters.NewCodexAdapter(),
    adapters.NewCopilotAdapter(),
    adapters.NewCursorAdapter(),
    adapters.NewAntigravityAdapter(),
}
```

Each adapter:
1. Detects whether its log source exists on this machine (returns empty
   `CollectResult` if not — not an error).
2. Reads log source using the exact file format from TRD §7/§8.
3. Calls `ActivityClassifier.Classify()` for each turn.
4. Returns `[]UsageEvent` — the domain contract.

Cursor and Antigravity adapters additionally use the WAL-copy pattern
(copy `.db`, `.db-wal`, `.db-shm` to a temp dir before reading) to avoid
SQLite WAL conflicts with the live agent process (TRD §8, §13.5).
