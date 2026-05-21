# Fee Normalization Model and Cross-Segment Consistency

> S428 | Status: active | Owner: execution domain

## Problem Statement

Prior to S428, the `FillRecord.Fee` field carried different semantics depending on the market segment:

- **Spot**: `Fee` = aggregated `commission` from Binance Spot fills[] array (actual trading fee).
- **Futures**: `Fee` = `cumQuote` from Binance Futures RESULT response (total notional value, NOT a fee).
- **Paper/DryRun**: `Fee` = "0" (no venue interaction).

This semantic divergence made cross-segment queries unreliable. A query like "show me all fills with fee > 1.0" would return Futures fills based on notional value (e.g., $65 for a 0.001 BTC fill) alongside Spot fills based on actual commission (e.g., 0.00006 BNB). The fields were structurally identical but semantically incompatible.

## Canonical Fee Model

S428 introduces two new fields to `FillRecord` and corrects the semantics of `Fee`:

| Field | Type | Semantics | Spot | Futures | Paper/DryRun |
|-------|------|-----------|------|---------|--------------|
| `Fee` | `string` | Actual trading commission charged by venue | Aggregated commission from fills[] | `"0"` (not available from RESULT response) | `"0"` |
| `FeeAsset` | `string` | Denomination of the fee | `commissionAsset` (e.g., "BNB") | `""` (not available) | `""` |
| `CostBasis` | `string` | Total notional value of the fill | `cummulativeQuoteQty` | `cumQuote` | `""` |

### Key Invariants

1. **Fee never carries notional value.** If the venue does not report a commission, Fee = "0".
2. **CostBasis never carries commission.** It represents price * quantity (or the venue's cumulative quote).
3. **FeeAsset is only populated when the venue provides it.** Futures RESULT response does not include commission details.
4. **New fields use `omitempty` in JSON** to avoid bloating Paper/DryRun payloads.

## Binance API Response Mapping

### Spot (/api/v3/order with FULL response type)

```json
{
  "executedQty": "0.001",
  "cummulativeQuoteQty": "65.43",
  "fills": [
    {"price": "65430.00", "qty": "0.001", "commission": "0.00006543", "commissionAsset": "BNB"}
  ]
}
```

Mapping:
- `Fee` = sum of all fills[].commission = "0.00006543"
- `FeeAsset` = fills[0].commissionAsset = "BNB"
- `CostBasis` = cummulativeQuoteQty = "65.43"
- `Price` = weighted average of fills[].price

### Futures (/fapi/v1/order with RESULT response type)

```json
{
  "avgPrice": "65432.10",
  "executedQty": "0.001",
  "cumQuote": "65.43210"
}
```

Mapping:
- `Fee` = "0" (commission not in RESULT response; requires separate `/fapi/v1/userTrades` call)
- `FeeAsset` = "" (not available)
- `CostBasis` = cumQuote = "65.43210"
- `Price` = avgPrice = "65432.10"

## Impact on Existing Surfaces

### Write Path (ClickHouse)
Fills are serialized as JSON into the `fills` column. The new fields (`fee_asset`, `cost_basis`) appear in the JSON payload and are queryable via ClickHouse JSON functions. No DDL changes required.

### Read Path (KV + ClickHouse)
`ParseFillsJSON` deserializes the new fields automatically (Go struct tags). Existing reads that only access `Fee` continue to work correctly.

### NATS Events
`VenueOrderFilledEvent` carries `ExecutionIntent` with fills. The new fields are included in the JSON payload via standard struct serialization.

## Backwards Compatibility

- **JSON deserialization**: Old fills without `fee_asset`/`cost_basis` deserialize with zero values ("", ""). This is the correct representation for Paper/DryRun fills.
- **Existing Spot fills in ClickHouse**: `Fee` value is unchanged (was already the real commission). `FeeAsset` and `CostBasis` will be empty for historical records.
- **Existing Futures fills in ClickHouse**: `Fee` contains the old `cumQuote` value. These are identifiable by `fee_asset=""` and `cost_basis=""`. A migration is not required but historical queries should account for this.

## Limitations

1. **Futures commission is structurally unavailable** from the RESULT response type. Obtaining real Futures commission requires a separate `/fapi/v1/userTrades` API call per order, which is not in scope for S428.
2. **FeeAsset uniformity assumption**: The model takes `commissionAsset` from the first fill leg. Binance uses a uniform commission asset within a single market order, but this is a venue-specific assumption.
3. **No historical backfill**: Existing Futures fills in ClickHouse retain the old `cumQuote`-in-Fee pattern. S428 normalizes going forward only.
