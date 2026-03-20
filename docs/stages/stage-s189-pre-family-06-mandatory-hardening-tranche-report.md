# Stage S189 — Pre-Family-06 Mandatory Hardening Tranche Report

**Date:** 2026-03-20
**Scope:** Mandatory hardening pass before Family 06 expansion
**Trigger:** S188 ceiling evidence — handler at 615/620 hard ceiling
**Status:** Complete

## Executive Summary

Family 05 (S187/S188) proved that the Wave B manual expansion pattern has reached its handler file ceiling at 615/620 lines. This stage translates the triggered findings into a single, targeted hardening item: extraction of `parseAnalyticalParams()` to consolidate the limit/since/until parsing duplicated verbatim across all 6 handler methods.

The extraction reduces the handler from 615 → 501 lines — eliminating 114 lines of pure duplication. All 924 lines of existing handler tests pass without modification, confirming zero behavioral change. The handler can now absorb ~2–3 more families before the ceiling returns, at which point codegen becomes mandatory.

## Triggers Acionados vs Adiados

### Acionados (Ação Tomada)

| # | Trigger | Ação | Resultado |
|---|---------|------|-----------|
| TRIG-1 | Handler at 615/620 hard ceiling | Extract `parseAnalyticalParams()` | Handler reduced to 501 lines. Ceiling unblocked. |

### Acionados (Decisão Tomada, Sem Implementação)

| # | Trigger | Decisão | Razão |
|---|---------|---------|-------|
| TRIG-2 | Codegen scope definition | Deferred to dedicated stage | Handler extraction is sufficient unblock. Codegen is multi-day effort, not minimum viable. |
| TRIG-3 | Reader 10-param signature | Monitored, not blocked | 11 params (Family 06) is tolerable. Codegen eliminates this. |

### Adiados (Sem Trigger Acionado)

| # | Item | Razão |
|---|------|-------|
| DEF-C2 | Schema coherence compile-time checks | 6 tables, ~95 columns — under 12/100 threshold |
| DEF-C3 | Handler file split | Extraction reduced to 501 lines — split unnecessary |
| DEF-C4 | Friction count gate | 3 frictions in F-05 but same root cause — not pattern breakdown |
| DEF-U1–U9 | 9 low-severity deferred items | None escalated in F-05 |

## Hardening Implementado

### H-5: `parseAnalyticalParams()` Extraction

**What changed:**
- New `analyticalParams` struct: `Limit int`, `Since int64`, `Until int64`
- New `parseAnalyticalParams(r *http.Request) (analyticalParams, *problem.Problem)` function
- All 6 handler methods (`GetCandleHistory`, `GetSignalHistory`, `GetDecisionHistory`, `GetStrategyHistory`, `GetRiskHistory`, `GetExecutionHistory`) updated to use the helper

**Files modified:** 1 — `internal/interfaces/http/handlers/analytical.go`

**Metrics:**

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Handler file lines | 615 | 501 | -114 |
| Inline limit parsing blocks | 6 | 0 | -6 |
| Inline since/until parsing blocks | 6 | 0 | -6 |
| `parseAnalyticalParams` calls | 0 | 6 | +6 |
| `strconv.Atoi(limitStr)` occurrences | 6 | 1 (in helper) | -5 |
| Test modifications | — | 0 | 0 |
| Behavioral changes | — | 0 | 0 |

**Runway gained:** Handler at 501 can absorb ~2–3 more families at ~100 lines each before approaching 700–800 lines, where codegen or split becomes mandatory.

## Validation

| Criterion | Result |
|---|---|
| `parseAnalyticalParams` exists | PASS — 8 occurrences (1 struct + 1 func + 6 calls) |
| All 6 handlers use helper | PASS — zero inline limit/since/until parsing remains |
| Handler under ceiling | PASS — 501 lines (was 615) |
| Build passes | PASS — `go build ./cmd/gateway/...` |
| All tests pass | PASS — `go test ./internal/interfaces/http/handlers/...` (0 modifications) |
| No scope creep | PASS — only `analytical.go` modified |
| No new frictions | PASS — pure mechanical extraction |

## Gains and Trade-offs

### Gains

1. **Ceiling unblocked.** Handler can absorb Family 06 (~601 lines) without triggering the 620-line ceiling.
2. **Duplication eliminated.** 114 lines of verbatim copy-paste removed. Each handler method is now ~25 lines shorter.
3. **Consistency enforced.** Limit validation (1–500), since/until parsing, and defaults are now defined in exactly one place.
4. **Zero-risk change.** No behavioral change. No test modifications. No cross-file changes.
5. **Pattern established.** `parseAnalyticalParams` follows the same extraction pattern as `parseQueryKeyParams` — proven, idiomatic.

### Trade-offs

1. **One additional function call per request.** Negligible performance impact — function is inlined by the compiler.
2. **Handler file still at 501 lines.** Not dramatically reduced, but comfortably under the 620 ceiling with room for growth.
3. **Codegen not addressed.** This is intentional — codegen is a strategic improvement, not a minimum viable unblock.

### Open Debts

| Item | Status | Priority | Next Gate |
|------|--------|----------|-----------|
| Codegen implementation | Deferred | High | Family 07 boundary (mandatory) |
| Reader 10-param signature | Monitored | Low | Absorbed by codegen |
| Schema coherence tooling | Deferred | Low | ~12 tables / 100+ columns |
| 9 low-severity deferred items | Stable | Low | Various |

## Blockers for Family 06

See `family-06-blockers-and-hardening-success-criteria.md` for the formal gate document.

**Summary:** The sole blocker (H-5 handler extraction) has been resolved. Family 06 is unblocked pending gate confirmation.

## Preparation for S190

Recommended next stage options:

1. **Family 06 trigger assessment and selection.** Evaluate candidates, select the next family, define the analytical contract. This is the natural next step if the team wants to continue expanding coverage.

2. **Codegen scope definition.** Define templates, inputs, outputs, and generation approach for readers, handlers, use cases, and tests. This is the strategic investment that makes all future expansions ~2 minutes instead of ~45 minutes.

3. **Wave B retrospective and pattern sustainability report.** Synthesize learnings from 6 families + 2 hardening tranches into a durable reference. Useful if the team wants to capture institutional knowledge before changing the expansion model.

**Recommendation:** S190 should be the Family 06 trigger assessment. Codegen can follow as S191 before Family 07. The handler extraction provides sufficient runway for one more manual expansion.

## Deliverables

| # | Document | Status |
|---|----------|--------|
| 1 | `docs/architecture/pre-family-06-mandatory-hardening-tranche.md` | Complete |
| 2 | `docs/architecture/triggered-vs-deferred-hardening-items-after-family-05.md` | Complete |
| 3 | `docs/architecture/family-06-blockers-and-hardening-success-criteria.md` | Complete |
| 4 | `docs/stages/stage-s189-pre-family-06-mandatory-hardening-tranche-report.md` | This document |
| 5 | H-5 implementation (`parseAnalyticalParams` extraction) | Complete — verified |
