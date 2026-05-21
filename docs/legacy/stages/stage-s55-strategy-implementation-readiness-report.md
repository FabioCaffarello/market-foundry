# Stage S55 — Strategy Implementation Readiness Report

> Date: 2026-03-18
> Status: Complete
> Objective: Close config, governance, and wiring prerequisites for strategy implementation.

---

## 1. Executive Summary

S55 closed the remaining technical prerequisite (P-6: `strategy_families` in settings schema) and hardened the config dependency chain so that `strategy` can be implemented in S56 without improvisation. All config surfaces, validation rules, tests, governance checks, and documentation are now aligned.

---

## 2. What Was Done

### 2.1 Settings Schema (`internal/shared/settings/schema.go`)

| Change | Detail |
|---|---|
| Added `knownStrategyFamilies` registry | `mean_reversion_entry` registered as canonical name |
| Added `strategyDependsOnDecision` map | `mean_reversion_entry → rsi_oversold` |
| Added `StrategyFamilies` field to `PipelineConfig` | `strategy_families` JSON key, opt-in semantics |
| Added `IsStrategyFamilyEnabled()` | Same pattern as decision (opt-in, not backward-compatible) |
| Added `EnabledStrategyFamilies()` | Defensive copy, nil when empty |
| Extended `ValidatePipeline()` | Steps 6 (reject unknown) and 7 (enforce strategy→decision dependency) |

### 2.2 Settings Tests (`internal/shared/settings/settings_test.go`)

Added 8 new tests:

| Test | Validates |
|---|---|
| `TestValidatePipelineRejectsUnknownStrategyFamily` | Typo protection for strategy families |
| `TestValidatePipelineRejectsStrategyWithoutDecision` | mean_reversion_entry fails without rsi_oversold |
| `TestValidatePipelineAcceptsStrategyWithDecision` | Valid 3-layer chain (signal + decision + strategy) |
| `TestValidatePipelineAcceptsFullChain` | Complete 4-layer chain validation |
| `TestIsStrategyFamilyEnabledOptIn` | Empty list = no activation |
| `TestEnabledStrategyFamiliesReturnsNilWhenEmpty` | Nil semantics |
| `TestEnabledStrategyFamiliesReturnsCopy` | Mutation safety |

All 27 settings tests pass.

### 2.3 Deploy Configs

| File | Change |
|---|---|
| `deploy/configs/derive.jsonc` | Added `strategy_families` comment placeholder |
| `deploy/configs/store.jsonc` | Added `strategy_families` comment placeholder |

Configs are symmetric (both have the same commented entry). Activation happens in S56 by uncommenting.

### 2.4 Architecture Documentation

| File | Change |
|---|---|
| `docs/architecture/family-config-dependency-rules.md` | Added strategy layer to all sections: activation semantics, known families, dependency rules, cross-service consistency, adding-a-family guide, failure modes |
| `docs/architecture/strategy-implementation-readiness.md` | **New** — Complete readiness checklist with runtime wiring items for S56 |

---

## 3. Files Changed

| File | Type |
|---|---|
| `internal/shared/settings/schema.go` | Modified — strategy config surface |
| `internal/shared/settings/settings_test.go` | Modified — strategy test coverage |
| `deploy/configs/derive.jsonc` | Modified — strategy placeholder |
| `deploy/configs/store.jsonc` | Modified — strategy placeholder |
| `docs/architecture/family-config-dependency-rules.md` | Modified — strategy layer |
| `docs/architecture/strategy-implementation-readiness.md` | New — readiness checklist |
| `docs/stages/stage-s55-strategy-implementation-readiness-report.md` | New — this report |

---

## 4. Readiness State

### Config (all green)

| Prerequisite | Status |
|---|---|
| `strategy_families` in PipelineConfig | Done |
| Known family registry | Done |
| Dependency map (strategy→decision) | Done |
| `IsStrategyFamilyEnabled()` | Done |
| `EnabledStrategyFamilies()` | Done |
| Unknown name rejection | Done |
| Dependency validation | Done |
| Deploy config placeholders | Done |

### Governance (from S54, verified)

| Prerequisite | Status |
|---|---|
| STRATEGY_EVENTS in canonical streams | Done |
| Drift rules STD-1 to STD-5 | Done |
| Guardrails SG-1 to SG-10 | Done |
| Coverage map includes domain-strategy | Done |
| Config symmetry check | Done |

### Documentation (all complete)

| Document | Status |
|---|---|
| strategy-domain-design.md | Done (S53) |
| strategy-stream-families.md | Done (S53) |
| strategy-activation-and-ownership.md | Done (S53) |
| strategy-query-surface-guidelines.md | Done (S53) |
| family-config-dependency-rules.md | Updated (S55) |
| strategy-implementation-readiness.md | Done (S55) |
| cli-strategy-drift-rules.md | Done (S54) |
| cli-strategy-guardrails.md | Done (S54) |

---

## 5. Gaps Remaining

| ID | Gap | Severity | When to Address |
|---|---|---|---|
| **G-1** | raccoon-cli drift-detect reports ~30 missing artifact errors for strategy | Expected | These are the implementation checklist — each maps to a file S56 must create |
| **G-2** | Strategy config not yet active (commented) | By design | Uncomment in S56 after implementation lands |
| **G-3** | Multi-decision strategy pattern not designed | Low | Deferred to post-S56; single-decision proven first |
| **G-4** | Strategy history projection deferred | Low | No concrete consumer; latest-only in Phase 1 |
| **G-5** | Domain boundary invariants not enforced by CLI | Medium | Code review + tests; AST analysis out of scope |

None of these gaps block S56 implementation.

---

## 6. Dependency Chain Validated

```
observation → evidence (candle) → signal (rsi) → decision (rsi_oversold) → strategy (mean_reversion_entry)
```

Each hop is enforced at config validation time. Full chain acceptance tested in `TestValidatePipelineAcceptsFullChain`.

---

## 7. Recommendation

**S56 is clear to open.** All config, governance, and documentation prerequisites are satisfied. The implementation should follow the runtime wiring checklist in `strategy-implementation-readiness.md` section 6, which maps 1:1 to the ~30 raccoon-cli drift errors that serve as a living checklist.

### S56 scope recommendation:
1. Implement domain model + resolver (pure function)
2. Wire NATS adapters (registry, publisher, consumer, gateway, KV)
3. Wire actors (derive: resolver + publisher; store: consumer + projection)
4. Wire HTTP routes (handler + routes)
5. Integrate into supervisors (derive + store)
6. Activate config (uncomment strategy_families)
7. Verify raccoon-cli drift errors resolve to zero
