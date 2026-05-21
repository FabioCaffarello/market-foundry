# Final Venue Smoke Usage, Results, and Limitations

> **Stage:** S330
> **Date:** 2026-03-21
> **Type:** Evidence and limits record
> **Phase:** 31b (Production Wiring Tranche)

---

## Purpose

This document records the concrete results, ergonomics, and remaining
limitations of the S330 composed pipeline smoke — the final operational
verification step before the production wiring tranche gate (S331).

---

## Smoke Execution Results

### Execution Summary

| Metric | Value |
|--------|-------|
| Entrypoint | `make smoke-composed` |
| Total phases | 5 |
| Tests exercised | SC-01..SC-07 + VP-01..VP-09 + EC-S325-1..10 + full regression suite |
| Total runtime | ~35 seconds |
| Stack required | No |
| Credentials required | No |
| Result | **PASS** |

### Phase-by-Phase Results

| Phase | Result | Detail |
|-------|--------|--------|
| 1 — Build verification | PASS | `go vet` clean on execution + actor packages |
| 2 — Composition (SC-01..07) | PASS | All 7 decorator composition tests pass |
| 3 — Venue path (VP-01..09) | PASS | All 9 venue path verification tests pass |
| 4 — Error classification | PASS | All 10 venue error code classification tests pass |
| 5 — Regression gate | PASS | Full execution package suite passes (~32s) |

---

## Verified Capabilities

| Capability | Test Coverage | Verdict |
|-----------|--------------|---------|
| Retry recovery (transient failures) | SC-01, VP-01, VP-05, VP-08 | Operational |
| Post-200 reconciliation | SC-01, VP-02, VP-08 | Operational |
| Halt checker / kill switch abort | SC-04, VP-05, VP-06 | Operational |
| Paper mode (retry-only, no reconciler) | SC-05, VP-07 | Operational |
| Safety gate (staleness + kill switch) | VP-09 | Operational |
| Fill event field preservation | VP-04 | Intact (12 fields) |
| Tracker counters (observability) | VP-05 | Accurate |
| Structured log retry metadata | VP-01, VP-03, VP-08 | Present |
| Error code classification enrichment | EC-S325-1..10 | Correct |
| JSON serialization round-trip | VP-02, VP-04 | Stable |

---

## Ergonomic Assessment

| Aspect | Assessment |
|--------|-----------|
| Entrypoint discovery | `make smoke-help` lists `smoke-composed` |
| Time to result | ~35s — acceptable for developer feedback loop |
| Infrastructure cost | Zero — no stack, no credentials |
| Failure diagnosis | Verbose `-v` output per phase; clear PASS/FAIL |
| Reproducibility | Deterministic; no external state dependencies |
| CI suitability | Yes — can run in any environment with Go toolchain |

---

## Remaining Limitations

| Limitation | Origin | Impact | Deferred To |
|-----------|--------|--------|-------------|
| No live NATS publish in smoke | S330 scope decision | Pipeline proven at Go test level; NATS covered by `smoke-live-stack` | Existing coverage |
| No real venue HTTP call | S330 scope decision | Adapter proven in S316; credentials needed for real calls | `make smoke-venue` |
| Retry policy not config-driven | R-S323-3 | Uses `DefaultRetryPolicy()` in tests; hardcoded but proven | Post-tranche |
| Reconciliation timeout not config-driven | S322 | Uses fixed timeout in tests | Post-tranche |
| No circuit breaker | Design scope | Outside tranche charter | Post-tranche |
| No OpenTelemetry/tracing | Design scope | slog-based observability proven; OTel deferred | Post-tranche |
| Startup log field verification | S329 recommendation | Not automated in smoke; visible in `make logs SERVICE=execute` | Manual verification |

---

## Evidence for Tranche Gate (S331)

This smoke, combined with the test suites it exercises, provides the following
evidence for the production wiring tranche gate:

| Gate Criterion | Evidence Source | Status |
|---------------|----------------|--------|
| PWT-1: RetrySubmitter around adapter | SC-01..SC-07, VP-01 | DONE |
| PWT-2: Post200Reconciler around retry | SC-01, VP-02, VP-08 | DONE |
| PWT-3: Observability hooks | SC-07, VP-03, VP-05 | DONE |
| PWT-4: Composed pipeline stable | S330 smoke PASS | DONE |
| Zero regressions | Phase 5 full suite | CONFIRMED |

---

## Recommendation

The composed pipeline is operationally verified and regression-free. The
tranche is ready to proceed to the production wiring gate (S331).
