# Order Lifecycle Invariants, Transitions, and Boundaries

> Stage: S383 — OMS Foundation Wave (S382–S387)
>
> Companion: [Canonical Order Model and Lifecycle State Machine](canonical-order-model-and-lifecycle-state-machine.md)
>
> Predecessor: [Order Lifecycle Semantics, States, and Non-Goals (S309)](order-lifecycle-semantics-states-and-non-goals.md)

## 1. Purpose

This document catalogs every invariant that the canonical order model must
satisfy, maps each to code enforcement, and identifies test coverage status.

Invariants are grouped by category.  Each invariant has:
- an ID (stable, never reused);
- a formal statement;
- code location where it is enforced;
- test coverage status (covered, gap, or deferred).

## 2. State Transition Invariants (ST)

These invariants govern which state transitions are valid.

### 2.1 Valid Transition Enumeration

**Source:** `internal/domain/execution/execution.go:54–60`

| ID | From | To | Valid | Enforced by |
|---|---|---|---|---|
| ST-1 | submitted | sent | YES | `ValidTransition()` |
| ST-2 | submitted | accepted | YES | `ValidTransition()` |
| ST-3 | submitted | rejected | YES | `ValidTransition()` |
| ST-4 | sent | accepted | YES | `ValidTransition()` |
| ST-5 | sent | rejected | YES | `ValidTransition()` |
| ST-6 | accepted | filled | YES | `ValidTransition()` |
| ST-7 | accepted | partially_filled | YES | `ValidTransition()` |
| ST-8 | accepted | cancelled | YES | `ValidTransition()` |
| ST-9 | partially_filled | filled | YES | `ValidTransition()` |
| ST-10 | partially_filled | cancelled | YES | `ValidTransition()` |

**10 valid transitions.  All 39 other pairs are invalid.**

### 2.2 Invalid Transition Categories

| ID | Category | Examples | Count |
|---|---|---|---|
| ST-INV-1 | Terminal → any | filled→accepted, rejected→sent, cancelled→filled | 21 pairs |
| ST-INV-2 | Backward | accepted→submitted, sent→submitted, filled→accepted | covered by ST-INV-1 + below |
| ST-INV-3 | Skip-level | submitted→filled, submitted→partially_filled, submitted→cancelled | 3 pairs |
| ST-INV-4 | Self-loop | submitted→submitted, accepted→accepted, etc. | 7 pairs |
| ST-INV-5 | Cross-branch | sent→partially_filled, sent→filled, sent→cancelled | 3 pairs |

### 2.3 Transition Enforcement

```go
func ValidTransition(from, to Status) bool {
    targets, ok := validTransitions[from]
    if !ok {
        return false  // terminal states have no entry → always false
    }
    for _, t := range targets {
        if t == to {
            return true
        }
    }
    return false
}
```

**Property:** Terminal states (`filled`, `rejected`, `cancelled`) are not
keys in `validTransitions`, so `ok` is `false` and the function returns
`false` for any transition from a terminal state.

**Test coverage:** Gap — no tests for `ValidTransition()` exist.  S383 must
add exhaustive coverage.

## 3. Terminal State Invariants (TERM)

| ID | Invariant | Code enforcement | Test status |
|---|---|---|---|
| TERM-1 | Terminal states are absorbing: `ValidTransition(terminal, any)` returns `false` | Terminal states absent from `validTransitions` map | Gap |
| TERM-2 | `Final` must be `true` for terminal-state intents | Convention — not enforced at domain level | Gap |
| TERM-3 | `Fills` array is frozen after terminal state | Convention — not enforced at domain level | Gap |
| TERM-4 | `FilledQuantity` is frozen after terminal state | Convention — not enforced at domain level | Gap |
| TERM-5 | Terminal intents remain materialized (never deleted) | KV store never deletes, only overwrites forward | Implicit |

**Note:** TERM-2 through TERM-4 are conventions enforced by the adapter
layer, not by the domain type.  The domain provides `IsTerminal()` and
`ValidTransition()` as building blocks; the adapter is responsible for
setting `Final=true` and not mutating fills/quantity after terminal state.

## 4. Fill Record Invariants (FR)

| ID | Invariant | Formal statement | Test status |
|---|---|---|---|
| FR-1 | Filled requires fills | `status == filled` ⇒ `len(Fills) >= 1` | Gap |
| FR-2 | Partial fill requires fills | `status == partially_filled` ⇒ `len(Fills) >= 1` | Gap |
| FR-3 | Rejected has no fills | `status == rejected` ⇒ `len(Fills) == 0` | Gap |
| FR-4 | Cancelled with quantity has fills | `status == cancelled AND FilledQuantity > "0"` ⇒ `len(Fills) >= 1` | Gap |
| FR-5 | Filled matches quantity | `status == filled` ⇒ `FilledQuantity == Quantity` | Gap |
| FR-6 | Partial fill below quantity | `status == partially_filled` ⇒ `FilledQuantity < Quantity` | Gap |
| FR-7 | Quantity monotonicity | `FilledQuantity[t] >= FilledQuantity[t-1]` | Gap |
| FR-8 | Fills append-only | Existing fill entries never modified or removed | Convention |
| FR-9 | Fill after intent | `fill.Timestamp >= intent.Timestamp` | Gap |

### Fill Discrimination

| Source | Simulated | Price | VenueOrderID prefix |
|---|---|---|---|
| DryRunSubmitter | `true` | `"0"` (G1 gap) | `dryrun-` |
| PaperVenueAdapter | `true` | `"0"` (G1 gap) | `paper-` |
| BinanceFuturesTestnetAdapter | `false` | Venue-reported | Venue-assigned |

## 5. Intent-Fill Consistency Invariants (IFC)

| ID | Invariant | Formal statement |
|---|---|---|
| IFC-1 | Filled has fills | `status == filled` ⇒ `len(Fills) >= 1` |
| IFC-2 | Filled fills match quantity | `status == filled` ⇒ `FilledQuantity == Quantity` |
| IFC-3 | Partial fill consistent | `status == partially_filled` ⇒ `len(Fills) >= 1 AND FilledQuantity < Quantity` |
| IFC-4 | Rejected clean | `status == rejected` ⇒ `len(Fills) == 0` |
| IFC-5 | Cancelled with partial | `status == cancelled AND FilledQuantity > "0"` ⇒ `len(Fills) >= 1` |
| IFC-6 | Cancelled clean | `status == cancelled AND FilledQuantity == "0"` ⇒ `len(Fills) == 0` |
| IFC-7 | Submitted clean | `status == submitted` ⇒ `len(Fills) == 0 AND (FilledQuantity == "" OR FilledQuantity == "0")` |

**Note:** IFC-1 through IFC-4 overlap with FR-1 through FR-3 and FR-5.
Both sets are retained for traceability to S309.

## 6. Quantity Monotonicity Invariants (QM)

| ID | Invariant | Formal statement |
|---|---|---|
| QM-1 | Never decreases | For sequential updates: `FilledQuantity[t] >= FilledQuantity[t-1]` |
| QM-2 | Never exceeds | `FilledQuantity <= Quantity` |
| QM-3 | Append-only | Fill records are append-only; existing entries never modified |

## 7. Status Monotonicity Invariants (SM)

| ID | Invariant | Formal statement |
|---|---|---|
| SM-1 | Forward only | Status transitions follow `validTransitions` — no backward moves |
| SM-2 | Single entry point | `submitted` is the only valid initial state |
| SM-3 | One terminal | Each intent reaches at most one terminal state |
| SM-4 | Final alignment | `Final == true` if and only if `status.IsTerminal() == true` |

## 8. Safety Invariants (SAFE)

| ID | Invariant | Enforcement | Source |
|---|---|---|---|
| SAFE-1 | Safety gate before every VenuePort call | `VenueAdapterActor.onIntent()` checks `SafetyGate.Check()` before `SubmitOrder()` | GR-1 |
| SAFE-2 | Kill switch blocks submission | `SafetyGate` reads `ControlGate.IsHalted()` | GR-1 |
| SAFE-3 | Staleness blocks submission | `StalenessGuard.IsStale()` rejects old intents | GR-1 |
| SAFE-4 | DryRunSubmitter never delegates | `DryRunSubmitter.SubmitOrder()` never calls `inner.SubmitOrder()` | FC-10 |
| SAFE-5 | Nil DryRun defaults to true | `VenueConfig.IsDryRun()` returns `true` when `DryRun` is nil | FC-8 |
| SAFE-6 | Paper + dry_run=false rejected | Config validation rejects this combination | FC-9 |
| SAFE-7 | Context deadline on venue calls | `context.WithTimeout(submitTimeout)` wraps all venue calls | GR-6 |

## 9. Correlation Invariants (CORR)

| ID | Invariant | Formal statement |
|---|---|---|
| CORR-1 | CorrelationID preserved | `VenueOrderFilledEvent.Metadata.CorrelationID == PaperOrderSubmittedEvent.Metadata.CorrelationID` |
| CORR-2 | CausationID chained | `VenueOrderFilledEvent.Metadata.CausationID == PaperOrderSubmittedEvent.Metadata.ID` |
| CORR-3 | PartitionKey stable | `PartitionKey` is deterministic from `{source}.{symbol}.{timeframe}` |
| CORR-4 | DeduplicationKey stable | `DeduplicationKey` is deterministic from intent fields + timestamp |

## 10. Boundary Rules

### 10.1 Derive–Execute Boundary

| Rule | Statement |
|---|---|
| DE-1 | Only derive produces `submitted` status |
| DE-2 | Derive never transitions past `submitted` |
| DE-3 | Execute never creates new intents — only transitions existing ones |
| DE-4 | The NATS event `PaperOrderSubmittedEvent` is the sole crossing point |

### 10.2 Execute–Store Boundary

| Rule | Statement |
|---|---|
| ES-1 | Execute produces `VenueOrderFilledEvent` — store consumes it |
| ES-2 | Store never mutates intent status — only materializes |
| ES-3 | KV store rejects stale writes (timestamp monotonicity) |
| ES-4 | ClickHouse writer is append-only |

### 10.3 Paper–Venue Boundary

| Rule | Statement |
|---|---|
| PV-1 | Shared domain model (`ExecutionIntent`, `FillRecord`) |
| PV-2 | Discrimination via `FillRecord.Simulated` flag |
| PV-3 | VenueOrderID prefix distinguishes source: `dryrun-`, `paper-`, venue-assigned |
| PV-4 | Composite read model works identically regardless of `Simulated` flag |

### 10.4 Lifecycle–Outcome Boundary

The lifecycle (state machine) and the outcome (fill details) are
**independent concerns** that travel together on the same entity.

| Concern | Fields | Owner |
|---|---|---|
| Lifecycle | Status, Final | VenueAdapter (transitions) |
| Outcome | Fills, FilledQuantity, Price, Fee | VenueAdapter (fill extraction) |
| Identity | CorrelationID, CausationID, PartitionKey | Event propagation |
| Context | Risk, Parameters, Metadata | PaperOrderEvaluator (immutable after creation) |

**Invariant:** Lifecycle transitions and outcome recording happen atomically
in the adapter response mapping.  There is no intermediate state where
status is `filled` but fills are empty, or where fills exist but status is
still `submitted`.

## 11. Mode-Specific Transition Paths

### 11.1 Exhaustive Path Catalog

| Mode | Side | Path | Terminal |
|---|---|---|---|
| dry_run | buy | submitted → filled | filled |
| dry_run | sell | submitted → filled | filled |
| dry_run | none | submitted → accepted | accepted* |
| paper | buy | submitted → filled | filled |
| paper | sell | submitted → filled | filled |
| paper | none | submitted → accepted | accepted* |
| venue_live | buy | submitted → accepted → filled | filled |
| venue_live | buy | submitted → rejected | rejected |
| venue_live | sell | submitted → accepted → filled | filled |
| venue_live | sell | submitted → rejected | rejected |
| venue_live | none | submitted → accepted | accepted* |
| venue_live | buy | submitted → accepted → partially_filled → filled | filled |
| venue_live | buy | submitted → accepted → partially_filled → cancelled | cancelled |
| venue_live | buy | submitted → accepted → cancelled | cancelled |

*`accepted` is not formally terminal but is the final state for SideNone intents.

### 11.2 Unreachable Paths (Current Implementation)

| Path | Why unreachable |
|---|---|
| `submitted → sent → ...` | `sent` only used in async models (not implemented) |
| `accepted → partially_filled → cancelled` with fills | Partial cancellation requires async venue events (not implemented) |

These paths are valid per the state machine but not exercised by any
current adapter.  They exist for forward compatibility.

## 12. Test Coverage Matrix

| Invariant Group | Count | Covered | Gap | Deferred |
|---|---|---|---|---|
| State Transitions (ST) | 10 valid + 39 invalid | 0 | 49 | 0 |
| Terminal States (TERM) | 5 | 0 | 5 | 0 |
| Fill Records (FR) | 9 | 1 (FR simulated flag) | 8 | 0 |
| Intent-Fill Consistency (IFC) | 7 | 0 | 7 | 0 |
| Quantity Monotonicity (QM) | 3 | 0 | 3 | 0 |
| Status Monotonicity (SM) | 4 | 0 | 4 | 0 |
| Safety (SAFE) | 7 | 5 | 2 | 0 |
| Correlation (CORR) | 4 | 2 | 2 | 0 |
| **Total** | **49** | **8** | **41** | **0** |

**S383 closes 41 invariant test gaps at the domain level.**

The recommended test file is:
`internal/domain/execution/s383_lifecycle_invariants_test.go`

## 13. Non-Goals Reaffirmed

This document catalogs invariants for the **existing** model.  It does not:

- Add new states to the lifecycle (NG-14)
- Add new order types (NG-5)
- Add amendment or cancellation initiation (NG-6)
- Add async fill reconciliation (NG-7)
- Add position tracking or portfolio aggregation (NG-2)
- Modify the transition matrix (frozen since S309)

If the existing model is found insufficient during testing, that is a
**finding** to be documented, not a fix to be applied within this stage.
