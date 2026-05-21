# Tooling Documentation

## Purpose

This directory holds tooling-internal reference material for `market-foundry`,
primarily the `raccoon-cli` guardrails, drift rules, and topology audits.

Use this directory when you need to understand what the tooling enforces, not
when you only need the day-to-day command surface.

Canonical workflow note:

- prefer `make` for the repository-level workflow contract;
- use [`../development/owners.md`](../development/owners.md) when deciding whether a topic belongs to development, tooling, architecture, or history;
- use [`../development/commands-and-proofs.md`](../development/commands-and-proofs.md) when deciding whether a tooling action belongs in the main contributor lifecycle or only in expert tooling flows;
- use direct `raccoon-cli` commands when you need expert inspection, machine-readable output, or you are changing the CLI/tooling layer itself.

## Start Here

| Need | Primary document |
|---|---|
| Tooling entrypoint and taxonomy | [`cli-overview.md`](cli-overview.md) |
| Command lifecycle and deprecation policy | [`raccoon-cli-command-lifecycle-and-deprecation-strategy.md`](raccoon-cli-command-lifecycle-and-deprecation-strategy.md) |
| Command catalog maturity and governance | [`raccoon-cli-command-catalog-maturity-model-and-governance.md`](raccoon-cli-command-catalog-maturity-model-and-governance.md) |
| Development CLI reliability strategy | [`development-cli-reliability-and-command-testing-strategy.md`](development-cli-reliability-and-command-testing-strategy.md) |
| CLI trustworthiness and error semantics | [`raccoon-cli-command-trustworthiness-and-error-semantics.md`](raccoon-cli-command-trustworthiness-and-error-semantics.md) |
| Raccoon CLI internal architecture | [`raccoon-cli-internal-modularity-and-command-architecture.md`](raccoon-cli-internal-modularity-and-command-architecture.md) |
| Raccoon CLI module rules | [`raccoon-cli-module-boundaries-and-evolution-rules.md`](raccoon-cli-module-boundaries-and-evolution-rules.md) |
| Advanced CLI architecture refinement | [`raccoon-cli-advanced-architecture-refinement.md`](raccoon-cli-advanced-architecture-refinement.md) |
| Internal refactor and extension rules | [`raccoon-cli-internal-refactor-rules-and-extension-guidelines.md`](raccoon-cli-internal-refactor-rules-and-extension-guidelines.md) |
| Human-facing workflow boundary | [`../development/commands-and-proofs.md`](../development/commands-and-proofs.md) |
| Legacy CLI bridge docs | [`../archive/operations/README.md`](../archive/operations/README.md) |
| Architecture guardrails enforced by the CLI | [`cli-architecture-guardrails.md`](cli-architecture-guardrails.md) |
| Topology audit rules | [`cli-topology-audit.md`](cli-topology-audit.md) |

## Owner And Reference Split

| Type | Documents |
|---|---|
| Area owner | [`cli-overview.md`](cli-overview.md), [`README.md`](README.md) |
| Internal architecture and governance | [`raccoon-cli-command-lifecycle-and-deprecation-strategy.md`](raccoon-cli-command-lifecycle-and-deprecation-strategy.md), [`raccoon-cli-command-catalog-maturity-model-and-governance.md`](raccoon-cli-command-catalog-maturity-model-and-governance.md), [`raccoon-cli-command-trustworthiness-and-error-semantics.md`](raccoon-cli-command-trustworthiness-and-error-semantics.md), [`raccoon-cli-internal-modularity-and-command-architecture.md`](raccoon-cli-internal-modularity-and-command-architecture.md), [`raccoon-cli-module-boundaries-and-evolution-rules.md`](raccoon-cli-module-boundaries-and-evolution-rules.md), [`raccoon-cli-advanced-architecture-refinement.md`](raccoon-cli-advanced-architecture-refinement.md), [`raccoon-cli-internal-refactor-rules-and-extension-guidelines.md`](raccoon-cli-internal-refactor-rules-and-extension-guidelines.md), [`development-cli-reliability-and-command-testing-strategy.md`](development-cli-reliability-and-command-testing-strategy.md) |
| Rule catalogs | [`cli-architecture-guardrails.md`](cli-architecture-guardrails.md), [`cli-topology-audit.md`](cli-topology-audit.md), [`cli-drift-rules.md`](cli-drift-rules.md), [`cli-signal-guardrails.md`](cli-signal-guardrails.md), [`cli-signal-drift-rules.md`](cli-signal-drift-rules.md), [`cli-decision-guardrails.md`](cli-decision-guardrails.md), [`cli-decision-drift-rules.md`](cli-decision-drift-rules.md), [`cli-strategy-guardrails.md`](cli-strategy-guardrails.md), [`cli-strategy-drift-rules.md`](cli-strategy-drift-rules.md), [`cli-risk-guardrails.md`](cli-risk-guardrails.md), [`cli-risk-drift-rules.md`](cli-risk-drift-rules.md), [`cli-execution-guardrails.md`](cli-execution-guardrails.md), [`cli-execution-drift-rules.md`](cli-execution-drift-rules.md), [`cli-execute-drift-rules.md`](cli-execute-drift-rules.md) |
| Legacy bridge docs | [`../archive/operations/README.md`](../archive/operations/README.md) |

## Tooling Map

### Core CLI references

- [`cli-overview.md`](cli-overview.md)
- [`raccoon-cli-command-lifecycle-and-deprecation-strategy.md`](raccoon-cli-command-lifecycle-and-deprecation-strategy.md)
- [`raccoon-cli-command-catalog-maturity-model-and-governance.md`](raccoon-cli-command-catalog-maturity-model-and-governance.md)
- [`development-cli-reliability-and-command-testing-strategy.md`](development-cli-reliability-and-command-testing-strategy.md)
- [`raccoon-cli-command-trustworthiness-and-error-semantics.md`](raccoon-cli-command-trustworthiness-and-error-semantics.md)
- [`raccoon-cli-internal-modularity-and-command-architecture.md`](raccoon-cli-internal-modularity-and-command-architecture.md)
- [`raccoon-cli-module-boundaries-and-evolution-rules.md`](raccoon-cli-module-boundaries-and-evolution-rules.md)
- [`raccoon-cli-advanced-architecture-refinement.md`](raccoon-cli-advanced-architecture-refinement.md)
- [`raccoon-cli-internal-refactor-rules-and-extension-guidelines.md`](raccoon-cli-internal-refactor-rules-and-extension-guidelines.md)
- [`cli-architecture-guardrails.md`](cli-architecture-guardrails.md)
- [`cli-topology-audit.md`](cli-topology-audit.md)
- [`cli-drift-rules.md`](cli-drift-rules.md)

### Domain guardrails and drift rules

- [`cli-signal-guardrails.md`](cli-signal-guardrails.md)
- [`cli-signal-drift-rules.md`](cli-signal-drift-rules.md)
- [`cli-decision-guardrails.md`](cli-decision-guardrails.md)
- [`cli-decision-drift-rules.md`](cli-decision-drift-rules.md)
- [`cli-strategy-guardrails.md`](cli-strategy-guardrails.md)
- [`cli-strategy-drift-rules.md`](cli-strategy-drift-rules.md)
- [`cli-risk-guardrails.md`](cli-risk-guardrails.md)
- [`cli-risk-drift-rules.md`](cli-risk-drift-rules.md)
- [`cli-execution-guardrails.md`](cli-execution-guardrails.md)
- [`cli-execution-drift-rules.md`](cli-execution-drift-rules.md)
- [`cli-execute-drift-rules.md`](cli-execute-drift-rules.md)

## Naming Notes

- `cli-execution-*` documents the execution domain and its cross-service rules.
- `cli-execute-*` documents execute-binary-specific governance that extends the
  execution domain rules.

This distinction is easy to miss when scanning filenames, so the index makes it
explicit.

## Placement Rules

- Keep analyzer behavior, rule catalogs, and audit definitions in this directory.
- Keep user-facing workflow guidance in [`../development/`](../development/README.md).
- Update the relevant tooling doc whenever analyzer behavior changes.
