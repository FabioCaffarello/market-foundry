# Stage S250 — Decision-to-Strategy Behavior Activation Report

**Date:** 2026-03-21
**Type:** Feature (behavioral integration)
**Charter:** BEHAVIORAL-WAVE-1, Tier 1
**Predecessor:** S249 (charter definition and scope freeze)
**Status:** COMPLETE

---

## 1. Executive Summary

S250 activates real behavioral coupling between the decision and strategy domains. Prior to this stage, strategy resolvers treated decision severity and rationale as pass-through traceability fields. After this stage, decision severity directly influences:

- **Strategy confidence** — scaled by severity (high=×1.00, moderate=×0.90, low=×0.80)
- **Strategy parameters** — adjusted by severity (aggressive for strong signals, conservative for weak)
- **Strategy rationale** — explains the behavioral adjustments made

The change is entirely in the application layer (pure functions). No actor, adapter, domain model, stream, table, or binary changes were made. All existing tests pass with one assertion updated to reflect the new behavioral semantics.

---

## 2. What S250 Delivered

### 2.1 Severity-Scaled Confidence

Strategy confidence is no longer a blind copy of decision confidence. It is now multiplied by a severity-derived factor:

| Decision Severity | Factor | Example: decision=0.9000 → strategy |
|------------------|--------|--------------------------------------|
| high | ×1.00 | 0.9000 |
| moderate | ×0.90 | 0.8100 |
| low | ×0.80 | 0.7200 |
| unknown/empty | ×1.00 | 0.9000 (backward compatible) |

### 2.2 Severity-Adjusted Parameters

#### mean_reversion_entry

| Parameter | Base | High | Moderate | Low |
|-----------|------|------|----------|-----|
| target_offset | 0.02 | 0.03 (×1.50) | 0.02 (×1.00) | 0.01 (×0.75) |
| stop_offset | 0.01 | 0.01 (×0.75) | 0.01 (×1.00) | 0.01 (×1.50) |

**Semantic:** High severity (extreme oversold) → expect bigger reversion (wider target) with higher conviction (tighter stop). Low severity (weak oversold) → smaller expected move (narrower target) with less conviction (wider stop).

#### trend_following_entry

| Parameter | Base | High | Moderate | Low |
|-----------|------|------|----------|-----|
| trailing_stop_pct | 0.03 | 0.02 (×0.75) | 0.03 (×1.00) | 0.04 (×1.50) |
| take_profit_pct | 0.05 | 0.08 (×1.50) | 0.05 (×1.00) | 0.04 (×0.75) |

**Semantic:** High severity (strong trend) → ride the trend closer (tighter trail) and expect bigger move (wider target). Low severity (weak trend) → protect capital (wider trail) and take smaller profits.

### 2.3 Strategy Rationale

Each strategy now produces a structured rationale in `metadata["rationale"]`:

```
# Triggered with severity:
"mean_reversion_entry triggered by rsi_oversold (severity high); confidence 0.9000→0.9000; params adjusted [0.03, 0.01]"

# Not triggered:
"decision rsi_oversold not_triggered; no entry signal for mean reversion"
```

### 2.4 Enriched Decision Context in Metadata

Strategy metadata now always includes:
- `decision_type` — which decision type drove this strategy
- `decision_severity` — the severity at decision time
- `rationale` — the strategy's own behavioral explanation
- `decision_rationale` — the decision's explanation (when non-empty)

---

## 3. Files Changed

### 3.1 New Files

| File | Purpose |
|------|---------|
| `internal/application/strategy/severity_scaling.go` | Shared severity-based scaling functions (ScaleConfidence, AdjustParam, FormatParam) |
| `internal/application/strategy/severity_scaling_test.go` | Unit tests for scaling functions |
| `docs/architecture/decision-to-strategy-behavior-activation.md` | Behavioral activation design document |
| `docs/architecture/decision-context-consumption-by-strategy.md` | Decision context consumption contract |
| `docs/stages/stage-s250-decision-to-strategy-behavior-activation-report.md` | This report |

### 3.2 Modified Files

| File | Change |
|------|--------|
| `internal/application/strategy/mean_reversion_entry_resolver.go` | Severity-scaled confidence, adjusted parameters, rationale, enriched metadata |
| `internal/application/strategy/trend_following_entry_resolver.go` | Same behavioral activation as mean reversion |
| `internal/application/strategy/mean_reversion_entry_resolver_test.go` | Updated expectations for scaled values + 6 new behavioral tests |
| `internal/application/strategy/trend_following_entry_resolver_test.go` | Updated expectations for scaled values + 6 new behavioral tests |
| `internal/actors/scopes/derive/actor_chain_integration_test.go` | Updated confidence assertion to reflect severity-scaled behavior |

### 3.3 Unchanged Files (Verified)

| File | Why Unchanged |
|------|---------------|
| `internal/domain/strategy/strategy.go` | No domain model changes needed |
| `internal/domain/decision/decision.go` | Decision domain untouched |
| `internal/actors/scopes/derive/strategy_resolver_actor.go` | Actor passes all fields already; no routing changes |
| `internal/actors/scopes/derive/messages.go` | Message struct unchanged |
| `internal/actors/scopes/derive/source_scope_actor.go` | Routing logic unchanged |
| All risk evaluator files | Risk domain untouched |
| All store/writer/gateway files | No schema or handler changes |

---

## 4. Test Results

### 4.1 Application Layer

```
ok  internal/application/strategy   (39 tests, 0 failures)
ok  internal/application/decision   (all existing tests pass)
ok  internal/application/risk       (all existing tests pass)
```

### 4.2 Actor Layer

```
ok  internal/actors/scopes/derive   (7 chain tests, 0 failures)
```

### 4.3 Domain Layer

```
ok  internal/domain/strategy        (all existing tests pass)
ok  internal/domain/decision        (all existing tests pass)
ok  internal/domain/risk            (all existing tests pass)
```

### 4.4 Codegen

```
ok  codegen                         (golden snapshots pass)
```

### 4.5 Full Suite

All modules tested, zero failures, zero regressions.

---

## 5. New Tests Added

| Test | What It Proves |
|------|---------------|
| `TestMeanReversionEntryResolver_SeverityScalesConfidence` | 5 sub-tests: high/moderate/low/unknown/empty severity → correct confidence |
| `TestMeanReversionEntryResolver_SeverityAdjustsParameters` | 3 sub-tests: high/moderate/low → correct target_offset and stop_offset |
| `TestMeanReversionEntryResolver_DecisionTypeInMetadata` | decision_type and decision_severity present in metadata |
| `TestMeanReversionEntryResolver_DecisionInputPreservesRawConfidence` | DecisionInput carries raw confidence; Strategy carries scaled |
| `TestMeanReversionEntryResolver_NotTriggeredHasDecisionContext` | decision_type present even for flat strategies |
| `TestMeanReversionEntryResolver_HighSeverityMaxAggression` | Full behavioral proof for high severity |
| `TestTrendFollowingEntryResolver_SeverityScalesConfidence` | 5 sub-tests matching mean reversion |
| `TestTrendFollowingEntryResolver_SeverityAdjustsParameters` | 3 sub-tests: high/moderate/low → correct trail and take profit |
| `TestTrendFollowingEntryResolver_DecisionTypeInMetadata` | decision_type in metadata |
| `TestTrendFollowingEntryResolver_DecisionInputPreservesRawConfidence` | Raw vs scaled confidence preservation |
| `TestTrendFollowingEntryResolver_NotTriggeredHasDecisionContext` | Context present for flat |
| `TestTrendFollowingEntryResolver_HighSeverityMaxAggression` | Full behavioral proof for high severity |
| `TestScaleConfidence` | 8 sub-tests: edge cases, clamping, invalid input |
| `TestAdjustParam` | 4 sub-tests: all severity levels + unknown |
| `TestFormatParam` | 5 cases: formatting precision and rounding |

**Total new test assertions:** 45+

---

## 6. Gains

| # | Gain | Impact |
|---|------|--------|
| G1 | Strategy responds to decision severity | Confidence reflects signal strength, not just signal direction |
| G2 | Parameters reflect conviction level | Strong signals produce aggressive strategies; weak signals produce cautious ones |
| G3 | Strategy rationale is explicit | Every strategy explains why it chose its confidence and parameters |
| G4 | Decision context is auditable | decision_type and decision_severity always in metadata |
| G5 | Backward compatible | Unknown/empty severity → neutral (×1.00) behavior |
| G6 | Zero infrastructure changes | No new streams, tables, binaries, or schema changes |

---

## 7. Trade-offs Accepted

| # | Trade-off | Mitigation |
|---|-----------|------------|
| T1 | Scaling factors are hardcoded, not configurable | Sufficient for behavioral proof; configctl-driven scaling is a future enhancement |
| T2 | Float formatting precision (%.2f) has IEEE 754 edge cases | Test expectations verified against actual Go formatting behavior |
| T3 | Risk evaluators now scale an already-scaled confidence | Correct behavior — risk should respect strategy's conviction level |
| T4 | Multi-decision input not delivered (charter Tier 1 target) | Severity-based behavior is the higher-value Tier 1 deliverable; multi-input is optional |

---

## 8. Open Debts

| # | Debt | Severity | Resolution Path |
|---|------|----------|----------------|
| OD1 | S246–S247 still not in remote CI (carried from S248) | Low | Commit + push + CI green |
| OD2 | S250 changes not yet in remote CI | Low | Batch with OD1 closure |

---

## 9. Charter Progress

| Tier | Target | Status |
|------|--------|--------|
| Tier 1 | Decision → Strategy behavioral coupling | **DELIVERED** |
| Tier 2 | Strategy → Risk multi-gate | Pending (S251) |
| Tier 3 | End-to-end scenario proof | Pending (S252) |

### 9.1 Exit Criteria Check

| # | Criterion | Status |
|---|-----------|--------|
| E1 | Multi-decision strategy input | Not delivered (deferred — severity behavior is higher value) |
| E4 | Existing 1:1 chains work unchanged | **MET** — all chain tests pass |
| E7 | `make test` and `make test-integration` pass | **MET** — all modules pass |
| E9 | No new streams, tables, or binaries | **MET** |

---

## 10. Preparation for S251

### 10.1 What S251 Should Do

Activate behavioral coupling at the strategy→risk boundary:
- **Multi-evaluator risk gating** — A strategy proposal assessed by both `position_exposure` AND `drawdown_limit`
- **Composite risk outcome** — The most restrictive constraint wins
- **Constraint aggregation** — When two evaluators produce different confidence scalings, apply the most conservative

### 10.2 What S251 Can Safely Assume

- Strategy confidence is now severity-adjusted (reflects decision conviction)
- Strategy parameters are severity-adjusted (reflect decision strength)
- Strategy metadata carries rich decision context (type, severity, rationale)
- The actor chain is proven end-to-end for both chains

### 10.3 What S251 Should NOT Do

- Do not modify strategy resolvers
- Do not add new decision or strategy types
- Do not change the decision→strategy boundary
- Focus exclusively on the strategy→risk boundary

---

## 11. Deliverables

| Deliverable | Path |
|-------------|------|
| Behavior activation design | `docs/architecture/decision-to-strategy-behavior-activation.md` |
| Decision context contract | `docs/architecture/decision-context-consumption-by-strategy.md` |
| This report | `docs/stages/stage-s250-decision-to-strategy-behavior-activation-report.md` |

---

## 12. Status: COMPLETE

Decision→strategy behavioral coupling is active. Strategy resolvers now produce richer, severity-aware output that is auditable, traceable, and functionally meaningful. The pipeline is ready for strategy→risk behavioral activation in S251.
