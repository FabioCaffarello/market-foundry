# Documentation Canonical Map After Consolidation (S215)

**Date:** 2026-03-20

---

## Summary

| Metric | Count |
|--------|-------|
| Active architecture docs (`docs/architecture/`) | 240 |
| Archived docs (`docs/archive/`, 16 subdirectories) | 245 |
| Stage reports (`docs/stages/`) | 212 (including INDEX.md) |

The consolidation waves (S205-S215) reduced documentation entropy by archiving superseded, redundant, and phase-specific artifacts into semantically grouped subdirectories under `docs/archive/`. The 238 surviving architecture documents represent the canonical, authoritative set.

---

## Canonical Architecture Documents by Category

### System & Vision

| File | Purpose |
|------|---------|
| `system-principles.md` | Core system principles and invariants |
| `system-vision.md` | Long-term system vision |
| `non-goals.md` | Explicit non-goals and boundaries |
| `market-foundry-evolution-playbook.md` | Evolution playbook and phase sequencing |
| `domain-readiness.md` | Domain readiness assessment |
| `expansion-playbooks-refined.md` | Refined expansion playbooks |
| `how-to-introduce-new-runtimes-domains-and-families.md` | Guide for introducing new runtimes, domains, families |
| `structural-anti-patterns-and-when-not-to-expand.md` | Anti-patterns and expansion gates |
| `foundation-confidence-rules.md` | Confidence rules for foundation decisions |
| `technical-governance-refinement.md` | Governance refinement decisions |

### Governance & Conventions

| File | Purpose |
|------|---------|
| `monorepo-structure-and-engineering-conventions.md` | Monorepo layout and engineering conventions |
| `monorepo-documentation-and-stage-governance.md` | Documentation and stage governance rules |
| `naming-conventions-for-domains-families-and-runtimes.md` | Naming conventions |
| `stage-definition-of-done.md` | Definition of done for stages |
| `opus-guidance-rules.md` | Guidance rules for Opus agent |
| `governance-hygiene-status.md` | Governance hygiene tracking |
| `prohibited-carryovers.md` | Prohibited carryovers between phases |
| `adoption-matrix.md` | Adoption matrix for patterns and conventions |

### Runtime Architecture

| File | Purpose |
|------|---------|
| `runtime-composition-pattern.md` | Runtime composition pattern |
| `runtime-assembly-guidelines.md` | Runtime assembly guidelines |
| `runtime-lifecycle-and-shutdown-model.md` | Lifecycle and shutdown model |
| `runtime-invariants-and-shared-behavior-rules.md` | Shared behavior invariants |
| `runtime-target.md` | Runtime target definition |
| `runtime-recentralization.md` | Recentralization decisions |
| `runtime-config-and-operational-closure.md` | Config and operational closure |
| `dependency-injection-and-composition-roots.md` | DI and composition roots |
| `gateway-pattern.md` | Gateway pattern |
| `gateway-read-surface-guidelines.md` | Gateway read surface guidelines |
| `registry-driven-runtime-assembly.md` | Registry-driven assembly pattern |
| `actor-ownership.md` | Actor ownership model |
| `embedded-nats-integration-proof.md` | Embedded NATS integration proof |
| `mesh-vs-transport.md` | Mesh vs transport distinction |
| `operational-contracts-and-cross-runtime-conventions.md` | Cross-runtime operational contracts |

### Stream Mesh

| File | Purpose |
|------|---------|
| `stream-families.md` | Stream families overview |
| `stream-mesh-model.md` | Stream mesh model |
| `stream-taxonomy.md` | Stream taxonomy |
| `stream-ownership-matrix.md` | Stream ownership matrix |
| `stream-family-catalog.md` | Stream family catalog |
| `stream-family-01-adoption.md` | Family 01 adoption record |
| `mesh-gaps-and-next-moves.md` | Mesh gaps and next moves |
| `projection-families-model.md` | Projection families model |
| `projection-family-matrix.md` | Projection family matrix |
| `query-contracts-by-family.md` | Query contracts by family |
| `latest-history-by-family.md` | Latest/history query by family |

### Domain Design -- Signal

| File | Purpose |
|------|---------|
| `signal-domain-design.md` | Signal domain design |
| `signal-first-slice.md` | Signal first slice |
| `signal-projection-pattern.md` | Signal projection pattern |
| `signal-family-01-contracts.md` | Signal family 01 contracts |
| `signal-stream-families.md` | Signal stream families |
| `signal-activation-and-ownership.md` | Signal activation and ownership |
| `signal-query-surface-guidelines.md` | Signal query surface guidelines |
| `signal-replay-idempotency-rules.md` | Signal replay idempotency rules |

### Domain Design -- Decision

| File | Purpose |
|------|---------|
| `decision-domain-design.md` | Decision domain design |
| `decision-first-slice.md` | Decision first slice |
| `decision-projection-pattern.md` | Decision projection pattern |
| `decision-family-01-contracts.md` | Decision family 01 contracts |
| `decision-stream-families.md` | Decision stream families |
| `decision-activation-and-ownership.md` | Decision activation and ownership |
| `decision-query-surface-guidelines.md` | Decision query surface guidelines |
| `decision-replay-idempotency-rules.md` | Decision replay idempotency rules |
| `cc-02-family-definition.md` | CC-02 family definition |
| `cc-02-implementation-notes.md` | CC-02 implementation notes |

### Domain Design -- Strategy

| File | Purpose |
|------|---------|
| `strategy-domain-design.md` | Strategy domain design |
| `strategy-first-slice.md` | Strategy first slice |
| `strategy-projection-pattern.md` | Strategy projection pattern |
| `strategy-stream-families.md` | Strategy stream families |
| `strategy-activation-and-ownership.md` | Strategy activation and ownership |
| `strategy-query-surface-guidelines.md` | Strategy query surface guidelines |
| `strategy-replay-idempotency-rules.md` | Strategy replay idempotency rules |

### Domain Design -- Risk

| File | Purpose |
|------|---------|
| `risk-domain-design.md` | Risk domain design |
| `risk-first-slice.md` | Risk first slice |
| `risk-projection-pattern.md` | Risk projection pattern |
| `risk-family-01-contracts.md` | Risk family 01 contracts |
| `risk-stream-families.md` | Risk stream families |
| `risk-activation-and-ownership.md` | Risk activation and ownership |
| `risk-query-surface-guidelines.md` | Risk query surface guidelines |
| `risk-replay-idempotency-rules.md` | Risk replay idempotency rules |

### Domain Design -- Execution

| File | Purpose |
|------|---------|
| `execution-domain-design.md` | Execution domain design |
| `execution-first-slice.md` | Execution first slice |
| `execution-projection-pattern.md` | Execution projection pattern |
| `execution-family-01-contracts.md` | Execution family 01 contracts |
| `execution-stream-families.md` | Execution stream families |
| `execution-activation-and-ownership.md` | Execution activation and ownership |
| `execution-query-surface-guidelines.md` | Execution query surface guidelines |
| `execution-replay-idempotency-rules.md` | Execution replay idempotency rules |
| `execution-lifecycle-model.md` | Execution lifecycle model |
| `execution-fill-model.md` | Execution fill model |
| `execution-fill-projection-pattern.md` | Execution fill projection pattern |
| `execution-failure-recovery-model.md` | Execution failure recovery model |
| `execution-control-and-kill-switch.md` | Execution control and kill switch |
| `execution-status-propagation-model.md` | Execution status propagation model |
| `execution-projection-failure-semantics.md` | Execution projection failure semantics |
| `execution-trace-persistence.md` | Execution trace persistence |
| `execution-family-separation-after-paper-step.md` | Family separation after paper step |
| `execution-query-surface-after-execute.md` | Query surface after execute |
| `execution-read-side-authority-after-execute.md` | Read-side authority after execute |
| `execution-operational-validation-matrix.md` | Operational validation matrix |
| `execution-integrated-operational-validation-matrix.md` | Integrated operational validation matrix |
| `execute-actor-critical-test-coverage.md` | Execute actor critical test coverage |
| `execute-actor-safety-model.md` | Execute actor safety model |
| `execute-governance-and-activation-model.md` | Execute governance and activation model |
| `execute-observability-and-runtime-health.md` | Execute observability and runtime health |
| `execute-operational-platform-integration.md` | Execute operational platform integration |
| `execute-runtime-and-activation-model.md` | Execute runtime and activation model |

### Evidence & Projection

| File | Purpose |
|------|---------|
| `evidence-derivation-pattern.md` | Evidence derivation pattern |
| `evidence-query-model-consolidation.md` | Evidence query model consolidation |
| `evidence-read-model-guidelines.md` | Evidence read model guidelines |
| `evidence-type-01-contracts.md` | Evidence type 01 contracts |
| `projection-confidence-and-dual-write-review.md` | Projection confidence and dual-write review |
| `projection-writer-pattern.md` | Projection writer pattern |
| `multi-projection-pattern.md` | Multi-projection pattern |
| `derive-actor-confidence-rules.md` | Derive actor confidence rules |
| `derive-family-processor-pattern.md` | Derive family processor pattern |
| `derive-pipeline-pattern.md` | Derive pipeline pattern |
| `read-model-authority.md` | Read model authority |
| `replay-idempotency-rules.md` | Replay idempotency rules (cross-domain) |

### Config & Activation

| File | Purpose |
|------|---------|
| `config-activation-and-dependency-map-model.md` | Config activation and dependency map model |
| `config-driven-activation-hardening.md` | Config-driven activation hardening |
| `config-validation-and-sync-rules.md` | Config validation and sync rules |
| `family-config-dependency-rules.md` | Family config dependency rules |
| `family-runtime-registration-rules.md` | Family runtime registration rules |

### ClickHouse & Analytical

| File | Purpose |
|------|---------|
| `clickhouse-core-schema-design.md` | ClickHouse core schema design |
| `clickhouse-core-tables-and-ddl-rationale.md` | Core tables and DDL rationale |
| `clickhouse-entry-architecture.md` | ClickHouse entry architecture |
| `clickhouse-schema-versioning-and-evolution-rules.md` | Schema versioning and evolution rules |
| `analytical-boundary-and-responsibility-model.md` | Analytical boundary and responsibility model |
| `analytical-generated-path-consolidation.md` | Analytical generated path consolidation |
| `analytical-implementation-closure.md` | Analytical implementation closure |
| `analytical-observability-and-runbook.md` | Analytical observability and runbook |
| `analytical-runtime-lifecycle-and-recovery.md` | Analytical runtime lifecycle and recovery |
| `analytical-scope-and-planning-summary.md` | Analytical scope and planning summary |
| `analytical-storage-strategy.md` | Analytical storage strategy |
| `analytical-vs-generated-ownership-and-boundaries.md` | Analytical vs generated ownership and boundaries |
| `analytical-writer-correctness-and-test-foundation.md` | Analytical writer correctness and test foundation |
| `writer-service-architecture.md` | Writer service architecture |
| `writer-service-failure-and-delivery-semantics.md` | Writer service failure and delivery semantics |
| `writer-service-initial-event-coverage-and-limits.md` | Writer service initial event coverage and limits |
| `writer-service-minimal-implementation.md` | Writer service minimal implementation |
| `writer-service-optionality-and-runtime-boundaries.md` | Writer service optionality and runtime boundaries |
| `migrations-infrastructure-architecture.md` | Migrations infrastructure architecture |
| `migration-naming-ordering-and-versioning-rules.md` | Migration naming, ordering, and versioning rules |
| `cmd-migrate-and-migration-catalog.md` | cmd/migrate and migration catalog |
| `core-clickhouse-migrations-and-activation-proof.md` | Core ClickHouse migrations and activation proof |
| `core-schema-application-validation-notes.md` | Core schema application validation notes |
| `operational-vs-analytical-query-boundaries.md` | Operational vs analytical query boundaries |
| `historical-query-surface-minimal-extension.md` | Historical query surface minimal extension |

### Codegen

| File | Purpose |
|------|---------|
| `codegen-boundaries-and-governance.md` | Codegen boundaries and governance |
| `codegen-current-usage-boundaries-and-limitations.md` | Codegen current usage, boundaries, and limitations |
| `codegen-path-stabilization-or-freeze-decision.md` | Codegen path stabilization or freeze decision |
| `codegen-specification-and-schema.md` | Codegen specification and schema |
| `codegen-tranche-scoping.md` | Codegen tranche scoping |
| `codegen-validation-and-ci-strategy.md` | Codegen validation and CI strategy |

### Wave B

| File | Purpose |
|------|---------|
| `wave-b-family-01-lifecycle-record.md` | Wave B family 01 lifecycle record |
| `wave-b-family-02-lifecycle-record.md` | Wave B family 02 lifecycle record |
| `wave-b-family-checklist-schema-writer-reader-gateway-tests-runbook.md` | Wave B family checklist |
| `wave-b-family-expansion-pattern-v2.md` | Wave B family expansion pattern v2 |
| `wave-b-iteration-constraints-and-non-goals.md` | Wave B iteration constraints and non-goals |

### Family Lifecycle Records

| File | Purpose |
|------|---------|
| `family-03-lifecycle-record.md` | Family 03 lifecycle record |
| `family-04-lifecycle-record.md` | Family 04 lifecycle record |
| `family-05-lifecycle-record.md` | Family 05 lifecycle record |
| `family-06-lifecycle-record.md` | Family 06 lifecycle record |

### Venue & Real Trading

| File | Purpose |
|------|---------|
| `venue-credentials-and-activation-prerequisites.md` | Venue credentials and activation prerequisites |
| `venue-execution-family-01-contracts.md` | Venue execution family 01 contracts |
| `venue-integrated-execution-design.md` | Venue integrated execution design |
| `venue-integration-activation-gate.md` | Venue integration activation gate |
| `venue-integration-entry-prerequisites.md` | Venue integration entry prerequisites |
| `venue-routing-and-ownership-split.md` | Venue routing and ownership split |
| `real-venue-activation-and-secret-handling.md` | Real venue activation and secret handling |
| `real-venue-activation-gate-ceremony.md` | Real venue activation gate ceremony |
| `real-venue-entry-prerequisites.md` | Real venue entry prerequisites |
| `real-venue-go-no-go-checklist.md` | Real venue go/no-go checklist |
| `real-venue-minimal-operational-scope.md` | Real venue minimal operational scope |
| `real-venue-rollback-and-abort-plan.md` | Real venue rollback and abort plan |
| `minimal-real-venue-adapter-contracts.md` | Minimal real venue adapter contracts |
| `first-guarded-real-venue-step.md` | First guarded real venue step |
| `first-guarded-venue-execution-step.md` | First guarded venue execution step |
| `first-real-smoke-test-findings.md` | First real smoke test findings |
| `first-real-smoke-test-procedure.md` | First real smoke test procedure |
| `first-real-venue-adapter-design.md` | First real venue adapter design |
| `async-fill-and-venue-intake-design.md` | Async fill and venue intake design |
| `pre-venue-fill-reconciliation-model.md` | Pre-venue fill reconciliation model |
| `marketmonkey-adoption-sequencing.md` | Marketmonkey adoption sequencing |
| `marketmonkey-translation-map.md` | Marketmonkey translation map |

### Vertical Slice & Live Pipeline

| File | Purpose |
|------|---------|
| `vertical-slice-01-definition.md` | Vertical slice 01 definition |
| `vertical-slice-01-implementation-notes.md` | Vertical slice 01 implementation notes |
| `live-pipeline-minimal-activation-procedure.md` | Live pipeline minimal activation procedure |
| `live-pipeline-minimal-activation-scope.md` | Live pipeline minimal activation scope |
| `first-slice-acceptance-criteria.md` | First slice acceptance criteria |
| `first-slice-contracts.md` | First slice contracts |
| `first-vertical-slice.md` | First vertical slice |
| `pre-slice-repository-alignment.md` | Pre-slice repository alignment |
| `residual-drift-cleanup-before-vertical-slice.md` | Residual drift cleanup before vertical slice |

### Capability

| File | Purpose |
|------|---------|
| `controlled-capability-01-definition.md` | Controlled capability 01 definition |
| `controlled-capability-01-implementation-notes.md` | Controlled capability 01 implementation notes |

### Operational

| File | Purpose |
|------|---------|
| `current-baseline-cold-start-and-state-limits.md` | Current baseline cold start and state limits |
| `current-baseline-operational-diagnostics.md` | Current baseline operational diagnostics |
| `current-baseline-recovery-and-restart-semantics.md` | Current baseline recovery and restart semantics |
| `current-baseline-runbook.md` | Current baseline runbook |
| `current-capability-baseline-definition.md` | Current capability baseline definition |
| `current-capability-baseline-success-criteria.md` | Current capability baseline success criteria |
| `current-capability-ergonomics-and-governance.md` | Current capability ergonomics and governance |
| `diagnostic-surfaces-and-runtime-signals.md` | Diagnostic surfaces and runtime signals |
| `minimal-operational-baseline.md` | Minimal operational baseline |
| `minimal-observability-foundation.md` | Minimal observability foundation |
| `minimal-live-operation-checks-and-invariants.md` | Minimal live operation checks and invariants |
| `operational-smoke-ci-and-runbook-closure.md` | Operational smoke CI and runbook closure |
| `ci-smoke-analytical-integration.md` | CI smoke analytical integration |
| `error-handling-and-degradation-policy.md` | Error handling and degradation policy |

### Refactoring Phase

| File | Purpose |
|------|---------|
| `refactor-priority-map-high-medium-low.md` | Refactor priority map (high/medium/low) |
| `refactor-tranche-01-changes-rationale-and-impact.md` | Refactor tranche 01 changes, rationale, and impact |
| `refactor-wave-charter-and-entry-freeze.md` | Refactor wave charter and entry freeze |
| `refactor-wave-entry-exit-and-freeze-criteria.md` | Refactor wave entry/exit and freeze criteria |
| `refactor-wave-permitted-vs-prohibited-changes.md` | Refactor wave permitted vs prohibited changes |
| `repository-architecture-census-and-refactor-map.md` | Repository architecture census and refactor map |
| `repository-boundaries-coupling-duplication-and-smells.md` | Repository boundaries, coupling, duplication, and smells |
| `repository-sanitization-audit.md` | Repository sanitization audit |
| `stabilization-responsibility-map.md` | Stabilization responsibility map |
| `stabilization-scope-freeze-and-must-finish-matrix.md` | Stabilization scope freeze and must-finish matrix |
| `stabilization-wave-entry-exit-criteria.md` | Stabilization wave entry/exit criteria |
| `strategic-runtime-and-package-refactor.md` | Strategic runtime and package refactor |
| `pre-refactor-technical-debt-registry-and-cleanup-plan.md` | Pre-refactor technical debt registry and cleanup plan |
| `next-phase-refactor-and-documentation-wave-scope.md` | Next phase refactor and documentation wave scope |

### Consolidation Outputs

| File | Purpose |
|------|---------|
| `documentation-consolidation-and-noise-removal.md` | Documentation consolidation and noise removal |
| `documentation-entropy-archive-delete-consolidate-map.md` | Documentation entropy archive/delete/consolidate map |
| `documentation-changes-archive-delete-consolidate-log.md` | Changes log for archive/delete/consolidate actions |
| `documentation-canonical-map-after-consolidation.md` | This document |
| `deferred-work-registry.md` | Deferred work registry |
| `gains-tradeoffs-and-open-debts-timeline.md` | Gains, tradeoffs, and open debts timeline |
| `next-wave-recommendations-timeline.md` | Next wave recommendations timeline |
| `manual-generated-derived-operational-artifact-model.md` | Manual/generated/derived operational artifact model |

### Hardening

| File | Purpose |
|------|---------|
| `mandatory-hardening-tranche-before-family-03.md` | Mandatory hardening tranche before family 03 |
| `mandatory-hardening-tranche-implementation-notes.md` | Mandatory hardening tranche implementation notes |
| `struct-di-smoke-extraction-helper-renaming-rationale.md` | Struct DI smoke extraction helper renaming rationale |
| `boundary-naming-and-interface-hygiene.md` | Boundary naming and interface hygiene |
| `anti-debt-checklist.md` | Anti-debt checklist |
| `causal-chain-guidelines.md` | Causal chain guidelines |
| `end-to-end-traceability.md` | End-to-end traceability |
| `fail-fast-vs-graceful-degradation-rules.md` | Fail-fast vs graceful degradation rules |

### Timeframe Coverage

| File | Purpose |
|------|---------|
| `timeframe-coverage-01-definition.md` | Timeframe coverage 01 definition |
| `timeframe-coverage-01-implementation-notes.md` | Timeframe coverage 01 implementation notes |

---

## Archive Structure

The `docs/archive/` directory contains 245 files across 16 semantically grouped subdirectories. These are superseded, phase-specific, or redundant documents preserved for audit trail purposes.

| Subdirectory | Files | Content |
|--------------|-------|---------|
| `analytical/` | 23 | Superseded analytical planning, hardening, and wave-A documents |
| `capability/` | 8 | Superseded capability consolidation and baseline documents |
| `cc-02/` | 7 | Superseded CC-02 family expansion intermediate documents |
| `clickhouse-entry/` | 7 | Superseded ClickHouse entry planning and preparation documents |
| `codegen/` | 30 | Superseded codegen planning, drift, equivalence, and golden snapshot documents |
| `deferred-work/` | 14 | Superseded deferred-work fragments consolidated into `deferred-work-registry.md` |
| `domain-lifecycle/` | 14 | Superseded domain lifecycle intermediate documents |
| `families/` | 34 | Superseded family 03-06 planning, selection, and validation documents |
| `gains-tradeoffs/` | 18 | Superseded per-wave gains/tradeoffs documents consolidated into timeline |
| `gates/` | 30 | Superseded gate and readiness review documents |
| `live-pipeline/` | 3 | Superseded live pipeline intermediate documents |
| `next-wave/` | 18 | Superseded per-wave next-wave recommendations consolidated into timeline |
| `superseded/` | 5 | Previously superseded documents from earlier consolidation rounds |
| `timeframe/` | 8 | Superseded timeframe coverage intermediate documents |
| `vertical-slice/` | 6 | Superseded vertical slice intermediate documents |
| `wave-b/` | 20 | Superseded wave-B planning, pattern, and intermediate documents |

---

## Stage Reports

The file `docs/stages/INDEX.md` provides a chronological index of 212 stage reports across 17 phases (pre-numbered foundation through S210+). Each report is an immutable audit trail artifact documenting inputs, outputs, decisions, and findings for its stage.

---

## Notes

- All file paths are relative to `docs/architecture/` unless otherwise noted.
- This map is the canonical reference for what exists after S215 consolidation. Any document not listed here and not in `docs/archive/` should be investigated as potential drift.
- The archive is read-only: no archived document should be modified or referenced as authoritative. Use the active architecture documents listed above.
