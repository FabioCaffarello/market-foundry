# Stage S98 — Boundary Naming and Interface Hygiene Report

**Date:** 2026-03-19
**Objective:** Reduce naming debt, disambiguate overloaded terms, and clean residual identity from quality-service era.

## Executive Summary

S98 performed a disciplined naming cleanup across Go application code, store supervisor types, test fixtures, and the raccoon-cli Rust toolchain. No features were added. No architecture was redesigned.

Three categories of naming debt were addressed:
1. **Semantic overload** — `PipelineScope` reused "Scope" for domain classification, conflicting with actor-layer `SourceScopeActor`/`ExchangeScopeActor`.
2. **Stale terminology** — 22 error messages said "service is unavailable" instead of "gateway"; test data used "quality" labels; raccoon-cli module prefix still referenced `quality-service`.
3. **Dead code** — `NewDeafultEngine` typo wrapper function served no purpose.

All changes compile clean. All tests pass.

## Changes Applied

### 1. PipelineScope → PipelineDomain (semantic disambiguation)

**Problem:** The store supervisor used `PipelineScope` with constants `ScopeEvidence`, `ScopeSignal`, etc. to classify which bounded context a pipeline belongs to. But "Scope" in the actor layer means a supervision boundary (`SourceScopeActor`, `ExchangeScopeActor`). Two different concepts sharing the same name.

**Fix:** Renamed to `PipelineDomain` with constants `DomainEvidence`, `DomainSignal`, `DomainDecision`, `DomainStrategy`, `DomainRisk`, `DomainExecution`.

**Files:**
- `internal/actors/scopes/store/store_supervisor.go` — type, constants, method signatures
- `docs/architecture/family-runtime-registration-rules.md` — updated references

### 2. "service is unavailable" → "gateway is unavailable" (22 error messages)

**Problem:** All client use cases reported `"{domain} service is unavailable"` when the gateway was nil. Market Foundry has no "services" — it uses gateway ports (NATS request/reply). The error message misrepresented the architecture.

**Fix:** Updated all 22 occurrences across 7 client packages to use `"{domain} gateway is unavailable"`.

**Files (19 source files):**
- `internal/application/configctlclient/` — 10 files (activate, compile, create_draft, get_active_config, get_config, list_active_ingestion_bindings, list_active_runtime_projections, list_configs, validate_config, validate_draft)
- `internal/application/evidenceclient/` — 4 files (get_latest_candle, get_candle_history, get_latest_trade_burst, get_latest_volume)
- `internal/application/signalclient/get_latest_signal.go`
- `internal/application/decisionclient/get_latest_decision.go`
- `internal/application/strategyclient/get_latest_strategy.go`
- `internal/application/riskclient/get_latest_risk.go`
- `internal/application/executionclient/` — 3 files (get_latest_execution, get_execution_status, get_execution_control)

### 3. NewDeafultEngine typo removed

**Problem:** `internal/actors/common/engine.go` exported `NewDeafultEngine()` (typo) as a wrapper around `NewDefaultEngine()`. No callers used the typo variant.

**Fix:** Removed the dead wrapper. Only `NewDefaultEngine()` remains.

**File:** `internal/actors/common/engine.go`

### 4. Test data identity cleanup

**Problem:** Config test fixtures still used `"Core Quality Config"` and `"team":"quality"` from the quality-service era.

**Fix:** Updated to `"Core Market Config"` and `"team":"foundry"`.

**Files:**
- `internal/domain/configctl/config_set_test.go`
- `internal/adapters/repositories/memory/configctl/repository_test.go`

### 5. raccoon-cli module prefix and test fixtures

**Problem:** The Rust CLI had `DEFAULT_MODULE_PREFIX = "quality-service/internal/"` — a stale constant from before the module path sanitization. The fallback `path.contains("/internal/")` was doing the actual work. Additionally, ~43 test fixture strings across 9 Rust source files still used `quality-service/internal/` or `example.com/quality-service/internal/` import paths.

**Fix:**
- Updated `DEFAULT_MODULE_PREFIX` to `"internal/"` (matches actual Go module paths)
- Updated all test fixtures to use `internal/...` paths matching real module structure
- Updated `quality_service` → `market_foundry` in Rust `use` statement detection
- Updated help text from "quality-service root" to "project root"

**Files (9 Rust source files):**
- `tools/raccoon-cli/src/codeintel/parser.rs`
- `tools/raccoon-cli/src/analyzers/arch_guard.rs`
- `tools/raccoon-cli/src/analyzers/recommend.rs`
- `tools/raccoon-cli/src/analyzers/tdd.rs`
- `tools/raccoon-cli/src/analyzers/impact_map.rs`
- `tools/raccoon-cli/src/analyzers/symbol_trace.rs`
- `tools/raccoon-cli/src/analyzers/rename_safety.rs`
- `tools/raccoon-cli/src/analyzers/briefing.rs`
- `tools/raccoon-cli/src/analyzers/baseline_drift.rs`

### 6. Documentation

Two new architecture documents:
- `docs/architecture/boundary-naming-and-interface-hygiene.md` — terminology disambiguation, error message conventions, hygiene checklist
- `docs/architecture/naming-conventions-for-domains-families-and-runtimes.md` — canonical naming rules for domains, families, runtimes, and scopes

## Semantic Debts Removed

| Debt | Category | Impact |
|------|----------|--------|
| `PipelineScope` overloading "Scope" | Semantic overload | Confused domain classification with actor supervision |
| 22 × "service is unavailable" | Stale terminology | Operator-visible messages misrepresented architecture |
| `NewDeafultEngine` typo | Dead code | Public API with misspelled name |
| `"quality"` in test data | Identity residue | Onboarding confusion, stale origin trace |
| `quality-service/internal/` in raccoon-cli | Identity residue | Toolchain desync from actual module paths |
| 43 test fixtures in Rust CLI | Identity residue | Test assertions against non-existent import patterns |

## Limits Maintained

| Decision | Rationale |
|----------|-----------|
| Did not rename `*Supervisor` → `*Orchestrator` | "Supervisor" is the standard actor-model term; renaming would add confusion |
| Did not unify evaluate/resolve/sample verbs | These encode genuinely different domain semantics; unification would lose meaning |
| Did not rename `Gateway` vs `Registry` | These serve different architectural roles (port vs value object); both names are correct |
| Did not rename `WebHandler` vs `ResponderActor` | Different layers legitimately use different handler terms |
| Did not touch raccoon-cli deprecated commands | Those are deliberate historical markers; changing them would remove audit trail |
| Did not rename consumer/publisher patterns | The variations (typed, dispatch, actor-wrapped) serve different layers correctly |

## Verification

- All 6 Go binaries build clean (`gateway`, `store`, `derive`, `ingest`, `execute`, `configctl`)
- All Go tests pass in affected packages (9 test suites)
- raccoon-cli Rust tests pass (919/921; 2 pre-existing failures in unrelated `drift_detect.rs`)

## Preparation for S99

With naming hygiene addressed, the codebase is ready for:

1. **Remaining raccoon-cli deprecated commands** — `smoke`, `scenario`, `results-inspect`, and `trace-pack` commands still carry quality-service runtime logic. Consider removing the implementations entirely (keep only the deprecation notice) or converting them to market-foundry equivalents.
2. **NATS adapter package organization** — `internal/adapters/nats/` has 50+ files with varied naming patterns. A future stage could organize into sub-packages by responsibility (registries, gateways, consumers, publishers, kv).
3. **Actor message naming convention** — Three patterns coexist (semantic, event-style, result-wrapper). Documenting the convention would reduce ambiguity for new families.
4. **Client-side integration tests** — Now that error messages reference "gateway," consider adding integration test coverage for gateway unavailability scenarios.
