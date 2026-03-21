# Behavioral Edge Hardening (S256)

## Purpose

S256 closes the cheapest, highest-value edges remaining from BEHAVIORAL-WAVE-1
before the transition gate at S257. It does not open new feature scope.

## Hardening Scope

Three targeted improvements were selected based on cost/benefit ratio:

### 1. Severity Normalization (OD-BW4 partial)

**Problem:** Severity lookups in both strategy and risk layers used exact string
matching. Values like `"HIGH"`, `" high "`, or `"Moderate"` silently defaulted to
neutral (1.0x), producing invisible behavioral degradation.

**Fix:** Added `strings.TrimSpace` + `strings.ToLower` normalization at the
lookup boundary in:
- `internal/application/risk/risk_scaling.go:lookupSeverityFactor`
- `internal/application/strategy/severity_scaling.go:severityFactor`

**Design choice:** Normalization happens at lookup, not at input. Metadata and
rationale fields preserve the original value for observability.

### 2. Risk Rejection Path (OD-BW3)

**Problem:** Risk evaluators only produced `approved` or `modified` dispositions.
Zero or negative confidence inputs generated degenerate assessments (approved with
0-size positions), making `DispositionRejected` a dead constant.

**Fix:** Both evaluators now emit `DispositionRejected` when parsed confidence
is <= 0:
- `internal/application/risk/drawdown_limit_evaluator.go` — rejects with rationale
- `internal/application/risk/position_exposure_evaluator.go` — rejects with rationale

**Design choice:** Rejection returns `(assessment, true)` not `(zero, false)`.
This means the rejection is a valid, observable assessment that flows through the
pipeline with full metadata, rather than a silent drop.

### 3. Edge Case Test Coverage

**10 new tests** covering:
- Zero confidence rejection (both evaluators)
- Negative confidence rejection (both evaluators)
- Severity casing normalization: `"HIGH"`, `"High"`, `" high "`, `"LOW"`, `"  moderate  "`
- Whitespace-only severity defaults to neutral
- `"NONE"` (uppercase) treated as neutral
- Boundary confidence (1.0000 max, 0.0001 tiny positive)

## Files Changed

| File | Change |
|------|--------|
| `internal/application/risk/risk_scaling.go` | Severity normalization |
| `internal/application/risk/risk_scaling_test.go` | 8 new edge case tests |
| `internal/application/risk/drawdown_limit_evaluator.go` | Rejection path |
| `internal/application/risk/position_exposure_evaluator.go` | Rejection path |
| `internal/application/strategy/severity_scaling.go` | Severity normalization |
| `internal/application/strategy/severity_scaling_test.go` | 2 new normalization test suites |

## Invariants Preserved

- All 37 risk scaling tests pass (10 new + 27 existing)
- All strategy tests pass (10 new + existing)
- All actor integration tests pass
- No domain model changes
- No infrastructure changes
- No new dependencies beyond `strings` (stdlib)
