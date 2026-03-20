# Stage S226 — Real CI-on-Push Closure Report

**Date:** 2026-03-20
**Type:** Operational evidence closure
**Scope:** Close the inherited CI-on-push pending trail by executing real remote runs, capturing objective evidence, and updating the gate corpus
**Status:** COMPLETE — evidence trail closed; gate remains FAIL

---

## 1. Executive Summary

S226 treated CI-on-push as a formal tranche-exit artifact, not as an implied consequence of local verification.

The stage executed real pushes to `main`, inspected the actual GitHub Actions outcomes, removed bounded mechanical blockers revealed by the runner, and converted the inherited pending condition into a formal remote adjudication.

That adjudication is now explicit:

1. CI-on-push is real and repeatedly exercised,
2. the inherited `PENDING` state is closed,
3. XC-6 / EC-7 is **FAIL**, not **PASS**,
4. the decisive remaining fault is the failed analytical smoke step on run `23361711481`, where the runner reported endpoint `503` and missing ClickHouse configuration in the gateway path under test.

---

## 2. What Was Executed

S226 executed the following remote sequence:

1. reviewed the inherited remote failure on run `23355160034`,
2. corrected the first set of remote blockers and pushed commit `61fd1362fdfd4155f43164733ce7a4f32ab4ec7c`,
3. inspected run `23360347842` and its failure artifact,
4. corrected the second set of remote blockers and pushed commit `f9c1b631701bacfcde5b55ab5bd93b0c09f765a2`,
5. inspected run `23360780900` and its failure artifact,
6. corrected the control-route collision and pushed commit `ece78165099597933889dbd356cd6c678fbe3726`,
7. added gateway-readiness discipline and pushed commit `ca6df12b4149f2ca575319c04c61248502ff64f0`,
8. corrected the seed identifier parsing and pushed commit `43aa2b01c41d8490477174aaf49ff3276d49247f`,
9. adjudicated the resulting remote run `23361711481`.

Local preconditions were revalidated along the way:

- `make check` — PASS
- `make verify` — PASS
- targeted HTTP tests — PASS

These local checks were used as discipline, not as substitute evidence.

---

## 3. Evidence Registered

S226 produced and aligned the following gate-facing artifacts:

- `docs/architecture/real-ci-on-push-closure.md`
- `docs/architecture/ci-evidence-log-and-gate-satisfaction.md`
- `docs/stages/stage-s226-real-ci-on-push-closure-report.md`

These documents record:

1. which real runs were executed,
2. which commit SHAs they correspond to,
3. what each run proved,
4. which defects were mechanical and are now closed,
5. which limitation remains after the final adjudication run,
6. why the current gate state is a formal remote `FAIL` rather than a local or pending inference.

---

## 4. Limits and Scope Discipline

S226 deliberately did not:

1. redesign the GitHub Actions workflow,
2. open a new functional slice,
3. introduce new operational capability,
4. treat local success as final proof,
5. suppress the final remote limitation once the runner exposed it.

The stage remained bounded to real-pipeline proof and evidence capture.

---

## 5. Formal Disposition

The honest final disposition of S226 is:

1. the CI-on-push trail is formally closed as an evidence problem,
2. XC-6 / EC-7 is no longer `PENDING`,
3. XC-6 / EC-7 is currently **FAIL** on run `23361711481`,
4. codegen and unit are green remotely,
5. smoke is not green remotely because `Run smoke-analytical E2E` failed after startup, readiness, seed, and writer flush had already passed.

This is sufficient for S227 because the remaining work is now sharply bounded to a real, archived runtime defect instead of a vague CI verification debt item.
