# Writer Service Architecture

> Formal architectural decision for the analytical writer service.
> Stage: S145 — Writer Service Architecture Decision.

## 1. Decision Summary

The writer is a **standalone, dedicated runtime** (`cmd/writer/`) that consumes canonical events from NATS JetStream and appends them to ClickHouse as an analytical projection. It is the sole bridge between the operational pipeline and the analytical layer.

**Decision:** Dedicated service. Not an optional module inside `store`, not a sidecar, not a library.

## 2. Rationale

| Alternative Considered | Verdict | Why Rejected |
|------------------------|---------|--------------|
| Optional module in `store` | Rejected | Couples analytical failure to operational KV projection; violates INV-01 and R-01 |
| Sidecar per operational service | Rejected | Multiplies ClickHouse connections; no single consumer cursor; complicates lifecycle |
| Shared library consumed by `store` | Rejected | Same coupling problem as module; `store` would import ClickHouse driver |
| Dedicated `cmd/writer/` service | **Accepted** | Clean failure isolation, independent lifecycle, trivially removable from compose |

The dedicated service model achieves three non-negotiable properties:
1. **Failure isolation** — writer crash or ClickHouse outage cannot affect KV projections or the operational pipeline.
2. **Lifecycle independence** — writer can start, stop, restart, or be absent without any operational service noticing.
3. **Structural optionality** — removing the `writer` container from docker-compose is the complete removal path; no config flags, no conditional branches.

## 3. Runtime Structure

The writer follows the canonical 6-phase composition root established by all market-foundry runtimes:

```
Phase 1: Logger installation
Phase 2: Actor engine creation
Phase 3: Health tracker creation (one pair per pipeline family)
Phase 4: Supervisor spawn (WriterSupervisor)
Phase 5: Health server start (background)
Phase 6: Shutdown coordination (WaitTillShutdown)
```

### 3.1 Entry Point

```
cmd/writer/
  main.go         — bootstrap.Main("writer", Run)
  run.go          — 6-phase composition root
```

### 3.2 Actor Hierarchy

```
WriterSupervisor
  ├── candle-consumer          → candle-inserter
  ├── signal-consumer          → signal-inserter
  ├── decision-consumer        → decision-inserter
  ├── strategy-consumer        → strategy-inserter
  ├── risk-consumer            → risk-inserter
  └── execution-consumer       → execution-inserter
```

Each pipeline family is a **consumer–inserter pair**:
- **Consumer actor** — owns a NATS durable consumer, deserializes events, forwards to inserter via actor message.
- **Inserter actor** — accumulates events in a batch buffer, flushes to ClickHouse on size or time threshold.

### 3.3 Declarative Pipeline Catalog

The writer supervisor declares pipelines using the same declarative pattern as `store`:

```go
type WriterPipeline struct {
    Family       string
    ConsumerName string     // "writer-{family}-consumer"
    InserterName string     // "writer-{family}-inserter"
    IsEnabled    func(WriterConfig) bool
    NewConsumer  func(...) actor.Producer
    NewInserter  func(...) actor.Producer
}
```

Each pipeline entry is self-contained. Enabling or disabling a family is a configuration-level decision, not a code change.

## 4. NATS Consumption Pattern

### 4.1 Consumer Identity

The writer uses **independent durable consumer names** prefixed with `writer-`:

| Family | Store Consumer Name | Writer Consumer Name |
|--------|---------------------|----------------------|
| candle | `store-evidence-candle-consumer` | `writer-evidence-candle-consumer` |
| signal | `store-signal-consumer` | `writer-signal-consumer` |
| decision | `store-decision-consumer` | `writer-decision-consumer` |
| strategy | `store-strategy-consumer` | `writer-strategy-consumer` |
| risk | `store-risk-consumer` | `writer-risk-consumer` |
| execution | `store-execution-consumer` | `writer-execution-consumer` |

This guarantees:
- Store and writer maintain **independent cursors** — neither blocks the other.
- Writer restart replays from its own last-acked position, not store's.
- No mutual awareness between store and writer.

### 4.2 Subject Consumption

The writer subscribes to the same NATS subjects as store:

| Family | Subject Pattern |
|--------|----------------|
| candle | `EVIDENCE_EVENTS.candle.sampled` |
| signal | `signal.*` |
| decision | `decision.*` |
| strategy | `strategy.*` |
| risk | `risk.*` |
| execution | `execution.*` |

No new NATS subjects, streams, or consumers are created beyond the `writer-*` durable consumers.

### 4.3 Message Flow

```
NATS JetStream
  → writer durable consumer delivers message
    → Consumer actor deserializes JSON → Go struct
    → Consumer actor sends typed message to inserter actor
      → Inserter actor appends to batch buffer
      → On threshold: flush batch to ClickHouse
      → On success: ack all messages in batch
      → On failure: buffer continues; see failure semantics
```

**Critical rule:** The consumer actor **never acks a message** directly. Ack responsibility belongs to the inserter actor after successful ClickHouse write. This ensures that unwritten events are re-delivered on restart.

## 5. ClickHouse Write Policy

### 5.1 Batch Buffering

| Parameter | Default | Description |
|-----------|---------|-------------|
| `batch_size` | 1000 | Events per flush |
| `flush_interval` | 5s | Maximum time before flush (even if batch_size not reached) |
| `max_pending` | 10000 | Maximum buffered events before drop policy activates |

The inserter actor flushes when **either** threshold is met (whichever comes first).

### 5.2 Insert Mechanics

Each flush executes a single ClickHouse batch INSERT:

```sql
INSERT INTO {table} (col1, col2, ...) VALUES (?, ?, ...), (?, ?, ...), ...
```

- One INSERT per table per flush (no cross-table transactions).
- ClickHouse's MergeTree engine handles append-only ingestion natively.
- `ingested_at` is populated by ClickHouse via `DEFAULT now64(3)`.

### 5.3 Data Transformation

The inserter performs **minimal, mechanical transformation** — no business logic:

| Step | Description |
|------|-------------|
| Metadata extraction | `event_id`, `occurred_at`, `correlation_id`, `causation_id` from event envelope |
| Decimal parsing | String decimal fields → `Float64` |
| JSON serialization | Nested structs (inputs, constraints, fills) → JSON string |
| LowCardinality mapping | Enum-like fields (source, symbol, timeframe, phase, outcome) passed as strings |

**Anti-pattern:** The inserter must never filter, aggregate, deduplicate, or enrich events. It is a mechanical append path.

## 6. Configuration

### 6.1 Config File

```jsonc
// deploy/configs/writer.jsonc
{
  "log": { "level": "info", "format": "text" },
  "http": { "addr": ":8086" },
  "nats": {
    "enabled": true,
    "url": "nats://nats:4222",
    "request_timeout": "2s"
  },
  "clickhouse": {
    "dsn": "clickhouse://clickhouse:9000/market_foundry",
    "max_open_conns": 5,
    "dial_timeout": "5s"
  },
  "writer": {
    "batch_size": 1000,
    "flush_interval": "5s",
    "max_pending": 10000,
    "families": ["candle", "signal", "decision", "strategy", "risk", "execution"]
  }
}
```

### 6.2 Docker Compose

```yaml
writer:
  build:
    context: .
    dockerfile: build/Dockerfile
    args:
      SERVICE: writer
  depends_on:
    nats:
      condition: service_healthy
    clickhouse:
      condition: service_healthy
  volumes:
    - ./deploy/configs/writer.jsonc:/app/config.jsonc:ro
  restart: unless-stopped
```

**Critical:** Only `writer` declares `depends_on: clickhouse`. No operational service references ClickHouse.

## 7. Health Model

| Endpoint | Behavior |
|----------|----------|
| `/healthz` | Always 200 (process alive) |
| `/readyz` | 200 only if NATS **and** ClickHouse reachable |
| `/statusz` | Per-pipeline tracker stats (event counts, error counts, idle time, custom counters) |
| `/diagz` | Detailed diagnostic output |

**Readiness justification:** Writer's readiness correctly includes ClickHouse because writing to ClickHouse is its sole purpose. No other service's readiness includes ClickHouse.

### 7.1 Tracker Counters

Each inserter pipeline exposes:

| Counter | Meaning |
|---------|---------|
| `events_buffered` | Events currently in batch buffer |
| `events_flushed` | Total events written to ClickHouse |
| `events_dropped` | Events dropped due to max_pending overflow |
| `flush_errors` | Failed ClickHouse write attempts |
| `flush_count` | Total successful flushes |

## 8. Shutdown Sequence

```
1. SIGINT/SIGTERM received
2. WaitTillShutdown(engine, writerSupervisorPID) with 10s timeout
3. Supervisor sends poison to all children
4. Each inserter actor:
   a. Attempts final flush of buffered events (best-effort, bounded by shutdown timeout)
   b. Logs buffer state (events flushed / events dropped)
5. Each consumer actor:
   a. Stops NATS consumer (unsubscribes)
   b. Logs consumer position
6. All actors stop
7. Health server graceful shutdown (5s)
8. ClickHouse connection pool closes (deferred)
9. NATS connection closes (deferred)
```

## 9. First Version Limits

The first writer implementation is deliberately narrow:

| Limit | Description |
|-------|-------------|
| L-01 | 6 families only (candle, signal, decision, strategy, risk, execution) |
| L-02 | No deduplication (MergeTree accepts duplicates; query-time dedup if needed) |
| L-03 | No transformation beyond mechanical type mapping |
| L-04 | No materialized views or pre-aggregations |
| L-05 | No backfill capability (processes only from consumer cursor forward) |
| L-06 | No dynamic family registration (families are compile-time pipeline entries) |
| L-07 | Single ClickHouse instance (no cluster, no sharding) |

These limits are intentional guard rails, not technical debt. Each may be relaxed in future stages with explicit justification.
