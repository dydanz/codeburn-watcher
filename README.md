# codeburn-watcher

CLI tool for monitoring AI coding agent token and cost usage across multiple providers — locally, privately, without sending data to any third party.

Tracks Claude Code, Gemini CLI, Codex, GitHub Copilot, Cursor, and Antigravity. Stores events in a local SQLite database and renders terminal dashboards, HTML reports, and signed team exports.

---

## Features

| Capability | Details |
|---|---|
| **Multi-agent collection** | Claude Code · Gemini CLI · Codex · Copilot · Cursor · Antigravity |
| **Terminal dashboard** | Token counts, cost, cache hit ratio, activity breakdown, top recs |
| **HTML report** | Dark-themed dashboard with ratio bars and project movers |
| **Deep analysis** | Session cost ranking, fix-iteration detection, context bloat, tool error rates |
| **LLM pipe** | Stream deep-analysis payload to a local agent (`--llm`) |
| **Team exports** | Ed25519-signed JSON exports; merge + verify from multiple members |
| **Reconciliation** | Compare local token counts against Anthropic/OpenAI billing APIs |
| **Background collection** | Auto-installs launchd (macOS) or cron (Linux) scheduler |
| **Pure Go** | Zero CGO — `modernc.org/sqlite`, builds to a single static binary |

---

## Architecture

DDD + Hexagonal (Ports & Adapters). 7 bounded contexts, each isolated behind port interfaces:

```
internal/
  shared/          # StoredEvent — stable cross-BC contract
  collection/      # BC1 Telemetry — AgentAdapter port, ActivityClassifier
  analytics/       # BC2 Usage Analytics — MetricsEngine, PricingService
  insights/        # BC3 Insights — RecommendationEngine, TrendAnalyzer, Persona
  trust/           # BC4 Trust & Identity — Ed25519 signing, CanonicalizationService
  team/            # BC5 Team Collaboration — SignedExport, ExportBuilder, ExportMerger
  reconciliation/  # BC6 Reconciliation — ReconciliationService, ProviderUsage
  deepanalysis/    # BC7 Deep Analysis — SessionAnalyzer, ToolAnalyzer, LlmPipeline

infrastructure/
  adapters/        # Agent log readers (claudecode, geminicli, codex, copilot, cursor, antigravity)
  providers/       # Billing API adapters (Anthropic, OpenAI)
  sqlite/          # EventStore, FollowThroughStore
  fs/              # Keystore, DeployConfig
  http/            # ExportTransport (multipart push + fetch)
  scheduler/       # Launchd (macOS) / Cron (Linux)
  render/          # TerminalRenderer, HtmlRenderer

application/       # Command/query handlers wiring domain to adapters
cmd/token-monitor/ # Cobra CLI entrypoint + dependency wiring
```

---

## Requirements

- Go 1.25+
- No CGO required

---

## Install

```bash
go install github.com/dydanz/codeburn-watcher/cmd/token-monitor@latest
```

Or build from source:

```bash
git clone https://github.com/dydanz/codeburn-watcher
cd codeburn-watcher
go build -o token-monitor ./cmd/token-monitor/...
```

---

## Quick start

```bash
# 1. First-time setup — generate Ed25519 keypair, run first collect, install scheduler
token-monitor init

# 2. Manual collect (scanner runs all adapters)
token-monitor collect

# 3. Terminal report (default 7-day window)
token-monitor report
token-monitor report --days 30

# 4. HTML dashboard
token-monitor html --out report.html --days 30

# 5. Deep analysis
token-monitor analyze --days 14

# 6. Pipe analysis payload to local LLM agent
token-monitor analyze --llm --days 14

# 7. Reconcile against provider billing API
export ANTHROPIC_ADMIN_KEY=sk-ant-admin-...
token-monitor reconcile --provider anthropic --days 7

# 8. Print your signing fingerprint (share with team lead)
token-monitor fingerprint
```

---

## Team workflow

```bash
# Push signed export to team server
token-monitor push --days 7

# Merge exports from team members and render team report
token-monitor merge alice.json bob.json charlie.json

# Verify signatures before merging
token-monitor merge --verify --keys keyring.json alice.json bob.json
```

---

## Scheduler

```bash
# Install background collection (every 10 min)
token-monitor schedule

# Remove
token-monitor schedule --remove
```

On macOS: writes `~/Library/LaunchAgents/com.token-monitor.collect.plist`.
On Linux: appends to crontab (`*/10 * * * *`).

---

## Run with Docker

**Build and run all tests:**

```bash
docker build --target tester -t codeburn-watcher:test .
```

**Build runtime image:**

```bash
docker build -t codeburn-watcher:latest .
```

**Collect from your real agent logs:**

```bash
docker run --rm \
  -v "$HOME/.claude:/root/.claude:ro" \
  -v "$HOME/.gemini:/root/.gemini:ro" \
  -v "$HOME/.token-monitor:/root/.token-monitor" \
  codeburn-watcher:latest collect
```

**Terminal report:**

```bash
docker run --rm \
  -v "$HOME/.token-monitor:/root/.token-monitor:ro" \
  codeburn-watcher:latest report --days 30
```

**HTML report written to your Desktop:**

```bash
docker run --rm \
  -v "$HOME/.token-monitor:/root/.token-monitor:ro" \
  -v "$HOME/Desktop:/out" \
  codeburn-watcher:latest html --days 30 --out /out/token-report.html
```

| Volume | Purpose |
|---|---|
| `$HOME/.claude:/root/.claude:ro` | Claude Code session logs |
| `$HOME/.gemini:/root/.gemini:ro` | Gemini CLI chat logs |
| `$HOME/.codex:/root/.codex:ro` | Codex JSONL logs |
| `$HOME/.token-monitor:/root/.token-monitor` | SQLite DB + keypair (read-write) |

---

## Development

```bash
# Build
go build ./...

# All tests
go test ./...

# Unit + integration only (skip E2E subprocess tests)
go test -short ./...

# Vet
go vet ./...
```

Test coverage:
- `internal/` — unit tests for all 7 BCs
- `infrastructure/` — integration tests with `t.TempDir()` SQLite and `httptest.NewServer` mocks
- `application/` — handler tests with in-memory fake ports (no disk I/O)
- `cmd/token-monitor/` — E2E subprocess tests: `collect → report` pipeline, HTML output, error paths

---

## Data privacy

All data stays local. The SQLite database lives at `~/.token-monitor/events.db`. Nothing is sent to any remote service unless you explicitly run `token-monitor push` with a configured `push_url`.

---

## Design references

| Document | Contents |
|---|---|
| `trd/domain-model.md` | Aggregates, entities, value objects |
| `trd/hexagonal-architecture.md` | Port interfaces + adapter contracts |
| `trd/application-layer.md` | Commands, queries, domain events |
| `trd/go-package-layout.md` | Concrete package structure |
| `trd/context-map.md` | BC relationships and data flows |
