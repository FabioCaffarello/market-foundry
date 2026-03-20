# Writer Service — Minimal Implementation

## Purpose

This document describes the first implementation of the writer service (`cmd/writer/`), a standalone runtime that consumes canonical domain events from NATS JetStream and appends them to ClickHouse as an analytical projection.

The writer is **lateral, optional, and append-only**. Removing or stopping it has zero impact on the operational pipeline.

## Runtime Architecture

The writer follows the canonical 6-phase composition root used by all market-foundry runtimes:

1. **Config** — Load and validate from JSONC (includes ClickHouse section).
2. **Logger** — Structured slog with `runtime=writer`.
3. **ClickHouse client** — Native protocol connection to ClickHouse.
4. **Actor engine** — Hollywood actor framework.
5. **Supervisor** — Spawns consumer-inserter pairs per enabled pipeline family.
6. **Health server** — `/healthz`, `/readyz` (NATS + ClickHouse), `/statusz`, `/diagz`.

## Actor Hierarchy

```
writerSupervisor
├── writer-candle-inserter
├── writer-candle-consumer
├── writer-signal-rsi-inserter
├── writer-signal-rsi-consumer
├── writer-decision-rsi-oversold-inserter
├── writer-decision-rsi-oversold-consumer
├── writer-strategy-mean-reversion-entry-inserter
├── writer-strategy-mean-reversion-entry-consumer
├── writer-risk-position-exposure-inserter
├── writer-risk-position-exposure-consumer
├── writer-execution-paper-order-inserter
└── writer-execution-paper-order-consumer
```

Each pipeline family produces a **consumer-inserter pair**:

- **Consumer actor** — Wraps the existing NATS consumer types (EvidenceConsumer, SignalConsumer, etc.) with a handler that maps decoded events to ClickHouse row values and sends them to the inserter actor.
- **Inserter actor** — Buffers rows and batch-inserts into ClickHouse when either the batch size is reached or the flush interval timer fires.

## NATS Consumer Design

Writer consumers reuse the existing NATS consumer infrastructure (`internal/adapters/nats/`). Each writer consumer has an independent durable name with `writer-` prefix, guaranteeing independent cursors from the store consumers.

| Family | Durable Name | Stream | Filter Subject |
|--------|-------------|--------|---------------|
| candle | `writer-candle` | EVIDENCE_EVENTS | `evidence.events.candle.sampled.>` |
| rsi | `writer-signal-rsi` | SIGNAL_EVENTS | `signal.events.rsi.generated.>` |
| rsi_oversold | `writer-decision-rsi-oversold` | DECISION_EVENTS | `decision.events.rsi_oversold.evaluated.>` |
| mean_reversion_entry | `writer-strategy-mean-reversion-entry` | STRATEGY_EVENTS | `strategy.events.mean_reversion_entry.resolved.>` |
| position_exposure | `writer-risk-position-exposure` | RISK_EVENTS | `risk.events.position_exposure.assessed.>` |
| paper_order | `writer-execution-paper-order` | EXECUTION_EVENTS | `execution.events.paper_order.submitted.>` |

## Batch Buffering Strategy

| Parameter | Default | Config Key |
|-----------|---------|------------|
| Batch size | 1000 | `clickhouse.batch_size` |
| Flush interval | 5s | `clickhouse.flush_interval` |
| Max pending | 10000 | `clickhouse.max_pending` |

**Flush triggers:**
1. Buffer reaches `batch_size` → immediate flush.
2. Flush interval timer fires → flush whatever is buffered.
3. Actor stopped → drain remaining buffer.

**Overflow:** When buffer exceeds `max_pending`, oldest rows are evicted (FIFO). Evictions are logged at WARN and counted in the `events_dropped` tracker counter.

## Event-to-Row Mapping

Each domain event type is mechanically mapped to ClickHouse column values by a dedicated mapper function in `cmd/writer/mappers.go`. The mapping performs:

- **Metadata extraction** — `event_id`, `occurred_at`, `correlation_id`, `causation_id` from `events.Metadata`.
- **Decimal-to-Float64 conversion** — Go decimal strings (`Open`, `High`, `Close`, etc.) parsed to `float64`.
- **Nested-to-JSON serialization** — Maps, slices, and structs serialized to JSON strings for ClickHouse String columns.
- **Type narrowing** — `Timeframe` int → `uint32`, enum strings passed through.

No transformation, filtering, aggregation, or deduplication is performed.

## ClickHouse Adapter

The ClickHouse adapter (`internal/adapters/clickhouse/`) provides a minimal native-protocol client:

- `Open(Config)` — Establishes connection.
- `Ping(ctx)` — Readiness check.
- `InsertBatch(ctx, insertSQL, rows)` — Batch append using the ClickHouse batch protocol.
- `Close()` — Shutdown.

The adapter is a separate Go module. Only `cmd/writer/` imports it — no operational service has any ClickHouse dependency.

## Health Model

| Endpoint | Behavior |
|----------|----------|
| `/healthz` | Always 200 (liveness). |
| `/readyz` | NATS TCP dial + ClickHouse ping. Returns 503 if either fails. |
| `/statusz` | Per-family consumer and inserter tracker stats. |
| `/diagz` | Full diagnostic summary with readiness checks and tracker details. |

## Configuration

Writer uses the standard `AppConfig` with an additional `clickhouse` section:

```jsonc
{
  "clickhouse": {
    "addr": "clickhouse:9000",
    "database": "default",
    "username": "default",
    "password": "clickhouse",
    "batch_size": 1000,
    "flush_interval": "5s",
    "max_pending": 10000
  }
}
```

The `clickhouse` config section is optional in `AppConfig` and ignored by all other services.

## Docker Compose Integration

The writer service depends on `nats` (healthy) and `clickhouse` (healthy). No other service depends on the writer. Removing the writer entry from docker-compose restores the baseline topology.

## Package Boundaries

| Package | Purpose |
|---------|---------|
| `cmd/writer/` | Runtime entry point, supervisor, consumer/inserter actors, mappers |
| `internal/adapters/clickhouse/` | ClickHouse native protocol client |
| `internal/adapters/nats/` | Writer consumer specs (WriterXxxConsumer functions) |

**Forbidden:** No operational service (`cmd/gateway/`, `cmd/store/`, `cmd/derive/`, `cmd/ingest/`, `cmd/execute/`, `cmd/configctl/`) imports `internal/adapters/clickhouse/`.
