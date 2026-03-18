# Risk Entry Prerequisites

> Concrete conditions that must be satisfied before a `risk` domain can be designed and implemented in Market Foundry.
> Date: 2026-03-18 | Stage: S59

## Purpose

This document defines the prerequisites for opening a `risk` layer. Each prerequisite has a status, acceptance criteria, and clear rationale. Prerequisites are ordered by criticality.

---

## P-1: Strategy Domain Maturity

**Status**: MET

**Requirement**: Strategy domain must be fully implemented, tested, and governed before any downstream consumer (`risk`) can be designed.

**Current state**: Strategy domain is production-ready:
- Domain model: 14 tests, direction enum, DecisionInput provenance
- Application: MeanReversionEntryResolver (pure, 8 tests), GetLatestStrategyUseCase (5 tests)
- Adapters: Registry (8 tests), KV store (8 tests), publisher, consumer, gateway
- Store projection: 21 tests including multi-symbol isolation
- HTTP: Handler (10 tests), routes (3 tests)
- Config: `pipeline.strategy_families` with dependency chain validation
- Governance: STD-1..STD-5 drift rules, CLI guardrails documented

**Acceptance criteria**:
- [x] Strategy domain model complete with validation and events
- [x] At least one strategy family implemented end-to-end
- [x] Strategy projection actor tested with three-gate pattern
- [x] Strategy config dependency chain validated at startup
- [x] Strategy drift rules defined in raccoon-cli

---

## P-2: Strategy Governance Verified

**Status**: MET

**Requirement**: Raccoon-CLI must have strategy-specific drift rules that pass against the current codebase.

**Current state**: Five strategy drift rules (STD-1..STD-5) are defined in `tools/raccoon-cli/src/analyzers/drift_detect.rs`. Strategy guardrails documented in `docs/tooling/cli-strategy-guardrails.md` and `docs/tooling/cli-strategy-drift-rules.md`.

**Acceptance criteria**:
- [x] STD-1: Strategy architecture docs exist
- [x] STD-2: Strategy adapter files present (registry, publisher, consumer, gateway, KV store)
- [x] STD-3: Strategy domain and application files present
- [x] STD-4: Config symmetry between derive and store for strategy families
- [x] STD-5: Strategy runtime contracts (streams, durables, query subjects)

---

## P-3: Config Dependency Chain Complete

**Status**: MET

**Requirement**: The full dependency chain from observation through strategy must be validated at startup. Adding `risk` must be a mechanical extension of this chain.

**Current state**: `ValidatePipeline()` in `internal/shared/settings/schema.go` enforces:
- Signal requires evidence (candle)
- Decision requires signal (rsi)
- Strategy requires decision (rsi_oversold)

**Acceptance criteria**:
- [x] Dependency maps defined for all current layers
- [x] Unknown family names rejected at validation time
- [x] Binary refuses to start with invalid dependency chain
- [x] Pattern is mechanically extensible (add `riskDependsOnStrategy` map)

---

## P-4: Projection Authority Model Consistent

**Status**: MET

**Requirement**: Single-writer invariant must hold for all existing domains. Store must be the sole projection authority.

**Current state**: Every KV bucket has exactly one writer actor. Gateway is stateless. Documented in stream-ownership-matrix.md and projection-family-matrix.md.

**Acceptance criteria**:
- [x] No dual-writer buckets
- [x] Gateway has no write access to any KV bucket
- [x] Monotonicity guards on all KV stores
- [x] Three-gate projection pattern applied consistently

---

## P-5: Query Surfaces Clean and Tested

**Status**: MET

**Requirement**: All existing HTTP endpoints must be conditionally registered, tested, and follow the established pattern that `risk` will also follow.

**Current state**: 9 HTTP endpoints across evidence, signal, decision, and strategy — all with handler and route tests, all conditionally registered based on available use cases.

**Acceptance criteria**:
- [x] All endpoints tested (handler + route level)
- [x] Conditional registration prevents ghost routes
- [x] Consistent pattern: `GET /{domain}/{type_or_family}/{operation}`

---

## P-6: Mesh Integrity Verified

**Status**: MET

**Requirement**: All JetStream streams must have single publishers, documented consumers, and consistent naming.

**Current state**: 5 streams (OBSERVATION_EVENTS, EVIDENCE_EVENTS, SIGNAL_EVENTS, DECISION_EVENTS, STRATEGY_EVENTS) — all healthy, single-writer, consumed by downstream services.

**Acceptance criteria**:
- [x] No orphan streams
- [x] No multi-writer streams
- [x] Naming follows `{DOMAIN}_EVENTS` convention
- [x] Raccoon-cli verifies stream registry consistency

---

## P-7: Adapter Test Coverage Baseline

**Status**: NOT MET

**Requirement**: Publisher and consumer adapters should have unit tests before adding a 6th domain that will also lack them.

**Current state**: **Zero** publisher or consumer adapter tests exist across all 5 domains. This debt compounds with each new domain.

**Acceptance criteria**:
- [ ] At least one publisher adapter has unit tests (pattern established)
- [ ] At least one consumer adapter has unit tests (pattern established)
- [ ] Test patterns documented for replication across domains

**Remediation**: Dedicate a hardening stage (S60) to establish adapter test patterns.

---

## P-8: Derive Actor Test Coverage

**Status**: NOT MET

**Requirement**: Derive scope actors (samplers, evaluators, resolvers) should have unit tests verifying message flow and error handling.

**Current state**: **Zero** derive actor test files exist. All derive actors are tested only indirectly through integration.

**Acceptance criteria**:
- [ ] At least one derive actor scope has unit tests (pattern established)
- [ ] Actor message flow verified (receive → process → send)
- [ ] Error paths tested (invalid input, publish failure)

**Remediation**: Dedicate a hardening stage (S61) to establish derive actor test patterns.

---

## P-9: Risk Domain Design Document

**Status**: NOT MET (expected — this is the design prerequisite)

**Requirement**: Before implementation, a `risk-domain-design.md` must exist following the `strategy-domain-design.md` pattern (18+ sections covering identity, boundaries, invariants, activation, projection, queries, and deferred items).

**Current state**: Does not exist yet. This is the output of the design stage (S62).

**Acceptance criteria**:
- [ ] risk-domain-design.md exists with all canonical sections
- [ ] Domain boundary invariants defined (RBI-1..RBI-N)
- [ ] What-is / what-is-not boundaries clear
- [ ] Binary placement justified (derive vs. separate)
- [ ] Activation model documented
- [ ] Deferred items listed

---

## P-10: Risk Governance Rules in Raccoon-CLI

**Status**: NOT MET (expected — this follows design)

**Requirement**: Raccoon-CLI must have risk-specific drift rules before risk implementation begins.

**Current state**: No risk drift rules exist. Expected to be created in S63.

**Acceptance criteria**:
- [ ] RD-1..RD-5 drift rules defined
- [ ] Risk guardrails documented
- [ ] Known risk families added to drift-detect constants
- [ ] Risk adapter/domain/actor file expectations defined

---

## Summary

| Prerequisite | Status | Blocking for Design? | Blocking for Implementation? |
|--------------|--------|---------------------|------------------------------|
| P-1: Strategy maturity | MET | — | — |
| P-2: Strategy governance | MET | — | — |
| P-3: Config dependency chain | MET | — | — |
| P-4: Projection authority | MET | — | — |
| P-5: Query surfaces | MET | — | — |
| P-6: Mesh integrity | MET | — | — |
| P-7: Adapter test coverage | NOT MET | No | Yes (recommended) |
| P-8: Derive actor tests | NOT MET | No | Yes (recommended) |
| P-9: Risk domain design | NOT MET | N/A (is the design) | Yes |
| P-10: Risk governance rules | NOT MET | No | Yes |

**Conclusion**: 6 of 10 prerequisites are met. The 4 unmet prerequisites split into:
- **2 hardening prerequisites** (P-7, P-8): Recommended before implementation but not blocking for design
- **2 sequential prerequisites** (P-9, P-10): Expected outputs of pre-implementation stages
