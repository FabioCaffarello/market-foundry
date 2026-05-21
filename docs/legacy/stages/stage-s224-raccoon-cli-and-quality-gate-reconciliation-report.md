# Stage S224 - Raccoon-CLI and Quality-Gate Reconciliation Report

## Executive Summary

S224 reconciled the governance tooling with the current `market-foundry` architecture. The work removed stale assumptions introduced by the pre-S218/S219/S220 layout while preserving the substantive guard rails. The result is that `quality-gate` now validates the real repository topology instead of demanding deleted files, flattened NATS adapters, or consolidated-away documents.

## Objective

The objective was not functional expansion. The objective was to realign governance tooling with the actual repository surface so that the next charter starts from a coherent base.

## Reconciliation Applied

### Tooling updates

- Updated registry discovery in `raccoon-cli` to support both legacy `*_registry.go` files and current `*/registry.go` files.
- Updated consumer extraction so analyzers recognize `natskit.NewConsumerSpec(...)` in addition to literal `ConsumerSpec{...}` blocks.
- Repointed adapter presence checks from flat `internal/adapters/nats/*.go` files to the current sub-package layout.
- Replaced stale store consumer actor expectations with validation of `generic_consumer_actor.go` plus domain wiring in `store_supervisor.go`.
- Rebased domain doc completeness checks on the canonical post-consolidation document set.
- Updated help and remediation output to reference the current paths.

### Operational documentation updates

- Updated `tools/raccoon-cli/README.md`
- Updated `docs/tooling/cli-architecture-guardrails.md`
- Updated `docs/tooling/cli-topology-audit.md`
- Added the S224 reconciliation docs under `docs/architecture/`

## Files Changed

### Tooling

- `tools/raccoon-cli/src/analyzers/contracts/registry.rs`
- `tools/raccoon-cli/src/analyzers/runtime_bindings/source.rs`
- `tools/raccoon-cli/src/analyzers/topology/source.rs`
- `tools/raccoon-cli/src/analyzers/topology.rs`
- `tools/raccoon-cli/src/analyzers/drift_detect.rs`

### Operational docs

- `tools/raccoon-cli/README.md`
- `docs/tooling/cli-architecture-guardrails.md`
- `docs/tooling/cli-topology-audit.md`

### New reconciliation docs

- `docs/architecture/raccoon-cli-and-quality-gate-reconciliation.md`
- `docs/architecture/governance-tooling-before-and-after-restructure.md`
- `docs/stages/stage-s224-raccoon-cli-and-quality-gate-reconciliation-report.md`

## Before and After of Tooling Assumptions

### Before

- Expected flat NATS adapter files directly under `internal/adapters/nats/`
- Expected deleted store consumer actor wrappers
- Expected removed readiness-style documents as active governance artifacts
- Missed valid durable consumers declared through `natskit.NewConsumerSpec(...)`

### After

- Recognizes the current domain sub-package adapter layout
- Validates store-side consumption through generic consumer infrastructure
- Uses the surviving canonical docs set as the documentation proof surface
- Recognizes both explicit and factory-based consumer declarations

## Validation

The following validations were run successfully:

- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml analyzers::drift_detect::tests::full_report_passes_when_aligned -- --exact`
- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml analyzers::runtime_bindings::source::tests::extract_durables_from_factory_calls -- --exact`
- `make quality-gate`

Observed result:

- `doctor` PASS
- `topology-doctor` PASS
- `contract-audit` PASS
- `runtime-bindings` PASS
- `arch-guard` PASS
- `drift-detect` PASS

## Limits and Trade-Offs

1. S224 reconciles the tooling with the current structure; it does not redesign the governance program.
2. Domain-specific tooling docs outside the minimum operational surface may still need consolidation in S225.
3. No new guard rails were introduced. The work focused on restoring correctness and relevance of the existing ones.

## Recommended Preparation for S225

1. Use the now-clean `quality-gate` output as the baseline for active documentation reconciliation.
2. Consolidate any remaining domain-specific tooling docs that still narrate the pre-restructure surface.
3. Keep future tooling changes tied to canonical architecture docs so the proof surface moves with the codebase rather than behind it.
