# Stage S480 — Canonical Round-Trip and Leg-Pairing Model

**Wave**: Round-Trip Pairing (S479--S483)
**Stage type**: Domain modeling + matching rules
**Date**: 2026-03-26
**Predecessor**: S479 (Wave Charter)

---

## 1. Objective

Design, validate, and document a canonical model for round-trip trades and leg pairing that:
- Defines unambiguous semantics for entry/exit legs, open/closed/unresolved states, and realized/unrealized results
- Provides deterministic FIFO matching rules with auditable invariants
- Handles partial fills, unmatched legs, and edge cases with explicit reason codes
- Prepares the foundation for read-model integration (S481) and attribution pipeline wiring

---

## 2. What Was Done

### 2.1 Codebase Analysis

Mapped how fills, intents, and outcomes appear today across:
- `internal/domain/execution/execution.go` — ExecutionIntent, FillRecord, Side, Status, lifecycle state machine
- `internal/domain/effectiveness/effectiveness.go` — Classify(), ClassifyPair(), Attribution, Outcome
- `internal/domain/lineage/lineage.go` — ChainLink, Chain, causal stage ordering
- `internal/domain/consistency/consistency.go` — cross-domain consistency checks

**Key finding**: The pipeline processes individual ExecutionIntents as atomic units. `Classify()` always returns `unresolved` for single-leg fills because it has no exit to pair against. `ClassifyPair(entry, exit)` exists and works correctly but is never called in the batch evaluation pipeline because there is no mechanism to find and match pairs.

### 2.2 Domain Types Created

New package: `internal/domain/pairing/`

| Type | Purpose |
|------|---------|
| `Leg` | One side of a round-trip: direction, side, symbol, source, fill data, correlation_id |
| `LegDirection` | `entry` \| `exit` |
| `RoundTrip` | Paired or unpaired trade lifecycle with entry, exit, state, reason, matched quantity |
| `PairingState` | `paired` \| `unmatched_entry` \| `unmatched_exit` |
| `UnmatchedReason` | `no_exit_found` \| `no_entry_found` \| `quantity_mismatch_remainder` \| `session_boundary` \| `rejected_leg` \| `cancelled_leg` |
| `MatchingConfig` | Controls partial-match behavior |
| `PairingResult` | Summary statistics: paired count, unmatched counts, resolved rate |

### 2.3 FIFO Matching Algorithm

Implemented `MatchFIFO(legs, config)` with 7 matching invariants:

| ID | Invariant |
|----|-----------|
| M1 | Same symbol |
| M2 | Same source/segment |
| M3 | Opposite side |
| M4 | Temporal ordering (entry <= exit) |
| M5 | FIFO priority (earliest first) |
| M6 | One-to-one (no double-counting) |
| M7 | Deterministic (same input → same output) |

Partial-fill handling: proportional scaling of cost basis and fees when quantities differ, with remainder as separate unmatched leg.

### 2.4 Intent-to-Leg Conversion

Implemented `IntentToLeg(intent, strategyDirection)` that:
- Aggregates multiple fills into a single leg (summed quantities/fees/costs, weighted average price)
- Infers direction from side + strategy direction (long/short)
- Preserves CorrelationID for lineage traceability
- Handles edge cases: no fills, simulated fills, single fill

### 2.5 Documentation

Created two architecture documents:

1. **canonical-round-trip-and-leg-pairing-model.md** — Core entity definitions, state semantics, matching rules, integration points, limitations
2. **entry-exit-legs-pairing-rules-open-closed-unresolved-semantics-and-limitations.md** — Operational reference for leg identification, pairing rules, open/closed/unresolved taxonomy, edge cases, residual ambiguity

---

## 3. Test Coverage

26 tests covering:

| Category | Tests | What They Verify |
|----------|-------|-----------------|
| Type validation | 2 | LegDirection, PairingState enums |
| Intent-to-Leg conversion | 6 | Long/short direction inference, default convention, fill aggregation, no-fills fallback |
| FIFO matching | 12 | Empty input, single entry/exit, perfect pair, short pair, temporal ordering, symbol/source/side guards, partial match, multiple pairs, determinism, zero quantity |
| Summary statistics | 2 | Empty input, mixed results with resolved rate |
| RoundTrip methods | 2 | IsPaired(), IsOpen() |

All 26 tests pass. Zero regressions across all 12 domain packages.

---

## 4. Capabilities Addressed

| Capability | Status | Evidence |
|-----------|--------|---------|
| C-RT1: Canonical Round-Trip Model | FULL | `Leg`, `RoundTrip`, `PairingState`, `UnmatchedReason` types defined with complete semantics |
| C-RT2: FIFO Leg-Matching Strategy | FULL | `MatchFIFO()` implements all 7 invariants, partial-fill handling, deterministic output |

---

## 5. Governing Questions Progress

| ID | Question | Status | Evidence |
|----|----------|--------|---------|
| Q-RT1 | Can the system identify and pair entry/exit legs with canonical matching rules? | YES | 7 matching invariants, 12 matching tests, direction inference logic |
| Q-RT3 | Are paired outcomes correctly classified with accurate P&L? | PARTIAL | Matching produces paired round-trips; `ClassifyPair()` integration deferred to S481 |

---

## 6. What Today Was Implicit or Insufficient

| Before S480 | After S480 |
|-------------|------------|
| No formal definition of "round-trip" | `RoundTrip` type with explicit state, entry, exit, matched quantity |
| `unresolved` was a single opaque bucket | 6 distinct reason codes decompose unresolved into actionable categories |
| No definition of entry vs exit | Direction inference rules: long (buy=entry) / short (sell=entry) |
| No matching rules | 7 auditable invariants (M1-M7) |
| No partial-fill handling | Proportional scaling with remainder tracking |
| `ClassifyPair()` exists but has no input mechanism | `IntentToLeg()` + `MatchFIFO()` provide the structured inputs |
| No distinction between open/closed/partially resolved | Explicit state machine with 3 states and clear transitions |
| No distinction between realized/unrealized/not-classifiable | Three-category result model with conditions documented |

---

## 7. Guard Rails Observed

| Guard Rail | Observed |
|-----------|----------|
| No OMS expansion | Yes — no new order types, no position tracking |
| No new ClickHouse tables | Yes — pairing is domain logic only |
| No write-path changes | Yes — zero modifications to execution pipeline |
| No portfolio analytics | Yes — per-symbol, per-segment only |
| No domain type refactoring | Yes — existing types unchanged, new package added |
| Additive only | Yes — zero changes to existing behavior |
| No position or risk engine | Yes — unmatched entries are fragments, not positions |

---

## 8. Limitations Documented

1. FIFO only (no LIFO/HIFO) — aligns with temporal execution model
2. Single-venue — source must match exactly
3. No cross-session pairing beyond CorrelationID scope
4. Futures fee gap (Fee="0") carries through to round-trip P&L
5. Paper/dry-run zero pricing makes P&L unclassifiable
6. Strategy direction is required input; defaults to long
7. Float64 quantity precision with epsilon tolerance
8. No causal validation in matching (structural, not CorrelationID-based)

---

## 9. Residual Ambiguity

| Ambiguity | Status | Resolution Path |
|-----------|--------|-----------------|
| `session_boundary` vs `no_exit_found` reason assignment | Deferred | S481 read model has session context for this |
| Should matching enforce CorrelationID? | Decided: NO | Structural matching is broader; causal filtering is future |
| Futures funding rates in P&L | Out of scope (NG-RT15) | Documented limitation |

---

## 10. Files Changed

| File | Action |
|------|--------|
| `internal/domain/pairing/pairing.go` | NEW — Domain types and FIFO matching |
| `internal/domain/pairing/pairing_test.go` | NEW — 26 tests |
| `docs/architecture/canonical-round-trip-and-leg-pairing-model.md` | NEW — Canonical model document |
| `docs/architecture/entry-exit-legs-pairing-rules-open-closed-unresolved-semantics-and-limitations.md` | NEW — Operational semantics reference |
| `docs/stages/stage-s480-round-trip-model-report.md` | NEW — This report |

---

## 11. Next Stage

**S481 — Pairing Read Model and Attribution Integration**

S481 will:
- Implement leg-matching query from existing ClickHouse execution data via CompositeReader
- Wire matched pairs through `ClassifyPair()` to produce paired effectiveness attributions
- Integrate into the batch evaluation pipeline
- Assign `session_boundary` reason codes using session context
- Track matching statistics and resolved rate improvement

The domain model and matching algorithm from S480 are the inputs. S481 provides the read-path execution.

---

## 12. References

- [Wave Charter and Scope Freeze](../architecture/round-trip-pairing-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](../architecture/round-trip-pairing-capabilities-questions-and-non-goals.md)
- [Canonical Round-Trip Model](../architecture/canonical-round-trip-and-leg-pairing-model.md)
- [Entry/Exit Legs Pairing Rules](../architecture/entry-exit-legs-pairing-rules-open-closed-unresolved-semantics-and-limitations.md)
- [S479 Charter Report](stage-s479-round-trip-pairing-charter-report.md)
