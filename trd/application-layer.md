# Application Layer

The application layer orchestrates domain services and port calls for each
CLI use case. Handlers contain no business logic — they wire together the
domain and call ports. Each handler corresponds to one CLI sub-command.

---

## Domain Events

Domain events document significant state changes. In the initial Go port they
are plain structs propagated via direct function return values (no event bus).

```go
// Package: internal/shared/events

// UsageEventsCollected is emitted after a collection run.
type UsageEventsCollected struct {
    Source   shared.Source
    Found    int
    Inserted int
    OccurredAt time.Time
}

// MetricsComputed is emitted after the analytics engine runs.
type MetricsComputed struct {
    Window     analytics.QueryWindow
    Metrics    analytics.Metrics
    OccurredAt time.Time
}

// RecommendationsEnriched is emitted when recommendations are ready.
type RecommendationsEnriched struct {
    Recs         []insights.EnrichedRecommendation
    PotentialBill insights.PotentialBill
    OccurredAt   time.Time
}

// ExportSigned is emitted when a signed export is produced.
type ExportSigned struct {
    Username    string
    Fingerprint trust.Fingerprint
    GeneratedAt string
    OccurredAt  time.Time
}

// ReconciliationBreached is emitted when local tokens exceed provider billed.
type ReconciliationBreached struct {
    Provider    string
    BreachRows  []reconciliation.ReconcileRow
    OccurredAt  time.Time
}
```

---

## Commands & Queries

### Command: `Collect`

```go
// application/collect.go

type CollectCommand struct {
    // Optionally restrict to one source; empty = run all adapters.
    Source shared.Source
}

type CollectHandler struct {
    Adapters   []collection.AgentAdapter // all six, or filtered by cmd.Source
    Writer     shared.ports.EventWriter
    Classifier collection.ActivityClassifier
}

func (h CollectHandler) Handle(ctx context.Context, cmd CollectCommand) ([]events.UsageEventsCollected, error) {
    var allEvents []collection.UsageEvent
    var results []events.UsageEventsCollected
    for _, adapter := range h.Adapters {
        evs, result, err := adapter.Collect(ctx)
        if err != nil {
            // fail-soft: log to stderr, continue with other adapters
            fmt.Fprintf(os.Stderr, "warning: %s collect failed: %v\n", result.Source, err)
            continue
        }
        allEvents = append(allEvents, evs...)
        results = append(results, events.UsageEventsCollected{
            Source: result.Source, Found: result.Found, Inserted: result.Inserted,
            OccurredAt: time.Now(),
        })
    }
    inserted, err := h.Writer.WriteEvents(ctx, allEvents)
    if err != nil {
        return nil, fmt.Errorf("writing events: %w", err)
    }
    // Reconcile inserted count against per-adapter totals for the summary line.
    return results, nil
}
```

### Query: `Report`

```go
// application/report.go

type ReportQuery struct {
    Window  analytics.QueryWindow
    JSON    bool   // --json flag: emit ExportV1 instead of terminal report
    OutFile string // --out: write to file instead of stdout
}

type ReportDeps struct {
    Reader        shared.ports.EventReader
    MetricsEngine analytics.MetricsEngine
    Recommender   insights.RecommendationEngine
    Trends        insights.TrendAnalyzer
    FollowThrough insights.FollowThroughService
    Persona       insights.PersonaAssigner
    ExportBuilder team.ExportBuilder
    Identity      trust.SigningIdentity
    Renderer      render.TerminalRenderer
    HtmlRenderer  render.HtmlRenderer // used by html sub-command
}

type ReportHandler struct{ Deps ReportDeps }

func (h ReportHandler) Handle(ctx context.Context, q ReportQuery) error {
    evs, err := h.Deps.Reader.LoadEvents(ctx, q.Window)
    if err != nil { return err }

    m := h.Deps.MetricsEngine.Compute(evs)
    // emit MetricsComputed domain event (log/debug only in initial port)

    recs, bill := h.Deps.Recommender.Enrich(evs, m, q.Window.Days)
    trends, movers := h.Deps.Trends.ComputeTrends(evs, q.Window.Days, h.Deps.MetricsEngine)
    followThrough, err := h.Deps.FollowThrough.Sync(recs, m)
    if err != nil { return err }
    persona := h.Deps.Persona.Assign(m)

    if q.JSON {
        payload := team.ExportPayload{
            Days: q.Window.Days, Overall: m,
            Recommendations: recs, Trends: trends, ProjectMovers: movers,
            FollowThrough: followThrough, Persona: persona,
        }
        export, err := h.Deps.ExportBuilder.Build(h.Deps.Identity, payload)
        if err != nil { return err }
        return writeJSON(export, q.OutFile)
    }

    output := h.Deps.Renderer.RenderReport(m, recs, trends, movers, q.Window.Days)
    return writeText(output, q.OutFile)
}
```

### Query: `Analyze`

```go
// application/analyze.go

type AnalyzeQuery struct {
    Window analytics.QueryWindow
    LLM    bool   // --llm: pipe payload to local agent CLI
    Agent  string // --agent: override auto-detected agent
    JSON   bool   // --json: emit raw payload JSON
}

type AnalyzeHandler struct {
    Reader      shared.ports.EventReader
    Metrics     analytics.MetricsEngine
    Sessions    deepanalysis.SessionAnalyzer
    Tools       deepanalysis.ToolAnalyzer
    Recommender insights.RecommendationEngine
    Persona     insights.PersonaAssigner
    Pipeline    deepanalysis.LlmPipeline
    Renderer    render.TerminalRenderer
}

func (h AnalyzeHandler) Handle(ctx context.Context, q AnalyzeQuery) error {
    evs, err := h.Reader.LoadEvents(ctx, q.Window)
    if err != nil { return err }

    m := h.Metrics.Compute(evs)
    deep := h.Sessions.DeepAnalyze(evs)
    recs, _ := h.Recommender.Enrich(evs, m, q.Window.Days)

    if q.LLM || q.JSON {
        payload := h.Pipeline.BuildPayload(evs, m, recs, q.Window.Days)
        if q.JSON {
            return writeJSON(payload, "")
        }
        prompt := h.Pipeline.BuildPrompt(payload)
        agent, ok := h.Pipeline.DetectAgent()
        if !ok {
            return fmt.Errorf("no supported LLM agent found on PATH (claude, gemini, codex)")
        }
        os.Exit(h.Pipeline.Run(prompt, agent))
    }

    output := h.Renderer.RenderAnalysis(deep, m, recs, q.Window.Days)
    fmt.Print(output)
    return nil
}
```

### Command: `Html`

```go
// application/html.go

type HtmlCommand struct {
    Window  analytics.QueryWindow
    OutFile string // required: --out <path>
    Team    bool   // --team: render team rollup HTML
}

// HtmlHandler.Handle:
//   LoadEvents → MetricsEngine.Compute → Recommender.Enrich → Trends.ComputeTrends
//   → HtmlRenderer.RenderHtml → write to OutFile
//   (if --team: also loads and merges exports, calls RenderTeamHtml)
```

### Command: `Merge`

```go
// application/merge.go

type MergeCommand struct {
    Files    []string // positional: export JSON files
    Verify   bool     // --verify: verify all signatures
    KeysFile string   // --keys: path to keyring JSON
    HTML     bool     // --html: render HTML team report
    OutFile  string   // --out: write rendered output here
}

type MergeHandler struct {
    Verifier  trust.VerificationService
    Keystore  trust.ports.KeystorePort
    Merger    team.ExportMerger
    Renderer  render.TerminalRenderer
    HtmlRenderer render.HtmlRenderer
}

func (h MergeHandler) Handle(ctx context.Context, cmd MergeCommand) error {
    exports, err := loadExportFiles(cmd.Files) // parse JSON → []team.SignedExport
    if err != nil { return err }

    if cmd.Verify {
        var keyring trust.Keyring
        if cmd.KeysFile != "" {
            keyring, err = h.Keystore.LoadKeyring(cmd.KeysFile)
            if err != nil { return err }
        }
        for _, e := range exports {
            result := h.Verifier.VerifyWithKeyring(e, keyring, e.Username)
            if !result.Valid {
                return fmt.Errorf("export from %s: %s", e.Username, result.Error)
            }
        }
    }

    deduped := h.Merger.Dedupe(exports)
    // For team report: build rollup from TeamConfig (loaded from DeployConfig or --keys).
    rollups := h.Merger.Rollup(deduped, teamConfig)

    if cmd.HTML {
        html := h.HtmlRenderer.RenderTeamHtml(rollups, days)
        return writeFile(cmd.OutFile, html)
    }
    output := h.Renderer.RenderTeamReport(rollups, days)
    fmt.Print(output)
    return nil
}
```

### Command: `Init`

```go
// application/init.go

type InitCommand struct {
    TeamConfigURL string // --team: URL to fetch team config from
}

// InitHandler.Handle:
//   1. FetchConfig(url) → parse TeamConfig → save DeployConfig
//   2. EnsureKeypair(~/.token-monitor/) → derive Fingerprint
//   3. Print fingerprint for enrollment ("Share this with your team lead: <fingerprint>")
//   4. CollectHandler.Handle (first collection run)
//   5. SchedulerPort.Install (launchd or cron)
```

### Command: `Push`

```go
// application/push.go

type PushCommand struct {
    Window analytics.QueryWindow
}

// PushHandler.Handle:
//   1. ReportHandler.Handle with JSON=true → SignedExport bytes
//   2. Derive filename from ExportFilename(username, generatedAt) (TRD §10)
//   3. ExportTransportPort.Push(filename, bytes, pushURL)
```

### Command: `Schedule`

```go
// application/schedule.go

type ScheduleCommand struct {
    Remove bool // --remove: uninstall instead of install
}

// ScheduleHandler.Handle:
//   if Remove: SchedulerPort.Remove()
//   else: SchedulerPort.Install(os.Executable(), deployConfig)
```

### Command: `Reconcile`

```go
// application/reconcile.go

type ReconcileCommand struct {
    Provider string
    Days     int // capped at 31
    Env      func(string) string // os.Getenv in prod, stub in tests
}

type ReconcileHandler struct {
    Reader   shared.ports.EventReader
    Providers map[string]reconciliation.ProviderSpec
    Service  reconciliation.ReconciliationService
    Renderer render.TerminalRenderer
}

func (h ReconcileHandler) Handle(ctx context.Context, cmd ReconcileCommand) (exitCode int, err error) {
    spec, ok := h.Providers[cmd.Provider]
    if !ok {
        return 1, fmt.Errorf(`unknown provider "%s". Supported: %s`,
            cmd.Provider, strings.Join(supportedProviders(h.Providers), ", "))
    }
    key := cmd.Env(spec.KeyEnv)
    if key == "" {
        return 1, fmt.Errorf("%s is not set. ...", spec.KeyEnv)
    }
    endMs := time.Now().UnixMilli()
    startMs := endMs - int64(cmd.Days)*86_400_000
    baseURL := strings.TrimSuffix(cmd.Env(spec.URLEnv), "/")
    if baseURL == "" { baseURL = spec.DefaultURL }

    evs, err := h.Reader.LoadEvents(ctx, analytics.QueryWindow{Days: cmd.Days})
    if err != nil { return 1, err }

    providerData, err := spec.Adapter.FetchUsage(ctx, baseURL, key, startMs, endMs)
    if err != nil { return 1, err }

    result := h.Service.Compare(evs, providerData)
    fmt.Print(h.Renderer.RenderReconcile(cmd.Provider, result, cmd.Days))

    if result.HasBreach() { return 1, nil }
    return 0, nil
}
```

### Query: `Fingerprint`

```go
// application/fingerprint.go

// FingerprintHandler.Handle:
//   1. KeystorePort.LoadKeypair(~/.token-monitor/)
//   2. Derive Fingerprint
//   3. Print to stdout
```

---

## Dependency wiring (`cmd/token-monitor/wire.go`)

All dependencies are assembled in `main.go` (or a `wire.go` helper). No
dependency injection framework — plain Go constructor calls.

```go
func buildDeps(dbPath, keyDir string) (deps AppDeps, err error) {
    db, err := sqlite.Open(dbPath)
    if err != nil { return deps, err }

    eventStore := sqlite.NewEventStore(db)
    followStore := sqlite.NewFollowThroughStore(db)
    pricing := analytics.NewPricingService() // loads static PRICES table
    keystore := fs.NewFileKeystore()

    identity, err := keystore.EnsureKeypair(keyDir)
    if err != nil { return deps, err }

    return AppDeps{
        CollectHandler: application.CollectHandler{
            Adapters: adapters.AllAdapters(),
            Writer:   eventStore,
        },
        ReportHandler: application.ReportHandler{Deps: application.ReportDeps{
            Reader:        eventStore,
            MetricsEngine: analytics.NewMetricsEngine(pricing),
            Recommender:   insights.NewRecommendationEngine(pricing),
            Trends:        insights.TrendAnalyzer{},
            FollowThrough: insights.NewFollowThroughService(followStore),
            Persona:       insights.PersonaAssigner{},
            ExportBuilder: team.NewExportBuilder(trust.CanonicalizationService{}),
            Identity:      identity,
            Renderer:      render.TerminalRenderer{},
        }},
        ReconcileHandler: application.ReconcileHandler{
            Reader:    eventStore,
            Providers: reconciliation.ProviderRegistry,
            Service:   reconciliation.ReconciliationService{},
            Renderer:  render.TerminalRenderer{},
        },
        // ... other handlers
    }, nil
}
```

All application handlers are constructed with their dependencies injected —
no global state, no `init()` side effects, full testability.
