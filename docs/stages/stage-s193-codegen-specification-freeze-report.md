# Stage S193 — Codegen Specification Freeze Report

| Field | Value |
|-------|-------|
| **Stage** | S193 |
| **Title** | Codegen Specification Freeze |
| **Status** | COMPLETE |
| **Predecessor** | S192 — Codegen Tranche Scoping |
| **Successor** | S194 — Codegen Equivalence Definition and Golden Test Foundation |
| **Date** | 2026-03-20 |

## Executive Summary

S193 transforms the codegen scoping decisions from S192 into a formally frozen specification. The spec schema, ownership boundaries, validation invariants, and manual-vs-generated limits are now explicit, auditable, and locked. No codegen engine or template may be implemented without conforming to this frozen contract.

This stage deliberately does **not** implement any codegen tooling. Its sole purpose is to eliminate ambiguity about what the codegen will do, what it will own, what it requires, and what it must never touch.

## What Was Done

### 1. Specification Freeze Established

**Deliverable**: [codegen-specification-freeze.md](../architecture/codegen-specification-freeze.md)

Frozen decisions:
- **D1**: Single source of truth — one YAML spec per family at `codegen/families/{family_name}.yaml`
- **D2**: Two-tier generation model — Tier 1 (6 write-path artifacts) immediate; Tier 2 (17 full artifacts) deferred
- **D3**: Template expansion, not runtime framework — generated code is standalone Go
- **D4**: Golden test equivalence — primary validation via structural comparison with 6 hand-crafted families
- **D5**: CI verifies, does not generate — generated files committed; CI checks consistency
- **D6**: Existing families immutable — 6 hand-crafted families serve as golden references, never regenerated

### 2. Spec Schema, Fields, Invariants, and Ownership Defined

**Deliverable**: [codegen-spec-schema-fields-invariants-and-ownership.md](../architecture/codegen-spec-schema-fields-invariants-and-ownership.md)

Key outcomes:
- **Canonical YAML schema** defined with exact field types, constraints, and nesting
- **14 required fields** across 4 sections (`family`, `nats`, `writer`, `domain`)
- **2 conditional sections** (`domain.columns` when mapper is generated; `schema` for Tier 2)
- **12 explicitly prohibited fields** with rationale for each exclusion
- **3 uniqueness invariants** (family name, durable name, pipeline key)
- **5 referential integrity invariants** (table→migration, event_type→Go source, stream→NATS, mapper→Go function, config_array→writer.jsonc)
- **6 structural invariants** (tier/schema consistency, mapper/columns consistency, column/DDL alignment)
- **5 naming pattern invariants** with regex patterns
- **7 ownership rules** governing spec authority, header markers, edit prohibition, append-only integration, template ownership, spec authorship, golden spec immutability
- **6-step validation execution order** (fail-fast: schema → naming → uniqueness → referential → structural → column alignment)
- **Spec evolution rules** requiring a new architectural stage for any schema change

### 3. Manual vs Generated Boundaries Defined

**Deliverable**: [codegen-manual-vs-generated-boundaries.md](../architecture/codegen-manual-vs-generated-boundaries.md)

Key outcomes:
- **3-condition boundary test**: repetitive (≥3 instances) + mechanical (zero decisions) + spec-derivable (no inference)
- **12 always-human-owned artifact types** with rationale for each
- **6 codegen-owned Tier 1 artifacts** with mechanical justification
- **11 codegen-owned Tier 2 artifacts** (not yet authorized)
- **10 always-human decisions** that codegen never automates or infers
- **6 mechanical decisions** that codegen can make
- **5-step boundary test** for evaluating new candidates
- **Append-only integration protocol** with marker format, 5 rules, and 6 integration points
- **5 strategic manual-by-choice artifacts** (could generate but deliberately don't)
- **5 boundary violation types** with detection and response procedures

## What Was NOT Done (By Design)

| Not Done | Reason |
|----------|--------|
| Codegen engine implementation | Out of scope — spec must be frozen before building engine |
| Template creation | Out of scope — templates implement the spec; spec comes first |
| Family generation | Out of scope — no families generated until engine and equivalence validated |
| Golden spec file creation | Deferred to S194 — requires equivalence definition |
| CI pipeline integration | Deferred — CI steps implement the validation strategy; strategy defined here |
| `codegen/` directory structure | Deferred to implementation stage — spec defines what goes there, not the tooling |

## Validation of Acceptance Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Codegen spec formally frozen | PASS | [codegen-specification-freeze.md](../architecture/codegen-specification-freeze.md) with explicit frozen status |
| Source of truth explicit | PASS | Single YAML file per family at `codegen/families/` — no alternatives |
| Ownership explicit | PASS | Three-tier model (HUMAN/CODEGEN/SPEC) with 7 rules |
| Fields clear | PASS | 14 required, 2 conditional, 12 prohibited — all enumerated |
| Invariants clear | PASS | 3 uniqueness + 5 referential + 6 structural + 5 naming = 19 total invariants |
| Boundaries clear | PASS | Boundary map with rationale for every artifact |
| Manual vs generated unambiguous | PASS | 3-condition test + boundary map + 10 human decisions + 6 mechanical decisions |
| Base ready for S194 equivalence | PASS | Schema, ownership, and invariants provide foundation for golden test design |

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No engine implemented | COMPLIANT — zero codegen code written |
| No families generated | COMPLIANT — no generated artifacts exist |
| No critical behavior left implicit | COMPLIANT — every field, invariant, and boundary documented |
| No excessive framework theory | COMPLIANT — spec is concrete: fields, types, constraints, examples |
| Manual-by-choice documented | COMPLIANT — 5 strategic manual items with rationale |

## Architectural Artifacts Produced

| Document | Purpose |
|----------|---------|
| `docs/architecture/codegen-specification-freeze.md` | Principal freeze document — authoritative contract |
| `docs/architecture/codegen-spec-schema-fields-invariants-and-ownership.md` | Schema, fields, invariants, ownership rules |
| `docs/architecture/codegen-manual-vs-generated-boundaries.md` | Boundary policy between human and generated |
| `docs/stages/stage-s193-codegen-specification-freeze-report.md` | This report |

## Relationship to Prior Documents

| S192 Document | S193 Status |
|---------------|-------------|
| `codegen-tranche-scoping.md` | Scoping decisions frozen into specification |
| `codegen-anti-patterns-non-goals-and-human-decision-boundaries.md` | Anti-patterns and non-goals incorporated into boundary definitions |
| `codegen-source-of-truth-artifact-coverage-and-ownership.md` | Source of truth and ownership formalized with validation invariants |
| `codegen-validation-drift-and-ci-strategy.md` | Validation strategy referenced; CI implementation deferred |

S192 documents remain valid as context and rationale. S193 documents are the authoritative frozen contracts.

## Preparation for S194

S194 should define **codegen equivalence** — the precise rules for determining whether generated output is structurally equivalent to hand-crafted code.

### Recommended S194 Scope

1. **Golden spec creation** — author YAML specs for all 6 existing hand-crafted families
2. **Equivalence definition** — define what "structurally equivalent" means (AST-level? normalized text? semantic?)
3. **Allowed vs unacceptable differences** — formalize the variance table (import ordering OK, missing fields not OK)
4. **Golden test procedure** — step-by-step procedure for running equivalence checks
5. **Equivalence failure protocol** — what happens when equivalence fails (template fix? golden update? spec adjustment?)

### S194 Entry Conditions

All met by S193:
- Spec schema frozen with all fields defined
- Ownership model explicit
- Validation invariants enumerated
- Boundary between manual and generated clear
- Anti-patterns documented

### What S194 Must NOT Do

- Implement the codegen engine (that's S195+)
- Generate any family
- Create templates
- Modify CI pipeline

S194 establishes the **measurement standard**. S195+ builds the tool that must pass that standard.

## Open Items

None. All items within S193 scope are resolved.

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Spec schema proves insufficient during implementation | Low | Schema evolution rules allow controlled extension via new stage |
| Golden test equivalence too strict or too loose | Medium | S194 defines precise equivalence rules; adjustable before engine exists |
| Append-only marker protocol proves fragile | Low | Only 6 integration points; can be replaced with dedicated generated files if needed |
| Tier 2 scope drifts before authorization | Low | Tier 2 explicitly locked behind Tier 1 production validation gate |
