# Documentation Governance Entry Points And Taxonomy

## Purpose

This document is the canonical governance reference for documentation
entrypoints, taxonomy, naming, and maintenance in `market-foundry`.

Use it when deciding where documentation belongs, which document is canonical
for a topic, and which files must be updated together.

## Canonical Entry Points

| Document type | Canonical entrypoint | Scope |
|---|---|---|
| Repository overview | [`../../README.md`](../../README.md) | What the repository is and the high-level support surface |
| Daily engineering workflow | [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md) | Setup, bring-up, validate, smoke, troubleshoot |
| Documentation-system navigation | [`documentation-system-hardening.md`](documentation-system-hardening.md) | How the documentation system fits together |
| Documentation governance and taxonomy | [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md) | Placement, naming, source-of-truth, maintenance |
| Repository platform strategic operating model | [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md) | Unified long-term operating contract for the repository as a development platform |
| Repository platform applied governance model | [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md) | Applied governance, health, review, and sustainability rules |
| Repository sustainability model | [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md) | Long-term documentation/tooling/entrypoint sustainability rules |
| Sustainability review routines | [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md) | Lightweight review loops for entropy control |
| Periodic repository review model | [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md) | Cadence model for recurring review of the development environment |
| Review cadence triggers and follow-through rules | [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md) | Trigger model and proportional-response rules for repository review |
| Stage-documentation governance | [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md) | Rules for readable, coherent stage history |
| Stage-history traceability model | [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md) | Reading and linking model from charter to gate |
| Operations docs | [`README.md`](README.md) | User-facing support docs and command-surface guidance |
| Tooling docs | [`../tooling/README.md`](../tooling/README.md) | Analyzer rules, guardrails, drift catalogs |
| Architecture docs | [`../architecture/README.md`](../architecture/README.md) | Canonical design and binding governance |
| Stage evidence | [`../stages/INDEX.md`](../stages/INDEX.md) | Immutable stage history |
| Archive | [`../archive/README.md`](../archive/README.md) | Superseded and historical material |

## Taxonomy

| Surface | Canonical role | What belongs there | What does not belong there |
|---|---|---|---|
| Root docs | Repository entrypoints | Overview, workflow, AI operating contract | Deep architecture, stage evidence, tool rule catalogs |
| `docs/operations/` | Repository support and documentation-system governance | User-facing workflow docs, command-surface docs, documentation-system maps, maintenance rules | Deep architecture records, analyzer internals, immutable stage evidence |
| `docs/tooling/` | Tool-internal reference | `raccoon-cli` rule catalogs, guardrails, topology and drift docs | Operator workflow, onboarding, troubleshooting journeys |
| `docs/architecture/` | Canonical architecture and structural governance | System principles, domain design, runtime rules, binding conventions, architecture runbooks | One-off delivery evidence, support-surface navigation docs |
| `docs/stages/` | Immutable historical evidence | Stage reports and the stage index | Current workflow, current policy, canonical architecture |
| `docs/archive/` | Non-canonical history | Superseded docs retained for traceability | Current source of truth |

## Source-Of-Truth Rules

- One topic must have one canonical home.
- A cross-surface index may point to a topic; it does not become the canonical
  topic owner by linking to it.
- If a document exists mainly to explain how to navigate the repository or how
  to maintain documentation, it belongs in `docs/operations/`.
- If a document exists mainly to define how the system is designed or what
  runtime or boundary invariants must hold, it belongs in `docs/architecture/`.
- If a stage introduces a lasting convention, move the lasting convention into
  `docs/operations/` or `docs/architecture/` and keep the stage report as
  historical rationale only.
- `docs/stages/` and `docs/archive/` are never the canonical source for current
  behavior.

## Naming Conventions

### General

- Use lowercase kebab-case filenames.
- Name files for the durable question they answer, not for the temporary work
  item that created them.
- Prefer explicit terms such as `entrypoints`, `taxonomy`, `governance`,
  `runbook`, `guardrails`, `boundaries`, or `conventions` over vague labels.

### README and index conventions

- Use `README.md` for area entrypoints that describe the current active surface.
- Use `INDEX.md` for inventory-style historical indexes, such as stage history.
- Do not create multiple README-like entrypoints inside the same directory
  unless they clearly serve different durable purposes.

### Stage reports

- Format: `stage-{id}-{slug}-report.md`
- Keep stage IDs stable.
- Keep reports historical; do not rename them for stylistic cleanup.

### Tooling docs

- Use `cli-` prefixes for `raccoon-cli` references.
- Keep `execution` for the domain and `execute` for execute-binary-specific
  material.

## Link And Duplication Rules

- Prefer linking to a canonical document instead of restating its full guidance.
- An index should summarize ownership and route readers; it should not become a
  second full version of the underlying document.
- If two active docs overlap, either:
  - narrow one so the scope is distinct; or
  - designate one as canonical and turn the other into a short bridge doc.
- Historical reorganization or stage docs may remain, but they must point to the
  current canonical entrypoint when they overlap with active policy.

## Maintenance Triggers

Review these files when the documentation tree or support surface changes:

| Change | Minimum review set |
|---|---|
| Root entrypoint or workflow change | `README.md`, `DEVELOPMENT.md`, `docs/README.md`, `docs/operations/README.md` |
| Documentation taxonomy or placement change | `docs/README.md`, `docs/operations/documentation-system-hardening.md`, `docs/operations/documentation-governance-entrypoints-and-taxonomy.md`, `docs/operations/long-term-documentation-and-operational-sustainability-model.md`, `docs/operations/repository-sustainability-review-routines-and-entropy-control.md`, `docs/operations/periodic-review-model-for-repository-development-environment.md`, `docs/operations/repository-review-cadence-triggers-and-follow-through-rules.md`, `docs/architecture/monorepo-documentation-and-stage-governance.md` |
| Repository-platform operating model or applied governance-model change | `docs/operations/README.md`, `docs/README.md`, `docs/operations/strategic-operating-model-for-the-repository-as-a-development-platform.md`, `docs/operations/repository-platform-governance-health-review-and-sustainability-model.md`, `docs/operations/developer-environment-strategic-health-model.md`, `docs/operations/periodic-review-model-for-repository-development-environment.md`, `docs/operations/support-surface-sunset-consolidation-and-retirement-strategy.md` |
| Repository-review cadence or trigger-model change | `docs/operations/README.md`, `docs/README.md`, `docs/operations/periodic-review-model-for-repository-development-environment.md`, `docs/operations/repository-review-cadence-triggers-and-follow-through-rules.md`, `docs/operations/developer-environment-strategic-health-model.md`, `docs/operations/repository-sustainability-review-routines-and-entropy-control.md` |
| Tooling-surface change | `docs/tooling/README.md`, `docs/operations/raccoon-cli-command-reference.md`, relevant `docs/tooling/cli-*.md` |
| Architecture-governance change | `docs/architecture/README.md`, relevant canonical architecture doc, `docs/operations/README.md` if operator navigation is affected |
| New stage report | `docs/stages/INDEX.md`, stage file, any canonical doc promoted from the stage |
| Stage-history governance change | `docs/stages/INDEX.md`, `docs/operations/stage-documentation-governance-and-narrative-coherence.md`, `docs/operations/stage-history-traceability-and-linking-model.md`, `docs/architecture/monorepo-documentation-and-stage-governance.md` |

## Simple Evolution Rules

1. Start by updating an existing canonical doc before creating a new one.
2. Add a new doc only when the repository gains a new durable concern or a new
   stable entrypoint.
3. Prefer improving navigation over moving files.
4. Keep root docs short; push detail into the owning canonical doc.
5. If a historical doc still matters, keep it, but mark the current canonical
   successor clearly.
6. When in doubt between operations and architecture:
   - choose `docs/operations/` for repository usage and documentation-system rules;
   - choose `docs/architecture/` for system-shape and binding runtime rules.

## Decision Checklist Before Creating A New Document

1. Is the topic current workflow, documentation governance, tooling internals,
   architecture, stage evidence, or archive history?
2. Which canonical entrypoint already covers this area?
3. Can an index, a section, or a link solve the problem without a new file?
4. If a new file is needed, which README or INDEX must link to it?
5. Does the file name describe the durable concern rather than the stage name?

## Related Documents

- [`documentation-system-hardening.md`](documentation-system-hardening.md)
- [`README.md`](README.md)
- [`../README.md`](../README.md)
- [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md)
- [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md)
- [`../tooling/README.md`](../tooling/README.md)
- [`../architecture/README.md`](../architecture/README.md)
- [`../architecture/monorepo-documentation-and-stage-governance.md`](../architecture/monorepo-documentation-and-stage-governance.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
