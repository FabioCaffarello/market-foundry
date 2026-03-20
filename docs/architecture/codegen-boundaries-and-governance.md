# Codegen Boundaries and Governance

> **Consolidated document.** Merges content from:
> - `codegen-anti-patterns-non-goals-and-human-decision-boundaries.md`
> - `codegen-manual-vs-generated-boundaries.md`
> - `codegen-next-phase-readiness-or-freeze-conditions.md` (S207)
>
> Originals archived to `docs/archive/codegen/`.

---

## 1. Non-Goals

### What This Codegen Tranche Is NOT

| Non-Goal | Why |
|----------|-----|
| A generic Go code generator | Market Foundry analytical family generator only. Knows NATS, ClickHouse, writer pipeline, analytical reader. Not general-purpose. |
| A runtime framework | Zero runtime dependency on codegen tool. No reflection, no code loading, no plugin system. |
| A replacement for domain modeling | Domain types are human-authored. Codegen consumes them, does not produce them. |
| A configuration management system | Produces Go code and SQL. Config entries are a narrow exception. |
| An abstraction layer | No new abstractions. Generated code uses same patterns as hand-crafted code. |
| A test framework | Standard Go `testing` package. No test DSL or codegen-specific test infrastructure. |
| A migration tool | Can produce DDL files but does not run migrations. |
| A schema evolution manager | Schema changes are architectural decisions. |

### Features Explicitly Excluded

1. **Auto-discovery of event types**: Family creation is explicit (creating a spec file), not auto-scanned.
2. **Dependency resolution**: Generated files use hardcoded, known import paths.
3. **Incremental generation**: All artifacts regenerated from scratch. No patching, merging, or conflict resolution.
4. **Multi-family composition**: One family at a time. Cross-family queries are out of scope.
5. **Self-modifying templates**: Static text with variable substitution and simple iteration only.
6. **Code formatting decisions**: `gofmt`-formatted, no custom style rules.

---

## 2. Anti-Patterns

### AP1: The Abstraction Trap
Introducing `GenericFamily[T]` or `RegisterFamily()` runtime types. **Wrong** because it creates a runtime framework where a bug affects all families simultaneously.
**Rule**: Generated code must be copy-paste equivalent. No shared codegen runtime types.

### AP2: The Config Explosion
Spec files growing to 200+ lines with nested config for every option.
**Rule**: Spec files contain only what varies per family. Everything else is in shared infrastructure.

### AP3: The Template Monolith
Single template generating all artifacts for a family.
**Rule**: One template per artifact type. Templates compose via the spec, not via template inheritance.

### AP4: The Silent Divergence
Templates evolving independently from hand-crafted code, introducing new patterns.
**Rule**: Golden test equivalence is mandatory. Template changes that break equivalence require updating golden families first (manually), then templates follow.

### AP5: The Over-Generator
Generating domain types, stream definitions, compose entries, CI config, documentation.
**Rule**: Only generate artifacts that repeat identically across families with zero creative decisions.

### AP6: The Magic Marker Section
Marker sections proliferating across many files.
**Rule**: Minimize marker sections. Prefer dedicated generated files. When unavoidable, clearly bounded and append-only.

### AP7: The Premature Tier 2
Building Tier 2 templates before Tier 1 is validated end-to-end.
**Rule**: Tier 1 first. Tier 2 authorized only after Tier 1 validated in production with at least one generated family.

---

## 3. The Manual vs Generated Boundary

### Governing Principle

Codegen generates **only** artifacts satisfying all three conditions:
1. **Repetitive** -- implemented 3+ times with identical structure across families.
2. **Mechanical** -- requires zero creative or architectural decisions.
3. **Spec-derivable** -- every value comes directly from the spec file, with no inference.

If any condition is not met, the artifact is human-owned.

### Always Human-Owned

| Artifact | Why Human-Owned |
|----------|-----------------|
| Domain event types | Business semantics, field relationships, type design |
| NATS stream definitions | Infrastructure architecture |
| Writer core logic | Shared infrastructure (consumer.go, inserter.go, supervisor.go) |
| ClickHouse client | Connection management, error handling |
| Health/observability framework | Cross-cutting shared infrastructure |
| Gateway composition root | Dependency graph and startup order |
| HTTP server/router setup | Framework-level with cross-cutting middleware |
| Shared helpers | Used by multiple families; changes affect all consumers |
| CI configuration | Repository-wide concern |
| Migration DDL (Tier 1) | Schema decisions are architectural |
| Template files | Meta-level human ownership |
| Spec files | Encode architectural decisions |

### Decisions That Are Always Human

| Decision | Why It Cannot Be Automated |
|----------|---------------------------|
| Which event types become families | Business/architectural judgment |
| Domain type design | Core domain modeling |
| Schema design for new layers | Query pattern understanding required |
| Stream and subject naming | Infrastructure architecture decision |
| Which filters an endpoint supports | API surface decisions |
| When to create new table vs reuse existing | Storage architecture trade-off |
| Mapper field transformations | Domain knowledge required |
| Retry/backoff policy | Operational judgment |
| Whether a family warrants codegen or manual | Complexity and edge case assessment |
| Template evolution | Architectural change affecting all families |

### Decisions Codegen Can Make

| Decision | Why Mechanical |
|----------|----------------|
| Consumer spec construction | subject + durable + stream -> ConsumerSpec struct |
| Pipeline entry wiring | family + table + SQL + consumer -> pipeline declaration |
| Mapper column ordering | Follows DDL column order deterministically |
| Test case generation | Validation matrix from field types |
| Config array entry | Append family name to array |
| Smoke test phase | Endpoint + required fields from spec |

### The Boundary Test

```
1. Implemented >=3 times with identical structure?  NO -> manual
2. Every value from spec, no inference?             NO -> manual
3. Wrong version causes silent data corruption?     YES -> generate WITH golden test, OR manual
4. Defines API contract visible to consumers?       YES -> generate but require human review
5. All conditions met -> candidate for generation
```

---

## 4. Append-Only Integration Protocol

### Marker Format

```go
// codegen:begin <artifact_type> family=<family_name> source=<spec_path>
// ... generated entries ...
// codegen:end <artifact_type> family=<family_name>
```

### Rules

1. Markers are placed once, manually. Codegen never creates markers.
2. Content between markers is fully codegen-owned. Regeneration replaces all content.
3. Content outside markers is fully human-owned. Codegen never touches it.
4. One marker pair per family × artifact in mixed files. No nesting.
5. Each regeneration produces the complete set of entries, sorted deterministically.

### Integration Points

| File | Section Purpose |
|------|----------------|
| `cmd/writer/pipeline.go` | Pipeline entry declarations |
| `internal/adapters/nats/natssignal/registry.go` | Generated consumer spec functions for currently governed signal families |

Current active codegen scope remains limited to A1+A2 (consumer specs and writer pipeline entries). Generated mappers, config entries, and smoke phases remain deferred and must not be treated as live governance behavior.

---

## 5. Boundary Violations and Response

| Violation | Detection | Response |
|-----------|-----------|----------|
| Manual edit to generated file | CI regeneration comparison | Revert; fix template/spec; regenerate |
| Codegen modifies human code outside markers | Code review; markers | Fix codegen |
| Codegen infers value not in spec | Golden test divergence | Fix template to require explicit spec field |
| Generated file missing header | CI header check | Fix template |
| Spec contains undocumented field | Schema validation | Reject spec |

---

## 6. Next-Phase Readiness (S207)

### What Carries Forward (Active)

| Component | Role |
|-----------|------|
| CI gates (4 jobs) | Continue blocking -- prevent drift during refactoring |
| Golden snapshots (14 files) | Reference artifacts for template validation |
| Integrated slices (4 governed regions) | Protected during refactoring |
| Spec validation | Ensures spec integrity |
| `integrated.yaml` manifest | Tracks governed regions |

### What Is Passive

Templates, family specs, and codegen CLI are unchanged unless refactoring requires adjustment.

### Automatic Freeze Triggers

1. **CI gate failure** that cannot be resolved within refactoring scope
2. **Governance marker corruption** that is non-trivial to restore
3. **Spec schema incompatibility** from domain/NATS/schema refactoring
4. **Template output divergence** from structural changes to consumer specs or pipeline entries

### Discretionary Freeze Triggers

5. Keeping golden snapshots aligned consumes >30 minutes per refactoring PR
6. Architectural pivot changing writer pipeline architecture fundamentally

### Conditions For Expanding Codegen

**Gate 1 -- Integration Proof**: >=3 families integrated with markers, >=2 layers represented, all integrated families pass live event flow.

**Gate 2 -- Tooling Maturity**: Automated file insertion, golden snapshot regeneration, manifest auto-discovery implemented.

**Gate 3 -- Artifact Extension**: `domain.columns` spec extension designed, mapper template (A3) + mapper test template (A4) implemented with golden snapshots.

**Gate 4 -- Governance**: New architecture stage opened, S193 spec schema formally unfrozen, cross-spec validation updated, CI gates extended.

### What the Next Phase Should NOT Do

1. Do not integrate remaining 5 families without specific need.
2. Do not modify templates unless refactoring changes consumer spec/pipeline entry structure.
3. Do not add new artifact types (requires dedicated stage).
4. Do not remove CI gates.
5. Do not treat codegen as a refactoring tool.
6. Do not bulk-regenerate without per-family validation.

---

## 7. Scope Freeze

### Frozen For This Tranche

1. Tier 1 artifact list (6 types)
2. Spec shape (fields defined in specification document)
3. Template granularity (one template per artifact type)
4. Validation approach (golden test equivalence against 6 families)
5. CI integration model (verification, not generation)

### May Evolve

1. Template content (with golden test validation)
2. Tier 2 scope (after Tier 1 proven)
3. Spec shape extensions (new fields for Tier 2; existing fields stable)
4. CI step implementation (tooling may change; principles fixed)

### Requires a New Tranche

1. Cross-family features (aggregations, composite queries)
2. Non-analytical codegen (operational queries, control plane)
3. Multi-service codegen (artifacts across multiple Go modules)
4. External consumer generation (client SDKs, API docs)

---

## Related Documents

- [codegen-specification-and-schema.md](codegen-specification-and-schema.md) -- frozen spec, schema, ownership
- [codegen-validation-and-ci-strategy.md](codegen-validation-and-ci-strategy.md) -- validation and CI details
- [codegen-path-stabilization-or-freeze-decision.md](codegen-path-stabilization-or-freeze-decision.md) -- active decision record
- [codegen-current-usage-boundaries-and-limitations.md](codegen-current-usage-boundaries-and-limitations.md) -- active reference
- [codegen-tranche-scoping.md](codegen-tranche-scoping.md) -- original scoping (S192)
