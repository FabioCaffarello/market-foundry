# Stage S411: Rejection Persistence and Read-Path Closure

Wave: Production Readiness Hardening | Date: 2026-03-23

## Objective

Close the ClickHouse rejection writer gap (RG-1) identified in S409, ensuring venue order rejections persist to ClickHouse and are queryable through the analytical read-path.

## Strategic Context

RG-1 was the highest-priority residual gap from the S409 evidence gate. The rejection event path (S386) and KV projection (S387) were functional, but the ClickHouse writer pipeline was never wired -- leaving rejection history invisible to the analytical store. This stage closes that specific gap without expanding scope.

## What Was Done

### 1. Rejection Writer Pipeline Wiring

Added the `venue_rejection` pipeline to `cmd/writer/pipeline.go`:

- Consumer: `WriterVenueMarketOrderRejectionConsumer()` (spec existed since S386)
- Table: `executions` (shared with paper orders and venue fills)
- Enablement: `venue_market_order` execution family flag
- No DDL migration required

### 2. Rejection Row Mapper and Consumer Starter

Added `NewVenueRejectionStarter` and `mapVenueRejectionRow` to `internal/adapters/clickhouse/writerpipeline/support.go`:

- Maps `VenueOrderRejectedEvent` to the standard 20-column executions schema
- Embeds rejection-specific audit fields into the `metadata` JSON column
- Mirrors the KV projection metadata enrichment pattern from S407

### 3. Read-Path Verification

Confirmed the existing analytical read-path requires no changes:

- `ExecutionReader.QueryExecutionHistory` supports `status="rejected"` filter
- Gateway HTTP endpoint surfaces rejection history through standard execution queries
- Rejection audit fields are preserved in the metadata JSON column

## Files Changed

| File | Change | Purpose |
|------|--------|---------|
| `internal/adapters/clickhouse/writerpipeline/support.go` | Added `NewVenueRejectionStarter`, `mapVenueRejectionRow` | Consumer starter and row mapper for rejection events |
| `internal/adapters/clickhouse/writerpipeline/support_test.go` | Added 5 tests for rejection row mapper | Validates column count, status, metadata enrichment, nil safety, empty field handling |
| `cmd/writer/pipeline.go` | Added `venue_rejection` pipeline entry | Wires rejection consumer to ClickHouse inserter |
| `docs/architecture/rejection-persistence-and-read-path-closure.md` | New | End-to-end rejection persistence architecture |
| `docs/architecture/rejection-clickhouse-wiring-queryability-and-lifecycle-alignment.md` | New | Wiring spec, queryability contracts, lifecycle alignment |

## Evidence

### Tests (all pass)

| Test | Result |
|------|--------|
| `TestMapVenueRejectionRow_ColumnCount` | PASS -- 20 columns match DDL |
| `TestMapVenueRejectionRow_StatusIsRejected` | PASS -- status is `rejected` |
| `TestMapVenueRejectionRow_MetadataContainsRejectionFields` | PASS -- code, reason, venue details embedded |
| `TestMapVenueRejectionRow_NilMetadataCreatesNew` | PASS -- nil metadata safely handled |
| `TestMapVenueRejectionRow_EmptyRejectionFieldsNotEmbedded` | PASS -- empty strings not polluting metadata |

### Compilation

- `go build ./cmd/writer/...` -- clean
- `go build ./internal/adapters/clickhouse/writerpipeline/...` -- clean
- `go test ./internal/adapters/clickhouse/writerpipeline/...` -- all pass
- `go test ./cmd/writer/...` -- all pass

## RG-1 Disposition

| Before S411 | After S411 |
|-------------|------------|
| Rejection events reach NATS KV only (latest-only) | Rejection events reach both NATS KV and ClickHouse |
| No historical rejection persistence | Append-only historical persistence with 90-day TTL |
| Rejection history invisible to analytical queries | Queryable via `status=rejected` filter on execution history endpoint |
| RG-1 severity: **Medium** | RG-1 severity: **Closed** |

## Residual Gaps

| ID | Description | Severity | Note |
|----|-------------|----------|------|
| L-S411-1 | Rejection code not a first-class ClickHouse column | Low | Queryable via JSON extraction; add column if demand grows |
| L-S411-2 | No dedicated rejection analytical endpoint | Low | Standard execution endpoint with status filter is sufficient |
| L-S411-3 | Batch flush delay for ClickHouse writes | Low | NATS KV provides sub-second latest; ClickHouse is for history |

## Preparation for S412

This stage leaves the system ready for endurance/soak hardening:

- Rejection persistence is now complete end-to-end
- All execution lifecycle states (submitted, filled, partially_filled, rejected) persist to ClickHouse
- The writer binary's pipeline catalog is aligned with all NATS consumer specs in the registry
- No orphaned consumer specs remain in the execution domain
