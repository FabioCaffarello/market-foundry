# Canonical Order Model and Lifecycle State Machine

> Stage: S383 — OMS Foundation Wave (S382–S387)
>
> Companion: [Order Lifecycle Invariants, Transitions, and Boundaries](order-lifecycle-invariants-transitions-and-boundaries.md)
>
> Predecessor: [OMS and Order Lifecycle Charter (S309)](oms-and-order-lifecycle-charter.md)

## 1. Purpose

This document consolidates the canonical order model as it exists in code,
maps it across the three execution modes (dry_run, paper, venue_live), and
defines the lifecycle state machine with exhaustive transitions.

This is not a redesign.  The model was defined in S309, implemented across
S321–S381, and is proven by 48+ packages of tests.  This document is the
**composition proof** that the model is coherent.

## 2. The Canonical Order Entity: ExecutionIntent

The Foundry does not have an "order" type.  The canonical entity is
`ExecutionIntent` — a discrete, typed, fire-and-forget intent derived from
a risk assessment.

**Source:** `internal/domain/execution/execution.go:98–116`

### 2.1 Minimum Fields

| Field | Type | Semantics | Source |
|---|---|---|---|
| Type | string | Intent family (e.g., `paper_order`) | PaperOrderEvaluator |
| Source | string | Exchange adapter name (e.g., `binancef`) | Config |
| Symbol | string | Instrument (e.g., `btcusdt`) | Config |
| Timeframe | int | Candle period in seconds | Config |
| Side | Side | Direction: `buy`, `sell`, `none` | PaperOrderEvaluator |
| Quantity | string | Requested quantity (decimal string) | PaperOrderEvaluator |
| FilledQuantity | string | Accumulated fill quantity (decimal string) | VenueAdapter |
| Status | Status | Lifecycle state | Lifecycle transitions |
| Risk | RiskInput | Causal risk context | PaperOrderEvaluator |
| Fills | []FillRecord | Ordered list of fill events | VenueAdapter |
| Parameters | map[string]string | Intent-specific context | PaperOrderEvaluator |
| Metadata | map[string]string | Caller-owned metadata | PaperOrderEvaluator |
| CorrelationID | string | Trace chain identifier | Event propagation |
| CausationID | string | Direct causal predecessor | Event propagation |
| Final | bool | Terminal state marker | VenueAdapter |
| Timestamp | time.Time | Intent creation time | PaperOrderEvaluator |

### 2.2 Side Semantics

| Value | Meaning | Action |
|---|---|---|
| `buy` | Long entry | Venue submission with quantity |
| `sell` | Short entry | Venue submission with quantity |
| `none` | No action (flat signal or rejected risk) | No venue call; accepted without fills |

**Source:** `internal/domain/execution/execution.go:10–17`

### 2.3 FillRecord

Each fill is an atomic, append-only record of a partial or full execution.

| Field | Type | Semantics |
|---|---|---|
| Price | string | Fill price (decimal string) |
| Quantity | string | Fill quantity (decimal string) |
| Fee | string | Fee charged (decimal string) |
| Simulated | bool | `true` for paper/dryrun fills, `false` for venue fills |
| Timestamp | time.Time | When the fill occurred |

**Source:** `internal/domain/execution/execution.go:76–83`

### 2.4 RiskInput

Causal context from the risk assessment that produced this intent.
Execution-owned copy — does not import from the risk domain.

| Field | Type | Semantics |
|---|---|---|
| Type | string | Risk family (e.g., `position_exposure`) |
| Disposition | string | `approved`, `modified`, `rejected` |
| Confidence | string | Risk confidence score (decimal string) |
| Timeframe | int | Risk evaluation timeframe |
| StrategyType | string | Strategy family (e.g., `mean_reversion_entry`) |
| DecisionSeverity | string | Behavioral severity from decision layer |

**Source:** `internal/domain/execution/execution.go:85–96`

## 3. Lifecycle State Machine

### 3.1 States

Seven states organized into three tiers:

| Tier | States | Semantics |
|---|---|---|
| Initial | `submitted` | Intent created, not yet dispatched to venue |
| In-flight | `sent`, `accepted`, `partially_filled` | Intent dispatched, awaiting terminal outcome |
| Terminal | `filled`, `rejected`, `cancelled` | Absorbing — no further transitions |

**Source:** `internal/domain/execution/execution.go:27–35`

### 3.2 Transition Matrix

The complete, exhaustive transition graph.  Every pair not listed is
**invalid** and must be rejected by `ValidTransition()`.

```
                    ┌──────────┐
                    │submitted │
                    └──┬───┬───┘
                       │   │   ╲
                       │   │    ╲
                       ▼   ▼     ▼
                   ┌────┐ ┌────────┐ ┌────────┐
                   │sent│ │accepted│ │rejected│
                   └──┬─┘ └┬──┬──┬─┘ └────────┘
                      │    │  │  │     (terminal)
                      │    │  │  │
                      ▼    │  │  ▼
                ┌────────┐ │  │ ┌─────────────────┐
                │accepted│ │  │ │partially_filled  │
                └┬──┬──┬─┘ │  │ └──────┬──────┬────┘
                 │  │  │   │  │        │      │
                 │  │  ▼   ▼  │        ▼      ▼
                 │  │  ┌──────┤   ┌──────┐ ┌─────────┐
                 │  │  │filled│   │filled│ │cancelled│
                 │  │  └──────┘   └──────┘ └─────────┘
                 │  │  (terminal)           (terminal)
                 │  ▼
                 │ ┌─────────────────┐
                 │ │partially_filled  │
                 │ └──────┬──────┬───┘
                 │        │      │
                 │        ▼      ▼
                 │   ┌──────┐ ┌─────────┐
                 │   │filled│ │cancelled│
                 │   └──────┘ └─────────┘
                 ▼
            ┌─────────┐
            │cancelled│
            └─────────┘
```

**Transition table (code-authoritative):**

| From | To | Valid |
|---|---|---|
| submitted | sent | YES |
| submitted | accepted | YES |
| submitted | rejected | YES |
| sent | accepted | YES |
| sent | rejected | YES |
| accepted | filled | YES |
| accepted | partially_filled | YES |
| accepted | cancelled | YES |
| partially_filled | filled | YES |
| partially_filled | cancelled | YES |

**10 valid transitions.  All other 39 combinations are invalid.**

**Source:** `internal/domain/execution/execution.go:54–60`

### 3.3 Terminal State Properties

| Property | Enforcement |
|---|---|
| Absorbing | `ValidTransition(terminal, any)` returns `false` — no outgoing transitions |
| Final flag | `Final` must be `true` for terminal states, `false` for non-terminal |
| Fills frozen | No appends to `Fills` after terminal state is reached |
| Quantity frozen | `FilledQuantity` cannot change after terminal state |
| Observable | Terminal intents are still materialized to KV and ClickHouse |

**Source:** `IsTerminal()` at `execution.go:48–52`

## 4. Mode-Specific Lifecycle Paths

### 4.1 Dry-Run Mode (`dry_run=true`, default)

The outermost decorator `DryRunSubmitter` intercepts all submissions.
**No inner adapter is ever called.**

| Step | Actor | Transition |
|---|---|---|
| Intent created | PaperOrderEvaluator (derive) | → `submitted` |
| Intent received | VenueAdapterActor (execute) | safety gate check |
| Submit intercepted | DryRunSubmitter | `submitted` → `accepted` → `filled` |
| Fill published | Publisher | VenueOrderFilledEvent to EXECUTION_FILL_EVENTS |

**Fill characteristics:**
- `Simulated=true`
- `VenueOrderID="dryrun-{hex16}"`
- `Price="0"` ← **G1 gap: to be closed by making price realistic**
- `Quantity=intent.Quantity` (full fill)
- Single fill per intent (no partials)

**Dominant path:** `submitted → filled` (via accepted, instantaneous)

### 4.2 Paper Mode (`venue.type=paper_simulator`, `dry_run=false`)

`PaperVenueAdapter` simulates venue execution with optional latency.

| Step | Actor | Transition |
|---|---|---|
| Intent created | PaperOrderEvaluator (derive) | → `submitted` |
| Intent received | VenueAdapterActor (execute) | safety gate check |
| Submit to paper adapter | PaperVenueAdapter | `submitted` → `accepted` → `filled` |
| Fill published | Publisher | VenueOrderFilledEvent to EXECUTION_FILL_EVENTS |

**Fill characteristics:**
- `Simulated=true`
- `VenueOrderID="paper-{hex16}"`
- `Price="0"` ← same G1 gap
- `Quantity=intent.Quantity` (full fill)
- Optional `fillDelay` for latency simulation

**Dominant path:** `submitted → filled` (via accepted)

**Note:** `paper_simulator` + `dry_run=false` is rejected at config validation
(fail-closed property FC-9).  Paper mode always runs with `dry_run=true`.

### 4.3 Venue Live Mode (`venue.type=binance_futures_testnet`, `dry_run=false`)

`BinanceFuturesTestnetAdapter` submits real orders to Binance testnet.

| Step | Actor | Transition |
|---|---|---|
| Intent created | PaperOrderEvaluator (derive) | → `submitted` |
| Intent received | VenueAdapterActor (execute) | safety gate check |
| HTTP POST to venue | BinanceFuturesTestnetAdapter | `submitted` → `accepted` |
| Venue response | BinanceFuturesTestnetAdapter | `accepted` → `filled` (or `rejected`) |
| Fill published | Publisher | VenueOrderFilledEvent to EXECUTION_FILL_EVENTS |

**Fill characteristics:**
- `Simulated=false`
- `VenueOrderID=venue-assigned order ID`
- `Price=venue-reported fill price` (realistic)
- `Quantity=venue-reported fill quantity`
- `Fee=venue-reported fee`

**Dominant path:** `submitted → accepted → filled` (~95%)
**Rejection path:** `submitted → rejected`

**Venue status mapping (Binance Futures):**

| Binance Status | Domain Status |
|---|---|
| NEW | accepted |
| FILLED | filled |
| PARTIALLY_FILLED | partially_filled |
| CANCELED / CANCELLED | cancelled |
| REJECTED | rejected |
| EXPIRED | rejected |

### 4.4 SideNone Handling (All Modes)

When `Side=none` (flat signal or rejected risk), the intent is accepted
without venue submission and without fills.

| Mode | Behavior |
|---|---|
| dry_run | DryRunSubmitter returns `StatusAccepted`, no fills |
| paper | PaperVenueAdapter returns `StatusAccepted`, no fills |
| venue_live | No submission to venue; adapter returns `StatusAccepted` |

**Path:** `submitted → accepted` (terminal for no-action intents)

**Note:** `accepted` is not a terminal state in the formal state machine.
For `SideNone` intents, the system treats the intent as finalized despite
the non-terminal status.  This is a design choice, not a bug — `SideNone`
intents have `Quantity="0"` and `FilledQuantity=""`, so no fills are
expected or possible.

## 5. Ownership Matrix

| Concern | Owner | Binary |
|---|---|---|
| Intent creation | PaperOrderEvaluator | derive |
| Initial status (`submitted`) | PaperOrderEvaluator | derive |
| Safety gate enforcement | SafetyGate + StalenessGuard | execute |
| Post-submitted transitions | VenueAdapter (decorated) | execute |
| Fill record creation | VenueAdapter (paper/venue/dryrun) | execute |
| Terminal state determination | VenueAdapter response mapping | execute |
| KV materialization | Store consumer | store |
| ClickHouse persistence | Writer consumer | writer (within store) |
| HTTP query surface | Gateway responder | gateway |
| Kill switch state | ControlKVStore | store (authority), execute (reader) |

**Invariant:** Only the derive binary produces `submitted`.  Only the execute
binary transitions past `submitted`.  No other binary mutates intent status.

## 6. Venue Pipeline Composition

The venue pipeline is a decorator chain assembled at VenueAdapterActor
startup.  The outermost decorator is called first.

```
VenueAdapterActor.onIntent()
    │
    ▼ safety gate check (kill switch + staleness)
    │
    ▼ context.WithTimeout(submitTimeout)
    │
    ▼ a.venue.SubmitOrder(ctx, VenueOrderRequest{Intent})
    │
    ├── [DryRunSubmitter]     ← outermost, if dry_run=true
    │     intercepts ALL submissions
    │     never delegates to inner
    │
    ├── [Post200Reconciler]   ← if VenueQueryPort available
    │     recovers body-read-failure-after-200
    │     uses QueryOrder for reconciliation
    │
    ├── [RetrySubmitter]      ← exponential backoff
    │     retries transient failures
    │     checks kill switch between attempts
    │
    └── [rawAdapter]          ← PaperVenueAdapter or BinanceFuturesTestnetAdapter
          actual venue interaction
```

**Key property:** When `dry_run=true` (default), `DryRunSubmitter` is the
outermost layer and **never delegates** to any inner adapter.  The raw
adapter, retry submitter, and reconciler are never invoked.

## 7. Correlation and Traceability

### 7.1 Correlation Chain

```
StrategyResolvedEvent.Metadata.CorrelationID
    ↓ preserved by StrategyConsumerActor
PaperOrderSubmittedEvent.Metadata.CorrelationID
    ↓ preserved by VenueAdapterActor
VenueOrderFilledEvent.Metadata.CorrelationID
    ↓ preserved by Store/Writer consumers
KV entry / ClickHouse row / HTTP response
```

### 7.2 Causation Chain

```
StrategyResolvedEvent.Metadata.ID
    ↓ becomes CausationID of
PaperOrderSubmittedEvent.Metadata.CausationID
    ↓ PaperOrderSubmittedEvent.Metadata.ID becomes CausationID of
VenueOrderFilledEvent.Metadata.CausationID
```

### 7.3 Deduplication

| Event | Dedup Key | Scope |
|---|---|---|
| PaperOrderSubmittedEvent | `exec:{type}:{source}:{symbol}:{timeframe}:{unix}` | EXECUTION_EVENTS stream |
| VenueOrderFilledEvent | `fill:{venue_order_id}:{unix}` | EXECUTION_FILL_EVENTS stream |

### 7.4 Partition Key

`{source}.{symbol}.{timeframe}` — used for KV bucket keys and subject
routing.  Ensures per-instrument isolation.

## 8. Identity Model

| Identity | Format | Scope | Generator |
|---|---|---|---|
| DeduplicationKey | `exec:{type}:{source}:{symbol}:{timeframe}:{unix}` | JetStream Msg-Id | ExecutionIntent.DeduplicationKey() |
| ClientOrderID | SHA-256(DeduplicationKey)[0:32] hex | Venue submission | ClientOrderID() |
| VenueOrderID | Venue-assigned (e.g., Binance order ID) | Venue namespace | Venue response |
| DryRunOrderID | `dryrun-{hex16}` | Dry-run namespace | DryRunSubmitter |
| PaperOrderID | `paper-{hex16}` | Paper namespace | PaperVenueAdapter |
| PartitionKey | `{source}.{symbol}.{timeframe}` | KV bucket key | ExecutionIntent.PartitionKey() |

**Property:** `ClientOrderID` is deterministic — the same `ExecutionIntent`
always produces the same client order ID.  This enables safe retries and
post-200 reconciliation.

## 9. G1 Gap: Price Realism in Dry-Run and Paper Fills

### Current State

Both `DryRunSubmitter` and `PaperVenueAdapter` produce fills with
`Price="0"`.  This was accepted in S381 as LOW severity because pipeline
correctness was the priority.

### Resolution Direction

The `DryRunSubmitter` should use the last observed market price for the
intent's symbol.  Two options:

| Option | Mechanism | Complexity |
|---|---|---|
| A: NATS KV lookup | Read `OBSERVATION_LATEST` KV for symbol's last price | Low — KV already exists |
| B: In-memory cache | Subscribe to observation events, cache last price per symbol | Medium — requires subscription |

**Recommendation:** Option A (NATS KV lookup) for S383.  The observation KV
bucket already stores the latest candle per symbol, which includes close
price.  A single KV Get() call per intent is sufficient.

**Constraint:** If the KV lookup fails or returns no data, the fill should
use `Price="0"` as fallback.  Price realism is best-effort, not
safety-critical.

## 10. What This Document Does Not Cover

| Topic | Reason | Where |
|---|---|---|
| Write-path integration testing | S384 scope | Next stage |
| Persistence consistency | S385 scope | Two stages out |
| End-to-end compose proof | S386 scope | Three stages out |
| Advanced order types | NG-5 (frozen non-goal) | Never in this wave |
| Order amendments | NG-6 (frozen non-goal) | Never in this wave |
| State machine extension | NG-14 (frozen non-goal) | Never in this wave |
