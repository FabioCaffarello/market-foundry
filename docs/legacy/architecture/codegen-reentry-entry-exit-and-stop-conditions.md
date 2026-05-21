# Codegen Re-entry: Entry, Exit, and Stop Conditions

**Stage:** S258
**Charter:** codegen-reentry-charter-and-scope-freeze.md
**Date:** 2026-03-21

---

## 1. Entry Conditions

All entry conditions must be satisfied before proceeding to S259 (spec/template reconciliation).

| ID | Condition | Evidence | Status |
|---|---|---|---|
| EN-1 | BEHAVIORAL-WAVE-1 formally closed with PASS | S257 gate report | MET |
| EN-2 | Zero medium-or-higher-risk behavioral debts | S257 debt ledger | MET |
| EN-3 | 47 behavioral tests passing in CI | CI logs | MET |
| EN-4 | Codegen module structurally intact (spec.go, render.go, compare.go, main.go) | File existence | MET |
| EN-5 | All 10 family specs parseable by `codegen validate-all` | `codegen validate-all` exit 0 | TO VERIFY at S259 start |
| EN-6 | All 20 golden snapshots present | File count in `golden-snapshots/` | MET |
| EN-7 | raccoon-cli guardian operational | CI step presence | MET |
| EN-8 | Charter (this document + scope freeze) approved | Document existence | MET (this document) |

---

## 2. Exit Conditions

All exit conditions must be satisfied to close the codegen re-entry wave at S262 (post-codegen gate).

| ID | Condition | Verification method |
|---|---|---|
| EX-1 | All 10 specs validate individually | `codegen validate <spec>` × 10, all exit 0 |
| EX-2 | Cross-spec validation passes | `codegen validate-all` exit 0 |
| EX-3 | All 20 golden snapshots match generated output | `codegen check-all` exit 0 |
| EX-4 | `integrated.yaml` contains exactly 20 entries | YAML parse + count = 20 |
| EX-5 | Each entry in `integrated.yaml` has valid `target`, `marker`, `golden`, `spec` | Field presence validation |
| EX-6 | All 10 target files contain `codegen:begin`/`codegen:end` marker pairs | Grep count = 20 pairs (10 files × 2 artifacts each, or distributed across target files) |
| EX-7 | Code within markers matches golden snapshots (post-normalization) | `codegen compare` per family per artifact |
| EX-8 | 47 behavioral tests pass | `go test ./...` in CI |
| EX-9 | CI pipeline includes `codegen check-all` step | CI config inspection |
| EX-10 | No new template files created | `ls codegen/templates/` = 2 files |
| EX-11 | No new family spec files created | `ls codegen/families/` = 10 files |
| EX-12 | No domain logic generated | Code review of changes |

---

## 3. Stop Conditions

A stop condition triggers an immediate pause in the wave. Work must not continue until the condition is resolved or the charter is formally amended.

### 3.1 Hard stops (immediate halt, no workaround)

| ID | Condition | Action |
|---|---|---|
| ST-1 | Any behavioral test regresses (47-test suite) | Halt codegen work; fix regression; re-verify all behavioral tests before resuming |
| ST-2 | Codegen change modifies code outside `codegen:begin`/`codegen:end` markers in target files | Revert change; investigate root cause |
| ST-3 | New artifact template created | Revert; escalate to new charter |
| ST-4 | New family spec file created | Revert; escalate to new charter |
| ST-5 | Domain logic appears in templates (behavioral rules, scaling factors, thresholds) | Revert; this crosses the human-decision boundary |

### 3.2 Soft stops (pause and assess, may resume with justification)

| ID | Condition | Action |
|---|---|---|
| ST-6 | More than 3 spec files require field corrections beyond typos | Assess whether specs were built on outdated assumptions; may need spec audit stage |
| ST-7 | Template change breaks more than 2 golden snapshots simultaneously | Assess whether template is too brittle; may need Tier 2 template refinement |
| ST-8 | Manual code at integration point diverges significantly from generated output | Assess whether manual code has evolved beyond codegen's current capability; document as debt |
| ST-9 | Integration requires modifying imports or types outside markers | Assess whether target file structure supports codegen insertion; may need manual preparation |
| ST-10 | Tier 2 work exceeds 15% of total effort | Pause Tier 2; refocus on Tier 1 completion |

---

## 4. Amendment Conditions

The charter may be amended (not replaced) under these conditions:

| Condition | Allowed amendment | NOT allowed |
|---|---|---|
| Spec field needs a new `DerivedFields` computation | Add to existing `DerivedFields` struct | Add new template artifact |
| Template needs a minor conditional for layer-specific behavior | Add conditional using existing `Derived` fields | Add new template function |
| Golden snapshot format needs adjustment | Regenerate snapshots | Hand-edit snapshots |
| Target file location changed | Update `integrated.yaml` target path | Create new target files |
| CI step needs adjustment | Modify existing codegen CI step | Create separate CI workflow |

---

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Spec drift from domain | Medium | Spec fields don't match runtime values | Reconciliation in S259 catches this early |
| Template brittleness | Low | Template change cascades across snapshots | Golden comparison catches regressions immediately |
| Behavioral test regression | Low | Codegen change breaks behavioral guarantee | Hard stop ST-1; behavioral tests run before codegen CI |
| Scope creep via "small" additions | Medium | Gradual expansion beyond charter | Prohibited changes list; escalation rules; amendment log |
| Manual→generated divergence | Medium | Manual code evolved past codegen capability | Document as debt; do not force-fit generated code |
| Integration marker conflicts | Low | Markers interfere with existing code structure | Manual review of each target file before marker insertion |

---

## 6. Monitoring Checkpoints

| Checkpoint | When | What to verify |
|---|---|---|
| CP-1 | After S259 (reconciliation) | All 10 specs validate; all 20 snapshots match; zero template changes needed OR template changes documented |
| CP-2 | After each family integration (S260) | Markers inserted correctly; `integrated.yaml` updated; behavioral tests still pass |
| CP-3 | After CI gate setup (S261) | `codegen check-all` runs in CI; failure on intentional drift proven |
| CP-4 | S262 gate | All exit conditions met; no stop conditions active; debt ledger updated |

---

## 7. Relationship to Deferred Debts

The following debts from BEHAVIORAL-WAVE-1 remain deferred and are explicitly NOT addressed by this wave:

| Debt | Status | Interaction with codegen wave |
|---|---|---|
| OD-BW2: Configurable scaling factors | Deferred | Codegen must NOT parameterize AckWait/MaxDeliver (blocked by this debt) |
| OD-BW5: Performance budgets | Deferred | No codegen performance targets set |
| OD-BW6: Configctl activation | Deferred | Codegen does not interact with configctl |
| OD-BW4 remainder: Full severity validation | Deferred | Codegen does not generate severity logic |
| OD-BW7: Execution layer | Out of scope | paper_order spec exists but execution layer is not expanded |
