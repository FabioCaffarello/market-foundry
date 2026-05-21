# Breadth Charter and Scope Freeze

**Charter ID:** BREADTH-WAVE-1
**Opened:** S240
**Status:** OPEN — SCOPE FROZEN
**Type:** Breadth (non-derivable)
**Governed by:** Charter Amendment Rules (S239)

---

## 1. Charter Statement

This charter formally opens the first breadth wave of market-foundry. The singular objective is to achieve **≥2 evaluator/resolver types per domain** across Decision, Strategy, and Risk — delivering genuinely distinct evaluation logic, not depth enrichment of existing types.

Breadth is the **primary and non-derivable** objective. No stage within this charter may substitute depth work for breadth delivery without a formal pre-execution amendment.

---

## 2. Scope Definition

### 2.1 In Scope

| Domain | Existing Type | Second Type (Target) | Layer |
|--------|--------------|---------------------|-------|
| Decision | `rsi_oversold` | `ema_crossover` | `internal/application/decision/` |
| Strategy | `mean_reversion_entry` | `trend_following_entry` | `internal/application/strategy/` |
| Risk | `position_exposure` | `drawdown_limit` | `internal/application/risk/` |

Each second type requires:
- Domain constants/validation in `internal/domain/<domain>/`
- Pure application-layer evaluator/resolver in `internal/application/<domain>/`
- Codegen family YAML in `codegen/families/`
- Actor integration (fan-out, publisher messages) in `internal/actors/`
- Unit tests at domain, application, and actor layers
- Chain integration test extension

### 2.2 Out of Scope

The following are **explicitly prohibited** during this charter:

- Adding a third evaluator/type to any domain
- Enriching existing evaluators (severity expansion, metadata additions, confidence recalibration) — this is depth, not breadth
- Opening new analytical families beyond the three targets above
- Infrastructure expansion (CI pipeline changes, new deployment targets, monitoring additions) unless directly blocking a breadth deliverable
- Codegen framework evolution or generalization
- Execution domain expansion (paper_order is sufficient; no second execution type)
- Signal domain expansion (EMA signal already exists; no new signal types needed)
- Evidence domain expansion (candle evidence is sufficient)

### 2.3 Permitted Supporting Work

- Minimal refactoring required to support fan-out (e.g., actor message routing for multiple evaluator types) — must be documented as prerequisite, not primary deliverable
- Test infrastructure additions strictly required by new evaluator types
- Bug fixes discovered during implementation — must not expand scope

---

## 3. Governance Framework

### 3.1 Amendment Rules

All amendments follow the five rules codified in S239:

1. **Pre-execution documentation required** — No scope change may be implemented before an amendment record is written and appended to this charter
2. **Exit criteria explicitly updated** — Amendment must state which criteria change and what replaces them
3. **Mid-charter gate mandatory** — After S242 (strategy resolver delivery), a formal checkpoint evaluates breadth progress against original criteria
4. **No retroactive modification** — Amendments append; original charter text is immutable
5. **Post-hoc amendments flagged** — If a deviation is discovered after execution, it must be acknowledged, explained, and corrective action documented

### 3.2 Amendment Threshold

An amendment is required if any of the following occur:
- A target evaluator/resolver is changed to a different candidate
- A domain's second type is deferred or dropped
- Depth work exceeds 20% of any single stage's effort
- Implementation order changes from the planned sequence

### 3.3 Stop Conditions

The charter must be **suspended and reviewed** if:
- Two consecutive stages fail to deliver their primary breadth deliverable
- A mid-charter gate reveals ≥2 domains still at single-type coverage
- Depth work consumption exceeds the 20% hardening budget in any stage
- A blocking architectural issue requires redesign affecting >1 domain

---

## 4. Planned Stage Sequence

| Stage | Primary Deliverable | Type | Domain |
|-------|-------------------|------|--------|
| S240 | Charter definition and scope freeze | Governance | — |
| S241 | `ema_crossover` decision evaluator | Feature | Decision |
| S242 | `trend_following_entry` strategy resolver | Feature | Strategy |
| S242.5 | **Mid-charter gate** | Governance | — |
| S243 | `drawdown_limit` risk evaluator | Feature | Risk |
| S244 | Breadth pipeline proof + chain integration | Integration | All |
| S245 | Breadth gate evaluation | Governance | All |

### 4.1 Sequencing Rationale

- **Decision first (S241):** EMA signal infrastructure already exists; decision is the chain entry point; delivers a second signal→decision path that subsequent stages consume.
- **Strategy second (S242):** Depends on having a second decision type to resolve; validates fan-out at the decision→strategy boundary.
- **Mid-charter gate (S242.5):** Two of three domains will have breadth; evaluates whether to proceed or amend.
- **Risk third (S243):** Depends on having a second strategy type to assess; validates fan-out at the strategy→risk boundary.
- **Integration proof (S244):** Full chain with two parallel paths exercised end-to-end.
- **Final gate (S245):** Binary pass/fail against exit criteria.

---

## 5. Breadth Definition (Binding)

For the purposes of this charter, **breadth** means:

> A new evaluator/resolver type with a **distinct type name**, **distinct evaluation logic**, and **distinct input interpretation** — consuming either a different signal source or applying a fundamentally different analytical model to the same source.

The following do **NOT** constitute breadth:
- Adding fields to an existing domain struct
- Adding severity levels or metadata to an existing evaluator
- Expanding test coverage without new feature logic
- Infrastructure improvements without feature pull
- Documentation without corresponding code delivery

---

## 6. Amendments Log

_No amendments recorded. Charter is in its original frozen state._

| # | Date | Description | Criteria Impact | Approved By |
|---|------|-------------|-----------------|-------------|
| — | — | — | — | — |
