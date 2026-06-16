# Domain Model

Go type signatures are illustrative. All value objects are immutable
(constructed via factory functions; no exported setters). Aggregates carry
identity and enforce invariants through their methods.

---

## Shared Kernel — `internal/shared`

```go
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
    ActivityThinking    Activity = "thinking"
    ActivityExploration Activity = "exploration"
    ActivityCoding      Activity = "coding"
    ActivityTesting     Activity = "testing"
    ActivityShipping    Activity = "shipping"
    ActivityConversation Activity = "conversation"
)

var Activities = []Activity{
    ActivityThinking, ActivityExploration, ActivityCoding,
    ActivityTesting, ActivityShipping, ActivityConversation,
}

// StoredEvent is the stable cross-BC event contract. Defined once; used by all
// downstream BCs as read-only input. Only BC1 (Collection) writes it.
type StoredEvent struct {
    Source              Source
    SessionID           string
    EventKey            string
    Project             string
    Branch              string
    Model               string
    Tools               []string // decoded from JSON column
    Activity            Activity
    TS                  string   // ISO8601 UTC, always ends in Z
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
    IsError             bool
}
```

---

## BC 1 — Telemetry Collection

### Value Objects

```go
// UsageEvent is the normalized output of one adapter turn.
// Immutable. Constructed by adapter Collect functions.
type UsageEvent struct {
    Source              shared.Source
    SessionID           string
    EventKey            string    // adapter-specific unique key for deduplication
    Project             string    // directory basename only
    Branch              string    // "" if not determinable
    Model               string
    Tools               []string
    Activity            shared.Activity
    TS                  string    // ISO8601 UTC with Z suffix
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
    IsError             bool
}

// CollectResult summarizes one collection run.
type CollectResult struct {
    Source   shared.Source
    Found    int // total events seen in log source
    Inserted int // new events written to store (Found - duplicates)
}

// DeduplicationKey is the unique identity for INSERT OR IGNORE.
// Enforced by the SQLite UNIQUE constraint on (source, event_key).
type DeduplicationKey struct {
    Source   shared.Source
    EventKey string
}
```

### Domain Services

```go
// ActivityClassifier classifies a turn into one of the six Activities.
// Pure function — no I/O.
type ActivityClassifier struct{} // stateless; instantiate once

func (ActivityClassifier) Classify(tools []string, model string) shared.Activity

// Internally uses the following (from TRD §3):
// - WriteTools, ShellTools, ReadTools, PlanTools, InteractiveTools sets
// - TestRE, ShipRE regexes over shell command strings
// - norm() to lowercase+trim tool names
// - 9-step priority chain
```

### Repository Port (defined here, implemented in infrastructure)

```go
// EventWriter is the outgoing port for persisting UsageEvents.
// Implemented by SqliteEventStore.
type EventWriter interface {
    WriteEvents(ctx context.Context, events []UsageEvent) (inserted int, err error)
}
```

### Adapter Port (incoming — adapters implement this)

```go
// AgentAdapter is the incoming port for each agent's log source.
// Each of the six adapters implements this interface.
type AgentAdapter interface {
    Collect(ctx context.Context) (events []UsageEvent, result CollectResult, err error)
}
```

---

## BC 2 — Usage Analytics

### Value Objects

```go
// QueryWindow defines the analysis scope.
type QueryWindow struct {
    Days          int
    ProjectFilter string // "" = all projects
    SourceFilter  string // "" = all sources
}

// ModelPrice is one entry in the static pricing table.
// Costs are in USD per million tokens.
type ModelPrice struct {
    Prefix          string  // matched with strings.HasPrefix(model, prefix)
    InputPerMToken  float64
    OutputPerMToken float64
    CacheReadPerMToken  float64
    CacheWritePerMToken float64
}

// Cost is the estimated USD cost for a set of token counts.
type Cost struct {
    Usd               float64
    Estimated         bool // true if model had no price entry
    UnpricedTokens    int  // tokens for which no price was found
}

// ActivityBreakdown holds per-activity token totals for a window.
type ActivityBreakdown struct {
    Activity shared.Activity
    Tokens   int
    Share    float64 // Tokens / total SpendTokens
    CostUsd  float64
}

// ModelBreakdown holds per-model token totals for a window.
type ModelBreakdown struct {
    Model   string
    Tokens  int
    Share   float64
    CostUsd float64
}

// ContextGrowth holds per-session context growth data.
type ContextGrowth struct {
    Ratio    float64 // late half avg / early half avg
    EarlyAvg float64
    LateAvg  float64
}

// Metrics is the complete computed snapshot for a QueryWindow.
// The central value object of the Analytics BC.
type Metrics struct {
    // Totals
    Events      int
    Sessions    int
    SpendTokens int
    CostUsd     float64
    CostEstimated      bool
    CostUnpricedTokens int

    // Efficiency ratios — all in [0, 1]
    CacheHitRatio      float64
    ReworkRatio        float64
    ThinkToCodeRatio   float64
    ContextBloatShare  float64
    ColdRestartShare   float64
    PremiumWasteShare  float64
    RetryShare         float64

    // Breakdowns
    ByActivity map[shared.Activity]ActivityBreakdown
    ByModel    map[string]ModelBreakdown // insertion-ordered via OrderedMap in impl

    // Raw counts for downstream computation
    CacheReadTokens     int
    CacheCreationTokens int
    RetryTokens         int
    ColdRestartTokens   int
    BloatSessions       int
    TotalLongSessions   int
}
```

### Aggregate Root: `MetricsSnapshot`

`MetricsSnapshot` wraps Metrics with its identity (QueryWindow + computed-at
timestamp). Downstream BCs consume `Metrics` directly; the snapshot wrapper
is the application-layer DTO.

```go
type MetricsSnapshot struct {
    Window     QueryWindow
    ComputedAt time.Time
    Metrics    Metrics
}
```

### Domain Services

```go
// PricingService computes costs from the static PRICES table.
// Pure — no I/O.
type PricingService struct {
    Prices []ModelPrice // 17 entries from TRD §4
}

func (p PricingService) CostOf(model string, input, output, cacheRead, cacheWrite int) Cost
func (p PricingService) IsPremiumModel(model string) bool // strings.Contains checks (TRD §4)

// MetricsEngine computes the full Metrics struct from a slice of StoredEvents.
// Two-pass algorithm (TRD §3): pass 1 aggregates per-session, pass 2 derives ratios.
// Pure — no I/O.
type MetricsEngine struct {
    Pricing PricingService
}

func (m MetricsEngine) Compute(events []shared.StoredEvent) Metrics
```

### Repository Port

```go
// EventReader is the outgoing port for querying stored events.
// Implemented by SqliteEventStore (shared with BC1's EventWriter).
type EventReader interface {
    LoadEvents(ctx context.Context, window QueryWindow) ([]shared.StoredEvent, error)
}
```

---

## BC 3 — Insights

### Value Objects

```go
// Persona is the behavioral archetype assigned to a developer or project.
type Persona struct {
    ID                  string // "firefighter" | "explorer" | "sprinter" | "surgeon" | "architect" | "balanced"
    Label               string
    Description         string
    GeneralRecommendations []string
}

// MetricKey identifies one of the 8 named Findings.
type MetricKey string

const (
    FindingCacheMiss      MetricKey = "cache-miss"
    FindingRework         MetricKey = "rework"
    FindingThinkOvercode  MetricKey = "think-overcode"
    FindingBloat          MetricKey = "bloat"
    FindingColdRestart    MetricKey = "cold-restart"
    FindingPremiumWaste   MetricKey = "premium-waste"
    FindingRetryLoop      MetricKey = "retry-loop"
    FindingHighCost       MetricKey = "high-cost"
)

// Target specifies the improvement goal for a Finding.
type Target struct {
    Metric   MetricKey
    Goal     float64  // the threshold to aim for
    Message  string   // human-readable description of the goal
    Personal bool     // true when derived from developer's own top-quartile sessions
}

// Evidence quantifies why a Finding is active.
type Evidence struct {
    CurrentValue  float64
    GoalValue     float64
    SavingsUsd    float64 // estimated $/month if fixed
    MetricLabel   string  // formatted "14.2%" etc.
    SavingsLabel  string  // formatted "$12/mo"
}

// EnrichedRecommendation is a confirmed Finding with quantified evidence.
type EnrichedRecommendation struct {
    Key      MetricKey
    Label    string
    Priority int // 1 = highest (most savings)
    Target   Target
    Evidence Evidence
}

// PotentialBill is the motivating headline savings estimate.
type PotentialBill struct {
    TotalSavingsUsd float64
    Sessions        int
    Days            int
}

// TrendVerdict is the direction of change.
type TrendVerdict string

const (
    TrendUp   TrendVerdict = "up"
    TrendDown TrendVerdict = "down"
    TrendFlat TrendVerdict = "flat"
)

// TrendRow captures one metric's trend across two windows.
type TrendRow struct {
    Key      MetricKey
    Label    string
    Current  float64
    Previous float64
    Delta    float64
    Verdict  TrendVerdict
    // Formatted strings for rendering
    FmtCurrent  string
    FmtDelta    string
}

// ProjectMover captures one project's spend change between windows.
type ProjectMover struct {
    Project  string
    Current  int     // SpendTokens this window
    Previous int     // SpendTokens previous window
    Delta    int     // Current - Previous
    Verdict  TrendVerdict
}

// BlendedRates holds cost-per-token rates averaged across all models/types.
type BlendedRates struct {
    InputRate  float64 // $/token — blended across all models by input volume
    OutputRate float64
}

// FollowThroughRecord is an entity: persisted, has identity (key + baseline ts).
type FollowThroughRecord struct {
    Key           MetricKey
    BaselineValue float64
    BaselineTS    time.Time
    CurrentValue  float64
    Status        FollowThroughStatus // improving | regressing | resolved
}

type FollowThroughStatus string

const (
    StatusImproving  FollowThroughStatus = "improving"
    StatusRegressing FollowThroughStatus = "regressing"
    StatusResolved   FollowThroughStatus = "resolved"
)
```

### Domain Services

```go
// PersonaAssigner assigns the Persona that best matches a Metrics snapshot.
// Uses ordered threshold evaluation — first match wins (TRD §4).
type PersonaAssigner struct{}

func (PersonaAssigner) Assign(m analytics.Metrics) Persona

// RecommendationEngine produces EnrichedRecommendations from Metrics + events.
type RecommendationEngine struct {
    Pricing analytics.PricingService
}

func (r RecommendationEngine) Enrich(
    events []shared.StoredEvent,
    m analytics.Metrics,
    days int,
) ([]EnrichedRecommendation, PotentialBill)

// TrendAnalyzer computes TrendRows and ProjectMovers by splitting the window.
type TrendAnalyzer struct{}

func (t TrendAnalyzer) SplitWindow(events []shared.StoredEvent, days int) (current, previous []shared.StoredEvent)
func (t TrendAnalyzer) ComputeTrends(events []shared.StoredEvent, days int, engine analytics.MetricsEngine) ([]TrendRow, []ProjectMover)

// FollowThroughService reconciles current Findings against stored records.
// Writes updates to the FollowThroughRepository. Not pure — has I/O.
type FollowThroughService struct {
    Repo FollowThroughRepository
}

func (f FollowThroughService) Sync(
    recs []EnrichedRecommendation,
    m analytics.Metrics,
) ([]FollowThroughRecord, error)
```

### Repository Port

```go
// FollowThroughRepository persists and queries FollowThroughRecords.
// Implemented by SqliteFollowThroughStore.
type FollowThroughRepository interface {
    LoadRecords(ctx context.Context) ([]FollowThroughRecord, error)
    UpsertRecords(ctx context.Context, records []FollowThroughRecord) error
}
```

---

## BC 4 — Trust & Identity

### Aggregate Root: `SigningIdentity`

```go
// SigningIdentity is the aggregate root for one machine's cryptographic identity.
// Created by EnsureIdentity; loaded by LoadIdentity.
// Invariant: private key and public key are a valid Ed25519 pair.
type SigningIdentity struct {
    privateKey ed25519.PrivateKey // unexported — never leaves the aggregate
    publicKey  ed25519.PublicKey
    fingerprint Fingerprint
}

func NewSigningIdentity(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) (SigningIdentity, error)
func (id SigningIdentity) Fingerprint() Fingerprint
func (id SigningIdentity) PublicKeyPEM() []byte // SPKI PEM — safe to embed in exports
func (id SigningIdentity) Sign(payload CanonicalJSON) (Signature, error)
```

### Value Objects

```go
// Fingerprint is the first 8 hex characters of SHA-256(DER(publicKey)).
type Fingerprint string

// CanonicalJSON is the deterministic JSON bytes of an ExportV1 payload.
// Produced by CanonicalizationService. This is what gets signed.
type CanonicalJSON []byte

// Signature is the Ed25519 signature bytes over CanonicalJSON.
type Signature []byte

func (s Signature) Base64() string
func NewSignatureFromBase64(b64 string) (Signature, error)

// Keyring maps username → trusted Fingerprint.
type Keyring map[string]Fingerprint

// VerifyResult is the outcome of verifying a SignedExport.
type VerifyResult struct {
    Valid    bool
    Signer   string // fingerprint of signer, empty if invalid
    Error    string // human-readable failure reason
}
```

### Domain Services

```go
// CanonicalizationService produces the canonical JSON bytes of an ExportV1.
// Invariants (TRD §10, §13.5):
// - Keys sorted alphabetically (UTF-8 order, matches ASCII-only key sort)
// - No undefined/zero-value fields omitted via omitempty struct tags
// - No HTML escaping (SetEscapeHTML(false) or custom serializer)
// - No whitespace
type CanonicalizationService struct{}

func (CanonicalizationService) Canonicalize(export team.ExportPayload) (CanonicalJSON, error)

// VerificationService verifies SignedExports.
type VerificationService struct{}

func (VerificationService) Verify(payload CanonicalJSON, sig Signature, pub ed25519.PublicKey) bool
func (VerificationService) VerifyWithKeyring(
    export team.SignedExport,
    keyring Keyring,
    username string,
) VerifyResult
// Pre-validates sig length (64 bytes) and pub length (32 bytes) before calling
// ed25519.Verify, to prevent panic (TRD §13.5).
```

### Port

```go
// KeystorePort manages the on-disk keypair.
// Implemented by FileKeystore.
type KeystorePort interface {
    EnsureKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error)
    LoadKeypair(dir string) (ed25519.PrivateKey, ed25519.PublicKey, error)
    LoadKeyring(path string) (Keyring, error)
}
```

---

## BC 5 — Team Collaboration

### Aggregate Root: `SignedExport`

```go
// SignedExport is one developer's signed, versioned export document.
// Identity: (Username, GeneratedAt). Immutable after signing.
// Invariant: Signature is valid over Payload canonical bytes.
type SignedExport struct {
    Version     int    // always 1
    Username    string
    GeneratedAt string // ISO8601 UTC
    PublicKey   string // base64 SPKI PEM for verification
    Signature   string // base64 Ed25519 signature

    Payload ExportPayload // the signed content
}

// ExportPayload is the data portion of a SignedExport.
// All fields here are within the signing boundary.
type ExportPayload struct {
    Days            int
    Overall         analytics.Metrics
    ByProject       map[string]analytics.Metrics
    Recommendations []insights.EnrichedRecommendation
    Trends          []insights.TrendRow
    ProjectMovers   []insights.ProjectMover
    FollowThrough   []insights.FollowThroughRecord
    Persona         insights.Persona
}
```

### Entities

```go
// TeamMember is enrolled within a team's trust model.
// Identity: Username.
type TeamMember struct {
    Username            string
    EnrolledFingerprint trust.Fingerprint // "" = not enrolled (no keyring check)
}
```

### Value Objects

```go
// TeamConfig is parsed from the team's hosted config file.
type TeamConfig struct {
    Username string
    PushURL  string
    Members  map[string][]string // group name → []username
}

// DeployConfig is the locally persisted team config for this machine.
type DeployConfig struct {
    Username string
    PushURL  string
    Members  map[string][]string
}

// Rollup is a merged metrics snapshot for a named group.
type Rollup struct {
    Label    string
    Axis     string // "group" | "all"
    Members  []string
    Metrics  analytics.Metrics
    Persona  insights.Persona
    Dominant shared.Activity
}
```

### Domain Services

```go
// ExportBuilder constructs a SignedExport from a collection of domain objects.
type ExportBuilder struct {
    Canonicalizer trust.CanonicalizationService
}

func (b ExportBuilder) Build(
    identity trust.SigningIdentity,
    payload ExportPayload,
) (SignedExport, error)

// ExportMerger implements merge, dedupe, and rollup logic.
// All methods are pure — no I/O.
type ExportMerger struct{}

func (ExportMerger) Dedupe(exports []SignedExport) []SignedExport
// Keeps latest GeneratedAt per username; preserves first-seen insertion order
// for deterministic rendering order.

func (ExportMerger) MergeMetrics(exports []SignedExport) analytics.Metrics
// Sums absolute values; weighted-averages ratios. Applies ?? 0 (Go zero-value).

func (ExportMerger) Rollup(exports []SignedExport, config TeamConfig) []Rollup
// Groups by TeamConfig.Members; one Rollup per group + one "all" Rollup.

func (ExportMerger) DominantActivity(m analytics.Metrics) shared.Activity
// Uses ACTIVITIES slice reduce: first element as accumulator, strict >.

// TeamConfigParser parses the INI/YAML-like team config format.
// Uses the exact regex from TRD §10 — not a general YAML parser.
type TeamConfigParser struct{}

func (TeamConfigParser) Parse(raw string) (TeamConfig, error)
```

### Ports

```go
// ExportTransportPort handles sending and receiving exports/configs.
type ExportTransportPort interface {
    // Push sends a signed export JSON to the configured destination.
    Push(ctx context.Context, filename string, body []byte, url string) error
    // FetchConfig retrieves a team config from a URL.
    FetchConfig(ctx context.Context, url string) (string, error)
}

// DeployConfigPort persists the local deploy configuration.
type DeployConfigPort interface {
    Load() (DeployConfig, error)
    Save(config DeployConfig) error
}

// SchedulerPort installs/removes the OS-level recurring job.
type SchedulerPort interface {
    Install(binaryPath string, config DeployConfig) error
    Remove() error
}
// Implementations: LaunchdScheduler (darwin), CronScheduler (linux)
```

---

## BC 6 — Reconciliation

### Aggregate Root: `ReconciliationResult`

```go
// ReconciliationResult is the outcome of one reconcile run.
// Identity: (Provider, Window). Immutable after Compute.
// Invariant: Breach == true iff any Row has Verdict == VerdictLocalExceedsAPI.
type ReconciliationResult struct {
    Provider string
    Window   analytics.QueryWindow
    Rows     []ReconcileRow
    Breach   bool
}

func (r ReconciliationResult) HasBreach() bool { return r.Breach }
```

### Value Objects

```go
// ProviderUsage is one model's authoritative token counts from the billing API.
type ProviderUsage struct {
    Model               string
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
}

// ReconcileVerdict is the per-model outcome.
type ReconcileVerdict string

const (
    VerdictOk                ReconcileVerdict = "ok"
    VerdictLocalExceedsAPI   ReconcileVerdict = "local-exceeds-api"
    VerdictLocalOnly         ReconcileVerdict = "local-only"
)

// ReconcileRow is one row in the comparison table.
type ReconcileRow struct {
    Model       string
    LocalTokens int
    ApiTokens   int
    Coverage    float64 // LocalTokens / ApiTokens; 0 for local-only
    Verdict     ReconcileVerdict
}

const ReconcileTolerance = 1.05 // local > api * 1.05 = breach
```

### Domain Services

```go
// ReconciliationService compares local event totals against provider API data.
// Pure — takes both inputs; no I/O.
type ReconciliationService struct{}

func (ReconciliationService) Compare(
    events []shared.StoredEvent,
    providerData []ProviderUsage,
) ReconciliationResult
```

### Port

```go
// ProviderUsagePort fetches token usage from a provider's billing API.
// One implementation per provider.
type ProviderUsagePort interface {
    FetchUsage(ctx context.Context, baseURL, key string, startMs, endMs int64) ([]ProviderUsage, error)
}
// Implementations: AnthropicUsageAdapter, OpenAIUsageAdapter
```

---

## BC 7 — Deep Analysis

### Value Objects

```go
// SessionStat is the behavioral summary of one agent session.
type SessionStat struct {
    SessionID           string
    Project             string
    Source              shared.Source
    Turns               int
    SpendTokens         int
    CostUsd             float64
    ErrorTurns          int
    FixIterations       int      // testing→coding transitions
    AvgContextTokens    int
    ContextGrowth       float64  // late/early context ratio; 0 if too short
    ColdRestartTurns    int
    ColdRestartTokens   int
    DurationMin         int
    Dominant            shared.Activity
}

// ToolStat summarizes error and retry behavior for one tool.
type ToolStat struct {
    Tool        string
    Turns       int
    ErrorTurns  int
    ErrorRate   float64
    RetryTokens int
}

// DeepAnalysis is the aggregated output of session + tool analysis.
type DeepAnalysis struct {
    ExpensiveSessions    []SessionStat // top 8 by SpendTokens desc
    FixLoopSessions      []SessionStat // FixIterations >= 2, top 8 by FixIterations desc
    ContextHeavySessions []SessionStat // Turns >= 10, top 8 by AvgContextTokens desc
    BloatTrendSessions   []SessionStat // ContextGrowth >= 2, top 8 by ContextGrowth desc
    ColdRestartSessions  []SessionStat // ColdRestartTurns > 0, top 8 by ColdRestartTokens desc
    ToolStats            []ToolStat    // top 15 by Turns desc
}

// LlmAgent describes one supported local LLM CLI.
type LlmAgent struct {
    Name string // "claude" | "gemini" | "codex"
    Cmd  string
    Args []string // args to pass before the prompt
}

// LlmPayload is the aggregates-only JSON object that crosses the PrivacyBoundary.
// It is constructed from Metrics + DeepAnalysis; never contains raw event content.
type LlmPayload struct {
    WindowDays  int
    Overall     LlmOverall
    Projects    []LlmProjectStat
    ExpensiveSessions    []LlmSessionSlim
    FixLoopSessions      []LlmSessionSlim
    ContextHeavySessions []LlmSessionSlim
    BloatTrendSessions   []LlmSessionSlim
    ColdRestartSessions  []LlmSessionSlim
    ToolErrorRates       []LlmToolErrorRate
    RuleBasedFindings    []insights.MetricKey
}
```

### Domain Services

```go
// SessionAnalyzer computes per-session behavioral statistics.
// Pure — no I/O.
type SessionAnalyzer struct {
    Pricing analytics.PricingService
}

func (s SessionAnalyzer) ComputeSessionStats(events []shared.StoredEvent) []SessionStat
func (s SessionAnalyzer) DeepAnalyze(events []shared.StoredEvent) DeepAnalysis

// ToolAnalyzer computes per-tool error and retry statistics.
// Pure — no I/O.
type ToolAnalyzer struct{}

func (ToolAnalyzer) ComputeToolStats(events []shared.StoredEvent) []ToolStat

// LlmPipeline builds the privacy-safe payload and manages LLM agent dispatch.
type LlmPipeline struct {
    Agents []LlmAgent // declaration order determines DetectAgent priority
}

func (l LlmPipeline) BuildPayload(
    events []shared.StoredEvent,
    m analytics.Metrics,
    recs []insights.EnrichedRecommendation,
    days int,
) LlmPayload

func (l LlmPipeline) BuildPrompt(payload LlmPayload) string
func (l LlmPipeline) DetectAgent() (LlmAgent, bool) // uses exec.LookPath
func (l LlmPipeline) Run(prompt string, agent LlmAgent) int // returns exit code
```
