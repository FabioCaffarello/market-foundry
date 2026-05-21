# Analytical Boundary Hardening: Writer, Reader, Gateway

## Purpose

This document records the boundary hardening applied in S158 to sharpen separation between the three analytical layer components: **writer** (write path), **reader** (read path), and **gateway** (HTTP surface). Each component has a distinct responsibility domain, and boundaries between them must remain explicit and enforceable.

## Component Responsibilities After Hardening

### Writer (`cmd/writer/`)

**Responsibility:** Consume canonical domain events from NATS JetStream and project them as append-only rows into ClickHouse analytical tables.

**Owns:**
- NATS consumer lifecycle (durable, independent consumer names with `writer-` prefix)
- Domain event → ClickHouse row mapping (`mappers.go`)
- Batch buffering, flush triggers, overflow eviction, and retry logic (`inserter.go`)
- Pipeline supervision and recovery with exponential backoff (`supervisor.go`)
- Per-pipeline health tracking (10+ counters per family)
- Pipeline family catalog and enablement checks (`pipeline.go`)
- **NEW in S158:** Startup validation of both ClickHouse config and pipeline family config

**Does NOT own:**
- Query construction or row scanning for reads
- HTTP endpoint registration or request handling
- Schema evolution or DDL execution (belongs to `cmd/migrate/`)

### Reader (`internal/adapters/clickhouse/`)

**Responsibility:** Query ClickHouse analytical tables and translate storage rows back into domain types.

**Owns:**
- ClickHouse connection management (`client.go`)
- Batch INSERT protocol for write path (`client.go`)
- Query execution and row iteration (`reader.go`)
- **MOVED in S158:** Candle query construction, parameterized SELECT building, and row scanning → domain type mapping (`candle_reader.go`)
- **MOVED in S158:** Float formatting for storage↔domain precision consistency (`candle_reader.go`)

**Does NOT own:**
- HTTP request parsing, validation, or response formatting
- Use case orchestration or business rule enforcement
- NATS consumption or event decoding

### Gateway (`cmd/gateway/`)

**Responsibility:** Expose the analytical HTTP query surface and compose adapter + use case dependencies.

**Owns:**
- ClickHouse client lifecycle (optional, deferred close)
- Composition of `CandleReader` adapter → `GetCandleHistoryUseCase` → HTTP handler
- Readiness decision: ClickHouse NOT in readiness checks (R-02 compliance)
- **SIMPLIFIED in S158:** The composition root now delegates reader creation to the adapter layer via a single-line factory function

**Does NOT own:**
- Query construction or row mapping (delegated to adapter)
- Batch writing, consumer management, or pipeline supervision
- Domain validation rules (delegated to use case layer)

## Boundary Changes Applied in S158

### 1. Reader Extraction from Gateway to Adapter Layer

**Before:** `analyticalCandleReader` lived in `cmd/gateway/analytical_reader.go` as an unexported type in the `main` package. This placed storage↔domain translation logic inside the composition root, blurring the boundary between gateway (composition) and adapter (translation).

**After:** The reader implementation lives in `internal/adapters/clickhouse/candle_reader.go` as the exported `CandleReader` type. The gateway's `analytical_reader.go` is now a single-line factory that delegates to the adapter.

**Why this matters:**
- The adapter is the canonical location for storage↔domain translation
- The reader is now testable in isolation (no `main` package constraint)
- The gateway composition root only does composition, not translation
- The same reader adapter can be used by other consumers (e.g., future CLI tools) without duplicating code

### 2. Writer Config Validation at Startup

**Before:** The writer validated ClickHouse connectivity (addr not empty) but did not validate the `ClickHouseConfig` structural parameters (batch_size, flush_interval, etc.) or pipeline family names before attempting to spawn pipelines.

**After:** `cmd/writer/run.go` now calls `config.ClickHouse.Validate()` and `config.Pipeline.ValidatePipeline()` before opening the ClickHouse connection. Invalid config causes a clean shutdown with a descriptive error, not a runtime panic or silent misconfiguration.

### 3. Compile-Time Interface Contract Verification

**After:** `cmd/gateway/analytical_reader_test.go` contains a compile-time assertion that `clickhouse.CandleReader` satisfies `analyticalclient.CandleReader`. This prevents silent interface drift between the adapter implementation and the application-layer contract.

## Boundary Invariants (Unchanged)

These invariants from prior stages remain enforced:

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

## Data Flow After Hardening

### Write Path
```
NATS JetStream
  → cmd/writer/consumer.go       (NATS consumer actor)
  → cmd/writer/mappers.go        (domain event → row tuple)
  → cmd/writer/inserter.go       (buffer + batch flush)
  → internal/adapters/clickhouse  (InsertBatch)
  → ClickHouse table
```

### Read Path
```
HTTP request
  → internal/interfaces/http/handlers/analytical.go  (parse params)
  → internal/application/analyticalclient             (validate + execute)
  → internal/adapters/clickhouse/candle_reader.go     (query + scan + map)
  → internal/adapters/clickhouse/reader.go            (raw query execution)
  → ClickHouse table
```

### Composition (Gateway)
```
cmd/gateway/run.go
  → buildAnalyticalClient()       (optional ClickHouse connection)
  → newAnalyticalCandleReader()   (delegates to adapter CandleReader)
  → NewGetCandleHistoryUseCase()  (application layer)
  → routes.AnalyticalFamilyDeps   (route registration)
```

## Limits Maintained

- No new functionality added — only boundary realignment
- No schema evolution or DDL changes
- No new analytical families or query types
- ClickHouse optionality fully preserved
- Write path mappers remain in `cmd/writer/` (consistent with composition-level translation for the write side)
