# Fees, Costs, and Commission: Persistence, Reconciliation, and Limitations

S499: Canonical reference for how fees, costs, and commissions flow through the system after the S428 normalization and S499 hardening.

## Canonical Fee Model

Each `FillRecord` carries:

| Field | Type | Semantics |
|-------|------|-----------|
| `Fee` | string (decimal) | Actual trading commission charged by venue |
| `FeeAsset` | string | Denomination of the fee (e.g. "BNB", "USDT") |
| `CostBasis` | string (decimal) | Total notional value of fill (price * quantity or cumQuote) |
| `FeeSource` | FeeSource enum | Provenance of fee data — why Fee has its current value |

### FeeSource Values

| Value | When | Fee | FeeAsset | CostBasis |
|-------|------|-----|----------|-----------|
| `venue` | Spot fills with real commission | Real commission | Real asset | Real cumQuoteQty |
| `unavailable` | Futures RESULT response | "0" | "" | Real cumQuote |
| `simulated` | Paper/DryRun fills | "0" | "" | "0" or resolved price |
| `fallback` | Spot without fills[] array (unexpected) | "0" | "" | Real cumQuoteQty |

## Write Path

### Venue Adapters → NATS Events

1. **Spot** (`binance_spot_testnet_adapter.go`):
   - Calls `computeSpotFillAggregates()` on `resp.Fills[]`
   - Produces weighted avg price, summed commission, fee asset
   - Sets `FeeSource = venue`
   - Validates FeeAsset uniformity (returns `mixed` flag)

2. **Futures** (`binance_futures_testnet_adapter.go`):
   - RESULT response has no commission fields
   - `Fee = "0"`, `FeeAsset = ""`
   - `CostBasis = cumQuote` (real notional)
   - Sets `FeeSource = unavailable`

3. **Paper** (`paper_venue_adapter.go`, `paper_fill_simulator.go`):
   - `Fee = "0"`, `FeeAsset = ""`
   - `CostBasis` = "0" or resolved price-based
   - Sets `FeeSource = simulated`

4. **DryRun** (`dry_run_submitter.go`):
   - Same as Paper, sets `FeeSource = simulated`

### NATS → ClickHouse Writer

Events carry `ExecutionIntent.Fills[]` through:
- `paper_order` consumer → `executions` table
- `venue_market_order` consumer → `executions` table
- `venue_rejection` consumer → `executions` table

Fills are serialized as JSON in the `fills` column. FeeSource is persisted as part of the JSON structure.

## Read Path

### ClickHouse → Execution Reader

`ParseFillsJSON()` deserializes the JSON blob back to `[]FillRecord`, preserving all fields including FeeSource.

### Pairing Pipeline

`IntentToLeg()` aggregates multi-fill records into a single `Leg`:
- `Fee = Σ fill.Fee`
- `CostBasis = Σ fill.CostBasis`
- `FeeAsset` from first fill
- `FeeSource` from first fill
- Weighted average price

`scaleLeg()` scales Fee and CostBasis proportionally during partial matches.

### Effectiveness Pipeline

`ClassifyPair()` computes:
- `TotalFees = entryFees + exitFees`
- `GrossPnL = exitCost - entryCost` (long) or `entryCost - exitCost` (short)
- `NetPnL = GrossPnL - TotalFees`
- `EntryCostBasis` and `ExitCostBasis` for verification

### Reconciliation Pipeline

`ReconcileRoundTrip()` produces:
- `FeeReliable`: true when both legs have fee > 0 OR FeeSource = "unavailable"
- `PnLReliable`: true when paired, classifiable outcome, non-zero cost basis
- Flags: `fee_gap`, `fee_ratio_anomaly`, `fee_source_fallback`, etc.

### Review Surface

`buildReviewSummary()` aggregates:
- `TotalFees`: sum across all round-trip attributions
- `TotalCostBasis`: sum of entry + exit cost basis
- `FeeCoverageRatio`: "N/M" fills with fee / total fills
- `FeeReliableCount`: round-trips with reliable fee data

### Session Audit Bundle

`NewAuditFeeSummary()` computes per-session:
- `FillsWithFee` / `FillsWithoutFee`
- `FeeCoverageRatio`
- `FeeAssets` set

### Verification

`checkFeeFields()` is FeeSource-aware:
- venue fills: expected non-zero fee → pass if present, warn if not
- unavailable/simulated fills: expected zero fee → pass
- fallback fills: unexpected → warn

## Reconciliation Flags

### Fee-Related Flags

| Flag | Meaning | When |
|------|---------|------|
| `fee_gap` | One/both legs have fee=0 | Always for Futures/Paper; Spot fallback |
| `fee_asset_mismatch` | Different fee assets on entry/exit | Rare; possible with BNB vs USDT commission |
| `fee_ratio_anomaly` | Fee > 10% of cost_basis | Data corruption or misattribution |
| `fee_source_fallback` | Spot without fills[] array | Unexpected API behavior |

### Reliability Assessment

| Scenario | FeeReliable | PnLReliable |
|----------|-------------|-------------|
| Spot pair, both fee > 0 | true | true |
| Futures pair, FeeSource=unavailable | true | true |
| Paper pair | false | false |
| Mixed Spot/Futures | depends on legs | depends on legs |
| Spot with fallback | false | true (if cost_basis present) |

## Limitations

### Structural Limitations (Accepted)

1. **Futures commission unavailable**: Binance Futures RESULT response does not include commission. This requires upgrading to `newOrderRespType=FULL` or fetching commission from `GET /fapi/v1/userTrades` — both are out of scope for S499.

2. **Fee=0 ambiguity for legacy data**: FillRecords created before S499 have `FeeSource=""` (empty). Downstream logic treats these the same as `FeeSource=""` with no special handling. Legacy data can be identified by the absence of FeeSource and should be treated as potentially unreliable.

3. **Single FeeAsset per order**: The system assumes Binance uses a uniform CommissionAsset within a single market order. If Binance changes this behavior, the mixed-asset detection in `computeSpotFillAggregates` will flag it but not handle it specially.

4. **Fee ratio threshold is static**: The 10% anomaly threshold is hardcoded. Exchange-specific fee schedules vary; this threshold may produce false positives for high-fee markets or BNB discount toggle changes.

### Not In Scope

- **Futures fee fetching**: Separate API call to retrieve commission post-trade (potential future stage).
- **Fee accounting/ledger**: No aggregated fee tracking, no fee P&L attribution beyond round-trip scope.
- **Portfolio-level fee analysis**: No cross-symbol or cross-session fee aggregation.
- **Historical backfill**: Existing data without FeeSource is not retroactively tagged.
- **Fee currency conversion**: No normalization of fees to a common denomination.
