# Stage S245 — Remote CI Closure for Breadth Wave Report

**Date:** 2026-03-21
**Charter:** BREADTH-WAVE-1 (hardening tranche)
**Type:** CI proof and debt closure
**Predecessor:** S244 (breadth integration and gate)
**Status:** COMPLETE

---

## 1. Executive Summary

S245 closed the high-severity debt D3 from the BREADTH-WAVE-1 charter: remote CI verification of all accumulated changes from S241–S244.

The first CI push exposed a real defect — a multi-statement ClickHouse migration that passed locally but failed remotely. The fix was applied and the second CI run achieved full green across all 4 pipeline jobs. This validates the project's requirement that remote CI evidence must exist before any new feature wave.

**D3 status: CLOSED.**

---

## 2. Remote CI Proof

| Metric | Value |
|--------|-------|
| Final green run | [23375533952](https://github.com/FabioCaffarello/market-foundry/actions/runs/23375533952) |
| Commit validated | `516236d` (includes `95c7cc2` breadth wave + migration fix) |
| Unit Tests | PASS (1m31s) |
| Codegen Golden Equivalence | PASS (30s) |
| Integration Tests | PASS (1m34s) |
| Smoke Analytical E2E | PASS (7m23s) |
| Total pipeline time | ~10m58s |

---

## 3. Defect Discovered and Fixed

| Field | Detail |
|-------|--------|
| What | Migration 007 used two `ALTER TABLE` statements in one file |
| Where | `deploy/migrations/007_add_decision_severity_rationale.sql` |
| Symptom | ClickHouse error code 62: "Multi-statements are not allowed" |
| Why invisible locally | Local stack either pre-applied migrations or used a different execution path |
| Fix | Combined into single `ALTER TABLE ... ADD COLUMN ..., ADD COLUMN ...` |
| Fix commit | `516236d` |

This is the exact class of defect that remote CI exists to catch.

---

## 4. Evidence and Scope

### 4.1 What Was Proved

- All breadth wave production code compiles and passes unit tests on ubuntu-latest
- All 10 codegen families (including 3 new) validate and match golden snapshots
- Actor chain integration tests pass with embedded NATS
- Full Docker Compose stack boots, migrations apply, and HTTP smoke tests pass
- The migration fix is correct and idempotent

### 4.2 What Was NOT Proved

- Smoke E2E does not yet exercise the 3 new breadth types (ema_crossover, trend_following_entry, drawdown_limit) — this remains debt D1, recommended for S246
- Chain B integration test with drawdown_limit remains D2 (low severity)

---

## 5. Debt Ledger Update

| # | Debt | Before S245 | After S245 |
|---|------|-------------|------------|
| D1 | Smoke test coverage for 3 new types | Medium — open | Medium — open (recommended S246) |
| D2 | Chain B integration test with drawdown_limit | Low — open | Low — open |
| D3 | Remote CI verification of accumulated changes | **High — open** | **CLOSED** |

---

## 6. Files Changed in S245

### Modified
- `deploy/migrations/007_add_decision_severity_rationale.sql` — fix multi-statement syntax

### New
- `docs/architecture/remote-ci-closure-for-breadth-wave.md`
- `docs/architecture/breadth-wave-remote-ci-evidence-log.md`
- `docs/stages/stage-s245-remote-ci-closure-for-breadth-wave-report.md`

---

## 7. Limits and Observations

1. **Migration tooling gap:** The migration runner does not validate statement count before applying. Consider adding a lint rule to catch multi-statement migrations before they reach CI.
2. **Go module cache:** Both CI runs showed cache misses for `go.sum`. This affects build speed (~30s overhead) but not correctness. The root cause is that `go.sum` lives in module subdirectories, not at the repo root.
3. **Node.js 20 deprecation:** GitHub Actions will force Node.js 24 starting June 2026. The project should update action versions before then.

---

## 8. Preparation for S246

S246 should expand the smoke E2E to cover the 3 new breadth types, closing debt D1:

1. **ema_crossover chain:** Seed an EMA crossover analytical, verify decision appears at `GET /decision/ema_crossover/latest`
2. **trend_following_entry chain:** Verify strategy appears at `GET /strategy/trend_following_entry/latest`
3. **drawdown_limit chain:** Verify risk assessment appears at `GET /risk/drawdown_limit/latest`

This would complete the breadth wave hardening and make the codebase fully ready for the next feature charter.
