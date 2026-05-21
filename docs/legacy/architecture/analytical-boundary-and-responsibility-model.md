# Analytical Boundary and Responsibility Model

> Consolidated from 6 source documents (archived in docs/archive/analytical/).
> Sources: analytical-boundaries-writer-reader-gateway-migrate-observability.md, analytical-boundary-hardening-writer-reader-gateway.md, analytical-contracts-and-adapter-boundaries.md, analytical-responsibility-anti-patterns-and-non-goals.md, analytical-responsibility-map-writer-reader-pipeline-observability.md, analytical-responsibility-review-and-restructuring-plan.md

---

## 1. Component Boundary Diagram

```
+---------------------------------------------------------------------------+
|                        ANALYTICAL LAYER                                    |
|                                                                            |
|  +--------------+    +--------------+    +--------------------------+     |
|  |  cmd/migrate  |    |  cmd/writer   |    |      cmd/gateway          |     |
|  |               |    |               |    |                          |     |
|  | Schema DDL    |    | NATS->CH      |    |  Operational   Analytical |     |
|  | Checksums     |    | Batch INSERT  |    |  (NATS KV)    (CH Query) |     |
|  | Drift detect  |    | Actor model   |    |                          |     |
|  +------+-------+    +------+-------+    +---------+--------+-------+     |
|         |                   |                      |        |             |
|         | DDL               | InsertBatch          | NATS   | Query       |
|         v                   v                      v        v             |
|  +-------------------------------------------------------------------+   |
|  |               internal/adapters/clickhouse                         |   |
|  |  client.go (Open, Ping, Close, InsertBatch)                       |   |
|  |  reader.go (Query, Rows)                                          |   |
|  +-------------------------------------------------------------------+   |
|                                |                                          |
|                                v                                          |
|                        [ ClickHouse ]                                     |
+---------------------------------------------------------------------------+

+---------------------------------------------------------------------------+
|                     CROSS-CUTTING                                          |
|                                                                            |
|  internal/shared/healthz     internal/shared/settings                      |
|  (Tracker, HealthServer)     (AppConfig, validation, family catalog)       |
|                                                                            |
|  scripts/diag-check.sh       scripts/utils/lib.sh                         |
|  (Diagnostic snapshot)       (Port map, service constants)                 |
+---------------------------------------------------------------------------+
```

---

## 2. Component Responsibilities

### 2.1 Writer (`cmd/writer/`)

**Responsibility:** Consume canonical domain events from NATS JetStream and project them as append-only rows into ClickHouse analytical tables.

**Owns:**
- NATS consumer lifecycle (durable, independent consumer names with `writer-` prefix)
- Domain event -> ClickHouse row mapping (`mappers.go`)
- Batch buffering, flush triggers, overflow eviction, and retry logic (`inserter.go`)
- Pipeline supervision and recovery with exponential backoff (`supervisor.go`)
- Per-pipeline health tracking (10+ counters per family)
- Pipeline family catalog and enablement checks (`pipeline.go`)
- Startup validation of both ClickHouse config and pipeline family config

**Does NOT own:**
- Query construction or row scanning for reads
- HTTP endpoint registration or request handling
- Schema evolution or DDL execution (belongs to `cmd/migrate/`)

### 2.2 Reader (`internal/adapters/clickhouse/`)

**Responsibility:** Query ClickHouse analytical tables and translate storage rows back into domain types.

**Owns:**
- ClickHouse connection management (`client.go`)
- Batch INSERT protocol for write path (`client.go`)
- Query execution and row iteration (`reader.go`)
- Candle query construction, parameterized SELECT building, and row scanning -> domain type mapping (`candle_reader.go`)
- Float formatting for storage<->domain precision consistency

**Does NOT own:**
- HTTP request parsing, validation, or response formatting
- Use case orchestration or business rule enforcement
- NATS consumption or event decoding

### 2.3 Gateway (`cmd/gateway/`)

**Responsibility:** Expose the analytical HTTP query surface and compose adapter + use case dependencies.

**Owns:**
- ClickHouse client lifecycle (optional, deferred close)
- Composition of `CandleReader` adapter -> `GetCandleHistoryUseCase` -> HTTP handler
- Readiness decision: ClickHouse NOT in readiness checks (R-02 compliance)
- Composition root delegates reader creation to the adapter layer via factory function

**Does NOT own:**
- Query construction or row mapping (delegated to adapter)
- Batch writing, consumer management, or pipeline supervision
- Domain validation rules (delegated to use case layer)

### 2.4 Migrate (`cmd/migrate/`)

**Responsibility:** ClickHouse DDL management -- create tables, evolve schema, validate checksums.

**Boundary:** Standalone CLI utility; runs once and exits. Zero coupling to writer, reader, gateway, NATS, or config system. Forward-only, idempotent, checksum-validated.

### 2.5 Observability (`internal/shared/healthz/` + scripts)

**Responsibility:** Health endpoints, tracker counters, phase classification, diagnostic scripts.

Cross-cutting but non-intrusive. Components register trackers and increment counters. HealthServer aggregates into phase classification. Diagnostic scripts poll endpoints without coupling to internals.

---

## 3. Boundary Definitions

### Migrate <-> Writer

- **Interaction surface:** ClickHouse DDL (tables exist before writer starts)
- **Runtime coupling:** Zero -- migrate runs before writer; no shared process
- **Coordination:** Deploy-time ordering (migrate up -> writer start)
- **Contract:** Migrations define the schema. Writer trusts the schema exists. If schema is missing or changed, writer's INSERT fails at runtime.

### Migrate <-> Reader

Identical to migrate<->writer. Reader trusts DDL. Schema drift causes runtime scan errors. Missing schema -> reader Query returns error -> 503 to client.

### Writer <-> Reader

- **Interaction surface:** ClickHouse tables (writer inserts rows; reader queries rows)
- **Runtime coupling:** Zero -- no direct communication; no shared process
- **Temporal coupling:** Reader sees data only after writer flushes; eventual consistency
- **Contract:** Writer and reader are completely independent. Writer failure does not affect reader availability.

### Writer <-> Gateway

- **Interaction surface:** None
- **Runtime coupling:** Zero -- separate processes; separate Docker containers
- **Contract:** No interaction. The database is the only shared resource, and they use independent connections.

### Writer <-> NATS Registries

- Writer instantiates registry consumers with `writer-*` durable names
- Writer depends on event type schemas from `internal/domain/`
- Consumer specs define stream, subject, durable name, delivery policy
- Writer never creates new NATS streams -- it only consumes from existing ones

### Reader <-> Gateway

- **Interaction surface:** Go interface `CandleReader` defined in `internal/application/analyticalclient/`
- Reader implementation is instantiated by gateway's `compose.go`
- Reader is nil when ClickHouse not configured -> handler returns 503
- Gateway never constructs SQL or touches ClickHouse directly

---

## 4. Contract Inventory

### Application-Layer Contracts (`internal/application/analyticalclient/`)

| Contract | Type | Owner |
|----------|------|-------|
| `CandleHistoryQuery` | Request struct | analyticalclient |
| `CandleHistoryReply` | Response struct | analyticalclient |
| `CandleReader` | Interface | analyticalclient |

**Ownership rule:** The interface is defined in the application layer (consumer side), following Go's "accept interfaces, return structs" convention. The adapter implements this interface but does not import or reference it.

### Domain Contracts (consumed, not owned)

The analytical layer uses domain types as the shared vocabulary between write and read paths:
- `evidence.EvidenceCandle`, `evidence.CandleSampledEvent`
- `signal.SignalGeneratedEvent`, `decision.DecisionEvaluatedEvent`
- `strategy.StrategyResolvedEvent`, `risk.RiskAssessedEvent`
- `execution.PaperOrderSubmittedEvent`

**Rule:** Domain types are the lingua franca. Both write-path mappers and read-path readers translate between domain types and ClickHouse storage. Neither side references the other's translation code.

### Schema Contracts (implicit, DDL-defined)

| Table | DDL Source | Write Consumer | Read Consumer |
|-------|-----------|----------------|---------------|
| evidence_candles | 001_create_evidence_candles.sql | `mapCandleRow` | candle_reader.go |
| signals | 002_create_signals.sql | `mapSignalRow` | signal_reader.go |
| decisions | 003_create_decisions.sql | `mapDecisionRow` | decision_reader.go |
| strategies | 004_create_strategies.sql | `mapStrategyRow` | strategy_reader.go |
| risk_assessments | 005_create_risk_assessments.sql | `mapRiskRow` | risk_reader.go |
| executions | 006_create_executions.sql | `mapExecutionRow` | execution_reader.go |

**Schema coherence rule:** Any DDL change to column names, types, or ordering must be reflected in both the write mapper and the read adapter. Coherence is validated by integration tests and reviewer discipline.

---

## 5. Adapter Boundary Rules

| Rule | Description |
|------|-------------|
| AB-01 | Adapter imports domain, never application. The ClickHouse adapter may import `internal/domain/*` but must never import `internal/application/*` or `internal/interfaces/*`. |
| AB-02 | Application defines interfaces, adapter provides structs. `CandleReader` interface lives in application layer; `clickhouse.CandleReader` struct satisfies it via structural typing. |
| AB-03 | Gateway composes, adapter translates. Gateway composition root connects adapters to use cases but contains no translation logic. |
| AB-04 | Writer mappers stay in the binary. Write-path mappers live in `cmd/writer/` because they wire NATS event decoding to ClickHouse row tuples (composition-specific). |
| AB-05 | No cross-path imports. Write-path code must not import read-path code, and vice versa. |
| AB-06 | ClickHouse client is shared, readers are not. `clickhouse.Client` is used by both writer and reader, but `CandleReader` is read-path-only. |

---

## 6. Boundary Invariants

| Rule | Description |
|------|-------------|
| R-01 | No operational service depends on ClickHouse |
| R-02 | No readiness check references ClickHouse (except writer) |
| R-03 | No event path blocks on ClickHouse |
| R-04 | Writer uses independent consumer names (writer- prefix) |
| R-05 | Writer tolerates ClickHouse absence (buffers, drops on overflow) |
| R-06 | Smoke tests pass without ClickHouse and writer |
| R-07 | No conditional behavior in operational services based on ClickHouse |
| R-08 | Historical/analytical endpoints are additive (new routes only) |
| R-09 | Cold-start bootstrap is opportunistic |
| R-10 | Configuration does not require ClickHouse |

---

## 7. Failure Isolation Matrix

| Failing Component | Writer Impact | Reader Impact | Gateway Operational | Gateway Analytical | Migrate |
|-------------------|---------------|---------------|--------------------|--------------------|---------|
| Writer crash | Pipeline stops; NATS redelivers on restart | None | None | None (stale data) | None |
| ClickHouse down | Buffer overflow -> drop | Query error -> 503 | None | 503 responses | Cannot run |
| NATS down | Consumer cannot connect | None | Operational queries fail | None | None |
| Reader error | None | 503 response | None | Affected endpoint only | None |
| Gateway crash | None | None | All HTTP down | All HTTP down | None |
| Migrate failure | Tables may be missing | Tables may be missing | None | Queries fail | Stopped |

### Lifecycle Independence

| Component | Can be absent? | Can restart independently? | Affects readiness of others? |
|-----------|---------------|---------------------------|------------------------------|
| cmd/migrate | Yes (deploy-time only) | Yes | No |
| cmd/writer | Yes (lateral, optional) | Yes | No |
| Reader (in gateway) | Yes (ClickHouse optional) | Restarts with gateway | No (R-02 compliance) |
| ClickHouse | Yes (analytical disabled) | Yes | Writer + reader degrade gracefully |

---

## 8. Data Flow Paths

### Write Path

```
NATS JetStream
  -> cmd/writer/consumer.go       (NATS consumer actor)
  -> cmd/writer/mappers.go        (domain event -> row tuple)
  -> cmd/writer/inserter.go       (buffer + batch flush)
  -> internal/adapters/clickhouse  (InsertBatch)
  -> ClickHouse table
```

### Read Path

```
HTTP request
  -> internal/interfaces/http/handlers/analytical.go  (parse params)
  -> internal/application/analyticalclient             (validate + execute)
  -> internal/adapters/clickhouse/candle_reader.go     (query + scan + map)
  -> internal/adapters/clickhouse/reader.go            (raw query execution)
  -> ClickHouse table
```

### Composition (Gateway)

```
cmd/gateway/run.go
  -> buildAnalyticalClient()       (optional ClickHouse connection)
  -> newAnalyticalCandleReader()   (delegates to adapter CandleReader)
  -> NewGetCandleHistoryUseCase()  (application layer)
  -> routes.AnalyticalFamilyDeps   (route registration)
```

### Schema Evolution

```
deploy/migrations/NNN_*.sql -> internal/migrate/catalog.go -> runner.go -> ClickHouse
```

---

## 9. Anti-Patterns to Avoid

| ID | Anti-Pattern | Correction |
|----|-------------|-----------|
| AP-01 | Reader implementation in composition root | Move to `internal/adapters/clickhouse/` (done in S158) |
| AP-02 | Distributed schema knowledge without contract | Document 3-point coordination rule (DDL + mapper + reader) |
| AP-03 | Asymmetric observability (writer instrumented, reader not) | Add structured logging + counters to read path (done in S160) |
| AP-04 | Config validation gap for writer | Validate family names and batch params at startup (done in S161) |
| AP-05 | No automated integration test for data path | Proof script validates NATS -> writer -> ClickHouse -> reader -> HTTP |

---

## 10. Design Patterns to Protect

| ID | Pattern | Description |
|----|---------|-------------|
| DP-01 | Lateral optionality | Writer and reader are optional; removal has zero operational impact |
| DP-02 | Independent durable consumers | Writer uses `writer-*` prefixed NATS durables, isolated from `store-*` |
| DP-03 | Graceful degradation | Gateway returns 503 for analytical endpoints when ClickHouse unavailable, without affecting operational readiness |
| DP-04 | Composition root purity | `cmd/*/run.go` files compose dependencies; no business logic |
| DP-05 | Mechanical transformation | Writer mappers do no filtering, aggregation, or enrichment |
| DP-06 | Forward-only migration | No rollback migrations; changes are additive |
| DP-07 | Fail-fast configuration | Settings validation catches misconfiguration before listeners start |
| DP-08 | Actor-based lifecycle | Hollywood actors for consumer/inserter lifecycle with supervisor-managed restart |

---

## 11. Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-01 | Materialized views | No query patterns justify aggregation at current scale |
| NG-02 | Dead-letter queue | Drop-after-retry is appropriate for analytical projection tolerances |
| NG-03 | Auto-recovery from degraded | Operator restart is acceptable at single-operator scale |
| NG-04 | Distributed tracing | Structured logs + healthz counters are sufficient |
| NG-05 | Shared column constants | Go's type system makes this awkward for ClickHouse positional INSERTs |
| NG-06 | Query-time deduplication | Duplicates tolerable; `DISTINCT`/`argMax` at query time if needed |
| NG-07 | Dynamic family registration | Static JSONC config is sufficient |
| NG-08 | Backfill mechanism | Replay from NATS is future capability |

---

## 12. Expansion Protocol

When adding a new analytical reader family:

1. Create `{family}_reader.go` in `internal/adapters/clickhouse/` with query builder + row scanner
2. Define `{Family}Reader` interface in `internal/application/analyticalclient/`
3. Create `Get{Family}HistoryUseCase` in `internal/application/analyticalclient/`
4. Add handler method in `internal/interfaces/http/handlers/analytical.go`
5. Register route in `internal/interfaces/http/routes/analytical.go`
6. Wire in `cmd/gateway/compose.go` via `AnalyticalFamilyDeps`
7. Add compile-time interface assertion in `cmd/gateway/analytical_reader_test.go`

No step touches the writer. No step modifies existing read-path code for other families.

### Configuration Boundaries

| Config Section | writer | gateway | Other services |
|---------------|--------|---------|----------------|
| `clickhouse` | **Required** | **Optional** | Not used |
| `pipeline` | Yes | No | Yes (derive/store/execute) |
| `nats` | Yes | Yes | Yes |
