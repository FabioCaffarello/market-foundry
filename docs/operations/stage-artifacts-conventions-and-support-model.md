# Stage Artifacts Conventions And Support Model

## Purpose

This document defines the practical conventions for stage artifacts in
`market-foundry`: where they live, how they are named, what minimum completeness
they should have, and which support checks protect them.

It is intentionally operational. It does not replace the architectural
Definition of Done.

## Artifact Classes

| Artifact class | Canonical location | Purpose |
|---|---|---|
| Stage report | `docs/stages/` | Immutable completion record for one stage |
| Stage index entry | `docs/stages/INDEX.md` | Historical navigation and traceability |
| Promoted lasting convention | `docs/operations/` or `docs/architecture/` | Current source of truth after the stage |
| Charter / gate / recommendation doc | usually `docs/architecture/` | Current governing artifact for wave boundaries or formal decisions |
| Support-stage report | `docs/stages/` | Same as any other stage report; support stages do not get a parallel evidence model |

## Naming Conventions

### Stage reports

- Format: `stage-{id}-{slug}-report.md`
- Use lowercase kebab-case for `{slug}`
- Keep the stage ID stable once published
- Use the same stage ID in the report title and index entry

Examples:

- `docs/stages/stage-c15-stage-tooling-and-execution-governance-support-report.md`
- `docs/stages/stage-s312-venue-adapter-hardening-report.md`

### Operations and architecture docs promoted by a stage

- Name the durable concern, not the temporary stage
- Prefer nouns that remain useful after the stage closes
- Avoid stage-number-based filenames outside `docs/stages/`

Good:

- `stage-tooling-and-execution-governance-support.md`
- `stage-artifacts-conventions-and-support-model.md`

Avoid:

- `c15-stage-process.md`
- `next-stage-helper-v1.md`

## Minimum Completeness For A Stage Report

Every stage report should make the following easy to find:

1. what the stage was trying to achieve
2. what actually changed
3. what stayed out of scope
4. how the result was validated
5. what the next stage should know

The repository support floor is:

- one level-1 title
- multiple level-2 sections
- explicit scope-boundary signals
- validation section
- index entry in `docs/stages/INDEX.md`

The recommended practical shape is:

```markdown
# Stage C15 Report: Title

## Summary
## Diagnosis Or Objective
## Scope Boundaries
### In scope
### Out of scope
### Not changed
## Changes Applied
## Validation
## Limits And Deferred Follow-Ups
## Preparation For Next Stage
```

This is a support model, not a rigid prose template. Equivalent section names
are acceptable when they keep the same meaning.

## Minimum Completeness For Stage-Support Docs

When a stage promotes lasting support guidance into `docs/operations/`, the doc
should answer at least one of these questions clearly:

- what support surface should contributors use now
- what artifact convention should they follow
- what check or helper should they run
- what remains intentionally manual

If a stage creates a lasting support rule but leaves it only inside the report,
the stage has not fully promoted its result.

## Checkpoints And Waves

The repository does not maintain a heavyweight checkpoint registry.

Instead, checkpoints should be expressed through normal artifacts:

- architecture gate or charter docs for wave-level boundaries
- stage reports for bounded execution evidence
- promoted operations docs when the support workflow changes

This keeps checkpoints visible without introducing a second tracking system.

For the expected narrative relationship between these artifacts, use:

- [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md)
- [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md)

## Minimum Traceability Expectations

When a stage participates in a broader wave, make the following chain easy to
recover without reading the whole directory:

1. the opening charter or freeze artifact
2. the stage report for the bounded work
3. any promoted operations or architecture docs created by the stage
4. the gate or closure artifact that evaluates the wave

This does not require a new registry. It requires deliberate links in the index,
stage report, and promoted docs.

## Supported Tooling

### `make stage-scaffold`

Use when:

- a governed stage needs a report file opened quickly
- the stage should start from the repository's minimum support shape

Do not use it as a justification to create unnecessary stages.

### `make stage-check`

Use when:

- you want a quick active-stage traceability and completeness check
- the stage adds canonical docs or multiple support artifacts
- you want to reduce reliance on memory before closing the stage

Recommended pattern:

```bash
make stage-check \
  STAGE_ID=C15 \
  STAGE_SLUG=stage-tooling-and-execution-governance-support \
  STAGE_REQUIRE=docs/operations/stage-tooling-and-execution-governance-support.md,docs/operations/stage-artifacts-conventions-and-support-model.md
```

## Support Boundaries

The support model intentionally does not enforce:

- exact report prose style
- mandatory artifact tables for every stage
- automatic edits to `docs/stages/INDEX.md`
- stage sequencing or approval workflows
- wave planning semantics

Those concerns either belong to human judgment or to architecture governance.

## Related Documents

- [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md)
- [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md)
- [`../architecture/stage-definition-of-done.md`](../architecture/stage-definition-of-done.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
