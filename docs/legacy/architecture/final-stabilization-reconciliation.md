# Final Stabilization Reconciliation

**Date:** 2026-03-20
**Stage:** S227
**Scope:** Final reconciliation of the short closure tranche S223-S226 across code, tooling, active documentation, and gate evidence
**Status:** Reconciled locally; final remote gate proof still required

---

## 1. Purpose

S227 does not open a new charter.

Its purpose is to reconcile the last short closure tranche honestly so the repository stops describing multiple realities at once. The intended end state is:

1. code and runtime behavior aligned to the same operational baseline,
2. tooling and validation scripts aligned to the current repository and host environment,
3. active docs and evidence describing the same gate state,
4. any residual blocker named exactly, without euphemism.

---

## 2. What S227 Reviewed

S227 reviewed the frozen closure sequence and the artifacts produced in it:

1. S223 freeze and closure-plan artifacts,
2. S224 `raccoon-cli` / `quality-gate` reconciliation,
3. S225 active-doc drift closure,
4. S226 real CI-on-push evidence and the remote analytical-smoke failure trail.

The review confirmed that S224 and S225 closed their intended local/documentary surfaces, but two residual inconsistencies remained:

1. XC-1 had not been explicitly disposed; the repo had reconciled active docs, but the numeric count target still lacked a formal S227 decision.
2. The S226 evidence corpus correctly preserved the remote `FAIL`, but its final diagnosis overfit the smoke-script message "`Gateway has no ClickHouse config`" instead of the broader runtime-alignment drift that current local reproduction exposed.

---

## 3. Reconciliation Applied

### 3.1 Tooling and host-environment alignment

S227 removed three operational/tooling drifts that were still real on the current workspace:

1. `scripts/utils/lib.sh` no longer depends on Bash associative arrays, restoring compatibility with the repository host's `bash 3.2`.
2. `scripts/smoke-analytical-e2e.sh`, `scripts/diag-check.sh`, and `scripts/live-pipeline-activate.sh` no longer build `docker compose -f ...` as an unsafe string, so they now work from workspace paths containing spaces.
3. `make up` now waits for ClickHouse and applies `make migrate-up`, restoring the behavior already claimed by the Makefile help and operating docs.

### 3.2 Runtime/schema alignment

S227 reproduced the S226 analytical-smoke blocker locally and refined it to concrete runtime drift:

1. `cmd/migrate` bootstrapped the `market_foundry` database,
2. `deploy/configs/gateway.jsonc` and `deploy/configs/writer.jsonc` were still targeting `default`,
3. the writer insert SQL omitted explicit column lists after the DDL introduced `ingested_at`,
4. the smoke script queried ClickHouse table inventory against the wrong database and misclassified every analytical `503` as missing config.

S227 reconciled those surfaces by:

1. aligning gateway and writer ClickHouse database config to `market_foundry`,
2. updating writer insert SQL to specify the exact persisted columns for all analytical tables,
3. updating smoke diagnostics to query `market_foundry` explicitly and report `503` more honestly.

### 3.3 Evidence and active-doc alignment

S227 updated the active evidence/docs surface so it no longer preserves the wrong final interpretation as current truth:

1. `real-ci-on-push-closure.md` and `ci-evidence-log-and-gate-satisfaction.md` now carry an S227 reconciliation note explaining that the S226 `503` diagnosis was too narrow and was later reproduced locally as schema/bootstrap plus database-target drift.
2. Current-state config examples were aligned to `market_foundry`.
3. `system-vision.md` now reflects that the closure-and-reconciliation phase extended through S227 rather than stopping at S225.

---

## 4. Validation on the Reconciled Baseline

S227 validated the corrected local baseline with:

1. `make check` — **PASS**
2. `make verify` — **PASS**
3. `make up` — **PASS** (including automatic migration application)
4. `make seed` — **PASS**
5. `make smoke-analytical` — **PASS** on a clean stack

Most importantly, the corrected local baseline now proves the analytical path end-to-end again:

1. ClickHouse schema is present in `market_foundry`,
2. writer receives events and persists candle rows,
3. gateway analytical endpoints return `200`,
4. the smoke path no longer fails on the S226 failure class.

---

## 5. XC-1 Disposition

S223 required XC-1 to be closed explicitly, not implicitly.

S227 closes XC-1 by **formal re-baseline**, not by reopening a repository-wide archival campaign.

### 5.1 Why the old numeric target is no longer a useful gate criterion

The inherited target from S209/S211/S216 was:

- `docs/architecture/` active docs `<= 150`

That target is no longer a reliable closure criterion for this tranche because:

1. the closure tranche itself necessarily produced gate, evidence, and reconciliation artifacts,
2. the repository intentionally preserved active governance traceability rather than collapsing historical gate material into a fresh archival wave,
3. S223 explicitly allowed re-baselining if bounded archival would reopen scope.

### 5.2 Evidence for the re-baseline decision

Measured corpus counts:

1. before S227 deliverables: `docs/architecture/ = 263`, `docs/stages/ = 223`
2. after S227 deliverables: `docs/architecture/ = 265`, `docs/stages/ = 224`

S227 therefore ratifies the closure criterion as:

1. active docs used as current guidance must align with current code, tooling, and gate evidence,
2. historical documents may remain active only when they are explicitly framed as stage snapshots,
3. numeric count alone is no longer a gate blocker for this tranche.

Under that ratified rule, XC-1 is **CLOSED**.

---

## 6. Final Reconciled State

| Surface | S227 result |
|---|---|
| Local guard rails | **PASS** |
| Local Go/test baseline | **PASS** |
| Local analytical smoke | **PASS** |
| Active-doc drift from S225/S226 | **RECONCILED** |
| XC-1 disposition | **CLOSED by formal re-baseline** |
| XC-6 / EC-7 remote proof on S227 baseline | **NOT YET RE-RUN** |
| XC-11 repository tag | **NOT CREATED** |

---

## 7. Residual Blocker

S227 leaves one real blocker before a clean final gate:

1. the corrected S227 baseline has not yet been pushed and revalidated by GitHub Actions, so XC-6 / EC-7 cannot be upgraded from historical `FAIL` to fresh `PASS`, and XC-11 remains blocked behind that missing remote proof.

This is a real blocker, not a local ambiguity.

The old blocker has changed shape:

1. it is no longer an unexplained analytical `503`,
2. it is now the absence of fresh remote proof on the reconciled baseline.

---

## 8. Readiness for S228

S228 should not spend time rediscovering tranche-closure drift.

The repository is now ready for S228 in the specific sense that:

1. local code/tooling/runtime/docs align on one coherent baseline,
2. the remaining gate action is narrow and mechanical,
3. the next decision can be made from a stable closure record instead of competing interpretations.

The correct immediate next move is:

1. rerun CI on push against the S227 baseline,
2. if green, create the gate tag,
3. only then treat expansion discussion as reopened.
