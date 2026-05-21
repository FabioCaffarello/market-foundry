# Documentation Taxonomy And Authoring Conventions

## Purpose

This document defines the active documentation taxonomy for `market-foundry` and
the conventions for creating or updating new documents.

The goal is to keep documentation easy to navigate, hard to duplicate, and aligned
with the real repository structure.

## Status

This document remains valid as the C5 baseline for documentation placement and
authoring. For the current canonical governance surface, use:

- [`documentation-system-hardening.md`](documentation-system-hardening.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)

## Active Taxonomy

| Surface | Role | What belongs there | What does not belong there |
|---|---|---|---|
| Root docs | Repository entrypoints | Overview, development workflow, AI operating contract | Deep architecture, stage evidence, tool rule catalogs |
| `docs/operations/` | Operational support and doc-system guidance | Make targets, scripts, user-facing CLI usage, documentation navigation, authoring conventions | Binding architecture rules, analyzer internals, immutable stage records |
| `docs/tooling/` | Tool-internal reference | Guardrails, drift rules, topology audits, analyzer capability docs | Daily workflow docs, operator runbooks |
| `docs/architecture/` | Canonical architecture and governance | Patterns, runtime rules, system principles, domain design, canonical runbooks that are also architecture | One-off delivery evidence, temporary implementation notes that belong only to a stage |
| `docs/stages/` | Historical stage evidence | Stage reports and stage index | Current workflow guidance, canonical architecture, operator navigation |
| `docs/archive/` | Non-canonical history | Archived or superseded docs kept for traceability | Current source of truth |

## Placement Rules

### Put a document in `docs/operations/` when it answers:

- how to run or validate repository support workflows
- how to use the public command surface
- how to navigate the documentation system
- how new docs should be authored or placed

### Put a document in `docs/tooling/` when it answers:

- what `raccoon-cli` enforces
- what a guardrail or drift rule checks
- how a tooling analyzer interprets the repository

### Put a document in `docs/architecture/` when it answers:

- how the system is designed
- what conventions are binding
- what runtime or domain invariants must hold
- what canonical runbook remains part of architecture governance

### Put a document in `docs/stages/` when it answers:

- what a bounded stage changed
- what was delivered, deferred, or proven during that stage

### Put a document in `docs/archive/` when:

- it has been superseded or consolidated
- it is needed only for historical traceability

## Canonical-Source Rules

- One topic should have one canonical home.
- Link across surfaces instead of copying the same guidance.
- If a workflow doc depends on an architecture invariant, link the architecture doc.
- If a stage creates a lasting convention, move the convention into
  `docs/architecture/` or `docs/operations/` and keep the rationale in the report.
- `docs/archive/` and `docs/stages/` are never the first reference for current work.

## Naming Conventions

### General

- Use lowercase kebab-case filenames.
- Prefer descriptive names over short aliases.
- Name for the question the doc answers, not the stage that created it.

Examples:

- `documentation-taxonomy-and-authoring-conventions.md`
- `makefile-targets-reference-and-conventions.md`
- `operational-contracts-and-cross-runtime-conventions.md`

### Stage reports

- Format: `stage-{id}-{slug}-report.md`
- Keep stage IDs stable and explicit.
- Do not rename historical stage files unless there is a repository-wide migration.

### Tooling docs

- Use `cli-` prefixes for `raccoon-cli` reference docs.
- Keep domain terms explicit: `signal`, `decision`, `strategy`, `risk`, `execution`.
- Use `execute` only for execute-binary-specific material, not for the broader
  execution domain.

### Operations docs

- Prefer names that describe the user-facing task or convention:
  - `scripts-catalog-and-usage-guide.md`
  - `documentation-reorganization-and-operational-navigation.md`

## Authoring Conventions

### Required structure

For new operational, tooling, or architecture docs:

1. Start with `## Purpose`.
2. State the canonical scope and non-goals when ambiguity is likely.
3. Link to related canonical docs instead of repeating large blocks of context.
4. Prefer tables for maps, taxonomies, and command inventories.
5. Keep historical rationale in stage reports unless it is needed as active policy.

### Duplication control

- Update the existing canonical doc when the topic already has one.
- Create a new doc only when the repository gains a genuinely new concern.
- If two docs overlap, designate one as canonical and link from the other.

### Documentation updates that must travel together

When the command surface changes, review:

- `README.md`
- `DEVELOPMENT.md`
- `Makefile` `docs` target
- `docs/operations/README.md`
- `docs/operations/repository-support-surface-canonical-model.md`
- `docs/operations/repository-architecture-convergence.md`
- `docs/tooling/README.md`

When documentation structure changes, review:

- `docs/README.md`
- `docs/operations/documentation-reorganization-and-operational-navigation.md`
- `docs/architecture/monorepo-documentation-and-stage-governance.md`
- `docs/stages/INDEX.md`

## Decision Checklist For New Docs

Before creating a new document, answer in order:

1. Is this current workflow guidance, tooling guidance, architecture, stage evidence,
   or archive material?
2. Does a canonical doc for this topic already exist?
3. Can I update an index or add cross-links instead of creating another full doc?
4. If I create a new doc, which index pages must link to it?

## Maintenance Checklist

- Keep root docs concise.
- Keep area README files current when new docs are added.
- Prefer adding navigation over moving files when moves would mostly create churn.
- Do not let `docs/architecture/` become the default home for every operator-facing
  document.
- Do not let `docs/operations/` duplicate binding architecture rules.
