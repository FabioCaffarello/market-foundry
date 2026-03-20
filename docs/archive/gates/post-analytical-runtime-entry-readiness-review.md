# Post-Analytical Runtime Entry — Readiness Review

> Stage S150 formal review. Covers S143–S149 as a single capability wave.

## 1. Executive Summary

The first entry of ClickHouse, migrations, writer, and historical query surface into market-foundry succeeded in its primary objective: proving that an analytical projection layer can be added to the pipeline without contaminating the operational baseline. The implementation is small, disciplined, and structurally optional.

However, "succeeded" is not "complete." The wave delivered a viable skeleton — not a production-grade analytical runtime. Several gaps remain: zero test coverage on the writer and reader paths, silent data loss under buffer overflow, no failure recovery in the writer supervisor, and no observability beyond basic health checks. These are acceptable for a first entry at paper-trading scale but would be blocking for any production or multi-user deployment.

The readiness verdict: **the analytical runtime is a valid first projection, not a proven capability.** The next wave should harden before expanding.

---

## 2. Review Criteria and Findings

### 2.1 Was ClickHouse introduced without contaminating the baseline?

**Verdict: Yes — clean separation confirmed.**

Evidence:
- Gateway builds and starts without ClickHouse. The `compose.go` path returns `nil` when `config.ClickHouse.Addr == ""`, and analytical routes are conditionally registered only when the use case is available.
- Gateway readiness checks include only NATS, configctl, and store. ClickHouse is explicitly excluded.
- Docker Compose: gateway has no `depends_on` clause for ClickHouse. Only the writer service declares that dependency.
- No operational service (`ingest`, `derive`, `store`, `execute`, `configctl`) imports the ClickHouse adapter or any analytical module.
- Settings validation skips ClickHouse config when `addr` is empty.
- Smoke tests (`smoke-first-slice.sh`) validate the operational path without ClickHouse.

Remaining risk: the `ClickHouseConfig` struct lives in `internal/shared/settings/schema.go`, which means all binaries that parse settings carry the field definition. This is a cosmetic coupling — the struct is never populated in non-analytical binaries — but it slightly widens the shared surface.

### 2.2 Is `cmd/migrate` and the migration catalog robust enough?

**Verdict: Adequate for current scale. Not yet robust.**

What works:
- Forward-only migration model with checksum drift detection.
- Catalog parsing with strict naming validation, duplicate detection, and version sorting.
- 8 unit tests covering catalog and checksum logic.
- Idempotent DDL (`CREATE TABLE IF NOT EXISTS`) prevents re-application failures.
- Self-bootstrapping `_migrations` metadata table.

What is missing:
- **No integration tests** for the runner. `Up`, `Status`, and `Validate` methods are untested against a real database.
- **No concurrency protection.** Two simultaneous `migrate up` runs could apply the same migration twice. At single-developer scale this is unlikely; at team scale it becomes a real hazard.
- **No transaction semantics.** Each migration applies individually. A failure mid-catalog leaves the database in a partially-migrated state with no automated recovery.
- **Dry-run is limited.** Only `up` supports dry-run; `validate` does not offer a preview mode.

Assessment: the migration tool is functional and well-factored but has not been tested under adversarial conditions. It is sufficient for a solo operator managing a single ClickHouse instance. It would need hardening before team-scale adoption.

### 2.3 Was the core schema a good first base?

**Verdict: Yes — conservative and aligned with domain events.**

Strengths:
- 6 tables map 1:1 to the 6 pipeline domain events. No speculative schema.
- Uniform metadata columns (`event_id`, `occurred_at`, `correlation_id`, `causation_id`, `ingested_at`) enable cross-table correlation.
- `LowCardinality(String)` for bounded enums, `Float64` for decimals, `DateTime64(3)` for timestamps — all reasonable choices at current scale.
- 90-day TTL prevents unbounded growth.
- Monthly partitioning (`toYYYYMM`) is appropriate for the expected data volume.
- All migrations are well-commented with domain references and reversibility notes.

Known trade-offs accepted consciously:
- `Float64` loses precision vs. `Decimal128`. Acceptable for paper-trading analytics; migration path documented.
- No deduplication at storage level (`MergeTree`, not `ReplacingMergeTree`). Duplicates tolerated; `SELECT DISTINCT` at query time is the documented mitigation.
- Complex nested structures stored as `String` (JSON), not native ClickHouse JSON type. Queryable via `JSONExtract*` but without indexing.
- No secondary indexes beyond the ORDER BY key.

Assessment: the schema is a solid minimal foundation. The 3 deferred tables (tradebursts, volumes, fills) were correctly excluded — they have no active analytical consumers.

### 2.4 Is the writer service well-delimited?

**Verdict: Well-delimited in scope. Fragile in failure handling.**

Scope boundaries are clear:
- Standalone binary with its own `go.mod` and configuration.
- Consumes NATS events via independent durable consumers (`writer-*` prefix).
- Mechanical type mapping only — no filtering, aggregation, or enrichment.
- Append-only writes; never reads from ClickHouse.
- Never publishes back to NATS.
- 6 pipeline families covering all 6 core tables.

Failure handling concerns:
- **No recovery for failed pipelines.** If a consumer or inserter actor fails, the entire family stops permanently until the writer process restarts. There is no internal restart mechanism.
- **Silent data loss on buffer overflow.** When `max_pending` (10,000) is exceeded, oldest rows are evicted with only a warning log. No metric is exported; no alarm fires.
- **Single INSERT attempt.** Despite the architecture document describing exponential backoff (1s–30s, 5 attempts), the actual implementation makes a single attempt and drops the batch on failure.
- **Mapper silent fallbacks.** Parse errors produce zero values; JSON marshal errors produce `"{}"`. These are invisible data quality issues.

Test coverage:
- **Zero tests.** No unit tests for mappers, inserters, consumers, or supervisor. No integration tests for the end-to-end write path. This is the single largest gap in the analytical runtime.

Assessment: the writer is correctly scoped and structurally isolated. Its runtime behavior under failure conditions is weaker than documented. The gap between documented failure semantics and actual implementation needs to be closed before any expansion.

### 2.5 Was the historical query surface introduced with clear boundaries?

**Verdict: Yes — minimal, additive, and well-bounded.**

Boundary enforcement:
- Single endpoint: `GET /analytical/evidence/candles` with required params (`source`, `symbol`, `timeframe`) and optional time range/limit.
- Route prefix `/analytical/*` is distinct from operational `/evidence/*`.
- Conditional registration: routes only exist when ClickHouse is configured.
- Returns `503 Unavailable` when ClickHouse is down (not an error in operational readiness).
- Response includes `source: "clickhouse"` to distinguish from operational results.
- Max 500 rows per request prevents unbounded result sets.

Test coverage:
- 7 unit tests for the use case (validation, error handling, defaults).
- 6 unit tests for the HTTP handler (param parsing, error responses).
- **Zero tests** for the reader adapter (`analytical_reader.go`), which contains the actual ClickHouse query logic.

Assessment: the query surface is correctly minimal and does not leak into the operational path. The reader adapter needs tests before the query surface is expanded to additional tables.

---

## 3. Limits and Risks That Remain

### 3.1 Critical (must be addressed before expansion)

| Risk | Impact | Current State |
|------|--------|--------------|
| Writer has zero test coverage | Data quality and reliability unknown under failure | No unit or integration tests |
| Writer has no pipeline recovery | Single family failure stops that event projection permanently | Requires process restart |
| Reader adapter has zero tests | Query correctness unverified | No unit or integration tests |
| INSERT failure handling diverges from docs | Architecture says retry with backoff; code does single-attempt-and-drop | Gap between specification and implementation |
| Silent data loss on buffer overflow | Analytical gaps invisible to operators | Warning log only; no metric or alert |

### 3.2 Significant (should be addressed during hardening)

| Risk | Impact | Current State |
|------|--------|--------------|
| No migration runner integration tests | Schema application correctness not verified in CI | Unit tests only |
| No concurrent migration protection | Two simultaneous runs could corrupt state | Single-operator assumption |
| Mapper silent fallbacks | Parse errors become zero values without visibility | Logged but not counted |
| No observability on write/read paths | Batch latency, row throughput, error rates invisible | Health check only |
| Default credentials in config templates | Security gap if deployed without customization | `default/clickhouse` hardcoded |

### 3.3 Acceptable at current scale (monitor, do not fix now)

| Item | Rationale |
|------|-----------|
| Float64 precision loss | Paper trading; documented migration path to Decimal128 |
| No deduplication | Low event rate; `SELECT DISTINCT` sufficient |
| No secondary indexes | 2 symbols × 4 timeframes; table scans are fast |
| No query caching | Sub-second response times at current data volume |
| 90-day hardcoded TTL | Sufficient for backtesting verification |

---

## 4. Readiness Verdict

The analytical runtime entry (S143–S149) met its structural objectives:

- ClickHouse is optional and removable.
- The operational baseline is uncontaminated.
- The schema is small and canonical.
- The writer is scoped and isolated.
- The query surface is minimal and bounded.
- Optionality rules (R-01 through R-10) are enforced.

The analytical runtime entry did **not** meet production-grade reliability:

- Writer and reader paths are untested.
- Failure handling is weaker than specified.
- No observability beyond health checks.
- No recovery mechanism for failed pipelines.

**The system is ready for hardening. It is not ready for expansion.**

---

## 5. Recommendation for Next Wave

**Primary recommendation: Hardening of the analytical runtime.**

Before expanding the schema, adding new query endpoints, or attempting cold-start bootstrap, the existing skeleton must be tested, observed, and made resilient.

Specific sequence:

1. **Writer test coverage** — unit tests for mappers, inserter batch logic, and supervisor; integration test for NATS-to-ClickHouse flow.
2. **Reader test coverage** — unit tests for query building and row scanning; integration test for ClickHouse-to-HTTP flow.
3. **Migration runner integration tests** — verify `Up`, `Status`, `Validate` against a real ClickHouse instance.
4. **Align implementation with documented failure semantics** — implement retry with backoff as specified, or update the architecture document to match the simpler single-attempt behavior (decide which is correct, then make code and docs agree).
5. **Basic observability** — per-family counters for events flushed, events dropped, batch latency. These do not need Prometheus; structured log counters are sufficient.
6. **Supervisor recovery** — add pipeline restart capability so that a transient consumer failure does not require process restart.

Only after hardening should expansion be considered:
- Additional query endpoints (signals, decisions, executions).
- Additional writer families (tradeburst, volume, ema_crossover, venue_market_order).
- Cold-start bootstrap (derive querying ClickHouse for warm-up data).

See [next-wave-recommendations-after-analytical-runtime-entry.md](next-wave-recommendations-after-analytical-runtime-entry.md) for detailed sequencing.
