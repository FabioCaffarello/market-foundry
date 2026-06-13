# HTTP API

Gateway HTTP endpoints reference for market-foundry. The gateway
(`cmd/gateway/`) is the only HTTP-exposed binary in the system; all
endpoints listed here are served by it.

For the architecture rationale of why gateway is structured this way,
see [`ARCHITECTURE.md`](ARCHITECTURE.md). For the operational topology
(ports, dependencies, healthchecks), see [`RUNTIME.md`](RUNTIME.md).

---

## Conventions

### Base URL

In local development:

```
http://127.0.0.1:8080
```

The port is configured in `deploy/configs/gateway.jsonc` and exposed
on loopback only by default. Production deployment may vary — see
[`RUNTIME.md`](RUNTIME.md).

### Response format

All responses are JSON. Error responses follow the RFC 7807 Problem
Details shape, served by the `internal/shared/problem` package:

```json
{
  "type": "about:blank",
  "title": "Description",
  "status": 400,
  "detail": "...",
  "instance": "..."
}
```

### Path parameters

Path parameters use the syntax `:name` in route declarations and are
extracted via `httprouter.Params`. They are required parts of the URL.

### Query parameters

Query parameters use `?name=value` and are extracted via
`r.URL.Query().Get()`. They are typically optional unless documented
otherwise. Common query params across read endpoints:

- `source` — venue/segment identifier (e.g., `binance_spot`, `binance_futures`)
- `base`, `quote`, `contract` — the canonical instrument trio
  (e.g., `base=btc&quote=usdt&contract=perpetual`; contract ∈
  `{spot, perpetual, usdtfutures, coinfutures}`), validated by
  `instrument.New`. **The venue-native `base`+`quote`+`contract` parameter was
  retired in H-6.e.2** (read-contract canonical cutover; zero
  external consumers at the time — see ADR-0021 criterion #2
  erratum). Where the instrument is an optional filter, the trio
  is all-or-none.
- `timeframe` — candle interval in seconds (e.g., `60` for 1m, `300` for 5m)

The tuple `(source, instrument, timeframe)` is the partition key for
operational reads; most `*_LATEST` endpoints parse it via a shared
helper (`parseQueryKeyParams` in `handlers/evidence.go` — this doc
previously pointed at `handlers/common.go`, corrected in H-6.e.2).

### Authentication

There is currently no authentication on gateway endpoints. The default
deployment binds to loopback only as the primary access control. Live
deployments are expected to add a reverse proxy with auth in front.

This is a deliberate gap — the system is single-operator and
locally-bound by design.

### Conditional endpoint registration

**All endpoints except `/healthz`, `/readyz`, and `/metrics` are
conditional.** Each route registers itself only when its backing
dependency is wired in the gateway composition root
(`cmd/gateway/run.go`). If a dependency is not configured, the
endpoint is silently absent — calling it returns 404 with no
indication that it would exist when wired.

Practically: in a minimally-wired gateway, only the three
unconditional endpoints respond. As family use cases are added to the
composition root, their endpoints become available. This pattern is
implemented via per-family `*FamilyDeps` structs whose `HasAny()`
methods gate the whole family in the composition root.

The specific gating dependencies per endpoint are noted in each
group's table below where they are non-obvious.

---

## Endpoint groups

The 62 registered routes (per `cmd/gateway/boot_test.go`) fall into 14
functional groups. Each group is documented below with its routes,
path/query params, and a one-line purpose.

### 1. Health and operations (3 routes)

Health probes used by docker-compose, monitoring systems, and
operators. These three are the only **unconditional** routes — they
are always registered regardless of which other deps are wired.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/healthz` | — | — | Liveness probe — process is up |
| GET | `/readyz` | — | — | Readiness probe — gateway+configctl reachable; family deps optional |
| GET | `/metrics` | — | — | Prometheus metrics endpoint (served by the metrics package's `HandlerFunc()`) |

### 2. configctl — configuration lifecycle (8 routes)

Configuration document lifecycle management. configctl is the single
authority for configuration state; gateway exposes its surface over
HTTP for tooling and operators.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| POST | `/configctl/configs` | — | — | Create a new draft config from JSON body |
| GET | `/configctl/config-versions` | — | `scope_kind`, `scope_key` (optional filters) | List config versions across the lifecycle |
| GET | `/configctl/config-versions/:id` | `:id` config version id | — | Retrieve a specific config version |
| GET | `/configctl/configs/active` | — | `scope_kind`, `scope_key` (optional filters) | Currently active config (if any) |
| POST | `/configctl/configs/validate` | — | — | Validate a config JSON body without persisting |
| POST | `/configctl/config-versions/:id/validate` | `:id` | — | Re-validate an existing draft version (Draft → Validated) |
| POST | `/configctl/config-versions/:id/compile` | `:id` | — | Compile a validated version (Validated → Compiled) |
| POST | `/configctl/config-versions/:id/activate` | `:id` | — | Activate a compiled version (Compiled → Active) |

The configctl lifecycle progression is:

```
Draft → Validated → Compiled → Active → Deactivated → Archived
```

POST endpoints drive transitions; GET endpoints inspect state. See
[`ARCHITECTURE.md`](ARCHITECTURE.md) for the lifecycle rationale.

### 3. Evidence (4 routes)

Latest and historical evidence data — candles, trade bursts, volumes.
Latest values are served from store's NATS KV projections; history is
served from ClickHouse via writer's read adapter. Each route is gated
on its own dep (`GetLatestCandle`, `GetCandleHistory`,
`GetLatestTradeBurst`, `GetLatestVolume`).

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/evidence/candles/latest` | — | `source`, `base`+`quote`+`contract`, `timeframe` | Latest candle for the partition |
| GET | `/evidence/candles/history` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `limit`, `since`, `until` | Candle history within an optional time range |
| GET | `/evidence/tradeburst/latest` | — | `source`, `base`+`quote`+`contract`, `timeframe` | Latest trade-burst aggregate |
| GET | `/evidence/volume/latest` | — | `source`, `base`+`quote`+`contract`, `timeframe` | Latest volume aggregate |

The `latest` endpoints have low latency (NATS KV lookup); the
`history` endpoint touches ClickHouse and has higher latency.

### 4. Domain latest (4 routes)

Latest values for the per-type domain projections — signal, decision,
strategy, risk. Each uses `:type` as a path wildcard to select the
specific type within the family. Each route is gated on its respective
dep (`GetLatestSignal`, `GetLatestDecision`, `GetLatestStrategy`,
`GetLatestRisk`).

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/signal/:type/latest` | `:type` ∈ `{ema_crossover, rsi, macd, bollinger, atr, vwap}` | `source`, `base`+`quote`+`contract`, `timeframe` | Latest signal value of `:type` for partition |
| GET | `/decision/:type/latest` | `:type` ∈ `{rsi_oversold, ema_crossover, bollinger_squeeze}` | `source`, `base`+`quote`+`contract`, `timeframe` | Latest decision evaluation of `:type` |
| GET | `/strategy/:type/latest` | `:type` ∈ `{mean_reversion_entry, squeeze_breakout_entry, trend_following_entry}` | `source`, `base`+`quote`+`contract`, `timeframe` | Latest strategy resolution of `:type` |
| GET | `/risk/:type/latest` | `:type` ∈ `{position_exposure, drawdown_limit}` | `source`, `base`+`quote`+`contract`, `timeframe` | Latest risk assessment of `:type` |

Note: not every domain `:type` has a corresponding KV bucket. Some
types flow through the stream and persist only in ClickHouse without
operational projection. See [`RUNTIME.md`](RUNTIME.md) → "KV buckets"
for the current coverage map.

### 5. Execution (3 routes)

Execution intent inspection and control. The `:type` path param has
overloaded semantics across these three routes — handlers dispatch
on its value.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/execution/:type/latest` | `:type` (execution intent type, or `status` for status query) | `source`, `base`+`quote`+`contract`, `timeframe` | Latest execution intent for the partition. When `:type == "status"`, the handler dispatches to `GetExecutionStatus`. Gated on `GetLatestExecution \|\| GetExecutionStatus`. |
| GET | `/execution/:type` | `:type == "control"` (required) | — | Read the execution control gate state. Gated on `GetExecutionControl`. |
| PUT | `/execution/:type` | `:type == "control"` (required) | — (body) | Set the execution control gate state from JSON body. Gated on `SetExecutionControl`. |

The split between `/execution/:type/latest` and `/execution/:type`
preserves a single wildcard slot while supporting two distinct
operations. Handlers reject inputs whose `:type` does not match the
operation they implement.

### 6. Activation (1 route)

Aggregate activation status across the system. Used by operators to
verify the overall activation surface. Gated on `GetActivationSurface`.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/activation/surface` | — | — | Aggregate activation readiness (configctl + ingest + derive bindings) |

### 7. Execution source explain (1 route)

Composite explainability endpoint aggregating activation, gate,
config, and status for a given source/instrument/timeframe partition.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/execution-source-explain` | — | `source`, `base`+`quote`+`contract`, `timeframe` | Aggregate explainability for a partition |

**Path note:** This used to be `/execution/source-explain` but was
renamed in P0.6 to avoid an httprouter trie conflict with
`/execution/:type`. The hyphenated form is the current canonical path.

**Gating note:** Conditional on `GetSourceExplanation` (provided by
`SourcePathConfigProvider`). In the default local environment that
provider is not wired, so this endpoint returns 404 unless explicitly
configured.

### 8. Analytical reads (10 routes)

Historical reads of domain data from ClickHouse via writer's read
adapter. Higher latency than operational reads, but provide time-range
queries, filtering, and aggregations. Each route is independently
gated on its own dep (`GetCandleHistory`, `GetSignalHistory`, etc.).

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/analytical/evidence/candles` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `limit`, `since`, `until` | Historical candles within an optional time range |
| GET | `/analytical/signal/history` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `type`, `limit`, `since`, `until` | Historical signal events filtered by `type` |
| GET | `/analytical/decision/history` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `type`, `outcome`, `limit`, `since`, `until` | Historical decision events filtered by `type` / `outcome` |
| GET | `/analytical/strategy/history` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `type`, `direction`, `limit`, `since`, `until` | Historical strategy resolutions filtered by `type` / `direction` |
| GET | `/analytical/risk/history` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `type`, `disposition`, `limit`, `since`, `until` | Historical risk assessments filtered by `type` / `disposition` |
| GET | `/analytical/execution/history` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `type`, `side`, `status`, `limit`, `since`, `until` | Historical execution intents and fills |
| GET | `/analytical/execution/lifecycle` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `side`, `status`, `limit`, `since`, `until` | Execution lifecycle events as a time series |
| GET | `/analytical/execution/list` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `limit`, `since`, `until` | List of execution intents |
| GET | `/analytical/execution/summary` | — | `source`, `base`+`quote`+`contract`, `timeframe`, `since`, `until` | Aggregated execution summary statistics |
| GET | `/analytical/execution/explain` | — | `type`, `source`, `base`+`quote`+`contract`, `side`, `status` | Per-intent explainability bundle for a session |

`limit` defaults to a backend-defined value when omitted. `since`/`until`
take UNIX timestamps (seconds) and bound the query window.

### 9. Analytical composite reads (15 routes)

Higher-level composite queries that span multiple domains, powering
operator-facing inspection (decision chains, pairing reviews,
effectiveness summaries). All gated under a single envelope check
(at least one composite dep wired) plus a per-route check.

#### 9a. Chain, funnel, dispositions (4)

| Method | Path | Query params | Purpose |
|---|---|---|---|
| GET | `/analytical/composite/chain` | `correlation_id`, `base`+`quote`+`contract` | Full event chain for a correlation id |
| GET | `/analytical/composite/chains` | `correlation_id`, `base`+`quote`+`contract` | List of chains matching the filters |
| GET | `/analytical/composite/funnel` | `type` | Pipeline funnel breakdown by stage |
| GET | `/analytical/composite/dispositions` | `type` | Disposition (final outcome) breakdown |

#### 9b. Decision review and effectiveness (5)

| Method | Path | Query params | Purpose |
|---|---|---|---|
| GET | `/analytical/composite/decision/review` | `correlation_id`, `base`+`quote`+`contract`, `outcome` | One-decision review bundle |
| GET | `/analytical/composite/decision/reviews` | `outcome` | List of decision review bundles |
| GET | `/analytical/composite/decision/effectiveness` | `correlation_id`, `base`+`quote`+`contract` | Effectiveness for a specific decision chain |
| GET | `/analytical/composite/decision/effectiveness/batch` | `decision_type`, `strategy_type`, `severity`, `effectiveness` (filters) | Batch effectiveness across many decisions |
| GET | `/analytical/composite/decision/effectiveness/summary` | `group_by`, `decision_type`, `strategy_type`, `severity` | Aggregated effectiveness summary by group |

#### 9c. Pairing and continuity (6)

| Method | Path | Query params | Purpose |
|---|---|---|---|
| GET | `/analytical/composite/pairing` | (see handler) | List pairings (entry/exit round-trips) |
| GET | `/analytical/composite/pairing/chain` | (see handler) | Pairings grouped by chain |
| GET | `/analytical/composite/pairing/review` | (see handler) | Pairing review bundles |
| GET | `/analytical/composite/pairing/review/chain` | (see handler) | Pairing reviews grouped by chain |
| GET | `/analytical/composite/pairing/cross-session` | `state`, `side` | Pairings that cross session boundaries |
| GET | `/analytical/composite/pairing/continuity-review` | (see handler) | Cross-session continuity review |

### 10. Triage (4 routes)

Operational triage surfaces. Used to inspect failures, anomalies, and
operational state across recent sessions, decisions, and roundtrips.
Each route gated on its respective dep.

| Method | Path | Query params | Purpose |
|---|---|---|---|
| GET | `/analytical/triage/sessions` | `limit`, `status`, `check`, `severity` | Sessions filtered by triage status / check name / severity |
| GET | `/analytical/triage/decisions` | `limit`, `severity` | Decisions flagged for operational review |
| GET | `/analytical/triage/roundtrips` | `limit`, `severity` | Round-trip pairings flagged for review |
| GET | `/analytical/triage/overview` | `timeframe`, `since`, `until`, `session_status`, `source`, `base`+`quote`+`contract` | Overview snapshot across triage categories |

### 11. Sessions (6 routes)

Execution session inspection. Includes list/audit operations on
sessions and per-session detail. `:id` is the session identifier.
Each route is independently gated on its own dep (`ListSessions`,
`BatchAuditSession`, `VerifySession`, `AuditSession`, `UnifiedReport`,
`GetSession`).

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/session-list` | — | (see handler) | List recent sessions |
| GET | `/session-batch-audit` | — | `status`, `ids` (comma-separated) | Batch audit of sessions matching filter |
| GET | `/session/:id/verify` | `:id` | — | Run automated PO verification checks (S461) |
| GET | `/session/:id/audit` | `:id` | — | Consolidated audit bundle for human review (S462) |
| GET | `/session/:id/report` | `:id` | — | Unified operational report (S491) |
| GET | `/session/:id` | `:id` | — | Session metadata |

**Path note:** `/session-list` and `/session-batch-audit` are
hyphenated (renamed from `/session/list` and `/session/batch-audit` in
P0.6) to avoid httprouter trie conflicts with the `/session/:id`
wildcard. The `/session/:id/...` family preserves the wildcard form
because httprouter accepts static children of a wildcard segment as
long as they are registered first. The route file registers the
specific sub-paths (`verify`, `audit`, `report`) before the bare
`/session/:id` to make registration order match this constraint.

### 12. Monitoring (1 route)

Aggregate monitoring state of the runtime. Gated on `GetOperationalState`.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/monitoring/state` | — | — | Aggregate operational health and runtime state |

### 13. Venues (1 route)

Multi-venue capabilities introspection (ADR-0022 R2, H-7.a). Returns
the union of all shipping adapters' static `Capabilities()`
declarations. Consumers MUST tolerate absence of undeclared event
types (R3) and MAY query this surface at startup to confirm which
venues will produce the event types they subscribe to.

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/venues/capabilities` | — | — | Union of adapter capability declarations: `{"venues": [{venue, event_types, contracts, notes?}, …]}` |

Declarations change only on deploy (static per ADR-0022 R1), so the
response is stable for the lifetime of the process.

### 14. Insights (2 routes)

Decision-support analytics (ADR-0027) — read-only descriptive views
of market structure. H-8.a ships volume profile (VPVR); H-8.b ships
TPO (Time-Price Opportunity); cross-venue fusion follows in H-8.c.
Read directly from the KV latest bucket (the gateway is a free KV
reader).

| Method | Path | Path params | Query params | Purpose |
|---|---|---|---|---|
| GET | `/insights/volume-profile/latest` | — | `source`, `base`, `quote`, `contract`, `timeframe` | Latest price-bucketed volume profile (buy/sell notional per level) for the partition |
| GET | `/insights/tpo/latest` | — | `source`, `base`, `quote`, `contract`, `timeframe` | Latest TPO profile (time-at-price: which periods A–X traded at each price level; POC/value-area/initial-balance) for the partition |

---

## Conditional endpoints summary

As noted in **Conventions**, all routes except `/healthz`, `/readyz`,
and `/metrics` are conditional on their backing dep being wired in the
gateway composition root. The gating pattern in code is
`if deps.X != nil { ... }` per route, with a family-level envelope
check via `*FamilyDeps.HasAny()` in `internal/interfaces/http/routes/core.go`.

| Family | Envelope check | Per-route gates |
|---|---|---|
| Evidence | `deps.Evidence.HasAny()` | `GetLatestCandle`, `GetCandleHistory`, `GetLatestTradeBurst`, `GetLatestVolume` |
| Signal | `deps.Signal.HasAny()` | `GetLatestSignal` |
| Decision | `deps.Decision.HasAny()` | `GetLatestDecision` |
| Strategy | `deps.Strategy.HasAny()` | `GetLatestStrategy` |
| Risk | `deps.Risk.HasAny()` | `GetLatestRisk` |
| Execution | `deps.Execution.HasAny()` | `GetLatestExecution` / `GetExecutionStatus` / `GetExecutionControl` / `SetExecutionControl` |
| Activation | `deps.Activation.HasAny()` | `GetActivationSurface` |
| SourceExplain | `deps.SourceExplain.HasAny()` | `GetSourceExplanation` |
| Analytical | `deps.Analytical.HasAny()` | per-route: `GetCandleHistory`, `GetSignalHistory`, …, plus composite deps |
| Session | `deps.Session.HasAny()` | per-route: `ListSessions`, `BatchAuditSession`, `VerifySession`, `AuditSession`, `UnifiedReport`, `GetSession` |
| Monitoring | `deps.Monitoring.HasAny()` | `GetOperationalState` |
| Triage | `deps.Triage.HasAny()` | `GetSessionTriage`, `GetDecisionTriage`, `GetRoundTripTriage`, `GetTriageOverview` |
| Venues | `deps.Venues.HasAny()` | static `Capabilities` slice (always wired in production — ships with the binary) |
| Insights | `deps.Insights.HasAny()` | `GetLatestVolumeProfile` + `GetLatestTPOProfile` (KV-direct; each wired when its insights KV reader connects) |

A minimally-wired gateway responds only on `/healthz`, `/readyz`,
`/metrics`, and the configctl group (always available because
configctl readiness is required for gateway boot).

---

## Handler implementation map

The handlers live in `internal/interfaces/http/handlers/`. Mapping
from route group to handler file(s):

| Group | Handler file(s) |
|---|---|
| 1. Health and operations | `healthz.go`, `readyz.go` (`/metrics` is served by `metrics.HandlerFunc()` outside the handler package) |
| 2. configctl | `configctl.go` |
| 3. Evidence | `evidence.go` |
| 4. Domain latest | `signal.go`, `decision.go`, `strategy.go`, `risk.go` (one handler file each) |
| 5. Execution | `execution.go` (latest/status dispatch) + `execution_control.go` (control GET/PUT) |
| 6. Activation | `activation.go` |
| 7. Source explain | `source_explain.go` |
| 8. Analytical | `analytical.go` |
| 9. Composite | `composite.go` |
| 10. Triage | `triage.go` |
| 11. Sessions | `session.go` (single handler covers all 6 session routes) |
| 12. Monitoring | `monitoring.go` |
| 13. Venues | `venues.go` |
| 14. Insights | `insights.go` |

A shared helper `parseQueryKeyParams(r)` in `handlers/common.go`
extracts `source` / `base`+`quote`+`contract` / `timeframe` for `*_LATEST` endpoints.
The path param helper `pathParam(r, "name")` extracts httprouter
params.

---

## Boot-time registration verification

The route registration is verified at test time by
`cmd/gateway/boot_test.go` (added in P0.6 after three httprouter
trie conflicts were discovered in production). The test exercises
the full route table in a hermetic environment with noop handlers,
catching any static-vs-wildcard conflicts at CI time before container
boot.

If you add a new route, you **must** also add it to the test's
`routes` slice. Without that, a future conflict will only be
discovered when the gateway actually boots.

The test currently registers 62 routes — matching the production
registration count.

---

## Reading further

| If you want | Go to |
|---|---|
| Architecture overview | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Runtime topology, ports, streams | [`RUNTIME.md`](RUNTIME.md) |
| Current state and known gaps | [`RESUMPTION.md`](RESUMPTION.md) |
| Daily development workflow | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| Domain-specific data shapes | [`domain/`](domain/README.md) |
