# Tooling Documentation

## Purpose

This directory holds tooling-internal reference material for `market-foundry`,
primarily the `raccoon-cli` guardrails, drift rules, and topology audits.

Use this directory when you need to understand what the tooling enforces, not
when you only need the day-to-day command surface.

Canonical workflow note:

- prefer `make` for the repository-level workflow contract;
- use [`../operations/development-environment-architecture-and-lifecycle.md`](../operations/development-environment-architecture-and-lifecycle.md) and [`../operations/development-lifecycle-entrypoints-and-canonical-flows.md`](../operations/development-lifecycle-entrypoints-and-canonical-flows.md) when deciding whether a tooling action belongs in the main developer lifecycle or only in expert tooling flows;
- use direct `raccoon-cli` commands when you need expert inspection, machine-readable output, or you are changing the CLI/tooling layer itself.

## Start Here

| Need | Primary document |
|---|---|
| Tooling entrypoint and taxonomy | [`cli-overview.md`](cli-overview.md) |
| Development CLI reliability strategy | [`development-cli-reliability-and-command-testing-strategy.md`](development-cli-reliability-and-command-testing-strategy.md) |
| CLI trustworthiness and error semantics | [`raccoon-cli-command-trustworthiness-and-error-semantics.md`](raccoon-cli-command-trustworthiness-and-error-semantics.md) |
| Raccoon CLI internal architecture | [`raccoon-cli-internal-modularity-and-command-architecture.md`](raccoon-cli-internal-modularity-and-command-architecture.md) |
| Raccoon CLI module rules | [`raccoon-cli-module-boundaries-and-evolution-rules.md`](raccoon-cli-module-boundaries-and-evolution-rules.md) |
| User-facing `raccoon-cli` commands | [`../operations/raccoon-cli-command-reference.md`](../operations/raccoon-cli-command-reference.md) |
| CLI UX taxonomy and guard rails | [`../operations/raccoon-cli-ux-taxonomy-and-guard-rails.md`](../operations/raccoon-cli-ux-taxonomy-and-guard-rails.md) |
| Architecture guardrails enforced by the CLI | [`cli-architecture-guardrails.md`](cli-architecture-guardrails.md) |
| Topology audit rules | [`cli-topology-audit.md`](cli-topology-audit.md) |

## Tooling Map

### Core CLI references

- [`cli-overview.md`](cli-overview.md)
- [`development-cli-reliability-and-command-testing-strategy.md`](development-cli-reliability-and-command-testing-strategy.md)
- [`raccoon-cli-command-trustworthiness-and-error-semantics.md`](raccoon-cli-command-trustworthiness-and-error-semantics.md)
- [`raccoon-cli-internal-modularity-and-command-architecture.md`](raccoon-cli-internal-modularity-and-command-architecture.md)
- [`raccoon-cli-module-boundaries-and-evolution-rules.md`](raccoon-cli-module-boundaries-and-evolution-rules.md)
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
- Keep user-facing workflow guidance in [`../operations/`](../operations/README.md).
- Update the relevant tooling doc whenever analyzer behavior changes.
