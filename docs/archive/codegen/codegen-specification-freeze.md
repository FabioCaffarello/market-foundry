# Codegen Specification Freeze

> Status: **FROZEN** — this specification is the authoritative contract for all codegen behavior in Market Foundry.
> Effective from: S193.
> Changes require a new stage with explicit architectural review.

## Purpose

This document freezes the codegen specification established during S192 into a formal, auditable contract. It eliminates ambiguity about what the codegen does, what it owns, what it requires, and what remains outside its scope. No codegen engine, template, or generated artifact may be implemented without conforming to this specification.

## Frozen Decisions

### D1 — Single Source of Truth

The **sole** source of truth for any generated analytical family is a YAML specification file at:

```
codegen/families/{family_name}.yaml
```

One file per family. No family may be generated without a corresponding spec file. No spec file may exist without producing at least one artifact.

### D2 — Two-Tier Generation Model

| Tier | Scope | Artifacts | Activation |
|------|-------|-----------|------------|
| **Tier 1** | Within-layer write-path | 6 | Immediate (S193+) |
| **Tier 2** | Full new-layer (read + write) | 17 | Deferred until Tier 1 proven in production |

Tier 2 is not authorized until Tier 1 has been validated through at least one generated family (EMA Crossover) passing end-to-end smoke tests and gate review.

### D3 — Template Expansion, Not Runtime Framework

Generated code is standalone Go source. Zero runtime dependency on codegen tooling. Any generated file must compile and function identically if the codegen tool were deleted from the repository.

### D4 — Golden Test Equivalence

Codegen correctness is validated by regenerating specs for the 6 existing hand-crafted families and comparing structural equivalence. This is the primary validation mechanism.

### D5 — CI Verifies, Does Not Generate

Generated files are committed to the repository. CI checks that committed files match codegen output. CI never produces build artifacts via codegen.

### D6 — Existing Families Are Immutable Golden References

The 6 hand-crafted families (candle, rsi, rsi_oversold, mean_reversion, position_exposure, paper_order) are never retroactively regenerated. They serve as golden references for template validation.

## Frozen Spec Schema

The canonical YAML shape is defined in [codegen-spec-schema-fields-invariants-and-ownership.md](codegen-spec-schema-fields-invariants-and-ownership.md). Any field not listed there is **invalid** and must be rejected by spec validation.

## Frozen Artifact Scope

### Tier 1 — Generated Artifacts (6)

| # | Artifact | Condition | Target File |
|---|----------|-----------|-------------|
| 1 | Writer consumer spec | Always | `internal/adapters/nats/{domain}_registry.go` |
| 2 | Writer pipeline entry | Always | `cmd/writer/pipeline.go` |
| 3 | Writer mapper | When `mapper: "generate"` | `cmd/writer/mappers.go` |
| 4 | Writer mapper tests | When mapper generated | `cmd/writer/mappers_test.go` |
| 5 | Writer config entry | Always | `deploy/configs/writer.jsonc` |
| 6 | Smoke test phase | Always | `scripts/smoke-analytical-e2e.sh` |

### Never Generated (Frozen Exclusion List)

These artifact types are **permanently excluded** from codegen scope:

1. Domain event types (`internal/domain/`)
2. NATS stream definitions
3. Writer core logic (`consumer.go`, `inserter.go`, `supervisor.go`)
4. ClickHouse client infrastructure
5. Health/observability framework
6. Gateway `compose.go` core logic
7. HTTP server/router setup
8. Shared helpers (`parseFloat`, `marshalJSON`, `parseAnalyticalParams`)
9. CI configuration (`.github/workflows/`)
10. Template files themselves

## Frozen Ownership Model

```
┌─────────────────────────────────────────────────────┐
│  HUMAN-OWNED (never generated, never overwritten)    │
│  Domain types, stream definitions, infrastructure,   │
│  composition root, shared helpers, CI config,        │
│  template design, schema design, API surface         │
├─────────────────────────────────────────────────────┤
│  CODEGEN-OWNED (generated, validated, replaceable)   │
│  Consumer specs, pipeline entries, mappers (cond.),  │
│  mapper tests, config entries, smoke phases          │
├─────────────────────────────────────────────────────┤
│  SPEC-OWNED (source of truth, human-authored)        │
│  codegen/families/*.yaml, codegen/templates/*        │
└─────────────────────────────────────────────────────┘
```

Ownership rules, invariants, and validation are detailed in [codegen-spec-schema-fields-invariants-and-ownership.md](codegen-spec-schema-fields-invariants-and-ownership.md).

## Frozen Boundaries Between Manual and Generated

The boundary between human-owned and codegen-owned artifacts is defined in [codegen-manual-vs-generated-boundaries.md](codegen-manual-vs-generated-boundaries.md). Key principles:

1. **Codegen never makes architectural decisions.** It produces mechanical, repetitive artifacts from explicit spec declarations.
2. **Human decisions are never inferred.** If a value is not in the spec, codegen cannot derive or guess it.
3. **Generated files carry header markers.** Every generated file includes a comment identifying its spec source, template version, and generation timestamp.
4. **Manual edits to generated files are forbidden.** Fixes go to the template or spec; the file is regenerated.
5. **Append-only integration.** Where codegen must modify human-owned files, it uses clearly marked append-only sections.

## Frozen Validation Strategy

| Mechanism | Scope | Trigger |
|-----------|-------|---------|
| Golden test equivalence | Template correctness | Template or golden spec changes |
| Compilation gate | `go build ./...`, `go vet ./...` | Every PR |
| Unit test gate | Generated test files pass | Every PR |
| Header comment check | Codegen markers present and valid | Every PR |
| Regeneration comparison | Committed files match codegen output | PRs with generated files |
| Spec schema validation | All spec files conform to schema | Every PR |

## What This Freeze Means

1. **No new artifact types** may be added to Tier 1 without a new stage.
2. **No spec fields** may be added, removed, or made optional/required without updating this document and its companions.
3. **No ownership boundary** may shift without explicit architectural review.
4. **No validation mechanism** may be weakened or removed.
5. **Tier 2 activation** requires a separate authorization stage after Tier 1 production validation.

## Freeze Exceptions

The following may evolve without a new freeze:

- **Template content**: implementation details within templates may change as long as golden test equivalence is maintained.
- **CI step implementation**: how CI runs verification steps (scripting, tooling) may change.
- **Spec field documentation**: clarifications to field descriptions that do not change semantics.

Everything else is frozen.

## Related Documents

- [codegen-spec-schema-fields-invariants-and-ownership.md](codegen-spec-schema-fields-invariants-and-ownership.md) — schema, fields, invariants, ownership rules
- [codegen-manual-vs-generated-boundaries.md](codegen-manual-vs-generated-boundaries.md) — manual vs generated boundary policy
- [codegen-tranche-scoping.md](codegen-tranche-scoping.md) — original scoping (S192)
- [codegen-anti-patterns-non-goals-and-human-decision-boundaries.md](codegen-anti-patterns-non-goals-and-human-decision-boundaries.md) — anti-patterns and non-goals
- [codegen-source-of-truth-artifact-coverage-and-ownership.md](codegen-source-of-truth-artifact-coverage-and-ownership.md) — artifact coverage details
- [codegen-validation-drift-and-ci-strategy.md](codegen-validation-drift-and-ci-strategy.md) — validation and CI details
