# Pre-Wave-B Analytical Gains, Trade-offs, and Open Debts

## Purpose

Honest accounting of what was gained, what was traded, and what remains unresolved across the post-Wave-A hardening sequence (S157–S161). This document exists so that Wave B decisions are informed by reality, not by the momentum of completed work.

---

## Gains

### G1. Responsibility clarity (S157)

Complete responsibility map covering all 6 analytical components. Five boundary issues identified and prioritized. Eight design patterns validated. Ten non-goals explicitly deferred with rationale.

**Concrete impact:** New contributors can understand who owns what without reading implementation. Expansion decisions have a documented framework.

### G2. Boundary hardening (S158)

Reader extracted from gateway composition root to adapter layer. Compile-time interface assertion prevents silent drift. Writer config validation enforces field-level correctness at startup before I/O.

**Concrete impact:** Adding a new read-path adapter follows a documented, mechanically repeatable pattern. Invalid writer configuration fails immediately with actionable error messages instead of causing silent runtime failures.

### G3. Integration proof (S159)

Automated 7-phase end-to-end validation covering infrastructure readiness, migration status, writer pipeline health, ClickHouse data persistence, reader→HTTP query surface, error handling (3 negative cases), and writer observability.

**Concrete impact:** The complete analytical data path (NATS → writer → ClickHouse → reader → HTTP) is proven with evidence. Boundary coherence across all 5 integration segments is verified. Regression detection is possible via `make smoke-analytical`.

### G4. Read path observability (S160)

Three-layer instrumentation: adapter (wall-clock timing, error logging), use case (duration, row count, QueryMeta), HTTP handler (Server-Timing header, meta in JSON response). Response contract extended. Logger propagation consistent.

**Concrete impact:** Read path problems are now visible to operators before users discover them. Query performance is measurable without external tooling. Runbook has 5 failure scenario playbooks.

### G5. Startup robustness (S161)

Fail-fast 3-phase startup: config validation (no I/O) → connections → run. Writer-specific validation with aggregated, field-level error messages. Gateway gracefully disables analytical on invalid ClickHouse config.

**Concrete impact:** Misconfiguration is caught at deploy time, not at first request time. Error messages name the offending field and the required fix.

### G6. Documentation discipline

11 new architecture documents, 6 stage reports, all with explicit gap sections. Every stage documented what was NOT covered alongside what was delivered. Anti-patterns cataloged and traced to specific files.

**Concrete impact:** Honest accounting prevents false confidence. Gap tracking ensures nothing is silently dropped.

---

## Trade-offs

### T1. Script-based integration proof, not Go integration tests

**What was traded:** Integration validation is a bash script against a live Docker stack, not a Go test binary with programmatic assertions.

**Why:** Compose-based Go integration tests require test infrastructure investment disproportionate to current needs. The script validates the same boundary contracts.

**Consequence:** Integration proof cannot run in `go test`. CI integration requires Docker-in-Docker or equivalent.

**Acceptable because:** Script covers all 5 boundary segments. Pattern is repeatable. CI integration is recommended for early Wave B, not blocked on it.

### T2. Structured logging, not metrics

**What was traded:** No Prometheus counters, no OpenTelemetry spans, no push-based alerting for the read path.

**Why:** External observability tooling is infrastructure overhead disproportionate to single-operator scale. Structured logs with Server-Timing headers provide equivalent operational visibility at current query volume.

**Consequence:** Trend analysis requires log aggregation. No automated alerting on read path degradation.

**Acceptable because:** Current operational model is single-operator. Pull-based observability is adequate. This trade-off should be revisited if the team grows or analytical query volume increases by 10x.

### T3. Reviewer-enforced schema coherence, not compile-time validation

**What was traded:** Column order alignment between DDL, write-path mappers, and read-path adapters is verified by code review, not by a shared constant or codegen.

**Why:** Go's type system does not express column-order constraints. A shared constant package would add import coupling between independently deployable binaries.

**Consequence:** Schema drift is possible if a reviewer misses a column change. Risk scales linearly with table count.

**Acceptable because:** 6 tables, low change frequency. Integration test catches gross misalignment. Trade-off should be revisited at ~12 tables or if schema changes become frequent.

### T4. Sticky degradation, not auto-recovery

**What was traded:** Degraded pipeline families remain degraded until process restart. No automatic recovery attempt after restart budget is exhausted.

**Why:** Auto-recovery adds complexity and risk of infinite restart storms. Simple mental model: 5 attempts, then stop. Operator decides when to restart.

**Consequence:** Extended outages require manual intervention. No self-healing under sustained ClickHouse failure.

**Acceptable because:** Pipeline family count is small (6). Process restart resets budget. At current scale, operator intervention is fast and low-cost.

### T5. Candle-only read path, not all families

**What was traded:** Only evidence_candles has a read-path adapter and HTTP endpoint. Signals, decisions, strategies, risks, and executions are write-only.

**Why:** Expanding all readers simultaneously multiplies work without validating the expansion pattern. Candle-first proves the pattern; other families follow mechanically.

**Consequence:** 5 of 6 analytical tables are append-only with no query surface. Direct ClickHouse access is the only way to read them.

**Acceptable because:** No consumer of non-candle analytical data exists yet. Adding readers when demand appears is straightforward and follows the documented expansion protocol.

### T6. Per-family counters, not per-request metrics

**What was traded:** Writer observability tracks aggregate counters (events_received, events_flushed, events_dropped), not per-request latency distributions.

**Why:** Per-request metrics require histogram infrastructure not justified at current throughput.

**Consequence:** Latency outliers are invisible. Only `flush_duration_ms` gauge provides batch-level timing.

**Acceptable because:** Batch-level timing is sufficient for detecting degradation. Per-request granularity would be valuable only under high-throughput scenarios not yet present.

---

## Open Debts

### D1. CI integration of smoke-analytical (Medium priority)

`scripts/smoke-analytical-e2e.sh` exists but is not wired into any CI pipeline. Regressions are detectable only by manual execution.

**Risk if ignored:** Integration regressions land silently. Discovered only during next manual smoke run.

**Recommendation:** Wire into CI early in Wave B, before second family expansion.

### D2. No backoff jitter (Low priority)

Writer retry backoff is deterministic (1s, 2s, 4s, 8s, 16s). Multiple pipelines retrying simultaneously create a thundering herd on ClickHouse recovery.

**Risk if ignored:** Low at 6 families. Increases if family count grows or if ClickHouse becomes load-sensitive.

**Recommendation:** Add random jitter during Wave B. Trivial change, disproportionate benefit.

### D3. No NATS consumer lag visibility (Medium priority)

Writer consumes from NATS JetStream durables but does not expose consumer lag. Lag buildup is invisible until buffer overflow.

**Risk if ignored:** Slow consumers are detected only via buffer_depth or overflow counters — lagging indicators.

**Recommendation:** Address during Wave B if throughput increases. Not blocking for initial expansion.

### D4. No connection pool monitoring (Low priority)

ClickHouse driver manages connection pooling internally. No visibility into pool state, idle connections, or connection churn.

**Risk if ignored:** Connection exhaustion would manifest as query timeouts with no diagnostic signal.

**Recommendation:** Defer unless connection-related issues appear in practice.

### D5. No load testing (Medium priority)

No performance baseline established. Writer batch throughput, ClickHouse insert latency, and reader query latency under load are unknown.

**Risk if ignored:** Performance problems discovered in production instead of staging.

**Recommendation:** Establish baseline before third Wave B family expansion. Not required for initial controlled expansion.

### D6. Non-candle families untested end-to-end (Low priority)

Integration proof covers candles only. Other families use identical machinery but have no E2E validation.

**Risk if ignored:** Low — same consumer/inserter/supervisor machinery. Family-specific bugs would be in mappers (which have unit tests).

**Recommendation:** Extend smoke-analytical to cover first Wave B family as part of that stage's validation.

### D7. No chaos testing (Low priority)

No validation of behavior under ClickHouse restart, network partition, NATS disconnect, or partial infrastructure failure.

**Risk if ignored:** Recovery semantics are proven in unit tests but unvalidated under real failure conditions.

**Recommendation:** Defer to post-Wave-B. Current scale does not justify chaos testing investment.

### D8. Gateway readiness does not reflect analytical health (By design)

Gateway `/readyz` checks NATS + configctl + store, NOT ClickHouse. Analytical endpoint unavailability is invisible to readiness probes.

**Risk if ignored:** Orchestration (k8s, compose healthcheck) cannot detect analytical degradation.

**Recommendation:** This is an intentional design decision (R-02: operational baseline must not depend on analytics). Document in runbook: "gateway ready ≠ analytical ready." Revisit only if analytical becomes critical-path.

---

## Debt Trajectory

| Category | Post-Wave-A (S156) | Post-Hardening (S161) | Change |
|---|---|---|---|
| S156 preconditions | 3 open | 0 open | All resolved |
| Boundary issues (S157) | 5 identified | 5 addressed | All corrected or documented |
| Integration testing | No automation | 7-phase script | Major improvement |
| Read path observability | Zero instrumentation | Three-layer, Server-Timing | Major improvement |
| Startup validation | Absent for writer | Fail-fast, field-level | Major improvement |
| CI integration | None for analytical | Still none | **Unchanged — debt carried** |
| Load testing | None | None | **Unchanged — debt carried** |
| Consumer lag visibility | None | None | **Unchanged — debt carried** |

---

## Items Not Worth the Cost Now

These were considered and explicitly rejected:

| Item | Why Not Now |
|---|---|
| Prometheus/Grafana integration | Infrastructure overhead exceeds single-operator benefit |
| Dead-letter queue | Events remain in NATS JetStream; manual replay sufficient |
| Distributed tracing (OpenTelemetry) | Structured logs + healthz sufficient at current scale |
| Shared column constants package | Import coupling between independent binaries; review discipline adequate at 6 tables |
| Auto-recovery from degraded | Complexity and restart storm risk exceed benefit at small family count |
| Per-family batch configuration | Global config adequate; all families share similar throughput patterns |
| Event schema versioning | Writer and migrate deployed together; no independent schema evolution needed |
| Cold-start bootstrap from NATS replay | Bounded by 90-day TTL; manual replay sufficient |
| Concurrent migration protection | Single-operator deployment; no concurrent scenario exists |
| Materialized views | Not needed at current data volume; premature optimization |
