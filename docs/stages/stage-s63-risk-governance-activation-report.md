# Stage S63 — Risk Governance Activation Report

> **Date:** 2026-03-18
> **Status:** Complete
> **Predecessor:** S62 (Risk Domain Design)
> **Successor:** S64 (Risk First Slice)

## Executive Summary

Stage S63 adds the `risk` domain to the `raccoon-cli` governance framework. The risk domain is now **actively governed** but **not yet implemented**. This follows the same proven pattern used for signal (S40), decision (S44), and strategy (S54).

The CLI now enforces two categories of checks:
1. **Design integrity:** Verifies S62 architecture documents exist (7 docs).
2. **Premature entry prevention:** Blocks any risk implementation code (adapters, domain files, actors, HTTP, config entries, streams, subjects) from entering the codebase before S64 formally opens.

Additionally, the full post-implementation drift detection infrastructure (adapter/domain/config/contracts checks) is prepared in the codebase with `#[allow(dead_code)]` and will be activated when S64 begins.

## Changes Made

### 1. `tools/raccoon-cli/src/analyzers/drift_detect.rs`

**Constants added:**
- `RISK_DOCS` — 7 required architecture documents
- `RISK_EXPECTED_SUBJECTS` — 2 expected NATS subjects
- `RISK_EXPECTED_DURABLES` — 1 expected durable consumer
- `RISK_EXPECTED_BUCKETS` — 1 expected KV bucket
- `RISK_ADAPTER_FILES` — 5 expected adapter files
- `RISK_DOMAIN_FILES` — 6 expected domain/application files
- `RISK_EVENTS` added to `PROHIBITED_STREAMS`

**Active checks added (Phase 6 in `analyze()`):**
- `check_risk_docs_drift()` — verifies S62 design docs
- `check_risk_premature_implementation()` — blocks premature code entry (adapters, domain, actors, HTTP, config)

**Prepared checks (for S64 activation):**
- `check_risk_adapter_drift()` — adapter file completeness
- `check_risk_domain_drift()` — domain/application/actor/HTTP file completeness
- `check_risk_config_drift()` — symmetric config validation
- `check_risk_contracts_drift()` — subjects, durables, KV buckets in source

### 2. `tools/raccoon-cli/src/analyzers/runtime_bindings/source.rs`

- Added 5 risk adapter file detection fields (`has_risk_publisher`, `has_risk_consumer`, `has_risk_gateway`, `has_risk_kv_store`, `has_risk_registry`)
- Added risk subject classification in `extract_subjects()` (`risk.events.*` → publish, `risk.query.*` → query)

### 3. `tools/raccoon-cli/src/analyzers/runtime_bindings.rs`

- Updated test fixture to include risk adapter fields (all `false` — correct pre-implementation state)

### 4. `tools/raccoon-cli/src/analyzers/coverage_map.rs`

- Added `domain-risk` sensitive area with dimensions: `architecture`, `contracts`, `drift`
- Updated `actors-derive` and `actors-store` descriptions to mention `RISK_EVENTS (S64)`

### 5. Documentation

- `docs/tooling/cli-risk-guardrails.md` — 10 guardrails (8 active, 2 prepared)
- `docs/tooling/cli-risk-drift-rules.md` — 6 drift rules (2 active, 4 prepared) + S64 activation checklist

## Files Changed

| File | Action | Lines |
|------|--------|-------|
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Modified | +350 |
| `tools/raccoon-cli/src/analyzers/runtime_bindings/source.rs` | Modified | +15 |
| `tools/raccoon-cli/src/analyzers/runtime_bindings.rs` | Modified | +5 |
| `tools/raccoon-cli/src/analyzers/coverage_map.rs` | Modified | +8 |
| `docs/tooling/cli-risk-guardrails.md` | Created | new |
| `docs/tooling/cli-risk-drift-rules.md` | Created | new |
| `docs/stages/stage-s63-risk-governance-activation-report.md` | Created | new |

## Governance Gaps That Remain

| Gap | Severity | Mitigation |
|-----|----------|------------|
| **Cannot verify evaluator purity** | Low | Risk evaluator must not perform side effects. Code review enforces this; CLI check is a possible future improvement. |
| **No semantic validation of doc content** | Low | CLI checks docs exist but not that their content matches implementation. Human review required. |
| **Deferred families not tracked** | Low | RF-02 (Drawdown Guard), RF-03 (Correlation Limit), RF-04 (Volatility Scaler) are planned but not yet governance-protected. Will be added when their design stages open. |
| **No adapter/domain test existence check** | Medium | CLI verifies files exist but not that corresponding `_test.go` files exist. This is consistent with signal/decision/strategy governance. |
| **Config value validation** | Low | CLI checks `risk_families` key existence but not that the value matches `["position_exposure"]`. Full validation relies on `ValidatePipeline()` at runtime. |

## Test Results

```
97 passed; 0 failed; 0 ignored
```

All existing tests continue to pass. New risk governance checks are validated by the existing project-level integration tests (drift-detect runs on the actual project, which correctly has no premature risk code).

## Impact on S64 Readiness

S64 (Risk First Slice) can now proceed with confidence:

1. **Design docs are verified** — S62 output is governance-protected
2. **Premature entry is blocked** — no risk code can slip in accidentally
3. **Full drift infrastructure is prepared** — S64 only needs to flip the switch (see activation checklist in `cli-risk-drift-rules.md`)
4. **Coverage map is ready** — `domain-risk` area will automatically trigger appropriate validation
5. **Subject classification is ready** — `risk.events.*` and `risk.query.*` subjects will be correctly classified when they appear

The estimated effort to activate post-implementation checks in S64: **~15 minutes** (constant updates + remove `#[allow(dead_code)]`).

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| CLI detects drift relevant to risk | **Met** — docs-drift + premature-entry checks active |
| Families, subjects, buckets, docs enter governance | **Met** — all defined in constants, docs checked |
| Stage reduces risk of design-implementation divergence | **Met** — premature entry guard is the primary mechanism |
| Solution is useful and proportional | **Met** — 2 active checks, 4 prepared; no decorative inflation |
| Domain is more ready for safe implementation in S64 | **Met** — full activation checklist documented |
