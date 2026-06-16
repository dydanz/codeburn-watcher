# codeburn-watcher

Go CLI for monitoring AI agent token/cost usage across multiple providers (Claude Code, Gemini CLI, Codex, Copilot, Cursor).

## Stack

- **Language:** Go (target 1.24+)
- **Architecture:** DDD + Hexagonal (Ports & Adapters)
- **Design docs:** `trd/` — 7 bounded contexts + shared kernel
- **Operator:** Dandi (solo)

## Bounded Contexts

| BC | Component label |
|---|---|
| Telemetry Collection | `component/telemetry` |
| Usage Analytics | `component/analytics` |
| Insights | `component/insights` |
| Trust & Identity | `component/identity` |
| Team Collaboration | `component/collaboration` |
| Reconciliation | `component/reconciliation` |
| Deep Analysis | `component/analysis` |
| Shared Kernel | `component/shared` |
| Presentation (adapters) | `component/presentation` |

## SDLC

See `.claude/agents/sdlc.md`. 5-phase flow: Intake → Spec → Implement → Review → Merge.

## Key invariants

- `StoredEvent` in `internal/shared/` is the stable contract between Collection and all downstream BCs. Changes require coordinated updates across all BCs that read it.
- Value objects are immutable — no exported setters, factory functions only.
- Domain layer has zero I/O imports. All I/O lives in adapters.
- No dependency between BCs except through port interfaces.

## Commands

```bash
go build ./...
go test ./...
go vet ./...
```

## Design references

- `trd/domain-model.md` — aggregates, entities, value objects
- `trd/hexagonal-architecture.md` — port interfaces + adapters
- `trd/application-layer.md` — commands, queries, domain events
- `trd/go-package-layout.md` — concrete package structure
- `trd/context-map.md` — BC relationships and data flows
