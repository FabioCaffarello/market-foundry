# Behavioral Feature Charter and Scope Freeze

**Charter ID:** BEHAVIORAL-WAVE-1
**Opened:** S249
**Status:** OPEN — SCOPE FROZEN
**Type:** Behavioral integration (cross-domain interaction)
**Predecessor charter:** BREADTH-WAVE-1 (S240–S244, PASSED)
**Hardening tranche:** S245–S248 (CONDITIONAL PASS, OD1 pending)
**Governed by:** Charter Amendment Rules (S239), Evolution Playbook

---

## 1. Charter Statement

This charter formally opens the first behavioral wave of market-foundry. The singular objective is to make the existing breadth types **interact across domain boundaries** — transforming independent evaluators and resolvers into coherent cross-domain pipelines that produce meaningful end-to-end trading behavior.

Behavioral integration is the **primary and non-derivable** objective. No stage within this charter may substitute breadth expansion, depth enrichment, or infrastructure work for behavioral delivery without a formal pre-execution amendment.

The breadth wave proved that the system can host multiple types per domain. This wave proves that those types can **compose into real scenarios** where a decision drives a strategy which is gated by risk — and where the output is auditable, traceable, and functionally useful.

---

## 2. Strategic Rationale

### 2.1 Why Behavior, Not Breadth

The system currently has 6 analytical types (2 per domain) that operate in parallel but do not interact intelligently:

- `rsi_oversold` triggers `mean_reversion_entry`, which is gated by `position_exposure`
- `ema_crossover` triggers `trend_following_entry`, which is gated by `drawdown_limit`

These two chains exist as **independent pipelines**. They share infrastructure but not behavior. The next value increment is not a 7th type — it is making the existing 6 types interact in ways that produce richer, more realistic trading scenarios.

### 2.2 Behavioral Gaps

| Gap | Current State | Target State |
|-----|--------------|-------------|
| Decision → Strategy coupling | 1:1 hardcoded (each decision triggers exactly one strategy) | Configurable routing — a decision can inform multiple strategies |
| Strategy → Risk coupling | 1:1 hardcoded (each strategy is gated by exactly one risk evaluator) | Multi-constraint gating — a strategy proposal passes through multiple risk evaluators |
| Cross-chain awareness | Chain A and Chain B are completely independent | Scenarios where both chains contribute to a composite outcome |
| Scenario traceability | Events flow but there is no end-to-end scenario identity | Correlation-based scenario tracing from decision through risk |

### 2.3 Value Proposition

After this wave, the system can answer questions like:

- "When RSI oversold fires AND EMA crossover confirms, what does the combined strategy look like?"
- "When a mean reversion entry is proposed, can it survive BOTH position exposure AND drawdown limit checks?"
- "Given a specific market scenario, what is the full audit trail from signal to risk gate?"

---

## 3. Scope Definition

### 3.1 In Scope — Behavioral Targets

The charter defines three behavioral capability tiers, each building on the previous:

#### Tier 1: Decision → Strategy Integration (P0)

Make the decision→strategy boundary intelligent:

- **Multi-decision strategy input** — A strategy resolver can consume decisions from more than one evaluator type. Example: `trend_following_entry` considers both `ema_crossover` (primary) and `rsi_oversold` (confirmation).
- **Decision correlation** — When multiple decisions fire for the same symbol/timeframe, the strategy layer can observe them as a correlated group.
- **Configurable routing** — The mapping from decision types to strategy resolvers is driven by configuration, not hardcoded fan-out.

#### Tier 2: Strategy → Risk Multi-Gate (P1)

Make the strategy→risk boundary compositional:

- **Multi-evaluator risk gating** — A strategy proposal is assessed by multiple risk evaluators (e.g., both `position_exposure` AND `drawdown_limit`), with the most restrictive constraint winning.
- **Composite risk outcome** — The final risk assessment reflects the intersection of all evaluator constraints, not just one.
- **Constraint aggregation** — When two risk evaluators produce different confidence scalings, the system applies the most conservative result.

#### Tier 3: End-to-End Scenario Proof (P2)

Prove that the full pipeline produces coherent behavior:

- **Scenario tests** — Integration tests that exercise complete chains from signal arrival through risk gate, validating that behavioral composition produces correct and traceable outcomes.
- **Correlation tracing** — Events within a scenario share a correlation ID that allows end-to-end audit.
- **Multi-chain scenarios** — Tests where both Chain A and Chain B fire for the same symbol, and the system produces a coherent combined result.

### 3.2 Out of Scope

The following are **explicitly prohibited** during this charter:

| Prohibition | Rationale |
|-------------|-----------|
| Adding a 3rd evaluator/resolver to any domain | This is breadth — belongs to a future BREADTH-WAVE-2 |
| New signal types (MACD, Bollinger, etc.) | Signal domain expansion is frozen |
| New evidence types | Evidence domain expansion is frozen |
| Execution domain activation | Execution is downstream of this wave; premature |
| New NATS streams or JetStream streams | Behavioral work uses existing streams |
| New ClickHouse tables | Behavioral work uses existing tables |
| New binaries or services | Existing binary topology is sufficient |
| Codegen framework evolution | Codegen is a tool, not a charter objective |
| Marketmonkey absorption | Separate initiative |
| Observability/monitoring infrastructure | Only if directly pulled by a behavioral feature |
| CI pipeline expansion beyond feature-pull | Hardening budget applies |
| Raccoon-cli overhaul | Guardian is stable; only extend if behavioral features require new rules |
| Documentation cleanup wave | Only produce docs for what this charter delivers |

### 3.3 Permitted Supporting Work

- Refactoring actor message routing to support multi-input strategy resolution — documented as prerequisite, not primary deliverable
- Extending the `Envelope` or domain models with correlation metadata — only if required by scenario tracing
- Adding integration test infrastructure for multi-chain scenarios
- Bug fixes discovered during implementation — must not expand scope
- Lightweight hardening subordinate to behavioral feature delivery (≤20% budget)

---

## 4. Domain Interaction Model

### 4.1 Current State (Post-Breadth)

```
Signal (RSI) ──→ Decision (rsi_oversold) ──→ Strategy (mean_reversion_entry) ──→ Risk (position_exposure)
Signal (EMA) ──→ Decision (ema_crossover) ──→ Strategy (trend_following_entry) ──→ Risk (drawdown_limit)
```

Two independent chains. No cross-chain interaction. 1:1 coupling at every boundary.

### 4.2 Target State (Post-Behavioral)

```
Signal (RSI) ──→ Decision (rsi_oversold) ──┐
                                            ├──→ Strategy (mean_reversion_entry) ──┬──→ Risk (position_exposure) ──┐
Signal (EMA) ──→ Decision (ema_crossover) ──┘                                     │                               ├──→ Composite Risk Gate
                                            ┌──→ Strategy (trend_following_entry) ─┤                               │
Signal (EMA) ──→ Decision (ema_crossover) ──┘                                     └──→ Risk (drawdown_limit) ──────┘
```

Multi-input strategies. Multi-evaluator risk gates. Cross-chain scenarios possible.

### 4.3 Message Flow Invariants (Preserved)

- All communication through NATS — no direct function calls between domains
- Single-writer per stream — unchanged
- Envelope uniformity — preserved, extended with correlation metadata if needed
- Acyclic data flow — no feedback loops introduced
- Configctl drives activation — behavioral routing is configuration-driven

---

## 5. Minimum Viable Scenarios

The charter is not complete until the following scenarios are proven:

### Scenario 1: Single-Chain Decision→Strategy (Baseline Validation)

**Purpose:** Confirm that existing 1:1 chains continue to work under the new routing model.

- RSI signal arrives → `rsi_oversold` evaluates → `mean_reversion_entry` resolves → `position_exposure` gates
- EMA signal arrives → `ema_crossover` evaluates → `trend_following_entry` resolves → `drawdown_limit` gates
- Both chains produce correct, traced output

### Scenario 2: Multi-Decision Strategy Input

**Purpose:** Prove that a strategy can consume multiple decision types.

- RSI signal AND EMA signal arrive for the same symbol/timeframe
- `trend_following_entry` receives both `ema_crossover` (primary trigger) and `rsi_oversold` (confirmation signal)
- Strategy resolution reflects the combined decision context
- Output is traceable to both input decisions

### Scenario 3: Multi-Evaluator Risk Gate

**Purpose:** Prove that a strategy proposal survives multiple risk checks.

- A strategy proposal (e.g., `mean_reversion_entry`) is assessed by BOTH `position_exposure` AND `drawdown_limit`
- The composite risk outcome reflects the most restrictive constraint
- Confidence scaling applies the most conservative factor
- Output constraints are the union of individual evaluator constraints

### Scenario 4: Cross-Chain End-to-End

**Purpose:** Prove coherent behavior when both chains fire simultaneously.

- RSI and EMA signals arrive for the same symbol in the same timeframe window
- Both decision chains evaluate
- Both strategy resolvers produce proposals
- Both risk evaluators gate both proposals
- All events share a correlation context allowing full audit trail reconstruction

---

## 6. Success Criteria

The charter is successful when **all** of the following are true:

| # | Criterion | Verification Method |
|---|-----------|-------------------|
| E1 | Multi-decision strategy input works for at least one strategy type | Integration test with 2 decision inputs |
| E2 | Multi-evaluator risk gating works for at least one strategy proposal | Integration test with 2 risk evaluators |
| E3 | All 4 minimum viable scenarios pass | Dedicated scenario integration tests |
| E4 | Existing 1:1 chains continue to work unchanged | Existing integration tests still pass |
| E5 | Correlation tracing enables end-to-end audit | Scenario test validates correlation ID propagation |
| E6 | Behavioral routing is configuration-driven | Routing controlled by configctl, not hardcoded |
| E7 | `make test` and `make test-integration` pass | CI verification |
| E8 | Remote CI green for at least one commit | CI run evidence |
| E9 | No new streams, tables, or binaries introduced | Architecture review |

---

## 7. Non-Success Criteria

The following do **NOT** count as charter progress:

- Adding new evaluator/resolver types without behavioral integration
- Adding domain model fields without corresponding cross-domain logic
- Infrastructure improvements without behavioral feature pull
- Documentation without code backing
- Refactoring without behavioral capability gain
- Test additions without behavioral scenario coverage

---

## 8. Hardening Budget

The charter allocates a **maximum of 20% of stage effort** to hardening activities:

- In a projected 5-stage implementation wave (S250–S254), at most 1 stage equivalent can be pure hardening
- Hardening embedded within feature stages does not count against this budget
- If hardening threatens to exceed 20%, the charter requires a stop-and-reassess
- Hardening is **subordinate to behavioral value** — it must serve the scenarios, not the other way around

---

## 9. Planned Stage Sequence

| Stage | Primary Deliverable | Type | Tier |
|-------|-------------------|------|------|
| S249 | Charter definition and scope freeze | Governance | — |
| S250 | Decision → Strategy multi-input routing | Feature | Tier 1 |
| S251 | Strategy → Risk multi-evaluator gating | Feature | Tier 2 |
| S251.5 | **Mid-charter gate** | Governance | — |
| S252 | End-to-end scenario proof and correlation tracing | Feature | Tier 3 |
| S253 | Behavioral integration gate + hardening closure | Integration | All |

### 9.1 Sequencing Rationale

- **Decision→Strategy first (S250):** The decision→strategy boundary is the entry point for behavioral composition. Multi-input routing here creates richer inputs for everything downstream.
- **Strategy→Risk second (S251):** Multi-evaluator risk gating depends on strategy proposals existing. Benefits from the enriched strategy context delivered in S250.
- **Mid-charter gate (S251.5):** Two of three tiers will have behavioral proof. Evaluates whether to proceed or amend.
- **End-to-end scenarios (S252):** Full chain behavioral proof. Depends on both Tier 1 and Tier 2 being delivered.
- **Final gate (S253):** Binary pass/fail against exit criteria. Optional hardening closure for any behavioral gaps.

---

## 10. Governance Framework

### 10.1 Amendment Rules

All amendments follow the five rules codified in S239:

1. **Pre-execution documentation required** — No scope change may be implemented before an amendment record is written and appended to this charter
2. **Exit criteria explicitly updated** — Amendment must state which criteria change and what replaces them
3. **Mid-charter gate mandatory** — After S251 (Tier 2 delivery), a formal checkpoint evaluates behavioral progress
4. **No retroactive modification** — Amendments append; original charter text is immutable
5. **Post-hoc amendments flagged** — Deviations discovered after execution must be acknowledged and corrected

### 10.2 Amendment Threshold

An amendment is required if any of the following occur:

- A behavioral tier is deferred or dropped
- A minimum viable scenario is changed or removed
- Breadth or depth work exceeds 20% of any single stage's effort
- Implementation order changes from the planned sequence
- A new stream, table, or binary is proposed

### 10.3 Stop Conditions

The charter must be **suspended and reviewed** if:

- Two consecutive stages fail to deliver their primary behavioral deliverable
- A mid-charter gate reveals ≥2 tiers still undelivered
- Hardening consumption exceeds the 20% budget in any stage
- A blocking architectural issue requires redesign affecting >1 domain
- The behavioral work inadvertently introduces new breadth (new types)

---

## 11. Amendments Log

_No amendments recorded. Charter is in its original frozen state._

| # | Date | Description | Criteria Impact | Approved By |
|---|------|-------------|-----------------|-------------|
| — | — | — | — | — |
