# Wave B Family Checklist — Schema, Writer, Reader, Gateway, Tests, Runbook

> Mandatory checklist for every new analytical family introduced during Wave B.
> A family ships only when every item is checked. No exceptions.

## Entry Checklist (Before Starting a Family)

Before writing any code for a new family, confirm:

- [ ] Previous family iteration passed its gate review.
- [ ] No unresolved blocking debts from prior iteration.
- [ ] The NATS subject for the family's events already exists and is actively published by the operational pipeline.
- [ ] The event structure is stable (no pending schema changes in the domain layer).
- [ ] The ClickHouse migration sequence has no gaps (all prior migrations applied cleanly).
- [ ] CI smoke-analytical integration is in place and passing (required — no family merges without green CI).

If any entry condition is not met, the family expansion MUST NOT begin.

---

## Schema Checklist

- [ ] Migration file created at `deploy/migrations/NNN_create_<family>.sql`.
- [ ] File follows naming convention: three-digit zero-padded sequence number, `create_` prefix, snake_case family name.
- [ ] DDL uses `CREATE TABLE IF NOT EXISTS` for idempotency.
- [ ] Reverse DDL uses `DROP TABLE IF EXISTS`.
- [ ] Metadata columns present in order: `event_id String`, `occurred_at DateTime64(3)`, `correlation_id String`, `causation_id String`.
- [ ] Domain columns match the Go event struct field-by-field.
- [ ] Type mapping follows established rules:
  - [ ] Decimal strings → `Float64`.
  - [ ] Enum strings → `LowCardinality(String)`.
  - [ ] Nested objects → `String` (JSON-encoded).
  - [ ] Timestamps → `DateTime64(3)`.
- [ ] `ingested_at DateTime64(3) DEFAULT now64(3)` as final column.
- [ ] Engine: `MergeTree`.
- [ ] Partitioning: `toYYYYMM(<timestamp_column>)` (add domain key if cardinality warrants it).
- [ ] Ordering: `(source, symbol, timeframe, <type_if_applicable>, <timestamp_column>)`.
- [ ] TTL: `toDateTime(<timestamp_column>) + INTERVAL 90 DAY`.
- [ ] `cmd/migrate` applies the migration without error.
- [ ] `_migrations` metadata table records the new migration with correct checksum.

## Writer Checklist

### Mapper

- [ ] `map<Family>Row` function added to `cmd/writer/mappers.go`.
- [ ] Extracts metadata fields (event_id, occurred_at, correlation_id, causation_id) from NATS message envelope.
- [ ] Extracts domain fields from event payload.
- [ ] Uses `parseFloat` for decimal string conversions.
- [ ] Uses `marshalJSON` for nested object serialization.
- [ ] Column order exactly matches DDL column order.
- [ ] Returns `[]any` row slice.
- [ ] Row length equals DDL column count.

### Pipeline

- [ ] `WriterPipeline` entry added to catalog in `cmd/writer/pipeline.go`.
- [ ] Subject matches the NATS subject published by the operational pipeline.
- [ ] Table name matches the migration DDL table name.
- [ ] Mapper references the function from the mapper checklist above.

### Writer Tests

- [ ] Mapper unit test: happy path with all fields populated.
- [ ] Mapper unit test: missing optional fields default correctly.
- [ ] Mapper unit test: zero-value decimals parse to `0.0`.
- [ ] Mapper unit test: empty nested objects serialize to `"{}"`.
- [ ] Mapper unit test: row length assertion matches DDL column count.
- [ ] Writer starts without error with the new pipeline entry.
- [ ] Writer `/statusz` shows the new pipeline as active.

## Reader Checklist

### Adapter

- [ ] `Query<Family>History` method added in `internal/adapters/clickhouse/`.
- [ ] Uses parameterized query (no string interpolation of user input).
- [ ] Wall-clock timing: `time.Now()` before query, `time.Since()` after.
- [ ] DEBUG log on success: includes query duration and row count.
- [ ] ERROR log on failure: includes error message and query context.
- [ ] Returns domain structs (not raw database rows).
- [ ] FormatFloat used for decimal reconversion where applicable.

### Query Builder

- [ ] `Build<Family>Query` function for deterministic testing.
- [ ] Supports the same filter parameters as the HTTP endpoint.
- [ ] Query column list matches DDL column list exactly.

### Reader Tests

- [ ] Query builder unit test: default parameters produce valid SQL.
- [ ] Query builder unit test: all filter combinations produce correct WHERE clauses.
- [ ] Query builder unit test: column count in SELECT matches DDL.
- [ ] Adapter method test: verifies struct mapping from mock rows.

## Gateway Checklist

### Handler

- [ ] Handler function added in `internal/interfaces/http/handlers/analytical.go` (or separate file if warranted by size).
- [ ] Parses query parameters from request URL.
- [ ] Validates required parameters; returns 400 with descriptive error if missing.
- [ ] Validates parameter ranges; returns 400 with descriptive error if invalid.
- [ ] Sets `Server-Timing` header with `total` and `query` durations.
- [ ] Returns JSON response with structure: `{ data: [...], source: "clickhouse", meta: { query_ms, row_count } }`.

### Route

- [ ] Route registered in `internal/interfaces/http/routes/analytical.go`.
- [ ] Pattern follows `GET /analytical/<domain>/<family>`.
- [ ] Route only registered when ClickHouse is configured.
- [ ] Existing routes unmodified.

### Gateway Tests

- [ ] Handler unit test: valid request → 200 with expected JSON structure.
- [ ] Handler unit test: missing required param → 400 with error message.
- [ ] Handler unit test: invalid param value → 400 with error message.
- [ ] Handler unit test: response includes Server-Timing header.

## Contracts Checklist

- [ ] Query struct added to `internal/application/analyticalclient/contracts.go`.
- [ ] Reply struct added with `[]<Family>` data, source string, and `QueryMeta`.
- [ ] Reader interface extended or new interface declared for the family.
- [ ] Use case function added in `internal/application/analyticalclient/`.
- [ ] Use case unit test: delegates to reader and assembles reply correctly.

## Integration and Smoke Checklist

- [ ] `scripts/smoke-analytical-e2e.sh` extended with new family section.
- [ ] Smoke verifies: ClickHouse table exists after migration.
- [ ] Smoke verifies: writer consumes events and rows appear in ClickHouse.
- [ ] Smoke verifies: HTTP endpoint returns 200 with correct data structure.
- [ ] Smoke verifies: HTTP endpoint returns 400 for invalid parameters.
- [ ] All pre-existing smoke sections still pass.
- [ ] No regressions in operational smoke tests (`smoke-first-slice.sh`, `smoke-multi-symbol.sh`).

## Runbook Checklist

- [ ] Family-specific operational notes documented (if any deviations from candle baseline).
- [ ] Known limits documented (e.g., expected event volume, query patterns).
- [ ] Diagnostic signals identified: which `/statusz` and `/diagz` fields to monitor.
- [ ] Failure mode documented: what happens if writer pipeline for this family degrades.
- [ ] Recovery action documented: restart semantics (automatic per-family restart via supervisor).

## Schema Coherence Verification

- [ ] DDL column count matches mapper row length.
- [ ] DDL column order matches mapper field order.
- [ ] DDL column types are consistent with mapper transformations.
- [ ] Reader SELECT column list matches DDL column list.
- [ ] Reader scan targets match DDL column types.
- [ ] At least one test per layer (mapper, reader) explicitly asserts column alignment.

## Observability Verification

- [ ] Writer inserter counters active for new pipeline (automatic via inserter infrastructure).
- [ ] Reader adapter logs query timing on every call.
- [ ] HTTP handler sets Server-Timing header.
- [ ] Writer `/statusz` includes new pipeline stats.
- [ ] Writer `/diagz` includes new pipeline diagnostics.

## Non-Regression Verification

- [ ] Existing unit tests pass.
- [ ] Existing smoke tests pass.
- [ ] Operational pipeline unaffected (no changes to operational NATS consumers, store, or gateway operational routes).
- [ ] ClickHouse optionality preserved: operational smoke tests pass without ClickHouse running.

---

## Gate Review Sign-Off

After completing all checklists above:

- [ ] All checklist items marked complete.
- [ ] All unit tests pass (`make test`).
- [ ] Smoke-analytical passes end-to-end (`make smoke-analytical`).
- [ ] **CI pipeline passes on the branch** (GitHub Actions green).
- [ ] No regressions in existing families' smoke phases.
- [ ] Schema coherence table documented for the new family.
- [ ] Any deviations documented with rationale.
- [ ] Family-specific limits documented.
- [ ] Friction log captured (new frictions with disposition: accept/defer/fix).
- [ ] Next iteration unblocked.

**A family that cannot satisfy this full checklist does not ship.**
