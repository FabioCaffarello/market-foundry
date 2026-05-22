# execution — Execution intents and venue control

The `execution` domain is the largest in the system (1264 production
LOC, 3014 test LOC, 14 test files in the domain package alone, plus
62 files in `internal/application/execution/`). This reflects its
operational weight: this is where strategy outputs become real (or
simulated) orders, where venue interactions happen, and where the
trading lifecycle is governed.

If you're tracking why the `execute` binary's composition
(`cmd/execute/run.go`) is 3–5× larger than other services, the
answer is here.

---

## What this domain models

Execution covers four overlapping concerns:

1. **Intent lifecycle.** A strategy resolution becomes an
   `ExecutionIntent`, which progresses through states from submission
   through to a terminal verdict (filled, rejected, cancelled).
2. **Session lifecycle.** Intents and their fills are grouped into
   bounded `Session`s. Sessions have explicit close semantics
   including a "halted" alternative for forced termination.
3. **Venue control.** Communication with exchange APIs (or with the
   `PaperVenueAdapter` for dry-run mode). Includes order submission,
   fill reception, rejection handling, and post-order verification.
4. **Audit & verification.** After-the-fact inspection of sessions
   (`SessionAuditBundle`, `BatchAuditResult`), explanation surfaces
   (`SourcePathExplanation`), unified reports
   (`UnifiedOperationalReport`).

The combination is unusually large. A future architectural pass
might split this into 2–3 sub-domains (intent, session, audit) but
the current single-domain shape reflects how the system grew.

---

## Modes

Execution operates in one of three modes, selected via configuration:

| Mode | Venue contact | Money | Use |
|---|---|---|---|
| **paper** (dry-run) | None — `PaperVenueAdapter` synthesizes fills locally | Fake | Default safe mode; primary testing surface |
| **testnet** | Real Binance Testnet WebSocket and REST | Fake | Protocol integration testing |
| **mainnet** | Real Binance production endpoints | Real | Operational use; requires explicit credentials |

Promotion between modes is **deliberate, not automated**. A strategy
expected to graduate goes through paper → testnet → mainnet by
explicit config changes (separate JSONC config files under
`deploy/configs/execute-*.jsonc`).

For details on which config file targets which mode, see
[`../RUNTIME.md`](../RUNTIME.md) → "Deployment modes".

A fourth shape worth knowing: **mainnet-dry-run**. Configured via
`docker-compose.mainnet-dry-run.yaml`, it loads mainnet credentials
and exercises the full mainnet code path, but a `DryRunSubmitter`
(`internal/application/execution/dry_run_submitter.go`) intercepts
all venue calls before they hit the network. This validates
credential resolution and connectivity without risking real orders.

---

## Core types

The 30+ public types organize into three layers.

### Lifecycle layer

The types that carry an intent from submission to terminal verdict.

| Type | Purpose |
|---|---|
| `ExecutionIntent` | The canonical intent struct (with `Validate() *problem.Problem`) |
| `Session` | The bounded execution window (with `Validate() *problem.Problem`) |
| `SessionConfigSnapshot` | Snapshot of the configctl projection at session start |
| `SessionActivationSnapshot` | Snapshot of activation state at session start |
| `SessionSegmentCounters` | Per-segment (Spot/Futures) counters for the session |
| `FillRecord` | A single fill event within an intent (price, qty, fee, `FeeSource`, `Simulated` flag) |
| `RiskInput` | Snapshot of the risk verdict that admitted the intent |
| `Status` | Intent state enum (see state machine below) |
| `SessionStatus` | Session state enum (see state machine below) |
| `FeeSource` | Fill fee provenance enum (see "FillRecord and FeeSource" below) |

Two `Validate()` methods returning `*problem.Problem` exist in this
domain — on `ExecutionIntent` and on `Session` — both canonical.

#### ExecutionIntent state machine

`Status` is defined in `internal/domain/execution/execution.go` with
these values:

| Status | Terminal? |
|---|---|
| `submitted` | no |
| `sent` | no |
| `accepted` | no |
| `partially_filled` | no |
| `filled` | **terminal** |
| `rejected` | **terminal** |
| `cancelled` | **terminal** |

Allowed transitions are enforced by `func ValidTransition(from, to Status) bool`
(execution.go:63) consulting a `validTransitions` map. The map — not
this doc — is the authority on which arrows are legal.

The typical happy-path flow:

```
submitted → sent → accepted → partially_filled* → filled
```

Where `partially_filled` may be skipped (single full fill) or
revisited as multiple partial fills land. Terminal alternatives are
`rejected` (venue or pre-flight refusal) or `cancelled` (operator or
session-close initiated).

#### Session state machine

`SessionStatus` is defined in `internal/domain/execution/session.go`
with only three values:

| Status | Meaning |
|---|---|
| `open` | Session is accepting intents and fills |
| `closed` | Session was closed deliberately (operator-initiated or scheduled) |
| `halted` | Session was forced to terminate (failure, kill-switch, integrity gate) |

There is no fine-grained transition state machine for sessions
beyond these three: a session is one of the three statuses at any
moment. Helpers in `session.go` (`IsValid()`, `IsTerminal()`)
distinguish the categories. The session-close path
(`s500_lifecycle_close_test.go`) covers in-flight surfacing,
reconciliation, and carryover semantics — these are concerns
within the close transition, not separate session statuses.

### Control layer

The types that govern whether an intent **should** be issued at all.

| Type | Purpose |
|---|---|
| `ControlGate` | Runtime constraint on activation (e.g., "configctl ready" + "all required projections exist") |
| `ActivationSurface` | Aggregate readiness across the system, exposed via `GET /activation/surface` |
| `ActivationDimensions` | The dimensional breakdown of the activation surface |
| `SourcePathExplanation` | Explainability bundle aggregating activation, gate, config, and status for a partition **(unwired — see Anomalies / G1)** |
| `SourcePathConfig` | Configuration snapshot for the source-driven path |
| `VerificationScope` | Scope of a post-order verification check |

### Audit & verification layer

The types that allow operators to inspect what happened.

| Type | Purpose |
|---|---|
| `SessionAuditBundle` | Full audit data for a single session |
| `AuditLifecycleEntry` | One lifecycle step in the audit timeline |
| `AuditOrderActivity` | Per-order activity slice of an audit |
| `AuditFeeSummary` | Fee aggregation across the session |
| `AuditConsistency` | Consistency-check findings for the session |
| `AuditCheckIndex` | Index into the audit's check results |
| `BatchAuditResult` | Multi-session audit result (surface behind `GET /session-batch-audit`) |
| `BatchAuditEntry`, `BatchAuditSummary`, `BatchCheckAggregation` | Components of batch audit output |
| `POCheckResult`, `POSummary`, `POVerificationReport` | Post-order verification primitives (S461) |
| `UnifiedOperationalReport` | Comprehensive report aggregating intent + fills + session + decision chain (S491) |
| `ReportGap`, `ReportVerificationSection`, `ReportAuditSection`, `ReportOperationalStateSection`, `ReportTriageSection` | Sections within the unified report |

### Event types

| Type | Stream | Purpose |
|---|---|---|
| `PaperOrderSubmittedEvent` | `EXECUTION_EVENTS` | An intent was admitted for paper or venue submission |
| `VenueOrderFilledEvent` | `EXECUTION_FILL_EVENTS` | The venue (or paper adapter) reported a fill |
| `VenueOrderRejectedEvent` | `EXECUTION_REJECTION_EVENTS` | The venue rejected the order |
| `SessionLifecycleEvent` | `SESSION_LIFECYCLE_EVENTS` | Session opened, closed, or halted |

### FillRecord and FeeSource

Fills carry explicit provenance via `FeeSource`, defined in
`execution.go:78` (introduced in S499):

| Value | Meaning |
|---|---|
| `venue` | Real commission from the exchange (Spot fills with non-empty `fills[]`) |
| `unavailable` | Venue did not return commission (e.g., Binance Futures RESULT response). Fee=0 is expected, not a data gap. |
| `simulated` | Paper/dry-run fill, no real fee |
| `fallback` | Venue fill where `fills[]` was unexpectedly empty (Fee=0 may be a data gap) |

This explicit provenance distinguishes "zero fee is expected" from
"zero fee is a data gap" in downstream reconciliation.

---

## Event flow

execution participates in **four** streams. Most domains participate
in one.

### Streams written by `execute`

| Stream | Consumers |
|---|---|
| `EXECUTION_FILL_EVENTS` | store, writer |
| `EXECUTION_REJECTION_EVENTS` | store, writer |
| `SESSION_LIFECYCLE_EVENTS` | gateway |

### Streams consumed by `execute`

| Stream | Writer | Why execute consumes |
|---|---|---|
| `EXECUTION_EVENTS` | derive | execute receives admitted intents to act on |
| `STRATEGY_EVENTS` | derive | execute consumes specific strategy types (partial — see [strategy.md](strategy.md)) |

The `STRATEGY_EVENTS` consumption is **partial**: only
`execute-strategy-mean-reversion-entry` durable exists today.
`squeeze_breakout_entry` and `trend_following_entry` resolve and
persist without driving execution.

### Per-domain durables

From `internal/adapters/nats/natsexecution/registry.go`:

| Durable | Owner | Stream |
|---|---|---|
| `store-execution-paper-order` | store | EXECUTION_EVENTS |
| `writer-execution-paper-order` | writer | EXECUTION_EVENTS |
| `execute-venue-market-order-intake` | execute | EXECUTION_EVENTS |
| `store-execution-venue-market-order-fill` | store | EXECUTION_FILL_EVENTS |
| `writer-execution-venue-fill` | writer | EXECUTION_FILL_EVENTS |
| `store-execution-venue-rejection` | store | EXECUTION_REJECTION_EVENTS |
| `writer-execution-venue-rejection` | writer | EXECUTION_REJECTION_EVENTS |
| `gateway-verification-trigger` | gateway | SESSION_LIFECYCLE_EVENTS (S490) |

A multi-segment variant of the execute-side intake consumer also
exists (`ExecuteVenueIntakeConsumerForSegments`) for the unified
Spot+Futures compose variant.

### Subject patterns

```
execution.events.paper_order.submitted
execution.fill.venue_market_order
execution.rejection.venue_market_order
execution.session.lifecycle
```

The partition key (source, symbol, timeframe) is appended by the
publisher. For exact form, see
`internal/adapters/nats/natsexecution/registry.go`.

---

## Adapters

### NATS adapter
`internal/adapters/nats/natsexecution/` — 4 stream registrations,
3 publishers (intent/fill/rejection), 8 consumer specs (above), 3 KV
stores (control / paper-order / session + venue), session-lifecycle
consumer plumbing for gateway.

### Application packages

| Package | Files | Purpose |
|---|---|---|
| `internal/application/execution/` | 62 | Producers, lifecycle logic, venue adapters, safety gates, staleness guards, segment router, credential providers, integration tests |
| `internal/application/executionclient/` | 24 | Gateway-side read client: use cases for HTTP queries, contracts, audit/verification/report flows |

### Venue adapters

5 mode-specific venue adapters plus 2 submitter wrappers, all in
`internal/application/execution/`:

| File | Targets |
|---|---|
| `paper_venue_adapter.go` | Paper/dry-run (no venue contact; uses `paper_fill_simulator.go`) |
| `binance_spot_testnet_adapter.go` | Binance Spot testnet |
| `binance_spot_mainnet_adapter.go` | Binance Spot mainnet (real orders, real money) |
| `binance_futures_testnet_adapter.go` | Binance Futures testnet |
| `binance_futures_mainnet_adapter.go` | Binance Futures mainnet |
| `dry_run_submitter.go` | Wrapper that intercepts real-adapter calls in mainnet-dry-run mode |
| `retry_submitter.go` | Wrapper that adds retry-with-backoff to any submitter |

The exchange-side connectivity (WebSocket lifecycle, REST API calls,
reconnection, venue-specific quirks) lives in:
- `internal/adapters/exchanges/binances/` — Binance Spot
- `internal/adapters/exchanges/binancef/` — Binance Futures

These adapters handle the venue-specific quirks (e.g., `fills[]`
array in Spot vs. RESULT response in Futures — directly motivating
the FeeSource provenance distinction above).

### ClickHouse

`internal/adapters/clickhouse/execution_reader.go` exposes the
`executions` table (created by migration `006_create_executions.sql`).
Reads for `/analytical/execution/*` endpoints go through writer's
read adapter. Composite reads pull from the execution reader plus
decision/strategy/risk readers.

---

## KV bucket coverage

Execution uses a **different naming convention** than the other
derivation domains. Where signal/decision/strategy/risk use
`{DOMAIN}_{TYPE}_LATEST`, execution mixes action-noun projections
with named state buckets:

| Bucket | Shape | What it projects |
|---|---|---|
| `EXECUTION_PAPER_ORDER_LATEST` | `_LATEST` | Latest paper-order submission per partition |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | `_LATEST` | Latest venue order per partition |
| `EXECUTION_VENUE_REJECTION_LATEST` | `_LATEST` | Latest venue rejection per partition |
| `EXECUTION_CONTROL` | named | Execution control gate state (read/written by `/execution/:type` GET and PUT) |
| `EXECUTION_SESSION` | named | Current session metadata per partition |

The naming convention difference is intentional: execution's
projections are keyed by **what action happened** (paper order,
venue rejection) or by **named operational state** (control,
session), not by **what type was processed** as elsewhere.

---

## HTTP surface

### Direct execution endpoints
- `GET /execution/:type/latest` — latest state for execution `:type`
- `GET /execution/:type` — control state (requires `:type == "control"`)
- `PUT /execution/:type` — update control state (requires `:type == "control"`)

### Activation endpoint
- `GET /activation/surface` — aggregate readiness across the system

### Source explanation (gated, currently unwired — G1 gap)

- `GET /execution-source-explain` — composite explainability bundle

The endpoint exists in code, the route registers conditionally on
`deps.GetSourceExplanation != nil` (in
`internal/interfaces/http/routes/source_explain.go`). However, the
use-case constructor `NewGetSourceExplanationUseCase` (in
`internal/application/executionclient/get_source_explanation.go`)
**has no caller in `cmd/gateway/`** — neither in `compose.go`,
`run.go`, nor any other composition file. Result: `deps.GetSourceExplanation`
is always `nil`, the route never registers, and the endpoint returns
404 in every deployment, not just local default.

The handler also requires a `SourcePathConfigProvider` implementation;
no concrete implementation exists anywhere in the repository today.

To activate:
1. Implement `SourcePathConfigProvider` in `cmd/gateway/` (likely
   `compose.go`).
2. Construct the use case via
   `executionclient.NewGetSourceExplanationUseCase(gateway, configProvider)`.
3. Pass the result into `SourceExplainFamilyDeps.GetSourceExplanation`.

Documented as **G1** in [`../RESUMPTION.md`](../RESUMPTION.md).

### Sessions endpoints
- `GET /session-list` — list of sessions
- `GET /session-batch-audit` — multi-session audit
- `GET /session/:id` — single-session detail
- `GET /session/:id/verify` — S461 verification report
- `GET /session/:id/audit` — S462 audit bundle
- `GET /session/:id/report` — S491 unified operational report

The hyphenated `/session-list` and `/session-batch-audit` were
renamed in P0.6 from `/session/list` and `/session/batch-audit` due
to httprouter trie conflict with the `/session/:id` wildcard. The
wildcard family preserved its form. Documented as **D1** surface debt
in [`../RESUMPTION.md`](../RESUMPTION.md).

### Analytical endpoints
- `/analytical/execution/{history,lifecycle,list,summary,explain}`

For full HTTP reference see [`../HTTP-API.md`](../HTTP-API.md).

---

## Known anomalies and patterns

### 1. KV naming convention differs from other domains

Documented above. Mix of action-noun (PAPER_ORDER, VENUE_REJECTION)
and named-state (CONTROL, SESSION) patterns, vs the type-name
pattern (RSI, EMA_CROSSOVER) used elsewhere. Intentional; reflects
the operational shape of execution where projections are not
1:1 with intent types.

### 2. Stage-tagged test files survived the reset

Tests under `internal/domain/execution/` carry stage names in their
filenames:

| File | Covers |
|---|---|
| `s384_lifecycle_invariants_test.go` | Intent lifecycle invariants |
| `s386_rejection_event_test.go` | Rejection event publish/consume path |
| `s461_verification_test.go` | Post-order verification (PO checks) |
| `s462_audit_bundle_test.go` | Session audit bundle |
| `s467_audit_bundle_test.go` | Batch audit bundle |
| `s485_verification_scope_test.go` | Verification scope semantics |
| `s500_lifecycle_close_test.go` | Session-close path with in-flight surfacing, reconciliation, carryover |

These tests provide real coverage. The stage-tag naming is residual
debt. The application package (`internal/application/execution/`)
also carries ~25 stage-tagged tests (s384, s385, s387, s400, s405–
s407, s412–s418, s422–s424, s428, s433–s441, s460). Listed as **D4**
in [`../RESUMPTION.md`](../RESUMPTION.md). A future rename wave is
the resolution path.

### 3. SourcePathConfigProvider is unwired

The G1 gap. Documented above and in
[`../RESUMPTION.md`](../RESUMPTION.md).

### 4. Asymmetric strategy consumption

Strategy events flow to execute only for `mean_reversion_entry`.
Other strategy types resolve and persist without execution.
Documented in [strategy.md](strategy.md) and revisited here.

### 5. Session has only three statuses

`SessionStatus` is open / closed / halted — no progression states.
This is unusual relative to `ExecutionIntent`'s seven-state
transition graph. Session phases live in helper code (close
sequencing, in-flight surfacing) rather than as explicit enum
values. If you expect to find `SessionState{Validating, Closing}`
or similar, you won't — those concerns are buried in functions
under `session.go` and exercised by `s500_lifecycle_close_test.go`.

### 6. Composition complexity in `cmd/execute/run.go`

The execute binary's run.go (~328 lines) is 3–5× the size of other
binaries' run.go files. This is inherent to the domain's scope:
each mode requires distinct adapter wiring, multiple venue handlers
(Spot + Futures × testnet + mainnet × dry-run + retry wrappers),
session lifecycle management, and audit pipeline composition.

---

## Reading further

| If you want | Go to |
|---|---|
| The strategies feeding execution | [strategy.md](strategy.md) |
| Win/loss classification of fills | [effectiveness.md](effectiveness.md) |
| FIFO entry/exit matching across fills | [pairing.md](pairing.md) |
| Mode-specific deployment | [`../RUNTIME.md`](../RUNTIME.md) → Deployment modes |
| HTTP endpoint reference | [`../HTTP-API.md`](../HTTP-API.md) |
| G1 (source-explain) gap context | [`../RESUMPTION.md`](../RESUMPTION.md) |
| D1, D4 (paths, stage-tagged tests) surface debt | [`../RESUMPTION.md`](../RESUMPTION.md) |
