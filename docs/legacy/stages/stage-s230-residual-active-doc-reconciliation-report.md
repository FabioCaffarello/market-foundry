# Stage S230 — Residual Active Doc Reconciliation Report

**Date:** 2026-03-20
**Type:** Surgical documentation reconciliation
**Scope:** Close the residual active-doc drift identified in S228
**Status:** COMPLETE — all four S228 drift items resolved

---

## 1. Executive Summary

S230 closed the last documentation gap identified in S228.

Four drift items in three active reference documents were corrected:
- 1 ambiguous database wording clarified,
- 2 flat registry paths updated to per-domain package structure,
- 1 deprecated codegen marker format replaced with current format.

The corrections are minimal, traceable, and scoped strictly to active
reference docs. No historical records were altered. No new content was added.

---

## 2. Objective

Reconcile the residual active-doc drift so that:
1. developers following active docs produce code matching the current architecture,
2. the documentation corpus stops contradicting the tooling and gates,
3. the base is ready for a clean remote CI proof in S231.

---

## 3. Drift Resolved

| # | S228 Item | File Changed | Change |
|---|-----------|-------------|--------|
| 1 | obsolete default-database execution-flow wording | `cmd-migrate-and-migration-catalog.md` | "default database" → "initial bootstrap connection to system database" |
| 2 | obsolete flat registry target path | `codegen-current-usage-boundaries-and-limitations.md` | `signal_registry.go` → `natssignal/registry.go` (2 lines) |
| 3 | obsolete flat registry target path | `codegen-specification-and-schema.md` | `{domain}_registry.go` → `nats{domain}/registry.go` |
| 4 | obsolete codegen marker example | `codegen-specification-and-schema.md` | `BEGIN/END CODEGEN MANAGED SECTION` → `codegen:begin/end` |

---

## 4. Files Changed

Active docs corrected:
```
docs/architecture/cmd-migrate-and-migration-catalog.md
docs/architecture/codegen-current-usage-boundaries-and-limitations.md
docs/architecture/codegen-specification-and-schema.md
```

Companion documentation:
```
docs/architecture/residual-active-doc-reconciliation.md
docs/architecture/residual-active-docs-change-log.md
docs/stages/stage-s230-residual-active-doc-reconciliation-report.md (this file)
```

---

## 5. Files NOT Changed (intentional exclusions)

11 documents containing old references were reviewed and excluded because they are:
- historical stage records where old paths were correct at time of writing,
- diagnostic documents that cite drift as a finding (not as guidance),
- documents that already self-document the supersession.

Full exclusion list with rationale: `docs/architecture/residual-active-docs-change-log.md`.

---

## 6. Validation

```
make check          → PASS (84 checks, 0 errors, 0 warnings)
make quality-gate-ci → PASS (84 checks, 0 errors, 0 warnings)
```

---

## 7. Limits and Trade-offs

1. **Historical docs untouched** — 11 docs contain old references in historical
   context. Changing them would falsify the record. They are correctly excluded.

2. **Document count unchanged** — the architecture corpus remains large (~270 files).
   S230 was scoped to reconcile drift, not consolidate volume.

3. **Codegen spec `mappers.go` reference** — `codegen-specification-and-schema.md`
   line 162 still references `cmd/writer/mappers.go` as a referential integrity
   check target. This is a spec-level path (where the mapper artifact would go),
   not a current-state assertion. Left as-is.

---

## 8. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Active corpus stops contradicting architecture/tooling | **MET** — all 4 items fixed |
| Residual drift reduced to minimum | **MET** — remaining refs are historical |
| Reconciliation is traceable | **MET** — change log documents every change and exclusion |
| Base ready for remote CI proof (S231) | **MET** — local gates green, docs aligned |
| Scope remained surgical | **MET** — 3 files changed, 4 edits total |

---

## 9. Preparation for S231

S231 should execute the final remote CI proof:

1. **Commit S229 + S230 changes** — the raccoon-cli analyzer fixes and doc
   reconciliation are ready to push.
2. **Push to main** — trigger GitHub Actions CI pipeline.
3. **Capture evidence** — record the run ID, commit SHA, and job results.
4. **Update evidence log** — if CI passes, update
   `ci-evidence-log-and-gate-satisfaction.md` with the new PASS entry.
5. **Adjudicate XC-6 / EC-7** — if all required jobs pass, upgrade the
   gate status from FAIL to PASS.

The S229/S230 corrections ensure that `quality-gate-ci` is an honest gate
and the documentation is aligned. S231 is the final mechanical step:
proving it on the remote runner.
