# Stage S240 — Breadth Charter and Scope Freeze Report

**Stage:** S240
**Type:** Governance
**Charter:** BREADTH-WAVE-1
**Status:** COMPLETE

---

## 1. Executive Summary

S240 formally opens the first breadth wave of market-foundry. The charter establishes breadth (≥2 evaluator/resolver types per domain) as the primary, non-derivable objective with frozen scope, explicit exit criteria, hardened amendment rules, and automatic stop conditions.

Three second-type targets were selected based on existing infrastructure availability, logic distinctness, and architectural validation value:

| Domain | Existing | Second Target | Delivery Stage |
|--------|----------|---------------|----------------|
| Decision | `rsi_oversold` | `ema_crossover` | S241 |
| Strategy | `mean_reversion_entry` | `trend_following_entry` | S242 |
| Risk | `position_exposure` | `drawdown_limit` | S243 |

No code was written. No new evaluators were implemented. This stage is purely governance.

---

## 2. Deliverables

| # | Deliverable | Path | Status |
|---|------------|------|--------|
| D1 | Breadth charter and scope freeze | `docs/architecture/breadth-charter-and-scope-freeze.md` | Delivered |
| D2 | Breadth targets by domain | `docs/architecture/breadth-targets-by-domain-decision-strategy-risk.md` | Delivered |
| D3 | Exit criteria, amendment rules, stop conditions | `docs/architecture/breadth-exit-criteria-amendment-rules-and-stop-conditions.md` | Delivered |
| D4 | Stage report (this document) | `docs/stages/stage-s240-breadth-charter-and-scope-freeze-report.md` | Delivered |

---

## 3. Key Decisions

### 3.1 Why `ema_crossover` for Decision

- EMA signal family already defined (`codegen/families/ema.yaml`) and sampler actor exists
- Consumes a different signal source than `rsi_oversold` (EMA vs RSI) — proves `SignalInput` abstraction
- Crossover detection is fundamentally different logic from threshold comparison
- Rejected alternatives: volume spike (requires new signal type), price momentum (overlaps with EMA), multi-timeframe RSI (depth, not breadth)

### 3.2 Why `trend_following_entry` for Strategy

- Natural downstream consumer of EMA crossover decisions
- Opposite trading philosophy from mean reversion: enters *with* the trend, not *against* it
- Uses trailing stops instead of fixed targets — distinct parameter model
- Rejected alternatives: breakout entry (needs price-level data), momentum continuation (overlaps)

### 3.3 Why `drawdown_limit` for Risk

- Orthogonal risk dimension: portfolio drawdown vs individual position sizing
- Circuit-breaker pattern (reject all trades above threshold) is distinct from position capping
- No new infrastructure required — existing `Disposition` enum covers all outcomes
- Rejected alternatives: correlation exposure (needs multi-asset data), stop-loss optimizer (overlaps with position_exposure)

---

## 4. Governance Hardening (vs S233–S237 Wave)

| Aspect | Previous Wave | This Charter |
|--------|---------------|-------------|
| Amendment rules | Implicit | Five explicit rules, codified |
| Scope freeze | Documented but drifted | Frozen with mandatory amendment triggers |
| Mid-charter gate | None | Mandatory after S242 |
| Stop conditions | None | Five automatic suspension triggers |
| Depth budget | Unlimited | ≤20% per stage, monitored |
| Partial exit | Allowed informally | Explicitly prohibited |
| Breadth definition | Ambiguous | Binding definition with non-examples |

---

## 5. Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| EMA signal not fully wired | Low | Sampler actor exists; YAML defined; gap is only decision evaluator |
| Fan-out routing complexity | Medium | Addressed as permitted prerequisite work in S241 |
| Drawdown state management | Medium | Can use stateless per-assessment approach initially |
| Depth creep | Medium | 20% budget cap + amendment trigger |
| Scope expansion pressure | Low | Explicit out-of-scope list + stop conditions |

---

## 6. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Charter formally opened and scope frozen | PASS |
| Second types per domain explicitly chosen | PASS — ema_crossover, trend_following_entry, drawdown_limit |
| Breadth is non-derivable exit criterion | PASS — nine explicit criteria, all required |
| Amendment rules explicit and hardened | PASS — five rules + six amendment triggers |
| Base ready for S241 decision expansion | PASS — EMA signal infrastructure exists |

---

## 7. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No new types/evaluators implemented | PASS — zero code changes |
| No new analytical families opened | PASS |
| Breadth not "intention only" | PASS — binding exit criteria with measurement matrix |
| Amendment not implicit | PASS — explicit rules, triggers, and record format |
| Out-of-scope documented | PASS — explicit exclusion list |

---

## 8. Preparation for S241

S241 will implement the `ema_crossover` decision evaluator. Prerequisites confirmed:

1. **EMA signal family YAML** — exists at `codegen/families/ema.yaml`
2. **EMA sampler actor** — exists at `internal/actors/scopes/derive/ema_crossover_signal_sampler_actor.go`
3. **Decision domain structs** — `SignalInput`, `Decision`, `Outcome`, `Severity` all support a second evaluator without modification
4. **Actor chain** — `decision_evaluator_actor.go` may need fan-out routing for a second evaluator type; this is permitted prerequisite work
5. **Test patterns** — `rsi_oversold_evaluator_test.go` provides the template for `ema_crossover_evaluator_test.go`

**S241 scope:** Implement `EMA CrossoverEvaluator` with domain validation, application logic, actor wiring, codegen family YAML, and full test coverage. No other work.
