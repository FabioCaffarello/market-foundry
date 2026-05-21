# Rejection Persistence and Read-Path Closure

Stage: S411 | Wave: Production Readiness Hardening | Date: 2026-03-23

## Context

S409 evidence gate identified **RG-1** (medium severity): the ClickHouse rejection writer was declared in the NATS registry but never wired into the writer binary's pipeline catalog. Rejection events reached the NATS KV read model (latest-only semantics) but not ClickHouse, creating two gaps:

1. **No historical persistence** -- rejected orders existed only as the latest entry per source/symbol/timeframe in NATS KV, with no append-only audit trail.
2. **No analytical queryability** -- the ClickHouse `executions` table contained paper orders and venue fills but zero rejection rows, making rejection history invisible to the analytical read-path.

## Rejection Path Before S411

```
execute binary
  VenueAdapterActor
    -> PublishRejection (NATS JetStream)
       -> EXECUTION_REJECTION_EVENTS stream
          -> store binary: RejectionConsumer -> RejectionProjectionActor -> NATS KV (latest-only)
          -> writer binary: [MISSING] -- consumer spec existed, pipeline not wired
```

## What Changed

### 1. Writer Pipeline Wiring

Added the `venue_rejection` pipeline entry in `cmd/writer/pipeline.go`:

- Family: `venue_rejection`
- Consumer spec: `WriterVenueMarketOrderRejectionConsumer()` (already defined in S386)
- Target table: `executions` (same table as paper orders and venue fills)
- Enablement: gated by `venue_market_order` execution family flag (rejections are part of venue outcome)
- INSERT SQL: identical column order to fill/paper pipelines -- all execution lifecycle events share the same schema

### 2. Rejection Row Mapper

Added `NewVenueRejectionStarter` and `mapVenueRejectionRow` in `writerpipeline/support.go`:

- Maps `VenueOrderRejectedEvent` to the `executions` table's 20-column schema
- Embeds rejection-specific fields (`rejection_code`, `rejection_reason`, venue details) into the `metadata` JSON column with prefixed keys
- Mirrors the pattern established by the KV projection actor (S407) for metadata enrichment
- Preserves the full correlation/causation chain through `exec_correlation_id` and `exec_causation_id`

### 3. Read-Path (Already Functional)

No read-path changes required. The existing analytical read-path supports rejection queries:

- `ExecutionReader.QueryExecutionHistory` accepts `status="rejected"` as an optional filter
- The `BuildExecutionQuery` function generates `AND status = ?` with `"rejected"` value
- The `metadata` JSON column preserves rejection audit fields for downstream extraction
- Gateway HTTP endpoint `/api/v1/analytical/executions` surfaces rejection history with all standard filters

## Rejection Path After S411

```
execute binary
  VenueAdapterActor
    -> PublishRejection (NATS JetStream)
       -> EXECUTION_REJECTION_EVENTS stream
          -> store binary: RejectionConsumer -> RejectionProjectionActor -> NATS KV (latest-only)
          -> writer binary: RejectionConsumer -> inserterActor -> ClickHouse executions table
```

## Metadata Enrichment Strategy

Rejection-specific fields are embedded into the intent's `Metadata` map before ClickHouse serialization:

| Key | Source | Example |
|-----|--------|---------|
| `rejection_code` | `VenueOrderRejectedEvent.RejectionCode` | `INSUFFICIENT_MARGIN` |
| `rejection_reason` | `VenueOrderRejectedEvent.RejectionReason` | `margin below minimum` |
| `venue_detail.{key}` | `VenueOrderRejectedEvent.VenueDetails` | `venue_detail.exchange_code = -2019` |

This approach:
- Requires no DDL migration (reuses existing `metadata` column)
- Matches the KV projection pattern (S407) for consistent read-path extraction
- Preserves original metadata keys without collision (rejection keys are namespaced)

## Invariants Preserved

- **One table, multiple lifecycle states**: paper orders (submitted), venue fills (filled/partially_filled), and rejections (rejected) coexist in the `executions` table, differentiated by `status` column.
- **Correlation chain**: `exec_correlation_id` and `exec_causation_id` link rejections back to the originating paper order and strategy chain.
- **Idempotent consumer**: the writer consumer uses NATS deduplication keys from the rejection event publisher.
- **Same enablement flag**: rejection pipeline is enabled by `venue_market_order` family flag, matching the fill pipeline.

## Limitations

- **No dedicated rejection columns**: rejection code/reason live in JSON metadata, not first-class columns. Querying by specific rejection codes requires JSON extraction in ClickHouse. This is acceptable at current scale but may warrant a migration if rejection analytics become high-frequency.
- **No rejection-only analytical endpoint**: rejections are queried through the general execution history endpoint with `status=rejected`. A dedicated endpoint could be added if operator workflows require it.
- **NATS KV remains latest-only**: the KV read model still stores only the latest rejection per key. Historical rejection audit relies entirely on ClickHouse.
