# Operational Hardening Wave — Charter and Scope Freeze

**Stage**: S498
**Date**: 2026-03-28
**Status**: OPEN — Scope frozen
**Predecessor wave**: Cross-Session Position Continuity (S493–S497, FULL PASS)
**Wave span**: S498–S502

---

## 1. Strategic Context

The Foundry has closed three consecutive analytical-layer waves since S452a, culminating in the S497 FULL PASS for cross-session position continuity. The analytical and operational read layer is substantially complete: historical execution queries, round-trip pairing (intra- and cross-session), effectiveness measurement with cohort grouping, reconciliation flags for data quality, continuity attribution, and operational automation are all delivered and proven.

The next bottleneck is not analytical — it is operational/runtime. Documented gaps across multiple waves (S412, S428, S482, S488, S496) point to three recurring themes:

1. **Fee persistence and reconciliation gaps** — Futures commission unavailable from venue API (RG-22), cross-session fee gaps flagged but not reconciled, pre-S428 historical data with overloaded fee semantics (NB-8).
2. **Lifecycle close edge cases** — session close relies on supervisor stop without explicit close logic, duplicate legs from improper session closure (L-S496-3), no time-based drift detection (L-S412-1/L-S412-2).
3. **Writer and runtime stability under sustained operation** — endurance proven for Spot synthetic cycles but not real wall-clock time (L-S412-1), no Futures segment endurance (L-S412-6), batch flush lag documented but not bounded by SLO.

This wave hardens these three operational surfaces, then closes with an evidence gate.

---

## 2. Wave Objective

Harden the operational behavior of the execution pipeline so that:

- fee data is as accurate as the venue allows, and gaps are reconcilable;
- session lifecycle transitions are robust against edge cases at close;
- the writer pipeline is proven stable under sustained real-segment operation;
- an evidence gate validates all three dimensions before any further expansion.

---

## 3. Wave Structure

| Stage | Title | Scope |
|-------|-------|-------|
| **S498** | Operational Hardening Wave Charter and Scope Freeze | This document — opens the wave, freezes scope |
| **S499** | Fee Persistence and Reconciliation Hardening | Close Futures fee gap to best achievable accuracy; historical fee field normalization; fee reconciliation rule hardening |
| **S500** | Lifecycle Close Edge Cases Hardening | Explicit session close logic; duplicate leg prevention; session boundary timestamp guards |
| **S501** | Sustained Runtime and Writer Stability Proof | Futures endurance coverage; wall-clock-aware stability assertions; batch flush SLO definition |
| **S502** | Operational Hardening Evidence Gate | Formal gate evaluation, residual gap registry, wave closure |

---

## 4. Capabilities

### 4.1 Capability Register

| ID | Capability | Priority | Stage |
|----|-----------|----------|-------|
| C-OH1 | Futures fee retrieval or bounded estimation | MUST | S499 |
| C-OH2 | Historical fee field normalization (pre-S428 data handling) | SHOULD | S499 |
| C-OH3 | Fee reconciliation rule tightening across segments | MUST | S499 |
| C-OH4 | Explicit session close with terminal state enforcement | MUST | S500 |
| C-OH5 | Duplicate leg prevention at session boundaries | MUST | S500 |
| C-OH6 | Session boundary timestamp guard (±buffer enforcement) | SHOULD | S500 |
| C-OH7 | Futures segment endurance coverage | MUST | S501 |
| C-OH8 | Wall-clock-aware stability assertions | SHOULD | S501 |
| C-OH9 | Batch flush SLO definition and enforcement | SHOULD | S501 |

### 4.2 Priority Summary

- **MUST**: 5 capabilities (C-OH1, C-OH3, C-OH4, C-OH5, C-OH7)
- **SHOULD**: 4 capabilities (C-OH2, C-OH6, C-OH8, C-OH9)
- **MAY**: none — scope is intentionally narrow

---

## 5. Governing Questions

| Q-ID | Question | Required Answer for PASS |
|------|----------|--------------------------|
| Q-OH1 | Is Futures fee data as accurate as the Binance API structurally allows? | YES |
| Q-OH2 | Can an operator distinguish reliable from unreliable fee data across all segments and historical periods? | YES |
| Q-OH3 | Does session close produce a deterministic terminal state with no orphan or duplicate legs? | YES |
| Q-OH4 | Is the writer pipeline proven stable across both Spot and Futures segments under sustained load? | YES |
| Q-OH5 | Are batch flush lag bounds defined and enforced? | YES |

All five questions must answer YES for a FULL PASS.

---

## 6. Non-Goals (Frozen)

| NG-ID | What is explicitly out of scope | Rationale |
|-------|--------------------------------|-----------|
| NG-1 | OMS expansion (new order types, position engine, portfolio management) | Separate macro-direction; requires its own charter |
| NG-2 | Multi-exchange support (venues beyond Binance) | Requires adapter-layer expansion; not an operational hardening concern |
| NG-3 | Broad observability platform (distributed tracing, log aggregation, dashboard suite) | This wave hardens specific operational edges, not observability breadth |
| NG-4 | New strategy families or signal types | Analytics and strategy are stable; this wave is runtime-only |
| NG-5 | Structural redesign (binary topology, NATS subject model, ClickHouse schema redesign) | Architecture is stable; changes are targeted hardening within existing structure |
| NG-6 | Portfolio/risk engine or exposure management | Separate domain; requires its own evidence chain |
| NG-7 | Dashboard or UI surfaces | No operator-facing UI work in this wave |
| NG-8 | Real-time streaming analytics or alerting platform | Operator surfaces are retrospective/query-based; real-time alerting is a separate concern |
| NG-9 | Cross-binary health aggregation (ingest/store health signals) | Constrained to execute/writer pipeline; broader health is a future wave |
| NG-10 | Write-path schema changes beyond fee field normalization | ClickHouse schema is stable; only fee-related fields may be touched |

---

## 7. Guard Rails

| GR-ID | Rule |
|-------|------|
| GR-1 | No new infrastructure dependencies (no new databases, no new message brokers) |
| GR-2 | No write-path schema changes except fee field normalization in S499 |
| GR-3 | No new HTTP endpoints — existing surfaces may be enriched but no new routes |
| GR-4 | No changes to the NATS subject model |
| GR-5 | No changes to the domain event envelope contract |
| GR-6 | Each stage must close independently with its own evidence |
| GR-7 | No scope addition after S498 freeze without a new charter ceremony |
| GR-8 | Total wave span ≤ 5 stages (S498–S502) |

---

## 8. Risk Register

| R-ID | Risk | Likelihood | Impact | Mitigation |
|------|------|-----------|--------|------------|
| R-1 | Futures fee retrieval requires a separate API call that may add latency or rate-limit pressure | MEDIUM | MEDIUM | Investigate post-fill async enrichment; fall back to bounded estimation if call is impractical |
| R-2 | Historical fee normalization may require ClickHouse migration | LOW | MEDIUM | GR-2 limits schema changes; normalization may be query-time only |
| R-3 | Session close hardening may uncover undocumented edge cases | LOW | LOW | Test coverage for all terminal transitions already at 200 cycles; hardening is targeted |
| R-4 | Futures endurance test may reveal segment-specific instability | LOW | MEDIUM | Architecture is segment-agnostic by design (S398–S403); risk is bounded |

---

## 9. Success Criteria

The wave passes when:

1. All MUST capabilities are at FULL classification.
2. All SHOULD capabilities are at SUBSTANTIAL or higher.
3. All governing questions answer YES.
4. All guard rails are compliant.
5. Zero regressions across all existing test suites.
6. No critical or high residual gaps.

---

## 10. Dependencies and Preconditions

### Met (from prior waves)

- Fee normalization canonical model (S428) — field semantics defined
- Writer stability baseline (S412) — 200 cycles, zero failures
- Session lifecycle model (execute_supervisor) — open/close/error states
- Reconciliation framework (S482, S496) — `fee_gap`, `fee_asset_mismatch`, `cross_session_fee_gap` flags
- Cross-session pairing (S493–S497) — boundary handling proven

### Not required

- No new API keys
- No new live trading session
- No new venue connectivity
- No new infrastructure provisioning

---

## 11. Alignment with Consolidated Capabilities

This wave builds directly on:

| Capability | Wave | How this wave uses it |
|-----------|------|----------------------|
| Fee normalization | S428 | Extends fee accuracy for Futures; tightens reconciliation rules |
| Reconciliation flags | S482, S496 | Hardens flag triggers; adds historical awareness |
| Writer endurance | S412 | Extends to Futures segment; adds wall-clock dimension |
| Session lifecycle | S444–S448 | Hardens close logic; adds duplicate prevention |
| Operational automation | S484–S492 | Operational verification already automated; this wave hardens what is verified |

---

## References

- [Capabilities, Questions, and Non-Goals](operational-hardening-capabilities-questions-and-non-goals.md)
- [S497 Evidence Gate (predecessor)](../stages/stage-s497-cross-session-continuity-evidence-gate-report.md)
- [Fee Semantics](fees-commission-assets-cross-segment-semantics-and-limitations.md)
- [Writer Stability](sustained-execution-state-consistency-writer-stability-and-limitations.md)
- [Fills and Fee Reconciliation](fills-fees-pairing-result-reconciliation-semantics-and-limitations.md)
- [Cross-Session Continuity Evidence Matrix](cross-session-continuity-evidence-matrix-residual-gaps-and-next-ceremony.md)
