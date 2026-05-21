# Real CI-on-Push Closure

**Date:** 2026-03-20  
**Stage:** S226  
**Purpose:** Convert the inherited CI-on-push pending item into explicit remote evidence, using real pushes and real GitHub Actions runs rather than local inference.

**S227 reconciliation note:** The remote S226 run facts are unchanged. What changed in S227 is the interpretation of the final analytical `503`: local reproduction on 2026-03-20 showed that the smoke script's "missing ClickHouse config" message was too narrow. The failure surface also included omitted schema bootstrap in `make up`/CI, runtime configs still pointing to `default` while migrations targeted `market_foundry`, and writer insert SQL that omitted explicit column lists after the DDL introduced `ingested_at`. Those local drifts were reconciled; remote CI still needs a fresh run on the corrected baseline.

---

## 1. Scope Discipline

S226 is not a new feature stage and is not a CI redesign stage.

Its narrow purpose is:

1. execute the real push-triggered CI flow,
2. inspect the actual remote result,
3. remove bounded mechanical blockers revealed only by the remote runner,
4. record the evidence in gate-facing form,
5. state explicitly whether XC-6 / EC-7 is now satisfied.

This stage treats CI-on-push as a formal gate artifact, not as an incidental follow-up.

---

## 2. Real Push Sequence Executed

S226 converted the inherited pending trail into a concrete remote run ledger:

| Sequence | Commit SHA | Run ID | Result | What it proved |
|---|---|---|---|---|
| Baseline inherited from pre-S226 | `13838ecbef2be33ab23717c29f88108155300a78` | `23355160034` | **FAIL** | The declared CI flow existed remotely, but the real runner exposed unresolved breakage that local-only validation had not proven away |
| First remediation push | `61fd1362fdfd4155f43164733ce7a4f32ab4ec7c` | `23360347842` | **FAIL** | Codegen and unit jobs were green remotely; smoke advanced far enough to reveal runtime-specific failures rather than setup-only failures |
| Second remediation push | `f9c1b631701bacfcde5b55ab5bd93b0c09f765a2` | `23360780900` | **FAIL** | The first smoke blockers were removed; the remaining defect narrowed to `/execution/control` route collision in the live smoke path |
| Third remediation push | `ece78165099597933889dbd356cd6c678fbe3726` | `23361089050` | **FAIL** | Route conflicts were removed, but the runner still failed in `Seed configctl`, proving one more contract-level blocker remained |
| Fourth remediation push | `ca6df12b4149f2ca575319c04c61248502ff64f0` | `23361436729` | **FAIL** | Gateway readiness wait was not the missing piece; smoke still failed at `Seed configctl` on the real runner |
| Closure adjudication push | `43aa2b01c41d8490477174aaf49ff3276d49247f` | `23361711481` | **FAIL** | Codegen, unit, startup, readiness, and seeding all passed remotely; the remaining failure is now an explicit analytical smoke defect, not a CI-on-push inference gap |

---

## 3. Remote Findings Actually Observed

### 3.1 Baseline runner breakage

Run `23355160034` failed in two distinct places:

1. `Codegen Golden Equivalence` failed at `Verify integrated slices match golden snapshots`.
2. `Smoke Analytical E2E` failed during stack startup on the clean runner.

This invalidated any claim that local verification alone was enough to satisfy XC-6 / EC-7.

### 3.2 First remote narrowing wave

Run `23360347842` on `61fd136...` produced:

- `Codegen Golden Equivalence` — `success`
- `Unit Tests` — `success`
- `Smoke Analytical E2E` — `failure`

The failed `Seed configctl` step and uploaded compose logs exposed two runner-real defects:

1. gateway route collision between `/execution/status/latest` and `/execution/:type`,
2. JetStream allocation failure with `err_code=10047` and `insufficient storage resources available`.

### 3.3 Second remote narrowing wave

Run `23360780900` on `f9c1b63...` again produced green codegen and unit jobs, but smoke still failed at `Seed configctl`.

The compose logs reduced the remaining fault to a second router collision:

- `/execution/control` conflicted with the wildcard prefix `/execution/:type`.

### 3.4 Seed-stage contract failure

Run `23361089050` on `ece78165...` and run `23361436729` on `ca6df12...` both failed after startup and readiness, again at `Seed configctl`.

The direct runner evidence proved that:

1. startup was no longer the issue,
2. gateway readiness was no longer the issue,
3. the remaining blocker was inside the seeding contract itself.

The specific root cause was then established from combined evidence:

- remote failure location stayed fixed at `Seed configctl`,
- a fresh local reproduction against a clean stack succeeded only after changing the script to accept `config.id` as well as `config.version_id`.

That makes the seed-parser diagnosis an inference from runner evidence plus fresh reproduction, not a guess.

### 3.5 Final remote adjudication

Run `23361711481` on `43aa2b0...` produced the first end-to-end remote progression past the seed phase:

- `Codegen Golden Equivalence` — `success`
- `Unit Tests` — `success`
- `Smoke Analytical E2E` reached:
  - `Start stack (compose up)` — `success`
  - `Wait for infrastructure readiness` — `success`
  - `Seed configctl` — `success`
  - `Wait for writer flush` — `success`
  - `Run smoke-analytical E2E` — `failure`

The failed smoke log recorded the remaining runner-real defect explicitly:

1. analytical endpoint availability returned `503`,
2. the smoke script reported `Gateway has no ClickHouse config. Check deploy/configs/gateway.jsonc.`,
3. S227 later reproduced the same failure class locally and traced it to schema/bootstrap and database-target drift rather than an actually absent ClickHouse section

This is the decisive S226 result: the inherited CI-on-push item is no longer pending or inferential, but XC-6 / EC-7 is still not green.

---

## 4. Bounded Corrections Applied During S226

The stage changed only what was necessary to close remote mechanical blockers:

### 4.1 Workflow and codegen corrections

- materialized `deploy/envs/local.env` in CI from `deploy/envs/local.env.example`,
- corrected the integrated codegen target path to `internal/adapters/nats/natssignal/registry.go`,
- aligned the governed integrated slice with the actual generated snapshot output,
- added gateway readiness checking in the smoke workflow.

### 4.2 Runtime-path corrections

- removed the `/execution/status/latest` static-vs-wildcard router collision without changing the public request shape,
- reduced NATS stream `MaxBytes` values to a CI/local-safe retention size,
- removed the `/execution/control` static-vs-wildcard router collision by routing through the wildcard tree and validating `type=control`,
- corrected the seed script to accept the actual config draft identifier returned by the API.

No new operational feature was opened. No pipeline redesign was introduced.

---

## 5. Formal Gate Standard for XC-6 / EC-7

For this tranche, XC-6 / EC-7 is satisfied only when all of the following are true on a real push-triggered GitHub Actions run:

1. `Codegen Golden Equivalence` concludes `success`,
2. `Unit Tests` concludes `success`,
3. `Smoke Analytical E2E` concludes `success`,
4. the green result is tied to a specific closure-baseline commit SHA,
5. the evidence is recorded in the gate corpus.

Local `make check` and `make verify` remain necessary preconditions, but they are not substitute evidence.

---

## 6. Current S226 Disposition

S226 achieved formal closure of the CI-on-push evidence trail, but not a green gate:

1. the inherited CI-on-push item is no longer `PENDING`,
2. XC-6 / EC-7 can now be adjudicated from objective remote evidence,
3. the current adjudication is **FAIL**, based on run `23361711481`,
4. the remaining blocker captured by S226 is not startup, readiness, route collision, or seeding; it is the analytical smoke failure that S227 later refined from "missing ClickHouse config" to a broader runtime-alignment problem on the closure baseline.

Therefore S226 closes the evidence problem and leaves S227 with a precise, archivable runtime issue instead of a vague CI-on-push pending condition.
