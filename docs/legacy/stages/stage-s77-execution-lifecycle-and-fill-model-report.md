# Stage S77 — Execution Lifecycle and Fill Model Report

**Status**: Complete
**Date**: 2026-03-18

## Executive Summary

S77 enriched the execution domain with a minimal lifecycle model and fill model, moving beyond the single `submitted` status. Paper execution now produces `filled` intents with simulated fill records for actionable orders. The domain models six lifecycle states with explicit transition rules, three terminal states, and a `FillRecord` type that captures fill details including a `simulated` flag for paper/real distinction. All changes preserve contracts-first discipline, store authority, gateway cleanliness, and latest-only projection semantics.

## Lifecycle Model Introduced

### States
| Status | Purpose | Terminal |
|--------|---------|----------|
| `submitted` | Intent created by evaluator | No |
| `accepted` | Execution boundary acknowledged order | No |
| `filled` | Full quantity executed | Yes |
| `partially_filled` | Partial quantity executed | No |
| `rejected` | Order rejected at execution boundary | Yes |
| `cancelled` | Order cancelled before full fill | Yes |

### Valid Transitions
```
submitted → accepted | rejected
accepted → filled | partially_filled | cancelled
partially_filled → filled | cancelled
```

### Domain Functions
- `ValidTransition(from, to Status) bool` — enforces transition rules.
- `Status.IsTerminal() bool` — reports terminal states.
- `ValidStatus(st Status) bool` — validates status values (extended for all six states).

## Fill Model Introduced

### FillRecord
```go
type FillRecord struct {
    Price     string    `json:"price"`
    Quantity  string    `json:"quantity"`
    Fee       string    `json:"fee"`
    Simulated bool      `json:"simulated"`
    Timestamp time.Time `json:"timestamp"`
}
```

### ExecutionIntent Fields Added
- `FilledQuantity string` — cumulative filled quantity.
- `Fills []FillRecord` — ordered fill history.

### Paper Fill Behavior
- Buy/sell orders: `submitted → filled` with one simulated fill record (price=0, fee=0, simulated=true).
- No-action orders (side=none): stay `submitted`, no fills.

## Files Changed

### Domain
| File | Change |
|------|--------|
| `internal/domain/execution/execution.go` | Added 5 lifecycle statuses, `FillRecord` type, `FilledQuantity`+`Fills` fields on `ExecutionIntent`, `ValidTransition()`, `IsTerminal()`, updated `ValidStatus()` |
| `internal/domain/execution/execution_test.go` | Added tests: all statuses valid, transition tests (valid + invalid), terminal state tests, fill record validation |

### Application
| File | Change |
|------|--------|
| `internal/application/execution/paper_fill_simulator.go` | **New**. Pure function: transitions submitted intents to filled with simulated fill records |
| `internal/application/execution/paper_fill_simulator_test.go` | **New**. Tests: buy/sell fill, no-action stays submitted, non-submitted rejection, field preservation, multi-symbol isolation |

### Actors
| File | Change |
|------|--------|
| `internal/actors/scopes/derive/execution_evaluator_actor.go` | Added `PaperFillSimulator` to actor; evaluator calls simulator after evaluation; enriched log fields |
| `internal/actors/scopes/store/execution_projection_actor.go` | Enriched materialization log with `filled_quantity` and `fills_count` |
| `internal/actors/scopes/store/execution_projection_actor_test.go` | Updated `validExecutionIntent` fixture to use `StatusFilled` with fill records |

### Architecture Docs
| File | Description |
|------|-------------|
| `docs/architecture/execution-lifecycle-model.md` | **New**. Lifecycle states, transitions, terminal semantics, paper mode behavior, intentional limitations |
| `docs/architecture/execution-fill-model.md` | **New**. FillRecord structure, fill semantics, consistency rules, query surface impact, limitations |

## What Did NOT Change

- **Events**: `PaperOrderSubmittedEvent` retained as-is. Event naming reflects the trigger (order submission), not the final state.
- **NATS subjects/streams/consumers**: No infrastructure changes. Same stream, same consumer, same subjects.
- **KV store**: Same bucket, same monotonicity guard. New fields serialize naturally via JSON.
- **Query surface**: Same routes, same parameters. Response now includes richer data via expanded `ExecutionIntent`.
- **Gateway**: No changes.
- **Settings/configuration**: No changes.

## Limits and Simplifications Maintained

| Limit | Reason |
|-------|--------|
| No venue integration | S77 scope is domain enrichment only |
| No OMS/router | Out of scope for lifecycle model |
| No portfolio tracking | Execution does not own position state |
| No multi-venue | Single paper venue only |
| No event-per-transition | Paper mode publishes one event with final state |
| No transition history | Latest-only projection; no event sourcing |
| No price/fee simulation | Paper fills use zero values |
| No partial fill production | `partially_filled` is modeled, not produced |
| No fill consistency in Validate() | Deferred to keep validation gate minimal |
| No timeout/expiry logic | No automatic `accepted → cancelled` on timeout |

## Test Coverage

| Package | Tests Added | Status |
|---------|-------------|--------|
| `internal/domain/execution` | 12 (statuses, transitions, terminal, fills) | Pass |
| `internal/application/execution` | 8 (simulator: buy, sell, no-action, non-submitted, preservation, validation, multi-symbol) | Pass |
| `internal/actors/scopes/store` | Updated fixture | Pass |
| `internal/actors/scopes/derive` | Compile verified | Pass |
| `internal/adapters/nats` | No changes needed | Pass |
| `internal/interfaces/http` | No changes needed | Pass |

## Preparation for S78

With lifecycle and fill model in place, the recommended next steps are:

1. **Transition history / event sourcing**: If auditability demands more than latest-only, introduce per-transition events (`PaperOrderAcceptedEvent`, `PaperOrderFilledEvent`) with a dedicated stream.
2. **Fill consistency enforcement**: Add `Validate()` checks for fill/status consistency (CR-1 through CR-5) once the model is stable.
3. **Paper price simulation**: Introduce simulated fill prices based on latest evidence candle close price, making paper fills more realistic.
4. **Venue integration readiness gate**: With the lifecycle model defined, the next venue step can map real venue responses (accepted, filled, rejected) to existing statuses.
5. **Partial fill support**: Implement `partially_filled` production for venues that report incremental fills.
