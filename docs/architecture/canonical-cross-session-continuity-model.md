# Canonical Cross-Session Continuity Model

**Stage**: S494
**Wave**: Cross-Session Position Continuity (S493–S497)
**Date**: 2026-03-26
**Predecessor**: S493 (Charter and Scope Freeze — COMPLETE)

---

## 1. Purpose

This document defines the canonical model for cross-session continuity in
market-foundry. It answers a precise question: **what does it mean for an
execution leg to carry forward across a session boundary, and how does the
system represent that relationship?**

The model is read-side only. It does not modify the write path, carry state
at runtime, or introduce position tracking. Sessions remain isolated at
runtime; continuity is computed retrospectively from existing fill data.

---

## 2. Domain Types

### 2.1 ContinuityState

Classifies whether a leg's lifecycle is resolved when considering data from
multiple sessions.

| Value | Meaning |
|-------|---------|
| `resolved` | Leg was paired with a counterpart (intra-session or cross-session). Round-trip closed. |
| `open` | Leg has no counterpart across all sessions in the lookback window. May resolve in a future session. |
| `genuine_unresolved` | Leg cannot resolve due to a structural condition (rejected, cancelled, orphan exit). Permanently unresolved. |
| `artificial_unresolved` | Leg classified as unresolved solely because it sits at a session boundary. A counterpart may exist in an adjacent session. Primary target of cross-session resolution. |

**Key distinction**: `artificial_unresolved` is reducible — the system can
eliminate it by expanding the pairing window across sessions. `genuine_unresolved`
is irreducible — no amount of cross-session data will produce a valid counterpart.

### 2.2 SessionLeg

A `Leg` (from `internal/domain/pairing`) annotated with session provenance:

```
SessionLeg {
    Leg                     // embedded: direction, side, symbol, source, timeframe,
                            //           correlation_id, price, qty, fee, cost_basis,
                            //           simulated, timestamp
    SessionID       string  // "session_20260326_120030"
    SessionStartedAt time.Time
    SessionClosedAt  *time.Time
}
```

**Why**: When an entry from session N pairs with an exit from session M, both
session IDs must be available for audit, reconciliation, and operator review.
The existing `Leg` type does not carry session origin; `SessionLeg` adds it
without modifying the existing type.

### 2.3 CrossSessionWindow

Defines the temporal and filtering scope for a cross-session discovery query:

```
CrossSessionWindow {
    Symbol       string     // instrument filter
    Source       string     // venue/segment filter (e.g. "binance_spot")
    Timeframe    int        // candle interval filter
    Since        time.Time  // earliest session start (inclusive)
    Until        time.Time  // latest session close (inclusive, zero = now)
    MaxSessions  int        // limit on number of sessions (most recent first)
}
```

**Defaults**: `DefaultLookbackDays = 30`, `DefaultMaxSessions = 30`.

**Validation**: Symbol, Source, Timeframe, and Since are all required.

### 2.4 CrossSessionLegSet

An ordered collection of `SessionLeg` instances drawn from multiple sessions:

```
CrossSessionLegSet {
    Window     CrossSessionWindow   // the scope that produced this set
    Sessions   []string             // contributing session IDs (chronological)
    Legs       []SessionLeg         // ordered by timestamp (ascending)
}
```

**Key property**: Legs are sorted by timestamp across all sessions. This
preserves the FIFO temporal invariant (M4) that `MatchFIFO` requires. The
algorithm does not know or care that legs originate from different sessions.

**Operations**:
- `ExtractLegs()` → `[]Leg` — strips session metadata for `MatchFIFO` input.
- `SessionLegIndex()` → `map[CorrelationID]SessionLeg` — for post-matching provenance lookup.

### 2.5 CarryForwardEligibility

Classifies whether an execution intent is eligible for cross-session carry-forward:

| Value | Condition | Eligible? |
|-------|-----------|-----------|
| `eligible` | Filled, terminal, has fill records | Yes |
| `ineligible_rejected` | Status = rejected | No — no fills |
| `ineligible_cancelled` | Status = cancelled, no fills | No — no fills |
| `ineligible_non_terminal` | Status ∈ {submitted, sent, accepted, partially_filled} | No — lifecycle incomplete |
| `ineligible_no_fills` | Terminal but zero fill records | No — no leg data |
| `ineligible_already_paired` | Paired within its session | No — already resolved |

### 2.6 CrossSessionRoundTrip

A `RoundTrip` annotated with session provenance and continuity classification:

```
CrossSessionRoundTrip {
    RoundTrip                     // embedded: entry, exit, state, unmatched_reason, etc.
    EntrySessionID   string       // session that produced entry leg
    ExitSessionID    string       // session that produced exit leg
    CrossSession     bool         // true when entry and exit from different sessions
    Continuity       ContinuityState  // lifecycle resolution classification
}
```

---

## 3. Carry-Forward Rules

These are the canonical rules that determine what crosses a session boundary.

### 3.1 Rules

| ID | Rule | Rationale |
|----|------|-----------|
| R-CF1 | Rejected intents are ineligible | No fills → no leg → nothing to carry |
| R-CF2 | Cancelled-before-fill intents are ineligible | Same as R-CF1 |
| R-CF3 | Non-terminal intents are ineligible | Lifecycle incomplete; the intent may still transition |
| R-CF4 | Terminal intents with zero fills are ineligible | Edge case: terminal but no trade data |
| R-CF5 | Filled intents with fills are eligible | The only case where a valid leg exists for carry-forward |

### 3.2 Already-paired filter

An intent that is eligible by R-CF5 but was already paired within its
originating session (intra-session `MatchFIFO` produced a `StatePaired`
round-trip) is not a carry-forward candidate. The caller applies this
filter after running intra-session matching.

### 3.3 Direction is preserved

Carry-forward preserves the leg direction (entry/exit) and side (buy/sell)
from the originating session. The cross-session matching algorithm does not
reinterpret direction.

---

## 4. Continuity Classification Rules

| ID | Condition | Classification |
|----|-----------|---------------|
| C-1 | Paired round-trip (both legs present) | `resolved` |
| C-2 | Unmatched entry with `ReasonSessionBoundary` | `artificial_unresolved` |
| C-3 | Unmatched entry with `ReasonRejectedLeg` or `ReasonCancelledLeg` | `genuine_unresolved` |
| C-4 | Unmatched entry with `ReasonNoExitFound` | `open` |
| C-5 | Unmatched exit (orphan) | `genuine_unresolved` |
| C-6 | Unmatched entry with `ReasonQuantityMismatchResidue` | `open` |

**Interpretation**:

- **`resolved`**: The trade lifecycle is complete. P&L is computable.
- **`open`**: No counterpart exists in the lookback window, but the position
  may still be live. Not an error — the exit may occur in a future session.
- **`genuine_unresolved`**: Structural failure. The leg cannot resolve
  regardless of future data. Includes rejects, cancels, and orphan exits.
- **`artificial_unresolved`**: Session-boundary artifact. The leg was
  classified as unresolved only because pairing ran within a single session.
  Cross-session matching can potentially resolve it.

---

## 5. Cross-Session Pairing Flow

The canonical flow for cross-session pairing is:

```
1. Construct CrossSessionWindow (symbol, source, timeframe, lookback)
2. Query sessions within window (from Session KV or ClickHouse)
3. For each session:
   a. Query execution intents from ClickHouse
   b. Filter by CarryForwardEligibility (R-CF1 through R-CF5)
   c. Convert to SessionLeg via IntentToLeg + session metadata
4. Collect into CrossSessionLegSet (sorted by timestamp)
5. Extract plain Legs via ExtractLegs()
6. Run MatchFIFO(legs, DefaultMatchingConfig())
7. Build SessionLegIndex from the leg set
8. AnnotateRoundTrips(roundTrips, sessionIndex)
   → produces CrossSessionRoundTrip[] with provenance and continuity
```

**Key insight**: Steps 5–6 reuse the existing `MatchFIFO` algorithm with
zero modifications. The algorithm is session-agnostic; it pairs by symbol,
source, opposite side, and temporal ordering. Cross-session pairing is
achieved by widening the input set, not by changing the matching rules.

---

## 6. Invariants

| ID | Invariant | Enforced by |
|----|-----------|-------------|
| INV-1 | `MatchFIFO` matching rules (M1–M7) apply unchanged across sessions | Algorithm is session-agnostic |
| INV-2 | Session provenance is preserved through pairing | `SessionLegIndex` + `AnnotateRoundTrips` |
| INV-3 | Cross-session pairing does not modify intra-session results | Additive query; existing pairing paths unchanged |
| INV-4 | Carry-forward eligibility is deterministic | Pure function `ClassifyCarryForward` |
| INV-5 | Continuity classification is deterministic | Pure function `ClassifyContinuity` |
| INV-6 | Lookback window is bounded | `DefaultLookbackDays` and `DefaultMaxSessions` |

---

## 7. What This Model Does NOT Do

| Exclusion | Rationale |
|-----------|-----------|
| Track live position state | Read-side only; no runtime carry-forward (GR-7) |
| Modify the `Leg` type | `SessionLeg` wraps `Leg`; existing type unchanged |
| Create new ClickHouse tables | Queries existing `executions` table |
| Carry runtime state between sessions | Sessions remain isolated (GR-1, GR-7) |
| Compute portfolio-level P&L | Per-symbol, per-segment only (GR-2) |
| Introduce position accumulator | Retrospective matching, not inventory (GR-2) |

---

## 8. Relationship to Existing Infrastructure

| Component | Role in Cross-Session Continuity |
|-----------|----------------------------------|
| `MatchFIFO` (S480) | Unchanged; receives cross-session leg list as input |
| `IntentToLeg` (S480) | Unchanged; converts any intent to a leg |
| `ClassifyPair` (S475) | Unchanged; classifies any entry+exit pair |
| `CompositeReader` | Provides time-range queries for multi-session data |
| Session KV store (S460) | Provides session metadata for window construction |
| `ReasonSessionBoundary` (S480) | Existing marker; used by `ClassifyContinuity` |
| `ReconcileRoundTrip` (S482) | Extended in S496 to flag cross-session pairs |

---

## 9. Preparation for S495

S495 (Read Model and Attribution) builds on this model by:

1. Implementing the discovery query (step 2–3 of the pairing flow).
2. Wiring `CrossSessionLegSet` construction into a use case.
3. Invoking `MatchFIFO` + `AnnotateRoundTrips` for cross-session results.
4. Applying `ClassifyPair` for P&L attribution on cross-session pairs.
5. Exposing results via an HTTP endpoint.

All domain types, rules, and invariants defined here are the contract that
S495 consumes. No additional domain modeling is required.

---

## 10. File Map

| File | Purpose |
|------|---------|
| `internal/domain/pairing/continuity.go` | All domain types defined in this document |
| `internal/domain/pairing/s494_continuity_test.go` | Tests for carry-forward, continuity classification, session leg, cross-session matching |
| `internal/domain/pairing/pairing.go` | Existing types (`Leg`, `RoundTrip`, `MatchFIFO`) — unchanged |
| `internal/domain/pairing/reconciliation.go` | Existing reconciliation flags — unchanged (extended in S496) |
