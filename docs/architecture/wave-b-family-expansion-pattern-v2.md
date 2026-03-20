# Wave B Family Expansion Pattern v2

> Hardened expansion pattern incorporating lessons from the first Wave B family (Signals/RSI).
> Supersedes `wave-b-family-expansion-pattern.md` as the canonical reference for family 2+.

## Changes from v1

| Area | v1 | v2 |
|---|---|---|
| CI gate | "Recommended before first family" | **Required before merge** — CI must pass on the branch |
| Gate review criteria | Informal self-review | 5 explicit criteria (tests, smoke, CI, no regressions, schema coherence table) |
| Documentation artifact | "Any family-specific limits or deviations" | 4 mandatory sections (coherence table, endpoint spec, known limits, friction log) |
| Smoke parameterization | No threshold | **Hard requirement at family 3** — extract `validate_analytical_family()` |
| Constructor ergonomics | No threshold | **Hard requirement at family 3** — switch to `AnalyticalHandlerDeps` struct |
| Naming cleanup | No plan | Rename `parseEvidenceKeyParams()` → `parseAnalyticalKeyParams()` at family 3 |

## Guiding Principle

Wave B grows by controlled iteration: **one family at a time**, each following the same template the candle family established. The pattern is small, mechanical, and verifiable. If a step cannot be completed, the family does not ship.

## Expansion Unit

One **expansion unit** consists of:

1. One ClickHouse table (migration DDL).
2. One writer mapper function.
3. One writer pipeline entry in the catalog.
4. One reader adapter method.
5. One application-layer use case with contracts.
6. One HTTP handler + route registration.
7. One integration test (HTTP-level).
8. One smoke-analytical-e2e section covering the new family.
9. Documentation: schema coherence table, endpoint spec, known limits, friction log.

No partial units. A family either delivers all nine artifacts or it does not merge.

## Iteration Cadence

```
┌─────────────────────────────────────────────────────────┐
│  Iteration N                                            │
│                                                         │
│  1. Schema  ──▶  2. Writer  ──▶  3. Reader              │
│                                                         │
│  4. Gateway  ──▶  5. Tests  ──▶  6. Smoke update        │
│                                                         │
│  7. Documentation  ──▶  8. CI passes  ──▶  9. Gate      │
│                                                         │
│  Gate passed? ──▶ Iteration N+1 unlocked                │
└─────────────────────────────────────────────────────────┘
```

Each iteration follows a strict left-to-right dependency chain. No step may begin until its predecessor is complete and verified.

## Step-by-Step Template

### Step 1 — Schema (Migration)

- Add a new migration file: `deploy/migrations/NNN_create_<family>.sql`.
- Follow the established DDL conventions:
  - Metadata columns: `event_id String`, `occurred_at DateTime64(3)`, `correlation_id String`, `causation_id String`.
  - Domain columns matching the Go event struct, typed per the column mapping rules (decimals as Float64, enums as LowCardinality(String), nested objects as String JSON).
  - `ingested_at DateTime64(3) DEFAULT now64(3)` as the final column.
  - MergeTree engine, 90-day TTL on `toDateTime(occurred_at)` or `toDateTime(timestamp)`.
  - `CREATE TABLE IF NOT EXISTS` for idempotency.
  - Reverse migration: `DROP TABLE IF EXISTS`.
- Verify: `cmd/migrate` applies the migration without error and records it in `_migrations`.

### Step 2 — Writer Mapper

- Add `map<Family>Row` function in `cmd/writer/mappers.go`.
- Follow the candle mapper pattern exactly:
  - Extract metadata fields (event_id, occurred_at, correlation_id, causation_id) from the NATS message envelope.
  - Extract domain fields from the event payload.
  - Use `parseFloat` for decimal strings, `marshalJSON` for nested objects.
  - Column order MUST match DDL column order exactly.
  - Return `[]any` row slice.
- Add unit tests for the mapper in `cmd/writer/mappers_test.go`:
  - Happy path with all fields populated.
  - Edge cases: missing optional fields, zero-value decimals, empty nested objects.

### Step 3 — Writer Pipeline Entry

- Add a `WriterPipeline` entry in the pipeline catalog (`cmd/writer/pipeline.go`).
- Fields: subject (NATS subject), table (ClickHouse table name), mapper (the function from Step 2).
- The pipeline automatically gets its own consumer-inserter pair via the supervisor.
- Verify: writer `/statusz` shows the new pipeline after restart.

### Step 4 — Reader Adapter

- Add `Query<Family>History` method in `internal/adapters/clickhouse/`.
- Follow the candle reader pattern:
  - Parameterized query construction (no string interpolation of user input).
  - Wall-clock timing via `time.Now()` / `time.Since()`.
  - Structured logging: DEBUG on success with timing, ERROR on failure.
  - Return domain structs, not raw rows.
- Add `Build<Family>Query` for deterministic query testing without ClickHouse.
- Add unit tests in the corresponding `_test.go` file.

### Step 5 — Application Contracts and Use Case

- Add query/reply structs in `internal/application/analyticalclient/contracts.go`.
- Add the use case function (e.g., `Get<Family>History`) in its own file under `internal/application/analyticalclient/`.
- The use case:
  - Accepts the query struct.
  - Calls the reader adapter.
  - Returns the reply struct with metadata (query_ms, row_count).
- Add unit tests for the use case.

### Step 6 — HTTP Handler and Route

- Add handler in `internal/interfaces/http/handlers/analytical.go` (or a new file if analytical.go grows beyond ~300 lines).
- Follow the candle handler pattern:
  - Parse and validate query parameters.
  - Return 400 with descriptive error for invalid input.
  - Set `Server-Timing` header.
  - Return JSON response with data + source + meta.
- Register route in `internal/interfaces/http/routes/analytical.go`:
  - Pattern: `GET /analytical/<domain>/<family>`.
  - Only registered when ClickHouse is configured.
- Add handler unit tests covering:
  - Valid request → 200 with expected structure.
  - Missing required params → 400.
  - Invalid param values → 400.

### Step 7 — Smoke Test Update

- Extend `scripts/smoke-analytical-e2e.sh` with a new section for the family.
- Required checks:
  - ClickHouse table exists and has rows after writer consumption.
  - HTTP endpoint returns 200 with correct structure.
  - Error handling returns appropriate 400 responses.
  - Server-Timing header present.
- The smoke test MUST pass end-to-end before the family can be considered complete.

### Step 8 — Documentation

Every family MUST produce documentation with these four sections:

1. **Schema coherence table:** DDL column ↔ mapper field ↔ reader column alignment (all columns listed).
2. **Endpoint specification:** HTTP method, path, query parameters, response contract (JSON shape).
3. **Known limits:** Any simplifications, deferred validations, or intentional gaps.
4. **Friction log:** New frictions discovered during the iteration, with disposition (accept/defer/fix).

### Step 9 — Gate Review

Gate review requires ALL of the following:

1. All unit tests pass (`make test`).
2. Smoke-analytical passes end-to-end (`make smoke-analytical`).
3. **CI pipeline passes on the branch** (GitHub Actions green).
4. No regressions in existing families' smoke phases.
5. Schema coherence table documented for the new family.

If any criterion is not met, the family does not ship.

## Naming Conventions

| Artifact | Pattern | Example (signals) |
|---|---|---|
| Migration file | `NNN_create_<family>.sql` | `002_create_signals.sql` |
| Writer mapper | `map<Family>Row` | `mapSignalRow` |
| Pipeline entry | `<family>Pipeline` | `signalPipeline` |
| Reader method | `Query<Family>History` | `QuerySignalHistory` |
| Query builder | `Build<Family>Query` | `BuildSignalQuery` |
| Use case | `Get<Family>History` | `GetSignalHistory` |
| Handler | `Get<Family>History` | `GetSignalHistory` |
| Route | `GET /analytical/<domain>/<family>` | `GET /analytical/signal/history` |
| Test file | `<component>_test.go` | `mappers_test.go`, `signal_reader_test.go` |

## Schema Coherence Rule

For every family, the following three artifacts MUST be column-aligned:

1. **DDL** (migration SQL): defines column names and types.
2. **Writer mapper**: produces `[]any` in the exact DDL column order.
3. **Reader adapter**: SELECT columns and scan targets match DDL.

Any mismatch between these three is a blocking defect. Schema coherence is verified by:
- Mapper unit tests asserting row length matches DDL column count.
- Reader unit tests asserting query column list matches DDL.

## Observability Parity Rule

Every family MUST have the same observability coverage as candles:
- Writer inserter counters: `events_buffered`, `events_flushed`, `events_dropped`, `flush_errors`, `flush_count`, `buffer_depth`.
- Reader adapter: wall-clock query timing, structured error logging.
- HTTP handler: `Server-Timing` header with total and query durations.
- Writer `/statusz` and `/diagz` include the new pipeline.

These are provided automatically by the inserter and supervisor infrastructure. The family author verifies they appear correctly.

## Hardening Thresholds (Family-Indexed)

These are hard requirements triggered at specific family counts:

| Threshold | Trigger | Action |
|---|---|---|
| Smoke parameterization | Family 3 | Extract `validate_analytical_family()` in smoke script |
| Constructor refactor | Family 3 | Switch to `AnalyticalHandlerDeps` struct in handler |
| Naming cleanup | Family 3 | Rename `parseEvidenceKeyParams()` → `parseAnalyticalKeyParams()` |
| Codegen evaluation | Family 4 | Assess whether to generate reader/handler/test boilerplate |

These thresholds are commitments, not suggestions. If family 3 ships without completing the family-3 thresholds, it is a blocking defect.

## What This Pattern Does NOT Cover

- Materialized views or aggregation tables.
- Cross-family joins or composite queries.
- Custom retention policies per family.
- Dynamic pipeline registration.
- Backfill or replay mechanisms.
- External observability infrastructure (Prometheus, Grafana).
- Dead-letter queues or overflow persistence.

These remain explicitly out of scope for Wave B per S162 constraints.
