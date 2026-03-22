# Documentary Ownership And Canonical Navigation

## Purpose

This document is the canonical ownership map for the active documentation
topology in `market-foundry`.

Use it when the question is:

- which document owns a subject;
- which documents are supporting references only;
- where historical rationale lives without competing with the current owner;
- which indexes should stay short and which ones may hold deeper catalogs.

## Ownership Model

- owner docs answer the current recurring question for a subject;
- reference docs deepen, operationalize, or exemplify the owner doc;
- historical docs preserve rationale, sequence, and evidence, but do not own the
  current rule;
- indexes route readers to owners and references, but should not become second
  copies of the owned content.

## Subject Ownership Map

| Subject | Owner doc | Reference docs | Historical rationale |
|---|---|---|---|
| Repository identity and current shape | [`../../README.md`](../../README.md) | [`repository-metadata-indexes-and-developer-navigation-system.md`](repository-metadata-indexes-and-developer-navigation-system.md), [`repository-navigation-maps-entrypoints-and-maintenance-rules.md`](repository-navigation-maps-entrypoints-and-maintenance-rules.md) | [`../stages/INDEX.md`](../stages/INDEX.md), [`../archive/README.md`](../archive/README.md) |
| Daily developer workflow | [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md) | [`development-environment-architecture-and-lifecycle.md`](development-environment-architecture-and-lifecycle.md), [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md), [`developer-workflow-unification.md`](developer-workflow-unification.md), [`developer-onboarding-and-troubleshooting-guide.md`](developer-onboarding-and-troubleshooting-guide.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Documentation ownership and canonical navigation | [`documentary-ownership-and-canonical-navigation.md`](documentary-ownership-and-canonical-navigation.md) | [`../../docs/README.md`](../../docs/README.md), [`README.md`](README.md), [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md) | [`documentation-system-hardening.md`](documentation-system-hardening.md), [`documentation-reorganization-and-operational-navigation.md`](documentation-reorganization-and-operational-navigation.md), [`documentation-taxonomy-and-authoring-conventions.md`](documentation-taxonomy-and-authoring-conventions.md) |
| Documentation placement, naming, and maintenance rules | [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md) | [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md), [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md), [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md), [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md) | [`documentation-system-hardening.md`](documentation-system-hardening.md), [`../stages/INDEX.md`](../stages/INDEX.md) |
| Repository-shape navigation | [`repository-metadata-indexes-and-developer-navigation-system.md`](repository-metadata-indexes-and-developer-navigation-system.md) | [`repository-navigation-maps-entrypoints-and-maintenance-rules.md`](repository-navigation-maps-entrypoints-and-maintenance-rules.md), local area READMEs under `cmd/`, `internal/`, `deploy/`, `scripts/`, and `tests/` | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Command-surface and proof-of-record usage | [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md) | [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md), [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md), [`smoke-and-operational-harness-governance.md`](smoke-and-operational-harness-governance.md), [`smoke-ux-and-proof-execution-ergonomics.md`](smoke-ux-and-proof-execution-ergonomics.md), [`proof-execution-user-flows-and-failure-diagnosis.md`](proof-execution-user-flows-and-failure-diagnosis.md), [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Repository support-surface policy and checks | [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md) | [`repository-architecture-convergence.md`](repository-architecture-convergence.md), [`lightweight-repository-guard-rails-and-consistency-checks.md`](lightweight-repository-guard-rails-and-consistency-checks.md), [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md), [`repository-policy-and-lightweight-enforcement-2.md`](repository-policy-and-lightweight-enforcement-2.md), [`repository-invariants-check-matrix-and-enforcement-policy.md`](repository-invariants-check-matrix-and-enforcement-policy.md) | [`repo-support-surface-audit.md`](repo-support-surface-audit.md), [`repo-support-prioritized-improvement-matrix.md`](repo-support-prioritized-improvement-matrix.md) |
| Stage execution support and stage-history navigation | [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md) | [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md), [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md), [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md), [`../stages/INDEX.md`](../stages/INDEX.md) | stage reports in [`../stages/`](../stages/INDEX.md) |
| Automation boundaries and sustainable routines | [`automation-support-for-waves-execution-continuity-and-repo-sustainability.md`](automation-support-for-waves-execution-continuity-and-repo-sustainability.md) | [`repository-automation-boundaries-high-value-routines-and-sustainability-rules.md`](repository-automation-boundaries-high-value-routines-and-sustainability-rules.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Development-platform operating model and governance | [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md) | [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md), [`strategic-checkpoints-for-the-development-platform.md`](strategic-checkpoints-for-the-development-platform.md), [`development-platform-checkpoint-triggers-scope-and-decision-model.md`](development-platform-checkpoint-triggers-scope-and-decision-model.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Development-platform readiness, wave opening, and prioritization | [`development-platform-readiness-model-for-future-foundry-waves.md`](development-platform-readiness-model-for-future-foundry-waves.md) | [`readiness-signals-saturation-signals-and-wave-opening-rules.md`](readiness-signals-saturation-signals-and-wave-opening-rules.md), [`criteria-for-opening-containing-or-rejecting-new-support-surfaces.md`](criteria-for-opening-containing-or-rejecting-new-support-surfaces.md), [`support-surface-expansion-decision-rules-and-examples.md`](support-surface-expansion-decision-rules-and-examples.md), [`continuous-prioritization-model-for-the-development-platform.md`](continuous-prioritization-model-for-the-development-platform.md), [`prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md`](prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md), [`canonical-workflow-hotspot-assessment-and-selection.md`](canonical-workflow-hotspot-assessment-and-selection.md), [`hotspot-candidates-prioritization-and-selection-rationale.md`](hotspot-candidates-prioritization-and-selection-rationale.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Repository sustainability, lifecycle control, and maintenance economics | [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md) | [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md), [`repository-maintenance-hotspots-and-cost-reduction-principles.md`](repository-maintenance-hotspots-and-cost-reduction-principles.md), [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md), [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md), [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md), [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md), [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md), [`support-surface-sunset-consolidation-and-retirement-strategy.md`](support-surface-sunset-consolidation-and-retirement-strategy.md), [`support-surface-lifecycle-signals-and-consolidation-criteria.md`](support-surface-lifecycle-signals-and-consolidation-criteria.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Tooling-internal rules and `raccoon-cli` internals | [`../tooling/README.md`](../tooling/README.md) | [`raccoon-cli-command-reference.md`](raccoon-cli-command-reference.md), [`raccoon-cli-ux-taxonomy-and-guard-rails.md`](raccoon-cli-ux-taxonomy-and-guard-rails.md), [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md), [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md) | tooling stage reports in [`../stages/INDEX.md`](../stages/INDEX.md) |
| Canonical architecture and runtime rules | [`../architecture/README.md`](../architecture/README.md) | relevant documents under `docs/architecture/` | [`../stages/INDEX.md`](../stages/INDEX.md), [`../archive/README.md`](../archive/README.md) |
| Immutable stage evidence | [`../stages/INDEX.md`](../stages/INDEX.md) | [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md) | stage reports themselves |

## Area Ownership Boundaries

| Surface | Ownership rule |
|---|---|
| `README.md` | Short repository identity and orientation only |
| `DEVELOPMENT.md` | Daily workflow only |
| `docs/README.md` | Cross-surface routing only |
| `docs/operations/README.md` | Detailed active support index for owner and reference docs |
| `docs/tooling/README.md` | Detailed tooling-internal index |
| `docs/architecture/README.md` | Architecture corpus entrypoint |
| `docs/stages/INDEX.md` | History inventory only |
| `docs/archive/README.md` | Archive inventory only |

## Compression Rules

- keep root docs curated and shallow;
- keep exactly one detailed active support catalog in
  [`README.md`](README.md);
- keep owner/reference separation explicit in tables and headings;
- preserve historical bridge docs, but label them as historical or bridge
  surfaces instead of allowing them to read as current owners;
- prefer adding one row to an ownership map over repeating the same catalog in
  multiple indexes.

## Minimum Update Set

When a new durable documentation concern is introduced, review at least:

- [`../../README.md`](../../README.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`../../docs/README.md`](../../docs/README.md)
- [`README.md`](README.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`../tooling/README.md`](../tooling/README.md) if tooling ownership changed
- [`../stages/INDEX.md`](../stages/INDEX.md) if a new stage report was added

## Related Documents

- [`../../docs/README.md`](../../docs/README.md)
- [`README.md`](README.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`documentation-system-hardening.md`](documentation-system-hardening.md)
- [`../tooling/README.md`](../tooling/README.md)
- [`../architecture/README.md`](../architecture/README.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
