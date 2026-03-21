# Paper Order Generation: Guard Rails and Boundaries

## Purpose

This document defines the explicit guard rails, safety boundaries, and scope limits that govern paper order generation in market-foundry. These constraints ensure that paper execution remains controlled, auditable, and unable to produce unintended external effects.

## Guard Rails

### 1. Paper Mode Only

All generated orders are of type `"paper_order"`. The `PaperOrderEvaluator` hardcodes `Type: "paper_order"` — there is no code path that produces any other order type from the derive actor chain. All fill records carry `Simulated: true`.

### 2. No Real Venue Interaction

The derive-side pipeline (signal → decision → strategy → risk → paper order) never contacts any external venue. The `PaperFillSimulator` produces simulated fills locally without network I/O. The `PaperVenueAdapter` exists for the execute-side but is a local simulator with zero external calls.

### 3. Risk-Gated Quantity

Order quantity is never self-determined by the execution layer. It is always `maxPositionPct` — the risk-constrained position size calculated by risk evaluators. The execution layer cannot inflate, override, or bypass this value.

### 4. Disposition-Gated Side

- `rejected` risk → `SideNone` (no order)
- `flat` strategy → `SideNone` (no order)
- Only `approved` or `modified` dispositions with `long`/`short` direction produce actionable orders

The execution evaluator has no ability to override a risk rejection.

### 5. Domain Validation

Every execution intent passes `intent.Validate()` before publishing. Invalid intents (missing required fields, invalid status) are logged and dropped — they never reach the publisher.

### 6. Status Lifecycle Enforcement

The domain enforces valid status transitions via `ValidTransition(from, to)`. An intent cannot skip states or regress to prior states. Terminal states (`filled`, `rejected`, `cancelled`) are final.

### 7. Kill Switch (ControlGate)

The `ControlGate` mechanism provides a runtime kill switch:
- `active` → execution proceeds
- `halted` → all execution blocked with reason
- Fail-closed: if the gate state cannot be read within timeout, execution is blocked

**Note:** SafetyGate integration into the derive-side actor chain is deferred to S267. The mechanism exists at the application layer and is proven in unit tests.

### 8. Staleness Guard

The `StalenessGuard` rejects intents older than a configurable `maxAge`. This prevents stale signals from producing delayed orders.

**Note:** Like SafetyGate, staleness enforcement in the actor chain is deferred to S267. The guard exists and is proven at the application layer.

### 9. Causal Traceability

Every paper order carries `CorrelationID` (end-to-end trace from signal) and `CausationID` (immediate causal parent — the risk event). This makes every order auditable: given any paper order, you can trace back to the exact signal, decision, strategy, and risk assessment that produced it.

### 10. Per-Symbol Isolation

- Each `PaperOrderEvaluator` is scoped to a single `(source, symbol, timeframe)` tuple
- Partition keys (`source.symbol.timeframe`) ensure KV storage isolation
- Deduplication keys prevent duplicate processing via NATS JetStream
- No cross-symbol bleed is possible

## Boundaries

### What Paper Order Generation Does

- Translates risk-approved domain context into paper execution intents
- Simulates fills for actionable orders (buy/sell)
- Preserves full causal context from signal through execution
- Publishes `PaperOrderSubmittedEvent` for downstream consumption

### What Paper Order Generation Does NOT Do

| Prohibited | Reason |
|-----------|--------|
| Open real venue connections | Paper mode only |
| Execute real trades | No real money path exists |
| Aggregate across symbols | Per-symbol isolation by design |
| Track portfolio state | No portfolio model exists |
| Route to multiple venues | No OMS or routing exists |
| Override risk constraints | Execution is risk-gated |
| Bypass kill switch | Fail-closed by design |
| Produce orders without trace | CorrelationID/CausationID are mandatory |

### Scope Freeze

The following are explicitly out of scope for the paper execution wave:

1. **Real venue adapters** — only `PaperVenueAdapter` is active
2. **Multi-venue routing** — single venue path only
3. **Portfolio management** — no position aggregation
4. **OMS (Order Management System)** — no order lifecycle management beyond intent/fill
5. **Performance optimization** — correctness over speed
6. **New domain types** — no new event types, only existing `PaperOrderSubmittedEvent`
7. **Schema changes** — no ClickHouse or NATS schema modifications
8. **New actors** — existing actors are sufficient

## Failure Modes

| Failure | Behavior | Visibility |
|---------|----------|-----------|
| Risk rejection | `SideNone`, quantity 0 — published as no-action intent | Logged, event emitted |
| Flat strategy | `SideNone`, quantity 0 — published as no-action intent | Logged, event emitted |
| Evaluation failure | Intent not produced, `ok=false` | Logged as error |
| Fill simulation failure | Intent not published | Logged as error |
| Validation failure | Intent not published | Logged as error with problem detail |
| Missing ScopePID | Risk assessment produced but no execution fan-out | Risk event still published |

All failure modes are visible via structured logging. No failure is silently swallowed.
