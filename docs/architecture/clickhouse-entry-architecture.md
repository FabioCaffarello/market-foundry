# ClickHouse Entry Architecture

> **Stage:** S143 — Migrations and ClickHouse Entry Architecture
> **Status:** Definitive
> **Scope:** Architectural definition only. No implementation.

---

## 1. Purpose

This document defines the formal architecture for ClickHouse's entry into Market Foundry. It establishes the role of ClickHouse in the system, the boundaries between operational and analytical layers, the semantic position of the writer, and the invariants that must hold throughout all implementation phases.

This is the architectural blueprint that PC-01 through PC-03 (from `clickhouse-and-migrations-preparation-gate.md`) will be built against. Nothing in this document authorizes implementation — it authorizes **design coherence**.

---

## 2. Role of ClickHouse in Market Foundry

### 2.1 What ClickHouse Is

ClickHouse is an **analytical projection layer** — a durable, queryable, time-series archive of events that already flow through NATS. It serves three purposes:

| Purpose | Description | Consumer |
|---------|-------------|----------|
| **Historical archive** | Persist domain events beyond NATS 72h retention | Gateway (future historical endpoints) |
| **Analytical surface** | Enable cross-session queries, trend analysis, backtesting | Developer / future tooling |
| **Cold-start bootstrap** | Provide historical candles for RSI warm-up on derive restart | Derive service (future, conditional) |

### 2.2 What ClickHouse Is NOT

| ClickHouse is NOT | Why |
|---|---|
| An operational dependency | P-01: pipeline runs without it |
| A replacement for NATS KV | KV serves "latest" queries; ClickHouse serves "history" queries |
| A source of truth for state | NATS events are the source of truth; ClickHouse is a projection |
| A participant in the hot path | Writer is async, off the event processing path |
| A message bus or event producer | Writer consumes only; never publishes back to NATS |

### 2.3 Architectural Classification

```
┌─────────────────────────────────────────────────────────┐
│                   OPERATIONAL LAYER                      │
│  (NATS JetStream + KV — the proven baseline)            │
│                                                          │
│  ingest → derive → store (KV projection) → gateway      │
│                        ↓ execute                         │
│                                                          │
│  Invariant: functions identically with or without CH     │
└─────────────────────────┬───────────────────────────────┘
                          │ NATS events (read-only tap)
                          │
┌─────────────────────────▼───────────────────────────────┐
│                   ANALYTICAL LAYER                        │
│  (ClickHouse — optional augmentation)                    │
│                                                          │
│  writer (NATS consumer) → ClickHouse tables              │
│                                  ↑                       │
│                           gateway (historical queries)   │
│                           derive  (cold-start bootstrap) │
│                                                          │
│  Invariant: removable without affecting operational loop │
└─────────────────────────────────────────────────────────┘
```

The **boundary between layers** is the NATS event stream. The operational layer produces events. The analytical layer consumes them. There is no bidirectional coupling.

---

## 3. Writer Architecture

### 3.1 Semantic Role

The writer is a **NATS consumer that projects events into ClickHouse** — semantically identical to how `store` projects events into NATS KV. Both are parallel consumers of the same event streams. Neither knows the other exists.

### 3.2 Deployment Decision: Standalone Service

The writer MUST be a standalone service (`cmd/writer/`), not an optional module inside `store`.

**Rationale:**

| Factor | Standalone `cmd/writer/` | Module inside `store` |
|--------|--------------------------|----------------------|
| Failure isolation | Writer crash doesn't affect KV projection | Writer bug could crash store |
| Lifecycle independence | Start/stop/restart independently | Coupled to store lifecycle |
| Resource isolation | Own memory/CPU limits in compose | Shares store's budget |
| Dependency clarity | Depends on NATS + CH only | Store would gain CH dependency |
| Optionality | Remove container from compose = gone | Conditional code paths in store |
| Consumer isolation | Own durable consumer names by design | Risk of consumer name collision |

### 3.3 Internal Architecture

```
cmd/writer/
├── main.go              # Bootstrap (same pattern as other services)
└── run.go               # AppConfig + supervisor spawn

internal/actors/scopes/writer/
├── writer_supervisor.go     # Spawns per-family consumer+inserter pairs
├── evidence_consumer.go     # NATS consumer for EVIDENCE_EVENTS
├── signal_consumer.go       # NATS consumer for signal events
├── decision_consumer.go     # NATS consumer for decision events
├── strategy_consumer.go     # NATS consumer for strategy events
├── risk_consumer.go         # NATS consumer for risk events
├── execution_consumer.go    # NATS consumer for execution events
└── ch_inserter_actor.go     # Batch buffer → ClickHouse INSERT

internal/adapters/clickhouse/
├── client.go                # Connection management, health check
├── evidence_writer.go       # INSERT for evidence tables
├── signal_writer.go         # INSERT for signal tables
├── ...                      # One writer per table family
└── batch.go                 # Batch accumulation and flush logic
```

### 3.4 Consumer Pattern

The writer follows the **exact same dual-actor pattern** as store:

1. **Consumer actor** — owns the NATS durable consumer, deserializes events, forwards to inserter
2. **Inserter actor** — accumulates a batch buffer, flushes to ClickHouse on size or time threshold

**Durable consumer names** use the prefix `writer-` (never `store-`):

| Stream | Store consumer | Writer consumer |
|--------|---------------|-----------------|
| EVIDENCE_EVENTS | `store-evidence-candle-consumer` | `writer-evidence-candle-consumer` |
| Signal events | `store-signal-rsi-consumer` | `writer-signal-rsi-consumer` |
| ... | `store-*` | `writer-*` |

### 3.5 Batch Buffering Strategy

The inserter accumulates events in memory and flushes to ClickHouse when either threshold is met:

| Parameter | Default | Rationale |
|-----------|---------|-----------|
| `batch_size` | 1000 events | ClickHouse prefers bulk inserts over individual rows |
| `flush_interval` | 5 seconds | Bounded latency even at low throughput |
| `max_pending` | 10000 events | Backpressure threshold — drop oldest if exceeded |

When ClickHouse is unavailable:
- Buffer accumulates up to `max_pending`
- Beyond `max_pending`, oldest events are dropped (logged as metric)
- No NATS Nak — the consumer continues advancing (events can be replayed from stream if needed)
- Writer logs degraded state; health endpoint reports unhealthy

### 3.6 Health and Observability

The writer follows the same health pattern as all services:

- `/healthz` — liveness (always 200 if process is running)
- `/readyz` — readiness (NATS connected AND ClickHouse pingable)
- `/statusz` — per-consumer tracker stats (received, inserted, dropped, errors)
- `/diagz` — runtime diagnostics

**Critical distinction:** The writer's readiness check includes ClickHouse. This is correct because the writer's purpose is to write to ClickHouse. However, **no other service's readiness check includes ClickHouse** — this is how P-01 (optional) is preserved.

---

## 4. Data Flow Architecture

### 4.1 Event Flow (With Writer Running)

```
Binance WS
    │
    ▼
  ingest ──publish──▶ NATS EVIDENCE_EVENTS
                          │
                ┌─────────┼─────────┐
                ▼         ▼         ▼
             derive     store     writer
               │          │         │
               ▼          ▼         ▼
          (pipeline)   NATS KV   ClickHouse
               │          │
               ▼          ▼
           execute     gateway
```

### 4.2 Event Flow (Without Writer — Baseline Mode)

```
Binance WS
    │
    ▼
  ingest ──publish──▶ NATS EVIDENCE_EVENTS
                          │
                ┌─────────┤
                ▼         ▼
             derive     store
               │          │
               ▼          ▼
           execute     gateway
```

The two diagrams are **functionally identical** for the operational pipeline. The writer's presence or absence is invisible to all other services.

### 4.3 Table-to-Stream Mapping

| ClickHouse Table | NATS Source | Event Type | Priority |
|-----------------|-------------|------------|----------|
| `evidence_candles` | EVIDENCE_EVENTS (filter: candle) | Candle close | P1 |
| `evidence_tradebursts` | EVIDENCE_EVENTS (filter: tradeburst) | Trade burst | P1 |
| `evidence_volumes` | EVIDENCE_EVENTS (filter: volume) | Volume profile | P1 |
| `signals` | signal.* | RSI, EMA crossover | P1 |
| `decisions` | decision.* | RSI oversold | P1 |
| `strategies` | strategy.* | Mean reversion entry | P1 |
| `risk_assessments` | risk.* | Position exposure | P1 |
| `executions` | execution.* | Paper order, venue order | P1 |
| `fills` | fill.* | Fill events | P1 |
| `runtime_telemetry` | (scraper, not NATS) | Operational metrics | P2 |

---

## 5. Schema Architecture

### 5.1 Schema Derivation Rule

Every ClickHouse table schema is **derived from the corresponding Go event struct**. The Go struct is the source of truth. The ClickHouse DDL is a projection.

```
Go struct (internal/domain/) → ClickHouse DDL (deploy/migrations/)
```

**Schema derivation checklist per table:**

1. Identify the Go event struct
2. Map each Go field to a ClickHouse column type
3. Add `ingested_at DateTime64(3) DEFAULT now64(3)` as a metadata column
4. Choose ENGINE (MergeTree for all domain tables)
5. Choose PARTITION BY (timeframe + month for time-series; month for non-timeframed)
6. Choose ORDER BY (source + symbol + timeframe + timestamp)
7. Choose TTL (per retention policy)

### 5.2 Type Mapping Convention

| Go Type | ClickHouse Type | Notes |
|---------|----------------|-------|
| `string` (low cardinality: source, symbol, phase) | `LowCardinality(String)` | Enum-like strings |
| `string` (free text) | `String` | |
| `float64` | `Float64` | Prices, volumes |
| `int`, `int64` | `Int64` | |
| `uint32` | `UInt32` | Counts, timeframe seconds |
| `bool` | `Bool` | |
| `time.Time` | `DateTime64(3)` | Millisecond precision |

### 5.3 Partitioning Strategy

| Table Category | PARTITION BY | Rationale |
|---------------|-------------|-----------|
| Evidence (candles, bursts, volume) | `(timeframe, toYYYYMM(open_time))` | Query patterns are per-timeframe-per-month |
| Signals, decisions, strategies | `toYYYYMM(timestamp)` | No timeframe dimension in signal events |
| Risk, executions, fills | `toYYYYMM(timestamp)` | Chronological access |
| Runtime telemetry | `toYYYYMM(timestamp)` | Chronological, high TTL churn |

### 5.4 Retention Policy

| Table Category | TTL | Rationale |
|---------------|-----|-----------|
| Evidence | 90 days | Sufficient for backtesting at current scale |
| Signals through executions | 90 days | Follows evidence lifecycle |
| Fills | 365 days | Audit/compliance trail |
| Runtime telemetry | 30 days | Operational, high volume |

TTL changes require a new migration (`ALTER TABLE ... MODIFY TTL`).

---

## 6. Docker-Compose Integration

### 6.1 Writer Service Definition

```yaml
writer:
  build:
    context: ../..
    dockerfile: build/writer/Dockerfile
  container_name: market-foundry-writer
  ports:
    - "127.0.0.1:8085:8085"
  env_file:
    - ../envs/local.env
  depends_on:
    nats:
      condition: service_healthy
    clickhouse:
      condition: service_healthy
  networks:
    - market-foundry-network
  deploy:
    resources:
      limits:
        memory: 512M
        cpus: "0.50"
```

### 6.2 Dependency Rules

| Service | Depends on ClickHouse? | Depends on Writer? |
|---------|----------------------|-------------------|
| nats | No | No |
| configctl | No | No |
| gateway | No | No |
| ingest | No | No |
| derive | No | No |
| store | No | No |
| execute | No | No |
| **writer** | **Yes** | N/A |
| **clickhouse** | N/A | No |

**No service except writer has any dependency on ClickHouse.** This is the structural enforcement of P-01.

---

## 7. Query Surface Architecture (Future — Out of S143 Scope)

For completeness, the intended query surface follows:

| Query Type | Source | Endpoint Pattern |
|-----------|--------|-----------------|
| Latest value | NATS KV (via store) | `GET /evidence/candle/{source}/{symbol}/{timeframe}` (existing) |
| Historical range | ClickHouse (via gateway) | `GET /evidence/candle/{source}/{symbol}/{timeframe}/history?since=7d` (future) |
| Aggregation | ClickHouse (via gateway) | `GET /evidence/candle/{source}/{symbol}/{timeframe}/daily` (future, via materialized view) |

**Boundary rule:** Existing endpoints NEVER change behavior based on ClickHouse availability. Historical endpoints are NEW endpoints that return 503 if ClickHouse is unavailable.

---

## 8. Cold-Start Bootstrap Architecture (Future — Out of S143 Scope)

For completeness, the intended bootstrap path:

1. Derive starts and detects RSI needs warm-up candles
2. Derive queries ClickHouse for last N candles per symbol/timeframe
3. If ClickHouse is unavailable or returns empty: fall back to current behavior (wait for live candles)
4. If candles returned: seed RSI calculator, skip warm-up wait

**Invariant:** Bootstrap is opportunistic, never blocking. Derive MUST start successfully without ClickHouse.

---

## 9. Implementation Sequencing

This architecture supports the following implementation order (each phase is an independent stage):

| Phase | Scope | Depends On | Delivers |
|-------|-------|------------|----------|
| **Phase 1** | `cmd/migrate` + `_migrations` table | ClickHouse container (existing) | Migration infrastructure |
| **Phase 2** | Core table DDL (9 tables) | Phase 1 | Schema catalog |
| **Phase 3** | `cmd/writer` service | Phase 1 + Phase 2 | Event persistence |
| **Phase 4** | Gateway historical endpoints | Phase 3 | Query surface |
| **Phase 5** | Derive cold-start bootstrap | Phase 3 | Faster recovery |

Phases 4 and 5 are independent of each other and can be parallelized or reordered.

---

## 10. Invariants

These invariants MUST hold at every phase of implementation:

| ID | Invariant | Verification |
|----|-----------|-------------|
| **INV-01** | Pipeline functions without ClickHouse | Remove CH + writer from compose; all smoke tests pass |
| **INV-02** | No service except writer depends on ClickHouse | `docker-compose.yaml` dependency graph inspection |
| **INV-03** | Writer never publishes to NATS | Code review: no `js.Publish` calls in writer |
| **INV-04** | Writer uses own durable consumer names | Consumer names prefixed `writer-`, never `store-` |
| **INV-05** | Schema follows events | Each DDL field maps to a Go struct field |
| **INV-06** | All schema changes go through migrations | No ad-hoc DDL; `_migrations` table tracks all changes |
| **INV-07** | Existing endpoints unchanged | No behavioral changes to current gateway routes |
| **INV-08** | Writer tolerates ClickHouse downtime | Buffer + drop strategy; no blocking, no Nak |

---

## 11. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Event schema changes after DDL is written | Medium | Migration needed | Schema versioning convention (PC-05) |
| Writer buffer grows unbounded during CH outage | Low | OOM | `max_pending` cap with drop policy |
| ClickHouse disk fills silently | Medium | Inserts fail | TTL enforcement + monitoring (future) |
| Dual consumer position drift | Low | Duplicate/missed events | Each consumer has independent position tracking |
| Scope creep into analytics features | High | Delayed infrastructure | Guard rails in stage definition |

---

## 12. Out of Scope

The following are explicitly **not part of this architecture definition**:

- Materialized views and pre-aggregations
- ClickHouse user management or RBAC
- Multi-environment deployment (dev/staging/prod)
- ClickHouse clustering or replication
- Grafana or dashboard integration
- Real-time alerting from ClickHouse queries
- Event schema versioning mechanism (deferred to dedicated design)
- Compression tuning or performance optimization
- Backup and disaster recovery for ClickHouse volumes
