# Stage S283 — Signal Evolution Charter and Scope Freeze Report

**Date:** 2026-03-21
**Status:** COMPLETE
**Predecessor:** S282 (CI enforcement and non-skipping test baseline)
**Successor:** S284 (MACD signal family delivery)

---

## 1. Executive Summary

Stage S283 formally opens the Signal Evolution Wave — the first feature delivery wave in market-foundry's history. The wave scope is frozen at 4 families (MACD, VWAP, ATR, Bollinger Squeeze) with explicit delivery order, acceptance criteria per family, and inflation protection rules. Interleaved Prometheus observability is scoped as a secondary concern limited to pipeline counters and writer gauges. The single-front discipline from prior waves is preserved.

---

## 2. Deliverables Produced

| # | Deliverable | Path | Purpose |
|---|-------------|------|---------|
| 1 | Wave charter and scope freeze | `docs/architecture/signal-evolution-wave-charter-and-scope-freeze.md` | Formal wave opening, boundaries, freeze rules |
| 2 | Family ordering and acceptance criteria | `docs/architecture/signal-evolution-family-ordering-and-acceptance-criteria.md` | Delivery sequence, per-family dependencies and acceptance |
| 3 | This report | `docs/stages/stage-s283-signal-evolution-charter-and-scope-freeze-report.md` | Stage closure evidence |

---

## 3. Decisions Made

### 3.1 Wave Scope Frozen

**4 families, no more, no fewer:**

| Family | Layer | Position |
|--------|-------|----------|
| MACD | Signal | 1st |
| ATR | Signal | 2nd |
| Bollinger Squeeze | Decision | 3rd |
| VWAP | Signal | 4th |

### 3.2 Delivery Order Rationale

MACD first (lowest risk, EMA analog) → ATR second (pure signal, volatility semantics) → Bollinger Squeeze third (first codegen-first decision, consumes existing signal) → VWAP last (most cross-layer wiring, volume dependency).

### 3.3 Codegen-First as Means

Every family follows the codegen-first workflow proven in S262 (Bollinger):

```
YAML spec → golden snapshots → integration markers → application logic → behavioral tests
```

The codegen framework itself is frozen — no new artifact templates, no generator changes.

### 3.4 Multi-Symbol Treatment

Multi-symbol is a **validation criterion** (smoke test must pass) not a delivery wave. JetStream subject isolation via wildcard `>` is architecturally sufficient. Multi-symbol optimization is deferred to post-wave gate.

### 3.5 Observability Treatment

Prometheus minimal metrics are interleaved with family delivery:

- Pipeline event counter and writer batch gauge enter with S284 (MACD)
- Control gate state gauge enters after all 4 families (S288)
- No tracing, dashboards, alerting, or dedicated observability stages

### 3.6 Stage Mapping

| Stage | Content |
|-------|---------|
| S284 | MACD signal family |
| S285 | ATR signal family |
| S286 | Bollinger Squeeze decision family |
| S287 | VWAP signal family |
| S288 | Post-Signal-Evolution-Wave gate |

---

## 4. Items Explicitly Excluded

| Item | Rationale |
|------|-----------|
| Multi-symbol scaling wave | JetStream isolation sufficient; no operational pressure |
| Venue readiness | Paper execution ceiling (S264–S269) |
| Full observability platform | No load to observe yet |
| Codegen expansion (new artifact types) | Side-effect only per S263 |
| New strategy/risk families | Current families sufficient to validate new signals |
| Configuration infrastructure | No user-facing surface needed |
| Actor topology changes | Proven in S270–S276 |
| Domain model changes | Stable since breadth wave S241–S244 |

---

## 5. Scope Inflation Protection

Three-layer protection against wave inflation:

1. **Family freeze**: exactly 4 families; additions require post-wave gate
2. **Acceptance freeze**: 8 universal + 4 family-specific criteria; no reductions
3. **Amendment gate**: any in-wave addition must satisfy all 5 criteria from charter §4.3

---

## 6. Pre-Wave Baseline

| Metric | Value | Source |
|--------|-------|--------|
| Codegen families | 11 | `codegen/integrated.yaml` |
| Golden snapshots | 22 | `codegen/golden-snapshots/` |
| Equivalence checks | 65 | `codegen-equivalence-check.sh` |
| Signal samplers | 3 (RSI, EMA, Bollinger) | `internal/application/signal/` |
| Decision evaluators | 2 (RSI Oversold, EMA Crossover) | `internal/application/signal/` |
| Behavioral tests | 52+ | CI pipeline |
| Auto-skipping tests in CI | 0 | S282 baseline |
| CI pipeline stages | 6 | `.github/workflows/ci.yml` |

Post-wave target: 15 codegen families, 30 golden snapshots, 6 signal samplers (or 5 + 1 decision evaluator), 65+ behavioral tests.

---

## 7. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| MACD reuses EMA logic that needs refactoring | EMA proven in ema_crossover_sampler; MACD reuses without modifying |
| Bollinger Squeeze requires upstream signal change | Squeeze reads bandwidth from existing output; no schema mutation |
| VWAP session reset ambiguity | Define reset boundary in YAML spec before implementation (S287) |
| Wave takes longer than expected | Each family is independently valuable; partial delivery still useful |

---

## 8. Gate Criteria Met

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Wave formally opened with charter | PASS | Charter document produced |
| Scope frozen with inflation protection | PASS | 3-layer protection documented |
| Family ordering justified | PASS | Ordering document with dependency analysis |
| Acceptance criteria defined per family | PASS | 8 universal + 4×4 family-specific criteria |
| Out-of-scope items explicit | PASS | 8 items with rationale and revisit timing |
| Observability bounded as interleaved | PASS | 3 metrics, 20% effort cap, no dedicated stages |
| Single-front discipline preserved | PASS | One family per stage, gated |
| S284 preparation clear | PASS | MACD delivery with full dependency and acceptance list |

---

## 9. Recommendation for S284

**S284: MACD Signal Family Delivery**

Scope:
1. Create `codegen/families/macd.yaml` spec
2. Generate golden snapshots via codegen toolchain
3. Insert integration markers in `cmd/writer/pipeline.go` and `internal/adapters/nats/natssignal/registry.go`
4. Implement `internal/application/signal/macd_sampler.go` with behavioral tests
5. Add ClickHouse schema for MACD signal table
6. Introduce pipeline event counter (interleaved Prometheus)
7. Verify full equivalence check and CI green

Entry condition: this report (S283) accepted.
Exit condition: all 8 universal acceptance criteria + MACD-1 through MACD-4 satisfied.
