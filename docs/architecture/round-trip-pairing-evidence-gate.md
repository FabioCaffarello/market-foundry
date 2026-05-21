# Round-Trip Pairing Evidence Gate

## Purpose

Formal evidence gate for the Round-Trip Pairing wave (S479–S483).

This document evaluates whether the wave delivered its chartered capabilities,
answered its governing questions, and strengthened the Foundry's round-trip
pairing, attribution, and effectiveness measurement layers with concrete,
verifiable evidence.

## Predecessor

The Strategy Effectiveness Measurement wave (S474–S478) closed with verdict
**PASS** and identified residual gap **G-SE1 MEDIUM**: single-leg fills dominate
effectiveness outcomes, causing most evaluations to return `unresolved`.

The Round-Trip Pairing wave was opened specifically to address G-SE1.

## Wave Structure

| Stage | Role | Status |
|-------|------|--------|
| S479 | Charter and scope freeze | COMPLETE |
| S480 | Canonical round-trip and leg-pairing model | COMPLETE |
| S481 | Pairing read model and attribution integration | COMPLETE |
| S482 | Round-trip review and outcome reconciliation | COMPLETE |
| S483 | Evidence gate (this document) | COMPLETE |

## Governing Questions — Verdict

| ID | Question | Answer | Evidence |
|----|----------|--------|----------|
| Q-RT1 | Can the system identify and pair entry/exit legs from existing execution data? | **YES** | `MatchFIFO()` with 7 invariants (M1–M7), 12 FIFO matching tests, `IntentToLeg()` with direction inference |
| Q-RT2 | Does pairing increase the resolved rate over single-leg classification? | **YES** | `executeBatch()` in `get_effectiveness.go` now runs FIFO matching before classification; `TestGetEffectivenessSummary_PairingIntegration_ReducesUnresolved` proves reduced unresolved count |
| Q-RT3 | Are paired outcomes correctly classified with accurate P&L? | **YES** | `ClassifyPair()` computes gross/net P&L; `TestGetPairing_Batch_PairedRoundTrip` (win), `TestGetPairing_Batch_LossRoundTrip` (loss); reconciliation flags surface unreliable P&L |
| Q-RT4 | Can the system surface paired outcomes and flag unmatched legs? | **YES** | Three HTTP endpoints: `/pairing`, `/pairing/chain`, `/pairing/review`, `/pairing/review/chain`; state/side/outcome/flagged filters; `UnmatchedReason` codes for every unmatched leg |
| Q-RT5 | Is pairing computable from existing data without new infrastructure? | **YES** | Read-path only; uses existing `CompositeReader` over existing ClickHouse tables; zero new tables, zero write-path changes |

All five governing questions answered **YES**. No partial or deferred answers remain.

## Capability Classification

| ID | Capability | Verdict | Evidence |
|----|-----------|---------|----------|
| C-RT1 | Canonical Round-Trip Model | **FULL** | `Leg`, `RoundTrip`, `PairingState`, `UnmatchedReason` types in `internal/domain/pairing/pairing.go`; 26 domain tests; architecture doc |
| C-RT2 | FIFO Leg-Matching Strategy | **FULL** | `MatchFIFO()` with 7 invariants, partial-fill proportional scaling, determinism guarantee; 12 matching tests |
| C-RT3 | Pairing Read Model | **FULL** | `GetPairingUseCase` with batch/single paths, CompositeReader integration, state/side filters; 9 pairing read model tests |
| C-RT4 | Paired Batch Effectiveness Integration | **FULL** | `executeBatch()` modified to run FIFO → ClassifyPair before single-leg classify; 3 integration tests prove reduced unresolved |
| C-RT5 | Round-Trip Review Endpoint | **FULL** | `GetRoundTripReviewUseCase` with outcome/flagged/state/side filters; 10 review tests |
| C-RT6 | Outcome Reconciliation Surface | **FULL** | 8 reconciliation flags, 3 reliability signals (`clean`, `fee_reliable`, `pnl_reliable`); 7 domain reconciliation tests |

**Result: 6/6 capabilities FULL.**

## Guard Rail Compliance

| # | Guard Rail | Compliant | Evidence |
|---|-----------|-----------|----------|
| 1 | No OMS expansion | YES | No position tracking, no portfolio engine added |
| 2 | No new ClickHouse tables | YES | All queries use existing execution/fill tables via CompositeReader |
| 3 | No new exchange connectivity | YES | No adapter or venue code modified |
| 4 | No write-path changes | YES | All work is read-path only; `cmd/writer/` untouched by wave |
| 5 | No portfolio analytics | YES | Per-decision attribution only, no cross-symbol aggregation |
| 6 | No real-time streaming | YES | Batch read-path computation at query time |
| 7 | No domain type refactoring | YES | New `internal/domain/pairing/` package; existing types unchanged |
| 8 | No UI or dashboards | YES | HTTP JSON endpoints only |
| 9 | No risk/position engine | YES | No position state, no risk engine |
| 10 | Additive only | YES | Zero changes to existing behavior; all modifications are additive wiring |

All 10 guard rails respected. No violations.

## Test Evidence

| Package | Tests | Pass | Regressions |
|---------|-------|------|-------------|
| `internal/domain/pairing` | 33 (26 pairing + 7 reconciliation) | 33 | 0 |
| `internal/domain/effectiveness` | existing suite | all | 0 |
| `internal/application/analyticalclient` | 22 new (12 S481 + 10 S482) + existing | all | 0 |

**Total new tests in wave: 55.**
**Regressions: zero across all affected packages.**

## Non-Goal Adherence

The charter froze 18 non-goals (NG-RT1 through NG-RT18). None were violated.
Key non-goals that remained correctly frozen:

- NG-RT6: No cross-session pairing (session-scoped or correlation-ID scoped only)
- NG-RT7: No LIFO/HIFO (FIFO only)
- NG-RT1: No OMS expansion
- NG-RT14: No ML scoring
- NG-RT16: No derivatives support beyond existing futures path

## G-SE1 Resolution Assessment

**Before this wave**: All single-leg fills classified as `unresolved`. Win rate computed on tiny subsample where both entry and exit happen to appear in the same evaluation window.

**After this wave**:
1. `MatchFIFO()` automatically finds entry/exit pairs from existing fill data
2. Paired round-trips classified as `win`/`loss`/`breakeven` via `ClassifyPair()`
3. `resolved_rate` metric quantifies the improvement per query
4. Reconciliation flags surface *why* some pairs still have unreliable P&L

**G-SE1 status: RESOLVED within wave scope.** The structural gap (no pairing mechanism) is closed. Residual unresolved outcomes now come from genuine data-quality conditions (paper fills, futures fee gap, session boundaries), not from missing infrastructure.

## Wave Verdict

**PASS.**

The Round-Trip Pairing wave delivered all 6 chartered capabilities at FULL level,
answered all 5 governing questions affirmatively, respected all 10 guard rails
and 18 non-goals, produced 55 new tests with zero regressions, and resolved
G-SE1 within its defined scope.

## Residual Gaps

See companion document: [round-trip-pairing-evidence-matrix-residual-gaps-and-next-ceremony.md](round-trip-pairing-evidence-matrix-residual-gaps-and-next-ceremony.md).
