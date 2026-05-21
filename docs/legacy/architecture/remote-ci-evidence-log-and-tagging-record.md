# Remote CI Evidence Log and Tagging Record

**Date:** 2026-03-20
**Stage:** S231
**Purpose:** Provide the formal record linking commit, CI run, job results, and release tag for the first full green baseline.

---

## 1. Validated Commit

| Field | Value |
|-------|-------|
| SHA | `edb30107e2f1af3caf22490d90b9a709e5be6bdf` |
| Short | `edb3010` |
| Message | `fix: rename migrate subpackage to avoid Go 1.25 cmd/ stdlib collision` |
| Branch | `main` |
| Date | 2026-03-20 |

---

## 2. CI Run Record

| Field | Value |
|-------|-------|
| Run ID | `23365571775` |
| Trigger | `push` to `main` |
| URL | `https://github.com/FabioCaffarello/market-foundry/actions/runs/23365571775` |
| Overall result | **success** |

### Job Results

| Job | Duration | Result |
|-----|----------|--------|
| Codegen Golden Equivalence | 25s | PASS |
| Unit Tests | 1m33s | PASS |
| Smoke Analytical E2E | 7m23s | PASS |

### Smoke Analytical E2E Steps

| Step | Result |
|------|--------|
| Build service binaries | PASS |
| Materialize compose env file | PASS |
| Start stack (compose up) | PASS |
| Wait for infrastructure readiness | PASS |
| Seed configctl | PASS |
| Wait for writer flush | PASS |
| Run smoke-analytical E2E | PASS |
| Scan for error-level logs | PASS |
| Tear down stack | PASS |

---

## 3. Release Tag

| Field | Value |
|-------|-------|
| Tag name | `v0.1.0-s231` |
| Type | Annotated |
| Target commit | `edb30107e2f1af3caf22490d90b9a709e5be6bdf` |
| Published to | `origin` (github.com:FabioCaffarello/market-foundry.git) |

---

## 4. Gate Adjudication

| Gate | Previous status | New status | Adjudicating evidence |
|------|----------------|------------|----------------------|
| XC-6 / EC-7 | FAIL (run `23361711481`) | **PASS** (run `23365571775`) | All three required jobs green on the remote runner |

---

## 5. Full CI Run History

| # | Run ID | Commit | Result | Stage |
|---|--------|--------|--------|-------|
| 1 | `23355160034` | `13838ec` | FAIL | S226 |
| 2 | `23360347842` | `61fd136` | FAIL | S226 |
| 3 | `23360780900` | `f9c1b63` | FAIL | S226 |
| 4 | `23361089050` | `ece7816` | FAIL | S226 |
| 5 | `23361436729` | `ca6df12` | FAIL | S226 |
| 6 | `23361711481` | `43aa2b0` | FAIL | S226 |
| 7 | `23365278860` | `5103f1c` | FAIL | S231 |
| 8 | `23365571775` | `edb3010` | **PASS** | S231 |

The defect narrowing trail spans 8 real pushes across 2 stages.
Each failure was diagnosed from runner evidence, not local inference.
