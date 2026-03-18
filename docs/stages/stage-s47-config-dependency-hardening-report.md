# Stage S47 — Config Dependency Hardening Report

**Status:** Complete
**Objective:** Harden validation of configurational dependencies between
evidence, signal, and decision layers to prevent incoherent or incomplete
configurations before the strategy domain is introduced.

## Executive Summary

S47 closes the gap where cross-layer and cross-service family dependencies
relied on implicit convention. Two concrete validation surfaces were added:

1. **Go startup validation** (`PipelineConfig.ValidatePipeline()`) — rejects
   unknown family names and enforces cross-layer dependency rules at service
   boot time.
2. **Raccoon-CLI static analysis** (`check_cross_config_family_consistency`) —
   detects family mismatches between derive and store deploy configs at
   CI/pre-deploy time.

Both are proportional: no DSL, no generic framework, no runtime overhead.
The dependency graph is declared as simple Go maps and validated with
straightforward iteration.

## Dependency Rules Hardened

### 1. Known Family Registry (typo protection)

Each layer now has a closed set of recognized names:

- Evidence: `candle`, `tradeburst`, `volume`
- Signal: `rsi`
- Decision: `rsi_oversold`

Unknown names in config are rejected at startup with a clear error message.

### 2. Cross-Layer Dependency Validation

| Rule | Example |
|------|---------|
| Signal → Evidence | `rsi` requires `candle` |
| Decision → Signal | `rsi_oversold` requires `rsi` |

If a decision family is enabled without its required signal (or a signal
without its required evidence), the config is rejected at startup.

### 3. Cross-Service Consistency (raccoon-cli)

The `runtime-bindings` analyzer now compares derive and store configs:

- Evidence families: flagged when explicit lists diverge
- Signal families: flagged when one side enables a family the other does not
- Decision families: same as signal

## Files Changed

### Go (runtime validation)

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added `knownEvidenceFamilies`, `knownSignalFamilies`, `knownDecisionFamilies` registries; `signalDependsOnEvidence`, `decisionDependsOnSignal` dependency maps; `ValidatePipeline()` method; integrated into `Validate()` |
| `internal/shared/settings/settings_test.go` | 9 new tests covering: known/unknown family rejection, cross-layer dependency enforcement, backward-compatible defaults |

### Rust (static analysis)

| File | Change |
|------|--------|
| `tools/raccoon-cli/src/analyzers/runtime_bindings/configs.rs` | Extended `ServiceConfig` with `families`, `signal_families`, `decision_families` fields; added `extract_string_array` helper; 1 new test |
| `tools/raccoon-cli/src/analyzers/runtime_bindings.rs` | Added `check_cross_config_family_consistency` check (Check 8); 5 new tests |

### Documentation

| File | Change |
|------|--------|
| `docs/architecture/family-config-dependency-rules.md` | New — canonical reference for dependency rules, known families, validation surfaces |
| `docs/stages/stage-s47-config-dependency-hardening-report.md` | New — this report |

## Test Results

- Go: 19/19 pass (`go test ./internal/shared/settings/`)
- Rust: 51/51 pass (`cargo test analyzers::runtime_bindings`)

## Gaps That Remain

| Gap | Risk | Mitigation path |
|-----|------|-----------------|
| Gateway has no family config — discovers families from KV at runtime | Low: if store doesn't project a family, gateway returns 404 naturally | Could add optional `pipeline` to gateway config in a future stage |
| No runtime health check that a projected family has live consumers | Medium: store spawns consumers but if NATS consumer group is stale, events accumulate | Healthz tracker already monitors consumer liveness per pipeline |
| Dependency rules are Go maps, not a config file | Low: intentional — keeps rules close to code, avoids external DSL | Revisit only if family count exceeds ~20 |
| No validation that timeframes are consistent across derive/store | Low: store projects whatever events arrive regardless of timeframe | Could add raccoon-cli check if needed |

## Impact on Readiness

### S48/S49 (Strategy Domain)

- **Positive:** Strategy families can follow the same pattern — add to
  `knownDecisionFamilies` (or new `knownStrategyFamilies`), declare
  dependencies, get validation for free.
- **Positive:** Cross-service consistency check already handles N layers;
  adding strategy to configs is automatically covered.
- **Positive:** Typo protection prevents the class of silent-failure bugs
  that become harder to diagnose as domain count grows.

### General

- Configuration errors are now caught **at startup** (Go) or **before deploy**
  (raccoon-cli) instead of manifesting as silent runtime degradation.
- The validation is additive and backward-compatible: empty `pipeline` blocks
  remain valid.

## Acceptance Criteria Checklist

- [x] Dependencies between families/layers are more explicit and safer
- [x] Incoherent configurations are detected earlier (startup + CI)
- [x] Architecture is less dependent on implicit convention
- [x] Solution is proportional — no framework, no DSL, no inflation
- [x] Reduces a real structural risk before strategy domain
- [x] Does not implement strategy
- [x] Does not inflate controlplane with excessive logic
- [x] Gaps are documented
