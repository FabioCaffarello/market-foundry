# Stage S300 — Multi-Symbol Operational Scaling Charter Report

> Opens the Multi-Symbol Operational Scaling Wave (Phase 29).
> Status: **COMPLETE**
> Date: 2026-03-21
> Predecessor: S299

---

## 1. Executive Summary

Stage S300 opens the Multi-Symbol Operational Scaling Wave, the next systemic pressure after the Composite Execution Observability Wave (S294–S299) closed with 6/7 FULL, 1/7 SUBSTANTIAL, and zero regressions.

The strategic decision is clear: the system has proven that a single symbol can flow through three families (EMA, Trend, Squeeze) with full composite observability. The next question is whether this holds when multiple symbols operate simultaneously.

This stage delivers:
- A formal wave charter with frozen scope
- Seven governing questions (MQ1–MQ7) targeting isolation, correctness, ordering, and scaling
- Ten explicit non-goals preventing scope inflation
- A recommended 6-stage sequence (S300–S305)

No code changes. No schema changes. No new capabilities. Pure architectural governance.

---

## 2. Rationale for This Wave

### Why multi-symbol scaling, not something else?

| Alternative | Why not now |
|-------------|------------|
| **New signal/decision families** | Families are already proven at 3 (EMA, Trend, Squeeze). Adding more without multi-symbol proof would increase single-symbol depth while leaving multi-symbol operation unvalidated. |
| **Venue readiness** | Real exchange connectivity introduces authentication, fill reconciliation, and regulatory concerns. The paper pipeline must prove scaling correctness before real venue work makes sense. |
| **Portfolio aggregation** | Cross-symbol aggregation requires per-symbol correctness as a prerequisite. You cannot aggregate what you have not isolated. |
| **Residual gap closure** | GAP-Q2-A and GAP-Q5-A from S299 are single-symbol depth enhancements, independent of scaling. They can be addressed later without blocking this wave. |
| **Performance optimization** | Optimization requires measurement, which requires multi-symbol load. This wave provides the measurement baseline; optimization is a logical successor. |

### The systemic gap

Phase 2 (S11–S17) proved multi-symbol **infrastructure readiness**: actor hierarchy, config-driven activation, partition key isolation. But infrastructure readiness is not operational correctness. The gap is:

- No evidence that 3 symbols produce correct, isolated chains simultaneously
- No evidence that composite read model queries are symbol-scoped under multi-symbol data
- No evidence that pipeline funnel and disposition counts are accurate per symbol
- No evidence of ordering consistency under concurrent multi-symbol event production
- No evidence of resource scaling behavior with N>1 symbols

This wave closes every item on that list.

---

## 3. Wave Charter Summary

| Attribute | Value |
|-----------|-------|
| **Wave name** | Multi-Symbol Operational Scaling |
| **Phase** | 29 |
| **Stages** | S300–S305 (6 stages) |
| **Symbols** | `btcusdt`, `ethusdt`, `solusdt` (3 symbols, all via `binancef`) |
| **Families** | EMA, Trend, Squeeze (existing only) |
| **New code** | Tests and configuration only — no new production code expected |
| **Schema changes** | None |
| **New endpoints** | None |
| **Governing questions** | MQ1–MQ7 |
| **Non-goals** | 10 explicit items (NG-1 through NG-10) |

---

## 4. Governing Questions (MQ1–MQ7)

| Question | Summary | Target Stage |
|----------|---------|-------------|
| **MQ1** | Symbol isolation — no cross-symbol contamination at any stage | S301 |
| **MQ2** | Composite chain correctness per symbol under concurrent data | S302 |
| **MQ3** | Batch query symbol scoping — only requested symbol's chains returned | S302 |
| **MQ4** | Funnel accuracy per symbol — stage counts reflect only that symbol | S302 |
| **MQ5** | Disposition accuracy per symbol — risk breakdowns are symbol-scoped | S302 |
| **MQ6** | Ordering and timestamp consistency under multi-symbol concurrency | S303 |
| **MQ7** | Resource scaling behavior — proportional, not exponential, with 3× load | S304 |

**Closure threshold**: All 7 at FULL or SUBSTANTIAL.

Full definitions: `docs/architecture/multi-symbol-capabilities-questions-and-non-goals.md`

---

## 5. Non-Goals Summary

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-1 | Real venue connectivity | Separate architectural domain |
| NG-2 | Order management system | Stateful write-side system, not scaling proof |
| NG-3 | Portfolio-level aggregation | Requires isolation first; successor wave |
| NG-4 | New families | Would confound scaling attribution |
| NG-5 | Operational dashboards | Presentation layer, not evidence |
| NG-6 | Write-side schema changes | Validates existing schema sufficiency |
| NG-7 | Codegen path extension | Frozen since S263; orthogonal |
| NG-8 | Performance optimization | Measure first, optimize later |
| NG-9 | New HTTP endpoints | Validate existing surface, not extend |
| NG-10 | S299 residual gap closure | Independent of multi-symbol scaling |

Full definitions: `docs/architecture/multi-symbol-capabilities-questions-and-non-goals.md`

---

## 6. Recommended Stage Sequence

### S301 — Multi-Symbol Config Activation and Isolation Proof

**Objective**: Activate 3 symbols (`btcusdt`, `ethusdt`, `solusdt`) through config changes. Prove partition isolation across NATS KV buckets, ClickHouse writes, and actor hierarchy instantiation.

**Key evidence**: Each symbol gets its own NATS KV partitions and ClickHouse rows. Cross-symbol queries return zero foreign results.

**Primary question answered**: MQ1 (Symbol Isolation).

---

### S302 — Cross-Symbol Composite Read Model Validation

**Objective**: Prove the composite read model (chain, batch, funnel, dispositions) returns correct, symbol-scoped results when all 3 symbols have concurrent data.

**Key evidence**: Per-symbol chain completeness. Batch queries return only requested symbol. Funnel counts conserve across per-symbol queries. Disposition totals sum correctly.

**Primary questions answered**: MQ2, MQ3, MQ4, MQ5.

---

### S303 — Multi-Symbol Concurrent Pipeline Consistency

**Objective**: Validate that concurrent event production from 3 symbols maintains correct temporal ordering and causal spine integrity within each chain.

**Key evidence**: No ordering violations (`signal.occurred_at <= decision.occurred_at <= ... <= execution.occurred_at`). Causation_id references remain internally consistent. No cross-symbol causation links.

**Primary question answered**: MQ6 (Ordering Consistency).

---

### S304 — Operational Scaling Boundary and Resource Proof

**Objective**: Measure resource consumption and query latency under 3-symbol load. Compare against single-symbol baseline.

**Key evidence**: Quantitative measurements of goroutine count, memory, ClickHouse query time. Scaling is proportional (linear or sub-linear), not exponential. No goroutine leaks. No unbounded queue growth.

**Primary question answered**: MQ7 (Resource Scaling).

---

### S305 — Multi-Symbol Operational Gate and Wave Closure

**Objective**: Formal evidence gate. Verify all MQ1–MQ7 are answerable. Verify Q1–Q7 from S299 still hold (zero regression). Issue wave closure verdict.

**Key evidence**: Answerability matrix. Regression matrix. Closure or non-closure verdict with rationale.

---

## 7. Preparation for S301

To begin S301, the following should be ready:

1. **Configuration templates** for 3-symbol activation — review existing config structure for `source.symbol.timeframe` triplets.
2. **Test infrastructure** — ensure integration test harness can spin up multi-symbol scenarios (NATS + ClickHouse with concurrent writers).
3. **Baseline measurements** — capture single-symbol chain count, query latency, and resource metrics as the comparison baseline for S304.
4. **Review NATS KV bucket naming** — confirm that `{source}.{symbol}.{timeframe}` partition keys are correctly used across all actors.

No code changes required for preparation — only review and documentation.

---

## 8. Deliverables

| # | File | Type |
|---|------|------|
| 1 | `docs/architecture/multi-symbol-operational-scaling-wave-charter-and-scope-freeze.md` | Architecture — wave charter |
| 2 | `docs/architecture/multi-symbol-capabilities-questions-and-non-goals.md` | Architecture — governing questions and non-goals |
| 3 | `docs/stages/stage-s300-multi-symbol-operational-scaling-charter-report.md` | Stage report (this document) |
