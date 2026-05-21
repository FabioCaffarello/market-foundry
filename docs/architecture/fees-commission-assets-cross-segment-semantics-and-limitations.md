# Fees, Commission, Assets: Cross-Segment Semantics and Limitations

> S428 | Status: active | Owner: execution domain

## Cross-Segment Fee Field Inventory

### Binance Spot

| Venue Field | Location | Domain Mapping | Notes |
|-------------|----------|----------------|-------|
| `fills[].commission` | Order response (FULL) | `FillRecord.Fee` (aggregated) | Per-leg commission, summed across all fill legs |
| `fills[].commissionAsset` | Order response (FULL) | `FillRecord.FeeAsset` | Denomination (e.g., "BNB", "USDT"). Uniform per order |
| `cummulativeQuoteQty` | Order response | `FillRecord.CostBasis` | Total notional: sum(price * qty) across legs |
| `fills[].price` | Order response (FULL) | `FillRecord.Price` (weighted avg) | Weighted average computed from all legs |
| `fills[].qty` | Order response (FULL) | `FillRecord.Quantity` (via executedQty) | Total filled from `executedQty` |

### Binance Futures

| Venue Field | Location | Domain Mapping | Notes |
|-------------|----------|----------------|-------|
| *(not available)* | RESULT response type | `FillRecord.Fee` = "0" | Commission requires separate `/fapi/v1/userTrades` call |
| *(not available)* | RESULT response type | `FillRecord.FeeAsset` = "" | Not in RESULT response |
| `cumQuote` | Order response | `FillRecord.CostBasis` | Total notional value |
| `avgPrice` | Order response | `FillRecord.Price` | Volume-weighted average from venue |
| `executedQty` | Order response | `FillRecord.Quantity` | Total filled base quantity |

### Paper / DryRun

| Field | Value | Notes |
|-------|-------|-------|
| `Fee` | "0" | No venue interaction |
| `FeeAsset` | "" | No commission |
| `CostBasis` | "" | No notional computation (price is best-effort from PriceSource) |
| `Simulated` | true | Always true for paper/dry-run |

## Semantic Differences Between Segments

### Fee: Commission vs Notional

**Before S428**: The `Fee` field was overloaded:
- Spot: actual commission (small value, e.g., "0.00006543")
- Futures: cumQuote / notional value (large value, e.g., "65.43210")

**After S428**: `Fee` = commission only. `CostBasis` = notional value. Cross-segment comparison is now valid for both fields independently.

### Commission Asset Availability

Spot provides `commissionAsset` per fill leg. Futures RESULT response does not include any commission information. This is a structural limitation of the Binance Futures API response type used (RESULT vs FULL — Futures does not support FULL for market orders).

### Price Derivation

- **Spot**: Weighted average computed from `fills[].price` and `fills[].qty`.
- **Futures**: `avgPrice` provided directly by the venue.
- Both produce a single price string in `FillRecord.Price`.

## Query Implications

### Valid Cross-Segment Queries (after S428)

```sql
-- Total commission paid across all segments
SELECT source, SUM(JSONExtractFloat(fill, 'fee')) as total_fee
FROM executions
WHERE status = 'filled' AND JSONExtractFloat(fill, 'fee') > 0

-- Total notional value traded
SELECT source, SUM(JSONExtractFloat(fill, 'cost_basis')) as total_notional
FROM executions
WHERE status = 'filled' AND JSONExtractString(fill, 'cost_basis') != ''
```

### Invalid / Misleading Queries

```sql
-- WRONG: comparing Fee across segments for historical data (pre-S428)
-- Old Futures fills have cumQuote in Fee, old Spot fills have commission in Fee
SELECT * FROM executions WHERE JSONExtractFloat(fill, 'fee') > 1.0
-- Fix: filter by fee_asset presence to distinguish real commission from legacy data
```

## Limitations and Trade-offs

1. **Futures commission gap**: Real commission data for Futures would require an additional API call to `/fapi/v1/userTrades` per order. This adds latency, a second HTTP request, and rate-limit pressure. Not justified for the current use case. If needed in the future, it should be a separate enrichment step, not part of the submit path.

2. **No accounting precision**: Fee values are stored as decimal strings with venue-provided precision (typically 8 decimal places for Binance). No arbitrary-precision arithmetic is applied. This is sufficient for operational observability but not for accounting/ledger reconciliation.

3. **Historical data asymmetry**: Pre-S428 Futures fills have cumQuote stored in the Fee field. These records are distinguishable by having empty `fee_asset` and `cost_basis` fields. A backfill migration could normalize them but is not in scope.

4. **Single-venue assumption**: The model is designed for Binance Spot and Futures. Other venues may have different commission structures (e.g., maker/taker split, rebates). The model is extensible but not pre-generalized.

5. **FeeAsset uniformity**: The model captures one `FeeAsset` per fill (from the first leg). If a venue ever splits commission across different assets within a single order, this would need revision. Binance does not do this for market orders.
