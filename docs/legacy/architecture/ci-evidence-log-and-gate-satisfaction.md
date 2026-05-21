# CI Evidence Log and Gate Satisfaction

**Date:** 2026-03-20  
**Stage:** S226  
**Purpose:** Provide a compact evidence ledger for the inherited CI-on-push gate item and state exactly how the evidence maps to XC-6 / EC-7.

**S227 reconciliation note:** The S226 smoke script treated any analytical `503` as "missing ClickHouse config." Fresh local reproduction on 2026-03-20 showed the same failure class can also come from schema/bootstrap drift and database-target drift. On the reconciled baseline, the blocking condition was closed by aligning `make up`, `gateway`, `writer`, and `cmd/migrate` on `market_foundry` and rebuilding the writer inserts to match the live DDL. Remote CI evidence itself has not yet been rerun on that corrected baseline.

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
| `23365278860` | `5103f1c` | push to `main` | codegen **PASS**, unit **PASS**, smoke **FAIL** | S227-S231 consolidated push; smoke failed at `make up` due to Go 1.25 `cmd/migrate/migrate` stdlib collision |
| `23365571775` | `edb30107e2f1af3caf22490d90b9a709e5be6bdf` | push to `main` | codegen **PASS**, unit **PASS**, smoke **PASS** | **FIRST FULL GREEN** — all three CI jobs passed on the remote runner. Gate closure achieved. Tagged `v0.1.0-s231`. |

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
| G | Analytical smoke fails with endpoint `503`; the smoke script reported missing ClickHouse config, but S227 later confirmed that this failure class also covered schema/bootstrap and database-target drift | failed step `Run smoke-analytical E2E` on run `23361711481`, then local S227 reproduction on the closure baseline | no S226 redesign or functional expansion; the residual runtime drift was closed in S227 locally and still needs fresh remote proof |
| H | Go 1.25 `cmd/migrate/migrate` stdlib path collision — `make up` fails during docker build because `cmd/` prefix is now reserved | run `23365278860` build failure annotation and `make up` exit code 2 | renamed subpackage from `cmd/migrate/migrate` to `cmd/migrate/engine` with package name `engine` |
| I | All defects closed — first full green | run `23365571775` all jobs green | gate upgraded to PASS, tag `v0.1.0-s231` created |

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

## 4. Gate Satisfaction Statement

### S226 (historical)

S226 closed the evidence-closure objective but did not upgrade the gate to PASS.

### S231 (current — gate closed)

S231 executed the final remote CI proof:

1. **XC-6 / EC-7 = PASS**
2. adjudicating run: `23365571775`
3. adjudicating commit: `edb30107e2f1af3caf22490d90b9a709e5be6bdf`
4. all three required jobs: `Codegen Golden Equivalence` (PASS), `Unit Tests` (PASS), `Smoke Analytical E2E` (PASS)
5. release tag: `v0.1.0-s231`

The gate is now satisfied. The pending mechanical state has been fully closed.

---

## 5. Residual Non-Blocking Notes

The remote runs also emitted recurring annotations that do not, by themselves, adjudicate XC-6 / EC-7:

1. Node.js 20 deprecation warnings for `actions/checkout@v4`, `actions/setup-go@v5`, and `actions/upload-artifact@v4`,
2. cache restore warning due to absent root-level `go.sum`,
3. a `cmd/migrate/migrate` annotation emitted alongside otherwise green unit/codegen jobs.

These are operational hygiene items, not the decisive gate defect recorded by S226.
