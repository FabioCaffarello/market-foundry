# Breadth Hardening Wave — Gains, Trade-offs, and Open Debts

**Stage:** S248
**Date:** 2026-03-21
**Scope:** Honest accounting of what the S245–S247 hardening tranche gained, what trade-offs were accepted, and what (if anything) remains open.

---

## 1. Gains

### G1 — Real Defect Discovery via Remote CI (S245)

The first remote CI run after the breadth wave caught a ClickHouse multi-statement migration failure that was invisible to the local stack. This validates the entire premise of D3: remote CI is not a formality — it catches real issues.

**Impact:** Migration `007_add_decision_severity_rationale.sql` fixed before it could break any fresh deployment. The decision domain depth columns (`severity`, `rationale`) now migrate correctly on any stack.

### G2 — Smoke Coverage Parity (S246)

The three breadth types (`ema_crossover`, `trend_following_entry`, `drawdown_limit`) now have identical smoke validation depth as their Chain A counterparts:

- ClickHouse row count verification
- HTTP endpoint 200 + JSON structure validation
- Item count > 0
- Server-Timing header presence
- Filter validation (outcome, direction, disposition)
- Domain depth propagation (severity → strategy → risk)
- Multi-symbol validation (2 symbols × 4 timeframes)
- Cross-symbol isolation (collision/bleed detection)
- Error handling (missing timeframe → 400)

**Impact:** Any regression in breadth types will be caught by the same smoke infrastructure that protects pre-breadth types. No asymmetry in operational coverage.

### G3 — Chain B End-to-End Integration Proof (S247)

The `drawdown_limit` risk type is now proven to work through the full Chain B actor pipeline, not just in isolation. The test validates 13 assertions including confidence scaling (×0.90), `stop_distance` constraint, decision severity propagation, and correlation_id preservation.

**Impact:** The risk domain is now symmetric at the integration level. Both risk types (`position_exposure`, `drawdown_limit`) have end-to-end chain proofs.

### G4 — Zero Production Code Changes

The entire hardening tranche (S245–S247) changed zero lines of production code (excluding the migration fix in S245, which was a defect correction). All additions were tests, smoke scripts, HTTP client files, and documentation.

**Impact:** The hardening tranche could not have introduced new bugs in production paths. Risk of regression is near zero.

---

## 2. Trade-offs

### T1 — Chain A + `drawdown_limit` Combination Not Tested

The integration test matrix covers:
- Chain A + `position_exposure` ✓
- Chain B + `position_exposure` ✓
- Chain B + `drawdown_limit` ✓
- Chain A + `drawdown_limit` ✗

This was a deliberate design choice: Chain A's natural risk evaluator is `position_exposure`, and the combinatorial expansion (2 chains × 2 risk types = 4 tests) was not justified by the risk. The `drawdown_limit` evaluator is proven both in isolation (21 unit tests) and in its natural chain (Chain B). Cross-chain wiring is not a production use case today.

**Accepted risk:** If a future feature wires `drawdown_limit` into Chain A, integration coverage for that combination will need to be added at that time.

### T2 — Smoke Scripts Require Live Infrastructure

The expanded smoke tests validate ClickHouse, NATS, and HTTP endpoints — they cannot run in a unit test context. This means smoke coverage is only exercised during CI's smoke-analytical-e2e job or manual local runs with the full stack up.

**Accepted risk:** Smoke regressions won't be caught by `go test` alone. The CI pipeline mitigates this, but local development cycles may not catch smoke-level issues until push.

### T3 — Warm-up Sensitivity

Chain B types require longer warm-up than Chain A (EMA Crossover: 21 candles vs RSI: 15 candles). The smoke scripts handle this gracefully by accepting null responses during warm-up, but this means the scripts cannot distinguish between "type not emitting yet because warm-up" and "type broken and not emitting."

**Accepted risk:** A silently broken breadth type could pass smoke during warm-up window. Mitigation: the `item count > 0` check after warm-up window catches this. The window is bounded (7m23s total CI run).

### T4 — CI Node.js Deprecation on Horizon

S245 noted that GitHub Actions Node.js 20 will be deprecated in June 2026. This is not a breadth-specific issue, but it was surfaced during breadth CI verification.

**Accepted risk:** CI will need action version updates before June 2026. Not blocking for breadth closure.

---

## 3. Open Debts

### OD1 — S246/S247 Not Yet in Remote CI (Severity: Low)

S246 and S247 implementation files are staged but not committed to main. Remote CI has not yet validated the expanded smoke scripts or the new chain integration test.

**Why this matters:** The hardening tranche's value is only fully realized when CI proves the expanded tests pass in a clean environment. Locally, everything passes — but S245 itself proved that local and remote can diverge.

**Resolution path:** Commit S246–S247 to main → push → observe CI green. This is mechanical, not architectural.

### OD2 — Migration Linting Not Yet Automated (Severity: Low)

S245 caught a multi-statement migration that ClickHouse rejects. There is no automated pre-flight check to prevent this class of error in future migrations.

**Why this matters:** The next migration could repeat the same mistake. The fix was ad-hoc.

**Resolution path:** Add a CI lint step that validates migration files contain exactly one statement, or explicitly use ClickHouse's multi-statement syntax if supported. This is a tooling improvement, not a breadth concern.

### OD3 — Go Module Cache Overhead in CI (Severity: Negligible)

`go.sum` files in subdirectories cause ~30s CI cache misses. Not blocking, but adds friction.

**Resolution path:** Consolidate module structure or configure CI cache keys to cover subdirectory modules. Low priority.

---

## 4. Summary

| Category    | Count | Detail                                                        |
|-------------|:-----:|---------------------------------------------------------------|
| Gains       | 4     | Real defect caught, smoke parity, chain proof, zero prod risk |
| Trade-offs  | 4     | All deliberate, all with known mitigation paths               |
| Open debts  | 3     | OD1 is mechanical closure; OD2-OD3 are tooling improvements   |

The hardening tranche delivered honest operational coverage without introducing complexity or risk. The trade-offs are proportionate and well-understood. The open debts are low-severity and have clear resolution paths.
