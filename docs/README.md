# Documentation Map

## Purpose

This file is the documentation entrypoint for daily repository work.

Use it to find the right documentation surface before going deeper into
architecture history or stage evidence.

## Start Here

| Need | Primary entrypoint | Why |
|---|---|---|
| Project overview and current repository shape | [`../README.md`](../README.md) | Fast orientation for contributors |
| Development workflow and validation loop | [`../DEVELOPMENT.md`](../DEVELOPMENT.md) | Canonical daily engineering flow |
| Documentation system map and cross-surface navigation | [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md) | Canonical map for how docs fit together |
| Documentation governance, entrypoints, and taxonomy rules | [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md) | Canonical placement and maintenance rules |
| Stage-documentation governance and narrative model | [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md) | Canonical rules for keeping stage history readable and coherent |
| Stage-history traceability and linking model | [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md) | Practical map from charter to execution, gate, and next-wave decision |
| Repository policy and lightweight enforcement | [`operations/repository-policy-and-lightweight-enforcement-2.md`](operations/repository-policy-and-lightweight-enforcement-2.md) | Current repository-policy enforcement model |
| Unified developer journey | [`operations/developer-workflow-unification.md`](operations/developer-workflow-unification.md) | One official setup/run/validate/smoke/troubleshoot model |
| Onboarding and first-line troubleshooting | [`operations/developer-onboarding-and-troubleshooting-guide.md`](operations/developer-onboarding-and-troubleshooting-guide.md) | Task-oriented runbook for real repository use |
| Operational docs and support workflows | [`operations/README.md`](operations/README.md) | Day-to-day repo operation, command surface, doc governance |
| Tooling and `raccoon-cli` internals | [`tooling/README.md`](tooling/README.md) | Analyzer catalog, drift rules, guardrails |
| Canonical architecture and governance | [`architecture/README.md`](architecture/README.md) | Binding system rules and design records |
| Stage history and delivery evidence | [`stages/INDEX.md`](stages/INDEX.md) | Immutable stage reports |
| Historical and superseded material | [`archive/README.md`](archive/README.md) | Research only, not current source of truth |

## Taxonomy

| Surface | Role | Canonical content |
|---|---|---|
| Root docs | Repository entrypoints | Overview, workflow, AI operating contract |
| `docs/operations/` | Operational support and documentation system | Make targets, scripts, CLI usage, doc navigation, authoring conventions |
| `docs/tooling/` | Tool-internal reference | Guardrails, drift rules, topology audits |
| `docs/architecture/` | Canonical architecture | Patterns, principles, runtime rules, governance artifacts |
| `docs/stages/` | Historical evidence | Stage completion reports only |
| `docs/archive/` | Non-canonical history | Superseded or archived documents |

## Canonical Entrypoints By Document Type

| Document type | Canonical entrypoint |
|---|---|
| Repository overview | [`../README.md`](../README.md) |
| Daily developer workflow | [`../DEVELOPMENT.md`](../DEVELOPMENT.md) |
| Documentation-system map | [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md) |
| Documentation governance and taxonomy | [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md) |
| Stage-documentation governance | [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md) |
| Stage-history traceability model | [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md) |
| Repository policy and lightweight enforcement | [`operations/repository-policy-and-lightweight-enforcement-2.md`](operations/repository-policy-and-lightweight-enforcement-2.md) |
| Operations and support docs | [`operations/README.md`](operations/README.md) |
| Tooling-internal docs | [`tooling/README.md`](tooling/README.md) |
| Canonical architecture | [`architecture/README.md`](architecture/README.md) |
| Stage evidence | [`stages/INDEX.md`](stages/INDEX.md) |
| Archived history | [`archive/README.md`](archive/README.md) |

## Fast Paths

### Daily development

1. Read [`../DEVELOPMENT.md`](../DEVELOPMENT.md).
2. Use [`operations/developer-workflow-unification.md`](operations/developer-workflow-unification.md) to follow the official path.
3. Use [`operations/developer-onboarding-and-troubleshooting-guide.md`](operations/developer-onboarding-and-troubleshooting-guide.md) for onboarding and incident-style troubleshooting.
4. Use [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md) when you need to know which doc surface owns a topic.
5. Use [`tooling/README.md`](tooling/README.md) only when you need direct `raccoon-cli` detail.

### Runtime and operator workflows

1. Start in [`operations/README.md`](operations/README.md).
2. Use [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md) when the boundary between operations and architecture is unclear.
3. Follow links there to the canonical runtime/runbook documents that remain in `docs/architecture/`.

### Architecture review or change design

1. Start in [`architecture/README.md`](architecture/README.md).
2. Use [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md) for the current stage-governance reading model.
3. Use [`stages/INDEX.md`](stages/INDEX.md) when you need the evolution trail behind a decision.

### Historical research

1. Confirm the active canonical doc first.
2. Use [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md) to find the right charter/gate/report chain.
3. Then consult [`archive/README.md`](archive/README.md) or [`stages/INDEX.md`](stages/INDEX.md).

## Source-Of-Truth Rules

- Do not use `docs/stages/` as the canonical source for current workflow or architecture.
- Do not use `docs/archive/` as the canonical source for current behavior.
- Prefer `docs/operations/` for how to work in the repository.
- Prefer [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md) for how the document system fits together.
- Prefer [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md) for placement, naming, and maintenance rules.
- Prefer [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md) for how stage history should stay readable, linked, and governable.
- Prefer [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md) for how to navigate charter, execution, and gate artifacts across waves.
- Prefer [`operations/repository-policy-and-lightweight-enforcement-2.md`](operations/repository-policy-and-lightweight-enforcement-2.md) for what repository-policy invariants are actively enforced.
- Prefer `docs/tooling/` for what the tooling enforces.
- Prefer `docs/architecture/` for how the system is designed and governed.

## Related Documents

- [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md)
- [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md)
- [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md)
- [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md)
- [`operations/repository-policy-and-lightweight-enforcement-2.md`](operations/repository-policy-and-lightweight-enforcement-2.md)
- [`operations/documentation-reorganization-and-operational-navigation.md`](operations/documentation-reorganization-and-operational-navigation.md)
- [`operations/documentation-taxonomy-and-authoring-conventions.md`](operations/documentation-taxonomy-and-authoring-conventions.md)
