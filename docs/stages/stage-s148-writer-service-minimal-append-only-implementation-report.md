# Stage S148 — Writer Service Minimal Append-Only Implementation Report

## Objective

Implement the first version of the writer service: a standalone runtime consuming canonical domain events from NATS and persisting them to ClickHouse as a lateral, optional, append-only analytical projection.

## Outcome

**Achieved.** The writer service exists as `cmd/writer/`, follows the canonical runtime composition pattern, and persists events from 6 pipeline families into the 6 core ClickHouse tables.

## What Was Implemented

### 1. ClickHouse Adapter (`internal/adapters/clickhouse/`)

New Go module providing a minimal native-protocol client:
- `Client.Open()` — connection management
- `Client.Ping()` — readiness verification
- `Client.InsertBatch()` — batch append using ClickHouse batch protocol
- `Client.Close()` — graceful shutdown

### 2. Writer Service (`cmd/writer/`)

Standalone runtime following the 6-phase canonical pattern:
- **main.go** — Bootstrap via `bootstrap.Main("writer", Run)`
- **run.go** — ClickHouse client init, actor engine, health server with NATS + ClickHouse readiness checks
- **supervisor.go** — Root actor spawning consumer-inserter pairs per enabled family
- **pipeline.go** — Declarative pipeline catalog (6 families) with factory closures
- **consumer.go** — Generic consumer actor wrapping existing NATS consumer types
- **inserter.go** — Batch inserter actor with configurable batch_size, flush_interval, max_pending, FIFO eviction
- **mappers.go** — Mechanical event-to-row mapping with decimal→float64 conversion and nested→JSON serialization

### 3. NATS Consumer Specs

Added writer-prefixed durable consumers to all 6 domain registries:
- `WriterCandleConsumer()` — evidence_registry.go
- `WriterRSISignalConsumer()` — signal_registry.go
- `WriterRSIOversoldDecisionConsumer()` — decision_registry.go
- `WriterMeanReversionEntryStrategyConsumer()` — strategy_registry.go
- `WriterPositionExposureRiskConsumer()` — risk_registry.go
- `WriterPaperOrderExecutionConsumer()` — execution_registry.go

### 4. Settings Extension

Added `ClickHouseConfig` to `AppConfig` with:
- Connection parameters (addr, database, username, password)
- Batching parameters (batch_size, flush_interval, max_pending)
- Validation (only when configured)
- Default helpers (BatchSizeOrDefault, FlushIntervalOrDefault, MaxPendingOrDefault)

### 5. Configuration

- `deploy/configs/writer.jsonc` — Full writer config with ClickHouse section

### 6. Docker Compose Integration

- Writer service added depending on `nats` (healthy) and `clickhouse` (healthy)
- No other service depends on the writer
- Port 8085 for health endpoints
- Healthcheck via `/readyz`

### 7. Go Workspace

- `cmd/writer` and `internal/adapters/clickhouse` added to `go.work`

## Files Changed

### New Files
| File | Purpose |
|------|---------|
| `internal/adapters/clickhouse/go.mod` | ClickHouse adapter module |
| `internal/adapters/clickhouse/client.go` | Native protocol client |
| `cmd/writer/go.mod` | Writer service module |
| `cmd/writer/main.go` | Entry point |
| `cmd/writer/run.go` | Bootstrap and composition root |
| `cmd/writer/supervisor.go` | Root supervisor actor |
| `cmd/writer/pipeline.go` | Pipeline declarations and tracker defs |
| `cmd/writer/consumer.go` | Generic consumer actor |
| `cmd/writer/inserter.go` | Batch inserter actor |
| `cmd/writer/mappers.go` | Event-to-row mapping functions |
| `deploy/configs/writer.jsonc` | Writer service configuration |
| `docs/architecture/writer-service-minimal-implementation.md` | Implementation architecture |
| `docs/architecture/writer-service-initial-event-coverage-and-limits.md` | Coverage, limits, semantics |

### Modified Files
| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added ClickHouseConfig to AppConfig |
| `internal/adapters/nats/evidence_registry.go` | Added WriterCandleConsumer |
| `internal/adapters/nats/signal_registry.go` | Added WriterRSISignalConsumer |
| `internal/adapters/nats/decision_registry.go` | Added WriterRSIOversoldDecisionConsumer |
| `internal/adapters/nats/strategy_registry.go` | Added WriterMeanReversionEntryStrategyConsumer |
| `internal/adapters/nats/risk_registry.go` | Added WriterPositionExposureRiskConsumer |
| `internal/adapters/nats/execution_registry.go` | Added WriterPaperOrderExecutionConsumer |
| `deploy/compose/docker-compose.yaml` | Added writer service |
| `go.work` | Added cmd/writer and internal/adapters/clickhouse |

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Writer exists and persists events to ClickHouse | Done — 6 families, 6 tables |
| Writing is lateral, optional, and append-only | Done — no service depends on writer |
| Minimal diagnostics present | Done — /healthz, /readyz, /statusz, /diagz with per-family trackers |
| Failure semantics and limits documented | Done — writer-service-initial-event-coverage-and-limits.md |
| Ready for minimal historical query | Done — tables populated, queryable via ClickHouse client |

## Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| Writer not in critical path | Compliant — no operational service depends on writer or ClickHouse |
| No cold-start bootstrap | Compliant — deferred |
| No excessive event coverage | Compliant — 6 primary families only (tradeburst, volume, ema_crossover, venue_market_order deferred) |
| No coupling to existing services | Compliant — writer imports from adapters only, no bidirectional dependency |
| Out-of-scope documented | Compliant — limits section in coverage doc |

## Invariant Compliance

| Invariant | Verification |
|-----------|-------------|
| INV-01: No operational dependency on ClickHouse | No operational service imports clickhouse adapter |
| INV-02: Writer uses independent consumer names | All durables prefixed `writer-` |
| INV-03: Writer tolerates ClickHouse absence | Buffer fills, evicts, consumer continues |
| INV-04: Existing smoke tests pass without writer | All tests pass (verified) |

## Preparation for S149

The following capabilities are ready for the next stage:

1. **Historical query endpoints** — ClickHouse tables are populated and queryable. Gateway can add `/evidence/candles/history` backed by ClickHouse.
2. **Additional family coverage** — Adding tradeburst, volume, ema_crossover, venue_market_order requires only a pipeline entry + mapper per family.
3. **Cold-start bootstrap** — Derive can query `evidence_candles` for warm-up data.
4. **INSERT retry** — Current single-attempt insert can be extended with exponential backoff.

## Recommended Next Steps

1. **S149**: Minimal historical query endpoint via gateway (ClickHouse-backed `/evidence/candles/history`).
2. Extend writer coverage to remaining families (tradeburst, volume, ema_crossover, venue_market_order).
3. Add writer to smoke test pipeline (optional — verify writes after live pipeline activation).
