# Stage S259 — Codegen Spec Reconciliation with Breadth and Behavior Report

**Stage:** S259
**Date:** 2026-03-21
**Predecessor:** S258 (Codegen Re-entry Charter — PASS)
**Verdict:** PASS — reconciliation complete, zero code changes required

---

## 1. Executive Summary

Stage S259 performed a comprehensive reconciliation of the codegen spec, templates, golden snapshots, and integrated manifest against the current state of the market-foundry domain following the breadth wave (S241–S244) and behavioral wave (S249–S257).

**Key finding:** The codegen infrastructure is already reconciled with the enriched domain. The spec's column-agnostic design (`writer.columns` as a free-form string) absorbed the breadth wave changes (severity, rationale columns in decisions) without requiring structural changes to `spec.go`, `render.go`, or templates. The YAML spec files were correctly updated during breadth wave stages. All 20 golden snapshots match template output. All 35 codegen tests pass.

No code changes were necessary. The deliverables for this stage are documentation artifacts proving reconciliation and making explicit the before/after assumptions.

---

## 2. Reconciliation Applied

### 2.1 Verification steps executed

| Step | Command/Action | Result |
|---|---|---|
| Spec validation | `codegen validate-all` | 10/10 VALID, zero collisions |
| Golden comparison | `codegen check-all` | 20/20 PASS |
| Test suite | `go test ./codegen/... -count=1` | 35/35 PASS (0.218s) |
| Column comparison | Manual diff of spec `columns` vs pipeline.go INSERT SQL | All 10 families match |
| NATS subject comparison | Manual diff of spec `subject` vs registry files | All 10 families match |
| Snapshot-to-manual comparison | Manual diff of golden snapshots vs pipeline.go entries | Pipeline entries match; consumer specs semantically equivalent (expanded vs factory form) |
| Migration cross-reference | Column lists checked against deploy/migrations/ | All consistent |

### 2.2 Findings

| Finding | Category | Impact |
|---|---|---|
| Decision specs already include severity/rationale columns | Spec drift | None — already updated |
| Strategy/risk specs already have correct column lists | Spec drift | None — columns unchanged |
| Consumer spec golden snapshots use expanded struct form | Style difference | Expected — factory form will be replaced at S260 marker insertion |
| JSON payloads (DecisionInput, StrategyInput) are richer | Domain enrichment | None — inside columns, outside codegen boundary |
| Behavioral logic (scaling, rejection) is invisible to codegen | Boundary | Correct — human-authored, not generated |
| Dual-risk fan-out is invisible to codegen | Boundary | Correct — actor wiring, not pipeline config |

---

## 3. Files Changed

**None.** Zero code changes were required. All reconciliation work produced documentation artifacts only.

---

## 4. Deliverables Produced

| # | Document | Path | Status |
|---|---|---|---|
| 1 | Spec reconciliation with breadth and behavior | `docs/architecture/codegen-spec-reconciliation-with-breadth-and-behavior.md` | DELIVERED |
| 2 | Assumptions before and after domain enrichment | `docs/architecture/codegen-assumptions-before-and-after-domain-enrichment.md` | DELIVERED |
| 3 | This report | `docs/stages/stage-s259-codegen-spec-reconciliation-with-breadth-and-behavior-report.md` | DELIVERED |

---

## 5. Before/After Summary

### What changed in the domain

| Change | When | Where |
|---|---|---|
| Decision severity enum (high/moderate/low) | S234 breadth | Domain model + ClickHouse column |
| Decision rationale field | S234 breadth | Domain model + ClickHouse column |
| DecisionInput enriched with severity/rationale/confidence/timeframe | S241 breadth | JSON payload inside `decisions` column |
| StrategyInput enriched with cascading decision context | S241 breadth | JSON payload inside `strategies` column |
| Severity-based confidence scaling in strategy | S250 behavioral | Application logic (severity_scaling.go) |
| Strategy-type-aware risk scaling | S251 behavioral | Application logic (risk_scaling.go) |
| Dual-risk fan-out (position_exposure + drawdown_limit) | S241 breadth | Actor wiring in derive supervisor |
| Rejection at confidence ≤ 0 | S256 hardening | Domain logic in risk evaluators |
| Severity input normalization | S256 hardening | Application logic (TrimSpace + ToLower) |

### What changed in codegen (already done before S259)

| Change | When | What |
|---|---|---|
| Decision spec YAML: added severity, rationale to columns | S241 | `codegen/families/rsi_oversold.yaml`, `ema_crossover.yaml` |
| Decision golden snapshots: regenerated with new columns | S241 | `codegen/golden-snapshots/rsi_oversold/pipeline_entry.go.golden`, etc. |
| Strategy/risk/execution spec YAML files: created | S241 | 6 new spec files in `codegen/families/` |
| Strategy/risk/execution golden snapshots: created | S241 | 12 new golden snapshot files |

### What did NOT change in codegen (and shouldn't)

| Item | Why |
|---|---|
| spec.go (FamilySpec struct) | `columns` is a string — absorbs any column list |
| render.go (template engine) | Column-agnostic — uses `{{.Derived.InsertSQL}}` |
| consumer_spec.go.tmpl | Not affected by column changes |
| pipeline_entry.go.tmpl | Uses `InsertSQL` derived field, not individual columns |
| compare.go | Normalization logic unchanged |

---

## 6. Limits and Trade-offs

### What this reconciliation proves

- All 10 family specs match the current production pipeline configuration
- Golden snapshots reflect the enriched column lists
- Templates are resilient to domain enrichment via string-based column handling
- The codegen boundary (mechanical wiring only) is well-defined and stable

### What this reconciliation does NOT prove

- That generated code will compile when inserted into target files (S260 scope)
- That codegen-governed markers won't conflict with existing imports (S260 scope)
- That CI can enforce codegen equivalence automatically (S261 scope)

### Trade-offs accepted

| Trade-off | Rationale |
|---|---|
| Columns as opaque string (not typed) | Simplicity over safety — codegen doesn't validate column types or count |
| JSON payload schemas invisible | Keeps codegen simple — JSON evolution is domain responsibility |
| Consumer spec style difference | Expanded form is preferred; migration happens at marker insertion |
| No per-family customization of AckWait/MaxDeliver | Blocked by OD-BW2 (config infrastructure debt) |

---

## 7. Entry Condition Verification

| ID | Condition | S258 status | S259 verification |
|---|---|---|---|
| EN-5 | All 10 specs parseable by `codegen validate-all` | TO VERIFY | **VERIFIED** — all 10 VALID |

All other entry conditions were already MET at S258.

---

## 8. Risk Assessment Update

| Risk | S258 assessment | S259 update |
|---|---|---|
| Spec drift from domain | Medium | **Low** — verified all specs match current pipeline |
| Template brittleness | Low | **Low** — confirmed templates are column-agnostic |
| Behavioral test regression | Low | **Not tested** (no code changes to test) |
| Manual→generated divergence | Medium | **Low for pipeline entries** (byte-match); **Expected for consumer specs** (factory→expanded migration at S260) |

---

## 9. Preparation for S260

S260 (Integration Expansion) should proceed with high confidence. Specific actions:

### 9.1 Consumer spec marker insertion (8 families)

For each of decision (2), strategy (2), risk (2), execution (1), evidence (1):

1. Replace factory-style `WriterXxxConsumer()` with codegen marker block containing expanded struct literal
2. Function signature and return type remain identical
3. Verify caller sites are unaffected (function name unchanged)
4. Note: evidence candle uses different naming conventions — template already handles this

### 9.2 Pipeline entry marker insertion (8 families)

For each non-signal family:

1. Wrap existing pipeline entry struct with `codegen:begin`/`codegen:end` markers
2. Verify generated output matches manual code (post-normalization)
3. No structural changes to pipeline.go beyond marker insertion

### 9.3 `integrated.yaml` expansion

Extend from 4 entries to 20 entries. Each new entry needs:
- family, artifact, spec, golden, target, marker, integrated_at, stage

### 9.4 Recommended family order for S260

1. **Decision** (rsi_oversold, ema_crossover) — highest breadth-wave impact, validate severity/rationale columns survive marker insertion
2. **Strategy** (mean_reversion_entry, trend_following_entry) — validate multi-word family naming
3. **Risk** (position_exposure, drawdown_limit) — validate risk table columns
4. **Execution** (paper_order) — validate execution-specific columns (exec_correlation_id, exec_causation_id)
5. **Evidence** (candle) — validate evidence-layer naming conventions (different from others)

### 9.5 Pre-S260 verification checklist

- [ ] Behavioral tests still pass (47 tests)
- [ ] `codegen validate-all` still passes (verified at S259)
- [ ] `codegen check-all` still passes (verified at S259)
- [ ] All target files identified and readable

---

## 10. Verdict

**PASS** — Codegen spec reconciliation complete. All 10 family specs are validated and aligned with the current domain. Golden snapshots match template output. No code changes were needed. The codegen's column-agnostic design absorbed domain enrichment without modification. The base is ready for integration expansion in S260.
