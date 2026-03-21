# Integration and CI Hardening for Behavioral Scenarios

## Purpose

This document describes the minimal CI/integration hardening applied in S253 to protect the behavioral scenarios validated in S252 from regression.

## Problem

The S252 behavioral scenarios (6 end-to-end tests + 7 actor-chain tests + 14 scaling tests = 27 behavioral tests) proved that the `decision → strategy → risk` pipeline produces coherent, severity-aware, strategy-type-sensitive output. However, these tests ran exclusively as part of the general `make test` target, making behavioral regressions invisible at the CI level — a failure in any of these tests would appear as a generic unit test failure with no attribution to the behavioral charter.

## Solution

### Dedicated Makefile Target

A new `make test-behavioral` target isolates the charter-protected surface:

```makefile
BEHAVIORAL_PACKAGES := ./internal/actors/scopes/derive/... ./internal/application/strategy/... ./internal/application/risk/...
BEHAVIORAL_PATTERN := ^(TestScenario_|TestActorChain_|TestPositionExposure_|TestDrawdown_|TestScaleConfidence|TestAdjustParam|TestFormatParam)

test-behavioral:
	@echo "Running behavioral scenario tests (charter-protected surface)..."
	@$(GO) test $(BEHAVIORAL_PACKAGES) -run '$(BEHAVIORAL_PATTERN)' -v -count=1
```

This target:
- Runs only the tests that validate behavioral properties (not structural/validation tests).
- Uses `-v` for explicit pass/fail attribution per scenario.
- Uses `-count=1` to prevent cache masking regressions.

### Dedicated CI Job

A new `behavioral-scenarios` job in `.github/workflows/ci.yml` runs in parallel with unit-tests, integration-tests, and codegen-golden:

```yaml
behavioral-scenarios:
  name: Behavioral Scenarios
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ env.GO_VERSION }}
    - name: Run behavioral scenario tests (charter-protected surface)
      run: make test-behavioral
```

This job:
- Provides a separate red/green signal for behavioral correctness.
- Makes the behavioral charter visible in PR checks (a failing behavioral test shows as "Behavioral Scenarios: failed", not buried in "Unit Tests: failed").
- Adds no infrastructure cost (no NATS, no ClickHouse, no Docker).

## Test Naming Convention

All behavioral tests follow predictable prefixes that enable the `-run` filter:

| Prefix | Layer | Count | File |
|--------|-------|-------|------|
| `TestScenario_` | End-to-end chain | 6 | `scenario_end_to_end_test.go` |
| `TestActorChain_` | Actor wiring | 7 | `actor_chain_integration_test.go` |
| `TestPositionExposure_` | Risk scaling (exposure) | 5 | `risk_scaling_test.go` |
| `TestDrawdown_` | Risk scaling (drawdown) | 6 | `risk_scaling_test.go` |
| `TestScaleConfidence` | Strategy severity scaling | 1 | `severity_scaling_test.go` |
| `TestAdjustParam` | Strategy severity scaling | 1 | `severity_scaling_test.go` |
| `TestFormatParam` | Strategy severity scaling | 1 | `severity_scaling_test.go` |

New behavioral tests should follow these prefixes to automatically join the protected surface.

## Design Decisions

1. **No build tags**: The behavioral tests have no external dependencies (no NATS, no ClickHouse). Using build tags would add friction without benefit. The `-run` pattern achieves isolation without requiring file-level annotations.

2. **Parallel with unit-tests, not replacing**: The behavioral tests also run in `make test` (unit-tests job). This is intentional — the behavioral job provides visibility, while the unit-tests job ensures nothing is accidentally excluded.

3. **No infrastructure expansion**: The behavioral tests use in-process Hollywood actors with `msgCollector` stand-ins. No Docker, no compose, no external services.

4. **Verbose output in CI**: The `-v` flag ensures each scenario appears in CI logs, making it easy to identify which behavioral property regressed.
