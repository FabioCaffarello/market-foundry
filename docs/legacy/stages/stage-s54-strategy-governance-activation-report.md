# S54 — Strategy Governance Activation Report

**Stage:** S54 — Strategy Governance Activation
**Status:** Complete
**Date:** 2026-03-18
**Prerequisite:** S53 (Strategy Domain Design)
**Successor:** S55 (Strategy First Slice Implementation)

## Objective

Add `strategy` domain to the raccoon-cli governance framework so that the domain is audited and protected against drift before any implementation code is written.

## Deliverables

### 1. Raccoon-CLI Changes

#### drift_detect.rs

- Added `STRATEGY_EVENTS` to `CANONICAL_STREAMS` (6 streams total)
- Added strategy governance constants:
  - `STRATEGY_DOCS` — 8 required architecture documents
  - `STRATEGY_EXPECTED_SUBJECTS` — 2 NATS subjects (event + query)
  - `STRATEGY_EXPECTED_DURABLES` — 1 durable consumer
  - `STRATEGY_EXPECTED_BUCKETS` — 1 KV bucket
  - `STRATEGY_ADAPTER_FILES` — 5 adapter files
  - `STRATEGY_DOMAIN_FILES` — 6 domain/application files
- Added Phase 5 checks in `analyze()`:
  - `check_strategy_docs_drift` — 8 architecture docs
  - `check_strategy_adapter_drift` — 5 NATS adapter files
  - `check_strategy_domain_drift` — domain + 4 actors + 2 HTTP files
  - `check_strategy_config_drift` — `strategy_families` symmetry between derive/store
  - `check_strategy_contracts_drift` — subjects, durables, KV buckets
- Updated test helper `make_source_topology` with `STRATEGY_EVENTS` stream

#### runtime_bindings.rs

- Added `STRATEGY_EVENTS` to `EXPECTED_STREAMS`
- Added `store-strategy-mean-reversion-entry` to `EXPECTED_DURABLES`
- Added `strategy.query.mean_reversion_entry.latest` to `EXPECTED_QUERY_SUBJECTS`
- Added 5 strategy adapter file checks (`strategy_publisher`, `strategy_consumer`, `strategy_gateway`, `strategy_kv_store`, `strategy_registry`)
- Added `strategy_families` cross-config consistency check (derive ↔ store alignment)
- Updated test helper `make_source()` with strategy streams, durables, and query subjects

#### runtime_bindings/source.rs

- Added `has_strategy_publisher`, `has_strategy_consumer`, `has_strategy_gateway`, `has_strategy_kv_store`, `has_strategy_registry` boolean fields
- Added strategy adapter file detection in `scan_runtime_bindings()`
- Added `strategy.events.*` and `strategy.query.*` subject classification in `extract_subjects()`
- Updated doc comment to include STRATEGY_EVENTS in the pipeline description

#### runtime_bindings/configs.rs

- Added `strategy_families: Vec<String>` to `ServiceConfig`
- Added extraction of `pipeline.strategy_families` from deploy configs

#### topology.rs

- Added strategy pipeline continuity check: `STRATEGY_EVENTS` ↔ `store-strategy-mean-reversion-entry` durable
- Updated test helper `make_source_topology()` with DECISION_EVENTS, STRATEGY_EVENTS, and their durables/subjects

#### coverage_map.rs

- Added `domain-strategy` sensitive area with patterns `internal/domain/strategy/` and required dimensions `architecture`, `contracts`, `drift`
- Updated actors-derive and actors-store descriptions to include STRATEGY_EVENTS

### 2. Documentation Created

| File | Purpose |
|------|---------|
| `docs/tooling/cli-strategy-guardrails.md` | 10 guardrails (SG-1 through SG-10) covering stream, adapter, domain, config, docs, bucket, and coverage |
| `docs/tooling/cli-strategy-drift-rules.md` | 5 drift rules (STD-1 through STD-5) with artifact matrices, severity, and known limitations |

### 3. Documentation Updated

| File | Change |
|------|--------|
| `docs/tooling/cli-drift-rules.md` | Added STRATEGY_EVENTS to canonical streams, added strategy durable example, added Rules 17–21 reference |

## Governance Matrix

| Artifact | Check | Severity |
|----------|-------|----------|
| 8 architecture docs | strategy-docs-drift | ERROR |
| 5 NATS adapter files | strategy-adapter-drift | ERROR |
| 6 domain/app files | strategy-domain-drift | ERROR |
| 4 actor files | strategy-domain-drift | ERROR |
| 2 HTTP files | strategy-domain-drift | ERROR |
| strategy_families config key | strategy-config-drift | ERROR (asymmetry) / WARNING (absent) |
| 2 NATS subjects | strategy-contracts-drift | ERROR |
| 1 durable consumer | strategy-contracts-drift | ERROR |
| 1 KV bucket | strategy-contracts-drift | ERROR |
| STRATEGY_EVENTS stream | runtime-bindings stream-ownership | ERROR |
| Pipeline continuity | topology-doctor pipeline-continuity | ERROR |
| Cross-config alignment | runtime-bindings cross-config-family | ERROR |

## Governance Gaps That Remain

1. **Domain boundary invariants (SBI-1 through SBI-10)** — Cannot enforce that strategy does not import decision/signal/evidence domains. Requires deeper AST analysis or `arch-guard` extension.
2. **Resolver purity** — Cannot verify that `mean_reversion_entry_resolver.go` is free of I/O side effects.
3. **Dependency chain enforcement** — Cannot enforce that `strategy_families: ["mean_reversion_entry"]` requires `decision_families: ["rsi_oversold"]` in the same config. The S53 design documents this as an operator responsibility with a warning.
4. **Direction semantics validation** — Cannot verify that strategy resolvers produce valid Direction values (long/short/flat only).
5. **KV bucket configuration** — Cannot verify that STRATEGY_MEAN_REVERSION_ENTRY_LATEST has correct retention (72h), max size (2 GB), or storage backend (file).
6. **Multi-decision strategies** — Only single-decision families are governed. STF-03 (Confluence Entry) governance is deferred.
7. **Cross-domain message contracts** — Cannot verify that strategy resolvers receive decision data via actor messages, not direct imports.

## Test Results

- All 97 raccoon-cli tests pass (including updated test helpers for strategy streams, durables, and query subjects)
- Build compiles cleanly (no new warnings introduced)

## Impact on S55/S56

### S55 Readiness

The CLI now provides a **living checklist** for strategy implementation. Running `raccoon-cli drift-detect` will report ~30 errors for missing strategy artifacts. Each error maps to a specific file that must be created during S55:

1. Domain entity and events (2 files)
2. Resolver logic (1 file)
3. Client contracts and use cases (3 files)
4. NATS adapters (5 files)
5. Actors (4 files)
6. HTTP handlers and routes (2 files)
7. Config key activation (2 config files)

### S56 Readiness

When S55 completes, the drift-detect errors will resolve to INFO findings. The strategy domain will be under active governance identical to signal and decision domains. Adding new strategy families (STF-02, STF-03) requires only extending the constant arrays in `drift_detect.rs` and `runtime_bindings.rs`.

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| CLI detects drift in strategy | Yes — 5 drift checks, stream/durable/bucket validation |
| Families, subjects, buckets enter governance | Yes — all constants defined and checked |
| Reduces divergence risk | Yes — living checklist prevents implementation without governance |
| Useful and proportional | Yes — follows established pattern, no new abstractions |
| Domain ready for safe implementation in S55/S56 | Yes — governance gate active before code |
