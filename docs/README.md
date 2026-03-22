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
| Ownership by subject and owner/reference split | [`operations/documentary-ownership-and-canonical-navigation.md`](operations/documentary-ownership-and-canonical-navigation.md) | Canonical map of owners, references, and historical surfaces |
| Repository-shape navigation and metadata model | [`operations/repository-metadata-indexes-and-developer-navigation-system.md`](operations/repository-metadata-indexes-and-developer-navigation-system.md) | Explains the lightweight navigation layer across the real tree |
| Practical repository area maps and maintenance rules | [`operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`](operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md) | Maps contributor tasks to the right top-level area |
| Documentation placement and maintenance rules | [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md) | Canonical placement, naming, and maintenance rules |
| Active operational support index | [`operations/README.md`](operations/README.md) | Detailed owner/reference catalog for active support docs |
| Tooling and `raccoon-cli` internals | [`tooling/README.md`](tooling/README.md) | Analyzer catalog, drift rules, guardrails |
| Canonical architecture and governance | [`architecture/README.md`](architecture/README.md) | Binding system rules and design records |
| Stage history and delivery evidence | [`stages/INDEX.md`](stages/INDEX.md) | Immutable stage reports |
| Historical and superseded material | [`archive/README.md`](archive/README.md) | Research only, not current source of truth |

## Surface Roles

| Surface | Role |
|---|---|
| Root docs | Short repository entrypoints |
| `docs/operations/` | Active support docs plus documentation governance |
| `docs/tooling/` | Tool-internal reference and rule catalogs |
| `docs/architecture/` | Canonical architecture and runtime governance |
| `docs/stages/` | Immutable stage evidence |
| `docs/archive/` | Superseded or historical material |

## Fast Paths

### Daily development

1. Read [`../DEVELOPMENT.md`](../DEVELOPMENT.md).
2. Use [`operations/documentary-ownership-and-canonical-navigation.md`](operations/documentary-ownership-and-canonical-navigation.md) if ownership between surfaces is unclear.
3. Use [`operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`](operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md) when you know the task but not the physical repository area.
4. Use [`operations/developer-workflow-unification.md`](operations/developer-workflow-unification.md) to follow the official path.
5. Use [`operations/developer-onboarding-and-troubleshooting-guide.md`](operations/developer-onboarding-and-troubleshooting-guide.md) for onboarding and incident-style troubleshooting.
6. Use [`tooling/README.md`](tooling/README.md) only when you need direct `raccoon-cli` detail.

### Runtime and operator workflows

1. Start in [`operations/README.md`](operations/README.md).
2. Use [`operations/documentary-ownership-and-canonical-navigation.md`](operations/documentary-ownership-and-canonical-navigation.md) when the boundary between operations and architecture is unclear.
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
- Prefer [`operations/documentary-ownership-and-canonical-navigation.md`](operations/documentary-ownership-and-canonical-navigation.md) when the question is "who owns this subject and which docs are references only?"
- Prefer [`operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`](operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md) when the problem is "where in the tree do I start?"
- Prefer local area entrypoints such as [`../cmd/README.md`](../cmd/README.md) and [`../internal/README.md`](../internal/README.md) before scanning directories blindly.
- Prefer [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md) for placement, naming, and maintenance rules.
- Prefer [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md) for how stage history should stay readable, linked, and governable.
- Prefer [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md) for how to navigate charter, execution, and gate artifacts across waves.
- Prefer [`operations/repository-policy-and-lightweight-enforcement-2.md`](operations/repository-policy-and-lightweight-enforcement-2.md) for what repository-policy invariants are actively enforced.
- Prefer `docs/tooling/` for what the tooling enforces.
- Prefer `docs/architecture/` for how the system is designed and governed.

## Related Documents

- [`operations/documentary-ownership-and-canonical-navigation.md`](operations/documentary-ownership-and-canonical-navigation.md)
- [`operations/documentation-system-hardening.md`](operations/documentation-system-hardening.md)
- [`operations/repository-metadata-indexes-and-developer-navigation-system.md`](operations/repository-metadata-indexes-and-developer-navigation-system.md)
- [`operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`](operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md)
- [`operations/documentation-governance-entrypoints-and-taxonomy.md`](operations/documentation-governance-entrypoints-and-taxonomy.md)
- [`operations/stage-documentation-governance-and-narrative-coherence.md`](operations/stage-documentation-governance-and-narrative-coherence.md)
- [`operations/stage-history-traceability-and-linking-model.md`](operations/stage-history-traceability-and-linking-model.md)
- [`operations/repository-policy-and-lightweight-enforcement-2.md`](operations/repository-policy-and-lightweight-enforcement-2.md)
