# Live Order Persistence, Read-Path, Fees, and Post-Session Findings

> Authority: S447 | Date: 2026-03-24 | Wave: Live Trading Enablement Ceremony (S444-S448)

## Purpose

This document records the factual findings from inspecting the persistence, read-path, fee handling, and backup infrastructure as part of the S447 post-session operational verification. Every statement below is grounded in code-level evidence, not assumption.

## 1. Persistence Architecture for Live Orders

### Write Path (ClickHouse)

When a live order fills on Binance Spot, the data flows through:

```
BinanceSpotMainnetAdapter.SubmitOrder()
  -> VenueOrderReceipt { Intent (with Fills[]), VenueOrderID }
    -> VenueAdapterActor.publishFill()
      -> NATS stream: EXECUTION_FILL_EVENTS
        -> Writer binary: VenueFillStarter consumer
          -> ClickHouse INSERT into executions table
```

**Source evidence:**
- Adapter response parsing: `binance_spot_testnet_adapter.go:236-302`
- Fill event publishing: `venue_adapter_actor.go:335-360`
- ClickHouse row mapping: `writerpipeline/support.go:364-393` (mapVenueFillRow)

### Write Path (NATS KV)

Independently, the fill event is consumed by the execute binary's KV projection:

```
VenueOrderFilledEvent on NATS stream
  -> Execute binary KV consumer
    -> EXECUTION_VENUE_MARKET_ORDER_LATEST bucket
      -> Key: <source>.<symbol>.<timeframe>.<type>
      -> Value: ExecutionIntent JSON (with Fills[], including fee data)
```

**Source evidence:**
- KV store implementation: `natsexecution/kv_store.go:61-94`
- Registry wiring: `natsexecution/registry.go`

### Rejection Path

If the order is rejected (by venue or by safety gates):

```
VenueAdapterActor.publishRejection()
  -> NATS stream: EXECUTION_REJECTION_EVENTS
    -> Writer binary: VenueRejectionStarter consumer -> ClickHouse
    -> Execute binary: RejectionProjectionActor -> NATS KV
```

Rejection metadata (code, reason, venue HTTP details) is embedded into `intent.Metadata` before KV storage (`rejection_projection_actor.go:100-114`).

## 2. ClickHouse Schema for Execution Records

Table: `executions` (MergeTree, partitioned by `toYYYYMM(timestamp)`)

| Column | Type | Content for Live Fill |
|--------|------|----------------------|
| `event_id` | String | Unique fill event ID |
| `occurred_at` | DateTime64(3) | Event creation time |
| `correlation_id` | String | Trace chain from ingest to fill |
| `causation_id` | String | Parent event ID |
| `type` | LowCardinality(String) | `venue_market_order` |
| `source` | LowCardinality(String) | `binance-spot-mainnet` |
| `symbol` | LowCardinality(String) | `BTCUSDT` |
| `timeframe` | UInt32 | Pipeline timeframe (minutes) |
| `side` | LowCardinality(String) | `buy` or `sell` |
| `quantity` | Float64 | Requested quantity |
| `filled_quantity` | Float64 | Actual filled quantity |
| `status` | LowCardinality(String) | `filled` or `rejected` |
| `risk` | String | Risk assessment JSON |
| `fills` | String | **JSON array of FillRecord** (see below) |
| `parameters` | String | Order parameters JSON |
| `metadata` | String | Enriched metadata JSON |
| `exec_correlation_id` | String | Execution-level correlation |
| `exec_causation_id` | String | Execution-level causation |
| `final` | Bool | `true` for terminal states |
| `timestamp` | DateTime64(3) | Canonical intent timestamp |
| `ingested_at` | DateTime64(3) | ClickHouse ingestion time |

### Fills JSON Structure (Per FillRecord)

```json
[
  {
    "Price": "87654.32",
    "Quantity": "0.00001",
    "Fee": "0.00000876",
    "FeeAsset": "BNB",
    "CostBasis": "0.87654320",
    "Simulated": false,
    "Timestamp": "2026-03-24T14:30:00Z"
  }
]
```

**Fields present for live Spot fills:**

| Field | Source | Present | Notes |
|-------|--------|---------|-------|
| Price | Weighted average from `computeSpotFillAggregates` | YES | Aggregated from per-leg fills |
| Quantity | `resp.ExecutedQty` | YES | Total filled |
| Fee | `SUM(fills[].Commission)` | YES | Aggregated commission |
| FeeAsset | `fills[0].CommissionAsset` | YES | Denomination (BNB, USDT, etc.) |
| CostBasis | `resp.CummulativeQuoteQty` | YES | Total notional value |
| Simulated | Hardcoded `false` for real adapters | YES | Distinguishes real from paper |
| Timestamp | `resp.TransactTime` converted | YES | Venue fill timestamp |

## 3. Fee/Commission Analysis for Spot Live Path

### What Binance Spot Returns

Binance Spot `POST /api/v3/order` with `newOrderRespType=FULL` returns:

```json
{
  "orderId": 123456,
  "executedQty": "0.00001",
  "cummulativeQuoteQty": "0.87654320",
  "fills": [
    {
      "price": "87654.32000000",
      "qty": "0.00001000",
      "commission": "0.00000001",
      "commissionAsset": "BNB"
    }
  ]
}
```

### How the Adapter Processes It

1. `parseOrderResponse()` extracts the response struct (`binance_spot_testnet_adapter.go:236-302`)
2. `computeSpotFillAggregates()` computes weighted average price and sums commission (`binance_spot_testnet_adapter.go:309-331`)
3. A single `FillRecord` is created with aggregated values
4. The record is attached to the `ExecutionIntent.Fills[]` array

### Fee Field Correctness Assessment

| Aspect | Status | Evidence |
|--------|--------|----------|
| Commission captured from exchange | YES | `fills[].Commission` parsed from each leg |
| Commission aggregated correctly | YES | `computeSpotFillAggregates` sums all legs |
| FeeAsset captured | YES | Taken from first fill leg (uniform per order) |
| CostBasis (notional) captured | YES | `cummulativeQuoteQty` from exchange |
| Fee = "0" for minimum orders | POSSIBLE | Exchange may waive commission for BNB holders |
| Simulated flag correct | YES | `false` for real adapters |

### Known Fee Limitations

| # | Limitation | Impact | Severity |
|---|-----------|--------|----------|
| 1 | Per-leg fill detail is aggregated into single FillRecord | Individual leg prices lost | LOW -- minimum orders typically have 1 fill leg |
| 2 | FeeAsset taken from first leg only | Incorrect if legs have different fee assets | NEGLIGIBLE -- Binance uses uniform fee asset per order |
| 3 | Fee stored as string in JSON column | Not directly aggregatable in SQL | LOW -- parseable with `JSONExtractString` |
| 4 | No post-fill fee query to exchange | Commission relies on order response only | LOW -- FULL response type provides complete data for Spot |
| 5 | BNB discount not explicitly tracked | System records actual fee, not the discount flag | INFORMATIONAL -- fee amount is correct regardless |

## 4. Read-Path Coverage

### NATS KV Query Routes

| Route | KV Bucket | Returns | Fee Data |
|-------|-----------|---------|----------|
| VenueMarketOrderLatest | `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Latest fill intent with Fills[] | YES |
| VenueRejectionLatest | `EXECUTION_VENUE_REJECTION_LATEST` | Latest rejection with audit detail | N/A (rejected) |
| StatusLatest | All three KV buckets | Composite: Intent + Result + Rejection + Gate | YES (if filled) |
| LifecycleList | All three KV buckets | Key enumeration with status | Partial (status only) |

### ClickHouse Direct Queries

```sql
-- Latest fill with fee data
SELECT event_id, symbol, side, status, filled_quantity, fills
FROM executions
WHERE symbol = 'BTCUSDT' AND status = 'filled'
ORDER BY timestamp DESC LIMIT 1;

-- Fee extraction from JSON
SELECT
    event_id,
    JSONExtractString(JSONExtractArrayRaw(fills, 1), 'Fee') AS fee,
    JSONExtractString(JSONExtractArrayRaw(fills, 1), 'FeeAsset') AS fee_asset,
    JSONExtractString(JSONExtractArrayRaw(fills, 1), 'CostBasis') AS cost_basis
FROM executions
WHERE symbol = 'BTCUSDT' AND status = 'filled'
ORDER BY timestamp DESC LIMIT 1;

-- Scope audit
SELECT count(), symbol, source
FROM executions
WHERE type = 'venue_market_order' AND timestamp > now() - INTERVAL 24 HOUR
GROUP BY symbol, source;
```

### Read-Path Consistency

| Store | Latency | Freshness | Authority |
|-------|---------|-----------|-----------|
| NATS KV | Sub-second | Real-time (event-driven) | Latest state only |
| ClickHouse | Seconds (async writer) | Near-real-time | Historical + current |

The KV store is updated synchronously in the event consumer, while ClickHouse writes are asynchronous via the writer binary. A brief window of inconsistency (seconds) is expected and acceptable.

## 5. Backup Verification

### Pre-Session Backup (PS-2)

| Item | Detail |
|------|--------|
| Script | `clickhouse-scheduled-backup.sh` |
| Naming | `pre_session_live_<timestamp>` |
| Tables | All MergeTree tables in `market_foundry` (auto-discovered) |
| Off-host | rsync to `BACKUP_OFFHOST_TARGET` if configured |
| Retention | Last 7 backups locally |

### Post-Session Backup (PO-2)

| Item | Detail |
|------|--------|
| Script | Same as pre-session |
| Naming | `post_session_live_<timestamp>` |
| Purpose | Captures state including live order data |
| Delta | Difference between pre and post contains exactly the session's execution records |

### Restore Capability

| Item | Detail |
|------|--------|
| Script | `clickhouse-restore.sh` |
| Method | `RESTORE TABLE ... FROM Disk('backups', '<backup_name>')` |
| Creates tables | Yes, if dropped |
| Idempotent | No -- restore to existing table may fail; drop first |

### Backup Coverage Assessment

| Table | Backed Up | Contains Session Data |
|-------|-----------|----------------------|
| `executions` | YES | Fill/rejection records |
| `signals` | YES | Signal events from pipeline |
| `decisions` | YES | Decision events |
| `strategies` | YES | Strategy events |
| `risk_assessments` | YES | Risk assessments that triggered intent |

## 6. Lifecycle Consistency Analysis

### Expected Terminal State for Filled Order

| Store | Field | Expected Value |
|-------|-------|----------------|
| ClickHouse `executions` | `status` | `filled` |
| ClickHouse `executions` | `final` | `true` |
| ClickHouse `executions` | `filled_quantity` | > 0 |
| NATS KV (venue market order) | `Status` | `filled` |
| NATS KV (venue market order) | `Final` | `true` |
| NATS KV (venue market order) | `Fills[0].Simulated` | `false` |

### Expected Terminal State for Rejected Order

| Store | Field | Expected Value |
|-------|-------|----------------|
| ClickHouse `executions` | `status` | `rejected` |
| ClickHouse `executions` | `final` | `true` |
| ClickHouse `executions` | `metadata` | Contains `rejection_code`, `rejection_reason` |
| NATS KV (venue rejection) | `Status` | `rejected` |
| NATS KV (venue rejection) | `Metadata` | Contains `rejection_code`, `rejection_reason`, `venue_detail.*` |

### Consistency Invariants

| # | Invariant | Verification |
|---|-----------|-------------|
| 1 | ClickHouse status = NATS KV status | Cross-query comparison |
| 2 | `final = true` in both stores | Both stores set final on terminal events |
| 3 | `filled_quantity` matches in both stores | Same source event drives both writes |
| 4 | Fills[] content identical | Same FillRecord serialized to both stores |
| 5 | Correlation chain traceable | `correlation_id` links intent to fill |

## 7. Scope Containment Findings

### What Was Authorized

| Dimension | Authorized | Enforcement |
|-----------|-----------|-------------|
| Exchange | Binance Spot mainnet | Config: `binance_spot_mainnet` adapter |
| Symbol | BTCUSDT | Pipeline config |
| Order type | Market | Domain model |
| Quantity | Minimum exchange quantity | Config |
| Count | 1 | Operator discipline + kill-switch |
| Segment | Spot only | Config: only `spot` enabled |

### How Scope Is Verified Post-Session

1. **Symbol containment**: Query `executions` for `type = 'venue_market_order'` and check that ALL records have `symbol = 'BTCUSDT'`
2. **Segment containment**: Check that `source = 'binance-spot-mainnet'` (not futures)
3. **Count containment**: Count total `venue_market_order` records in session window
4. **No Futures activity**: Confirm zero records with `source` containing `futures`

### Scope Leakage Detection

The PO-9 check in the operational script queries for:
- Total venue market orders in 24h
- Non-BTCUSDT venue orders in 24h

Any non-zero count for non-BTCUSDT is a **scope violation** requiring investigation.

## 8. Honest Assessment

### What Is Verified by Code Inspection

- The persistence write path is complete: adapter -> event -> NATS stream -> ClickHouse + KV
- Fee fields (Fee, FeeAsset, CostBasis) are populated by the Spot adapter from real exchange data
- The Simulated flag correctly distinguishes real from paper fills
- Backup infrastructure covers all execution tables
- Read-path queries can retrieve fill data with fee details
- Lifecycle consistency between ClickHouse and NATS KV is maintained by event-driven writes

### What Requires Live Execution to Confirm

- Actual ClickHouse rows exist (depends on writer binary being healthy during session)
- Actual fee values are non-zero (depends on Binance fee schedule and BNB balance)
- Actual NATS KV state reflects the fill (depends on KV consumer running)
- Actual backup contains session data (depends on backup running after persistence completes)
- Actual scope containment (depends on pipeline not generating unexpected intents)

### Residual Gaps

| # | Gap | Severity | Mitigation |
|---|-----|----------|------------|
| 1 | Fee stored as JSON string, not numeric column | LOW | Queryable via JSONExtract functions |
| 2 | No automated cross-store consistency check | LOW | PO-8 added in S447 for manual review |
| 3 | KV stores only latest state, not history | LOW | ClickHouse retains full history |
| 4 | No push notification on persistence failure | LOW | Operator monitors logs; SC-7 is a stop condition |
| 5 | Backup retention is local (7 backups) | LOW | Off-host replication available if configured |

## References

- [Post-Session Operational Verification Protocol](post-session-operational-verification.md) (S447)
- [Supervised Live Session Proof](supervised-live-session-proof.md) (S446)
- [Fee Normalization Model](fee-normalization-model-and-cross-segment-consistency.md) (S428)
- [Fee/Commission Cross-Segment Semantics](fees-commission-assets-cross-segment-semantics-and-limitations.md) (S428)
- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
