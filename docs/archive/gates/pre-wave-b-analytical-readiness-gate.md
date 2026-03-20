# Pre-Wave-B Analytical Readiness Gate

## Purpose

This document is the formal gate evaluation for the analytical layer before Wave B expansion.
It answers one question: **does the evidence from S157–S161 justify opening controlled expansion?**

The gate is pass/fail per criterion. Enthusiasm does not count. Only evidence.

---

## Gate Context

S156 closed Wave A with three explicit preconditions for Wave B:

| # | Precondition | Required By |
|---|---|---|
| P1 | Reader path minimum instrumentation (query timing, error logging) | S156 |
| P2 | One end-to-end integration test (NATS → writer → CH → reader → HTTP) | S156 |
| P3 | Writer config validation at startup (fail-fast on invalid values) | S156 |

S157–S161 were sequenced to address these preconditions and the surrounding architectural hygiene needed to make expansion safe.

---

## Gate Criteria Evaluation

### 1. Are analytical layer responsibilities clear enough?

**Verdict: PASS**

**Evidence:**
- S157 produced a complete responsibility map for all 6 analytical components (migrate, writer, reader, gateway, observability, ClickHouse adapter).
- Five boundary issues were identified and prioritized (AP-01 through AP-05).
- Eight canonical design patterns documented and validated (DP-01 through DP-08).
- Ten explicit non-goals cataloged (NG-01 through NG-10) with rationale for each.
- Responsibility anti-patterns identified, traced to specific files, and corrected or documented.

**Residual concern:** Distributed schema knowledge (column order defined in DDL, mappers, and reader) remains reviewer-enforced, not compile-time validated. This is a coordination cost that scales linearly with table count — acceptable at 6 tables, friction point beyond ~12.

---

### 2. Are writer/reader/gateway/migrate boundaries adequate?

**Verdict: PASS**

**Evidence:**
- S158 extracted the reader from `cmd/gateway/` to `internal/adapters/clickhouse/candle_reader.go` (106 lines), restoring gateway composition root purity.
- Compile-time interface assertion (`var _ analyticalclient.CandleReader = (*clickhouse.CandleReader)(nil)`) prevents interface drift.
- Six adapter boundary rules (AB-01 through AB-06) documented and enforced structurally.
- Failure isolation matrix confirmed: writer crash has no impact on gateway operational paths; ClickHouse down triggers buffer overflow in writer but gateway returns 503 only on analytical endpoints.
- Zero runtime coupling between writer, reader, gateway, and migrate — all independently deployable.
- R-01 through R-04 invariants (no operational dependency on ClickHouse) preserved across all changes.

**Residual concern:** Write-path mappers remain in `cmd/writer/` (intentional asymmetry — composition-specific). This is documented and acceptable but means new families require changes in two locations (adapter + binary).

---

### 3. Is end-to-end integration proven convincingly?

**Verdict: PASS**

**Evidence:**
- S159 delivered `scripts/smoke-analytical-e2e.sh`: automated 7-phase integration proof.
  - Phase 1: Infrastructure readiness (CH + writer + gateway health probes)
  - Phase 2: Migration status (7 tables exist, 6 migrations applied)
  - Phase 3: Writer pipeline health (NATS → writer event consumption verified)
  - Phase 4: ClickHouse data verification (rows persisted in evidence_candles)
  - Phase 5: Reader → HTTP query surface (endpoint returns candles, 12-field structure validated)
  - Phase 6: Error handling (3 negative cases: missing timeframe, invalid limit, since > until)
  - Phase 7: Writer observability (no degraded pipelines confirmed)
- `make smoke-analytical` target created, writer added to build targets and service registry.
- All five boundary segments confirmed coherent: NATS↔writer, writer↔CH, reader↔CH, gateway↔reader, analytical↔operational.

**Residual concern:** Integration proof is script-based (not Go test), runs against live Docker stack, and is not yet wired into CI. This is adequate for gate purposes but CI integration should happen early in Wave B. Non-candle families are not exercised — only candles have HTTP endpoints. Same machinery argument is sound but unproven for other families.

---

### 4. Is the read path observable enough?

**Verdict: PASS**

**Evidence:**
- S160 instrumented the read path at three layers:
  - **Adapter layer**: Wall-clock timing, DEBUG log on success (row count, elapsed_ms), ERROR log on query failure and scan errors.
  - **Use case layer**: Duration measurement, QueryMeta populated (query_ms, row_count), INFO on success, WARN on failure.
  - **HTTP handler layer**: Total handler timing, Server-Timing header (`total;dur=N, query;dur=M`), meta field in JSON response.
- Response contract extended from `{candles, source}` to `{candles, source, meta: {query_ms, row_count}}`.
- Logger propagation consistent: single root logger with named sub-components throughout compose chain.
- Runbook documented with 5 failure scenario playbooks and diagnostic actions.

**Residual concern:** No Prometheus/OpenTelemetry metrics (by design guard rail). No request counting middleware — no throughput visibility. No cross-service trace IDs (writer↔reader correlation missing). These are documented non-goals at current scale but will become liabilities if the analytical surface grows significantly.

---

### 5. Is writer config/startup robust enough?

**Verdict: PASS**

**Evidence:**
- S161 implemented `ValidateForWriter()` with field-specific, aggregated error reporting:
  - addr, database, username: non-empty
  - batch_size, max_pending, max_retries: non-negative
  - flush_interval, initial_backoff: valid Go duration strings
  - At least one pipeline family configured
- Writer startup consolidated into strict 3-phase pattern: Phase 0 (config validation, no I/O) → Phase 1 (open connections) → Phase 2 (run).
- Gateway validation: invalid ClickHouse config logs warning and disables analytical (never hard-exits).
- All error messages prefixed with `writer startup blocked:` and name the offending field.
- 11 new tests covering writer-specific validation; all 42 settings tests pass.
- Failure mode catalog with 9 conditions (F-01 through F-09) documenting writer vs gateway behavior.

**Residual concern:** Zero-value batching fields silently default (batch_size=0 → driver default). Password field not validated for emptiness (may be intentional in dev). No schema existence check at startup (delegated to cmd/migrate). These are documented and acceptable trade-offs.

---

## S156 Precondition Status

| # | Precondition | Status | Cleared By | Evidence |
|---|---|---|---|---|
| P1 | Reader path instrumentation | **SATISFIED** | S160 | Three-layer instrumentation, Server-Timing headers, response metadata |
| P2 | End-to-end integration test | **SATISFIED** | S159 | 7-phase automated script, all boundary segments verified |
| P3 | Writer config validation at startup | **SATISFIED** | S161 | Fail-fast phasing, field-specific errors, 11 new tests |

**All three S156 preconditions are satisfied.**

---

## Code Maturity Assessment

| Component | LOC (impl) | LOC (test) | Test Ratio | Maturity |
|---|---|---|---|---|
| Writer service | 573 | 1,059 | 1:1.85 | Alpha→Beta |
| ClickHouse adapter | 202 | 185 | 1:0.92 | Beta |
| Analytical client | 100 | 137 | 1:1.37 | Beta |
| HTTP handlers/routes | 183 | 149 | 1:0.81 | Beta |
| Migrate tool | 297 | 153 | 1:0.51 | Production-ready |
| **Total** | **1,355** | **1,683** | **1:1.24** | **Alpha→Beta** |

Overall test-to-code ratio of 1.24 indicates healthy test discipline (industry standard: 0.8–1.5).

---

## Gate Decision

### Is the analytical layer ready for Wave B?

**YES — with constraints.**

All three S156 preconditions are satisfied. Responsibilities are mapped, boundaries are hardened, integration is proven, the read path is observable, and startup validation is robust. The evidence supports controlled expansion.

### Constraints on Wave B

1. **Scope limit**: One new pipeline family + one new query endpoint per iteration. Do not expand all families simultaneously.
2. **Pattern discipline**: Every new family must meet the same hardening standard as candles — tests, retry semantics, observable, config-validated.
3. **CI integration**: `smoke-analytical-e2e.sh` must be wired into CI before the second Wave B family lands.
4. **Schema coherence**: Adding a new read-path adapter requires verifying column alignment with DDL and write-path mapper. No shortcut.
5. **Observability parity**: No new endpoint without three-layer instrumentation (adapter, use case, HTTP handler) matching S160 pattern.

### What this gate does NOT authorize

- Broad buildout of all 5 remaining read-path families in a single push.
- Introduction of external observability tooling (Prometheus, Grafana, distributed tracing).
- Auto-recovery from degraded state.
- Dead-letter queue or deduplication mechanisms.
- Backfill or cold-start bootstrap capabilities.
- Any change to operational baseline services.

---

## Gate Signatories

| Role | Assessment | Date |
|---|---|---|
| Architecture | PASS — all criteria met with documented residual concerns | 2026-03-19 |
