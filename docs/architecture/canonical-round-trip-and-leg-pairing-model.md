# Canonical Round-Trip and Leg-Pairing Model

**Stage**: S480
**Wave**: Round-Trip Pairing (S479--S483)
**Date**: 2026-03-26

---

## 1. Purpose

This document defines the canonical model for round-trip trades and leg pairing in market-foundry. It establishes unambiguous semantics for what constitutes an entry leg, an exit leg, a paired round-trip, an open/unresolved fragment, and a realized result.

The model exists to close the semantic gap identified in G-SE1: most effectiveness evaluations return `unresolved` because the pipeline processes individual orders, not paired trades. This model provides the vocabulary and rules for transforming single-leg fills into classifiable round-trip outcomes.

---

## 2. Core Entities

### 2.1 Leg

A **Leg** is the smallest unit of a round-trip. It represents one directional fill event within a trade lifecycle.

| Field | Type | Semantics |
|-------|------|-----------|
| `direction` | `entry` \| `exit` | Role of this leg in the round-trip |
| `side` | `buy` \| `sell` | Order side from the execution intent |
| `symbol` | string | Traded instrument (e.g. `BTCUSDT`) |
| `source` | string | Venue/segment (e.g. `binance_spot`, `binance_futures`) |
| `timeframe` | int | Candle interval that produced the signal chain |
| `correlation_id` | string | Chain-wide trace identifier (signal→fill) |
| `price` | decimal string | Fill price (venue precision) |
| `quantity` | decimal string | Filled quantity |
| `fee` | decimal string | Trading commission charged |
| `fee_asset` | string | Fee denomination (e.g. `BNB`, `USDT`) |
| `cost_basis` | decimal string | Total notional value (price × quantity) |
| `simulated` | bool | True for paper/dry-run fills |
| `timestamp` | time | When the fill occurred |

**Direction inference rules:**

| Strategy Direction | Buy Side | Sell Side |
|-------------------|----------|-----------|
| Long (default) | Entry | Exit |
| Short | Exit | Entry |

A leg is derived from an `ExecutionIntent` by aggregating its fills. When an intent has multiple fills (partial fills), they are aggregated into a single leg with summed quantity, summed fees, summed cost basis, and weighted average price.

### 2.2 RoundTrip

A **RoundTrip** represents the pairing attempt between an entry and an exit leg.

| Field | Type | Semantics |
|-------|------|-----------|
| `entry` | Leg (nullable) | Opening leg. Nil only for `unmatched_exit` state. |
| `exit` | Leg (nullable) | Closing leg. Nil only for `unmatched_entry` state. |
| `state` | PairingState | Lifecycle state of the pairing |
| `unmatched_reason` | UnmatchedReason | Why pairing failed (empty when paired) |
| `matched_quantity` | decimal string | Quantity that was actually paired |
| `symbol` | string | Denormalized for query convenience |
| `source` | string | Denormalized for query convenience |

### 2.3 PairingState

| State | Meaning | Entry | Exit | P&L Status |
|-------|---------|-------|------|------------|
| `paired` | Both legs matched | Present | Present | Realized, classifiable |
| `unmatched_entry` | Entry without exit | Present | Nil | Unrealized, open position |
| `unmatched_exit` | Exit without entry | Nil | Present | Data gap or orphan |

### 2.4 UnmatchedReason

| Reason | When Applied |
|--------|-------------|
| `no_exit_found` | Entry exists but no eligible exit leg was found |
| `no_entry_found` | Exit exists but no preceding entry leg was found |
| `quantity_mismatch_remainder` | Partial match consumed part of the quantity; this is the leftover |
| `session_boundary` | Pairing scope ended (session closed) before an exit appeared |
| `rejected_leg` | The counterpart leg was rejected by the venue |
| `cancelled_leg` | The counterpart leg was cancelled before fill |

---

## 3. Semantic Definitions

### 3.1 Open vs Closed vs Partially Resolved

| Term | Formal Definition | Observable Condition |
|------|-------------------|---------------------|
| **Open** | An entry leg exists with no paired exit. State = `unmatched_entry`. | RoundTrip has entry but no exit. P&L is unrealized. |
| **Closed** | Both entry and exit legs are paired. State = `paired`. | RoundTrip has both legs. P&L is realized and classifiable. |
| **Partially Resolved** | Entry quantity exceeds exit quantity (or vice versa). The matched portion is closed; the remainder is open. | One `paired` RoundTrip for the matched quantity + one `unmatched_entry` for the remainder. |

### 3.2 Realized vs Unrealized Result

| Term | Definition | Condition |
|------|-----------|-----------|
| **Realized** | P&L from a closed round-trip where both legs have fills. Computed by `ClassifyPair()`. | State = `paired`, both legs have non-zero cost basis. Outcome is win/loss/breakeven. |
| **Unrealized** | No P&L can be computed because the position is still open. | State = `unmatched_entry`. Outcome is `unresolved`. The entry cost basis is known but exit value is not. |
| **Not Classifiable** | Fills exist but cost basis is zero (paper/dry-run with zero pricing). | `paired` or `unmatched_entry` with cost_basis = "0". Outcome is `unresolved` regardless. |

### 3.3 Unresolved Cases (Exhaustive)

An outcome is `unresolved` when any of these conditions hold:

| Case | Root Cause | Pairing State |
|------|-----------|---------------|
| No exit leg found | Strategy did not produce a closing trade | `unmatched_entry` |
| Session ended before exit | Session closed/halted with open position | `unmatched_entry` + reason `session_boundary` |
| Exit rejected by venue | Venue rejected the closing order | `unmatched_entry` + reason `rejected_leg` |
| Exit cancelled | Closing order cancelled before fill | `unmatched_entry` + reason `cancelled_leg` |
| Zero cost basis | Paper/dry-run fill with no meaningful price | `paired` or `unmatched_entry` but P&L = 0 |
| Orphan exit | Exit fill has no preceding entry (data gap) | `unmatched_exit` |

---

## 4. Matching Rules

### 4.1 Invariants

All matching rules are enforced as invariants. Violation of any rule prevents pairing.

| ID | Rule | Rationale |
|----|------|-----------|
| M1 | Same symbol | A BTCUSDT entry cannot pair with an ETHUSDT exit |
| M2 | Same source/segment | A spot entry cannot pair with a futures exit |
| M3 | Opposite side | Buy entry pairs with sell exit (long); sell entry pairs with buy exit (short) |
| M4 | Temporal ordering | Entry timestamp must precede or equal exit timestamp |
| M5 | FIFO priority | Earliest unmatched entry pairs with earliest eligible exit |
| M6 | One-to-one | Each fill participates in at most one round-trip |
| M7 | Deterministic | Same input data always produces same pairing output |

### 4.2 FIFO Algorithm

```
1. Separate legs into entries[] and exits[], sorted by timestamp ascending.
2. For each entry (oldest first):
   a. Find the earliest exit where:
      - M1: symbol matches
      - M2: source matches
      - M3: side is opposite
      - M4: exit.timestamp >= entry.timestamp
   b. If found:
      - matched_qty = min(entry.remaining_qty, exit.remaining_qty)
      - Emit a `paired` RoundTrip with matched_qty
      - Subtract matched_qty from both legs' remaining quantities
      - If entry has remaining quantity, continue to next eligible exit
   c. If no eligible exit found, mark entry as `unmatched_entry`
3. Any exits with remaining quantity become `unmatched_exit`
```

### 4.3 Partial-Fill Handling

When entry and exit quantities differ:

1. The matched quantity is `min(entry_qty, exit_qty)`.
2. Cost basis and fees are scaled proportionally: `scaled_cost = original_cost × (matched_qty / total_qty)`.
3. The remainder produces a separate `unmatched_entry` or `unmatched_exit` with reason `quantity_mismatch_remainder`.
4. P&L is computed only on the matched portion.

Example:
```
Entry: buy 0.2 BTC @ 50000 (cost = 10000)
Exit:  sell 0.1 BTC @ 51000 (cost = 5100)

Result:
  Paired RoundTrip: 0.1 BTC, entry_cost=5000, exit_cost=5100, gross_pnl=100
  Unmatched Entry: 0.1 BTC, cost=5000, reason=no_exit_found
```

---

## 5. Integration with Existing Domain

### 5.1 Relationship to ExecutionIntent

A `Leg` is derived from an `ExecutionIntent` via `IntentToLeg()`. The conversion:
- Aggregates all `FillRecord` entries into a single leg
- Infers direction from side + strategy direction
- Preserves `CorrelationID` for lineage traceability
- Uses fill data (not intent requested data) for price/quantity/fees

### 5.2 Relationship to Effectiveness

Paired round-trips feed into `ClassifyPair(entry, exit)` which already exists in `internal/domain/effectiveness/`. The pairing model provides the inputs; effectiveness provides the classification.

| Pairing State | Effectiveness Outcome |
|--------------|----------------------|
| `paired` (non-zero cost) | `win` / `loss` / `breakeven` |
| `paired` (zero cost) | `unresolved` |
| `unmatched_entry` | `unresolved` |
| `unmatched_exit` | Not classified (no entry context) |

### 5.3 Relationship to Lineage

Each leg carries `CorrelationID` from its originating decision chain. This allows:
- Tracing a round-trip back to the signal that initiated it
- Joining pairing results with `DecisionReviewBundle`
- Maintaining full causal chain visibility through the pairing layer

### 5.4 Relationship to Sessions

Pairing is session-aware but not session-bounded:
- Legs from the same session naturally share temporal scope
- Cross-session pairing is out of scope for this wave (NG-RT6)
- Session boundaries can produce `unmatched_entry` with reason `session_boundary`

---

## 6. Domain Location

| Artifact | Path |
|----------|------|
| Domain types | `internal/domain/pairing/pairing.go` |
| Tests | `internal/domain/pairing/pairing_test.go` |

---

## 7. Limitations

1. **FIFO only.** No LIFO, HIFO, or specific-lot matching. FIFO aligns with the current temporal execution model. Alternative strategies are a future extension (NG-RT7).

2. **Single-venue.** No cross-exchange pairing. Source field must match exactly (NG-RT5).

3. **No cross-session pairing.** Entries from session A cannot pair with exits from session B beyond what CorrelationID scope provides (NG-RT6).

4. **Futures fee gap.** Futures fills have `Fee="0"` due to API limitation (G-SE3). Round-trip P&L for futures understates fee impact.

5. **Paper/dry-run zero pricing.** When `CostBasis="0"`, P&L is always zero and outcome is `unresolved` regardless of pairing state. This is a data quality gap (G1), not a pairing gap.

6. **No real-time pairing.** Pairing is computed on read from settled data. No streaming pair events (NG-RT4).

7. **Strategy direction required.** Correct entry/exit inference requires knowing whether the strategy is long or short. When unknown, the default convention is long (buy=entry, sell=exit).

8. **Quantity precision.** Matching uses float64 arithmetic with epsilon tolerance (1e-12). For extremely large quantities or extremely small remainders, floating-point precision limits apply.

---

## 8. References

- [Wave Charter and Scope Freeze](round-trip-pairing-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](round-trip-pairing-capabilities-questions-and-non-goals.md)
- [Effectiveness Domain](../../internal/domain/effectiveness/effectiveness.go) — `ClassifyPair()`
- [Execution Domain](../../internal/domain/execution/execution.go) — `ExecutionIntent`, `FillRecord`
- [Lineage Domain](../../internal/domain/lineage/lineage.go) — causal chain model
