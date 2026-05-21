# Round-Trip Read Surfaces: Realized vs Unresolved Attribution and Limitations

**Stage**: S481
**Wave**: Round-Trip Pairing (S479--S483)
**Date**: 2026-03-26

---

## 1. Purpose

This document defines the semantics of realized vs unresolved outcomes in the round-trip pairing read model, how they integrate with attribution surfaces, and the explicit limitations of the current approach.

---

## 2. Outcome Classification Semantics

### 2.1 Realized Outcomes (Paired Round-Trips)

A round-trip with `state=paired` has both entry and exit legs matched by FIFO rules. The effectiveness system computes realized P&L:

| Outcome | Condition | P&L computation |
|---------|-----------|-----------------|
| `win` | `net_pnl > breakeven_threshold (0.0001)` | `gross = exit_cost - entry_cost` (long) or `entry_cost - exit_cost` (short); `net = gross - total_fees` |
| `loss` | `net_pnl < -breakeven_threshold` | Same computation, result is negative |
| `breakeven` | `|net_pnl| <= breakeven_threshold` | Within tolerance band |

### 2.2 Unresolved Outcomes

| State | Reason Code | Meaning | Attribution |
|-------|-------------|---------|-------------|
| `unmatched_entry` | `no_exit_found` | Entry fill exists but no eligible exit | `unresolved` — cost basis recorded, P&L indeterminate |
| `unmatched_exit` | `no_entry_found` | Exit fill exists but no preceding entry | `unresolved` — orphan exit, data gap |
| `unmatched_entry` | `quantity_mismatch_remainder` | Partial match, residual entry quantity | `unresolved` — remainder from partial pairing |
| `unmatched_exit` | `quantity_mismatch_remainder` | Partial match, residual exit quantity | `unresolved` — remainder from partial pairing |
| `unmatched_entry` | `session_boundary` | Session closed before exit appeared | `unresolved` — position still open at session end |
| `unmatched_entry` | `rejected_leg` | Counterpart leg rejected by venue | `unresolved` — no trade occurred |
| `unmatched_entry` | `cancelled_leg` | Counterpart leg cancelled before fill | `unresolved` — no trade occurred |

### 2.3 Non-Participating Cases

| Case | Handling |
|------|----------|
| Rejected orders | Excluded from pairing entirely (no legs produced) |
| Orders with no fills | Excluded from pairing (no fill data to form legs) |
| Non-terminal orders | Excluded from pairing (not yet final) |

---

## 3. Read Surfaces

### 3.1 Pairing Surface

| Endpoint | Semantics |
|----------|-----------|
| `GET /analytical/composite/pairing` | Batch: returns all round-trips (paired + unmatched) with optional attribution |
| `GET /analytical/composite/pairing/chain` | Single-chain: shows one chain's leg in the pairing context |

**Response structure** (`PairingReply`):
- `round_trips[]`: Each has `state`, `entry`, `exit`, `unmatched_reason`, `matched_quantity`, and optional `attribution`
- `summary`: Aggregated counts (paired, unmatched, resolved_rate) plus effectiveness breakdown (win/loss/breakeven counts, total_pnl, total_fees)
- `meta`: Diagnostic signals (chains_scanned, legs_produced, round_trips, total_ms)

### 3.2 Effectiveness Surface (Enhanced)

| Endpoint | Change in S481 |
|----------|----------------|
| `GET /analytical/composite/decision/effectiveness/batch` | Now uses ClassifyPair for paired round-trips; single-leg chains remain unresolved |
| `GET /analytical/composite/decision/effectiveness/summary` | Same pairing integration for cohort aggregation |
| `GET /analytical/composite/decision/effectiveness` | Unchanged (single-chain, no pairing possible) |

### 3.3 Observable Impact

Before S481, all batch effectiveness evaluations for filled orders returned `unresolved` (single-leg classification). After S481:

- Chains with a matching entry/exit pair produce `win`, `loss`, or `breakeven`
- Chains without a matching counterpart remain `unresolved`
- The `resolved_rate` metric on the pairing surface quantifies the improvement
- `CohortSummary.UnresolvedCount` decreases proportionally to the number of pairs found

---

## 4. Integration Points

### 4.1 From Pairing to Effectiveness

```
RoundTrip (paired)
    |
    +-- entry.CorrelationID -> chainByCorr[id] -> ExecutionIntent (entry)
    +-- exit.CorrelationID  -> chainByCorr[id] -> ExecutionIntent (exit)
    |
    v
effectiveness.ClassifyPair(entry_intent, exit_intent) -> Attribution
```

### 4.2 From Effectiveness to Pairing

The pairing read model (`GetPairingUseCase`) embeds `Attribution` in each paired `RoundTripView`, providing a unified view of pairing + P&L in a single response.

---

## 5. Limitations

### 5.1 Scope Limitations

| Limitation | Severity | Rationale |
|------------|----------|-----------|
| Pairing is within-query scope only | MEDIUM | Cross-query or cross-session pairing would require state management and position tracking, which is out of scope (G1, G9) |
| FIFO matching only | LOW | LIFO/HIFO are non-goals per S479 charter |
| Single venue/segment | LOW | Cross-exchange matching is a non-goal per S479 charter |
| No real-time pairing | LOW | Pairing is a batch read-path computation, not streaming |

### 5.2 Attribution Limitations

| Limitation | Severity | Rationale |
|------------|----------|-----------|
| `ClassifyPair` uses full intent fills, not scaled partial quantities | MEDIUM | When partial matching splits a fill, the P&L computation uses the original intent's fills rather than the proportionally scaled subset. This is acceptable because cost basis scaling in `MatchFIFO` is proportional. |
| Strategy direction inference may default incorrectly | LOW | When strategy stage is absent, defaults to long. Short-strategy fills without strategy context will have entry/exit directions inverted. |
| Single-chain effectiveness lookup unchanged | INFORMATIONAL | The single-chain endpoint has only one chain — pairing is inherently a multi-chain operation. |

### 5.3 Residual Gaps for S482

| Gap | Target |
|-----|--------|
| Review surface for reconciliation of unresolved cases | S482 |
| Cross-session pairing with explicit boundary handling | Out of wave scope |
| Portfolio-level aggregation across symbols | Non-goal |

---

## 6. Invariants

| ID | Invariant | Verified |
|----|-----------|----------|
| I-1 | Paired round-trips always have both entry and exit legs | Test: `TestGetPairing_Batch_PairedRoundTrip` |
| I-2 | Unmatched legs never have attribution | Test: `TestGetPairing_Batch_UnmatchedEntry` |
| I-3 | Rejected orders produce zero legs | Test: `TestGetPairing_Batch_RejectedExcluded` |
| I-4 | Effectiveness batch now produces win/loss for paired chains | Test: `TestGetEffectiveness_Batch_PairedRoundTripProducesWin` |
| I-5 | Unpaired single-leg fills remain unresolved in effectiveness | Test: `TestGetEffectiveness_Batch_SingleLegRemainsUnresolved` |
| I-6 | Summary cohort reflects reduced unresolved count | Test: `TestGetEffectivenessSummary_PairingIntegration_ReducesUnresolved` |
| I-7 | All existing S476/S477 tests pass without modification | Zero regressions |
