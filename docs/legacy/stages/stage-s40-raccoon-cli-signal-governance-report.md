# Stage S40 — raccoon-cli Signal Governance

**Status:** Complete
**Date:** 2026-03-17
**Predecessor:** S39 (adapter test coverage sweep)
**Objective:** Transform `signal` from an implemented-but-ungoverned domain into a fully audited part of the market-foundry architecture, enforced by `raccoon-cli`.

---

## Executive Summary

S38 (decision readiness review) identified raccoon-cli's lack of signal governance as a concrete blocker for decision domain entry. This stage closes that gap by adding signal-aware checks to three CLI analyzers (`drift-detect`, `runtime-bindings`, `coverage-map`), covering subjects, durables, KV buckets, adapter files, domain files, actors, HTTP layer, config symmetry, and architecture docs.

After S40, `make check` detects drift in signal contracts, config, docs, and source — the same governance that already protects observation and evidence.

---

## Rules Added to CLI

### drift-detect (5 new checks)

| Check | What it validates |
|-------|-------------------|
| `signal-docs-drift` | 8 required architecture docs exist |
| `signal-adapter-drift` | 5 NATS adapter files exist |
| `signal-domain-drift` | Domain, application, actor, and HTTP layer files exist |
| `signal-config-drift` | `signal_families` appears symmetrically in derive and store configs |
| `signal-contracts-drift` | Signal subjects, durable consumers, and KV bucket names exist in source |

### topology-doctor (1 replaced check)

| Change | Existing check |
|--------|---------------|
| Premature signal entry guard → signal pipeline continuity (SIGNAL_EVENTS ↔ store-signal-rsi) | pipeline-continuity |

### runtime-bindings (4 additions to existing checks)

| Addition | Existing check |
|----------|---------------|
| SIGNAL_EVENTS stream | stream-ownership |
| `store-signal-rsi` durable → SIGNAL_EVENTS | consumer-binding |
| `signal.query.rsi.latest` subject | query-routing |
| 5 signal adapter files (publisher, consumer, gateway, kv_store, registry) | adapter-files |

### coverage-map (1 new sensitive area)

| Area | Dimensions |
|------|-----------|
| `domain-signal` (`internal/domain/signal/`) | architecture, contracts, drift |

---

## Files Changed

### Rust source (tools/raccoon-cli/src/)

| File | Change |
|------|--------|
| `analyzers/drift_detect.rs` | Added SIGNAL_EXPECTED_SUBJECTS, SIGNAL_EXPECTED_DURABLES, SIGNAL_EXPECTED_BUCKETS constants; added signal-family-01-contracts.md to SIGNAL_DOCS; added `check_signal_contracts_drift()` function with subject/durable/KV validation; added `scan_dir_for_string()` utility; updated test fixtures to include SIGNAL_EVENTS stream, signal subjects, and store-signal-rsi durable |
| `analyzers/runtime_bindings.rs` | Added SIGNAL_EVENTS to EXPECTED_STREAMS; added store-signal-rsi to EXPECTED_DURABLES; added signal.query.rsi.latest to EXPECTED_QUERY_SUBJECTS; added 5 signal adapter file checks to check_adapter_files(); updated test fixtures |
| `analyzers/runtime_bindings/source.rs` | Added has_signal_publisher, has_signal_consumer, has_signal_gateway, has_signal_kv_store, has_signal_registry fields to RuntimeBindingSource; added signal adapter file detection in scan_runtime_bindings(); added signal.events.* and signal.query.* classification in extract_subjects(); added 3 new tests |
| `analyzers/topology.rs` | Replaced premature signal entry guard with signal pipeline continuity check (SIGNAL_EVENTS ↔ store-signal-rsi); updated test fixtures to include SIGNAL_EVENTS stream, signal subjects, and store-signal-rsi durable |
| `analyzers/coverage_map.rs` | Added domain-signal sensitive area with architecture, contracts, drift dimensions; updated actors-derive and actors-store descriptions to mention SIGNAL_EVENTS; added domain-signal test |

### Documentation (docs/)

| File | Change |
|------|--------|
| `docs/tooling/cli-signal-guardrails.md` | **New** — 10 guardrails covering stream, durable, query, adapters, domain, actors, config, docs, KV, and coverage |
| `docs/tooling/cli-signal-drift-rules.md` | **New** — 5 drift rules (SD-1 through SD-5) with severity, purpose, and known limitations |
| `docs/stages/stage-s40-raccoon-cli-signal-governance-report.md` | **New** — this report |

---

## Test Results

- **916 unit tests** pass (including new signal-specific tests)
- **80 integration tests** pass
- **97 doc tests** pass
- **0 failures**

New tests added:
- `extract_subjects_classifies_signal_events` — verifies signal.events.* → publish_subjects
- `extract_subjects_classifies_signal_query` — verifies signal.query.* → query_subjects
- `scan_runtime_bindings_detects_signal_adapter_files` — verifies all 5 signal adapter flags
- `relevant_checks_for_domain_signal` — verifies domain-signal area triggers correct dimensions

---

## Governance Gaps That Remain

| Gap | Severity | Mitigation |
|-----|----------|------------|
| Signal sampler correctness (RSI logic) | Medium | Covered by Go unit tests, not static analysis |
| Actor wiring correctness | Medium | Smoke tests validate end-to-end; CLI cannot inspect Go actor trees |
| KV monotonicity guard logic | Low | Covered by signal_kv_store_test.go (S39) |
| Multi-signal family constants | Low | Adding MACD requires manual constant updates; documented in cli-signal-drift-rules.md |
| Signal-to-signal derivation | N/A | Not yet implemented; deferred by design |
| Cross-validation of signal_families config values vs adapter existence | Low | CLI checks substring presence, not parsed JSON values |
| Actor-scope tests | Systemic | Zero actor tests across all scopes; not signal-specific |

---

## Impact on Readiness for S41/S42

### S41 (multi-symbol verification)
- **Positive:** Signal pipeline is now governable — any drift introduced during multi-symbol scaling will be caught by `make check`.
- **Neutral:** S41 scope is runtime validation, not static governance.

### S42/S43 (decision domain design + first slice)
- **Positive — blocker resolved:** S38 listed "raccoon-cli has no signal governance rules" as a concrete blocker. This stage closes that gap.
- **Remaining blockers:** S41 (multi-symbol) must still pass before decision entry.
- **Template established:** The signal governance pattern (constants → checks → docs → tests) serves as the template for adding decision governance when that domain is implemented.

---

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| CLI detects drift in signal contracts | Done — subjects, durables, KV buckets, adapter files |
| Families, subjects, buckets, docs enter governance | Done — all checked in drift-detect and runtime-bindings |
| Reduces risk of architecture/implementation divergence | Done — any removal or rename of signal artifacts triggers errors |
| Solution is useful and proportional | Done — no decorative checks; all findings have actionable why/help |
| Signal domain more ready for decision consumption | Done — S38 blocker resolved |
