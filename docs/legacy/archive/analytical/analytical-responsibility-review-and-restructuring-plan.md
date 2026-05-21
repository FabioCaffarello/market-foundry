# Analytical Responsibility Review and Restructuring Plan

> Stage S157 — Formal review of the analytical architecture by responsibilities, with minimal restructuring plan aligned to Wave B preconditions.

## 1. Executive Summary

The analytical layer of market-foundry comprises five distinct responsibility domains: **schema evolution** (cmd/migrate), **write path** (cmd/writer), **read path** (gateway analytical reader), **query boundaries** (gateway routes/handlers), and **observability** (healthz trackers + diagnostic scripts). This review finds that the macro-architecture is sound — failure isolation, lateral optionality, and independent lifecycles are well-established. However, three boundary ambiguities and two structural gaps require minimal, targeted adjustments before Wave B expansion can proceed safely.

**Key findings:**
1. The read path implementation (`analytical_reader.go`) is structurally misplaced inside `cmd/gateway/`, blurring the boundary between gateway orchestration and analytical data access.
2. ClickHouse schema knowledge is distributed across three locations (DDL in migrations, column order in writer mappers, SELECT columns in reader) with no shared contract or compile-time validation.
3. The observability boundary is asymmetric: writer has 10+ counters and per-family tracking; reader has zero instrumentation.
4. Writer config validation does not verify required ClickHouse families at startup, violating fail-fast principles.
5. No automated integration test validates the full analytical data path (NATS → writer → ClickHouse → reader → HTTP).

**Proposed restructuring:** Five targeted adjustments (no redesign), directly aligned to S156 preconditions and boundary clarity.

---

## 2. Responsibility Map — Current State

### 2.1 Schema Evolution (Migrations)

| Aspect | Detail |
|--------|--------|
| **Owner** | `cmd/migrate` + `internal/migrate` + `deploy/migrations/*.sql` |
| **Responsibility** | ClickHouse DDL management — create tables, evolve schema, validate checksums |
| **Boundary** | Standalone CLI utility; runs once and exits; no runtime coupling to any service |
| **Dependencies** | ClickHouse native protocol only |
| **Coupling** | Zero coupling to writer, reader, gateway, NATS, or config system |
| **Assessment** | **Clean.** Forward-only, idempotent, checksum-validated. No boundary issues. |

### 2.2 Analytical Write Path (Writer)

| Aspect | Detail |
|--------|--------|
| **Owner** | `cmd/writer/` (supervisor, consumer, inserter, pipeline, mappers, run.go) |
| **Responsibility** | NATS event consumption → mechanical transformation → ClickHouse batch INSERT |
| **Boundary** | Lateral, append-only service; independent NATS cursors; failure-isolated from operational pipeline |
| **Dependencies** | NATS JetStream (consume), ClickHouse (write), Hollywood actors, shared healthz/settings |
| **Coupling** | Uses `internal/adapters/nats` consumer specs (writer-* durable names); uses `internal/adapters/clickhouse` InsertBatch; uses domain event types for deserialization |
| **Assessment** | **Clean.** Actor model provides robust lifecycle. Pipeline catalog is declarative. Mappers are mechanical. |

### 2.3 Analytical Read Path (Reader)

| Aspect | Detail |
|--------|--------|
| **Owner** | `cmd/gateway/analytical_reader.go` + `internal/application/analyticalclient/` |
| **Responsibility** | ClickHouse SELECT queries → domain type conversion → use case validation |
| **Boundary** | Optional; returns 503 when ClickHouse unavailable; not part of gateway readiness |
| **Dependencies** | `internal/adapters/clickhouse` Query method; `internal/domain/evidence` types |
| **Coupling** | Reader implementation (`analyticalCandleReader`) lives in `cmd/gateway/`, sharing the package with operational gateway wiring |
| **Assessment** | **Boundary blur.** The reader struct belongs to the gateway cmd package but implements an analytical concern. |

### 2.4 Gateway / Query Boundaries

| Aspect | Detail |
|--------|--------|
| **Owner** | `cmd/gateway/compose.go` + `internal/interfaces/http/routes/` + `internal/interfaces/http/handlers/` |
| **Responsibility** | HTTP request routing, conditional analytical endpoint registration, graceful degradation |
| **Boundary** | Analytical routes registered only when ClickHouse configured; R-02 compliance (readiness independent) |
| **Dependencies** | Uses `analyticalclient.GetCandleHistoryUseCase`; conditionally creates ClickHouse client |
| **Coupling** | `compose.go` wires both operational (NATS-backed) and analytical (ClickHouse-backed) dependencies in one function |
| **Assessment** | **Mostly clean.** Conditional wiring is correct. Minor concern: `buildRouteDependencies()` mixes operational and analytical assembly. |

### 2.5 Observability / Operability

| Aspect | Detail |
|--------|--------|
| **Owner** | `internal/shared/healthz/` + `scripts/diag-check.sh` + `scripts/utils/lib.sh` |
| **Responsibility** | Health endpoints, tracker counters, phase classification, diagnostic scripts |
| **Boundary** | Cross-cutting; every runtime inherits healthz server; scripts scan all runtimes uniformly |
| **Dependencies** | None (pure shared infrastructure) |
| **Coupling** | Writer registers 10+ tracker counters per family; gateway reader registers **zero** counters |
| **Assessment** | **Asymmetric.** Write path fully instrumented; read path uninstrumented. This is the most critical gap. |

### 2.6 Shared ClickHouse Adapter

| Aspect | Detail |
|--------|--------|
| **Owner** | `internal/adapters/clickhouse/` (client.go, reader.go, go.mod) |
| **Responsibility** | Native protocol connection, InsertBatch (write), Query/Rows (read) |
| **Boundary** | Pure data adapter; no domain knowledge; used by both writer and gateway |
| **Dependencies** | `github.com/ClickHouse/clickhouse-go/v2` |
| **Coupling** | Correctly shared — provides low-level protocol abstraction without leaking concerns |
| **Assessment** | **Clean.** Minimal API surface. Adapter layer is appropriate for dual-use. |

---

## 3. Boundary Blurs and Structural Gaps

### 3.1 Reader Placement in Gateway Cmd (Boundary Blur)

**Finding:** `analytical_reader.go` defines `analyticalCandleReader` inside `cmd/gateway/`. This struct implements the `CandleReader` interface from `internal/application/analyticalclient/` and contains ClickHouse query logic (SQL construction, row scanning, float formatting).

**Problem:** The `cmd/gateway/` package should be a composition root — wiring dependencies, not implementing data access logic. Placing the reader implementation here means:
- Reader logic is untestable without gateway context (though tests exist for query builder).
- Future analytical readers (signals, decisions, etc.) would accumulate in the gateway cmd package.
- The boundary between "gateway orchestration" and "analytical data access" is blurred.

**Severity:** Medium. Functional correctness is unaffected, but expansion to Wave B (more reader families) would amplify the blur.

### 3.2 Distributed Schema Knowledge (Structural Gap)

**Finding:** ClickHouse schema knowledge exists in three disconnected locations:
1. **DDL** — `deploy/migrations/001_create_evidence_candles.sql` defines column names and types.
2. **Writer mappers** — `cmd/writer/mappers.go` maps event fields to column positions (positional `[]any` slices).
3. **Reader** — `cmd/gateway/analytical_reader.go` hardcodes SELECT column list and `rows.Scan()` targets.

**Problem:** A schema change (add column, reorder, rename) requires synchronized edits in three separate locations with no compile-time safety net. The only validation is runtime failure (INSERT error in writer or scan mismatch in reader).

**Severity:** Medium. Current schema is stable, but Wave B adds more table families, multiplying coordination points.

### 3.3 Asymmetric Observability (S156 Precondition)

**Finding:** Writer has comprehensive instrumentation:
- Per-family tracker counters: `events_received`, `events_flushed`, `buffer_depth`, `flush_duration_ms`, `events_overflowed`, `events_dropped`, `flush_failures`, `pipeline_restarts`, `pipeline_degraded`.
- Supervisor logs restart attempts with exponential backoff timings.
- `/statusz` and `/diagz` expose all counters.

Reader has zero instrumentation:
- No query timing.
- No error counting.
- No structured logging in the analytical handler or reader.
- No tracker registration in gateway for analytical endpoints.

**Severity:** High. Without read path observability, diagnosing ClickHouse query performance, timeout issues, or degraded analytical responses is impossible.

### 3.4 Missing Writer Config Validation (S156 Precondition)

**Finding:** Writer's `run.go` validates ClickHouse address is non-empty but does NOT validate:
- Whether configured pipeline families match known families in `settings.schema.go`.
- Whether ClickHouse tables exist for configured families.
- Whether batch parameters are within sane ranges.

**Severity:** Medium. Misconfiguration is caught only at runtime (first consumer message or INSERT failure), not at startup.

### 3.5 Missing Integration Test (S156 Precondition)

**Finding:** No automated test validates the full analytical data flow:
`NATS publish → writer consumer → writer inserter → ClickHouse INSERT → reader Query → HTTP response`

Individual unit tests exist (mappers, inserter, supervisor, use case, handler), but the composed flow has no automated coverage.

**Severity:** Medium. Manual validation via `tests/http/analytical.http` exists but is not CI-ready.

---

## 4. Restructuring Plan

### Adjustment 1: Extract Reader to Adapter Layer

**What:** Move `analyticalCandleReader` from `cmd/gateway/analytical_reader.go` to `internal/adapters/clickhouse/candle_reader.go`.

**Why:** Restores the cmd/gateway package to its composition-root role. Reader implementation joins the adapter layer where it belongs, alongside `client.go` and `reader.go`. The `CandleReader` interface in `analyticalclient/` already defines the contract — the adapter implements it.

**Scope:** File move + import path update in `cmd/gateway/compose.go`. No behavioral change.

**Alignment:** Prepares clean expansion point for Wave B readers (signals, decisions, etc.) without polluting the gateway cmd.

### Adjustment 2: Add Reader Instrumentation

**What:** Add structured logging and tracker counters to the analytical read path:
- Log query execution with `slog.Info` (source, symbol, timeframe, row_count, duration_ms).
- Log errors with `slog.Error` (query failure details).
- Register a `healthz.Tracker` for analytical queries in gateway's health server.
- Track: `analytical_queries_total`, `analytical_query_errors`, `analytical_query_duration_ms`.

**Why:** Closes the S156 precondition for reader instrumentation. Makes the observability boundary symmetric.

**Scope:** Add logging to `analyticalCandleReader.QueryCandleHistory()` and tracker registration in `cmd/gateway/compose.go`.

### Adjustment 3: Writer Config Validation at Startup

**What:** Add validation in `cmd/writer/run.go` Phase 2 that:
- Verifies all configured pipeline families are in the known families catalog from `settings.schema.go`.
- Verifies batch parameters are within sane ranges (batch_size > 0, flush_interval > 0, max_pending >= batch_size).
- Fails fast with clear error message on misconfiguration.

**Why:** Closes the S156 precondition for writer config validation. Prevents silent misconfiguration from reaching runtime.

**Scope:** Add validation function in `cmd/writer/run.go` or `cmd/writer/pipeline.go`. No structural change.

### Adjustment 4: Document Schema Contract

**What:** Create a lightweight schema contract document (`docs/architecture/analytical-schema-contract-and-coordination-rules.md`) that:
- Lists each ClickHouse table, its owning migration, its writer mapper function, and its reader query.
- Defines the coordination rule: schema changes require synchronized updates to DDL, mapper, and reader.
- Establishes that `migrate validate` + writer/reader unit tests are the validation gate.

**Why:** Makes the distributed schema knowledge explicit and auditable. Does not add code complexity — governance through documentation.

**Scope:** Documentation only. No code change.

### Adjustment 5: Integration Test Skeleton

**What:** Create a test harness skeleton (`tests/integration/analytical_flow_test.go` or equivalent script) that validates the full data path when ClickHouse is available. Initially:
- Publish a known candle event to NATS.
- Wait for writer to flush (poll `/statusz` for `events_flushed` increment).
- Query `/analytical/evidence/candles` for the known event.
- Assert response contains expected data.

**Why:** Closes the S156 precondition for integration testing. Skeleton establishes the pattern; full coverage grows with Wave B families.

**Scope:** New test file. Can be gated behind `INTEGRATION=true` build tag or run only in `live-pipeline-activate.sh`.

---

## 5. Priority and Sequencing

| # | Adjustment | S156 Precondition | Boundary Clarity | Effort | Priority |
|---|-----------|-------------------|-----------------|--------|----------|
| 1 | Extract reader to adapter layer | No | High | Low | P1 |
| 2 | Add reader instrumentation | Yes (precondition 1) | High | Medium | P1 |
| 3 | Writer config validation | Yes (precondition 3) | Medium | Low | P1 |
| 4 | Document schema contract | No | High | Low | P2 |
| 5 | Integration test skeleton | Yes (precondition 2) | Medium | Medium | P2 |

**Recommended execution order:** 1 → 3 → 2 → 4 → 5

Rationale: Adjustment 1 (reader extraction) should come first because Adjustment 2 (reader instrumentation) builds on top of the extracted location. Adjustment 3 (config validation) is independent and low-effort. Adjustments 4 and 5 are documentation and test infrastructure.

---

## 6. Limits and Non-Goals

### Explicitly Out of Scope

- **No new ClickHouse tables or queries.** Wave B table expansion is not part of this stage.
- **No materialized views.** Query optimization via ClickHouse MVs remains deferred.
- **No dead-letter queue.** Writer failure mode (drop after retry) is accepted at current scale.
- **No auto-recovery from degraded state.** Requires operator restart; this is intentional.
- **No OpenTelemetry or distributed tracing.** Observability remains lightweight (structured logs + healthz counters).
- **No reader for non-evidence tables.** Only `evidence_candles` reader exists; signal/decision/strategy/risk/execution readers are Wave B scope.
- **No deduplication logic.** Writer remains at-least-once; dedup is query-time concern.
- **No shared column-order constants.** Schema contract is documented, not enforced at compile-time (Go's type system makes this awkward for ClickHouse positional INSERTs).

### Design Patterns to Reinforce

These patterns are already winning and must be preserved through restructuring:

1. **Lateral optionality** — Writer and analytical reader are optional; removing them has zero operational impact.
2. **Independent durable consumers** — Writer's `writer-*` NATS durables never compete with `store-*` durables.
3. **Graceful degradation** — Gateway returns 503 for analytical endpoints when ClickHouse is unavailable, without affecting readiness.
4. **Composition root isolation** — `cmd/*/run.go` files are pure wiring; no business logic.
5. **Mechanical transformation** — Writer mappers have no filtering, aggregation, or enrichment logic.
6. **Forward-only migrations** — No rollback migrations; changes are additive.
7. **Fail-fast configuration** — Settings validation catches dependency violations at startup.

---

## 7. Preparation for S158

After S157 restructuring is complete, the system will be positioned for S158 (Wave B analytical expansion) with:

- **Clear reader expansion point** — New family readers go in `internal/adapters/clickhouse/` alongside `candle_reader.go`.
- **Symmetric observability** — Both write and read paths have trackers, enabling baseline performance measurements before adding load.
- **Config validation** — New pipeline families are validated at startup, preventing silent misconfiguration.
- **Schema contract** — Adding a new table family follows a documented 3-point coordination rule (DDL + mapper + reader).
- **Integration test pattern** — New families can extend the test skeleton with additional assertions.

**Recommended S158 scope:** Expand analytical write + read coverage to signals and decisions (2 additional families), following the established patterns and validated by the integration test harness.
