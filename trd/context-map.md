# Context Map

## Bounded Context diagram

```
 ┌─────────────────────────────────────────────────────────────────────────┐
 │                         TOKEN-MONITOR                                    │
 │                                                                          │
 │  ┌──────────────────┐    StoredEvent    ┌───────────────────────────┐   │
 │  │  1. Telemetry    │ ─── Shared ────►  │  2. Usage Analytics       │   │
 │  │     Collection   │    Kernel         │  (MetricsSnapshot)        │   │
 │  │                  │                   └──────────┬────────────────┘   │
 │  │  Adapters:       │                              │ Metrics             │
 │  │  · claude-code   │                              ▼                    │
 │  │  · gemini-cli    │                   ┌───────────────────────────┐   │
 │  │  · codex         │                   │  3. Insights              │   │
 │  │  · copilot       │                   │  (Recs + Trends +         │   │
 │  │  · cursor        │                   │   Persona + FollowThrough)│   │
 │  │  · antigravity   │                   └───────────┬───────────────┘   │
 │  └──────────────────┘                               │                   │
 │          │ StoredEvent                              │ EnrichedRec[]      │
 │          │                     ┌────────────────────┤ TrendReport        │
 │          │                     │                    │ Persona            │
 │          ▼ (shared kernel)     ▼                    ▼                   │
 │  ┌──────────────────┐   ┌─────────────────────────────────────────┐    │
 │  │  6. Reconcil-    │   │  5. Team Collaboration                   │    │
 │  │     iation       │   │  (ExportV1 build + merge + rollup)       │    │
 │  │                  │   │                                          │    │
 │  │  Upstream:       │   │  Needs: Trust & Identity (signing)       │    │
 │  │  · Anthropic API │   └───────────────┬─────────────────────────┘    │
 │  │  · OpenAI API    │                   │ uses (downstream)             │
 │  └──────────────────┘                   ▼                              │
 │                              ┌───────────────────────────┐             │
 │                              │  4. Trust & Identity      │             │
 │                              │  (Ed25519 sign/verify)    │             │
 │                              └───────────────────────────┘             │
 │                                                                          │
 │  ┌──────────────────────────────────────────────────────────────────┐  │
 │  │  7. Deep Analysis   (SessionStat + ToolStat + LLM pipeline)      │  │
 │  │  Reads: StoredEvent (shared kernel)                              │  │
 │  └──────────────────────────────────────────────────────────────────┘  │
 │                                                                          │
 │  ┌──────────────────────────────────────────────────────────────────┐  │
 │  │  Presentation (infrastructure — not a BC)                        │  │
 │  │  · TerminalRenderer  · HtmlRenderer  · VSCodeBridge              │  │
 │  └──────────────────────────────────────────────────────────────────┘  │
 └─────────────────────────────────────────────────────────────────────────┘
```

## Integration patterns

### Shared Kernel: `StoredEvent`

`StoredEvent` is the stable cross-BC data contract. It is defined in
`internal/shared/event.go` and imported read-only by BCs 2, 3, 6, and 7.
BC 1 (Collection) is the only writer.

**Shared Kernel invariant**: the `StoredEvent` schema (columns, JSON tags,
deduplication key) changes only with deliberate coordination across all
consuming BCs. Any schema migration requires updating the SQLite DDL, the
`loadEvents` query, all adapter `Collect` functions, and downstream consumers.

### Upstream / Downstream

| Upstream BC | Downstream BC | Data crossing boundary |
|---|---|---|
| Telemetry Collection (1) | Usage Analytics (2) | `[]StoredEvent` (via `EventReader` port) |
| Telemetry Collection (1) | Reconciliation (6) | `[]StoredEvent` (via `EventReader` port) |
| Telemetry Collection (1) | Deep Analysis (7) | `[]StoredEvent` (via `EventReader` port) |
| Usage Analytics (2) | Insights (3) | `Metrics` struct (in-process, direct call) |
| Usage Analytics (2) | Team Collaboration (5) | `Metrics` + `ExportData` (in-process) |
| Insights (3) | Team Collaboration (5) | `[]EnrichedRecommendation`, `TrendReport`, `Persona` |
| Trust & Identity (4) | Team Collaboration (5) | `SignedExport` (Team is **conformist** to Trust's signing contract) |

### Anti-Corruption Layer (ACL)

**Reconciliation ↔ Provider APIs**: Provider APIs (Anthropic Admin, OpenAI
Usage) use different field names, different token semantics (OpenAI
`input_tokens` includes cached; Anthropic separates them), different time
formats (ISO vs unix-seconds), and different pagination schemes. The `ProviderUsageAdapter`
ACL translates both APIs into the BC-internal `ProviderUsage` value object
before the domain sees any data.

**Collection ↔ Agent Log Formats**: Each agent writes its own idiosyncratic
format (JSONL, JSON chat history, binary protobuf-encoded SQLite, nested JSON,
estimated-token pseudo-events). The six `AgentAdapter` implementations
are ACLs that normalize into `UsageEvent` before anything domain-related runs.

### Open Host Service

The **CLI** is the Open Host Service: a stable external API (flags, sub-command
names, stdout format) that the VS Code extension, shell scripts, and cron/launchd
jobs depend on. Changes to the CLI surface require backward compatibility
consideration across all consumers.

## Data flows by command

### `collect`
```
[agent log files / agent DBs]
  → AgentAdapter.Collect()          (BC1: Collection)
  → ActivityClassifier.Classify()   (BC1: domain service)
  → EventWriter.Write([]UsageEvent) (port → SQLite adapter)
```

### `report --json`
```
  EventReader.LoadEvents(window)    (port → SQLite)
  → MetricsEngine.Compute()         (BC2: Analytics)
  → PersonaAssigner.Assign()        (BC3: Insights)
  → RecommendationEngine.Enrich()   (BC3: Insights)
  → TrendAnalyzer.Compute()         (BC3: Insights)
  → FollowThroughService.Sync()     (BC3: Insights, writes recommendations table)
  → ExportBuilder.Build()           (BC5: Team Collaboration)
  → CanonicalizationService.Canon() (BC4: Trust & Identity)
  → SigningService.Sign()           (BC4: Trust & Identity)
  → [stdout: signed ExportV1 JSON]
```

### `merge --verify --keys`
```
  [signed ExportV1 JSON files]
  → VerificationService.VerifyWithKeyring()  (BC4: Trust & Identity)
  → ExportMerger.Dedupe()                    (BC5: Team Collaboration)
  → ExportMerger.MergeMetrics()              (BC5)
  → ExportMerger.Rollup()                    (BC5)
  → TerminalRenderer.RenderTeamReport()      (Presentation)
```

### `reconcile`
```
  EventReader.LoadEvents(window)                 (port → SQLite)
  → ProviderUsageAdapter.FetchUsage()            (BC6: port → provider ACL)
  → ReconciliationService.Compare()              (BC6: domain service)
  → TerminalRenderer.RenderReconcile()           (Presentation)
  → exit 1 if ReconciliationResult.Breach == true
```

### `analyze [--llm]`
```
  EventReader.LoadEvents(window)             (port → SQLite)
  → MetricsEngine.Compute()                 (BC2)
  → SessionAnalyzer.DeepAnalyze()           (BC7)
  → ToolAnalyzer.ComputeToolStats()         (BC7)
  → RecommendationEngine.Enrich()           (BC3)
  → TerminalRenderer.RenderAnalysis()       (Presentation)
  if --llm:
    → LlmPipeline.BuildPayload()            (BC7)
    → LlmPipeline.BuildPrompt()             (BC7)
    → LlmPipeline.DetectAgent()             (BC7)
    → LlmPipeline.Run(prompt, agent)        (BC7 → OS exec)
```
