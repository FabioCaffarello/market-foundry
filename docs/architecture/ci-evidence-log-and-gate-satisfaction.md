# CI Evidence Log and Gate Satisfaction

**Date:** 2026-03-20  
**Stage:** S226  
**Purpose:** Provide a compact evidence ledger for the inherited CI-on-push gate item and state exactly how the evidence maps to XC-6 / EC-7.

---

## 1. Evidence Log

| Run ID | Commit SHA | Trigger | Job results | Evidence value |
|---|---|---|---|---|
| `23355160034` | `13838ecbef2be33ab23717c29f88108155300a78` | push to `main` | codegen **FAIL**, unit **PASS**, smoke **FAIL** | Proved that the remote gate was not yet operationally satisfied |
| `23360347842` | `61fd1362fdfd4155f43164733ce7a4f32ab4ec7c` | push to `main` | codegen **PASS**, unit **PASS**, smoke **FAIL** | Proved that remote codegen and unit gates were real and green; isolated smoke/runtime defects |
| `23360780900` | `f9c1b631701bacfcde5b55ab5bd93b0c09f765a2` | push to `main` | codegen **PASS**, unit **PASS**, smoke **FAIL** | Proved the first smoke defects were actually removed and narrowed the remaining blocker |
| `23361089050` | `ece78165099597933889dbd356cd6c678fbe3726` | push to `main` | codegen **PASS**, unit **PASS**, smoke **FAIL** | Proved route repair alone was insufficient; seed-stage failure still existed on the runner |
| `23361436729` | `ca6df12b4149f2ca575319c04c61248502ff64f0` | push to `main` | codegen **PASS**, unit **PASS**, smoke **FAIL** | Proved gateway readiness wait did not close the remote seed-stage failure |
| `23361711481` | `43aa2b01c41d8490477174aaf49ff3276d49247f` | push to `main` | codegen **PASS**, unit **PASS**, smoke **FAIL** | Converted the inherited pending item into a formal remote verdict: CI-on-push exists and runs, but the gate is still red at the analytical smoke step |

---

## 2. Remote Defect Narrowing Log

| Sequence | Remote defect observed | Evidence source | Resulting action |
|---|---|---|---|
| A | Integrated codegen target path stale on runner | run `23355160034` codegen failure | corrected integrated target path and governed snapshot section |
| B | CI runner lacked materialized `deploy/envs/local.env` | run `23355160034` smoke startup failure | materialized env file from example in workflow |
| C | `/execution/status/latest` route collision | run `23360347842` compose logs artifact | moved status handling into wildcard route path |
| D | JetStream storage exhausted on runner | run `23360347842` compose logs artifact | reduced stream retention footprint to CI/local-safe size |
| E | `/execution/control` route collision | run `23360780900` compose logs artifact | moved control handling into wildcard route path with explicit validation |
| F | Seed script expected `config.version_id`, but the returned draft identifier could be `config.id` | repeated `Seed configctl` failures on runs `23361089050` and `23361436729`, then fresh clean-stack reproduction | updated `scripts/seed-configctl.sh` to accept both fields |
| G | Analytical smoke fails with endpoint `503`; gateway reports no ClickHouse config for the path under test | failed step `Run smoke-analytical E2E` on run `23361711481` | no S226 redesign or functional expansion; limitation carried forward explicitly for S227 |

---

## 3. Gate Satisfaction Rule

### XC-6 / EC-7 is PASS only if:

1. a real push to the remote repository triggered GitHub Actions,
2. all CI jobs required by the gate finished green,
3. the green run is tied to a concrete commit SHA,
4. the outcome is archived in the gate corpus.

### XC-6 / EC-7 remains FAIL if:

1. the push happened but any required remote job failed,
2. the remaining failure is real runner evidence rather than local inference,
3. the failure is recorded in the gate corpus.

---

## 4. S226 Satisfaction Statement

S226 satisfies the evidence-closure objective:

1. real pushes were executed,
2. remote runs were captured with run IDs and SHAs,
3. observed failures were diagnosed from runner evidence rather than inferred locally,
4. the evidence corpus documents the full narrowing trail,
5. XC-6 / EC-7 no longer depends on local-only interpretation.

The current gate adjudication is:

1. **XC-6 / EC-7 = FAIL**
2. adjudicating run: `23361711481`
3. adjudicating commit: `43aa2b01c41d8490477174aaf49ff3276d49247f`
4. decisive failed step: `Smoke Analytical E2E -> Run smoke-analytical E2E`
5. decisive observed condition: analytical endpoint returned `503`, and the smoke script reported missing ClickHouse configuration in the gateway path under test

This means S226 closes the inherited pending state, but it does **not** upgrade the gate to PASS.

---

## 5. Residual Non-Blocking Notes

The remote runs also emitted recurring annotations that do not, by themselves, adjudicate XC-6 / EC-7:

1. Node.js 20 deprecation warnings for `actions/checkout@v4`, `actions/setup-go@v5`, and `actions/upload-artifact@v4`,
2. cache restore warning due to absent root-level `go.sum`,
3. a `cmd/migrate/migrate` annotation emitted alongside otherwise green unit/codegen jobs.

These are operational hygiene items, not the decisive gate defect recorded by S226.
