# Wave B Iteration 01 — Gains, Trade-offs, and Open Debts

## Scope

This document covers the first Wave B iteration: Signal (RSI) family expansion (S163–S166), assessed as a complete unit from pattern definition through CI integration.

---

## 1. Gains

### G-1: The 9-artifact expansion unit works as designed

The canonical pattern (schema → writer → reader → gateway → tests → smoke → gate) produced a functioning family expansion with zero writer changes and a complete read path. Every artifact was delivered. No shortcuts were taken.

**Why this matters:** The pattern is not theoretical — it was exercised under real constraints and produced a verifiable result. The expansion unit is the correct granularity for Wave B.

### G-2: Schema coherence is testable without ClickHouse

Unit tests assert row length in mappers and column count in exported query builders. This catches DDL/code drift at compile time, not at runtime. No running ClickHouse instance is needed for coherence validation.

**Why this matters:** Schema coherence is the highest-risk area in a multi-location schema model. Having a fast, local verification mechanism reduces the cost of expanding by removing the need for infrastructure in the inner development loop.

### G-3: Write path was future-proof

The writer service already consumed RSI signals (pipeline registered in Wave A). The entire Signal family expansion was read-path only. This validates that the writer's pipeline architecture supports expansion without modification.

**Why this matters:** If the write path had required changes for the first expansion, the pattern's additive-only constraint (C-9) would have been violated immediately. The write path's stability is a structural advantage.

### G-4: Observability parity is automatic

The inserter/supervisor infrastructure provides per-pipeline health tracking, event counters, and degraded-state detection for every family without family-specific instrumentation. The read path follows a mechanical pattern: adapter timing → use case logging → Server-Timing header.

**Why this matters:** Observability parity is the constraint most likely to be skipped under time pressure. Making it automatic via infrastructure (write) and mechanical via pattern (read) removes the temptation.

### G-5: CI gates merges before expansion continues

GitHub Actions runs unit tests and smoke-analytical on every push and PR. The S162 constraint (C-3: CI before second family) is satisfied. Local and CI execution are identical (`make smoke-analytical`).

**Why this matters:** CI is the only automated enforcement mechanism for the expansion pattern. Without it, regressions would be discovered manually — which means they would be discovered late.

### G-6: Optionality invariant preserved

The gateway starts without ClickHouse. Analytical routes return 503 when unavailable. Operational routes are unaffected. This was verified by smoke tests (Phase 1) and by the gateway's conditional route registration.

**Why this matters:** The analytical layer remains genuinely optional. No operational service depends on it. This is the single most important architectural invariant in the system — losing it would convert analytical expansion into operational risk.

---

## 2. Trade-offs

### T-1: Mechanical duplication accepted over premature abstraction

~80% of code is identical between candle and signal paths at every layer (adapter, use case, handler, route, test, smoke). Each family is a near-copy with different field names and types.

**Given up:** DRY principle; any cross-cutting fix must be applied to each family independently.
**Gained:** Each family is fully independent; a bug in signal handling cannot affect candle handling. No shared abstraction to maintain or evolve.
**Acceptable because:** At 2 families, duplication cost is low. The commitment to evaluate codegen at family 4 provides an exit ramp. Premature abstraction at this point would ossify a pattern that is still being learned.

### T-2: Manual schema coherence verification accepted over compile-time enforcement

Schema knowledge lives in three locations (DDL, writer mapper, reader adapter). Alignment is verified by unit test assertions and code review, not by a shared type definition or code generation.

**Given up:** Guaranteed correctness at compile time; any column reordering or type change requires manual updates in 3 places.
**Gained:** Simplicity; no build tooling dependency; each layer can evolve its representation independently if needed.
**Acceptable because:** At 6–7 tables, the review cost is manageable. The unit test assertions catch the most common errors (wrong column count, wrong row length). Revisit at ~12 tables.

### T-3: Monolithic smoke test accepted over parameterized validation

The smoke test is a single script with linearly growing phases. Each family adds ~50 lines following the same structure but with different endpoints and assertions.

**Given up:** Per-family isolation in smoke; reusability; maintainability at scale.
**Gained:** Simplicity; sequential execution with clear phase boundaries; easy to read and debug.
**Acceptable because:** At 2 families, the script is ~400 lines — manageable. The committed extraction of `validate_analytical_family()` at family 3 provides the refactoring trigger.

### T-4: Sticky degradation accepted over auto-recovery

When ClickHouse becomes unavailable, the writer's supervisor exhausts its restart budget (5 attempts with exponential backoff) and enters degraded state. Recovery requires manual process restart. The gateway returns 503 for analytical endpoints until restarted with a live ClickHouse.

**Given up:** Automatic recovery from transient ClickHouse outages; self-healing pipeline.
**Gained:** Simple mental model; no reconnection bugs; no partial-state recovery logic.
**Acceptable because:** At the current scale (single operator, small family count), manual intervention is practical. Auto-recovery adds significant complexity (connection pooling, health probes, state reconciliation) that is not justified yet.

### T-5: Silent mapper fallbacks accepted over strict parsing

Writer mappers use `parseFloat` with 0.0 fallback on parse errors and `marshalJSON` with `"{}"` fallback. Invalid data is written as zero values or empty JSON, not rejected.

**Given up:** Data integrity guarantee; ability to detect upstream corruption at write time.
**Gained:** Pipeline resilience; a single malformed event does not halt ingestion.
**Acceptable because:** The writer logs WARN on fallback. ClickHouse data is append-only and TTL-bounded (90 days). The analytical layer is not authoritative — it is observational. Zero values in analytical queries are preferable to pipeline stalls.

### T-6: No backoff jitter accepted for now

Writer retry uses deterministic exponential backoff (1s, 2s, 4s, 8s, 16s). Multiple pipelines recovering simultaneously will hit ClickHouse at the same instant.

**Given up:** Thundering herd prevention on recovery.
**Gained:** Simpler retry logic; predictable timing.
**Acceptable because:** With 6 pipelines, the thundering herd is small. The debt is documented. Jitter is a trivial addition when needed.

---

## 3. Open Debts

### Debts with committed resolution points

| ID | Debt | Severity | Resolution trigger | Committed? |
|----|------|----------|-------------------|------------|
| D-1 | `parseEvidenceKeyParams` naming residue | Low | Family 3 | Yes — rename to `parseAnalyticalKeyParams` |
| D-2 | Handler constructor argument accumulation | Medium | Family 3 | Yes — switch to `AnalyticalHandlerDeps` struct |
| D-3 | Smoke test linear growth | Medium | Family 3 | Yes — extract `validate_analytical_family()` |
| D-4 | Codegen evaluation for reader/handler/test layers | Medium | Family 4 | Yes — evaluate, not necessarily adopt |

### Debts without committed resolution points

| ID | Debt | Severity | Why not committed | Risk if ignored |
|----|------|----------|-------------------|-----------------|
| D-5 | No backoff jitter in writer retry | Low | Trivial fix, no urgency at 6 pipelines | Thundering herd on ClickHouse recovery |
| D-6 | No NATS consumer lag visibility | Medium | Requires instrumentation not in scope | Buffer overflow with no warning; data loss invisible until /statusz shows drops |
| D-7 | Sticky degradation without auto-recovery | Medium | Significant complexity to implement correctly | Extended outages require manual intervention |
| D-8 | No load testing baseline | Medium | Infrastructure not available | Performance problems discovered late |
| D-9 | No pagination beyond 500 rows | Low | No current consumer needs more | Limits analytical depth for future use cases |
| D-10 | Metadata schema not validated at read | Low | Deserialized as map[string]string without key checks | Unexpected metadata shapes silent at read path |
| D-11 | Schema coherence is review-enforced, not compile-time | Medium | Tooling cost not justified at current table count | Drift risk scales linearly with table count |
| D-12 | ClickHouse client timeout not configurable | Low | Hard-coded 30s adequate for current queries | Becomes a problem with larger datasets or slower queries |

### Debts explicitly deferred (no-cost deferral)

These are debts that were considered and deliberately excluded from the near-term roadmap because they add complexity without current benefit:

- Dead-letter queue for failed writes
- Prometheus/Grafana/OpenTelemetry metrics
- Cold-start bootstrap from NATS replay
- Per-family batch configuration
- Event schema versioning
- Materialized views and secondary indexes
- Cross-family joins or composite endpoints
- Multi-instance ClickHouse or sharding
- Real-time streaming queries

---

## 4. Debt Trajectory

The debt profile is growing slowly and predictably. The committed resolution points (D-1 through D-4) at family 3 and family 4 prevent accumulation from becoming structural.

**Current debt load:** 12 active debts, 4 with committed triggers, 8 tracked without triggers.

**Projection at family 3:** D-1, D-2, D-3 resolved. D-4 evaluation begins. Net active debts: 9 (assuming no new debts from family 2).

**Projection at family 4:** D-4 resolution applied (codegen or explicit rejection). Net active debts: 8 or lower.

**Risk threshold:** If family 2 introduces more than 2 new frictions not captured in v2, the pattern is growing debt faster than it resolves it. This should trigger a hardening pause before family 3.
