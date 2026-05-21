# Stage S44: Raccoon-CLI Decision Domain Governance — Report

**Status:** Complete
**Date:** 2025-03-17
**Objective:** Strengthen raccoon-cli to audit and protect the `decision` domain as part of the architecture governance mesh.

## Executive Summary

S44 extends the raccoon-cli governance framework to cover the decision domain with the same rigor applied to evidence and signal. Five new drift detection checks (DD-1 through DD-5), expanded runtime-bindings validation, and a new coverage-map sensitive area bring decision under automated architectural protection. The decision domain is now governed across docs, adapters, domain files, actors, HTTP interfaces, configuration symmetry, subjects, durables, and KV buckets — reducing the risk of architecture-implementation divergence before any future strategy layer.

## Rules Added to the CLI

### drift-detect (5 new checks)

| Rule | Check Name | What It Validates |
|------|-----------|-------------------|
| DD-1 | `decision-docs-drift` | 8 required architecture docs exist |
| DD-2 | `decision-adapter-drift` | 5 NATS adapter files exist |
| DD-3 | `decision-domain-drift` | 6 domain/app files + 4 actors + 2 HTTP files exist |
| DD-4 | `decision-config-drift` | decision_families symmetry between derive.jsonc and store.jsonc |
| DD-5 | `decision-contracts-drift` | 2 subjects + 1 durable + 1 KV bucket in source |

### runtime-bindings (expanded)

| Addition | What Changed |
|----------|-------------|
| `DECISION_EVENTS` stream | Added to expected streams (now 5 canonical streams) |
| `store-decision-rsi-oversold` durable | Added to expected durable consumers |
| `decision.query.rsi_oversold.latest` | Added to expected query subjects |
| Decision adapter file checks | 5 new adapter file presence checks |
| Decision subject extraction | Source scanner now detects `decision.events.*` and `decision.query.*` |

### coverage-map (expanded)

| Addition | What Changed |
|----------|-------------|
| `domain-decision` sensitive area | `internal/domain/decision/` monitored with dimensions: architecture, contracts, drift |
| Actor scope descriptions | Updated to mention DECISION_EVENTS |

## Files Changed

### Rust Source (raccoon-cli)

| File | Changes |
|------|---------|
| `src/analyzers/drift_detect.rs` | +60 lines constants (DECISION_DOCS/SUBJECTS/DURABLES/BUCKETS/ADAPTER_FILES/DOMAIN_FILES), +5 check functions (~220 lines), +DECISION_EVENTS to CANONICAL_STREAMS, updated test helpers |
| `src/analyzers/runtime_bindings.rs` | +DECISION_EVENTS stream, +store-decision-rsi-oversold durable, +decision.query subject, +5 adapter file checks, updated test helpers |
| `src/analyzers/runtime_bindings/source.rs` | +5 `has_decision_*` fields to RuntimeBindingSource, +5 adapter file scans, +decision subject extraction in extract_subjects |
| `src/analyzers/coverage_map.rs` | +domain-decision sensitive area, updated actors-derive and actors-store descriptions |

### Documentation

| File | Purpose |
|------|---------|
| `docs/tooling/cli-decision-drift-rules.md` | Defines DD-1 through DD-5 rules |
| `docs/tooling/cli-decision-guardrails.md` | Documents DG-1 through DG-10 guardrails |
| `docs/stages/stage-s44-raccoon-cli-decision-governance-report.md` | This report |

## Governance Gaps That Remain

These gaps are documented honestly; they represent areas where CLI protection is not yet feasible or proportional:

| Gap | Reason | Risk Level |
|-----|--------|------------|
| Domain boundary invariants (DBI-1 through DBI-9) | Requires cross-package import analysis beyond current AST depth | Medium — manual review still needed |
| Evaluator purity (no I/O) | Cannot statically verify absence of side effects in Go | Low — covered by design pattern |
| Multi-family expansion | Only `rsi_oversold` governed; adding families requires constant updates | Low — new families follow established pattern |
| KV bucket configuration drift | CLI doesn't verify retention/maxsize match architecture docs | Low — operational, not structural |
| Cross-domain message primitive-only constraint | Cannot verify actor messages carry only primitive types | Medium — requires deeper AST analysis |
| Decision history projections | Not yet implemented; no governance needed | None — deferred by design |
| Adapter unit test presence | CLI checks file existence, not test coverage | Low — tracked separately |

## Impact on Readiness

### For S45+ (Strategy domain prerequisites)
- Decision is now under the same governance level as signal and evidence
- Any structural drift in decision will be caught by `raccoon-cli quality-gate --profile fast`
- The merge freeze safety net now extends to decision

### For future family expansion (MACD crossover, confluence)
- Pattern is established: add constants to `DECISION_EXPECTED_*` arrays, extend checks
- No structural changes needed to the checking framework

### For CI integration
- All new checks run in the `fast` and `ci` profiles (no infrastructure required)
- Zero additional runtime cost — checks are file-existence and string-scan based

## Test Results

- **916 unit tests** — all pass
- **80 integration tests** — all pass
- **97 validation matrix tests** — all pass
- **Total: 1093 tests, 0 failures**

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| CLI detects drift in decision domain | Yes — DD-1 through DD-5 |
| Families, subjects, buckets, docs governed | Yes — all checked |
| Reduces architecture-implementation divergence risk | Yes — automated on every quality-gate run |
| Proportional, no bureaucratic overhead | Yes — follows established signal pattern |
| Domain better prepared as base for strategy | Yes — governed at same level as signal/evidence |
