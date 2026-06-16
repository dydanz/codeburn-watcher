# Go Package Layout

Maps the DDD layer model to a concrete Go module structure. Each package
boundary corresponds to a bounded context or infrastructure concern.

```
github.com/yourorg/token-monitor/   (go.mod root)
│
├── cmd/
│   └── token-monitor/
│       ├── main.go          -- cobra root command + flag parsing
│       └── wire.go          -- dependency assembly (NewDeps → AppDeps)
│
├── internal/
│   │
│   ├── shared/              -- Shared Kernel: StoredEvent + Activity + Source
│   │   ├── event.go         -- StoredEvent, Source, Activity, Activities slice
│   │   ├── events/
│   │   │   └── domain_events.go -- UsageEventsCollected, MetricsComputed, ...
│   │   └── ports/
│   │       ├── event_writer.go  -- EventWriter interface
│   │       └── event_reader.go  -- EventReader interface
│   │
│   ├── collection/          -- BC 1: Telemetry Collection
│   │   ├── usage_event.go       -- UsageEvent, CollectResult, DeduplicationKey
│   │   ├── classifier.go        -- ActivityClassifier (WriteTools, ShellTools, TestRE, ShipRE, norm, Classify)
│   │   └── agent_adapter.go     -- AgentAdapter interface
│   │
│   ├── analytics/           -- BC 2: Usage Analytics
│   │   ├── query_window.go      -- QueryWindow
│   │   ├── model_price.go       -- ModelPrice, Cost, PRICES table (17 entries)
│   │   ├── metrics.go           -- Metrics, ActivityBreakdown, ModelBreakdown, ContextGrowth
│   │   ├── metrics_engine.go    -- MetricsEngine, two-pass Compute algorithm
│   │   ├── pricing_service.go   -- PricingService, CostOf, IsPremiumModel
│   │   └── ports/
│   │       └── event_reader.go  -- (re-exports shared/ports EventReader for BC-local use)
│   │
│   ├── insights/            -- BC 3: Insights (recommendations, trends, personas, follow-through)
│   │   ├── persona.go           -- Persona, PersonaAssigner, PERSONAS slice, Balanced
│   │   ├── finding.go           -- MetricKey constants, Finding, 8 structured findings
│   │   ├── recommendation.go    -- EnrichedRecommendation, Target, Evidence, PotentialBill
│   │   ├── recommendation_engine.go -- RecommendationEngine, Enrich, blendedRates, savingsUsd
│   │   ├── follow_through.go    -- FollowThroughRecord, FollowThroughStatus, FollowThroughService
│   │   ├── trend.go             -- TrendRow, TrendVerdict, ProjectMover, TrendAnalyzer, SplitWindow
│   │   └── ports/
│   │       └── follow_through_repo.go -- FollowThroughRepository interface
│   │
│   ├── trust/               -- BC 4: Trust & Identity
│   │   ├── identity.go          -- SigningIdentity aggregate, NewSigningIdentity, Fingerprint()
│   │   ├── fingerprint.go       -- Fingerprint type, ComputeFingerprint(pub) → Fingerprint
│   │   ├── canonical.go         -- CanonicalizationService, Canonicalize (no HTML-escape, sorted keys)
│   │   ├── signing.go           -- Signature, SigningService, Sign, Verify (with length pre-checks)
│   │   ├── keyring.go           -- Keyring, VerifyResult, VerificationService, VerifyWithKeyring
│   │   └── ports/
│   │       └── keystore.go      -- KeystorePort interface
│   │
│   ├── team/                -- BC 5: Team Collaboration
│   │   ├── export.go            -- ExportPayload, SignedExport (version, username, generatedAt, sig)
│   │   ├── export_builder.go    -- ExportBuilder, Build
│   │   ├── export_filename.go   -- ExportFilename(username, generatedAt) → string
│   │   ├── export_merger.go     -- ExportMerger, Dedupe, MergeMetrics, Rollup, DominantActivity
│   │   ├── team_config.go       -- TeamConfig, DeployConfig, TeamConfigParser (regex-based, no YAML lib)
│   │   ├── member.go            -- TeamMember entity
│   │   ├── rollup.go            -- Rollup value object
│   │   └── ports/
│   │       ├── export_transport.go -- ExportTransportPort interface
│   │       ├── deploy_config.go    -- DeployConfigPort interface
│   │       └── scheduler.go        -- SchedulerPort interface
│   │
│   ├── reconciliation/      -- BC 6: Reconciliation
│   │   ├── result.go            -- ReconciliationResult, ReconcileRow, ReconcileVerdict, Tolerance
│   │   ├── provider_usage.go    -- ProviderUsage, ProviderSpec
│   │   ├── service.go           -- ReconciliationService, Compare
│   │   └── ports/
│   │       └── provider_usage.go -- ProviderUsagePort interface
│   │
│   └── deepanalysis/        -- BC 7: Deep Analysis
│       ├── session_stat.go      -- SessionStat, ToolStat, DeepAnalysis value objects
│       ├── session_analyzer.go  -- SessionAnalyzer, ComputeSessionStats, DeepAnalyze
│       ├── tool_analyzer.go     -- ToolAnalyzer, ComputeToolStats
│       └── llm_pipeline.go     -- LlmAgent, LlmPayload, LlmPipeline, BuildPayload,
│                                   BuildPrompt, DetectAgent, Run
│
├── infrastructure/
│   │
│   ├── sqlite/              -- SQLite implementations of all repo/store ports
│   │   ├── db.go                -- Open(path) → *sql.DB, DDL migration (events + recommendations tables)
│   │   ├── event_store.go       -- SqliteEventStore: EventWriter + EventReader
│   │   └── followthrough_store.go -- SqliteFollowThroughStore: FollowThroughRepository
│   │
│   ├── adapters/            -- Six AgentAdapter implementations (BC 1 port)
│   │   ├── registry.go          -- AllAdapters() → []collection.AgentAdapter
│   │   ├── claudecode/
│   │   │   └── adapter.go       -- ClaudeCodeAdapter: JSONL parsing, DECLINED_RE, error-linking
│   │   ├── geminicli/
│   │   │   └── adapter.go       -- GeminiCliAdapter: chat JSON, dual-shape project map
│   │   ├── codex/
│   │   │   └── adapter.go       -- CodexAdapter: walk(), cumulative-counter delta-diff
│   │   ├── copilot/
│   │   │   └── adapter.go       -- CopilotAdapter: estimateTokens, session files, requestId fallback
│   │   ├── cursor/
│   │   │   └── adapter.go       -- CursorAdapter: WAL-copy, Bubble/ComposerDoc, workspace map
│   │   └── antigravity/
│   │       └── adapter.go       -- AntigravityAdapter: STEP_TOOLS, tsToIso, gen/step attribution
│   │
│   ├── protowire/           -- Protobuf varint decoder (used by cursor adapter)
│   │   └── protowire.go         -- readVarint (uint64), decodeMessage, IntField/StrField/MsgField
│   │
│   ├── providers/           -- ProviderUsagePort implementations (BC 6)
│   │   ├── registry.go          -- ProviderRegistry map[string]reconciliation.ProviderSpec
│   │   ├── anthropic.go         -- AnthropicUsageAdapter: paginated GET /v1/organizations/usage_report/messages
│   │   └── openai.go            -- OpenAIUsageAdapter: paginated GET /v1/organization/usage/completions
│   │
│   ├── fs/                  -- Filesystem port implementations
│   │   ├── keystore.go          -- FileKeystore: EnsureKeypair, LoadKeypair, LoadKeyring
│   │   └── deploy_config.go     -- FileDeployConfig: Load/Save ~/.token-monitor/config.json
│   │
│   ├── http/                -- HTTP port implementations
│   │   └── export_transport.go  -- HttpExportTransport + FilesystemExportTransport +
│   │                               CompositeExportTransport (dispatch by URL scheme)
│   │
│   ├── scheduler/           -- SchedulerPort implementations (BC 5)
│   │   ├── scheduler.go         -- NewScheduler() → platform-appropriate impl
│   │   ├── launchd.go           -- LaunchdScheduler (darwin): plist + launchctl
│   │   └── cron.go              -- CronScheduler (linux): crontab -l / crontab -
│   │
│   └── render/              -- Presentation output adapters (not a BC)
│       ├── terminal.go          -- TerminalRenderer: ANSI report, trend, recs, analysis, reconcile
│       ├── html.go              -- HtmlRenderer: self-contained dark HTML dashboard
│       └── format.go            -- FmtTokens, FmtCost, FmtTrendValue, bar, Table (runewidth)
│
└── application/             -- Application layer: command/query handlers + dependency wiring
    ├── deps.go              -- AppDeps struct (all handlers in one struct)
    ├── collect.go           -- CollectHandler
    ├── report.go            -- ReportHandler
    ├── analyze.go           -- AnalyzeHandler
    ├── html.go              -- HtmlHandler
    ├── merge.go             -- MergeHandler
    ├── init.go              -- InitHandler
    ├── push.go              -- PushHandler
    ├── schedule.go          -- ScheduleHandler
    ├── reconcile.go         -- ReconcileHandler
    └── fingerprint.go       -- FingerprintHandler
```

## Import rules (enforced by `internal/`)

```
cmd/                → application/, infrastructure/, internal/shared
application/        → internal/*, infrastructure/ ports only (no direct infra impl imports)
internal/*/         → internal/shared, internal/<other-bc> (interfaces only, not impls)
infrastructure/     → internal/* (to implement ports), external libs
```

No circular imports. Domain BCs (`internal/*`) never import infrastructure.
Infrastructure packages import domain packages to implement port interfaces.

## Key package dependency graph

```
shared
  ↑
collection ─→ shared/ports (EventWriter)
analytics  ─→ shared
insights   ─→ analytics, shared
trust      ─→ team (ExportPayload type — minimal coupling)
team       ─→ analytics, insights, trust (for signing), shared
reconciliation ─→ analytics (QueryWindow), shared
deepanalysis  ─→ analytics, insights, shared

infrastructure/sqlite ─→ collection, analytics, insights (shared), shared/ports
infrastructure/adapters/* ─→ collection, shared, infrastructure/protowire (cursor only)
infrastructure/providers/* ─→ reconciliation
infrastructure/fs ─→ trust
infrastructure/http ─→ team/ports
infrastructure/scheduler ─→ team, team/ports
infrastructure/render ─→ analytics, insights, team, reconciliation, deepanalysis

application/* ─→ all internal/*, infrastructure/(via port interfaces only)
cmd/ ─→ application/, infrastructure/ (for wiring concrete impls)
```

## go.mod dependencies (expected minimal set)

```go
module github.com/yourorg/token-monitor

go 1.23

require (
    github.com/spf13/cobra             v1.8.x   // CLI sub-command dispatch
    github.com/mattn/go-runewidth      v0.0.x   // Unicode terminal column widths
    modernc.org/sqlite                 v1.x.x   // Pure-Go SQLite (no CGO)
    // All crypto, JSON, HTTP, exec, os — stdlib
)
```

## Test strategy by layer

| Layer | Test type | Notes |
|---|---|---|
| `internal/collection` | Unit | Table-driven; fixture log files in `testdata/` |
| `internal/analytics` | Unit | Property tests on Metrics invariants (ratios sum to ≤ 1) |
| `internal/insights` | Unit | Threshold tests from TRD §5; persona assignment coverage |
| `internal/trust` | Unit | Round-trip sign → verify; canonical JSON parity vs TS output |
| `internal/team` | Unit | ParseTeamConfig regex; Dedupe ordering; MergeMetrics commutativity |
| `internal/reconciliation` | Unit | Compare with synthetic local/api data; breach detection |
| `internal/deepanalysis` | Unit | SessionStat correctness; LlmPayload privacy-boundary check |
| `infrastructure/sqlite` | Integration | Temp DB per test via `t.TempDir()`; real DDL + queries |
| `infrastructure/adapters/*` | Integration | Fixture log files + fixture SQLite DBs in `testdata/` |
| `infrastructure/providers/*` | Integration | `httptest.NewServer` mock for provider API |
| `infrastructure/render` | Snapshot | Compare terminal/HTML output against golden files |
| `application/*` | Integration | In-memory fake ports (not mocks) for EventReader/Writer |
| `cmd/` | E2E | Subprocess exec of real binary against temp dir + fixture data |
