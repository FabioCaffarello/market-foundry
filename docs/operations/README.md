# Operations Documentation

## Purpose

This directory is the operational and support-documentation home for
`market-foundry`.

It is the first stop for:

- daily development workflow navigation
- Makefile and script usage
- `raccoon-cli` command usage as an operator/developer tool
- documentation system rules and conventions

It is not the place for deep architecture history or immutable stage evidence.

## Start Here

| Need | Primary document |
|---|---|
| Documentation system map and cross-surface ownership | [`documentation-system-hardening.md`](documentation-system-hardening.md) |
| Documentation governance, entrypoints, and taxonomy | [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md) |
| Developer environment architecture and lifecycle | [`development-environment-architecture-and-lifecycle.md`](development-environment-architecture-and-lifecycle.md) |
| Canonical lifecycle entrypoints and flows | [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md) |
| Unified official developer workflow | [`developer-workflow-unification.md`](developer-workflow-unification.md) |
| Onboarding and first-line troubleshooting | [`developer-onboarding-and-troubleshooting-guide.md`](developer-onboarding-and-troubleshooting-guide.md) |
| Daily workflow | [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md) |
| Canonical support-surface model | [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md) |
| Repository support convergence decisions | [`repository-architecture-convergence.md`](repository-architecture-convergence.md) |
| Smoke and harness governance model | [`smoke-and-operational-harness-governance.md`](smoke-and-operational-harness-governance.md) |
| Operational proof entrypoints and ownership | [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md) |
| Smoke/proof UX model and ergonomic rules | [`smoke-ux-and-proof-execution-ergonomics.md`](smoke-ux-and-proof-execution-ergonomics.md) |
| Proof execution flows and failure diagnosis | [`proof-execution-user-flows-and-failure-diagnosis.md`](proof-execution-user-flows-and-failure-diagnosis.md) |
| Makefile command surface | [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md) |
| Lightweight repository consistency guard rails | [`lightweight-repository-guard-rails-and-consistency-checks.md`](lightweight-repository-guard-rails-and-consistency-checks.md) |
| Consistency invariants and severity policy | [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md) |
| Repository policy and lightweight enforcement v2 | [`repository-policy-and-lightweight-enforcement-2.md`](repository-policy-and-lightweight-enforcement-2.md) |
| Repository invariants matrix and enforcement policy | [`repository-invariants-check-matrix-and-enforcement-policy.md`](repository-invariants-check-matrix-and-enforcement-policy.md) |
| Stage tooling support model | [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md) |
| Stage artifact naming and completeness conventions | [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md) |
| Stage documentation governance and narrative coherence | [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md) |
| Stage history traceability and linking model | [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md) |
| Script entrypoints | [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md) |
| `raccoon-cli` user-facing command reference | [`raccoon-cli-command-reference.md`](raccoon-cli-command-reference.md) |
| Documentation navigation changes introduced in C5 | [`documentation-reorganization-and-operational-navigation.md`](documentation-reorganization-and-operational-navigation.md) |
| Taxonomy and authoring rules for new docs | [`documentation-taxonomy-and-authoring-conventions.md`](documentation-taxonomy-and-authoring-conventions.md) |

## Canonical Documentation System Entrypoints

| Concern | Canonical entrypoint | Notes |
|---|---|---|
| Repository overview | [`../../README.md`](../../README.md) | High-level orientation only |
| Daily workflow | [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md) | Operational loop, not taxonomy |
| Documentation system map | [`documentation-system-hardening.md`](documentation-system-hardening.md) | Canonical map between operations, architecture, tooling, stages, and archive |
| Documentation governance and taxonomy | [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md) | Canonical rules for placement, naming, and maintenance |
| Developer environment architecture and lifecycle | [`development-environment-architecture-and-lifecycle.md`](development-environment-architecture-and-lifecycle.md) | Canonical lifecycle model and support-surface hierarchy |
| Canonical lifecycle flows and entrypoints | [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md) | Flow-by-flow commands for bootstrap, dev loop, smoke, troubleshooting, and reset |
| Operations support docs | [`README.md`](README.md) | User-facing support surface index |
| Tooling reference | [`../tooling/README.md`](../tooling/README.md) | Analyzer and rule catalogs |
| Architecture | [`../architecture/README.md`](../architecture/README.md) | Binding architecture and governance |
| Stage evidence | [`../stages/INDEX.md`](../stages/INDEX.md) | Historical record only |
| Archive | [`../archive/README.md`](../archive/README.md) | Non-canonical history |

## Operational Navigation

### Daily workflow and command surface

- [`documentation-system-hardening.md`](documentation-system-hardening.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`development-environment-architecture-and-lifecycle.md`](development-environment-architecture-and-lifecycle.md)
- [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md)
- [`developer-workflow-unification.md`](developer-workflow-unification.md)
- [`developer-onboarding-and-troubleshooting-guide.md`](developer-onboarding-and-troubleshooting-guide.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md)
- [`repository-architecture-convergence.md`](repository-architecture-convergence.md)
- [`smoke-and-operational-harness-governance.md`](smoke-and-operational-harness-governance.md)
- [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md)
- [`smoke-ux-and-proof-execution-ergonomics.md`](smoke-ux-and-proof-execution-ergonomics.md)
- [`proof-execution-user-flows-and-failure-diagnosis.md`](proof-execution-user-flows-and-failure-diagnosis.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`lightweight-repository-guard-rails-and-consistency-checks.md`](lightweight-repository-guard-rails-and-consistency-checks.md)
- [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md)
- [`repository-policy-and-lightweight-enforcement-2.md`](repository-policy-and-lightweight-enforcement-2.md)
- [`repository-invariants-check-matrix-and-enforcement-policy.md`](repository-invariants-check-matrix-and-enforcement-policy.md)
- [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md)
- [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md)
- [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md)
- [`stage-history-traceability-and-linking-model.md`](stage-history-traceability-and-linking-model.md)
- [`makefile-command-ergonomics-and-hardening.md`](makefile-command-ergonomics-and-hardening.md)

### Scripts and harnesses

- [`smoke-and-operational-harness-governance.md`](smoke-and-operational-harness-governance.md)
- [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md)
- [`smoke-ux-and-proof-execution-ergonomics.md`](smoke-ux-and-proof-execution-ergonomics.md)
- [`proof-execution-user-flows-and-failure-diagnosis.md`](proof-execution-user-flows-and-failure-diagnosis.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
- [`scripts-normalization-and-harness-hygiene.md`](scripts-normalization-and-harness-hygiene.md)

### `raccoon-cli` as a support tool

- Prefer `make check`, `make tdd`, `make coverage-map`, and `make recommend` when those wrappers already match the workflow you need.
- Use direct `raccoon-cli` commands when you need expert inspection depth, JSON output, narrower scope, or you are evolving the CLI itself.
- [`../tooling/README.md`](../tooling/README.md)
- [`raccoon-cli-command-reference.md`](raccoon-cli-command-reference.md)
- [`raccoon-cli-ux-taxonomy-and-guard-rails.md`](raccoon-cli-ux-taxonomy-and-guard-rails.md)

### Historical support-surface audits and planning

- [`repo-support-surface-audit.md`](repo-support-surface-audit.md)
- [`repo-support-prioritized-improvement-matrix.md`](repo-support-prioritized-improvement-matrix.md)
- [`documentation-reorganization-and-operational-navigation.md`](documentation-reorganization-and-operational-navigation.md)
- [`documentation-taxonomy-and-authoring-conventions.md`](documentation-taxonomy-and-authoring-conventions.md)

## Canonical Runbooks That Remain In Architecture

These documents stay under `docs/architecture/` because they are also canonical
architecture records. They are linked here so operators do not need to search
the architecture corpus manually.

- [`../architecture/minimal-operational-baseline.md`](../architecture/minimal-operational-baseline.md)
- [`../architecture/current-baseline-runbook.md`](../architecture/current-baseline-runbook.md)
- [`../architecture/current-baseline-operational-diagnostics.md`](../architecture/current-baseline-operational-diagnostics.md)
- [`../architecture/operational-contracts-and-cross-runtime-conventions.md`](../architecture/operational-contracts-and-cross-runtime-conventions.md)
- [`../architecture/operational-smoke-ci-and-runbook-closure.md`](../architecture/operational-smoke-ci-and-runbook-closure.md)
- [`../architecture/analytical-observability-and-runbook.md`](../architecture/analytical-observability-and-runbook.md)

## Placement Rules

- Put user-facing operational workflow docs here.
- Keep documentation-system navigation and governance docs here when they govern repository usage rather than system architecture.
- Put tool-internal analyzer or rule docs in [`../tooling/`](../tooling/README.md).
- Put binding architectural rules in [`../architecture/`](../architecture/README.md).
- Put immutable execution evidence in [`../stages/`](../stages/INDEX.md).
- Link to canonical architecture docs instead of copying their content here.
- Keep stage-history readability rules here when they govern how contributors
  navigate reports rather than what the system architecture is.
