# Stage S249 — Behavioral Feature Charter and Scope Freeze Report

**Date:** 2026-03-21
**Type:** Charter definition (non-implementation)
**Charter:** BEHAVIORAL-WAVE-1
**Predecessor:** S248 (post-breadth hardening gate — CONDITIONAL PASS)
**Status:** COMPLETE

---

## 1. Executive Summary

S249 formally opens the BEHAVIORAL-WAVE-1 charter, transitioning market-foundry from breadth expansion to cross-domain behavioral integration. The charter freezes scope around making the 6 existing analytical types (2 per domain across decision, strategy, and risk) interact intelligently across domain boundaries.

The central objective is **behavior, not breadth**: the system already has enough types. The next value increment comes from making those types compose into realistic, auditable trading scenarios.

Three behavioral tiers are defined:
1. **Decision → Strategy multi-input** — strategies consuming multiple decision types
2. **Strategy → Risk multi-gate** — proposals assessed by multiple risk evaluators
3. **End-to-end scenario proof** — full chain behavioral validation with correlation tracing

Implementation is projected across S250–S253, with a mid-charter gate after Tier 2.

---

## 2. What S249 Delivered

### 2.1 Charter Document

`docs/architecture/behavioral-feature-charter-and-scope-freeze.md`

Defines:
- Charter statement and strategic rationale
- Three behavioral tiers (P0, P1, P2)
- In-scope vs out-of-scope boundaries
- Domain interaction model (current → target)
- 4 minimum viable scenarios
- 12 exit criteria
- Planned stage sequence (S250–S253)
- Governance framework (amendments, stop conditions)
- Hardening budget (≤20%)

### 2.2 Permitted vs Prohibited Changes

`docs/architecture/behavioral-wave-permitted-vs-prohibited-changes.md`

Provides:
- Explicit permitted changes table (feature, domain model, actor/adapter, hardening)
- Explicit prohibited changes table (breadth, depth, infrastructure, platform, documentation)
- Decision tree for ambiguous cases
- Audit trail requirements for every stage

### 2.3 Entry/Exit/Amendment/Stop Conditions

`docs/architecture/behavioral-wave-entry-exit-amendment-and-stop-conditions.md`

Codifies:
- 6 entry conditions (4 met, 1 pending OD1, 1 this stage)
- 12 exit criteria with verification methods
- Partial exit conditions
- Mid-charter gate protocol
- Amendment rules and process
- 7 stop conditions with severity levels
- Governance chain from S239 through S249

---

## 3. Charter Rationale

### 3.1 Why Behavior Now

The breadth wave (S240–S244) delivered 6 analytical types across 3 domains. The hardening tranche (S245–S248) proved them operationally. But the types operate in isolated 1:1 chains:

```
Chain A: rsi_oversold → mean_reversion_entry → position_exposure
Chain B: ema_crossover → trend_following_entry → drawdown_limit
```

No cross-chain interaction. No multi-input. No composite gating. The next value gain is not a 7th type — it is making the existing 6 types interact meaningfully.

### 3.2 Why Not Breadth

Adding more types would increase breadth linearly but not compositional value. Two types per domain already prove the FamilyProcessor pattern scales. A 3rd type adds mechanical value; cross-domain behavior adds functional value.

### 3.3 Why Not Infrastructure

The infrastructure is stable. No new streams, tables, or binaries are needed. The behavioral work lives entirely in application-layer logic and actor routing — exactly where domain value belongs.

---

## 4. Scope Freeze Summary

### 4.1 Frozen In

| Capability | Tier | Stages |
|-----------|------|--------|
| Multi-decision strategy input | P0 | S250 |
| Multi-evaluator risk gating | P1 | S251 |
| End-to-end scenario proof | P2 | S252 |
| Integration gate + hardening | — | S253 |

### 4.2 Frozen Out

| Category | Items |
|----------|-------|
| Breadth | No 3rd types in any domain |
| Depth | No enrichment of existing evaluator logic |
| Infrastructure | No new streams, tables, or binaries |
| Platform | No codegen evolution, raccoon-cli overhaul, module restructuring |
| External | No marketmonkey absorption, no execution domain |

---

## 5. Entry Gate Status

| # | Condition | Status |
|---|-----------|--------|
| EC1 | Breadth charter PASSED | DONE |
| EC2 | Hardening gate PASSED/CONDITIONAL | DONE |
| EC3 | OD1 closed (remote CI green) | **PENDING** |
| EC4 | Charter document accepted | DONE (this stage) |
| EC5 | Test pyramid green | DONE |
| EC6 | No blocking debts | DONE |

**Entry verdict: CONDITIONAL — S250 begins after EC3 (OD1 closure).**

---

## 6. Minimum Viable Scenarios

| # | Scenario | Purpose | Tier |
|---|----------|---------|------|
| S1 | Single-chain baseline | Confirm existing chains survive routing changes | Tier 1 |
| S2 | Multi-decision input | Strategy consumes 2 decision types | Tier 1 |
| S3 | Multi-evaluator risk | Proposal gated by 2 risk evaluators | Tier 2 |
| S4 | Cross-chain end-to-end | Both chains fire for same symbol, coherent combined result | Tier 3 |

---

## 7. Preparation Recommendations for S250

### 7.1 Immediate (Before S250)

1. **Close OD1** — Commit S246–S247, push, verify remote CI green. This is the only blocker.
2. **Review strategy resolver interface** — Understand how `trend_following_entry` and `mean_reversion_entry` currently receive decision input. Identify the coupling point that needs to become multi-input.
3. **Review actor routing** — Map the current actor topology in `derive_supervisor.go` for strategy scope actors. Identify where fan-out from decision to strategy is hardcoded.

### 7.2 Design Considerations for S250

- **Configctl integration** — The routing map (which decisions feed which strategies) should be configuration-driven. Design the config document schema before implementing routing logic.
- **Backward compatibility** — The 1:1 routing must continue to work as the default. Multi-input is an opt-in behavioral upgrade.
- **Correlation ID** — Decide whether correlation tracing requires an `Envelope` change or can be implemented with existing metadata fields. Prefer minimal model changes.
- **Test strategy** — Plan integration tests for Scenario 2 before writing feature code. TDD the behavioral contract.

### 7.3 What S250 Should NOT Do

- Do not touch risk domain logic — that's S251
- Do not attempt end-to-end scenarios — that's S252
- Do not add new evaluator types
- Do not refactor beyond what multi-input routing requires

---

## 8. Files Delivered

| Deliverable | Path |
|-------------|------|
| Charter and scope freeze | `docs/architecture/behavioral-feature-charter-and-scope-freeze.md` |
| Permitted vs prohibited changes | `docs/architecture/behavioral-wave-permitted-vs-prohibited-changes.md` |
| Entry/exit/amendment/stop conditions | `docs/architecture/behavioral-wave-entry-exit-amendment-and-stop-conditions.md` |
| This report | `docs/stages/stage-s249-behavioral-feature-charter-and-scope-freeze-report.md` |

### No Production Code Changes

S249 is a governance stage. No production code was modified.

---

## 9. Charter Opening

**BEHAVIORAL-WAVE-1 is hereby opened with scope frozen.**

- Charter opened: S249
- Projected closure: S253
- Projected duration: 5 stages (S249–S253)
- Behavioral tiers: 3
- Minimum viable scenarios: 4
- Exit criteria: 12
- Hardening budget: ≤20%
- Entry gate: CONDITIONAL (pending OD1)

The next step is to close OD1 and begin S250 (Decision → Strategy multi-input routing).

---

## 10. Status: COMPLETE

The behavioral feature charter is formally defined, scoped, and frozen. Implementation begins at S250 upon entry gate clearance.
