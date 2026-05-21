# Codegen Re-entry: Permitted vs Prohibited Changes

**Stage:** S258
**Charter:** codegen-reentry-charter-and-scope-freeze.md
**Date:** 2026-03-21

---

## Purpose

This document defines the explicit boundary between changes that are permitted and changes that are prohibited during the codegen re-entry wave. The goal is to prevent scope creep and protect the human-decision boundary.

---

## 1. Permitted Changes

### 1.1 Spec files (`codegen/families/*.yaml`)

| Change | Condition |
|---|---|
| Fix field values to match current domain (e.g., correct NATS subject) | Must reflect actual runtime values |
| Fix typos in family names, durable consumers, table names | Must match manually-written equivalents |
| Add missing required fields caught by validation | Field must be in existing `FamilySpec` schema |
| Adjust `event_type` or `event_package` to match domain events | Must match `internal/domain/*/events.go` |

**Prohibited in spec files:**
- Adding new YAML fields not in the `FamilySpec` struct
- Creating new spec files beyond the existing 10
- Changing layer assignments (evidence→signal, etc.)

### 1.2 Templates (`codegen/templates/*.tmpl`)

| Change | Condition |
|---|---|
| Fix rendering bugs discovered during reconciliation | Bug must be reproducible via `codegen check-all` |
| Adjust whitespace/formatting for consistency | Must not change semantic output |
| Handle edge cases in existing template logic | Edge case must exist in one of the 10 families |

**Prohibited in templates:**
- Creating new template files
- Adding new template functions beyond existing `Derived` fields
- Parameterizing currently-hardcoded values (AckWait, MaxDeliver)
- Adding conditional blocks for artifact types that don't exist yet

### 1.3 Golden snapshots (`codegen/golden-snapshots/`)

| Change | Condition |
|---|---|
| Regenerate snapshots after template fixes | Must be generated, never hand-edited |
| Update snapshots to reflect corrected spec values | Spec correction must be documented |

**Prohibited in snapshots:**
- Hand-editing snapshot files directly
- Creating snapshot directories for families that don't have specs
- Deleting snapshots without deleting the corresponding spec

### 1.4 Integration manifest (`codegen/integrated.yaml`)

| Change | Condition |
|---|---|
| Add entries for newly integrated families | Must have corresponding markers in target files |
| Update `integrated_at` and `stage` fields | Must reflect actual integration date and stage |
| Correct `target` paths if files were moved | Target file must exist |

**Prohibited in manifest:**
- Adding entries for artifact types other than `consumer_spec` and `pipeline_entry`
- Adding entries for families not in `codegen/families/`
- Removing existing entries without justification

### 1.5 Target files (files receiving generated code)

| Change | Condition |
|---|---|
| Insert `codegen:begin`/`codegen:end` marker pairs | Markers must bracket code that matches golden snapshot |
| Replace manual code within markers with generated equivalent | Generated output must be byte-equivalent (post-normalization) to manual code |

**Prohibited in target files:**
- Modifying code outside `codegen:begin`/`codegen:end` markers
- Changing function signatures, types, or imports outside markers
- Inserting markers around code that codegen does not generate (e.g., evaluators, resolvers)

### 1.6 Codegen tooling (`codegen/*.go`)

| Change | Condition |
|---|---|
| Fix bugs in `compare.go` diff reporting | Must improve clarity without changing comparison semantics |
| Add `toPascalCase` abbreviation entries | Only for abbreviations that exist in family names |
| Strengthen validation in `spec.go` | Must not reject currently-valid specs |

**Prohibited in tooling:**
- Adding new subcommands to `main.go`
- Adding new artifact types to `SupportedArtifacts()`
- Changing `DerivedFields` struct in ways that break existing templates
- Adding file-writing capabilities (codegen outputs to stdout; insertion is a separate concern)

### 1.7 CI configuration

| Change | Condition |
|---|---|
| Add `codegen check-all` step to CI pipeline | Must not replace or weaken existing CI steps |
| Add `codegen validate-all` step to CI pipeline | Same condition |

**Prohibited in CI:**
- Removing or weakening behavioral test gates
- Adding Docker/infrastructure requirements for codegen
- Creating separate codegen-only CI workflows

---

## 2. Prohibited Changes (Summary)

| Category | Prohibition | Rationale |
|---|---|---|
| New artifacts | No new template types (actor, evaluator, resolver, domain type) | Crosses human-decision boundary |
| New families | No 11th spec file | Opens breadth; requires separate charter |
| Domain logic | No generated severity scaling, confidence maps, or behavioral rules | Business logic is human-authored |
| Actor wiring | No generated supervisor registration or actor factories | Architectural decision, not mechanical generation |
| Infrastructure | No new Docker services, databases, or deployment scripts | Scope creep into infra |
| Config expansion | No parameterization of hardcoded template values | Blocked by OD-BW2 (config infrastructure debt) |
| Template proliferation | No "just in case" templates for future use | YAGNI; future needs get future charters |
| Cross-boundary integration | No codegen touching `internal/application/` or `internal/domain/` logic | These layers contain human-decided behavior |

---

## 3. Decision Boundary: Generated vs Human

The following table clarifies what belongs to codegen and what remains human-authored:

| Artifact | Owner | Rationale |
|---|---|---|
| NATS consumer specs | **Codegen** | Mechanical mapping from spec fields; no domain judgment |
| Pipeline entry structs | **Codegen** | Mechanical mapping from spec fields; no domain judgment |
| Evaluator/resolver logic | **Human** | Contains behavioral decisions (thresholds, scaling, rejection criteria) |
| Domain types (Decision, Strategy, RiskAssessment) | **Human** | Core domain model; changes require domain understanding |
| Severity/confidence scaling maps | **Human** | Business rules validated by behavioral tests |
| Actor lifecycle and routing | **Human** | Architectural decisions about message flow |
| Supervisor registration | **Human** | Family discovery and wiring is an architectural choice |
| NATS subject naming | **Human decides, codegen records** | Subject names are a contract; codegen captures them, doesn't invent them |
| ClickHouse table schemas | **Human** | Schema design requires data modeling judgment |
| Event types and packages | **Human decides, codegen records** | Event contracts are a domain decision |

---

## 4. Escalation Rules

If during execution a change is needed that falls outside the permitted list:

1. **Document the need** — describe what change is needed and why.
2. **Check the charter** — if the change requires a new artifact type, new family, or domain logic generation, it requires a **new charter**.
3. **If ambiguous** — treat as prohibited until explicitly amended in the charter's Amendments Log.
4. **If blocking** — record as a debt item with risk assessment; do not unblock by expanding scope.
