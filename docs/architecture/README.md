# Architecture Documentation

## Purpose

This directory contains the canonical architecture and governance record for
`market-foundry`.

These documents are authoritative for system shape, boundaries, conventions,
runtime behavior, and governed expansion.

## Start Here

| Need | Primary document |
|---|---|
| System identity and design direction | [`system-vision.md`](system-vision.md), [`system-principles.md`](system-principles.md) |
| Repository evolution governance | [`market-foundry-evolution-playbook.md`](market-foundry-evolution-playbook.md) |
| Stage completion rules | [`stage-definition-of-done.md`](stage-definition-of-done.md) |
| Debt-prevention rules | [`anti-debt-checklist.md`](anti-debt-checklist.md) |
| Repository boundaries and naming | [`monorepo-structure-and-engineering-conventions.md`](monorepo-structure-and-engineering-conventions.md), [`naming-conventions-for-domains-families-and-runtimes.md`](naming-conventions-for-domains-families-and-runtimes.md) |
| Documentation governance | [`monorepo-documentation-and-stage-governance.md`](monorepo-documentation-and-stage-governance.md) |
| Stage-history reading model | [`../operations/stage-documentation-governance-and-narrative-coherence.md`](../operations/stage-documentation-governance-and-narrative-coherence.md), [`../operations/stage-history-traceability-and-linking-model.md`](../operations/stage-history-traceability-and-linking-model.md) |
| Documentation-system entrypoints and taxonomy | [`../operations/documentation-system-hardening.md`](../operations/documentation-system-hardening.md), [`../operations/documentation-governance-entrypoints-and-taxonomy.md`](../operations/documentation-governance-entrypoints-and-taxonomy.md) |
| Canonical architecture map after prior consolidation | [`documentation-canonical-map-after-consolidation.md`](documentation-canonical-map-after-consolidation.md) |

## Navigate By Use

### Governance and repository rules

- [`market-foundry-evolution-playbook.md`](market-foundry-evolution-playbook.md)
- [`stage-definition-of-done.md`](stage-definition-of-done.md)
- [`anti-debt-checklist.md`](anti-debt-checklist.md)
- [`prohibited-carryovers.md`](prohibited-carryovers.md)
- [`monorepo-documentation-and-stage-governance.md`](monorepo-documentation-and-stage-governance.md)
- [`../operations/stage-documentation-governance-and-narrative-coherence.md`](../operations/stage-documentation-governance-and-narrative-coherence.md)
- [`../operations/stage-history-traceability-and-linking-model.md`](../operations/stage-history-traceability-and-linking-model.md)

### Runtime and operational architecture

- [`runtime-target.md`](runtime-target.md)
- [`runtime-assembly-guidelines.md`](runtime-assembly-guidelines.md)
- [`operational-contracts-and-cross-runtime-conventions.md`](operational-contracts-and-cross-runtime-conventions.md)
- [`minimal-operational-baseline.md`](minimal-operational-baseline.md)
- [`current-baseline-runbook.md`](current-baseline-runbook.md)

### Domain design

- Signal: [`signal-domain-design.md`](signal-domain-design.md)
- Decision: [`decision-domain-design.md`](decision-domain-design.md)
- Strategy: [`strategy-domain-design.md`](strategy-domain-design.md)
- Risk: [`risk-domain-design.md`](risk-domain-design.md)
- Execution: [`execution-domain-design.md`](execution-domain-design.md)

### Venue Readiness (Phase 30)

- [`venue-readiness-charter-and-scope-freeze.md`](venue-readiness-charter-and-scope-freeze.md)
- [`venue-readiness-capabilities-questions-and-non-goals.md`](venue-readiness-capabilities-questions-and-non-goals.md)

### Analytical and ClickHouse

- [`analytical-storage-strategy.md`](analytical-storage-strategy.md)
- [`analytical-observability-and-runbook.md`](analytical-observability-and-runbook.md)
- [`clickhouse-core-schema-design.md`](clickhouse-core-schema-design.md)
- [`cmd-migrate-and-migration-catalog.md`](cmd-migrate-and-migration-catalog.md)

## Usage Rules

- Use this directory for canonical architecture, not for day-to-day workflow help.
- If you need operational navigation first, start in
  [`../operations/README.md`](../operations/README.md).
- If you need documentation-system placement or entrypoint rules first, start in
  [`../operations/documentation-governance-entrypoints-and-taxonomy.md`](../operations/documentation-governance-entrypoints-and-taxonomy.md).
- If you need historical delivery evidence, use [`../stages/INDEX.md`](../stages/INDEX.md).
