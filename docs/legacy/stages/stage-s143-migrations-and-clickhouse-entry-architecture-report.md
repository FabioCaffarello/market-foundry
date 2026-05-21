# Stage S143 — Migrations and ClickHouse Entry Architecture

## Stage Identity

| Field | Value |
|---|---|
| Stage | S143 |
| Title | Migrations and ClickHouse Entry Architecture |
| Predecessor | S142 (Post-Consolidation Readiness Review and ClickHouse Preparation Gate) |
| Scope | Architectural definition — no implementation |
| Status | **Complete** |

---

## 1. Executive Summary

S143 defines the formal entry architecture for ClickHouse and migration infrastructure in Market Foundry. It resolves the architectural gap identified by S142 — the system had planning-level governance (principles, conventions, signal catalog) but lacked the design-level clarity needed to implement migration tooling, schema design, and the writer service.

**What this stage delivers:**

- The role of ClickHouse as an analytical projection layer — boundary between operational and analytical layers is explicit
- The writer as a standalone service (`cmd/writer/`) following the same dual-actor consumer pattern as `store`
- `cmd/migrate` as a standalone schema management tool with four commands (up, dry-run, status, validate) and forward-only semantics
- The migration catalog at `deploy/migrations/` with self-bootstrapping metadata (migration 000)
- 10 codified optionality rules (R-01 through R-10) with enforcement mechanisms and violation severity
- 8 architectural invariants (INV-01 through INV-08) that must hold across all implementation phases

**What this stage does NOT deliver:**

- No code. No DDL. No new services. No compose changes.
- No schema design (deferred to Phase 2)
- No event schema versioning mechanism (deferred)
- No query surface design (deferred to Phase 4)

---

## 2. Pre-Condition Resolution

S142 identified 4 critical pre-conditions (PC-01 through PC-04) blocking implementation. S143 addresses them at the design level:

| Pre-Condition | S142 Status | S143 Resolution |
|---|---|---|
| PC-01: Migration tool exists | NOT STARTED | **Architecture defined.** `cmd/migrate` design complete: commands, execution semantics, failure handling, self-bootstrapping metadata. Ready for implementation. |
| PC-02: Core tables schema designed | NOT STARTED | **Derivation rules defined.** Schema-follows-events rule, type mapping convention, partitioning strategy, retention policy. Table DDL writing is Phase 2 work. |
| PC-03: Writer architecture decided | NOT STARTED | **Architecture defined.** Standalone `cmd/writer/` service, dual-actor pattern (consumer + inserter), batch buffering strategy, ClickHouse-down tolerance, health model. Ready for implementation. |
| PC-04: ClickHouse remains optional | PASSED | **Codified.** 10 optionality rules (R-01 through R-10) with enforcement mechanisms. Structural, runtime, and process-level enforcement. |

---

## 3. Architectural Decisions Made

### AD-01: Writer Is a Standalone Service

**Decision:** The ClickHouse writer is `cmd/writer/`, a separate service, not an optional module inside `store`.

**Rationale:** Failure isolation (writer crash can't affect KV projection), lifecycle independence (start/stop independently), dependency clarity (only writer depends on ClickHouse), and clean optionality (remove from compose = gone).

**Trade-off:** One more service to deploy and monitor. Acceptable at current scale.

### AD-02: Forward-Only Migrations

**Decision:** `cmd/migrate` supports `up` only. No `down` command. No automatic rollback.

**Rationale:** ClickHouse's MergeTree is append-only — `DROP TABLE` is destructive and irreversible for data. Forward-only is simpler, safer, and auditable. Corrections are new migrations.

**Trade-off:** Reverting a bad migration requires writing and applying a new one. Acceptable — this is the safer failure mode.

### AD-03: Self-Bootstrapping Metadata

**Decision:** The `_migrations` table is managed as migration 000. The tool bootstraps itself on first run.

**Rationale:** Avoids chicken-and-egg problem. The metadata table is a migration like any other, with checksum tracking and idempotency.

### AD-04: Batch Buffering with Drop Policy

**Decision:** Writer buffers events in memory (default: 1000 events / 5s flush) and drops oldest events when buffer exceeds `max_pending` (default: 10000) during ClickHouse outage.

**Rationale:** Never block NATS consumer advancement. Dropped events can be replayed from NATS stream retention (72h) if needed. OOM prevention is more important than guaranteed delivery to an analytical store.

### AD-05: No Conditional ClickHouse Paths in Operational Services

**Decision:** Operational services have zero ClickHouse awareness. No `if clickhouseEnabled` branches. Historical query endpoints are new routes, not conditional modifications.

**Rationale:** Conditional paths create testing combinatorics and tend to degrade. The boundary is structural (separate service), not conditional (feature flag).

---

## 4. Boundaries Defined

### 4.1 Operational Layer vs. Analytical Layer

| Aspect | Operational Layer | Analytical Layer |
|--------|------------------|-----------------|
| Components | ingest, derive, store, execute, gateway, configctl | writer, cmd/migrate, ClickHouse |
| Persistence | NATS JetStream + KV | ClickHouse tables |
| Query model | Latest value (KV) | Historical range (SQL) |
| Availability requirement | Must always run | Optional — can be stopped |
| Coupling | Zero ClickHouse awareness | Consumes from NATS (read-only) |

### 4.2 cmd/migrate vs. cmd/writer

| Aspect | cmd/migrate | cmd/writer |
|--------|-------------|------------|
| Purpose | Schema management | Data ingestion |
| Runs | On-demand (developer/CI) | Continuously (as service) |
| Dependencies | ClickHouse only | NATS + ClickHouse |
| Lifecycle | Before services start | While pipeline runs |
| Shared code | None | None |

### 4.3 Store vs. Writer

| Aspect | store | writer |
|--------|-------|--------|
| Source | NATS events | NATS events (same streams) |
| Destination | NATS KV (latest) | ClickHouse (history) |
| Consumer names | `store-*` | `writer-*` |
| Mutual awareness | None | None |
| Failure impact | Gateway loses latest values | ClickHouse loses events (buffered/dropped) |

---

## 5. Invariants and Anti-Patterns

### 5.1 Invariants (8)

| ID | Invariant |
|----|-----------|
| INV-01 | Pipeline functions without ClickHouse |
| INV-02 | No service except writer depends on ClickHouse |
| INV-03 | Writer never publishes to NATS |
| INV-04 | Writer uses own durable consumer names (`writer-*`) |
| INV-05 | Schema follows events (Go struct → DDL) |
| INV-06 | All schema changes go through migrations |
| INV-07 | Existing endpoints unchanged by ClickHouse presence |
| INV-08 | Writer tolerates ClickHouse downtime (buffer + drop) |

### 5.2 Anti-Patterns (Consolidated)

| Anti-Pattern | Violates | Prevention |
|---|---|---|
| Making ClickHouse a startup dependency | P-01, R-01 | Structural: separate service |
| Manual DDL in ClickHouse shell | P-02, INV-06 | `migrate validate` catches drift |
| Writer publishing back to NATS | P-05, INV-03 | Code review |
| Shared consumer names between store and writer | R-04, INV-04 | Naming convention enforcement |
| Synchronous ClickHouse writes on hot path | P-07, R-03 | Writer is separate process |
| Editing applied migration files | V-02 | Checksum verification |
| Conditional `if clickhouseEnabled` in operational services | R-07 | Import-level prevention |
| Migration with domain data insertion | Catalog rules | Code review |
| Using multiple migration tools | P-03, V-03 | Single `cmd/migrate` binary |
| Cold-start bootstrap blocking derive startup | R-09 | Timeout + fallback |

---

## 6. Implementation Sequencing (Refined)

S143 refines the 5-phase sequence from S142 into concrete stage-sized deliverables:

| Phase | Recommended Stage | Scope | Depends On | Key Deliverable |
|-------|-------------------|-------|------------|----------------|
| **Phase 1** | S144 | Migration infrastructure | S143 (this stage) | `cmd/migrate` binary + `deploy/migrations/000` |
| **Phase 2** | S145 | Core schema design | Phase 1 | 9 DDL migration files (001–009) |
| **Phase 3** | S146 | Writer service | Phase 1 + Phase 2 | `cmd/writer/` service in compose |
| **Phase 4** | S147+ | Historical query surface | Phase 3 | Gateway `/history` endpoints |
| **Phase 5** | S148+ (conditional) | Cold-start bootstrap | Phase 3 | Derive ClickHouse bootstrap path |

**Phase 1 (S144) is the recommended next step.** It is self-contained, independently valuable, and unblocks all subsequent phases.

### Phase 1 Success Criteria (S144 Preview)

| Criterion | Verification |
|-----------|-------------|
| `cmd/migrate` binary compiles and runs | `go build ./cmd/migrate` |
| `migrate up` creates `_migrations` table on fresh ClickHouse | Run against empty CH |
| `migrate up` is idempotent | Run twice; second run is no-op |
| `migrate up --dry-run` shows pending without applying | Verify no tables created |
| `migrate status` lists applied and pending migrations | After partial apply |
| `migrate validate` detects checksum drift | Modify applied file; verify exit 1 |
| Makefile targets work | `make migrate-up`, `make migrate-status` |
| No NATS dependency | Run with NATS stopped |
| Smoke tests still pass | `smoke-first-slice.sh` unaffected |

---

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Event schema changes during implementation | Medium | Migration needed for each change | Schema-follows-events rule; schema should be designed after events stabilize |
| Writer performance insufficient for burst events | Low | Data lag in ClickHouse | Batch buffering absorbs bursts; ClickHouse handles high ingest natively |
| ClickHouse disk usage grows unbounded | Medium | Disk full | TTL enforcement per table (30–365 days) |
| Scope creep during implementation phases | High | Delays infrastructure delivery | Each phase has explicit deliverables and out-of-scope list |
| `cmd/migrate` becomes too complex | Medium | Maintenance burden | Minimal design: 4 commands, no rollback, no plugins |
| Developer runs manual DDL and bypasses migrations | Medium | Schema drift | `migrate validate` detects drift; CI enforcement (future) |

---

## 8. Out of Scope

The following are explicitly outside S143 and all implementation phases until specifically scheduled:

| Item | Why Deferred |
|------|-------------|
| Event schema versioning mechanism | Low urgency at single-developer scale; implicit versioning via Go structs sufficient |
| Materialized views and pre-aggregations | Optimization; needs query patterns first |
| ClickHouse clustering or replication | Single-node sufficient for dev/paper trading |
| Grafana or dashboard integration | Consumer of query surface; needs endpoints first |
| Backup and disaster recovery | Development environment; NATS is source of truth |
| Multi-environment deployment | Single environment currently |
| ClickHouse RBAC and user management | Default user sufficient for development |
| Automated CI pipeline validation | Manageable manually at current scale |
| Real-time alerting from ClickHouse | No operational consumers yet |
| Performance benchmarking | Premature; needs data volume first |

---

## 9. Deliverables

| # | Document | Path |
|---|---|---|
| 1 | ClickHouse Entry Architecture | `docs/architecture/clickhouse-entry-architecture.md` |
| 2 | Migrations Infrastructure Architecture | `docs/architecture/migrations-infrastructure-architecture.md` |
| 3 | Analytical Runtime Optionality Rules | `docs/architecture/analytical-runtime-optionality-rules.md` |
| 4 | Stage Report (this document) | `docs/stages/stage-s143-migrations-and-clickhouse-entry-architecture-report.md` |

---

## 10. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|---|---|---|
| ClickHouse entry is architecturally clear | **PASS** | Role defined (analytical projection), boundaries explicit (operational vs. analytical layer), data flow diagrams provided |
| Role of `cmd/migrate` is explicit | **PASS** | 4 commands, execution semantics, failure handling, self-bootstrapping metadata, Makefile integration |
| Runtime optionality preserved by design | **PASS** | 10 rules (R-01 through R-10), 3 enforcement levels (structural, runtime, process), violation severity matrix |
| Sequencing of next stages is delimited | **PASS** | 5 phases with dependencies, recommended next stage (S144) with preview success criteria |
| Base ready for schema design without improvisation | **PASS** | Type mapping convention, partitioning strategy, retention policy, naming conventions, file structure template |
| No ClickHouse implementation opened | **PASS** | No code, no DDL, no compose changes |
| No mandatory baseline dependency created | **PASS** | R-01 through R-10 prevent this structurally |
| Migrations not mixed with business config | **PASS** | `cmd/migrate` has no NATS dependency, no config lifecycle interaction |
| Writer not coupled to existing services | **PASS** | AD-01: standalone `cmd/writer/`, independent consumer names, separate compose service |
| Out of scope clearly documented | **PASS** | 10 items explicitly deferred with rationale |

---

## 11. Preparation for S144

S144 should implement Phase 1: Migration Infrastructure. The architecture is fully specified in `migrations-infrastructure-architecture.md`. The entry criteria for S144 are:

| Entry Criterion | Status |
|---|---|
| `cmd/migrate` architecture defined | PASSED (S143) |
| Self-bootstrapping metadata designed | PASSED (S143, migration 000) |
| Makefile targets specified | PASSED (S143) |
| Success criteria defined | PASSED (S143, section 6 of this report) |
| ClickHouse container available | PASSED (existing in docker-compose) |
| No prior implementation to conflict with | PASSED (greenfield) |

**S144 scope:** Build `cmd/migrate` + `internal/migrate/` + `deploy/migrations/000_create_migrations_metadata.sql`. Validate with the success criteria from section 6. Do not design domain tables (that is Phase 2 / S145).
