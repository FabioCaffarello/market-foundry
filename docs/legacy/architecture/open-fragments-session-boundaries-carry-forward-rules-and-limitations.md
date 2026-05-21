# Open Fragments, Session Boundaries, Carry-Forward Rules, and Limitations

**Stage**: S494
**Wave**: Cross-Session Position Continuity (S493–S497)
**Date**: 2026-03-26
**Companion to**: [canonical-cross-session-continuity-model.md](canonical-cross-session-continuity-model.md)

---

## 1. Purpose

This document defines the operational semantics of session boundaries,
explains what happens to open fragments (unmatched legs) at session close,
specifies the carry-forward rules that govern cross-session resolution,
and explicitly catalogs the limitations and residual ambiguities.

It is the reference for operators and developers who need to understand:
- Why some legs appear as `unresolved` after a session.
- Which of those are resolvable by looking across sessions.
- What the system will and will not attempt to resolve.

---

## 2. Session Boundary Semantics

### 2.1 What is a session boundary?

A session boundary is the timestamp at which a session transitions from
`open` to a terminal state (`closed` or `halted`). Specifically:

- **Open boundary** (`StartedAt`): The execute binary starts, creates a
  `Session` record in NATS KV, and begins accepting execution intents.
- **Close boundary** (`ClosedAt`): The execute binary shuts down gracefully
  (`closed`) or is halted with a reason (`halted`). Segment counters are
  recorded. A `SessionLifecycleEvent` is published.

### 2.2 What happens at session close?

At the close boundary:

1. **No in-flight cancellation**: Orders that have been submitted to the
   venue are not automatically cancelled. If an order is in `accepted` or
   `partially_filled` state when the session closes, it remains in that
   state in ClickHouse. The venue may still fill it after the session ends.

2. **No carry-forward in the write path**: The next session does not receive
   any state from the previous session. It starts clean. There is no
   "position handoff" between sessions.

3. **Counters are finalized**: The session records `processed`, `filled`,
   `rejected`, and `errors` per segment at close time.

4. **Pairing is scoped**: Intra-session pairing (`MatchFIFO` via
   `GetPairingUseCase`) runs over legs from a single time window. Legs
   without counterparts receive `ReasonNoExitFound` or `ReasonSessionBoundary`.

### 2.3 The boundary gap

The gap is that an entry fill in session N and a corresponding exit fill in
session N+1 are never seen together by intra-session pairing. The entry is
marked `unmatched_entry` with `ReasonSessionBoundary`, and the exit may be
marked `unmatched_exit` with `ReasonNoEntryFound` — even though together
they form a valid round-trip.

This is the structural source of **artificial unresolved** outcomes.

---

## 3. Open Fragments — Taxonomy

An "open fragment" is a filled leg that has no counterpart within its session.
Not all open fragments are equal:

### 3.1 Artificial unresolved (session boundary artifact)

**Signature**: `StateUnmatchedEntry` + `ReasonSessionBoundary`

**Meaning**: The leg was filled and represents a genuine position, but the
session ended before a matching exit occurred. The exit may exist in a
subsequent session.

**Resolvable**: Yes — by extending the pairing window across sessions.

**Example**: Session 1 buys 0.1 BTC at 10:00. Session 1 closes at 12:00.
Session 2 starts at 14:00 and sells 0.1 BTC at 15:00. The buy from session
1 and the sell from session 2 form a valid round-trip.

### 3.2 Genuinely open (position still live)

**Signature**: `StateUnmatchedEntry` + `ReasonNoExitFound`

**Meaning**: The entry was filled, but no matching exit exists in any
session within the lookback window. The position may still be live on
the exchange, or the exit may occur in a future session beyond the window.

**Resolvable**: Not yet — but may resolve when the lookback window expands
or new sessions produce the exit.

### 3.3 Genuine unresolved (structural failure)

**Signature**: Various, including:
- `StateUnmatchedEntry` + `ReasonRejectedLeg`
- `StateUnmatchedEntry` + `ReasonCancelledLeg`
- `StateUnmatchedExit` + `ReasonNoEntryFound`

**Meaning**: The leg cannot form a valid round-trip due to a structural
condition. Rejected orders have no fills. Cancelled-before-fill orders have
no fills. Orphan exits have no corresponding entry.

**Resolvable**: No — regardless of how many sessions are scanned.

### 3.4 Partial remainder (quantity split)

**Signature**: `StateUnmatchedEntry` + `ReasonQuantityMismatchResidue`

**Meaning**: Part of the leg was paired (e.g., bought 0.2, sold 0.1), and
the remainder (0.1) is unmatched. The remainder may pair with a future exit.

**Resolvable**: Potentially — if a future session produces a matching exit
for the remaining quantity.

---

## 4. Carry-Forward Rules

### 4.1 Definition

"Carry-forward" means: including a leg from a prior session in a
cross-session discovery query so that it may pair with a leg from a later
session.

Carry-forward is a **read-side operation**. It does not move data, create
new records, or modify the original session's state.

### 4.2 Eligibility rules

| Rule | Condition | Eligible? | Rationale |
|------|-----------|-----------|-----------|
| R-CF1 | `Status = rejected` | No | No fills, no leg, nothing to carry |
| R-CF2 | `Status = cancelled` and zero fills | No | No fills, no leg |
| R-CF3 | `Status ∈ {submitted, sent, accepted, partially_filled}` | No | Lifecycle incomplete; may still transition |
| R-CF4 | Terminal status but zero fill records | No | Edge case: status says done, but no trade data |
| R-CF5 | `Status ∈ {filled, cancelled}` and fill count > 0 | Yes | Valid fill data exists for pairing |

### 4.3 Already-paired filter

After applying R-CF1 through R-CF5, a second filter removes legs that
were already paired within their originating session. This prevents
double-counting: if session 1 had a buy and a sell that paired
intra-session, neither should appear in the cross-session candidate set.

The filter is applied by the caller (use case), not by the eligibility
classifier, because it requires the result of intra-session `MatchFIFO`.

### 4.4 Direction and side preservation

The leg direction (`entry`/`exit`) and side (`buy`/`sell`) from the
originating session are preserved verbatim. The cross-session matching
algorithm does not reinterpret them.

This means: if session 1 classified a buy as an entry (long strategy),
cross-session matching will look for a sell exit. It will not consider
the possibility that the strategy direction changed between sessions.

**Limitation**: If the operator runs session 1 with a long strategy and
session 2 with a short strategy for the same symbol, cross-session pairing
will not pair them (the sides won't align as entry/exit). This is correct
behavior — mixing strategy directions within a continuity window is not
supported and should be treated as a separate trading context.

---

## 5. Unresolved Real vs Unresolved Artificial

This is the core semantic distinction that S494 formalizes:

### 5.1 Unresolved artificial

- **Cause**: Session boundary cuts the pairing window.
- **Marker**: `ReasonSessionBoundary` on the unmatched leg.
- **Continuity state**: `artificial_unresolved`.
- **Action**: Cross-session matching can resolve it.
- **Prevalence**: Every session that closes with open positions produces
  these. This is the normal, expected case for strategies that hold
  positions across trading days.

### 5.2 Unresolved genuine

- **Cause**: Structural failure in the execution lifecycle.
- **Markers**: `ReasonRejectedLeg`, `ReasonCancelledLeg`, orphan exit.
- **Continuity state**: `genuine_unresolved`.
- **Action**: None — no amount of data will resolve it. Flagged for
  operator review.
- **Prevalence**: Low under normal operation. High prevalence indicates
  execution pipeline issues.

### 5.3 Unresolved open (ambiguous)

- **Cause**: No matching exit in the lookback window.
- **Marker**: `ReasonNoExitFound` without `ReasonSessionBoundary`.
- **Continuity state**: `open`.
- **Action**: May resolve when the lookback window extends or new sessions
  arrive. Cannot be classified as artificial or genuine without more data.
- **Prevalence**: Normal for positions opened near the end of the lookback
  window.

### 5.4 Decision flow for an operator

```
Unmatched entry leg
  │
  ├─ ReasonSessionBoundary? → artificial_unresolved
  │     └─ Run cross-session matching → may become resolved
  │
  ├─ ReasonRejectedLeg or ReasonCancelledLeg? → genuine_unresolved
  │     └─ Investigate execution pipeline
  │
  ├─ ReasonQuantityMismatchResidue? → open (partial)
  │     └─ May resolve with future exit for remaining qty
  │
  └─ ReasonNoExitFound? → open
        └─ Position may still be live; wait for future session
```

---

## 6. Lookback Window Semantics

### 6.1 Bounded by design

Cross-session queries operate within a bounded lookback window. This is a
deliberate constraint:

- **Default**: 30 days or 30 sessions (whichever is more restrictive).
- **Rationale**: Unbounded lookback has O(N) cost in sessions scanned and
  legs processed. For typical market-foundry usage, 30 days covers most
  position hold periods.

### 6.2 Window edge effects

Legs at the edge of the lookback window may have counterparts outside the
window. These will appear as `open` (not `artificial_unresolved`, because
there is no `ReasonSessionBoundary` relative to the window edge — the
marker is set at session close time, not at query time).

This is an accepted limitation. The operator can extend the lookback
window to resolve edge cases.

### 6.3 Window does not modify data

The lookback window is a query-time parameter. It does not modify session
records, leg data, or pairing state. Running the same query with different
windows will produce different results. The results are deterministic for
a given window and dataset.

---

## 7. Interaction with Existing Surfaces

### 7.1 Intra-session pairing (unchanged)

The existing `GET /analytical/pairing` endpoint continues to operate within
a single session's time window. It is not modified. Operators who want
single-session views continue to use this endpoint.

### 7.2 Effectiveness measurement (unchanged)

Single-leg `Classify()` still returns `OutcomeUnresolved` for legs without
paired exits. Cross-session effectiveness is computed separately via
`ClassifyPair()` on cross-session round-trips (S495).

### 7.3 Reconciliation flags (extended in S496)

`ReconcileRoundTrip` will gain a `FlagCrossSession` in S496 to mark
cross-session pairs. This flag is informational — it does not affect
`PnLReliable` or `FeeReliable` assessments.

### 7.4 Session audit bundle (optionally extended in S496)

The audit bundle may gain a cross-session continuity section showing how
many legs from a given session were resolved by pairing with other sessions.

---

## 8. Limitations

### L-1: No runtime carry-forward

Sessions do not share state at runtime. The execute binary starts clean
every time. This means:

- The system does not "know" it has an open position when a session starts.
- There is no automatic position-aware behavior (e.g., skip entries if
  already long).
- Cross-session awareness is purely retrospective.

### L-2: Strategy direction must be consistent

Cross-session pairing assumes consistent strategy direction within the
continuity window. If session 1 runs a long strategy and session 2 runs
short for the same symbol, their legs will not pair (buy entry from
session 1 needs sell exit, but session 2's sell would be classified as an
entry under short strategy).

### L-3: Non-terminal orders at session close are not resolved

If an order is in `accepted` state when the session closes, it is not
eligible for carry-forward (R-CF3). The venue may fill it later, but
the system has no mechanism to observe post-session fills for orders
submitted in a prior session.

**Mitigation**: The vast majority of market orders reach terminal state
within seconds. `accepted`-at-close is rare.

### L-4: Lookback window is finite

Positions held longer than the lookback window (default 30 days) will not
be resolved by cross-session matching. They will appear as `open`.

**Mitigation**: Operator can increase the lookback window for specific
queries. The system does not enforce a hard maximum, but performance
degrades for very large windows.

### L-5: No cross-symbol or cross-segment pairing

Cross-session matching operates per-symbol, per-segment. An entry in
BTCUSDT on `binance_spot` cannot pair with an exit in BTCUSDT on
`binance_futures`. This matches the existing `MatchFIFO` invariant M2.

### L-6: Fee schedule changes between sessions

If the exchange changes fee tiers between sessions, the entry and exit
legs will have fees recorded at different rates. This is correct — fees
are recorded at fill time, and P&L computation uses recorded fees.
However, the operator should be aware that fee comparisons across
sessions reflect the fee schedule at the time of each fill.

### L-7: No deduplication guard for concurrent queries

If two concurrent cross-session queries run with overlapping windows,
they will independently produce results. There is no shared cache or
deduplication. This is acceptable because cross-session pairing is
a read-side computation with no side effects.

---

## 9. Trade-Offs

| Decision | Alternative considered | Why this choice |
|----------|----------------------|-----------------|
| `SessionLeg` wraps `Leg` instead of extending it | Add `SessionID` field to `Leg` | Avoids modifying the existing type; `Leg` remains lightweight for intra-session use |
| Carry-forward is determined by execution status, not pairing state | Let the carry-forward rule check intra-session pairing directly | Separation of concerns: eligibility is about the intent's lifecycle; the already-paired filter is about the pairing result |
| Lookback bounded by default | Unbounded lookback | Performance and relevance; most positions close within days |
| Four-state continuity model | Binary (resolved / unresolved) | The four-state model distinguishes actionable from non-actionable unresolved, which is critical for operator triage |
| `artificial_unresolved` keyed on `ReasonSessionBoundary` | Infer from session timestamps | `ReasonSessionBoundary` is explicitly set by intra-session pairing; inference would be fragile |

---

## 10. What Stays Out of Scope

| Topic | Why excluded |
|-------|-------------|
| Position engine | GR-2: retrospective matching, not live tracking |
| Portfolio P&L | GR-2: per-symbol only |
| Runtime carry-forward | GR-7: sessions remain isolated |
| Multi-exchange pairing | GR-4: Binance only |
| Automatic position-aware execution | GR-1: no write-path changes |
| Risk/exposure from open positions | GR-2: no risk engine |
