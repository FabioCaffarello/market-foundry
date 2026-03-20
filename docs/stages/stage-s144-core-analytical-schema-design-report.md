# Stage S144 — Core Analytical Schema Design

## Stage Identity

| Field | Value |
|---|---|
| Stage | S144 |
| Title | Core Analytical Schema Design |
| Predecessor | S143 (Migrations and ClickHouse Entry Architecture) |
| Scope | Schema design and DDL rationale — no implementation |
| Status | **Complete** |

---

## 1. Executive Summary

S144 defines the core analytical schema for Market Foundry's ClickHouse entry: 6 event tables that project the canonical pipeline events into a queryable time-series archive.

**Key decisions:**

- **6 tables, not 9.** S143 projected 9 DDL files. This design reduces to 6 by deferring `evidence_tradebursts`, `evidence_volumes`, and a dedicated `fills` table. These are not on the core pipeline path and have no active analytical consumers.
- **Flat events, nested JSON.** Scalar fields that are likely query targets (source, symbol, timeframe, outcome, direction, disposition, side, status) are typed columns. Nested structures (signal inputs, decision inputs, constraints, fills, metadata maps) are stored as JSON strings. This avoids schema explosion while keeping primary filter axes queryable.
- **Float64 for decimals.** Go decimal strings are stored as ClickHouse Float64. The precision trade-off is explicit and acceptable for paper-trading analytics. A migration path to Decimal128 is documented.
- **MergeTree, no dedup.** All tables use plain MergeTree. Deduplication is deferred — at current scale, duplicates from writer restarts are rare and tolerable.
- **90-day uniform retention.** All 6 core tables use the same TTL. Differentiated retention is a future optimization.
- **No schema version column.** The migration catalog (`_migrations` table) is the single source of truth for schema state. Adding version columns to every table creates drift risk.

---

## 2. Pre-Condition Resolution

S144 resolves PC-02 (Core tables schema designed) from the preparation gate:

| Pre-Condition | S142 Status | S143 Status | S144 Resolution |
|---|---|---|---|
| PC-02: Core tables schema designed | NOT STARTED | Derivation rules defined | **RESOLVED.** 6 tables fully specified: columns, types, engines, partitioning, ordering, TTL, and column-level rationale. Ready for DDL file creation. |

S144 also partially resolves:

| Pre-Condition | Resolution |
|---|---|
| PC-05: Event schema versioning convention | **Partially resolved.** Versioning rules defined (EV-01 through EV-05). Formal event envelope versioning deferred — compile-time safety is sufficient at current scale. |
| PC-06: Retention policy defined | **Resolved for core tables.** 90-day uniform TTL. Telemetry retention (30 days) defined in S143 but not applicable until telemetry tables exist. |

---

## 3. Schema Summary

### 3.1 The 6 Core Tables

| Table | Pipeline Stage | Partition Key | Order Key | Columns | TTL |
|-------|---------------|---------------|-----------|---------|-----|
| `evidence_candles` | ingest | `(timeframe, toYYYYMM(open_time))` | `(source, symbol, timeframe, open_time)` | 17 | 90d |
| `signals` | derive | `toYYYYMM(timestamp)` | `(source, symbol, timeframe, type, timestamp)` | 14 | 90d |
| `decisions` | derive | `toYYYYMM(timestamp)` | `(source, symbol, timeframe, type, timestamp)` | 15 | 90d |
| `strategies` | derive | `toYYYYMM(timestamp)` | `(source, symbol, timeframe, type, timestamp)` | 16 | 90d |
| `risk_assessments` | derive | `toYYYYMM(timestamp)` | `(source, symbol, timeframe, type, timestamp)` | 18 | 90d |
| `executions` | execute | `toYYYYMM(timestamp)` | `(source, symbol, timeframe, type, timestamp)` | 20 | 90d |

### 3.2 Common Columns (All Tables)

| Column | Type | Source |
|--------|------|--------|
| `event_id` | `String` | `events.Metadata.ID` |
| `occurred_at` | `DateTime64(3)` | `events.Metadata.OccurredAt` |
| `correlation_id` | `String DEFAULT ''` | `events.Metadata.CorrelationID` |
| `causation_id` | `String DEFAULT ''` | `events.Metadata.CausationID` |
| `ingested_at` | `DateTime64(3) DEFAULT now64(3)` | ClickHouse insertion time |

### 3.3 Migration File Mapping

| Migration | File | Table |
|-----------|------|-------|
| 000 | `000_create_migrations_metadata.sql` | `_migrations` (from S143) |
| 001 | `001_create_evidence_candles.sql` | `evidence_candles` |
| 002 | `002_create_signals.sql` | `signals` |
| 003 | `003_create_decisions.sql` | `decisions` |
| 004 | `004_create_strategies.sql` | `strategies` |
| 005 | `005_create_risk_assessments.sql` | `risk_assessments` |
| 006 | `006_create_executions.sql` | `executions` |

---

## 4. Design Trade-offs

| # | Trade-off | Decision | Alternative | Why This Choice |
|---|-----------|----------|-------------|-----------------|
| T-01 | Float64 precision loss for decimal strings | Accept | Decimal128(18) | Sufficient for paper trading; migration path documented |
| T-02 | JSON strings for nested structures | Accept | Array(Tuple(...)) or separate tables | Avoids schema explosion; ClickHouse JSON functions allow ad-hoc queries |
| T-03 | No deduplication engine | Accept | ReplacingMergeTree | Duplicates are rare; dedup at query time is simpler |
| T-04 | Uniform 90-day retention | Accept | Per-table TTL differentiation | Simplicity; all tables have similar lifecycle at current scale |
| T-05 | No column codecs specified | Accept | Delta/Gorilla for Float64 | Negligible impact at current volume; optimization for later |
| T-06 | String DEFAULT '' instead of Nullable | Accept | Nullable(String) for correlation_id | Avoids null bitmap overhead; empty string semantics are clear |
| T-07 | 6 tables instead of 9 | Accept | 9 tables (S143 projection) | Guard rail: don't inflate schema beyond minimum; deferred tables have no consumers |

---

## 5. Explicit Limits

| Limit | Description | Mitigation |
|-------|-------------|------------|
| L-01 | Schema is designed for 2 symbols × 4 timeframes × 1 source | Tested at this cardinality; higher cardinality may need partition review |
| L-02 | Float64 columns cannot represent exact decimal values | Documented; Decimal128 migration path available |
| L-03 | JSON columns are not indexed | ClickHouse JSONExtract functions for ad-hoc use; materialized columns for hot paths |
| L-04 | No deduplication guarantee | Duplicates possible after writer restart; tolerable at paper-trading scale |
| L-05 | No automated schema sync with Go structs | Manual review per EV-01; compile-time safety catches type mismatches |
| L-06 | No cross-table referential integrity | ClickHouse is not OLTP; event chain is traced via correlation_id, not foreign keys |

---

## 6. Items Out of Scope

| Item | Why Deferred | When to Revisit |
|------|-------------|-----------------|
| `evidence_tradebursts` table | No active analytical consumer for trade burst data | When burst frequency analysis is an active need |
| `evidence_volumes` table | No active analytical consumer for volume profile data | When volume pattern analysis is an active need |
| Dedicated `fills` table | 0–1 fills per execution at paper scale; JSON array is sufficient | Real trading with partial fills |
| `runtime_telemetry` table | P2 priority; different ingestion pattern (scraper, not NATS consumer) | After core schema is proven |
| Materialized views | Query optimization; needs query patterns first | After query surface extension (Phase 4) |
| ClickHouse projections | Alternative sort orders for the same data | After query patterns reveal hot paths |
| Column-level codecs | Negligible at current volume | Data volume > 10M rows per table |
| Formal event schema versioning (envelope version field) | Single-developer, compile-time safety sufficient | Multiple developers or multiple writer versions |
| ReplacingMergeTree | Dedup not needed at current scale | Observed duplicate rate > 1% |

---

## 7. Invariants Preserved

| Invariant | S144 Impact |
|-----------|-------------|
| INV-01: Pipeline functions without ClickHouse | Schema design has no pipeline code impact |
| INV-02: No service except writer depends on ClickHouse | Schema design adds no dependencies |
| INV-05: Schema follows events | Every column maps to a Go struct field (verified per table) |
| INV-06: All schema changes go through migrations | Migration file mapping defined (001–006) |

---

## 8. Deliverables

| # | Document | Path |
|---|---|---|
| 1 | Core Schema Design | `docs/architecture/clickhouse-core-schema-design.md` |
| 2 | Core Tables and DDL Rationale | `docs/architecture/clickhouse-core-tables-and-ddl-rationale.md` |
| 3 | Schema Versioning and Evolution Rules | `docs/architecture/clickhouse-schema-versioning-and-evolution-rules.md` |
| 4 | Stage Report (this document) | `docs/stages/stage-s144-core-analytical-schema-design-report.md` |

---

## 9. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|---|---|---|
| 6 core tables clearly defined | **PASS** | Tables 4.1–4.6 in core schema design: columns, types, engines, partitioning, ordering, TTL |
| Schema is small, canonical, and justifiable | **PASS** | 6 tables (not 9); each maps 1:1 to a pipeline event; rationale per column |
| Ordering/partitioning/keys make sense for current stage | **PASS** | Evidence partitioned by (timeframe, month); pipeline tables by month; ordering follows natural query patterns |
| Trade-offs and limits are explicit | **PASS** | 7 trade-offs (T-01–T-07), 6 limits (L-01–L-06) documented |
| Base ready for real migrations | **PASS** | Migration files 001–006 mapped; DDL fully specified; `cmd/migrate` can apply them |
| No analytics modeled beyond minimum | **PASS** | No materialized views, no aggregations, no derived tables |
| No tables created by query convenience | **PASS** | Every table maps to a domain event, not a query pattern |
| Schema not inflated beyond minimum | **PASS** | Reduced from S143's 9 projections to 6; 3 tables explicitly deferred with triggers |
| Not coupled to a single product surface | **PASS** | Schema is event-oriented; no query-specific denormalization |
| What stays for later is documented | **PASS** | 9 items in out-of-scope section with triggers for revisiting |

---

## 10. Preparation for S145

S144 produces the complete schema design. The next stage should implement the migration infrastructure and apply the schema.

**Recommended S145 scope:** Build `cmd/migrate` + apply migrations 000–006.

| Entry Criterion | Status |
|---|---|
| `cmd/migrate` architecture defined | PASSED (S143) |
| Core schema designed with DDL | PASSED (S144) |
| Migration file mapping defined (000–006) | PASSED (S144) |
| Self-bootstrapping metadata designed | PASSED (S143, migration 000) |
| ClickHouse container available | PASSED (existing in docker-compose) |
| No prior schema to conflict with | PASSED (greenfield) |

### S145 Scope Preview

| Deliverable | Description |
|-------------|-------------|
| `cmd/migrate` binary | Go binary with `up`, `up --dry-run`, `status`, `validate` commands |
| `internal/migrate/` package | Runner, catalog, checksum, metadata, reporter |
| `deploy/migrations/000–006` | 7 SQL files (metadata + 6 core tables) |
| Makefile targets | `migrate-up`, `migrate-dry-run`, `migrate-status`, `migrate-validate` |
| Validation | All success criteria from S143 section 10 + schema matches S144 design |

### S145 Success Criteria Preview

| Criterion | Verification |
|-----------|-------------|
| `migrate up` creates all 6 core tables + `_migrations` | Run against empty ClickHouse |
| Tables match S144 schema design exactly | Compare DDL with design |
| `migrate up` is idempotent | Run twice; second is no-op |
| `migrate validate` detects drift | Modify a file; verify exit 1 |
| Smoke tests still pass | `smoke-first-slice.sh` unaffected (INV-01) |
| No NATS dependency | Run with NATS stopped |

---

## 11. Decision Log

| Decision | Options Considered | Chosen | Rationale |
|----------|-------------------|--------|-----------|
| Number of core tables | 6 (pipeline events only) vs. 9 (S143 projection) | 6 | Guard rail compliance; deferred tables have no active consumers |
| Decimal handling | Float64 vs. Decimal128 vs. String | Float64 | Queryable; sufficient precision for paper trading; migration path exists |
| Nested structure storage | JSON String vs. Array(Tuple) vs. Separate tables | JSON String | Simplest; schema-change resilient; ClickHouse JSON functions available |
| Deduplication engine | MergeTree vs. ReplacingMergeTree | MergeTree | Simpler; duplicates rare; query-time dedup sufficient |
| Nullable vs. Default | Nullable(String) vs. String DEFAULT '' | String DEFAULT '' | Avoids null bitmap; clear semantics; ClickHouse best practice |
| Partition granularity | Monthly vs. Daily | Monthly | Low partition count at current scale; prevents partition explosion |
| Retention uniformity | Uniform 90d vs. Per-table differentiated | Uniform 90d | Simplicity; all tables at same lifecycle stage |
| Schema versioning | Version column vs. Migration catalog | Migration catalog | Single source of truth; no drift risk |
