# Paper Execution — Permitted vs Prohibited Changes

Charter: PAPER-EXECUTION-WAVE-1
Stage: S264
Date: 2026-03-21
Status: Active

---

## 1. Purpose

This document defines the boundary between changes that are allowed and changes that are prohibited during the Paper Execution wave (S264–S268). The goal is to ensure that work stays focused on proving the operational loop in paper mode without expanding scope into venue real, OMS, portfolio, or new infrastructure.

## 2. Permitted Changes

### 2.1 Test Infrastructure

| Change | Condition |
|--------|-----------|
| New integration tests for paper execution loop | Must target one of the 7 minimum viable scenarios |
| New test fixtures for execution intents and fills | Must use existing domain types; no new domain models |
| Test helpers for safety gate and kill switch assertions | Must be scoped to `execution` package |
| Scenario-based end-to-end tests spanning decision → fill | Must use existing actor wiring; no new actors |

### 2.2 Existing Code Modifications

| Change | Condition |
|--------|-----------|
| Bug fixes in paper execution components | Must be discovered during scenario implementation |
| Wiring corrections in execution actors | Must be required to close the paper loop |
| Additional assertions in existing execution tests | Must strengthen scenario coverage |
| Minor refactoring within execution application layer | Must be required by a Tier 1 scenario; no speculative cleanup |

### 2.3 Documentation

| Change | Condition |
|--------|-----------|
| Architecture docs for paper execution findings | Must document observed behavior, not proposed features |
| Stage reports for S265–S268 | Required deliverables |
| Updates to execution domain design doc | Only if existing doc is factually incorrect |

### 2.4 Configuration

| Change | Condition |
|--------|-----------|
| Test configuration for paper execution scenarios | Must use existing settings schema; no new config keys |
| Feature flag or settings for paper mode enforcement | Only if required to prevent accidental real venue activation |

## 3. Prohibited Changes

| Category | Prohibition | Rationale |
|----------|------------|-----------|
| Real venue | No real exchange API calls, testnet connections, or venue credentials in test paths | Paper mode only; venue real is a future wave |
| OMS | No order management, order state machines beyond submitted/filled, or order lifecycle tracking | OMS is out of scope; paper fills are instant |
| Portfolio | No position aggregation, PnL calculation, or balance tracking | Portfolio is downstream of proven loop |
| Multi-venue | No venue routing, venue selection, or venue abstraction extensions | Single paper venue only |
| New domain surfaces | No new signal families, strategy resolvers, or risk evaluators | Breadth is frozen |
| New NATS infrastructure | No new streams, consumers, KV buckets, or subject hierarchies | Existing infrastructure sufficient |
| ClickHouse schema | No new tables, columns, or migrations for execution data | Existing writer pipeline sufficient |
| New actors | No new actor types in derive or store scope | Existing actors sufficient for paper loop |
| Performance work | No benchmarks, optimizations, or latency reduction efforts | Observation only; optimization is a future concern |
| New dependencies | No new Go modules, libraries, or external packages | Existing dependencies sufficient |
| Codegen expansion | No new codegen-governed artifacts beyond current 22 | Codegen wave is complete (S263) |
| Real money | No real orders, real positions, real funds, or real risk | Explicitly and permanently prohibited in this wave |

## 4. Decision Boundary

| Artifact type | Owner | Rationale |
|--------------|-------|-----------|
| Paper execution test scenarios | Human (this wave) | Scenarios require domain judgment about what to prove |
| Execution actor wiring fixes | Human (this wave) | Wiring corrections require understanding of actor lifecycle |
| Codegen-governed execution artifacts | Codegen pipeline | Existing governed artifacts remain under codegen control |
| Domain model changes | NOT ALLOWED | Domain models are frozen; no new types or fields |
| New execution events | NOT ALLOWED | Event schema is frozen; existing events sufficient |

## 5. Escalation Rules

1. If a scenario requires a change not listed in Permitted Changes, **stop and assess** before proceeding.
2. If a bug fix touches code outside `internal/application/execution/` or `internal/actors/scopes/*/execution_*`, escalate to charter review.
3. If any test imports a real venue adapter or makes a network call, this is a **hard stop** — revert and reassess.
4. If codegen equivalence checks fail after any change, **pause the wave** and fix drift before continuing.
5. If a Tier 1 scenario cannot be proven without a prohibited change, the scenario must be redesigned or deferred — not the prohibition lifted.

## 6. Decision Tree

```
Is the change adding a new test for a minimum viable scenario?
  ├── YES → Does it use only existing domain types and actors?
  │         ├── YES → PERMITTED
  │         └── NO  → PROHIBITED (redesign the test)
  └── NO  → Is the change fixing a bug discovered during scenario work?
            ├── YES → Is the fix within execution package boundaries?
            │         ├── YES → PERMITTED
            │         └── NO  → ESCALATE to charter review
            └── NO  → Is the change documentation?
                      ├── YES → PERMITTED (if factual, not speculative)
                      └── NO  → PROHIBITED
```

## 7. Audit Trail Requirements

1. Every permitted change must reference the Tier 1 scenario it serves
2. Every bug fix must include a test that demonstrates the bug and its resolution
3. No change may be merged without CI green on all existing tests + codegen equivalence
4. Stage reports must document which permitted changes were made and why
5. Any near-boundary decision must be recorded in the charter amendments log
