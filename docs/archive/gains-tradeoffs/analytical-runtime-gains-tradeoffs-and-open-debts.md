# Analytical Runtime — Gains, Trade-offs, and Open Debts

> Companion to the S150 readiness review. Enumerates what the S143–S149 wave actually gained, what it traded away, and what debts remain open.

---

## 1. Gains

### 1.1 Structural gains

| Gain | Evidence |
|------|----------|
| **ClickHouse is structurally optional** | Gateway builds and runs without ClickHouse. No `depends_on` in docker-compose. Readiness checks exclude ClickHouse. Smoke tests pass without it. |
| **Binary optionality over feature flags** | Writer is a separate binary (`cmd/writer/`). Remove from compose = complete removal. No conditional branches in operational code. |
| **Operational baseline is uncontaminated** | No operational service imports ClickHouse driver. No NATS adapter changes. No handler changes. All analytical code is additive. |
| **Migration infrastructure exists** | `cmd/migrate` with forward-only model, checksum validation, catalog conventions, and Makefile integration. Schema changes are versioned and auditable. |
| **Canonical schema established** | 6 tables covering all 6 pipeline families. Uniform metadata columns. Consistent partitioning and TTL. Schema follows events, not speculative queries. |
| **Write path exists** | NATS → writer → ClickHouse flow works. Events are persisted as they flow through the operational pipeline. |
| **Read path exists** | ClickHouse → gateway → HTTP flow works. Historical candle data is queryable via REST API. |
| **Pattern reuse** | Writer follows the canonical consumer-projection dual-actor pattern established by the store service. Cognitive overhead is low. |

### 1.2 Architectural gains

| Gain | Evidence |
|------|----------|
| **Operational/analytical boundary is codified** | 10 optionality rules (R-01–R-10), 5 boundary rules (B-01–B-05), 8 invariants (INV-01–INV-08). Documented and enforced. |
| **Route prefix separation** | `/evidence/*` = operational (NATS KV). `/analytical/*` = historical (ClickHouse). No ambiguity. |
| **Independent failure domains** | ClickHouse down = writer stops writing, gateway returns 503 on analytical endpoints. Operational pipeline continues unaffected. |
| **Forward-only migration model** | No down migrations. Corrections are new migrations. Prevents rollback-induced schema corruption. |
| **Deferral discipline** | 3 tables deferred (tradebursts, volumes, fills). 4 writer families deferred. Cold-start bootstrap deferred. Query surface limited to candles only. Each deferral was explicit and justified. |

### 1.3 Knowledge gains

| Gain | Evidence |
|------|----------|
| **ClickHouse operational knowledge** | Container setup, user XML configuration, TTL syntax on DateTime64, native protocol driver behavior. |
| **Schema correction feedback loop** | S146 drafted migrations that diverged from S144 design. S147 caught and corrected them before first application. Validates the multi-stage review approach. |
| **Integration complexity is known** | Write path requires mappers per family. Read path requires per-table query builders. Expansion cost is linear, not exponential. |

---

## 2. Trade-offs

### 2.1 Accepted trade-offs (conscious, documented)

| Trade-off | What was given up | What was gained | Reversal path |
|-----------|-------------------|-----------------|---------------|
| Float64 over Decimal128 | Sub-cent precision on financial values | Simpler mappers, better ClickHouse compression, faster aggregation | Migration to add Decimal128 columns; dual-write period; drop Float64 columns |
| No deduplication | Duplicate rows possible in ClickHouse | Simpler MergeTree engine; no ReplacingMergeTree version column management | Add ReplacingMergeTree or query-time `SELECT DISTINCT` (already documented) |
| JSON strings over native types | No ClickHouse-level indexing on nested fields | Domain events stored as-is; no schema explosion for maps/arrays | Migration to extract indexed columns from JSON when query patterns emerge |
| Forward-only migrations | No automated rollback | Simpler tool; no risk of rollback-induced corruption | Write corrective migrations; manual ClickHouse DDL for emergencies |
| Single query endpoint | Only candle history exposed | Minimal blast radius; proves the pattern before expanding | Add endpoints per table as needed |
| 90-day TTL | Data older than 3 months is lost | Bounded storage; no unbounded growth | Increase TTL via new migration; no data recovery for already-expired rows |

### 2.2 Implicit trade-offs (not explicitly documented, emerged during review)

| Trade-off | What was given up | Impact |
|-----------|-------------------|--------|
| Writer tests skipped | Confidence in write-path correctness | Data quality issues may go undetected; failure behavior is unverified |
| Reader tests skipped | Confidence in read-path correctness | Query results may contain subtle bugs (type conversion, null handling) |
| Single INSERT attempt vs. documented retry | Resilience under transient ClickHouse failures | Batches dropped on first failure; analytical gaps larger than documented |
| Silent mapper fallbacks | Data integrity signals | Parse errors become zero values; JSON errors become empty objects; invisible corruption |
| No observability hooks | Operational visibility into analytical pipeline health | Problems discoverable only via log inspection or ClickHouse row counts |
| Global batch config | Per-family tuning capability | High-frequency families (candles) and low-frequency families (executions) share the same buffer/flush settings |

---

## 3. Open Debts

### 3.1 Testing debts

| Debt | Scope | Priority | Effort |
|------|-------|----------|--------|
| Writer unit tests (mappers, inserter, supervisor) | `cmd/writer/` | **High** | Medium — mappers are deterministic; inserter needs mock ClickHouse client |
| Writer integration test (NATS → ClickHouse) | `cmd/writer/` | **High** | High — requires test containers or compose-based test harness |
| Reader unit tests (query building, row scanning) | `cmd/gateway/analytical_reader.go` | **High** | Low — mock ClickHouse query interface |
| Migration runner integration tests | `internal/migrate/` | Medium | Medium — requires test ClickHouse instance |
| Route registration tests | `internal/interfaces/http/routes/analytical.go` | Low | Low — simple conditional check |

### 3.2 Implementation debts

| Debt | Scope | Priority | Effort |
|------|-------|----------|--------|
| Writer pipeline recovery (restart failed families) | `cmd/writer/supervisor.go` | **High** | Medium — supervisor needs restart loop with backoff |
| INSERT retry with backoff (or document single-attempt as intentional) | `cmd/writer/inserter.go` | **High** | Low — implement backoff or update docs |
| Buffer overflow metrics | `cmd/writer/inserter.go` | Medium | Low — counter increment on eviction |
| Mapper error visibility | `cmd/writer/mappers.go` | Medium | Low — error counters or structured log fields |
| Per-family batch configuration | `cmd/writer/pipeline.go` | Low | Medium — config schema change + pipeline wiring |
| Concurrent migration protection | `internal/migrate/runner.go` | Low | Medium — advisory lock on `_migrations` table |

### 3.3 Schema debts

| Debt | Scope | Priority | Effort |
|------|-------|----------|--------|
| 3 deferred tables (tradebursts, volumes, fills) | `deploy/migrations/` | Low | Low per table — DDL + mapper + consumer |
| 4 deferred writer families | `cmd/writer/` | Low | Low per family — pipeline definition + mapper |
| No secondary indexes | All tables | Low | Low — `ALTER TABLE ADD INDEX` when query patterns emerge |
| No materialized views | ClickHouse | Low | Medium — requires query pattern analysis first |
| No schema evolution test (ALTER path) | `deploy/migrations/` | Medium | Medium — needs a real ALTER migration to prove the path |

### 3.4 Operational debts

| Debt | Scope | Priority | Effort |
|------|-------|----------|--------|
| No write-path observability (counters, latency) | `cmd/writer/` | Medium | Low — structured log counters |
| No read-path observability (query latency, row counts) | `cmd/gateway/` | Low | Low — structured log counters |
| Default credentials in config templates | `deploy/configs/writer.jsonc`, `gateway.jsonc` | Medium | Low — environment variable substitution |
| No ClickHouse backup/restore procedure | Operations | Low | Medium — document and script |
| No analytical smoke test in CI | `scripts/` | Medium | Medium — requires ClickHouse in test environment |

### 3.5 Architectural debts

| Debt | Scope | Priority | Effort |
|------|-------|----------|--------|
| Cold-start bootstrap (derive querying ClickHouse) | Cross-service | **Deferred** | High — requires careful optionality boundary |
| Event schema versioning | Cross-service | **Deferred** | High — acceptable at single-developer scale |
| ClickHouseConfig in shared settings struct | `internal/shared/settings/` | Low | Low — cosmetic; does not affect behavior |
| Cross-table correlation queries | `cmd/gateway/` | **Deferred** | Medium — requires multi-table query builder |

---

## 4. Debt Prioritization Summary

**Must address before expansion:**
1. Writer test coverage
2. Reader test coverage
3. Writer pipeline recovery
4. INSERT failure handling alignment (code vs. docs)

**Should address during hardening:**
5. Buffer overflow metrics
6. Mapper error visibility
7. Migration runner integration tests
8. Basic write-path observability

**Can defer without risk at current scale:**
- Deferred tables and families
- Secondary indexes and materialized views
- Cold-start bootstrap
- Event schema versioning
- Per-family batch configuration
- Cross-table correlation queries
