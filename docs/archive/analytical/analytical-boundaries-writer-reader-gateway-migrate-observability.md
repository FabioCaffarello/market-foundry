# Analytical Boundaries: Writer, Reader, Gateway, Migrate, Observability

> Canonical reference for the responsibility boundaries between each analytical component, their interaction surfaces, and isolation guarantees.

## 1. Component Boundary Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        ANALYTICAL LAYER                                 │
│                                                                         │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────────┐  │
│  │  cmd/migrate  │    │  cmd/writer   │    │      cmd/gateway          │  │
│  │               │    │               │    │                          │  │
│  │ Schema DDL    │    │ NATS→CH       │    │  Operational   Analytical │  │
│  │ Checksums     │    │ Batch INSERT  │    │  (NATS KV)    (CH Query) │  │
│  │ Drift detect  │    │ Actor model   │    │                          │  │
│  └──────┬───────┘    └──────┬───────┘    └─────────┬────────┬───────┘  │
│         │                   │                      │        │          │
│         │ DDL               │ InsertBatch          │ NATS   │ Query    │
│         ▼                   ▼                      ▼        ▼          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │               internal/adapters/clickhouse                      │   │
│  │  client.go (Open, Ping, Close, InsertBatch)                    │   │
│  │  reader.go (Query, Rows)                                       │   │
│  └─────────────────────────────┬───────────────────────────────────┘   │
│                                │                                       │
│                                ▼                                       │
│                        [ ClickHouse ]                                  │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                     CROSS-CUTTING                                       │
│                                                                         │
│  internal/shared/healthz     internal/shared/settings                   │
│  (Tracker, HealthServer)     (AppConfig, validation, family catalog)    │
│                                                                         │
│  scripts/diag-check.sh       scripts/utils/lib.sh                      │
│  (Diagnostic snapshot)       (Port map, service constants)              │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Boundary Definitions

### 2.1 Migrate ↔ Writer

| Property | Value |
|----------|-------|
| **Interaction surface** | ClickHouse DDL (tables exist before writer starts) |
| **Runtime coupling** | Zero — migrate runs before writer; no shared process |
| **Data coupling** | Writer assumes exact DDL column order from migrations |
| **Coordination** | Deploy-time ordering (migrate up → writer start) |
| **Failure isolation** | Migrate failure blocks deployment; writer failure is lateral |

**Boundary contract:** Migrations define the schema. Writer trusts the schema exists. If schema is missing or changed, writer's INSERT fails at runtime.

### 2.2 Migrate ↔ Reader

| Property | Value |
|----------|-------|
| **Interaction surface** | ClickHouse DDL (tables exist before reader queries) |
| **Runtime coupling** | Zero — migrate runs independently |
| **Data coupling** | Reader hardcodes SELECT column list matching DDL |
| **Coordination** | Schema must be applied before analytical endpoints are meaningful |
| **Failure isolation** | Missing schema → reader Query returns error → 503 to client |

**Boundary contract:** Identical to migrate↔writer. Reader trusts DDL. Schema drift causes runtime scan errors.

### 2.3 Writer ↔ Reader

| Property | Value |
|----------|-------|
| **Interaction surface** | ClickHouse tables (writer inserts rows; reader queries rows) |
| **Runtime coupling** | Zero — no direct communication; no shared process |
| **Data coupling** | Both depend on same ClickHouse schema (column names, types, order) |
| **Temporal coupling** | Reader sees data only after writer flushes; eventual consistency |
| **Failure isolation** | Writer failure → no new data; reader continues serving stale data |

**Boundary contract:** Writer and reader are completely independent. Writer's failure or absence does not affect reader availability. Reader simply returns fewer results.

### 2.4 Writer ↔ Gateway

| Property | Value |
|----------|-------|
| **Interaction surface** | None |
| **Runtime coupling** | Zero — separate processes; separate Docker containers |
| **Data coupling** | None — writer does not know about HTTP endpoints |
| **Failure isolation** | Complete — writer crash cannot affect gateway |

**Boundary contract:** Writer and gateway have no interaction. Gateway's analytical endpoints read from ClickHouse; writer writes to ClickHouse. The database is the only shared resource, and they use independent connections.

### 2.5 Writer ↔ NATS Registries

| Property | Value |
|----------|-------|
| **Interaction surface** | `internal/adapters/nats/*_registry.go` consumer specs |
| **Runtime coupling** | Writer instantiates registry consumers with `writer-*` durable names |
| **Data coupling** | Writer depends on event type schemas from `internal/domain/` |
| **Coordination** | Consumer specs define stream, subject, durable name, delivery policy |
| **Failure isolation** | NATS unavailable → writer consumer cannot start → supervisor retries |

**Boundary contract:** Writer uses existing NATS adapter consumers with independent durable names (`writer-candle`, `writer-signal-rsi`, etc.). Writer never creates new NATS streams — it only consumes from existing ones created by the operational pipeline.

### 2.6 Reader ↔ Gateway

| Property | Value |
|----------|-------|
| **Interaction surface** | Go interface `CandleReader` defined in `internal/application/analyticalclient/` |
| **Runtime coupling** | Reader implementation is instantiated by gateway's `compose.go` |
| **Data coupling** | Request/response DTOs in `analyticalclient/contracts.go` |
| **Conditional wiring** | Reader is nil when ClickHouse not configured → handler returns 503 |
| **Failure isolation** | Reader error → handler returns 503; gateway readiness unaffected (R-02) |

**Boundary contract:** Gateway owns the HTTP layer. Reader implements the `CandleReader` interface. Gateway never constructs SQL or touches ClickHouse directly — only through the reader abstraction.

### 2.7 Observability ↔ All Components

| Property | Value |
|----------|-------|
| **Interaction surface** | `internal/shared/healthz.Tracker` counters + health endpoints |
| **Runtime coupling** | Every service embeds a HealthServer; trackers registered at startup |
| **Data coupling** | Counter names and phase classification logic |
| **Current coverage** | Writer: 10+ counters per family; Reader: 0 counters (gap) |

**Boundary contract:** Observability is cross-cutting but non-intrusive. Components register trackers and increment counters. HealthServer aggregates into phase classification. Diagnostic scripts poll endpoints without coupling to internals.

---

## 3. Isolation Guarantees

### 3.1 Failure Isolation Matrix

| Failing Component | Writer Impact | Reader Impact | Gateway Operational | Gateway Analytical | Migrate |
|-------------------|---------------|---------------|--------------------|--------------------|---------|
| **Writer crash** | Pipeline stops; NATS redelivers on restart | None | None | None (stale data) | None |
| **ClickHouse down** | Buffer overflow → drop | Query error → 503 | None | 503 responses | Cannot run |
| **NATS down** | Consumer cannot connect | None | Operational queries fail | None | None |
| **Reader error** | None | 503 response | None | Affected endpoint only | None |
| **Gateway crash** | None | None | All HTTP down | All HTTP down | None |
| **Migrate failure** | Tables may be missing | Tables may be missing | None | Queries fail | Stopped |

### 3.2 Lifecycle Independence

| Component | Can be absent? | Can restart independently? | Affects readiness of others? |
|-----------|---------------|---------------------------|------------------------------|
| **cmd/migrate** | Yes (deploy-time only) | Yes | No |
| **cmd/writer** | Yes (lateral, optional) | Yes | No |
| **Reader (in gateway)** | Yes (ClickHouse optional) | Restarts with gateway | No (R-02 compliance) |
| **ClickHouse** | Yes (analytical disabled) | Yes | Writer + reader degrade gracefully |

---

## 4. Data Flow Boundaries

### 4.1 Write Path Flow

```
NATS JetStream Stream (e.g., EVIDENCE_EVENTS)
    │
    ▼ (durable consumer: writer-candle)
cmd/writer/consumer.go  [Actor: ConsumerActor]
    │ decode event → map to row
    ▼ (actor message: insertRowMsg)
cmd/writer/inserter.go  [Actor: InserterActor]
    │ buffer rows → batch flush
    ▼ (native protocol: InsertBatch)
internal/adapters/clickhouse/client.go
    │
    ▼
ClickHouse (evidence_candles table)
```

**Boundaries crossed:** NATS → Consumer Actor → Inserter Actor → ClickHouse Adapter → ClickHouse

### 4.2 Read Path Flow

```
HTTP GET /analytical/evidence/candles?source=...&symbol=...&timeframe=...
    │
    ▼
internal/interfaces/http/handlers/analytical.go  [Parse + validate HTTP params]
    │
    ▼
internal/application/analyticalclient/get_candle_history.go  [Validate + coordinate]
    │
    ▼ (CandleReader interface)
cmd/gateway/analytical_reader.go  [Build SQL + scan rows]
    │
    ▼ (Query method)
internal/adapters/clickhouse/reader.go
    │
    ▼
ClickHouse (evidence_candles table)
```

**Boundaries crossed:** HTTP Handler → Use Case → Reader Implementation → ClickHouse Adapter → ClickHouse

### 4.3 Schema Evolution Flow

```
deploy/migrations/NNN_*.sql
    │
    ▼ (ReadCatalog)
internal/migrate/catalog.go  [Discover + validate + sort]
    │
    ▼ (Up)
internal/migrate/runner.go  [Execute SQL + record in _migrations]
    │
    ▼ (native protocol)
cmd/migrate/main.go  [ClickHouse connection bootstrap]
    │
    ▼
ClickHouse (DDL applied, _migrations updated)
```

**Boundaries crossed:** SQL files → Catalog → Runner → ClickHouse

---

## 5. Configuration Boundaries

### 5.1 Who Reads What

| Config Section | configctl | ingest | derive | store | execute | writer | gateway |
|---------------|-----------|--------|--------|-------|---------|--------|---------|
| `log` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `http` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `nats` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `pipeline` | No | No | Yes | Yes | Yes | Yes | No |
| `clickhouse` | No | No | No | No | No | **Yes** | **Optional** |
| `venue` | No | No | No | No | Yes | No | No |

**Observation:** Only writer and gateway touch ClickHouse config. Gateway treats it as optional (graceful degradation). Writer treats it as required (fail-fast on empty addr).

### 5.2 Pipeline Family Validation

| Validation | Where | When |
|-----------|-------|------|
| Family name is known | `internal/shared/settings/schema.go` | Config load (all services) |
| Upstream dependencies satisfied | `internal/shared/settings/schema.go` | Config load (all services) |
| ClickHouse families match pipeline | **Not implemented** | **Gap: should be writer startup** |
| Batch parameters in range | **Not implemented** | **Gap: should be writer startup** |

---

## 6. Expansion Gate Boundaries

### 6.1 Adding a New Analytical Family (e.g., Wave B: signals reader)

**Checkpoints required:**

1. **Migration:** New SQL file in `deploy/migrations/` (already exists: `002_create_signals.sql`).
2. **Writer mapper:** New function in `cmd/writer/mappers.go` (already exists: `mapSignalRow()`).
3. **Writer pipeline:** Entry in `declareWriterPipelines()` (already exists for rsi).
4. **Reader implementation:** New reader in adapter layer (does not exist — Wave B scope).
5. **Use case:** New use case in `internal/application/analyticalclient/` (does not exist).
6. **Handler:** New handler method in `internal/interfaces/http/handlers/analytical.go` (does not exist).
7. **Route:** New route in `internal/interfaces/http/routes/analytical.go` (does not exist).
8. **Config:** Family already enabled in `deploy/configs/writer.jsonc` pipeline section.
9. **Test:** Integration test validates full flow (does not exist).
10. **Observability:** Reader tracker counters for new family (does not exist).

**Gate rule:** Steps 1–3 (write path) are already complete for all 6 families. Steps 4–10 (read path + validation) must be added per family during Wave B.
