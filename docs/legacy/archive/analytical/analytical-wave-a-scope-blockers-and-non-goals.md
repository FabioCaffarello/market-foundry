# Analytical Wave A — Scope, Blockers, and Non-Goals

> Defines what is in scope, what blocks expansion, and what is explicitly excluded from Wave A.

---

## 1. Frozen Scope

Wave A scope is frozen as of this document. Changes to scope require an explicit amendment with justification.

### 1.1 In Scope

| Item | Front | Stage | Justification |
|------|-------|-------|---------------|
| Writer mapper unit tests | WC | S152 | Zero coverage on data transformation logic |
| Writer inserter unit tests | WC | S152 | Flush logic untested; buffer overflow untested |
| Writer consumer unit tests | WC | S152 | Deserialization path untested |
| Writer supervisor unit tests | WC | S152 | Pipeline lifecycle untested |
| Reader adapter unit tests | WC | S152 | Query construction and row scanning untested |
| End-to-end integration test | WC | S152 | No proof that NATS → ClickHouse → HTTP works under test |
| INSERT retry alignment (code vs. docs) | FH | S153 | Documented behavior (retry with backoff) diverges from implementation (single attempt) |
| Fix buffer-clear-on-error bug | FH | S153 | Inserter clears buffer on INSERT failure — this is data loss, not a feature |
| Mapper error visibility | FH | S153 | parseFloat and marshalJSON silently produce fallback values |
| Pipeline recovery in supervisor | PR | S154 | Single pipeline failure kills the entire writer process |
| Per-family degraded state | PR | S154 | No way to mark a family as non-functional while others continue |
| Write-path structured counters | OB | S155 | No visibility into event flow rates, latencies, or error rates |
| Read-path structured counters | OB | S155 | No visibility into query performance |
| Periodic summary logging | OB | S155 | No ambient signal of writer health without active inspection |
| Diagnostic script extension | OB | S155 | `diag-check.sh` does not include writer |
| Expansion gate review | EG | S156 | Formal checkpoint before any Wave B work |

### 1.2 Scope Boundary: What "Tests" Means

- **Unit tests** use mocks/stubs for external dependencies (NATS, ClickHouse). They run in `go test` without infrastructure.
- **Integration tests** require running NATS and ClickHouse (via docker-compose). They validate the complete data path.
- Wave A requires **both** — unit tests for coverage breadth, at least 1 integration test for path confidence.
- Wave A does **not** require CI integration for integration tests. Running locally via `make test-integration` is sufficient.

---

## 2. Expansion Blockers

These conditions **must** be met before Wave B (controlled expansion) can begin. Each blocker maps to a specific Wave A exit criterion.

| Blocker ID | Condition | Front | How Verified |
|-----------|-----------|-------|-------------|
| BLK-01 | All 6 writer mappers have unit tests with edge cases | WC | `go test ./cmd/writer/...` passes |
| BLK-02 | Inserter batch logic tested (size, time, overflow, error paths) | WC | Unit tests in `cmd/writer/` |
| BLK-03 | Reader adapter has unit tests for query construction | WC | Unit tests in `cmd/gateway/` or `internal/adapters/clickhouse/` |
| BLK-04 | At least 1 integration test: event → row → HTTP response | WC | Integration test passes against compose stack |
| BLK-05 | INSERT failure behavior matches documentation | FH | Code review + test |
| BLK-06 | Buffer NOT cleared on INSERT failure | FH | Unit test proves rows retained |
| BLK-07 | Mapper errors logged with context | FH | Log output verified in test |
| BLK-08 | Supervisor restarts failed pipelines with backoff | PR | Unit test proves restart |
| BLK-09 | Degraded family visible via `/statusz` | PR | HTTP test or manual verification |
| BLK-10 | Write-path emits per-family counters | OB | Structured log or `/statusz` verification |
| BLK-11 | Gate review passes | EG | S156 report issued with "expand" verdict |

**Rule:** All 11 blockers must pass. There is no partial-pass path to Wave B.

---

## 3. Non-Goals

These items are **explicitly excluded** from Wave A. Each exclusion has a rationale.

### 3.1 Schema and Migration Non-Goals

| Non-Goal | Why Excluded |
|----------|-------------|
| New ClickHouse tables (tradebursts, volumes, fills) | No analytical consumer exists; tables would be empty projections |
| ALTER migrations (schema evolution) | Current schema is sufficient; ALTER path testing is Wave B scope |
| Materialized views | No query patterns justify aggregation; premature optimization |
| Secondary indexes | 2 symbols × 4 timeframes; table scans are fast at current scale |
| Concurrent migration protection | Single-developer scale; advisory locks are Wave B scope |

### 3.2 Endpoint and Query Non-Goals

| Non-Goal | Why Excluded |
|----------|-------------|
| New query endpoints (signals, decisions, executions, strategies, risk-assessments) | Expansion = Wave B; hardening first |
| Cross-table correlation queries | Requires multi-table query builder; complex; no current demand |
| Query caching | Sub-second response times at current volume |
| Pagination support | Max 500 rows per request is sufficient |

### 3.3 Writer Non-Goals

| Non-Goal | Why Excluded |
|----------|-------------|
| Deferred writer families (tradeburst, volume, ema_crossover, venue_market_order) | No corresponding tables or endpoints; adding families without consumers is waste |
| Per-family batch configuration | Global config is adequate at 6 families with similar event rates |
| Dynamic family registration | Families are compile-time; runtime registration adds complexity for no current benefit |
| Deduplication infrastructure | Low event rate; duplicates are rare and tolerable |
| Writer publishing to NATS | Violates INV-03; writer is append-only |

### 3.4 Infrastructure Non-Goals

| Non-Goal | Why Excluded |
|----------|-------------|
| Prometheus metrics export | Structured logs are sufficient for current observability needs |
| Grafana dashboards | No metrics backend; premature for single-operator deployment |
| Alerting rules | No alerting infrastructure; `diag-check.sh` is the current path |
| ClickHouse backup/restore procedure | Operational concern; not a hardening blocker |
| CI integration for integration tests | Local execution is sufficient; CI integration is post-Wave A |

### 3.5 Architectural Non-Goals

| Non-Goal | Why Excluded |
|----------|-------------|
| Cold-start bootstrap (derive queries ClickHouse) | Crosses operational/analytical boundary; Wave C scope |
| Event schema versioning | Single-developer scale; no schema evolution pressure |
| ClickHouseConfig extraction from shared settings | Cosmetic coupling; no behavioral impact |
| Writer as optional module in store | Architectural decision (S145) already made; dedicated service is final |

---

## 4. Risk: Mixing Hardening with Expansion

This section documents why scope discipline matters.

### 4.1 Anti-Patterns

| Anti-Pattern | Why Tempting | Why Dangerous |
|-------------|-------------|---------------|
| "While we're testing mappers, let's add the 4 deferred families" | Feels efficient; similar code | Doubles test surface; delays WC; untested families have no consumers |
| "While we're fixing INSERT retry, let's add deduplication" | Both involve write path | Dedup changes INSERT semantics; mixes correctness fix with new capability |
| "While we're adding observability, let's add Prometheus" | "We'll need it eventually" | Introduces external dependency; scope explosion; delays OB exit |
| "While we're testing the reader, let's add signal/decision endpoints" | "Pattern is established" | Each endpoint needs its own tests; multiplies WC scope by 5× |
| "Pipeline recovery is complex, let's redesign the supervisor" | "If we're touching it anyway" | Redesign is S145 scope, not hardening; introduces new failure modes |

### 4.2 Why Scope Creep Is Especially Dangerous Here

1. **The writer is running in production (docker-compose).** Hardening a live system requires discipline. Adding features while hardening creates moving targets.
2. **Test coverage is at zero.** The first priority is establishing a test baseline. Adding code before tests exist means the test debt grows faster than the test coverage.
3. **Failure handling has a known bug** (buffer cleared on INSERT error). Fixing bugs while adding features risks masking regressions.
4. **The expansion gate exists for a reason.** If hardening and expansion are mixed, the gate becomes meaningless — there is no clean checkpoint.

### 4.3 How to Resist Scope Creep

- **Each stage has a checklist.** If work is not on the checklist, it does not happen in that stage.
- **Each front has a "not in scope" section.** Reference it when tempted.
- **"Add it to Wave B"** is always a valid answer. Wave B exists specifically for controlled expansion.
- **The expansion gate (S156) is mandatory.** It cannot be skipped or combined with the last hardening stage.

---

## 5. Deferred Debts (Explicitly Acknowledged)

These debts exist and are known. They are deferred past Wave A with explicit rationale.

| Debt | Priority | Deferred To | Rationale |
|------|----------|------------|-----------|
| Migration runner integration tests | Medium | Wave B | Runner works; unit tests exist; integration tests are valuable but not blocking |
| Route registration tests | Low | Wave B | Simple conditional check; low risk |
| Float64 precision migration | Low | When precision becomes measurably problematic |
| Default credentials in config templates | Medium | Before any multi-user deployment | Single-developer risk only |
| ClickHouse backup/restore | Low | Before production deployment | Paper-trading scope |
| Analytical smoke test in CI | Medium | Wave B | Local validation is sufficient for now |
| Event schema versioning | Deferred | When schema evolution occurs | No evolution pressure exists |
| Cold-start bootstrap | Deferred | Wave C | Requires stable analytical layer |
