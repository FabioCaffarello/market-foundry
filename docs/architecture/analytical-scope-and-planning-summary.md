# Analytical Scope and Planning Summary

> Consolidated from 6 source documents (archived in docs/archive/analytical/).
> Sources: analytical-wave-a-hardening-plan.md, analytical-wave-a-scope-blockers-and-non-goals.md, analytical-end-to-end-validation-findings.md, analytical-end-to-end-integration-proof.md, analytical-closure-open-vs-closed-items.md, analytical-config-and-startup-validation-hardening.md

---

## 1. Wave A Hardening Summary

Wave A (S152-S156) closed the gap between a structurally correct analytical layer and an operationally reliable one. The S143-S149 wave delivered the projection layer; Wave A hardened it without expanding scope.

### Responsibility Fronts

| Front | Stage | Status |
|-------|-------|--------|
| Writer Correctness (WC) | S152 | Complete -- unit tests for all 6 mappers, inserter, consumer, reader adapter |
| Failure Handling (FH) | S153 | Complete -- INSERT retry with exponential backoff, buffer retention during retry |
| Pipeline Recovery (PR) | S154 | Complete -- supervisor restarts failed pipelines, degraded state management |
| Observability (OB) | S155 | Complete -- write path counters, read path instrumentation (S160) |
| Expansion Gate (EG) | S156 | Complete -- all preconditions met, "expand" verdict issued |

### Success Criteria (All Met)

1. All writer mappers have unit tests with edge cases
2. Inserter batch logic tested (size-triggered, time-triggered, overflow, error)
3. Reader adapter has unit tests for query construction and row scanning
4. Integration test proves NATS -> ClickHouse -> HTTP
5. INSERT failure behavior matches documentation
6. Buffer not cleared on INSERT failure (rows retained)
7. Mapper errors visible in structured logs
8. Supervisor restarts individual failed pipelines with backoff
9. Write path emits per-family structured counters
10. Read path logs query latency and row count
11. Expansion gate review passed

---

## 2. End-to-End Integration Proof

### Method

`scripts/smoke-analytical-e2e.sh` -- automated 7-phase verification:

| Phase | What | How |
|-------|------|-----|
| 1 | Infrastructure readiness | Health/readiness probes |
| 2 | Migration status | All 7 tables exist, 6+ migrations applied |
| 3 | Writer pipeline health | Writer /statusz tracker inspection |
| 4 | ClickHouse data verification | Row count + sample query |
| 5 | Reader -> HTTP query surface | curl + response structure validation |
| 6 | Error handling | Invalid params return 400 (3 negative tests) |
| 7 | Writer observability | No degraded pipelines |

### Acceptance Criteria

| Criterion | Metric |
|-----------|--------|
| Write path proven | evidence_candles row count > 0 |
| Read path proven | HTTP response contains candles with source="clickhouse" |
| Response structure correct | All 12 required candle fields present |
| Error handling correct | 400 for missing timeframe, invalid limit, since>until |
| No degraded pipelines | Writer /diagz shows 0 degraded families |

### Integration Targets

| Target | Description |
|--------|-------------|
| `make smoke-analytical` | Analytical E2E proof |
| `make smoke` | Operational path proof (unchanged) |
| `make smoke-multi` | Multi-symbol proof (unchanged) |

---

## 3. Validation Findings

### Resolved During Validation

| ID | Finding | Resolution |
|----|---------|------------|
| F-01 | Writer not in BUILDABLE_SERVICES | Added to Makefile |
| F-02 | lib.sh missing writer service | Added to ALL_SERVICES and SVC_PORTS (port 8085) |
| F-03 | No standalone analytical smoke target | Added `make smoke-analytical` |

### Accepted Limitations

| ID | Finding | Status |
|----|---------|--------|
| F-05 | Migration application is manual | Accepted -- deliberate, auditable step |
| F-06 | Batch flush timing creates test non-determinism | Polling with configurable timeout (default 120s) |
| F-07 | Non-candle families not exercised by read path proof | Accepted -- candle path is representative; other families use identical machinery |
| F-08 | Gateway readiness independent of ClickHouse | By design (R-02 compliance) |

---

## 4. Config and Startup Validation

### Writer Startup Sequence

| Step | Check | Failure |
|------|-------|---------|
| 1 | `nats.enabled == true` | Hard exit |
| 2 | `ClickHouseConfig.ValidateForWriter()` | Hard exit (checks addr, database, username, batching) |
| 3 | `PipelineConfig.ValidateForWriter()` | Hard exit (standard rules + at least one family) |
| 4 | Log validated config summary | -- |
| 5 | `clickhouse.Open()` | Hard exit on connection failure |
| 6 | `buildTrackers()` | Hard exit if no enabled families |
| 7 | Spawn supervisor | -- |

### ValidateForWriter() Rules

| Field | Rule |
|-------|------|
| `addr` | Must not be empty |
| `database` | Must not be empty |
| `username` | Must not be empty |
| `password` | Not validated (empty may be intentional) |
| `batch_size` | Must not be negative |
| `max_pending` | Must not be negative |
| `max_retries` | Must not be negative |
| `flush_interval` | Must be valid Go duration if set |
| `initial_backoff` | Must be valid Go duration if set |

All issues aggregated into a single `problem.Problem` response -- operators fix everything in one pass.

### Gateway Analytical Client Validation

1. `addr` empty -> log info, return nil (analytical disabled)
2. `addr` set but config invalid -> log warning, return nil (gateway continues)
3. Config valid but connection fails -> log warning, return nil
4. Connection succeeds -> log success with addr and database

Gateway never hard-exits on ClickHouse problems.

### What Is NOT Validated at Startup

| Item | Reason |
|------|--------|
| ClickHouse schema existence | Validated at query time; migrations are separate |
| NATS stream/consumer existence | Created at runtime by consumer actor |
| Network reachability | Validated by `clickhouse.Open()` |
| Password correctness | Validated at connection time |

---

## 5. Closure Status

### Closed Items

| # | Area | Item |
|---|------|------|
| C-01 | Reader | All 6 compile-time interface assertions added |
| C-02 | Tooling | Compiled binary removed from git, .gitignore updated |
| C-03 | Scripts | Writer added to live-pipeline health/readiness/diagnostics |
| C-04 | Scripts | Analytical endpoints validated in live-pipeline-activate |
| C-05 | Scripts | Gateway port added to SVC_PORTS map |
| C-06 | Writer | No TODOs/FIXMEs in writer service |
| C-07 | Reader | All 6 adapters, use cases, handlers, routes complete (205+ tests) |
| C-08 | Schema | All 7 migrations present and consistent |
| C-09 | Config | Settings validation for writer complete |
| C-10 | NATS | All 7 families consistently registered |
| C-11 | Gateway | Analytical wiring with optional ClickHouse lifecycle correct |
| C-12 | CI | Codegen golden equivalence + analytical E2E smoke in CI |
| C-13 | Diagnostics | diag-check.sh includes writer |

### Frozen Items (Consciously Deferred)

| # | Area | Item | Risk |
|---|------|------|------|
| F-01 | Dependencies | clickhouse-go version mismatch (v2.30.0 vs v2.43.0) | Low -- separate binaries |
| F-02 | Writer | No backpressure between consumer and inserter actors | Medium -- potential silent loss under sustained load |
| F-03 | Writer | parseFloat/marshalJSON default to 0/"{}" on error | Low -- analytical data |
| F-04 | Writer | Hardcoded 30s ClickHouse insert timeout | Low |
| F-05 | Writer | No per-family batch tuning | Low |
| F-06 | Migrations | No transaction wrapping for migration + metadata | Medium -- partial state possible on crash |
| F-07 | Migrations | 60s context timeout hardcoded | Low |
| F-08 | Smoke | Only tests 60s timeframe | Low |
| F-09 | Smoke | Hardcoded credentials across config/scripts | Low -- local only |
| F-10 | Tests | No integration tests for migration runner | Low -- CI provides indirect coverage |
| F-11 | Tests | No actor integration tests for writer | Medium -- failure recovery untested |
| F-12 | CI | Codegen integrated check doesn't block merge | Low |

---

## 6. Non-Goals (Explicitly Excluded)

### Schema and Migration

- No new ClickHouse tables (tradebursts, volumes, fills)
- No ALTER migrations (schema evolution)
- No materialized views, secondary indexes
- No concurrent migration protection

### Endpoints and Queries

- No new query endpoints (signals, decisions, etc. are Wave B)
- No cross-table correlation queries
- No query caching or pagination support

### Writer

- No deferred families (tradeburst, volume, ema_crossover, venue_market_order)
- No per-family batch configuration
- No dynamic family registration or deduplication
- No writer publishing to NATS (violates INV-03)

### Infrastructure

- No Prometheus, Grafana, or alerting rules
- No ClickHouse backup/restore procedure
- No CI integration for integration tests (local sufficient)

### Architectural

- No cold-start bootstrap (derive queries ClickHouse)
- No event schema versioning
- No ClickHouseConfig extraction from shared settings

---

## 7. Deferred Debts

| Debt | Priority | Deferred To |
|------|----------|------------|
| Migration runner integration tests | Medium | Wave B |
| Route registration tests | Low | Wave B |
| Float64 precision migration | Low | When measurably problematic |
| Default credentials in config templates | Medium | Multi-user deployment |
| ClickHouse backup/restore | Low | Production deployment |
| Analytical smoke test in CI | Medium | Wave B |
| Event schema versioning | Deferred | When evolution occurs |
| Cold-start bootstrap | Deferred | Wave C |

---

## 8. Expansion Readiness

Wave B expansion was unblocked after all 11 success criteria were met and the expansion gate (S156) issued "expand" verdict. The system provides:

- **Clear reader expansion point** -- new family readers go in `internal/adapters/clickhouse/` alongside `candle_reader.go`
- **Symmetric observability** -- both write and read paths have trackers
- **Config validation** -- new families validated at startup
- **Schema contract** -- 3-point coordination rule (DDL + mapper + reader)
- **Integration test pattern** -- new families extend the test skeleton
