# Stage Documentation Governance And Narrative Coherence

## Purpose

This document defines the lightweight governance model for keeping stage
documentation readable, traceable, and narratively coherent in
`market-foundry`.

It does not redefine what a stage is. It defines how stage artifacts should fit
together so the history remains usable as the repository grows.

## Why This Exists

The repository already has strong stage discipline, a durable stage index, and a
large body of charter, implementation, proof, and gate material.

The current risk is no longer missing documentation volume. The risk is history
becoming harder to read because:

- the stage trail is large enough that a flat index is not sufficient on its own
- wave-level artifacts often span `docs/stages/`, `docs/architecture/`, and
  `docs/operations/`
- recent waves follow a real charter-to-gate pattern, but that pattern remained
  mostly implicit
- a reader can find a report quickly without understanding whether it opened a
  wave, executed work inside it, or closed it

## Governance Goals

The stage-documentation model should make these questions easy to answer:

1. What strategic authority opened this work?
2. Which stage actually executed the bounded change?
3. Which gate or closure stage decided the outcome?
4. Which promoted docs became the durable source of truth afterward?
5. What should a future contributor read first?

## Canonical Roles

| Artifact role | Canonical home | Responsibility |
|---|---|---|
| Stage history inventory | `docs/stages/INDEX.md` | Historical navigation by phase and wave landmarks |
| Stage readability and maintenance rules | `docs/operations/` | Current guidance for keeping stage history coherent |
| Charter, gate, and structural authority | `docs/architecture/` | Current wave authority and lasting governance decisions |
| Stage report | `docs/stages/` | Immutable evidence for one bounded stage |

## Expected Narrative Chain

When a governed wave exists, the expected narrative chain is:

1. charter or scope-freeze authority
2. bounded execution stages
3. proof, hardening, or reconciliation stages as needed
4. gate or closure decision
5. next-wave recommendation or promoted canonical docs

Not every wave needs every artifact. The point is clarity, not ceremony.

## Minimum Coherence Rules

### 1. Make the role legible

A reader should be able to tell whether a stage is primarily:

- charter or scope freeze
- implementation or hardening
- validation or proof
- gate or closure
- support/governance improvement

This can be signaled through the title, summary, and index placement. No new
frontmatter system is required.

### 2. Link to the durable owner

If a stage establishes a lasting rule, pattern, or workflow, the report should
link to the promoted `docs/operations/` or `docs/architecture/` artifact that
owns that rule after the stage closes.

### 3. Preserve the decision chain

When a stage is part of a wave, at least one of these surfaces should make the
wave chain obvious:

- the stage report itself
- the stage index
- the promoted charter or gate doc

### 4. Keep indexes navigational

`docs/stages/INDEX.md` should remain an index, not a second copy of the
reports. It should expose the important narrative landmarks that help readers
choose where to start.

### 5. Avoid retrospective churn

Do not mass-edit historical reports only to normalize prose. Fix navigation and
entrypoints first. Edit old reports only when a concrete traceability break or
evidence correction exists.

## What Good Looks Like

For a healthy governed wave, a contributor should be able to:

1. open the relevant charter stage from the index
2. find the execution tranche from nearby index entries or report links
3. locate the gate or closure decision without directory-wide searching
4. discover the durable canonical docs promoted by that wave
5. understand whether the next step is implementation, hardening, or a new charter

## Maintenance Triggers

Review this model when:

- a new stage-support wave changes how reports are scaffolded or checked
- a new charter/gate pattern becomes common enough to deserve indexing support
- the stage index becomes materially less readable because of growth
- a gate or closure stage reveals evidence drift caused by weak linking

## Boundaries

This model intentionally does not introduce:

- a stage database
- mandatory metadata headers for every report
- automatic lifecycle state management
- mass retrofits of historical prose
- a second approval system beyond existing governance artifacts

## Related Documents

- [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md)
- [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md)
- [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`../architecture/monorepo-documentation-and-stage-governance.md`](../architecture/monorepo-documentation-and-stage-governance.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
