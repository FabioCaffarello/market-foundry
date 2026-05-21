# Fills, Fees, Pairing, and Result Reconciliation: Semantics and Limitations

> S482 — canonical reference for how fills, fees, pairing, and effectiveness results reconcile across the measurement layer.

## Data Flow

```
ClickHouse executions table
  └─ FillRecord[] (price, quantity, fee, fee_asset, cost_basis, simulated, timestamp)
      │
      ├─ IntentToLeg() → Leg (aggregated fills → single leg per direction)
      │   └─ MatchFIFO() → RoundTrip[] (entry/exit pairs via FIFO matching)
      │       └─ ClassifyPair() → Attribution (win/loss/breakeven/unresolved)
      │           └─ ReconcileRoundTrip() → ReconciliationResult (data-quality flags)
      │
      └─ Classify() → Attribution (single-leg, always unresolved)
```

## Fill Aggregation Rules

When an ExecutionIntent has multiple fills (partial fills from the venue):

| Field | Aggregation | Source |
|-------|------------|--------|
| Quantity | Sum | All fill quantities |
| Fee | Sum | All fill fees |
| CostBasis | Sum | All fill cost bases |
| Price | Weighted average (cost_basis / quantity) | Derived |
| Timestamp | First fill timestamp | Earliest |
| Simulated | OR across all fills | Any simulated → true |
| FeeAsset | From first fill | First fill's fee_asset |

## Fee Semantics by Segment

| Segment | Fee Source | FeeAsset | Fee Reliability |
|---------|-----------|----------|----------------|
| Spot (real) | Sum of fill commissions | commissionAsset (e.g., "BNB", "USDT") | Reliable |
| Futures (real) | "0" — unavailable from venue API | "" (empty) | **Unreliable** — `fee_gap` flag raised |
| Paper/Dry-run | "0" | "" (empty) | N/A — `simulated` flag raised |

### Impact on Net P&L

- **Spot:** `net_pnl = gross_pnl - (entry_fees + exit_fees)` — accurate.
- **Futures:** `net_pnl = gross_pnl - 0` — overstates return by the actual fee amount.
- **Paper:** `net_pnl` not meaningful — zero cost basis makes P&L unclassifiable.

## Pairing Reconciliation Rules

### Cost Basis Alignment

For a paired round-trip to produce reliable P&L:

1. **Both legs must have non-zero cost basis.** If either is zero (paper/dry-run), the outcome is unresolved and `cost_basis_zero` is flagged.
2. **Cost basis represents notional value** (price × quantity for spot, cumQuote for futures). The pairing algorithm scales cost basis proportionally when partial-matching.

### Fee Consistency

For fees to be meaningful in net P&L:

1. **Both legs should have non-zero fees.** If either has zero fees, `fee_gap` is flagged and `fee_reliable` is false.
2. **Fee assets should match.** If entry.FeeAsset ≠ exit.FeeAsset, `fee_asset_mismatch` is flagged. The system still sums numerically but the result may not be economically meaningful.

### Quantity Reconciliation

When MatchFIFO produces partial matches:

1. **Matched quantity = min(entry_qty, exit_qty).** The remainder produces an additional unmatched leg.
2. **Proportional scaling:** When a leg is split, cost_basis and fee are scaled proportionally: `scaled_value = original_value × (match_qty / total_qty)`.
3. **The split round-trip's P&L reflects only the matched portion.** The remainder's P&L is unresolved until it finds a future match.

## Outcome Classification After Reconciliation

| Pairing State | Cost Basis | Fee Status | Outcome | Flags |
|---------------|-----------|------------|---------|-------|
| paired | non-zero both | non-zero both | win/loss/breakeven | (clean) |
| paired | non-zero both | zero one/both | win/loss/breakeven | fee_gap |
| paired | zero one/both | any | unresolved | cost_basis_zero, outcome_unresolved |
| unmatched_entry | any | any | unresolved | unmatched_open |
| unmatched_exit | any | any | unresolved | orphan_exit |

## Cross-Surface Consistency

The review surface reconciles data from three existing layers:

1. **Pairing (S480/S481):** Structural matching — which legs paired, which didn't, why.
2. **Effectiveness (S476):** Outcome classification — win/loss/breakeven/unresolved with P&L.
3. **Reconciliation (S482):** Data quality — are the pairing and effectiveness results reliable.

### Invariants

- Every paired round-trip with `pnl_reliable=true` has a non-unresolved effectiveness outcome.
- Every round-trip with `clean=true` has zero flags, `fee_reliable=true`, and (if paired) `pnl_reliable=true`.
- Every unmatched round-trip has at least one flag (`unmatched_open` or `orphan_exit`).
- The sum of `clean_count + flagged_count` equals the total number of reviewed round-trips.

## Limitations

1. **Futures fee data is structurally absent** from the Binance Futures API response. The `fee_gap` flag surfaces this but cannot recover the actual fee amount. This is a venue-level limitation documented in S428.

2. **No currency conversion for fee-asset mismatches.** When entry fees are denominated in BNB and exit fees in USDT, the system sums the numeric values. An operator should interpret `fee_asset_mismatch` as a signal that total_fees is approximate.

3. **Proportional scaling uses float64 arithmetic** with epsilon tolerance (1e-12). For very large positions or very small fee amounts, precision limits may cause minor discrepancies.

4. **Cross-session pairing is limited by query window.** An entry from Session A and exit from Session B will only pair if both fall within the `since`/`until` range of the query.

5. **No retrospective correction.** If a round-trip is flagged today but the underlying data is later corrected (e.g., futures fees become available), the flag will clear on the next query — there is no historical correction log.

6. **Reconciliation does not validate venue data.** If the venue returns incorrect prices or quantities, the reconciliation layer will not detect the error — it only checks structural consistency of what the system recorded.
