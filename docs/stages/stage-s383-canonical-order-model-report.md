# Stage S383 — Canonical Order Model and Lifecycle State Machine Report

## Executive Summary

S383 maps the canonical order model (`ExecutionIntent`) and its lifecycle
state machine as they exist in code, validates coherence across the three
execution modes (dry_run, paper, venue_live), and catalogs 49 invariants
with their enforcement status.

**Key findings:**
- The model is sound.  Seven states, 10 valid transitions, 3 terminal
  states, fire-and-forget semantics — all coherent with S309 design.
- 41 of 49 invariants lack explicit domain-level test coverage.
- The G1 gap (Price="0" in dry-run/paper fills) has a clear resolution path
  via NATS KV lookup of last observed market price.
- No state machine extension is needed.  The existing model handles all
  three execution modes.

## Model Definition

### Canonical Entity

`ExecutionIntent` — 16 fields, defined at `internal/domain/execution/execution.go:98–116`.
Not "order".  The Foundry's abstraction is an intent derived from risk assessment.

### State Machine

Seven states in three tiers:

| Tier | States |
|---|---|
| Initial | `submitted` |
| In-flight | `sent`, `accepted`, `partially_filled` |
| Terminal (absorbing) | `filled`, `rejected`, `cancelled` |

10 valid transitions.  39 invalid pairs.  3 terminal states with no outgoing
transitions.  Enforcement via `ValidTransition()` at `execution.go:62–74`.

### Mode-Specific Paths

| Mode | Dominant path | Terminal |
|---|---|---|
| dry_run (buy/sell) | submitted → filled | filled |
| dry_run (none) | submitted → accepted | accepted* |
| paper (buy/sell) | submitted → filled | filled |
| venue_live (buy/sell) | submitted → accepted → filled | filled |
| venue_live (rejection) | submitted → rejected | rejected |

*SideNone intents finalize at `accepted` — not formally terminal but no
further transitions occur.

## Invariant Catalog

49 invariants across 8 categories:

| Category | Count | Covered | Gap |
|---|---|---|---|
| State Transitions (ST-1 through ST-10 + 39 invalid) | 49 pairs | 0 | 49 |
| Terminal States (TERM-1 through TERM-5) | 5 | 0 | 5 |
| Fill Records (FR-1 through FR-9) | 9 | 1 | 8 |
| Intent-Fill Consistency (IFC-1 through IFC-7) | 7 | 0 | 7 |
| Quantity Monotonicity (QM-1 through QM-3) | 3 | 0 | 3 |
| Status Monotonicity (SM-1 through SM-4) | 4 | 0 | 4 |
| Safety (SAFE-1 through SAFE-7) | 7 | 5 | 2 |
| Correlation (CORR-1 through CORR-4) | 4 | 2 | 2 |
| **Total** | **49** | **8** | **41** |

Full invariant definitions in [Order Lifecycle Invariants, Transitions, and
Boundaries](../architecture/order-lifecycle-invariants-transitions-and-boundaries.md).

## Boundaries

### Ownership

| Binary | Responsibility |
|---|---|
| derive | Creates `ExecutionIntent` with `Status=submitted` |
| execute | All transitions past `submitted`; fill creation |
| store | KV + ClickHouse materialization (read-only) |
| gateway | HTTP query surface (read-only) |

### Lifecycle vs. Outcome

Lifecycle (Status, Final) and outcome (Fills, FilledQuantity) are
independent concerns that travel together.  They are updated atomically
in the adapter response mapping — no intermediate state where one is
updated but the other is not.

## G1 Gap Resolution

| Aspect | Current | Resolution |
|---|---|---|
| DryRunSubmitter fill price | `"0"` | NATS KV lookup: `OBSERVATION_LATEST` for symbol's last close price |
| PaperVenueAdapter fill price | `"0"` | Same KV lookup mechanism |
| Fallback | N/A | `"0"` if KV unavailable or empty (best-effort) |

**Constraint:** Price realism is not safety-critical.  The fail-closed
guarantee is that dry-run never delegates to the real adapter.  Price value
is informational.

## Non-Goals Reaffirmed

| ID | Non-Goal |
|---|---|
| NG-5 | Advanced order types (limit, stop, OCO) |
| NG-6 | Order amendments and cancellations |
| NG-7 | Async order lifecycle (WebSocket fills) |
| NG-14 | State machine extension |

The seven-state model is sufficient.  The `sent` state exists for forward
compatibility with async models but is not exercised by any current adapter.

## Preparation for S384

S384 (Write-Path Integration Across Modes) should:

1. **Implement exhaustive lifecycle invariant tests:**
   - File: `internal/domain/execution/s383_lifecycle_invariants_test.go`
   - Cover all 49 transition pairs (10 valid, 39 invalid)
   - Cover TERM-1 through TERM-5
   - Cover FR-1 through FR-9
   - Cover IFC-1 through IFC-7
   - Cover QM-1 through QM-3, SM-1 through SM-4

2. **Close G1 (price realism):**
   - Modify `DryRunSubmitter` to accept a price source (NATS KV or interface)
   - Use last observed close price for fill records
   - Fallback to `"0"` if unavailable

3. **Write integration tests per mode:**
   - dry_run: composed pipeline produces correct transitions
   - paper: composed pipeline produces correct transitions
   - venue_live (testnet): composed pipeline produces correct transitions
   - safety gate: blocks correctly across all modes
   - correlation chain: preserved through composed pipeline

4. **Deliver:**
   - Tests: `internal/domain/execution/s383_lifecycle_invariants_test.go`
   - Tests: `internal/application/execution/s384_write_path_integration_test.go`
   - Architecture doc: `docs/architecture/oms-write-path-integration-across-modes.md`
   - Stage report: `docs/stages/stage-s384-write-path-integration-report.md`

## Promoted Documents

| Document | Location |
|---|---|
| Canonical Order Model and Lifecycle State Machine | `docs/architecture/canonical-order-model-and-lifecycle-state-machine.md` |
| Order Lifecycle Invariants, Transitions, and Boundaries | `docs/architecture/order-lifecycle-invariants-transitions-and-boundaries.md` |

## Acceptance Criteria Evaluation

| Criterion | Status |
|---|---|
| Canonical order model explicit and auditable | **MET** — `ExecutionIntent` fully mapped with 16 fields, 3 sides, 7 states |
| State machine clear with exhaustive transitions | **MET** — 10 valid, 39 invalid, 3 terminal, all cataloged |
| Ownership and boundaries clean | **MET** — derive/execute/store/gateway responsibilities defined |
| Stage prepares write-path integration for S384 | **MET** — 49 invariants cataloged, test plan specified, G1 resolution directed |

## Verdict

**S383 COMPLETE.** The canonical order model and lifecycle state machine are
documented, mapped across modes, and ready for invariant testing and
write-path integration in S384.
