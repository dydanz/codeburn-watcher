# Ubiquitous Language

Each section defines the terms used within that Bounded Context. The same
English word may mean different things in different BCs — where this happens
the BC prefix disambiguates (e.g., "Collection::Event" vs. "Analytics::Event").

---

## BC 1 — Telemetry Collection

| Term | Definition |
|---|---|
| **Agent** | An AI coding assistant whose activity is being tracked: Claude Code, Gemini CLI, Codex, GitHub Copilot Chat, Cursor, or Antigravity. |
| **Log Source** | The physical artifact an Agent writes to (a JSONL file, a SQLite database, a JSON directory), from which the adapter reads. |
| **Turn** | One request/response exchange between the developer and the Agent. A Turn has a tool use (or none), token counts, and optionally an error. |
| **Adapter** | The component that reads one Agent's Log Source and produces normalized `UsageEvent`s. There is exactly one Adapter per Agent. |
| **UsageEvent** | A fully normalized, immutable record of one Turn, ready for storage. Contains: Source, SessionID, Project, Branch, Model, Tools, Activity, token counts, IsError, and Timestamp. |
| **Source** | An enum identifying which Agent produced a UsageEvent: `claude-code`, `gemini-cli`, `codex`, `copilot`, `cursor`, `antigravity`. |
| **Activity** | A classification of what the Turn was doing: `thinking`, `exploration`, `coding`, `testing`, `shipping`, `conversation`. |
| **DeduplicationKey** | The combination of Source + EventKey that uniquely identifies a Turn across all ingestion runs. Enables idempotent collection. |
| **CollectResult** | Summary returned after a collection run: how many events were found and how many were new (inserted vs. already present). |
| **ColdRestart** | A Turn that resumes a session after a gap longer than the cache TTL (5 minutes). The developer must re-pay input tokens that the cache would have covered had the session continued. |

---

## BC 2 — Usage Analytics

| Term | Definition |
|---|---|
| **QueryWindow** | The time window + optional project and source filters over which Metrics are computed. Defined by (days, projectFilter, sourceFilter). |
| **StoredEvent** | A UsageEvent as retrieved from the SQLite store. Identical schema to UsageEvent; the distinction is that StoredEvent has been persisted and is the input to all analytics operations. |
| **Metrics** | The complete computed snapshot for a QueryWindow: token totals, cost, ratios (cacheHitRatio, reworkRatio, thinkToCodeRatio, contextBloatShare, coldRestartShare, premiumWasteShare, retryShare), breakdowns by Model and Activity, session/turn counts. |
| **SpendTokens** | Input + Output tokens (excludes cache-read and cache-creation from the "spend" total used for cost comparisons — though cost includes all four). |
| **CacheHitRatio** | Cache-read tokens / (input tokens + cache-read tokens + cache-creation tokens). A high ratio means the developer is reusing context effectively. |
| **ReworkRatio** | Tokens spent on Turns after the first error within a session / total SpendTokens. Measures how much time is lost to fix loops. |
| **ContextBloatShare** | Share of long sessions (≥ BloatMinTurns) where context grew ≥ 2× from early to late half without the cache keeping pace. |
| **ColdRestartShare** | Tokens re-paid on ColdRestart Turns / all fresh-paid input tokens. Measures waste from letting sessions go cold. |
| **PremiumWasteShare** | Premium-model tokens on exploration/conversation turns / all SpendTokens. Measures using an expensive model for low-value activities. |
| **RetryShare** | Tokens on Turns that re-run a tool immediately after that tool errored / total SpendTokens. |
| **ContextGrowth** | Late-half average context per Turn / early-half average context per Turn. Growth ≥ 2× flags a bloating session. |
| **ModelPrice** | A single entry in the static pricing table: model name-prefix, input/output/cacheRead/cacheWrite cost per million tokens. |
| **Cost** | Estimated USD cost for a set of token counts, computed from ModelPrice. Includes flags for whether the model was unpriced (CostEstimated) and how many tokens had no price (CostUnpricedTokens). |

---

## BC 3 — Insights

| Term | Definition |
|---|---|
| **Persona** | A behavioral archetype assigned to a developer or project based on Metrics: Firefighter, Explorer, Sprinter, Surgeon, Architect, or Balanced. Each has a distinct token-usage pattern and tailored guidance. |
| **Finding** | A specific inefficiency pattern detected in the Metrics: one of 8 named findings (e.g., "cache-miss", "rework", "premium-waste"). A Finding has a key, a human-readable label, and a predicate that evaluates to true/false against Metrics. |
| **EnrichedRecommendation** | A Finding that has been confirmed active (predicate = true) and augmented with: quantified evidence (current metric value, improvement target, estimated $/month savings). |
| **Target** | The goal value for a metric that an EnrichedRecommendation points toward. Can be personalized (derived from the developer's own top-quartile sessions) or static (from the global TARGETS table). |
| **PotentialBill** | The estimated total monthly savings if all active recommendations were acted on simultaneously. Displayed as the motivating headline of the recommendations section. |
| **Trend** | The direction of change for a metric comparing the current window to the previous equal-length window. Verdict is one of: up (improving), down (regressing), flat (no significant change). |
| **TrendRow** | One metric's trend: (metricKey, currentValue, previousValue, delta, verdict, formatted strings). |
| **ProjectMover** | A project whose spend changed significantly between windows. Surfaces which projects are growing or shrinking in token consumption. |
| **FollowThroughRecord** | A persistent record tracking whether a Finding is being acted on over time. Fields: metricKey, baseline value, current value, status (improving / regressing / resolved). |
| **BlendedRate** | The cost-per-token rate averaged across all models and token types weighted by their share of spend in the window. Used to convert token savings into dollar savings. |
| **AvoidableTokens** | The tokens that could have been saved by resolving a specific Finding. For bloat: tokens above the 2× growth threshold. For premium waste: premium tokens on low-value activities. |

---

## BC 4 — Trust & Identity

| Term | Definition |
|---|---|
| **SigningIdentity** | The aggregate root: one machine's persistent cryptographic identity, consisting of a Keypair and its derived Fingerprint. Created once per machine on `init`. |
| **Keypair** | An Ed25519 private + public key pair. The private key is stored at `~/.token-monitor/signing-key.pem` (mode 0600, PKCS8 PEM); the public key at `~/.token-monitor/public-key.pem` (SPKI PEM). |
| **Fingerprint** | The first 8 hex characters of SHA-256 of the DER-encoded public key. Serves as a stable, human-readable identity token for enrollment in team keyring files. |
| **Canonicalization** | The process of producing a deterministic, byte-for-byte stable JSON serialization of an ExportV1 payload — sorting keys alphabetically, omitting undefined fields, no whitespace, no HTML entity escaping. The canonical bytes are what gets signed. |
| **Signature** | The Ed25519 signature of the canonical bytes, base64-encoded. Included in every export. |
| **SignedExport** | An ExportV1 document with an attached signature and public key. Tamper-evident: any modification to the payload invalidates the signature. |
| **Keyring** | A JSON file mapping username → Fingerprint, maintained by the team lead. Used with `--keys` to additionally verify that the signer is the enrolled user for that username (prevents impersonation). |
| **VerifyResult** | The outcome of verifying a SignedExport: (valid bool, signerUsername string, error string). |

---

## BC 5 — Team Collaboration

| Term | Definition |
|---|---|
| **ExportV1** | The versioned export document format: contains a Metrics snapshot, Recommendations, Trend data, follow-through records, signing metadata (version, username, generatedAt, publicKey, signature). This is the unit exchanged between developers and the team lead. |
| **TeamConfig** | A configuration file (INI/YAML-like format, hosted at a URL) specifying the member roster and optional group structure. |
| **DeployConfig** | The locally persisted team configuration for this machine: username, push URL, member map. Stored at `~/.token-monitor/config.json`. |
| **TeamMember** | An enrolled developer identified by username and optionally a trusted Fingerprint. |
| **Dedupe** | The process of reducing a list of ExportV1 files (possibly including stale re-submissions) to one export per username — keeping the one with the latest `generatedAt`. |
| **MergedMetrics** | The result of combining multiple developers' Metrics snapshots into one: sums for absolute values (tokens, cost), weighted averages for ratios. |
| **Rollup** | A MergedMetrics snapshot for a logical group (a team, a discipline, the whole org). Multiple Rollups make up a RollupReport. |
| **DominantActivity** | The Activity with the highest token share across a merged set of exports. The tie-breaking rule is fixed by declaration order in ACTIVITIES. |
| **PushURL** | The destination for signed exports: either an `http(s)://` endpoint (POST) or a filesystem path (copy). |

---

## BC 6 — Reconciliation

| Term | Definition |
|---|---|
| **ProviderUsage** | Token-usage data returned by a provider's billing API for a given model and time window: inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens. Represents the provider's authoritative record. |
| **LocalTotal** | The sum of all token counts for a model from StoredEvents in the same window. Represents what this machine claims to have sent. |
| **ApiTotal** | The sum of ProviderUsage tokens for a model from the provider API. Covers the entire org (all machines, all users). |
| **Coverage** | LocalTotal / ApiTotal. Expected to be ≤ 1 (this machine's share of the org). |
| **ReconcileVerdict** | Per-model outcome: `ok` (local ≤ 1.05× API), `local-exceeds-api` (red flag — local claims more than the org was billed for), `local-only` (model in local DB but absent from API response). |
| **Tolerance** | The 5% buffer (1.05×) applied before flagging a breach. Accounts for minor clock skew and daily-bucket boundary effects. |
| **Breach** | True when any model's verdict is `local-exceeds-api`. Causes exit code 1, enabling CI-level alerting. |
| **AdminKey** | The provider's org-level API key. Read from the environment only; never stored, logged, or exported. Held by the team lead. |

---

## BC 7 — Deep Analysis

| Term | Definition |
|---|---|
| **SessionStat** | Behavioral summary of one Agent session: fix-loop count, average context per turn, context growth, cold-restart tokens, duration, dominant activity. |
| **FixIteration** | One testing → coding transition within a session. A session with FixIterations ≥ 2 is a fix-loop session — the agent repeatedly failed tests and was asked to fix them. |
| **ToolStat** | Usage and error statistics for one tool name: total turns using it, error turns, error rate, retry tokens (tokens spent re-running the tool right after it errored). |
| **DeepAnalysis** | The aggregated output of session and tool analysis: 5 ranked session lists (expensive, fix-loop, context-heavy, bloat-trend, cold-restart) + top 15 tools by usage. |
| **LlmPayload** | The aggregates-only JSON object sent to the local LLM agent CLI. Contains token counts, ratios, personas, project-level summaries, session-list slim rows, and rule-based findings — **never** prompt or code content. |
| **LlmAgent** | A locally installed agent CLI that can receive a prompt and produce analysis: `claude`, `gemini`, or `codex`. |
| **PrivacyBoundary** | The invariant that `LlmPayload` crosses: token counts, ratios, tool names, project basenames, activity labels — no prompt text, no code content, no file paths beyond basenames, no user input. |
