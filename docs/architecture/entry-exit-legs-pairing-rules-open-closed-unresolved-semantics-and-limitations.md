# Entry/Exit Legs, Pairing Rules, Open/Closed/Unresolved Semantics, and Limitations

**Stage**: S480
**Wave**: Round-Trip Pairing (S479--S483)
**Date**: 2026-03-26

---

## 1. Purpose

This document is the operational reference for leg-pairing semantics. It specifies how entries and exits are identified, how pairing rules are applied, what each lifecycle state means in practice, and where ambiguity remains.

Read this document when you need to understand:
- What makes a fill an "entry" vs an "exit"
- When a position is considered "open", "closed", or "partially resolved"
- How unresolved cases are categorized and what they mean
- What the matching algorithm does and does not guarantee
- Where residual ambiguity exists and how it is handled

For the canonical type definitions, see [canonical-round-trip-and-leg-pairing-model.md](canonical-round-trip-and-leg-pairing-model.md).

---

## 2. Entry and Exit Leg Identification

### 2.1 Direction Inference

A leg's direction (entry or exit) is determined by two inputs:

1. **Order side**: `buy` or `sell` (from `ExecutionIntent.Side`)
2. **Strategy direction**: `long` or `short` (from the strategy that produced the execution)

| Strategy Direction | Buy | Sell |
|-------------------|-----|------|
| Long | Entry | Exit |
| Short | Exit | Entry |

When strategy direction is unknown or empty, the system defaults to **long convention** (buy = entry, sell = exit). This is documented as a limitation, not a silent assumption.

### 2.2 Leg Eligibility

A leg is eligible for pairing when:

| Condition | Required |
|-----------|----------|
| Intent has at least one fill | Yes |
| Intent status is `filled` or `partially_filled` | Yes |
| Intent side is `buy` or `sell` (not `none`) | Yes |
| Fill quantity is greater than zero | Yes |

Legs from rejected or cancelled intents (with no fills) are excluded from pairing entirely. Cancelled intents with partial fills retain the filled portion as an eligible leg.

### 2.3 Fill Aggregation

When an `ExecutionIntent` contains multiple `FillRecord` entries (partial fills from the venue), they are aggregated into a single leg:

- **Quantity**: sum of all fill quantities
- **Fee**: sum of all fill fees
- **Cost Basis**: sum of all fill cost bases
- **Price**: weighted average (`total_cost_basis / total_quantity`)
- **Fee Asset**: from first fill (uniform within an intent)
- **Simulated**: true if any fill is simulated
- **Timestamp**: from first fill (earliest)

This aggregation preserves the economic total while reducing the pairing surface to one leg per intent.

---

## 3. Pairing Rules

### 3.1 Matching Invariants

Seven invariants govern all pairing decisions:

**M1 — Same Symbol**: `entry.symbol == exit.symbol`. A BTCUSDT entry never pairs with an ETHUSDT exit.

**M2 — Same Source/Segment**: `entry.source == exit.source`. A `binance_spot` entry never pairs with a `binance_futures` exit. This prevents cross-segment contamination.

**M3 — Opposite Side**: Entry and exit must have opposite sides. For longs: buy entry ↔ sell exit. For shorts: sell entry ↔ buy exit. Two buy fills never pair.

**M4 — Temporal Ordering**: `entry.timestamp <= exit.timestamp`. An exit cannot precede its entry. This prevents backwards pairing from data ordering artifacts.

**M5 — FIFO Priority**: When multiple exits are eligible for an entry, the earliest one is selected. When multiple entries compete for an exit, the earliest entry has priority. This produces deterministic, temporally-ordered pairing.

**M6 — One-to-One**: Each unit of quantity participates in at most one round-trip. No double-counting. If an entry of 0.2 BTC is partially matched with an exit of 0.1 BTC, the remaining 0.1 BTC is a separate unmatched fragment.

**M7 — Deterministic**: Given the same input legs, the algorithm always produces the same output round-trips. No randomness, no external state dependency.

### 3.2 What the Rules Do NOT Guarantee

| Not Guaranteed | Why |
|---------------|-----|
| Causal pairing | Two fills may pair by rules M1-M4 without sharing a CorrelationID. The matching is structural (same symbol, opposite side, temporal order), not causal. |
| Optimal pairing | FIFO may not minimize total unmatched quantity in all cases. FIFO optimizes for temporal consistency, not global quantity matching. |
| Cross-session pairing | An entry in session A will not pair with an exit in session B unless both are present in the same matching input. |
| Price improvement | The algorithm does not consider whether the exit price is "better" than the entry price. All eligible exits are candidates. |

---

## 4. Open/Closed/Unresolved Semantics

### 4.1 State Definitions

**Closed (Paired)**
- Both entry and exit legs exist with matched quantities
- P&L is realized: `gross_pnl = exit_cost - entry_cost` (long) or `entry_cost - exit_cost` (short)
- Net P&L includes fees: `net_pnl = gross_pnl - (entry_fees + exit_fees)`
- Outcome is classifiable: win (net > 0), loss (net < 0), breakeven (|net| <= 0.0001)

**Open (Unmatched Entry)**
- Entry leg exists, no exit found
- P&L is unrealized — the entry cost is known, the exit value is not
- The position is semantically "open" — capital is deployed but not recovered
- Outcome is always `unresolved`
- Reason code indicates why: `no_exit_found`, `session_boundary`, `rejected_leg`, `cancelled_leg`

**Partially Resolved**
- Entry quantity > exit quantity (or vice versa)
- The matched portion is **closed** with realized P&L
- The unmatched remainder is **open** with unrealized status
- Produces two RoundTrips: one `paired` + one `unmatched_entry` (or `unmatched_exit`)
- Reason for the remainder: `quantity_mismatch_remainder`

**Orphan Exit (Unmatched Exit)**
- Exit leg exists with no preceding entry
- This is a data quality anomaly, not a normal trading state
- Reason: `no_entry_found`
- Outcome: not classified (no entry context for P&L)

### 4.2 State Transition Diagram

```
ExecutionIntent (filled, buy)
  │
  ▼
Leg (entry, buy, 0.2 BTC)
  │
  ├── Exit found (0.2 BTC sell) ──► RoundTrip (paired, 0.2 BTC) ──► Realized P&L
  │
  ├── Exit found (0.1 BTC sell) ──► RoundTrip (paired, 0.1 BTC) ──► Realized P&L
  │                                  + RoundTrip (unmatched_entry, 0.1 BTC) ──► Unrealized
  │
  └── No exit found ──────────────► RoundTrip (unmatched_entry, 0.2 BTC) ──► Unresolved
```

### 4.3 Outcome Classification by State

| Pairing State | Cost Basis | Outcome | P&L Status |
|--------------|------------|---------|------------|
| paired | non-zero both legs | win / loss / breakeven | Realized |
| paired | zero (either leg) | unresolved | Not classifiable |
| unmatched_entry | any | unresolved | Unrealized |
| unmatched_exit | any | not classified | No entry context |

---

## 5. Unresolved Case Taxonomy

### 5.1 Why Unresolved Matters

Before this model, `unresolved` was a single opaque bucket. Any fill without a counterpart was `unresolved` with no further information. This made it impossible to distinguish:
- A trade that is genuinely open (waiting for exit signal)
- A trade whose exit was rejected
- A trade that crossed a session boundary
- A data quality gap

The reason-code system introduced here decomposes `unresolved` into actionable categories.

### 5.2 Reason Codes

| Code | Meaning | Typical Cause | Actionability |
|------|---------|---------------|---------------|
| `no_exit_found` | No eligible exit leg exists | Strategy hasn't produced a closing signal yet | Normal in active trading; monitor duration |
| `no_entry_found` | Exit without preceding entry | Data gap or out-of-order ingestion | Investigate data pipeline |
| `quantity_mismatch_remainder` | Partial match leftover | Entry qty ≠ exit qty | Normal with partial fills; track residuals |
| `session_boundary` | Session ended before exit | Session close/halt while position open | Review session close procedures |
| `rejected_leg` | Counterpart was rejected | Venue rejected the closing order | Investigate rejection reason |
| `cancelled_leg` | Counterpart was cancelled | Closing order cancelled before fill | Investigate cancellation trigger |

### 5.3 Unresolved Duration

The model does not track how long a leg has been unresolved. Duration-based analysis (e.g. "entries open for more than N minutes") is a read-path computation on the timestamp field, not a pairing concern. This is documented as out of scope for S480.

---

## 6. Realized vs Unrealized Result

### 6.1 Realized Result

A result is **realized** when:
1. The round-trip is `paired` (both legs present)
2. Both legs have non-zero cost basis
3. P&L is computed via `ClassifyPair()`:
   - Long: `gross = exit_cost - entry_cost`
   - Short: `gross = entry_cost - exit_cost`
   - Net: `net = gross - total_fees`

A realized result is final and immutable — it represents an actual economic outcome.

### 6.2 Unrealized (Open) Result

A result is **unrealized** when:
1. The entry leg exists but no exit has been matched
2. The entry cost basis is known
3. The exit value is unknown
4. No P&L can be computed

An unrealized result may eventually become realized if a matching exit appears in a future matching run (within the same pairing scope).

### 6.3 Not Classifiable

A result is **not classifiable** when:
1. Cost basis is zero (paper/dry-run with zero pricing — gap G1)
2. An exit exists without an entry (orphan — data anomaly)
3. Both legs are rejected (excluded from pairing entirely)

---

## 7. Edge Cases and Ambiguity

### 7.1 Multiple Entries, Single Exit

```
Entry A: buy 0.1 BTC @ 50000 (t=1)
Entry B: buy 0.1 BTC @ 50500 (t=2)
Exit C:  sell 0.1 BTC @ 51000 (t=3)
```

FIFO pairs A with C. B remains unmatched. This is deterministic but may not reflect trader intent (if the trader intended to close B specifically). The model acknowledges this limitation.

### 7.2 Simultaneous Fills

When two legs have the same timestamp, FIFO ordering falls back to input order (stable sort). The model does not attempt to resolve true simultaneity — it relies on the ordering of the input data.

### 7.3 Cancelled with Partial Fill

```
Intent: buy 0.2 BTC, status=cancelled, fills=[{qty=0.05}]
```

The 0.05 BTC partial fill is eligible for pairing. The unfilled 0.15 BTC is not represented as a leg. The intent's cancelled status does not invalidate the fills that did occur.

### 7.4 Same CorrelationID, Different Symbols

If the pipeline ever produces two execution intents with the same CorrelationID but different symbols, they will not pair (M1 prevents it). Each symbol is paired independently.

### 7.5 Repeated Entries Without Exits (Pyramiding)

```
Entry A: buy 0.1 BTC (t=1)
Entry B: buy 0.1 BTC (t=2)
Entry C: buy 0.1 BTC (t=3)
No exits
```

All three become `unmatched_entry` with reason `no_exit_found`. The model does not aggregate them into a "position" — each is an independent open fragment. Position tracking is out of scope (NG-RT2).

---

## 8. Limitations and Residual Ambiguity

### 8.1 Structural Limitations

| Limitation | Impact | Mitigation |
|-----------|--------|------------|
| FIFO may not reflect trader intent | Pairing is temporal, not intentional | Document as known; future LIFO/specific-lot (NG-RT7) |
| No causal validation in matching | Legs pair structurally, not by CorrelationID chain | Causal validation is available via lineage but not enforced in matching |
| Float64 precision for quantities | Extremely small remainders may accumulate | Epsilon threshold (1e-12) handles most cases |
| Strategy direction is required input | Without it, short strategies are misclassified | Default to long convention; document requirement |

### 8.2 Residual Ambiguity

| Ambiguity | Status | Resolution Path |
|-----------|--------|-----------------|
| When should `session_boundary` reason be assigned vs `no_exit_found`? | Deferred to S481 | Read model knows session scope; matching does not |
| Should pairing consider CorrelationID as a matching rule? | Decided: NO for S480 | Structural matching is broader and handles more cases; causal filtering is a future enhancement |
| How to handle futures funding rates in P&L? | Out of scope (NG-RT15) | Document as known gap |
| Cross-session pairing demand | Frozen (NG-RT6) | Future wave if validated by operator need |

### 8.3 What This Model Does NOT Define

- **Position size or exposure.** Unmatched entries are not aggregated into positions.
- **Risk metrics.** No drawdown, no Sharpe, no exposure limits.
- **Real-time state.** Pairing is a read-path computation on historical data.
- **Portfolio P&L.** No cross-symbol aggregation.
- **Slippage.** No comparison of fill price vs expected price.

---

## 9. References

- [Canonical Round-Trip Model](canonical-round-trip-and-leg-pairing-model.md)
- [Wave Charter and Scope Freeze](round-trip-pairing-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](round-trip-pairing-capabilities-questions-and-non-goals.md)
- [S479 Charter Report](../stages/stage-s479-round-trip-pairing-charter-report.md)
