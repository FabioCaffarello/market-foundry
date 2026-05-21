# Next-Wave Strategic Options Matrix

**Stage:** S281 — Post-Operational-Proof Feature Gate
**Date:** 2026-03-21
**Scope:** Comparative evaluation of candidate directions for the next major wave

---

## Evaluation Criteria

Each candidate is scored on six dimensions (1–5, where 5 = best):

| Criterion | Definition |
|-----------|-----------|
| **Reuse** | How much of the existing infrastructure, domain model, and codegen surface does this wave leverage? |
| **Architectural Pressure** | Does the wave pressure the architecture in a useful, revealing direction? |
| **Domain Value** | Does the wave deliver tangible feature or analytical capability? |
| **Regression Risk** | How likely is the wave to introduce regressions in the proven behavioral/operational surface? (5 = low risk) |
| **Scope Containment** | Can the wave be bounded to a disciplined charter without sprawl? |
| **Prerequisite Readiness** | Are the architectural and operational foundations sufficient to begin this wave now? |

---

## Candidate A: Composite Execution Observability

**Description:** Add metrics collection (Prometheus), distributed tracing (OTEL), graceful shutdown hardening, and operational dashboards for the execution pipeline.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Reuse | 4 | Instruments existing binaries and adapters; no new domain surfaces |
| Architectural Pressure | 3 | Forces metric/trace instrumentation patterns but doesn't validate domain composition |
| Domain Value | 2 | Operational maturity only; no new analytical or trading capability |
| Regression Risk | 5 | Additive instrumentation; does not modify behavioral or execution logic |
| Scope Containment | 4 | Bounded by adapter count and binary topology; clear exit criteria |
| Prerequisite Readiness | 5 | All binaries exist; structured logging already in place |
| **Total** | **23** | |

**Strengths:**
- Zero regression risk — purely additive
- Enables operational confidence for all future waves
- Closes CI enforcement gaps (OD-OH1, OD-OH2) as natural prerequisite
- Proves graceful shutdown and buffer drain semantics

**Weaknesses:**
- Fourth consecutive infrastructure wave — contradicts S263 pivot to domain value
- Does not validate codegen velocity or behavioral model under new families
- Low user-facing value; system capabilities unchanged after delivery

---

## Candidate B: Multi-Symbol Disciplined

**Description:** Prove the system operates correctly with multiple symbols and timeframes concurrently, including restart/recovery semantics per-symbol.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Reuse | 5 | Exercises all existing code paths; no new domain types |
| Architectural Pressure | 4 | Validates consumer isolation, KV partitioning, and concurrent pipeline behavior |
| Domain Value | 3 | Proves scale readiness but adds no new analytical capability |
| Regression Risk | 4 | Exercises existing paths at higher cardinality; moderate discovery risk |
| Scope Containment | 3 | Scope boundary between "2 symbols" and "N symbols" is fuzzy |
| Prerequisite Readiness | 4 | smoke-multi-symbol.sh exists; partial proof already delivered |
| **Total** | **23** | |

**Strengths:**
- Validates architecture at intended operational cardinality
- Exposes consumer isolation bugs before they become production incidents
- Partial proof already exists (smoke-multi-symbol.sh with 2x2 matrix)

**Weaknesses:**
- Diminishing returns — most isolation guarantees already proven by JetStream durable semantics
- Does not deliver new domain features
- Scope boundary ambiguous: how many symbols constitute "proven"?

---

## Candidate C: Signal Evolution / Dynamic Signals

**Description:** Deliver new signal families (MACD, VWAP, ATR) and corresponding decision families (Bollinger Squeeze) using codegen-first approach, with behavioral tests for each.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Reuse | 5 | Codegen-first leverages all infrastructure: codegen, behavioral model, NATS, ClickHouse, actors |
| Architectural Pressure | 5 | Validates codegen velocity, behavioral composition, and end-to-end pipeline per family |
| Domain Value | 5 | First real feature delivery; new analytical signals visible in ClickHouse and gateway |
| Regression Risk | 3 | New evaluators and samplers could interact with existing behavioral surface |
| Scope Containment | 4 | Bounded by family count; each family is a self-contained increment |
| Prerequisite Readiness | 3 | Requires CI enforcement closure first (OD-OH1/OH2); Bollinger already partially integrated |
| **Total** | **25** | |

**Strengths:**
- Highest domain value — delivers tangible new analytical capability
- Validates the entire infrastructure investment (codegen, behavioral, operational)
- Each family is independently charterable and testable
- Codegen-first ensures consistency with existing families
- Bollinger is already integrated (S262); MACD/VWAP/ATR are natural next families

**Weaknesses:**
- Requires CI enforcement as hard prerequisite (25 NATS KV tests auto-skip)
- New behavioral interactions could surface unexpected regressions
- No observability infrastructure to monitor new family behavior in compose

---

## Candidate D: Venue-Readiness Charter

**Description:** Design and implement real venue adapters (Binance testnet), order lifecycle management, fill reconciliation, and position tracking.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Reuse | 2 | Requires new adapter layer, order types, fill handling; limited reuse of paper path |
| Architectural Pressure | 5 | Forces real external dependency management, error handling, reconnection semantics |
| Domain Value | 4 | Moves toward real execution but only on testnet; no production money flow |
| Regression Risk | 2 | External dependencies introduce non-deterministic failure modes; high blast radius |
| Scope Containment | 2 | Venue integration scope is inherently expansive; fill reconciliation alone is a wave |
| Prerequisite Readiness | 1 | Paper execution barely proven; no observability; no graceful shutdown; CI gaps exist |
| **Total** | **16** | |

**Strengths:**
- Highest ambition — moves closest to production utility
- Forces real error handling and reconnection patterns

**Weaknesses:**
- Premature — paper execution proven only at S280 level; compose-level operational proof is days old
- External Binance dependency introduces non-deterministic test failures
- Fill reconciliation, partial fills, order amendments each deserve their own charter
- No observability to diagnose venue adapter failures
- Highest regression risk of all candidates

---

## Candidate E: Selective Codegen Expansion

**Description:** Expand codegen coverage to store consumers, starters, mappers, and migration generators for existing families.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Reuse | 5 | Extends existing codegen engine; all templates and patterns exist |
| Architectural Pressure | 2 | Internal tooling improvement; does not pressure domain or operational surfaces |
| Domain Value | 1 | Zero user-facing value; purely developer velocity improvement |
| Regression Risk | 4 | Codegen changes validated by golden snapshot equivalence tests |
| Scope Containment | 4 | Bounded by artifact type count; clear equivalence criteria |
| Prerequisite Readiness | 5 | Codegen engine exists; golden snapshots protect against drift |
| **Total** | **21** | |

**Strengths:**
- Low risk, high internal velocity payoff
- Golden snapshot tests prevent drift
- S263 identified these artifacts as natural side-effects

**Weaknesses:**
- S263 explicitly said codegen expansion should be side-effect, not primary wave objective
- Zero domain value delivery
- Fifth consecutive infrastructure wave would be unjustifiable

---

## Candidate F: CI Enforcement + Observability Foundation (Hybrid)

**Description:** Close CI enforcement gaps (OD-OH1, OD-OH2) and establish minimal metrics/tracing foundation before opening a feature wave.

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Reuse | 4 | Instruments existing binaries; NATS/ClickHouse CI services are standard |
| Architectural Pressure | 3 | CI enforcement is governance, not architecture; metrics add observability patterns |
| Domain Value | 1 | Zero feature delivery; purely operational |
| Regression Risk | 5 | Additive; CI enforcement catches existing regressions better |
| Scope Containment | 5 | Extremely bounded: add 2 CI services + basic metrics package |
| Prerequisite Readiness | 5 | All infrastructure exists; needs configuration only |
| **Total** | **23** | |

**Strengths:**
- Closes the most critical medium-severity debts (OD-OH1, OD-OH2)
- Enables all future waves to benefit from regression detection
- Very small scope (2-3 stages maximum)

**Weaknesses:**
- Too small to be a "wave" — more of a prerequisite tranche
- Does not deliver domain value
- If elevated to full wave, risks scope creep into full observability platform

---

## Comparative Matrix

| Candidate | Reuse | Arch. Pressure | Domain Value | Regression Risk | Scope Contain. | Prereq. Ready | **Total** |
|-----------|-------|----------------|--------------|-----------------|----------------|---------------|-----------|
| A: Observability | 4 | 3 | 2 | 5 | 4 | 5 | **23** |
| B: Multi-Symbol | 5 | 4 | 3 | 4 | 3 | 4 | **23** |
| **C: Signal Evo** | **5** | **5** | **5** | **3** | **4** | **3** | **25** |
| D: Venue Ready | 2 | 5 | 4 | 2 | 2 | 1 | **16** |
| E: Codegen Exp | 5 | 2 | 1 | 4 | 4 | 5 | **21** |
| F: CI + Obs Fdn | 4 | 3 | 1 | 5 | 5 | 5 | **23** |

---

## Key Observations

1. **Signal Evolution (C)** scores highest overall and is the only candidate with maximum domain value.
2. **Venue Readiness (D)** scores lowest — the system is not architecturally or operationally ready.
3. **CI + Observability Foundation (F)** is not a wave but a prerequisite tranche; it should precede any feature wave.
4. **Codegen Expansion (E)** and **Observability (A)** would be a fifth consecutive infrastructure wave — unjustifiable per S263 guidance.
5. **Multi-Symbol (B)** has merit but delivers no new capability; it validates existing capability at higher cardinality.
6. **Signal Evolution (C)** has a hard prerequisite: CI enforcement (OD-OH1/OH2) must close first so new families don't mask regressions.
