# Behavioral Full-Stack Smoke Closure

**Stage:** S255
**Status:** Closed
**Debt Addressed:** OD-BW1 (full-stack behavioral smoke)

## Purpose

This document records the closure of OD-BW1, the most important operational debt from
the BEHAVIORAL-WAVE-1 gate (S254). OD-BW1 required proof that behavioral semantics
survive the complete system round-trip, not just in-process actor chain tests.

## Problem Statement

The BEHAVIORAL-WAVE-1 delivered 27 passing in-process tests covering:
- Severity-driven confidence scaling (S250)
- Strategy-type-aware risk treatment (S251)
- Dual-risk fan-out and context preservation (S252)
- CI-dedicated behavioral gate (S253)

However, all tests executed within a single Go process using Hollywood actor
message collectors. No test validated that behavioral properties survive:
1. CBOR serialization to NATS JetStream
2. Writer pipeline row mapping (`mapDecisionRow`, `mapStrategyRow`, `mapRiskRow`)
3. ClickHouse batch insertion (string→Float64, JSON marshaling)
4. ClickHouse read-back (Float64→string, JSON unmarshaling)
5. HTTP response composition

This gap meant that a serialization bug, JSON field omission, or float precision
loss could silently corrupt behavioral data in the analytical path while all
in-process tests passed.

## Solution: Two-Layer Proof

### Layer 1: Serialization Round-Trip Unit Tests

**File:** `internal/adapters/clickhouse/writerpipeline/behavioral_roundtrip_test.go`
**CI Gate:** `make test-behavioral-roundtrip` (in behavioral-scenarios CI job)

These tests exercise the exact code path used by the writer and reader:
- `mapDecisionRow()` → `FormatFloat()` / `ParseSignalInputsJSON()` / `ParseMetadataJSON()`
- `mapStrategyRow()` → `FormatFloat()` / `ParseDecisionInputsJSON()` / `ParseMetadataJSON()`
- `mapRiskRow()` → `FormatFloat()` / `ParseStrategyInputsJSON()` / `ParseConstraintsJSON()` / `ParseMetadataJSON()`

**17 tests** covering 8 scenarios:

| # | Scenario | What It Proves |
|---|----------|----------------|
| 1 | Decision severity high | Severity enum "high" survives write→read |
| 2 | Decision severity low | Severity enum "low" survives write→read |
| 3 | All severity enum values | none/low/moderate/high all round-trip correctly |
| 4 | Strategy severity-scaled confidence | Scaled confidence and decision context (severity, rationale) survive in decisions[] JSON |
| 5 | Strategy low severity reduced confidence | Low severity produces strategy confidence < decision confidence after round-trip |
| 6 | Risk position_exposure counter-trend | Strategy-type metadata (confidence_factor=0.90), constraints, decision_severity all survive |
| 7 | Risk drawdown_limit pro-trend | Stop_type_factor=1.15, wider stops, strategy-type metadata all survive |
| 8 | Severity contrast high vs low | High severity → higher confidence + larger position than low severity after round-trip |
| 9 | Cross-chain risk profile divergence | Counter-trend vs pro-trend produce different confidence_factor in metadata |
| 10 | Not-triggered clean flow | severity=none, confidence=0, direction=flat all survive cleanly |
| 11 | Confidence precision (8 values) | Float64 round-trip preserves confidence to within 1e-10 tolerance |
| 12 | Full chain (decision→strategy→risk) | Correlation/causation IDs, severity propagation, rationale identity, confidence ordering, constraints non-zero |

### Layer 2: Enhanced Smoke Analytical Script

**File:** `scripts/smoke-analytical-e2e.sh` (Phase 8: Behavioral Semantic Verification)

Added 6 behavioral checks to the existing analytical E2E smoke script:

| Check | What It Validates |
|-------|-------------------|
| 8a. Severity enum fidelity | All decision severity values are valid enum members (none/low/moderate/high) |
| 8b. Confidence scaling | Strategy confidence ≤ decision confidence for triggered outcomes |
| 8c. Risk behavioral metadata | strategy_type and confidence_factor present in risk metadata |
| 8d. Risk constraints | Approved risk assessments have non-empty constraints |
| 8e. Dual-risk fan-out | Both position_exposure and drawdown_limit have data in ClickHouse |
| 8f. Chain B verification | trend_following → drawdown_limit carries stop_distance and strategy_type |

## CI Integration

The behavioral round-trip tests are integrated into the existing CI pipeline:

```yaml
behavioral-scenarios:
  steps:
    - Run behavioral scenario tests (charter-protected surface)   # existing
    - Run behavioral round-trip serialization tests (S255)         # NEW
```

The smoke behavioral checks run as part of `smoke-analytical` CI job, which
executes against a live Docker Compose stack.

## What OD-BW1 Closure Proves

After S255, the behavioral wave has evidence at three levels:

| Level | Scope | Where |
|-------|-------|-------|
| In-process | Actor chain behavioral logic | `scenario_end_to_end_test.go` (27 tests) |
| Serialization | Write→read field fidelity | `behavioral_roundtrip_test.go` (17 tests) |
| Full-stack | NATS→writer→CH→reader→HTTP | `smoke-analytical-e2e.sh` Phase 8 (6 checks) |

## Residual Risk

OD-BW1 is **closed** as a blocking debt. The remaining residual is:
- The serialization round-trip tests simulate ClickHouse column types via Go's
  native types (float64, string), not actual ClickHouse protocol encoding.
  The smoke-analytical script covers this final gap when run against live infra.
- CBOR envelope encoding/decoding is not tested in isolation, but is exercised
  by the integration tests (embedded NATS) and the smoke-analytical pipeline.

These residuals are non-blocking and can be addressed in future hardening if needed.
