# Rejection ClickHouse Wiring, Queryability, and Lifecycle Alignment

Stage: S411 | Wave: Production Readiness Hardening | Date: 2026-03-23

## Purpose

This document specifies the ClickHouse wiring for rejection events, the queryability contracts, and the lifecycle alignment between the NATS KV read model and the ClickHouse analytical store.

## ClickHouse Wiring

### Pipeline Definition

| Field | Value |
|-------|-------|
| Family | `venue_rejection` |
| Consumer name | `writer-execution-venue-rejection-consumer` |
| Inserter name | `writer-execution-venue-rejection-inserter` |
| Target table | `executions` |
| Consumer spec | `WriterVenueMarketOrderRejectionConsumer()` |
| Durable name | `writer-execution-venue-rejection` |
| Filter subject | `execution.rejection.venue_market_order.>` |
| Source stream | `EXECUTION_REJECTION_EVENTS` |
| Enablement | `pipeline.execution_families` contains `venue_market_order` |

### Column Mapping

The rejection row mapper produces the same 20 columns as fill and paper order mappers:

| Column | Source | Notes |
|--------|--------|-------|
| event_id | `Metadata.ID` | Unique per rejection event |
| occurred_at | `Metadata.OccurredAt` | Event timestamp |
| correlation_id | `Metadata.CorrelationID` | Envelope correlation |
| causation_id | `Metadata.CausationID` | Envelope causation |
| type | `ExecutionIntent.Type` | `venue_market_order` |
| source | `ExecutionIntent.Source` | e.g. `binances` |
| symbol | `ExecutionIntent.Symbol` | e.g. `btcusdt` |
| timeframe | `ExecutionIntent.Timeframe` | e.g. `60` |
| side | `ExecutionIntent.Side` | `buy` or `sell` |
| quantity | `ExecutionIntent.Quantity` | Requested quantity |
| filled_quantity | `ExecutionIntent.FilledQuantity` | Always `0` for rejections |
| status | `ExecutionIntent.Status` | Always `rejected` |
| risk | `ExecutionIntent.Risk` | JSON -- risk input that led to this order |
| fills | `ExecutionIntent.Fills` | JSON -- empty for rejections |
| parameters | `ExecutionIntent.Parameters` | JSON -- execution parameters |
| metadata | enriched map | JSON -- original metadata + rejection audit fields |
| exec_correlation_id | `ExecutionIntent.CorrelationID` | Intent-level correlation |
| exec_causation_id | `ExecutionIntent.CausationID` | Intent-level causation |
| final | `ExecutionIntent.Final` | Always `true` for rejections |
| timestamp | `ExecutionIntent.Timestamp` | Intent timestamp |

### Metadata Enrichment

The metadata column includes rejection-specific fields injected by `mapVenueRejectionRow`:

```json
{
  "origin": "testnet",
  "rejection_code": "INSUFFICIENT_MARGIN",
  "rejection_reason": "margin below minimum for ETHUSDT",
  "venue_detail.exchange_code": "-2019",
  "venue_detail.msg": "Margin is insufficient."
}
```

Empty rejection fields are not embedded. Original metadata keys are preserved.

## Queryability

### Analytical Read-Path

Rejections are queryable through the existing analytical execution history endpoint:

```
GET /api/v1/analytical/executions?type=venue_market_order&source=binances&symbol=btcusdt&timeframe=60&status=rejected&limit=50
```

The query translates to:

```sql
SELECT type, source, symbol, timeframe, side, quantity, filled_quantity, status,
       risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id,
       final, timestamp
FROM executions
WHERE type = 'venue_market_order' AND source = 'binances' AND symbol = 'btcusdt'
  AND timeframe = 60 AND status = 'rejected'
ORDER BY timestamp DESC
LIMIT 50
```

### NATS KV Read-Path (Unchanged)

The NATS KV read model provides latest-only rejection state:

- Dedicated route: `execution.query.venue_rejection.latest` -- returns intent + rejection detail
- Composite route: `execution.query.status.latest` -- returns intent, fill, rejection, gate, and propagation

### Correlation Chain

A rejection's full lineage is traceable via:

1. `exec_correlation_id` -- links to the originating pipeline execution
2. `exec_causation_id` -- links to the immediate cause (strategy/risk output)
3. `correlation_id` (envelope) -- links to the NATS message correlation
4. `metadata.rejection_code` -- venue-specific rejection reason
5. `metadata.venue_detail.*` -- raw venue response fields

## Lifecycle Alignment

### Execution Status Values in ClickHouse

| Status | Producer | Lifecycle stage |
|--------|----------|-----------------|
| `submitted` | derive binary (paper_order) | Intent created |
| `filled` | execute binary (venue fill) | Order accepted and filled |
| `partially_filled` | execute binary (venue fill) | Order partially filled |
| `rejected` | execute binary (venue rejection) | Order rejected by venue |

### Consistency Between Read Models

| Dimension | NATS KV | ClickHouse |
|-----------|---------|------------|
| Semantics | Latest-only per key | Append-only historical |
| Latency | Sub-second | Batch (flush interval) |
| Retention | Until overwritten | 90-day TTL |
| Rejection detail | Embedded in metadata | Embedded in metadata |
| Query granularity | Single key lookup | Range/filter queries |

Both read models use the same metadata enrichment pattern (rejection fields embedded in intent metadata), ensuring consistent extraction logic across read paths.

## Evidence

### Tests

| Test | File | Validates |
|------|------|-----------|
| `TestMapVenueRejectionRow_ColumnCount` | `writerpipeline/support_test.go` | 20-column alignment with DDL |
| `TestMapVenueRejectionRow_StatusIsRejected` | `writerpipeline/support_test.go` | Status field correctness |
| `TestMapVenueRejectionRow_MetadataContainsRejectionFields` | `writerpipeline/support_test.go` | Rejection code, reason, venue details in metadata |
| `TestMapVenueRejectionRow_NilMetadataCreatesNew` | `writerpipeline/support_test.go` | Safe handling of nil metadata map |
| `TestMapVenueRejectionRow_EmptyRejectionFieldsNotEmbedded` | `writerpipeline/support_test.go` | Empty strings not polluting metadata |

### Compilation

- `cmd/writer` compiles with the new pipeline entry
- `internal/adapters/clickhouse/writerpipeline` compiles with new starter and mapper

## Limitations and Residual Gaps

| ID | Description | Severity | Mitigation |
|----|-------------|----------|------------|
| L-S411-1 | Rejection code not a first-class ClickHouse column | Low | Queryable via JSONExtractString; add column if analytics demand grows |
| L-S411-2 | No dedicated rejection analytical endpoint | Low | Use general execution history with status=rejected filter |
| L-S411-3 | Batch flush delay means rejections are not immediately queryable in ClickHouse | Low | NATS KV provides sub-second latest-only read; ClickHouse is for history |
