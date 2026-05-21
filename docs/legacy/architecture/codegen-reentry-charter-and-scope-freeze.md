# Codegen Re-entry Charter and Scope Freeze

**Stage:** S258
**Status:** OPEN — scope frozen
**Predecessor:** S257 (Post-Behavioral Hardening Transition Gate — PASS)
**Date:** 2026-03-21

---

## 1. Charter Statement

Re-enter the codegen/generated path with a single, bounded objective: **reconcile all 10 existing family specs with their golden snapshots and extend integration coverage from 2 families (RSI, EMA) to all 10**, proving manual→generated equivalence for `consumer_spec` and `pipeline_entry` artifacts across every layer.

This charter does NOT open new artifact types, new template categories, or new domain generation responsibilities. The codegen pipeline already works for 2 families; this wave makes it work for all 10 and proves it.

---

## 2. Strategic Rationale

### Why now

- BEHAVIORAL-WAVE-1 closed with zero medium-or-higher-risk debts (S257 PASS).
- The domain surface has matured: decision (2 families), strategy (2 families), risk (2 families) joined the existing signal (2), evidence (3), and execution (1).
- The codegen module already carries specs for all 10 families but only 4 entries in `integrated.yaml` (RSI consumer_spec, RSI pipeline_entry, EMA consumer_spec, EMA pipeline_entry).
- Closing the integration gap now prevents spec drift while the domain is stable.

### Why not alternatives

| Alternative | Rejection reason |
|---|---|
| Open new artifact types (actor wrappers, evaluators) | Premature — current templates cover the writer pipeline; expanding scope risks capturing human decision territory |
| Skip straight to new families/layers | No foundation — unintegrated specs create false confidence in coverage |
| Defer codegen indefinitely | Drift risk — specs and manual code will diverge as behavioral tests evolve |

---

## 3. Scope Definition

### 3.1 In-scope (Tier 1 — required for charter exit)

| Item | Description |
|---|---|
| Spec reconciliation | Validate all 10 family YAML specs against current domain types and NATS subjects |
| Golden snapshot refresh | Regenerate all 20 golden snapshots; verify they match current templates |
| Integration expansion | Extend `integrated.yaml` from 4 entries to 20 entries (10 families × 2 artifacts) |
| Manual→generated equivalence | Prove that generated output matches manually-written code at every integration point |
| CI gate | `codegen check-all` and `codegen validate-all` pass in CI |
| Marker insertion | Insert `codegen:begin`/`codegen:end` markers in all 10 target files |

### 3.2 In-scope (Tier 2 — permitted if Tier 1 complete)

| Item | Description |
|---|---|
| Template refinement | Minor adjustments to existing `consumer_spec.go.tmpl` and `pipeline_entry.go.tmpl` to handle edge cases discovered during reconciliation |
| Spec validation tightening | Strengthen cross-spec validation rules (e.g., layer→stream mapping consistency) |
| Compare tooling | Improve diff reporting in `compare.go` for clearer CI output |

### 3.3 Out of scope (prohibited)

| Item | Why prohibited |
|---|---|
| New artifact templates | Actor wrappers, evaluators, resolvers, and domain types require human design decisions |
| New family specs | Adding families beyond the existing 10 opens breadth, violating scope freeze |
| Domain logic generation | Severity scaling, confidence mapping, and behavioral rules are human-authored by charter |
| Infrastructure expansion | No new CI jobs, Docker services, or deployment changes |
| Evaluator/resolver codegen | Business logic generation crosses the human-decision boundary |
| Supervisor registry generation | Registry wiring involves architectural choices that remain manual |
| Template parameterization of AckWait/MaxDeliver | Hardcoded values are deliberate; parameterizing is a config concern (OD-BW2) |

---

## 4. Domain Interaction Model

### Current state

```
codegen/families/*.yaml  ──→  codegen generate  ──→  stdout
                                                       │
codegen/golden-snapshots/  ←── codegen compare  ←──────┘
                                                       │
integrated.yaml (4 entries) ── tracks integration ─────┘
```

- 10 specs defined, 20 golden snapshots exist, but only 4 integrations tracked.
- 8 families have golden snapshots but no markers in target files.

### Target state

```
codegen/families/*.yaml  ──→  codegen generate  ──→  target files
                                    │                     │
codegen/golden-snapshots/  ←── compare ──────────────────┘
                                    │
integrated.yaml (20 entries) ── full tracking
                                    │
CI: codegen check-all  ←───────────┘
```

- All 10 families × 2 artifacts = 20 integrations tracked.
- Every integration point has `codegen:begin`/`codegen:end` markers.
- CI enforces equivalence on every commit.

### Preserved invariants

1. **Behavioral tests remain green** — 47 behavioral tests are a hard gate; no codegen change may break them.
2. **Golden snapshots are source of truth** — generated code must match snapshots exactly.
3. **Manual code outside markers is untouched** — codegen only writes within its markers.
4. **Cross-spec uniqueness** — no duplicate family names, NATS subjects, or durable consumers.
5. **raccoon-cli guardian** — architectural boundaries enforced independently of codegen.

---

## 5. Minimum Viable Scenarios

| # | Scenario | Proves |
|---|---|---|
| MV-1 | `codegen validate-all` passes with all 10 specs | Spec structural integrity |
| MV-2 | `codegen check-all` passes (all 20 golden comparisons) | Template→snapshot equivalence |
| MV-3 | Each of the 8 newly integrated families has markers in target files | Integration coverage |
| MV-4 | `integrated.yaml` contains 20 entries with correct target/marker metadata | Manifest completeness |
| MV-5 | CI runs `codegen check-all` and fails on intentional snapshot drift | CI enforcement |
| MV-6 | Manual code at integration points matches generated output byte-for-byte (post-normalization) | Manual→generated equivalence |

---

## 6. Success Criteria

| ID | Criterion | Verification |
|---|---|---|
| E1 | All 10 specs validate individually and cross-spec | `codegen validate-all` exit 0 |
| E2 | All 20 golden snapshots match generated output | `codegen check-all` exit 0 |
| E3 | `integrated.yaml` has 20 entries | YAML parse + count assertion |
| E4 | All target files contain correct markers | Grep for `codegen:begin`/`codegen:end` |
| E5 | Behavioral test suite (47 tests) passes | `go test ./...` in CI |
| E6 | No new artifact templates created | File count in `codegen/templates/` = 2 |
| E7 | No new family specs created | File count in `codegen/families/` = 10 |
| E8 | CI job enforces codegen equivalence | CI log shows `codegen check-all` step |

---

## 7. Non-Success Criteria

These outcomes do NOT count as charter completion:

- Generating actor wrappers or evaluator stubs (out of scope).
- Having specs validate but not being integrated into target files.
- Passing codegen tests while behavioral tests regress.
- Adding new families "because the template supports them."
- Refactoring templates for hypothetical future artifacts.

---

## 8. Hardening Budget

Maximum 15% of effort may go to Tier 2 items (template refinement, validation tightening, compare tooling). Tier 2 work is subordinate to Tier 1 completion and may not open new scope.

---

## 9. Planned Stage Sequence

| Stage | Deliverable | Depends on |
|---|---|---|
| S258 | This charter — scope freeze and governance | S257 PASS |
| S259 | Spec/template reconciliation — validate and refresh all 10 specs and 20 snapshots | S258 |
| S260 | Integration expansion — insert markers, extend `integrated.yaml` to 20 entries | S259 |
| S261 | CI enforcement and equivalence proof — codegen CI gate, manual→generated verification | S260 |
| S262 | Post-codegen re-entry gate — formal exit assessment | S261 |

### Sequencing rationale

Reconciliation (S259) must precede integration (S260) because stale specs or snapshots would produce incorrect markers. CI enforcement (S261) must follow integration (S260) because there is nothing to enforce until markers exist. The gate (S262) aggregates all evidence.

---

## 10. Governance Framework

### Amendment rules

- Scope changes require explicit justification and must be recorded in the Amendments Log below.
- Adding new artifact templates escalates to a new charter (not an amendment).
- Adding new family specs escalates to a new charter.

### Amendment threshold

- Tier 2 items may be activated without amendment if Tier 1 is on track.
- Any change that touches domain logic, actor wiring, or supervisor registration requires a new charter.

### Stop conditions

See `codegen-reentry-entry-exit-and-stop-conditions.md` for formal stop conditions.

### Behavioral test gate

The 47 behavioral tests from BEHAVIORAL-WAVE-1 are a **hard gate** throughout this wave. Any codegen change that causes a behavioral test regression is an immediate stop condition.

---

## 11. Amendments Log

| Date | Amendment | Justification |
|---|---|---|
| — | (none) | — |
