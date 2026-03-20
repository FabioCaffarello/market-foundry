# Stage S134: Timeframe-Driven Friction Capture — Report

> **Stage:** S134
> **Type:** Architectural Analysis
> **Wave:** TC-01 (Timeframe Coverage)
> **Predecessor:** S133 (End-to-End Timeframe Coverage Validation)
> **Status:** Complete

---

## 1. Executive Summary

Stage S134 captures the real frictions exposed by expanding the Market Foundry pipeline from 2 to 4 timeframes (TC-01). The analysis is grounded in evidence from S132 implementation, S133 validation, and direct code inspection.

**Result:** The expansion revealed **zero bugs**, **zero real refactor triggers**, and **no performance bottlenecks**. The architecture handles temporal scaling exactly as designed: config-driven, linearly growing, and structurally sound.

The captured friction consists of:
- **5 operational fragilities** — small gaps in validation, diagnostics, and documentation fixable with targeted enhancements.
- **6 structural debts** — design choices that are fine at 4 TFs but will need attention before TC-02 (8+ TFs).
- **8 acceptable boilerplate items** — inherent repetition that carries no structural cost.
- **6 non-frictions** — anticipated problems that did not materialize, validating S10-S15 architectural decisions.

Two P1 items (config validation, recovery runbook) can be resolved in ~1 hour. Five P2 items form the preparation checklist for TC-02 planning.

---

## 2. Frictions and Findings — Summary

### 2.1 Classification Breakdown

| Classification | Count | Examples |
|---------------|-------|---------|
| Bug | 0 | — |
| Operational Fragility | 5 | Config validation gap (F-02), per-TF idle detection (F-05), null response ambiguity (F-08), smoke test scope (F-16), recovery runbook (F-17) |
| Acceptable Boilerplate | 8 | Integer TF representation (F-03), log scaling (F-06), HTTP test duplication (F-09), actor/KV/subject cardinality (F-10/11/12), configctl scope (F-18) |
| Structural Debt | 6 | Global TF list (F-01), single tracker (F-04), no TF listing endpoint (F-07), window state loss (F-13), no interim snapshots (F-15), no aggregate view (F-19) |
| Real Refactor Trigger | 0 | — |

### 2.2 Priority Distribution

| Priority | Count | Action |
|----------|-------|--------|
| P1 (fix before TC-02 planning) | 2 | F-02, F-17 |
| P2 (address in TC-02 prep) | 5 | F-01, F-04, F-05, F-13, F-15 |
| P3 (track, revisit) | 3 | F-07, F-08, F-19 |
| P4 (accept permanently) | 9 | F-03, F-06, F-09, F-10, F-11, F-12, F-14, F-16, F-18 |

### 2.3 Items That Did Not Confirm as Problems

| ID | Anticipated Concern | Finding |
|----|-------------------|---------|
| NF-01 | NATS stream pressure from doubled subjects | 64 subjects is negligible for NATS |
| NF-02 | Fan-out latency from 4× iteration | ~1-2μs overhead, unmeasurable |
| NF-03 | KV write contention from doubled keys | Higher TFs write *less* frequently; total load < 30% increase |
| NF-04 | Dedup key collision across timeframes | Timeframe embedded in key; collision mathematically impossible |
| NF-05 | Cross-timeframe signal interference | Per-timeframe subject routing prevents any cross-contamination |
| NF-06 | Memory accumulation at 3600s | O(1) accumulator per candle, not O(trades); negligible |

---

## 3. Prioritized Friction Matrix — Key Decisions

### 3.1 P1 Items (Immediate)

**F-02: Timeframe Config Validation** — Add `ValidateTimeframes()` to `schema.go`: reject duplicates, reject <10s and >86400s. ~10 lines. Prevents silent misconfiguration.

**F-17: Recovery Runbook** — Document expected data loss per timeframe on crash/restart. Zero code change.

### 3.2 P2 Items (TC-02 Gate)

**F-13 + F-15: Window State Persistence** — The most significant structural debt. At 3600s, a crash can lose up to 60 minutes of accumulated state. At TC-02 (4h), this becomes 4 hours. Interim snapshots or WAL for in-progress candles should be evaluated before committing to TC-02.

**F-04 + F-05: Per-Timeframe Diagnostics** — At 8+ TFs, aggregate tracker visibility is insufficient. Per-timeframe tracker split plus timeframe-aware idle detection should be implemented before TC-02.

**F-01: Per-Binding Timeframes** — Only if TC-02 requires heterogeneous timeframe sets per symbol.

---

## 4. Trade-Offs Accepted

| Trade-Off | Rationale | Risk |
|-----------|-----------|------|
| Global timeframe list for all bindings | Keeps config simple; all symbols benefit equally from temporal coverage | Low until heterogeneous needs emerge |
| No interim candle snapshots | Avoids complexity of in-progress projections; final-only semantics are simpler and correct | Medium for 3600s (60-min dead zone); High for 4h+ |
| Signal warmup latency at high TFs | Inherent to low-frequency analysis; not an architectural trade-off | None — physics constraint |
| Aggregate-only tracker for derive | Adequate at 4 TFs; per-TF breakdown available via custom counters | Low now; Medium at 8+ TFs |
| Smoke test validates wiring, not data, at high TFs | Three-tier validation procedure handles data correctness separately | Low — procedure is documented |
| Integer-only timeframe representation | Unambiguous and machine-friendly; cosmetic concern only | None |

---

## 5. Files Produced

### 5.1 Architecture Documentation (2 files, new)

| File | Purpose |
|------|---------|
| `docs/architecture/timeframe-coverage-01-frictions-and-findings.md` | Detailed friction capture: 19 findings + 6 non-frictions, each with evidence, classification, and verdict |
| `docs/architecture/timeframe-coverage-01-prioritized-friction-matrix.md` | Priority-ordered matrix with effort estimates, dependency map, and TC-02 decision framework |

### 5.2 Stage Report (1 file, new)

| File | Purpose |
|------|---------|
| `docs/stages/stage-s134-timeframe-driven-friction-capture-report.md` | This report |

### 5.3 Go Source Code (0 files)

**No Go files were modified.** S134 is a pure analysis stage.

---

## 6. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No cardinality increase treated as automatic problem | PASS — F-10/11/12 explicitly classified as non-issues |
| No new product wave opened | PASS — zero new capabilities |
| No refactor justified without evidence | PASS — zero refactor triggers identified |
| Intentional limitations not confused with failures | PASS — F-14 (warmup latency) explicitly classified as physics constraint |
| Focus on real impact of temporal expansion | PASS — all findings tied to S132/S133 evidence |

---

## 7. Strategic Position

### 7.1 What S134 Proves

1. **TC-01 produced no bugs.** The 2→4 TF expansion is structurally clean.
2. **The architecture absorbs temporal scaling.** Linear growth, config-driven activation, and parameterized surfaces held under pressure.
3. **6 anticipated problems did not materialize.** The S10-S15 design is more robust than conservative estimates assumed.
4. **The real friction is operational, not structural.** Config validation, diagnostic granularity, and state persistence are the friction axes — not architectural redesign.

### 7.2 What S134 Does NOT Prove

1. **TC-02 readiness.** The P2 items (especially F-13 state persistence) must be evaluated before committing to 4h/daily timeframes.
2. **Long-term operational stability.** Extended multi-hour runtime validation was not performed in S134.
3. **Performance at higher symbol counts.** The analysis covers 2 symbols. Growth to 10+ symbols multiplies all dimensions.

### 7.3 Maturity Reading

The system's temporal maturity is **high for the current scope** and **guarded for the next expansion**:

| Dimension | TC-01 Status | TC-02 Readiness |
|-----------|-------------|----------------|
| Config activation | Solid | Needs per-binding TF evaluation (F-01) |
| Event flow | Solid | No concerns |
| Query surface | Solid | No concerns |
| Diagnostics | Adequate | Needs per-TF granularity (F-04/F-05) |
| State resilience | Acceptable | Needs state persistence evaluation (F-13/F-15) |
| Operational docs | Adequate | Needs recovery runbook (F-17) |

---

## 8. Preparation for S135

### 8.1 Recommended S135 Scope

S135 should close P1 items and establish TC-01 as complete:

1. **Implement F-02:** `ValidateTimeframes()` in `schema.go` — reject duplicates, enforce range [10, 86400].
2. **Document F-17:** Recovery expectations per timeframe in the validation procedure.
3. **Declare TC-01 wave complete** — all stages S131-S135 finished, friction captured and prioritized.

### 8.2 S135 Entry Conditions

| Condition | Status |
|-----------|--------|
| TC-01 frictions captured with evidence | PASS (this stage) |
| Priority matrix actionable | PASS |
| P1 items identified | PASS (F-02, F-17) |
| Trade-offs documented | PASS |
| Guard rails maintained | PASS |

### 8.3 Post-TC-01 Direction

| Outcome | Next Step |
|---------|-----------|
| S135 closes P1 items | TC-01 wave complete |
| TC-02 scoping begins | Resolve P2 decision framework (§3.2) |
| Other priority emerges | P2 items remain documented; revisit when relevant |

---

## 9. Conclusion

TC-01 is a validation success. The temporal expansion from 2 to 4 timeframes revealed no architectural deficiencies. The frictions captured are proportionate, well-classified, and prioritized for action. The system gains a substantially more mature understanding of where temporal growth imposes real cost (state persistence, diagnostic granularity) versus where it is trivially absorbed (event flow, query surfaces, actor count).

The base is ready for small, justified enhancements (P1) and informed TC-02 planning (P2).
