# Stage S192 — Codegen Tranche Scoping Report

## Stage Identity

| Field | Value |
|-------|-------|
| Stage | S192 |
| Title | Codegen Tranche Scoping |
| Objective | Define the codegen tranche that enables the next wave of analytical family expansion |
| Predecessor | S191 (Family 06 Trigger Assessment — ABORTED; redirected to codegen path) |
| Status | **COMPLETE** |

## Executive Summary

Stage S192 transforms the structural ceiling identified in S191 — no Family 06 candidate satisfies the "no write-path changes" gate condition — into a formal architectural plan for automated analytical family generation.

The tranche defines:
- A **single source of truth** (declarative YAML family specs) that drives all generation.
- **Two tiers** of generation: Tier 1 (within-layer write-path, 6 artifacts) for immediate use, and Tier 2 (full new-layer expansion, 17 artifacts) for future authorization.
- **Golden test equivalence** against 6 hand-crafted families as the primary validation mechanism.
- **CI integration as verification, not generation** — generated files are committed, reviewed, and drift-checked.
- **Explicit ownership boundaries** separating human decisions (domain types, schema design, API surface) from mechanical generation (consumer specs, pipeline entries, mappers).
- **Seven anti-patterns** that the codegen investment must actively avoid.

The first generated family (EMA Crossover, Tier 1) requires only 3 write-path artifacts: consumer spec, pipeline entry, and config entry — because the mapper (`mapSignalRow`) and all read-path artifacts are already generic.

## Rationale: From Manual Expansion to Codegen

### The Structural Finding

S191 proved that:
1. All 6 vertical layers (L1–L6) have analytical read-path coverage.
2. All readers are type-parameterized — `type=ema_crossover` works at the HTTP level today, returning empty results.
3. Every remaining candidate requires write-path changes (pipeline entries, consumer specs).
4. The S190 gate condition (C1: no write-path changes) correctly blocked all candidates.

The manual expansion pattern completed its mission: it proved a reproducible, zero-creative-decision template across 6 families. It now has no remaining pure read-path work. The structural bottleneck is the write-path, and the correct response is not to relax gate conditions but to automate the mechanical work that the pattern has already proven to be template-ready.

### Why Not Just Add Pipeline Entries Manually

Adding EMA Crossover's pipeline entry manually (~30 minutes) would solve one family but would:
- Set a precedent that gate conditions can be bypassed for "small" changes.
- Not address the systematic need (Families 07, 08, 09... each need the same work).
- Not capture the write-path pattern in reusable templates.
- Miss the opportunity to establish validation, drift detection, and CI integration.

Codegen is an investment in the expansion model, not just in a single family.

## Key Decisions

### D1: Source of Truth

**Decision**: One YAML spec file per family in `codegen/families/`. The spec declares event type, NATS identity, writer table, mapper reference, and column definitions (when mapper generation is needed).

**Rationale**: YAML is diffable, reviewable, tooling-friendly, and creates no Go runtime dependency. One file per family prevents cross-family coupling.

### D2: Two-Tier Generation Model

**Decision**: Tier 1 (within-layer, write-path only) is implemented first. Tier 2 (new-layer, full read+write path) is deferred until Tier 1 is validated in production.

**Rationale**: Tier 1 covers the immediate need (EMA Crossover and similar within-layer variants) with 6 artifacts. Tier 2 requires 17 artifacts and 3x template complexity — building both simultaneously would delay the first concrete proof point.

### D3: Template Expansion, Not Runtime Framework

**Decision**: Generated code is standalone Go. No runtime dependency on the codegen tool. No `GenericFamily[T]`, no reflection, no plugin system.

**Rationale**: The current pattern's strength is family independence — a bug in one family's mapper doesn't affect others. A shared runtime would introduce systemic risk.

### D4: Golden Test Equivalence as Primary Validation

**Decision**: Codegen templates are validated by generating output from specs that describe the 6 existing hand-crafted families and comparing against the actual hand-crafted code.

**Rationale**: The 6 families are proven, tested, and deployed. If generated output matches them structurally, the templates are correct. This is more rigorous than testing templates in isolation.

### D5: CI Verifies, Does Not Generate

**Decision**: Generated files are committed to the repository. CI checks that committed files match what the codegen tool would produce. CI does not run codegen as a build step.

**Rationale**: Committed files are reviewable in PRs, debuggable without tooling, and don't create a build-time dependency.

### D6: Existing Families Remain Hand-Crafted

**Decision**: The 6 existing families are not retroactively regenerated. They serve as golden references.

**Rationale**: Regenerating working code introduces regression risk with zero functional benefit.

## Artifact Coverage Summary

### Tier 1 (S193 Scope)

| Artifact | Generated? | Notes |
|----------|:----------:|-------|
| Writer consumer spec | ✅ | Function in NATS registry |
| Writer pipeline entry | ✅ | Entry in pipeline.go |
| Writer mapper | Conditional | Only if event struct differs from existing mapper |
| Writer mapper tests | Conditional | Only if mapper is generated |
| Writer config entry | ✅ | Family name in config array |
| Smoke test phase | ✅ | Endpoint + field assertions |

### Never Generated

| Artifact | Rationale |
|----------|-----------|
| Domain event types | Architectural decisions |
| NATS stream definitions | Infrastructure decisions |
| Writer core (consumer, inserter, supervisor) | Shared infrastructure |
| ClickHouse client, health framework | Shared infrastructure |
| Shared helpers (parseFloat, marshalJSON) | Shared utilities |

## Validation Strategy Summary

| Mechanism | Purpose | When |
|-----------|---------|------|
| Golden test equivalence | Template correctness | Template/spec changes |
| Compilation gate | Type safety | Every PR |
| Unit test gate | Behavioral correctness | Every PR |
| Header comment check | Drift detection | Every PR |
| Regeneration comparison | Authoritative drift detection | Every PR with generated files |
| Integration validation | End-to-end data flow | First generated family (EMA Crossover) |

## Ownership Boundaries Summary

```
HUMAN-OWNED: domain types, stream definitions, infrastructure,
             composition root, shared helpers, CI config,
             template design, schema design, API surface decisions

CODEGEN-OWNED: consumer specs, pipeline entries, mappers (conditional),
               mapper tests, config entries, smoke phases

SPEC-OWNED: codegen/families/*.yaml, codegen/templates/*
```

## Anti-Patterns Documented

1. **The Abstraction Trap** — no shared codegen runtime types
2. **The Config Explosion** — spec files stay minimal
3. **The Template Monolith** — one template per artifact type
4. **The Silent Divergence** — golden test equivalence is mandatory
5. **The Over-Generator** — only generate zero-creative-decision artifacts
6. **The Magic Marker Section** — minimize marker sections in shared files
7. **The Premature Tier 2** — Tier 1 first, Tier 2 after validation

## Deliverables Produced

| # | Document | Path |
|---|----------|------|
| 1 | Codegen tranche scoping (principal) | `docs/architecture/codegen-tranche-scoping.md` |
| 2 | Source of truth, artifact coverage, ownership | `docs/architecture/codegen-source-of-truth-artifact-coverage-and-ownership.md` |
| 3 | Validation, drift, and CI strategy | `docs/architecture/codegen-validation-drift-and-ci-strategy.md` |
| 4 | Anti-patterns, non-goals, human decision boundaries | `docs/architecture/codegen-anti-patterns-non-goals-and-human-decision-boundaries.md` |
| 5 | Stage report | `docs/stages/stage-s192-codegen-tranche-scoping-report.md` |

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Manual expansion abort is absorbed as architectural decision | ✅ S191 abort is the foundation for codegen rationale |
| Codegen tranche is clearly defined | ✅ Two tiers, explicit artifact lists, ownership boundaries |
| Source of truth is explicit | ✅ YAML family specs, one per family, no indirection |
| Artifact coverage is explicit | ✅ 18 artifacts enumerated; 6 in Tier 1 scope, with generation/manual classification |
| Ownership boundaries are explicit | ✅ Three-tier model (human/codegen/spec), 10 never-generated artifacts |
| Validation strategy against 6 families is concrete | ✅ Golden test equivalence, 6 golden spec files, structural comparison |
| CI entry is conceptually clear | ✅ Verification model (not generation); 3 CI steps defined |
| Base is ready for S193 implementation | ✅ Tier 1 templates, golden tests, EMA Crossover as first target |

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| Codegen not implemented yet | ✅ This stage is scoping/decision only |
| Family 06 not resurrected manually | ✅ No manual family expansion attempted |
| Codegen is not a framework | ✅ Explicit anti-pattern; generated code is standalone |
| Not everything is generated "because possible" | ✅ 10 artifact types explicitly excluded; boundary test documented |
| Human architectural decisions not collapsed into templates | ✅ 10 decision types remain exclusively human |
| Not all duplication treated as generation candidate | ✅ Boundary test: ≥3 repetitions with 0 creative decisions required |
| Manual-by-choice items documented | ✅ Domain types, streams, schema, infrastructure explicitly manual |

## Preparation for S193

### S193 Scope (Codegen Implementation)

S193 should:
1. Create `codegen/` directory with `families/`, `golden/`, `templates/` subdirectories.
2. Build Tier 1 templates: consumer spec, pipeline entry, config entry (mandatory); mapper + mapper tests (conditional).
3. Create 6 golden spec files describing the existing hand-crafted families.
4. Implement golden test: regenerate golden specs → compare with hand-crafted code.
5. Create first real spec: `codegen/families/ema_crossover.yaml`.
6. Generate EMA Crossover write-path artifacts from spec.
7. Validate: compilation, unit tests, structural equivalence.

### S193 Does NOT

- Build Tier 2 templates (new-layer read-path generation).
- Modify existing hand-crafted families.
- Integrate codegen into CI (that's Phase 2, post-S194).
- Generate more than one new family (EMA Crossover only).

### S193 Success Criteria

1. Golden test passes for all 6 existing families (structural equivalence).
2. EMA Crossover writer artifacts compile and pass unit tests.
3. Generated code has zero runtime dependency on codegen tool.
4. Every generated file is human-readable and reviewable.
5. Codegen tool is a simple CLI invocation, not a long-running service.

## Recommended Sequence After S192

```
S192: Codegen Tranche Scoping              ← THIS STAGE (COMPLETE)
  │
  ├── S193: Codegen Implementation (Tier 1)
  │   ├── Build Tier 1 templates (write-path)
  │   ├── Golden test: regenerate 6 families, compare
  │   ├── Generate EMA Crossover writer artifacts
  │   └── Validate: compilation + unit tests
  │
  ├── S194: First Generated Family E2E Validation
  │   ├── EMA Crossover deployed in compose stack
  │   ├── Writer persists EMA crossover events to ClickHouse
  │   ├── Existing SignalReader returns type=ema_crossover
  │   ├── Smoke test extended with EMA crossover phase
  │   └── Ceiling metrics measured (template count, spec complexity, generation time)
  │
  └── S195: Codegen Gate Review
      ├── Template coverage assessment
      ├── Golden test results
      ├── Cost comparison: manual (~45 min) vs generated (~2 min)
      ├── Drift detection viability
      └── Decision: authorize Tier 2 scope or iterate Tier 1
```

## Conclusion

Stage S192 fulfills its mission: the codegen tranche is formally scoped, bounded, and ready for implementation. The source of truth is defined (YAML specs), the artifact coverage is explicit (Tier 1: 6 write-path artifacts), the validation strategy is concrete (golden test equivalence against 6 families), the CI model is clear (verification, not generation), and the anti-patterns are documented (7 patterns to avoid).

The manual analytical expansion pattern is formally retired at 6 families. Its legacy — zero creative decisions, zero write-path modifications, and a complete specification evidence base — becomes the foundation for the codegen tranche. The next family (EMA Crossover) will be the first to prove that automated generation maintains the discipline, correctness, and governance that manual expansion established.
