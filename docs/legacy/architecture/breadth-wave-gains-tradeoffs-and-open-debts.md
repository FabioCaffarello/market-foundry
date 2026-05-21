# Breadth Wave Gains, Trade-offs, and Open Debts

**Charter:** BREADTH-WAVE-1
**Stages:** S240–S244
**Date:** 2026-03-21

---

## 1. Gains

### 1.1 Structural Breadth Achieved

The pipeline now has genuine type diversity across all three analytical domains:

| Domain | Before Charter | After Charter | Gain |
|--------|---------------|--------------|------|
| Decision | 1 type (rsi_oversold) | 2 types (+ema_crossover) | +100% |
| Strategy | 1 type (mean_reversion_entry) | 2 types (+trend_following_entry) | +100% |
| Risk | 1 type (position_exposure) | 2 types (+drawdown_limit) | +100% |

### 1.2 Pattern Validation

Having two types per domain proved the pipeline's extensibility pattern:

- **Fan-out routing works:** The derive supervisor's `DecisionFamilyProcessor`, `StrategyFamilyProcessor`, and `RiskFamilyProcessor` slices accept N families without architectural changes.
- **Shared tables work:** All types within a domain share ClickHouse tables (`decisions`, `strategies`, `risk_assessments`) using the `type` column for discrimination.
- **Shared streams work:** All types within a domain share NATS JetStream streams (`DECISION_EVENTS`, `STRATEGY_EVENTS`, `RISK_EVENTS`) using subject-based routing.
- **KV isolation works:** Each type gets its own NATS KV bucket, preventing cross-type interference in latest-value queries.
- **Config gating works:** All families are opt-in via `pipeline.{domain}_families` configuration arrays.

### 1.3 Distinct Analytical Models

Each new type delivers genuinely distinct logic, not parameter variations:

- **ema_crossover:** Categorical signal interpretation (bullish/bearish/neutral) vs. RSI's numeric threshold comparison
- **trend_following_entry:** Pro-trend direction resolution with trailing stops vs. mean_reversion_entry's counter-trend resolution
- **drawdown_limit:** Stop-loss/drawdown constraint dimension vs. position_exposure's portfolio sizing dimension

### 1.4 Traceability Chain

Decision severity and rationale propagate through the entire chain: decision → strategy metadata → risk metadata. This was a pre-existing depth capability (from the domain evolution charter) that the breadth wave exercised without modification.

### 1.5 Test Coverage

| Layer | New Tests Added | Coverage Area |
|-------|----------------|---------------|
| Application (decision) | 20 tests | ema_crossover evaluator |
| Application (strategy) | 12 tests | trend_following_entry resolver |
| Application (risk) | 21 tests | drawdown_limit evaluator |
| Actor (strategy) | 6 tests | trend_following_entry resolver actor |
| Actor (risk) | 4 tests | drawdown_limit evaluator actor |
| Integration | 3 tests | EMA chain paths + full Chain B |
| Codegen | 6 golden snapshots | 3 new family artifact comparisons |

**Total: 72 new test assertions across the wave.**

---

## 2. Trade-offs

### 2.1 Smoke Test Not Extended

**What:** The E2E smoke test (`scripts/smoke-analytical-e2e.sh`) was not extended to cover the three new breadth types.

**Why:** The smoke test requires a running NATS + ClickHouse infrastructure and exercises the deployed pipeline. Extending it was not a charter deliverable (out of scope per §2.2: "Infrastructure expansion unless directly blocking a breadth deliverable").

**Impact:** New types are proven at unit + actor + integration test layers but not at the deployed-pipeline E2E layer. This is acceptable for the charter but should be addressed before production deployment.

### 2.2 Chain B Risk Path Uses position_exposure

**What:** The Chain B integration test (`TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk`) routes through `position_exposure` risk, not `drawdown_limit`.

**Why:** The derive supervisor fans out strategy results to ALL registered risk evaluators. The integration test verifies one path end-to-end. Testing the full N×M fan-out matrix would require combinatorial test setup that exceeds the breadth charter scope.

**Impact:** `drawdown_limit` is proven in isolation (21 unit tests + 4 actor tests) but not in a full chain integration test originating from EMA signal.

### 2.3 Codegen Remains Descriptive

**What:** The codegen engine validates naming conventions via golden snapshot comparison. It does not generate production code that is compiled into binaries.

**Why:** The codegen framework was frozen per charter §2.2 ("Codegen framework evolution or generalization" is out of scope).

**Impact:** Adding a third type to any domain still requires manual implementation following the established pattern. The codegen YAMLs and golden snapshots serve as consistency checks, not code generators.

### 2.4 No Configuration-Driven Decision→Strategy Mapping

**What:** Which decision types feed which strategy resolvers is determined by code in the derive supervisor, not by external configuration.

**Why:** The charter explicitly prohibited infrastructure expansion. Making this mapping configuration-driven would require changes to the settings schema and derive supervisor orchestration.

**Impact:** Adding a new decision→strategy pairing requires a code change, not a config change.

---

## 3. Open Debts

### 3.1 Must-Address Before Next Feature Wave

| # | Debt | Severity | Estimated Effort |
|---|------|----------|-----------------|
| D1 | Smoke test coverage for new types | Medium | 1 stage |
| D2 | Chain B integration test with drawdown_limit risk | Low | <1 stage |
| D3 | Remote CI verification of all S241–S244 changes | High | <1 stage |

### 3.2 Can-Address in Future Charter

| # | Debt | Severity | Notes |
|---|------|----------|-------|
| D4 | Configuration-driven decision→strategy mapping | Low | Useful for runtime flexibility but not blocking |
| D5 | Codegen evolution from descriptive to generative | Medium | Would reduce boilerplate for future types |
| D6 | N×M fan-out integration test matrix | Low | Combinatorial; diminishing returns beyond current coverage |
| D7 | Third type per domain | — | Next breadth wave, if desired |

### 3.3 Explicitly Not Debts

These were out of scope by charter design and are not considered debts:

- No execution domain breadth (paper_order is sufficient per charter §2.2)
- No signal domain expansion (EMA signal already existed)
- No evidence domain expansion (candle evidence is sufficient)
- No deployment pipeline changes
- No monitoring/observability additions
