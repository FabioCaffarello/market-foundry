# Stage S247 — Chain B Integration Completion for drawdown_limit

**Status:** Complete
**Date:** 2026-03-21
**Predecessor:** S246 (smoke e2e breadth coverage expansion)
**Objective:** Close D2 debt from S244 — Chain B integration test gap for `drawdown_limit`

---

## 1. Executive Summary

S247 closes the last explicit technical debt from the breadth wave (S241-S244). The D2 gap — `drawdown_limit` not participating in a full Chain B integration test — is now eliminated. The `risk` domain achieves full integration symmetry between both risk types across all validation layers.

Single test added. No scope inflation. No new functionality introduced.

---

## 2. Gap Closed

**D2 (Low):** "Integration test Chain B does not pass through `drawdown_limit` risk."

The existing `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk` proved Chain B end-to-end but wired `position_exposure` at the risk stage. The new `TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk` proves the same chain with `drawdown_limit`, validating:

- EMA bullish signal → ema_crossover triggered
- ema_crossover → trend_following_entry long
- trend_following_entry → drawdown_limit approved
- Type = `drawdown_limit`, disposition = `approved`, final = `true`
- Strategy type = `trend_following_entry` preserved in risk assessment
- Decision severity survives full chain
- `stop_distance` constraint present (drawdown_limit-specific)
- Correlation ID preserved end-to-end
- `Validate()` passes

---

## 3. Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/actor_chain_integration_test.go` | Added `TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk` |
| `docs/architecture/chain-b-integration-completion-for-drawdown-limit.md` | New: gap analysis and resolution record |
| `docs/architecture/risk-breadth-integration-symmetry-notes.md` | New: symmetry matrix across all validation layers |
| `docs/stages/stage-s247-chain-b-integration-completion-for-drawdown-limit-report.md` | This report |

---

## 4. Before / After — Chain B Coverage

### Before (S244-S246)

```
Chain B:
  EMA → ema_crossover → trend_following_entry → position_exposure  ✅ (proven)
  EMA → ema_crossover → trend_following_entry → drawdown_limit    ❌ (D2 open)
```

Chain integration tests: **6 functions**

### After (S247)

```
Chain B:
  EMA → ema_crossover → trend_following_entry → position_exposure  ✅ (proven)
  EMA → ema_crossover → trend_following_entry → drawdown_limit    ✅ (proven)
```

Chain integration tests: **7 functions**

---

## 5. Debt Ledger Update

| # | Debt | S244 Severity | S247 Status |
|---|------|---------------|-------------|
| D1 | Smoke test coverage for 3 new types | Medium | Closed (S246) |
| D2 | Chain B integration test with drawdown_limit | Low | **Closed (S247)** |
| D3 | Remote CI verification of accumulated changes | High | Closed (S245) |

**All three breadth wave debts are now closed.**

---

## 6. Residual Limitations

| Aspect | Status | Justification |
|--------|--------|---------------|
| Chain A + drawdown_limit chain test | Not added | By design: Chain A's natural risk evaluator is position_exposure. drawdown_limit is exercised at actor-test level for Chain A inputs. The N x M combinatorial expansion is not justified. |
| Dedicated correlation_id test for Chain B | Not added | The new test validates correlation_id at the risk boundary. A standalone correlation test for Chain B would be redundant. |

These are design choices, not open debts.

---

## 7. Breadth Wave Gate Readiness

With D1, D2, and D3 all closed:

- All breadth types have unit, actor, chain integration, smoke, and HTTP test coverage
- Both risk types have symmetric integration proof across all layers
- Remote CI has verified the accumulated codebase
- No open debts remain from the breadth wave

**The breadth wave is ready for a final hardening gate.**

---

## 8. Recommended Next Steps (S248+)

| Priority | Action | Rationale |
|----------|--------|-----------|
| 1 | Remote CI verification of S247 changes | Maintains CI-proven invariant established in S245 |
| 2 | Breadth wave final gate report | Formal closure document with all debts resolved |
| 3 | Begin next architectural phase | Breadth wave is complete; the codebase is ready for the next initiative |
