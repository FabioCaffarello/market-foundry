# Analytical Responsibility Anti-Patterns and Non-Goals

> Explicit catalog of anti-patterns to avoid and non-goals to respect when evolving the analytical layer.

## 1. Anti-Patterns Identified

### AP-01: Reader Implementation in Composition Root

**What happened:** `analyticalCandleReader` is defined in `cmd/gateway/analytical_reader.go`, inside the gateway's composition root package.

**Why it's an anti-pattern:** Composition roots (`cmd/*/`) should wire dependencies, not implement data access logic. When the reader struct lives alongside `compose.go` and `run.go`, the package conflates two responsibilities: gateway lifecycle orchestration and ClickHouse query construction.

**Consequence if uncorrected:** Each Wave B family reader added to `cmd/gateway/` further dilutes the composition root. The gateway package grows domain-specific query logic that belongs in the adapter or application layer.

**Correction:** Move reader implementations to `internal/adapters/clickhouse/` where they join the existing adapter layer. The gateway composition root wires them via the `CandleReader` interface.

---

### AP-02: Distributed Schema Knowledge Without Contract

**What happened:** ClickHouse column order is defined in three places:
1. `deploy/migrations/001_create_evidence_candles.sql` — DDL column definitions.
2. `cmd/writer/mappers.go` — positional `[]any` slice per row.
3. `cmd/gateway/analytical_reader.go` — hardcoded SELECT column list and `rows.Scan()` argument order.

**Why it's an anti-pattern:** Positional coupling across three unrelated files creates a silent coordination requirement. No compile-time or static check validates consistency. A column addition, removal, or reorder breaks either the write path, read path, or both — discovered only at runtime.

**Consequence if uncorrected:** Wave B multiplies this 3×N across 6 table families. One misaligned column in a mapper silently corrupts analytical data.

**Correction (minimal):** Document the schema contract explicitly: each table maps to one migration file, one mapper function, and one reader query. Treat any schema change as requiring a 3-point synchronized edit. Long-term, consider shared column-name constants (deferred — Go positional INSERT makes this awkward).

---

### AP-03: Asymmetric Observability

**What happened:** Writer has 10+ tracker counters per family (events_received, events_flushed, buffer_depth, flush_duration_ms, events_overflowed, events_dropped, flush_failures, pipeline_restarts, pipeline_degraded). Reader has zero: no query timing, no error counting, no structured logging.

**Why it's an anti-pattern:** Asymmetric observability creates blind spots. When an analytical query is slow or returns unexpected results, there's no diagnostic path. The operator can see writer flushing correctly but cannot determine if the read path is healthy.

**Consequence if uncorrected:** Wave B adds more reader families, each uninstrumented. Debugging analytical issues becomes guess-and-restart.

**Correction:** Add structured logging (query timing, row count, errors) and healthz tracker counters to the reader path. Match the writer's instrumentation depth.

---

### AP-04: Config Validation Gap for Writer

**What happened:** Writer's `run.go` validates that ClickHouse addr is non-empty but does not validate:
- Whether configured family names are recognized.
- Whether batch parameters are sane (e.g., max_pending >= batch_size).
- Whether ClickHouse tables for configured families exist (ping only, not schema check).

**Why it's an anti-pattern:** Violates fail-fast principle. A typo in family name (e.g., `"candles"` instead of `"candle"`) causes the pipeline declaration to silently skip the family, producing no events — indistinguishable from a legitimately empty pipeline until manually diagnosed.

**Consequence if uncorrected:** Each new family added in Wave B increases the surface area for misconfiguration.

**Correction:** Add explicit validation in writer startup: all families must match known catalog from `settings.schema.go`; batch parameters must be in valid ranges.

---

### AP-05: No Automated Integration Test for Data Path

**What happened:** Unit tests cover individual components (mappers, inserter, supervisor, use case, handler), but no automated test validates the composed flow: NATS → writer → ClickHouse → reader → HTTP.

**Why it's an anti-pattern:** Component tests verify correctness in isolation but cannot catch integration failures: wrong column order between mapper and reader, wrong NATS subject pattern, wrong ClickHouse table name.

**Consequence if uncorrected:** Integration errors are discovered only via manual testing (`tests/http/analytical.http`) or production incident.

**Correction:** Create a minimal integration test that publishes a known event, waits for writer flush, and verifies the event appears in reader response.

---

## 2. Non-Goals — What This Stage Does NOT Do

### NG-01: No Wave B Expansion

This stage reviews and restructures by responsibility. It does NOT add new analytical families, tables, readers, or endpoints. Wave B expansion is a separate stage that builds on the restructured foundation.

### NG-02: No Materialized Views

ClickHouse materialized views (pre-aggregated query results, real-time rollups) are a performance optimization. They are architecturally valid but not needed at current scale and query patterns. Adding them prematurely introduces schema complexity and migration coordination overhead.

### NG-03: No Dead-Letter Queue

Writer's current failure mode (retry with backoff → drop batch → log ERROR) is intentional and appropriate for analytical projection tolerances. Adding DLQ infrastructure introduces operational complexity (monitoring, replaying, deduplication) that is not justified at single-operator scale.

### NG-04: No Auto-Recovery from Degraded

When a writer pipeline family exhausts its restart budget (5 attempts), it enters permanent degraded state until operator restarts the writer. Automatic recovery (e.g., periodic retry after degradation) adds state machine complexity. At current scale, operator intervention is acceptable.

### NG-05: No Distributed Tracing

OpenTelemetry, Jaeger, or similar distributed tracing infrastructure is not part of the analytical layer. Structured logging + healthz counters provide sufficient observability for the current architecture. Tracing becomes relevant when cross-service request correlation is needed at scale.

### NG-06: No Shared Column Constants

A type-safe column-order contract between migrations, mappers, and readers would be ideal. However, ClickHouse's batch INSERT protocol uses positional `[]any` slices, and Go's type system doesn't naturally express column-order constraints. The compile-time safety net would require code generation or reflection, adding maintenance burden disproportionate to current risk. Document the contract instead.

### NG-07: No Query-Time Deduplication

Writer delivers at-least-once. Crash-restart can cause duplicate rows in ClickHouse. The current approach is to tolerate duplicates and apply `DISTINCT` at query time if needed. Building insert-level deduplication (ClickHouse ReplacingMergeTree, event_id dedup logic) adds complexity without clear benefit at current data volumes.

### NG-08: No Dynamic Family Registration

Pipeline families are statically configured via JSONC. Dynamic runtime registration (add/remove families without restart) would require a control plane for the writer — unwarranted complexity.

### NG-09: No Backfill Mechanism

If writer misses events (due to outage or late start), there is no mechanism to replay historical NATS messages. Backfill requires either NATS stream replay (time-based seek) or operational pipeline re-ingestion. This is a future capability, not an S157 concern.

### NG-10: No Schema Versioning in Code

Schema version tracking exists only in `_migrations` table (version + checksum). There is no in-code schema version constant or compatibility check between writer binary version and migration catalog version. This is acceptable because writer and migrate are deployed together — schema drift is a deployment error, not a runtime state.

---

## 3. Design Patterns to Protect

These patterns are established, validated, and must be preserved during any restructuring:

### DP-01: Lateral Optionality
Writer and analytical reader are optional components. Their absence has zero impact on the operational pipeline. This must remain true: no operational service may import `internal/adapters/clickhouse/` or depend on writer health.

### DP-02: Independent Durable Consumers
Writer uses `writer-*` prefixed NATS durable consumer names, completely independent from `store-*` durables. Cursor positions, redelivery state, and ack status are isolated. This separation must be maintained when adding new families.

### DP-03: Graceful Degradation
Gateway returns 503 for analytical endpoints when ClickHouse is unavailable, without affecting operational readiness (R-02 compliance). This conditional wiring pattern (`nil` reader → 503 handler) must extend to all new analytical endpoints.

### DP-04: Composition Root Purity
`cmd/*/run.go` files compose dependencies via phases. They don't contain business logic, query construction, or data transformation. Restructuring must restore this purity where it has drifted (AP-01).

### DP-05: Mechanical Transformation
Writer mappers perform no filtering, aggregation, enrichment, or deduplication. They convert domain event fields to ClickHouse column values mechanically. This simplicity is an architectural choice, not a gap. Future analytics (aggregation, windowing) belong in ClickHouse materialized views or downstream systems, not in the writer.

### DP-06: Forward-Only Migration
No rollback migrations. Schema changes are additive. This simplifies the migration tool, eliminates destructive paths, and aligns with ClickHouse's append-oriented storage model.

### DP-07: Fail-Fast Configuration
Settings validation at startup catches misconfiguration before any listener starts. This must extend to ClickHouse-specific parameters (AP-04).

### DP-08: Actor-Based Lifecycle
Writer uses Hollywood actors for consumer/inserter lifecycle. Supervisors manage restart budgets with exponential backoff. This pattern handles partial failures gracefully and must not be replaced with ad-hoc goroutine management.
