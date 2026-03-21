# Behavioral Wave — Permitted vs Prohibited Changes

**Charter:** BEHAVIORAL-WAVE-1
**Stage:** S249
**Date:** 2026-03-21
**Status:** Active

---

## 1. Purpose

This document provides an explicit, auditable list of what changes are allowed and what changes are forbidden during the BEHAVIORAL-WAVE-1 charter. Every stage within the charter must be evaluated against this list before execution.

---

## 2. Permitted Changes

### 2.1 Feature Work (Primary)

| Change | Condition | Example |
|--------|-----------|---------|
| Multi-input strategy resolver logic | Must consume ≥2 decision types | `trend_following_entry` receives both `ema_crossover` and `rsi_oversold` |
| Multi-evaluator risk gating logic | Must apply ≥2 risk evaluators to one proposal | A proposal gated by both `position_exposure` and `drawdown_limit` |
| Decision correlation grouping | Must group decisions by symbol/timeframe for strategy consumption | Correlated decision batch for same `btcusdt.60` window |
| Composite risk outcome assembly | Must merge constraints from multiple evaluators | Union of `MaxPositionSize`, `MaxExposure`, `StopDistance` constraints |
| Configurable routing maps | Must replace hardcoded 1:1 decision→strategy and strategy→risk coupling | Configuration document defining which decisions feed which strategies |
| Correlation ID propagation | Must enable end-to-end tracing through the decision→strategy→risk chain | `CorrelationID` field in `Envelope` carried across domain boundaries |
| Scenario integration tests | Must exercise full cross-domain behavioral chains | Tests proving Scenarios 1–4 from the charter |

### 2.2 Domain Model Changes (Feature-Pulled)

| Change | Condition |
|--------|-----------|
| Add correlation metadata to existing domain events | Only if required by scenario tracing |
| Extend `Decision` model with grouping/batch semantics | Only if required by multi-decision strategy input |
| Extend `RiskAssessment` model with composite fields | Only if required by multi-evaluator gating |
| Add routing configuration to configctl documents | Only if behavioral routing requires it |

### 2.3 Actor/Adapter Changes (Feature-Pulled)

| Change | Condition |
|--------|-----------|
| Modify strategy scope actors to accept multiple decision sources | Must be required by Tier 1 behavioral target |
| Modify risk scope actors to fan-in from multiple evaluators | Must be required by Tier 2 behavioral target |
| Adjust NATS consumer filter subjects for multi-source input | Must not create new streams or consumers beyond existing stream families |
| Extend publisher messages with correlation metadata | Must preserve envelope uniformity |

### 2.4 Hardening (≤20% Budget)

| Change | Condition |
|--------|-----------|
| Integration test extensions | Must trace to a specific behavioral scenario |
| Smoke test updates for behavioral scenarios | Must validate cross-domain interaction |
| Codegen golden snapshot updates | Only if behavioral routing changes registered families |
| Raccoon-cli rule extensions | Only if behavioral work introduces patterns that need enforcement |
| Bug fixes discovered during implementation | Must not expand scope |

---

## 3. Prohibited Changes

### 3.1 Breadth Expansion (Hard Block)

| Prohibition | Rationale |
|-------------|-----------|
| Adding a 3rd evaluator type to Decision (`volume_spike`, `momentum`, etc.) | Breadth is frozen — belongs to BREADTH-WAVE-2 |
| Adding a 3rd resolver type to Strategy (`breakout_entry`, `momentum_continuation`, etc.) | Breadth is frozen — belongs to BREADTH-WAVE-2 |
| Adding a 3rd evaluator type to Risk (`correlation_exposure`, etc.) | Breadth is frozen — belongs to BREADTH-WAVE-2 |
| Adding new Signal types (MACD, Bollinger, etc.) | Signal domain frozen for this charter |
| Adding new Evidence types | Evidence domain frozen for this charter |

### 3.2 Depth Enrichment (Hard Block Unless Feature-Pulled)

| Prohibition | Rationale |
|-------------|-----------|
| Adding severity levels to existing evaluators | Depth, not behavior |
| Adding metadata fields to existing evaluators without behavioral justification | Depth, not behavior |
| Recalibrating confidence formulas without behavioral justification | Depth, not behavior |
| Expanding the evaluation logic of existing types beyond what multi-input requires | Depth, not behavior |

### 3.3 Infrastructure Expansion (Hard Block)

| Prohibition | Rationale |
|-------------|-----------|
| Creating new NATS JetStream streams | Behavioral work uses existing streams |
| Creating new ClickHouse tables | Behavioral work uses existing tables |
| Creating new binaries or services | Existing binary topology is sufficient |
| Adding new deployment targets (Docker images, Helm charts) | Infrastructure frozen |
| Adding monitoring/observability systems | Only if directly pulled by a behavioral feature |
| Adding new message broker infrastructure | NATS is sufficient |

### 3.4 Platform/Tooling (Hard Block)

| Prohibition | Rationale |
|-------------|-----------|
| Codegen framework evolution or generalization | Codegen is a tool, not a charter objective |
| Raccoon-cli overhaul or major feature additions | Guardian is stable |
| CI pipeline structural changes (new jobs, new runners) | Only extend if directly blocking a behavioral deliverable |
| Marketmonkey absorption | Separate initiative with its own charter |
| Go module restructuring | Structural, not behavioral |

### 3.5 Documentation (Soft Block)

| Prohibition | Rationale |
|-------------|-----------|
| Documentation cleanup wave | Only produce docs for what this charter delivers |
| Architecture document overhaul | Only update docs affected by behavioral changes |
| Playbook revision | Only amend if behavioral patterns necessitate new rules |

---

## 4. Decision Tree for Ambiguous Cases

When a proposed change does not clearly fall into permitted or prohibited:

```
1. Does this change directly enable one of the 4 minimum viable scenarios?
   → YES: Permitted (document the linkage in the stage report)
   → NO: Continue to 2

2. Does this change add a new evaluator/resolver type?
   → YES: PROHIBITED (breadth expansion)
   → NO: Continue to 3

3. Does this change create a new stream, table, or binary?
   → YES: PROHIBITED (infrastructure expansion) — requires formal amendment
   → NO: Continue to 4

4. Does this change modify existing evaluator logic beyond what multi-input requires?
   → YES: PROHIBITED (depth enrichment) — unless amendment filed
   → NO: Continue to 5

5. Is this change ≤20% of the stage's effort?
   → YES: Permitted as hardening (document as such)
   → NO: PROHIBITED — exceeds hardening budget
```

---

## 5. Audit Trail Requirements

Every stage in this charter must include in its report:

1. A table listing all changes made, classified as Feature/Hardening/Supporting
2. Confirmation that no prohibited changes were introduced
3. If any change required the decision tree: the reasoning path and conclusion
4. If hardening effort approached 20%: explicit measurement and justification
