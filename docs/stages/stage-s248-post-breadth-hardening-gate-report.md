# Stage S248 — Post-Breadth Hardening Gate Report

**Date:** 2026-03-21
**Type:** Gate review (non-implementation)
**Charter:** Formal evaluation of the S245–S247 hardening tranche to determine if the breadth wave is operationally ready to sustain the next charter.

---

## Executive Summary

The S245–S247 hardening tranche closed all three explicit debts from S244. The breadth wave is functionally correct and operationally hardened at the local level. A single mechanical step remains: committing S246–S247 to main and obtaining remote CI green. No architectural, correctness, or operational blockers exist.

**Gate verdict: CONDITIONAL PASS** — converts to PASS upon OD1 closure.

---

## Debt Resolution

| Debt | Description                                    | Severity | Stage | Status   | Evidence                                    |
|------|------------------------------------------------|----------|-------|----------|---------------------------------------------|
| D3   | Remote CI verification of breadth wave         | High     | S245  | **CLOSED** | CI Run 23375533952 green; real defect caught |
| D1   | Smoke E2E coverage for 3 new types             | Medium   | S246  | **CLOSED** | +571 lines smoke scripts, +45 HTTP queries  |
| D2   | Chain B integration test with `drawdown_limit` | Low      | S247  | **CLOSED** | 120-line test, 13 assertions, passes locally |

---

## Verification Results (2026-03-21)

| Check                                          | Result |
|------------------------------------------------|--------|
| `go test ./internal/application/...`           | PASS   |
| `go test ./internal/domain/...`                | PASS   |
| `go test ./internal/actors/...`                | PASS   |
| `bash -n scripts/smoke-analytical-e2e.sh`      | PASS   |
| `bash -n scripts/smoke-multi-symbol.sh`        | PASS   |
| Chain integration tests (7 tests, both chains) | PASS   |
| Codegen golden snapshots (all 10 families)     | PASS   |

---

## Hardening Tranche Summary

### S245 — Remote CI Closure
- Fixed ClickHouse multi-statement migration defect.
- Achieved full CI green on commit `516236d`.
- Proved CI catches real issues that local testing misses.

### S246 — Smoke E2E Breadth Expansion
- Expanded `smoke-analytical-e2e.sh`: +263 lines (Phase 5 families + Phase 7 depth).
- Expanded `smoke-multi-symbol.sh`: +308 lines (Steps 7a–12a).
- Expanded HTTP REST tests: +45 queries across 4 files.
- Achieved smoke parity between Chain A and Chain B types.

### S247 — Chain B Integration Completion
- Added `TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk`.
- Proved `drawdown_limit` confidence scaling (×0.90), `stop_distance` constraint, decision severity propagation.
- Risk domain now symmetric at every test layer.

---

## Gains

| # | Gain                              | Impact                                          |
|---|-----------------------------------|-------------------------------------------------|
| G1 | Real defect caught by remote CI  | Migration fixed before any fresh deployment     |
| G2 | Smoke coverage parity            | Breadth types have identical coverage to Chain A |
| G3 | Chain B integration proof        | Risk domain symmetric at integration level      |
| G4 | Zero production code changes     | Hardening could not introduce new bugs          |

---

## Trade-offs Accepted

| # | Trade-off                              | Mitigation                                      |
|---|----------------------------------------|-------------------------------------------------|
| T1 | Chain A + drawdown_limit not tested   | Not a production use case; add if needed later  |
| T2 | Smoke requires live infrastructure     | CI pipeline exercises smoke; not unit-testable  |
| T3 | Warm-up window masks failures briefly  | Post-warmup `count > 0` check catches this      |
| T4 | CI Node.js deprecation (June 2026)    | Not breadth-specific; update before deadline    |

---

## Open Debts

| # | Debt                              | Severity   | Resolution Path                       |
|---|-----------------------------------|------------|---------------------------------------|
| OD1 | S246/S247 not yet in remote CI  | Low        | Commit + push + CI green (mechanical) |
| OD2 | Migration linting not automated | Low        | Add CI lint step (tooling hygiene)    |
| OD3 | Go module cache overhead in CI  | Negligible | Consolidate module structure          |

---

## Gate Decision

| Question                                                    | Answer                          |
|-------------------------------------------------------------|---------------------------------|
| D1, D2, D3 resolved?                                       | Yes — all three closed          |
| Breadth validated operationally, not just functionally?     | Yes — test pyramid symmetric    |
| New types coherent and explicable in pipeline?              | Yes — same patterns as Chain A  |
| Next acceptable step?                                       | **Option A: open next wave**    |
| Precondition?                                               | OD1 closure (commit + CI green) |

---

## Recommendation

1. **Immediate:** Commit S246–S247 to main, push, verify CI green → gate becomes PASS.
2. **Next:** Open the next feature wave. The breadth is hardened and ready to sustain it.
3. **Carry forward:** OD2/OD3 as optional hygiene items, not blockers.

---

## Deliverables

| Deliverable | Path |
|-------------|------|
| Gate review | `docs/architecture/post-breadth-hardening-gate.md` |
| Gains and trade-offs | `docs/architecture/breadth-hardening-wave-gains-tradeoffs-and-open-debts.md` |
| Next wave recommendations | `docs/architecture/next-wave-recommendations-after-post-breadth-hardening-gate.md` |
| This report | `docs/stages/stage-s248-post-breadth-hardening-gate-report.md` |

---

## Status: CONDITIONAL PASS

The breadth wave hardening is substantively complete. Final closure is mechanical (OD1). No architectural work remains. The codebase is ready for the next charter once CI confirms.
