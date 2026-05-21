# Stage S199 — Generated Path Integration Plan Report

> **Status:** COMPLETE
> **Type:** Architecture & Governance
> **Predecessor:** S198 (Pre-Generated Family Gate — Conditional Pass)
> **Successor:** S200 (First Codegen-First Family Implementation)

---

## Executive Summary

S199 defines the formal integration plan for the codegen-generated path into the market-foundry monorepo. The generated path is now a governed, auditable process with explicit entry points, ownership rules, validation requirements, drift policies, and scope limits. The monorepo is ready for S200 — the first codegen-first family — under the constraints established in S197/S198 and reinforced here.

---

## Deliverables

| # | Document | Path | Purpose |
|---|----------|------|---------|
| D1 | Generated Path Integration Plan | `docs/architecture/generated-path-integration-plan.md` | Master plan: process flow, governance rules, scope limits, risk mitigation |
| D2 | Spec, Templates, Generated Outputs, and Runtime Boundaries | `docs/architecture/spec-templates-generated-outputs-and-runtime-boundaries.md` | Ownership model, data flow, runtime separation between all artifact categories |
| D3 | CI, Drift Detection, and Regeneration Policy | `docs/architecture/generated-path-ci-drift-and-regeneration-policy.md` | CI validation chain, drift taxonomy, regeneration procedures and rules |
| D4 | Stage Report | `docs/stages/stage-s199-generated-path-integration-plan-report.md` | This document |

---

## Key Decisions

### 1. Generated Path as Governed Process

The generated path is defined as an 8-step process:

1. Author YAML spec
2. Validate spec (local + CI)
3. Generate A1+A2 fragments
4. Create golden snapshots
5. Verify golden match (`check-all`)
6. Insert fragments into target files (manual)
7. Author manual artifacts (A3–A6, domain, migration)
8. CI validation (codegen-golden + codegen-test + unit-tests + smoke)

Every step has explicit inputs, outputs, and validation criteria.

### 2. Three-Tier Ownership Model

| Tier | Owner | Examples |
|------|-------|---------|
| Human-only | Developer/Architect | Domain types, migrations, stream defs, templates, specs |
| Machine-only | Codegen engine | Generated fragments (A1+A2), golden snapshots |
| Mixed | Human file + machine fragments | Target files (registry, pipeline.go) |

Rule: machine-owned artifacts are **never manually edited**. If wrong, fix the source (spec/template/derivation), not the output.

### 3. CI Verifies, Never Generates

CI runs the codegen engine to verify committed output matches expectations. It does not produce artifacts for deployment. This ensures deterministic builds and reviewable PRs.

### 4. Drift Detection with Known Gap

| Drift Type | Detection | Status |
|------------|-----------|--------|
| Spec ↔ Golden | Automated (`codegen-golden`) | Active |
| Template ↔ Golden | Automated (`codegen-golden`) | Active |
| Golden ↔ Target file | Manual (PR review) | Gap — mitigated by compilation + tests |

The golden-to-target gap is accepted for this phase. It will be closed when marker-based file integration is implemented.

### 5. Regeneration is Rule-Bound

- Spec change → regenerate affected family only
- Template/derivation change → regenerate ALL families
- Golden + target fragments updated in same PR (no split)
- Manual families never regenerated (they are golden references)
- Regeneration must be idempotent

### 6. Manual Families Remain Manual

The 6 existing families (candle, rsi, rsi_oversold, mean_reversion_entry, position_exposure, paper_order) are NOT retroactively converted. They serve as the golden reference baseline. Changes to them follow the manual path exclusively.

---

## Ownership Clarity Matrix

| Artifact | Owner | Location | Edit Authority |
|----------|-------|----------|---------------|
| YAML spec | Human | `codegen/families/*.yaml` | PR review |
| Template | Human (frozen) | `codegen/templates/*.go.tmpl` | Requires new stage |
| Golden snapshot | Machine | `codegen/golden-snapshots/` | Regeneration only |
| Consumer spec (A1) | Machine (in human file) | `internal/adapters/nats/` | Never manual |
| Pipeline entry (A2) | Machine (in human file) | `cmd/writer/pipeline.go` | Never manual |
| Mapper (A3) | Human | `cmd/writer/mappers.go` | Standard review |
| Domain type | Human | `internal/domain/` | Standard review |
| Migration DDL | Human | `deploy/migrations/` | Standard review |
| Reader/Handler/Route | Human | Various | Standard review |

---

## Risks and Anti-Patterns

### Risks Identified

| Risk | Severity | Mitigation |
|------|----------|-----------|
| Generated fragment silently wrong | HIGH | Golden comparison + compilation + smoke test |
| Boundary blur (manual edits to generated output) | HIGH | Revocation trigger policy + PR review checklist |
| Scope creep beyond A1+A2 | MEDIUM | S198 conditions + single-family iteration |
| Golden-to-target drift undetected | MEDIUM | Compilation catches wiring; review catches content |
| Template modification pressure | MEDIUM | Frozen policy + stage requirement |
| Over-reliance on codegen correctness | LOW | 4 of 6 Tier 1 artifacts remain manual |

### Anti-Patterns Explicitly Rejected

1. **"Generate everything"** — Codegen covers 2 of 10+ artifacts. Expansion requires evidence and authorization.
2. **"Edit the output"** — Generated fragments are immutable. Fixes go to source.
3. **"Skip the golden"** — Every generated family must have golden snapshots before merge.
4. **"Batch generation"** — One family per authorization cycle.
5. **"Retroactive conversion"** — Manual families stay manual.
6. **"Templates are just code"** — Template changes require dedicated stages with full regression.
7. **"CI can generate for us"** — CI verifies; humans generate and commit.

---

## Scope Limits (This Phase)

| Dimension | Limit |
|-----------|-------|
| Generated artifacts | A1 + A2 only |
| Families per iteration | 1 |
| Tiers | Tier 1 only |
| Layers | Existing 6 only |
| File integration | Manual copy |
| Template evolution | Frozen |
| Spec schema | Frozen (14 fields) |
| Mapper generation (A3) | Not authorized |
| Read-path generation (Tier 2) | Not authorized |

---

## Preparation for S200

S200 can proceed when a family is selected that satisfies all S198 conditions:

1. Targets an existing layer (evidence, signal, decision, strategy, risk, execution)
2. Uses a named mapper (not `mapper: "generate"`)
3. Does not require non-existent infrastructure
4. Can be implemented as A1+A2 generated + A3–A6 manual

The S199 deliverables provide:
- The exact process flow for S200 (D1, Section 5)
- Clear ownership rules to prevent ambiguity (D2)
- CI validation chain and failure response procedures (D3)
- Scope limits to prevent overreach (D1, Section 6)

**S200 is unblocked.** The generated path is formally defined, governed, and auditable.

---

## Success Criteria Evaluation

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| SC1 | Generated path formally defined | PASS | D1: 8-step process with governance rules |
| SC2 | Ownership of spec/templates/outputs unambiguous | PASS | D2: three-tier model with edit authority matrix |
| SC3 | CI integration and drift policy explicit | PASS | D3: 4-job chain, drift taxonomy, regeneration rules |
| SC4 | Manual vs generated coexistence safe | PASS | D1 Section 7: manual families untouched; D2 Section 4: ownership boundaries |
| SC5 | Foundation ready for S200 | PASS | Process flow defined; scope limits clear; anti-patterns documented |

---

## Gains, Tradeoffs, and Open Debts

### Gains

- **G1:** Generated path is no longer implicit — it has a formal process, rules, and limits
- **G2:** Ownership is unambiguous at every level (spec, template, golden, fragment, target file)
- **G3:** CI policy is explicit: verify-not-generate, with known gap documented and mitigated
- **G4:** Regeneration rules prevent partial updates and split PRs
- **G5:** Anti-patterns are named and rejected before they can emerge in practice

### Tradeoffs

- **T1:** Golden-to-target drift gap accepted (manual review mitigates; markers will close it)
- **T2:** Manual fragment insertion adds ~5 min per family (acceptable at current scale)
- **T3:** Single-family iteration is slow but safe — correctness over speed
- **T4:** Template freeze limits evolution but prevents regression cascades

### Open Debts

| Debt | Priority | Trigger |
|------|----------|---------|
| Marker-based file integration | HIGH | When manual insertion exceeds ~10 min/family |
| Golden-to-target CI validation | HIGH | When markers are implemented |
| Mapper generation (A3) | HIGH | After first codegen-first family validates |
| Cross-spec uniqueness in CI | MEDIUM | When spec count > 10 |
| Automated `make codegen-regen` | LOW | When regeneration becomes frequent |
| Tier 2 authorization | NOT SCHEDULED | After ≥2 codegen-first families validated |

---

## Next Wave Recommendations

| Priority | Recommendation |
|----------|---------------|
| **IMMEDIATE** | Proceed to S200: select and implement first codegen-first family under S198+S199 constraints |
| **AFTER S200** | Gate review: evaluate S197 success criteria against actual results |
| **AFTER S200** | If all criteria met: authorize second codegen-first family |
| **DEFERRED** | Mapper generation (A3): design `domain.columns` spec extension |
| **DEFERRED** | Marker sections: design and implement append-only integration protocol |
| **NOT SCHEDULED** | Tier 2 read-path generation |
