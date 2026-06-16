# token-monitor — DDD Redesign

Source of truth: [TRD.md](../TRD.md) + [docs/trd/](../docs/trd/)

## Why DDD

The original TypeScript source is organized by **technical concern** (adapters,
metrics, rendering, sign, team, deploy). This works for a single developer but
makes the following hard:

- Adding a new adapter without risking metric logic
- Testing business rules (recommendation thresholds, canonicalization) in
  isolation from I/O
- Swapping SQLite for a different store without touching domain logic
- Reasoning about what data crosses privacy boundaries

DDD reorganizes the same logic by **business concern** (bounded context),
separating domain rules from infrastructure and making each boundary explicit
via port interfaces. The Go port is the natural moment to apply this structure.

## Seven Bounded Contexts

| # | Bounded Context | Core responsibility |
|---|---|---|
| 1 | **Telemetry Collection** | Ingest raw agent logs → normalized `UsageEvent` |
| 2 | **Usage Analytics** | Compute `Metrics` from a window of stored events |
| 3 | **Insights** | Produce recommendations, trends, and personas from Metrics |
| 4 | **Trust & Identity** | Sign and verify exports via Ed25519 keypair |
| 5 | **Team Collaboration** | Build, merge, deduplicate, and roll up exports |
| 6 | **Reconciliation** | Cross-check local totals against provider billing APIs |
| 7 | **Deep Analysis** | Per-session behavioral stats + LLM prompt pipeline |

A separate **Presentation** layer (terminal renderer, HTML dashboard) is
infrastructure — not a BC — and sits in the hexagonal adapter ring.

## Design decisions

**DDD + Hexagonal (Ports & Adapters).** Each BC owns:
- A **domain layer** (aggregates, entities, value objects, domain services) —
  pure Go, no I/O, no framework imports.
- **Port interfaces** (Go `interface` types) that the domain defines and
  infrastructure implements.
- No dependency on other BCs except through well-defined contracts.

**No event bus.** The CLI is synchronous. Domain Events are defined in
[application-layer.md](application-layer.md) as plain structs for clarity and
potential future extension, but in the initial Go port they are propagated
via direct function calls, not a pub/sub mechanism.

**Shared Kernel: `StoredEvent`.** The `StoredEvent` struct (source, session_id,
project, branch, model, tools[], activity, timestamps, token counts, is_error)
is the stable contract between Collection (writes) and all downstream BCs
(reads). It is defined once in a `shared/` package imported by all BCs that
need it. Changes to `StoredEvent` require coordinated updates across BCs.

**Value Objects are immutable.** All Go structs acting as value objects have
no exported setter methods. They are constructed via factory functions and
compared structurally.

## File map

| File | Covers |
|---|---|
| [context-map.md](context-map.md) | BC relationships, integration patterns, data flows |
| [ubiquitous-language.md](ubiquitous-language.md) | Per-BC glossary |
| [domain-model.md](domain-model.md) | Aggregates, entities, value objects, domain services per BC |
| [hexagonal-architecture.md](hexagonal-architecture.md) | Port interfaces + adapter implementations |
| [application-layer.md](application-layer.md) | Commands, queries, handlers, domain events |
| [go-package-layout.md](go-package-layout.md) | Concrete Go package/file structure |
