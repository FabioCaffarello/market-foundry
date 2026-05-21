# Stage S253 — Integration and CI Hardening for Behavioral Scenarios

| Field | Value |
|-------|-------|
| Stage | S253 |
| Status | Complete |
| Predecessor | S252 (scenario-based end-to-end domain validation) |
| Successor | S254 (charter gate) |
| Date | 2026-03-21 |

## Objective

Apply minimal CI/integration hardening so that the 27 behavioral scenarios validated in S252 have dedicated pipeline protection, preventing behavioral regression from going undetected.

## Executive Summary

S252 proved that the `decision → strategy → risk` chain produces coherent, severity-aware, strategy-type-sensitive output through 6 end-to-end scenarios, 7 actor-chain wiring tests, and 14 scaling behavior tests. However, all 27 tests were invisible at the CI level — buried inside the general `unit-tests` job. S253 elevates these to a dedicated `behavioral-scenarios` CI job with its own red/green signal, giving the behavioral charter explicit pipeline protection without adding infrastructure.

## Hardening Applied

### 1. Makefile Target: `make test-behavioral`

New target that isolates the charter-protected test surface using package paths and test name patterns:

- **Packages**: `derive`, `strategy`, `risk` (the three behavioral packages)
- **Pattern**: `TestScenario_`, `TestActorChain_`, `TestPositionExposure_`, `TestDrawdown_`, `TestScaleConfidence`, `TestAdjustParam`, `TestFormatParam`
- **Flags**: `-v` (verbose, per-scenario attribution), `-count=1` (no cache)

### 2. CI Job: `behavioral-scenarios`

New GitHub Actions job that runs `make test-behavioral` in parallel with existing jobs. Provides a separate check in PRs — a behavioral regression shows as "Behavioral Scenarios: failed" rather than being hidden in "Unit Tests: failed".

### 3. No Infrastructure Added

The behavioral tests use in-process Hollywood actors with `msgCollector` stand-ins. No Docker, no compose, no NATS, no ClickHouse. The CI job needs only Go.

## Files Changed

| File | Change |
|------|--------|
| `Makefile` | Added `test-behavioral` target with `BEHAVIORAL_PACKAGES` and `BEHAVIORAL_PATTERN` |
| `.github/workflows/ci.yml` | Added `behavioral-scenarios` job |
| `docs/architecture/integration-and-ci-hardening-for-behavioral-scenarios.md` | New — hardening design and decisions |
| `docs/architecture/behavioral-scenarios-protected-surface-and-remaining-gaps.md` | New — protected surface inventory and gap analysis |
| `docs/stages/stage-s253-integration-and-ci-hardening-for-behavioral-scenarios-report.md` | New — this report |

## Protection Added to Pipeline

| CI Job | What It Protects | Dependencies |
|--------|------------------|--------------|
| `behavioral-scenarios` (NEW) | 27 behavioral tests: severity scaling, strategy-type risk asymmetry, dual-risk fan-out, context preservation, correlation ID survival | Go only |
| `unit-tests` (existing) | All tests including behavioral (overlap is intentional) | Go only |
| `integration-tests` (existing) | Embedded NATS wiring | Go + embedded NATS |
| `codegen-golden` (existing) | Schema/codegen correctness | Go only |
| `smoke-analytical` (existing) | Full-stack NATS→CH→HTTP | Docker + compose |

## Validation

```
$ make test-behavioral
Running behavioral scenario tests (charter-protected surface)...
--- PASS: TestActorChain_Signal_To_Decision_To_Strategy_To_Risk (0.05s)
--- PASS: TestActorChain_NotTriggered_FlowsThrough (0.05s)
--- PASS: TestActorChain_EMACrossover_Bullish_Triggered (0.05s)
--- PASS: TestActorChain_EMACrossover_Bearish_NotTriggered (0.05s)
--- PASS: TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk (0.05s)
--- PASS: TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk (0.05s)
--- PASS: TestActorChain_CorrelationID_PreservedEndToEnd (0.05s)
--- PASS: TestScenario_RSIOversold_MeanReversion_DualRisk (0.05s)
--- PASS: TestScenario_EMACrossover_TrendFollowing_DualRisk (0.05s)
--- PASS: TestScenario_SeverityContrast_HighVsLow (0.10s)
--- PASS: TestScenario_CrossChain_RiskProfileComparison (0.10s)
--- PASS: TestScenario_NotTriggered_BothChains_FlatApproved (0.10s)
--- PASS: TestScenario_ContextPreservation_RationaleEndToEnd (0.05s)
ok   internal/actors/scopes/derive   0.989s
[+ 14 scaling tests in strategy + risk packages]
ok   internal/application/strategy   0.189s
ok   internal/application/risk       0.334s

27 behavioral tests, 0 failures.
```

## Remaining Gaps

| Gap | Risk | Mitigation Path |
|-----|------|-----------------|
| Full-stack behavioral smoke (NATS→CH round-trip for behavioral families) | Medium | Future: add behavioral assertions to `smoke-analytical` |
| Multi-symbol concurrent behavioral isolation | Low | Unit tests cover ownership bleed |
| Timeframe-aware behavioral variation | Low | No timeframe-dependent logic exists yet |
| Performance budget enforcement | Low | Tests complete in ~1s; add benchmarks if latency becomes a concern |
| Behavioral golden-value snapshots | Low | Current exact assertions are stable |

## Preparation for S254

The base is ready for the charter gate:

1. **27 behavioral tests are CI-protected** — regression is now a pipeline-visible failure.
2. **Protected surface is documented** — S254 can audit the inventory in `behavioral-scenarios-protected-surface-and-remaining-gaps.md`.
3. **Gaps are explicit** — S254 can decide which gaps require closure before charter sign-off and which are acceptable as known limitations.
4. **Extension path is clear** — new behavioral tests auto-join the protected surface via naming convention.

## Guard Rail Compliance

- No new family opened.
- No observability infrastructure added.
- No Docker/compose/external dependencies in the new CI job.
- Partial coverage documented explicitly (not treated as total protection).
- Hardening is small (2 file changes), useful (dedicated CI signal), and proportional (matches the in-process nature of the tests).
