# Stage S281 — Post-Operational-Proof Feature Gate Report

**Date:** 2026-03-21
**Type:** Strategic direction gate
**Predecessor:** S278–S280 (operational reconciliation, OS-process smoke, restart recovery)
**Verdict:** PASS — transition to feature delivery wave

---

## Executive Summary

After completing the S278–S280 operational proof micro-wave, the Foundry has proven restart resilience, compose-level process isolation, and durable consumer recovery across 20 scenarios. The system is operationally mature for its current paper-execution scope. This gate evaluates six candidate directions for the next major wave and recommends **Signal Evolution** as the primary direction — the first feature delivery wave in the project's history.

The recommendation is grounded in three findings: (1) S263 explicitly directed the project toward domain value delivery, yet three additional infrastructure waves followed; (2) all 8 bounded contexts are domain-complete and adapter-wired; (3) the codegen-first approach (proven with 11 families) provides a disciplined, charterable delivery pattern for new signal families.

A mandatory CI enforcement micro-tranche (1 stage) must close OD-OH1/OH2 before the wave opens. Composite execution observability is recommended as an interleaved secondary direction (metrics foundation only, not a full platform).

---

## Gate Inputs

### Documents Produced

| Document | Path |
|----------|------|
| Strategic options matrix | [next-wave-strategic-options-matrix.md](../architecture/next-wave-strategic-options-matrix.md) |
| Feature gate assessment | [post-operational-proof-feature-gate.md](../architecture/post-operational-proof-feature-gate.md) |
| This report | [stage-s281-post-operational-proof-feature-gate-report.md](stage-s281-post-operational-proof-feature-gate-report.md) |

### Codebase Analysis Performed

- Reviewed all 9 S278–S280 architecture and stage documents
- Mapped 8 bounded contexts across domain/application/adapter/actor layers
- Inventoried 11 codegen families, 8 binaries, 129 test files, 10 operational scripts
- Analyzed 6 CI pipeline stages and identified 34 auto-skipping tests
- Assessed observability surface (structured logging only; no metrics or tracing)
- Synthesized strategic evolution across 5 prior gates (S254, S257, S263, S269, S274)

---

## Options Evaluated

### Comparative Matrix

| Candidate | Reuse | Arch. Pressure | Domain Value | Regression Risk | Scope Contain. | Prereq. Ready | **Total** |
|-----------|-------|----------------|--------------|-----------------|----------------|---------------|-----------|
| A: Composite Execution Observability | 4 | 3 | 2 | 5 | 4 | 5 | **23** |
| B: Multi-Symbol Disciplined | 5 | 4 | 3 | 4 | 3 | 4 | **23** |
| **C: Signal Evolution** | **5** | **5** | **5** | **3** | **4** | **3** | **25** |
| D: Venue-Readiness Charter | 2 | 5 | 4 | 2 | 2 | 1 | **16** |
| E: Selective Codegen Expansion | 5 | 2 | 1 | 4 | 4 | 5 | **21** |
| F: CI + Observability Foundation | 4 | 3 | 1 | 5 | 5 | 5 | **23** |

### Per-Option Assessment

**A — Composite Execution Observability (23/30):**
Operationally strong but would constitute a fourth consecutive infrastructure wave. S263 guidance against continued infrastructure focus without domain value delivery applies directly. Observability foundation is better delivered as interleaved secondary direction.

**B — Multi-Symbol Disciplined (23/30):**
Validates scale but delivers no new capability. Partial proof already exists via smoke-multi-symbol.sh. JetStream durable consumer semantics already guarantee per-symbol isolation. Better addressed as acceptance criterion within signal evolution wave.

**C — Signal Evolution (25/30):**
Highest overall score. Only candidate with maximum domain value (5/5). Validates the entire infrastructure investment: codegen engine, behavioral model, NATS streaming, ClickHouse materialization, actor composition. Each family is independently charterable. Hard prerequisite: CI enforcement must close first.

**D — Venue-Readiness Charter (16/30):**
Lowest score. Paper execution proven only at S280 level. No observability for venue adapter diagnostics. Fill reconciliation alone is a full wave. External Binance dependency introduces non-deterministic failures. Premature by at least one full wave.

**E — Selective Codegen Expansion (21/30):**
S263 explicitly directed codegen expansion to be side-effect, not primary wave. Zero domain value. Store consumers, starters, and mappers should emerge naturally during signal evolution.

**F — CI + Observability Foundation (23/30):**
Too small to be a wave (2-3 stages). Essential as prerequisite but should not be elevated to primary direction. OD-OH1/OH2 closure is a gate condition, not a strategic objective.

---

## Decision: Primary Direction

### Signal Evolution Wave

**What:** Deliver 2-3 new signal families (MACD, VWAP, ATR) and 1+ new decision family (Bollinger Squeeze) using codegen-first approach.

**Why this, why now:**
1. First real feature delivery in project history — validates infrastructure ROI
2. Exercises all proven layers: codegen → NATS → behavioral → ClickHouse → gateway
3. Each family is bounded, charterable, and independently testable
4. Codegen-first reduces delivery cost and ensures consistency
5. New behavioral interactions stress-test composition quality
6. Bollinger signal already integrated (S262) — natural expansion point

**Entry condition:** CI enforcement gaps (OD-OH1, OD-OH2) must close in prerequisite stage S282.

**Minimum viable scope:** 2 signal families + 1 decision family, each with codegen YAML, generated artifacts, application-layer logic, behavioral tests, ClickHouse reader, and golden snapshots.

---

## Decision: Secondary Direction

### Composite Execution Observability (Interleaved)

**What:** Establish minimal Prometheus metrics foundation during signal evolution, scoped to pipeline counters and health indicators.

**Why interleaved, not dedicated:**
1. New families benefit from observable pipeline behavior
2. Avoids dedicating a full wave to infrastructure
3. Metrics are additive and low-risk
4. Each signal evolution stage can include metric instrumentation for the new family

**Scope:** `/metrics` endpoint in each binary with pipeline counters (`events_processed_total`, `pipeline_errors_total`), writer gauges (`buffer_depth`, `flush_duration_seconds`), and control gate state.

**NOT in scope:** Distributed tracing, dashboards, alerting, Grafana.

---

## What Explicitly NOT to Open

| Direction | Reason | When |
|-----------|--------|------|
| Venue-readiness charter | Premature; no observability; paper execution barely proven | After signal evolution + observability foundation |
| Full observability platform | Premature without feature load to observe | After signal evolution reveals observability priorities |
| Codegen expansion as primary | S263 directive: side-effect only | Naturally during signal evolution |
| Multi-symbol scale wave | Partial proof exists; JetStream isolation sufficient | As acceptance criterion in signal evolution |
| Configuration infrastructure | No operational pressure yet (OD-BW2) | When runtime config changes become necessary |
| Parallel feature fronts | Every gate reinforces single-front discipline | Never simultaneously |

---

## Prerequisite Stages

### S282: CI Infrastructure Enforcement
- Add NATS JetStream and ClickHouse services to CI
- Close OD-OH1 (25 NATS KV tests) and OD-OH2 (9 ClickHouse tests)
- Exit: zero auto-skipping infrastructure tests

### S283: Signal Evolution Charter and Scope Freeze
- Select families (2-3 signal + 1 decision)
- Freeze behavioral test requirements per family
- Define codegen-first delivery pattern
- Set stop conditions and scope boundaries

---

## Post-Wave Gate Criteria

The signal evolution wave succeeds if:
1. ≥2 new signal families delivered end-to-end
2. ≥1 new decision family consuming new signals
3. All new families have behavioral tests enforced in CI
4. Codegen golden snapshots cover all new families
5. Multi-symbol smoke passes with new families
6. Pipeline metrics observable via `/metrics`
7. Zero regressions in existing behavioral/integration tests
8. No secondary fronts opened during the wave

---

## Debts Carried Forward

| Debt | Severity | Disposition |
|------|----------|-------------|
| OD-OH1 | Medium | **Must close in S282** |
| OD-OH2 | Medium | **Must close in S282** |
| OD-BW2 | Medium | Deferred — no operational pressure |
| OD-CG1 | Medium | Deferred — known codegen limitation |
| OD-PE2 | Low | Open — governance only |
| OD-OH5 | Low | Deferred by design |
| OD-OH6 | Low | Deferred by design |

---

## Conclusion

The Foundry transitions from infrastructure hardening to feature delivery. Signal Evolution is the strategically optimal next wave — it delivers the highest domain value, maximally reuses proven infrastructure, and validates the codegen-first approach at scale. The mandatory CI enforcement prerequisite ensures regression safety. Composite observability as interleaved secondary provides operational confidence without consuming a dedicated wave.

The system has earned the right to deliver features. S281 formally opens that path.
