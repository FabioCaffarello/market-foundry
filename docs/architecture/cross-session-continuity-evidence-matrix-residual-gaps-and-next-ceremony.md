# Cross-Session Position Continuity — Evidence Matrix, Residual Gaps, and Next Ceremony

**Stage**: S497
**Date**: 2026-03-27
**Wave**: Cross-Session Position Continuity (S493–S497)

---

## 1. Evidence Matrix

### 1.1 Capability × Stage Matrix

| Capability | S493 (Charter) | S494 (Model) | S495 (Read Model) | S496 (Reconciliation) | Classification |
|-----------|----------------|---------------|--------------------|-----------------------|----------------|
| **C-CS1**: Leg Discovery | Scope defined | `ClassifyCarryForward` (5 rules), `CarryForwardEligibility` (6 states), `CrossSessionWindow`, `CrossSessionLegSet` — 23 tests | Discovery flow via KV + ClickHouse, carry-eligible filtering — 14 tests | — | **FULL** |
| **C-CS2**: Multi-Session FIFO | Scope defined | `AnnotateRoundTrips`, `IsCrossSession`, `ClassifyContinuity` (6 rules), `CrossSessionRoundTrip` — 23 tests | `MatchFIFO` on multi-session legs, end-to-end 3-session scenario — 14 tests | — | **FULL** |
| **C-CS3**: P&L Attribution | Scope defined | `CrossSessionRoundTrip` carries session provenance | `ClassifyPair` attribution on cross-session pairs, `ContinuitySummary` — 14 tests | `ContinuityEffectivenessSummary` splits cross/intra P&L — 13 tests | **FULL** |
| **C-CS4**: HTTP Surface | Scope defined | — | `GET /analytical/composite/pairing/cross-session` endpoint, query contracts — 14 tests | `GET /analytical/composite/pairing/continuity-review` endpoint — 7 app tests | **FULL** |
| **C-CS5**: Reconciliation Flags | Scope defined | — | — | 3 flags (`cross_session`, `boundary_carryover`, `cross_session_fee_gap`), `carryover_reliable` — 8 domain tests + 6 app tests | **FULL** |
| **C-CS6**: Audit Bundle | Scope defined | — | — | Review response is self-contained audit bundle; no session-level integration | **SUBSTANTIAL** |

### 1.2 Governing Question × Evidence Matrix

| Q-ID | Question | Evidence Artifacts | Verdict |
|------|----------|--------------------|---------|
| Q-CS1 | Discover unmatched entries from prior sessions? | `ClassifyCarryForward()`, `CrossSessionLegSet`, KV session query, ClickHouse chain query, `s494_continuity_test.go` (23 tests) | **FULL** |
| Q-CS2 | Pair entries/exits across boundaries using FIFO? | `MatchFIFO` on `ExtractLegs()`, `AnnotateRoundTrips()`, `ClassifyContinuity()`, end-to-end 2-session and 3-session tests | **FULL** |
| Q-CS3 | Accurate P&L with full lineage? | `ClassifyPair` on `CrossSessionRoundTrip`, `ContinuityEffectivenessSummary`, `s496_continuity_review_test.go` (7 tests) | **FULL** |
| Q-CS4 | Query via HTTP? | Two endpoints registered, contracts validated, handler integration, `s496_continuity_review_test.go` (7 tests) | **FULL** |
| Q-CS5 | Distinguishable in reconciliation? | 3 flags, `ReconcileCrossSessionRoundTrip()`, `ContinuityReconciliationSummary`, `s496_continuity_reconciliation_test.go` (8 tests) | **FULL** |

### 1.3 Test Artifact Summary

| Test File | Count | Scope | Result |
|-----------|-------|-------|--------|
| `internal/domain/pairing/s494_continuity_test.go` | 23 | Continuity model, carry-forward rules, classification, leg sets, annotation, end-to-end FIFO | **ALL PASS** |
| `internal/domain/pairing/s495_continuity_summary_test.go` | 14 | Continuity summary, leg set operations, provenance, end-to-end 3-session | **ALL PASS** |
| `internal/domain/pairing/s496_continuity_reconciliation_test.go` | 8 | Reconciliation flags, fee gap, carryover reliability, summary, dedup | **ALL PASS** |
| `internal/application/analyticalclient/s496_continuity_review_test.go` | 7 | Use case unavailable, validation, no sessions, intra-session, cross-session, flagged filter | **ALL PASS** |
| **Total new tests** | **52** | | **ALL PASS** |
| **Total pairing domain tests** | **87** | Including pre-existing | **ALL PASS** |
| **Total analyticalclient tests** | **196** (subtests) | Including pre-existing | **ALL PASS** |

### 1.4 Architecture Documents Produced

| Document | Stage | Scope |
|----------|-------|-------|
| `cross-session-position-continuity-wave-charter-and-scope-freeze.md` | S493 | Wave charter, 6 capabilities, 5 questions, 8 guard rails, risk register |
| `cross-session-continuity-capabilities-questions-and-non-goals.md` | S493 | Capability details, non-goals, success metrics |
| `canonical-cross-session-continuity-model.md` | S494 | Domain types, carry-forward rules, continuity classification, invariants |
| `open-fragments-session-boundaries-carry-forward-rules-and-limitations.md` | S494 | Session boundary semantics, fragment taxonomy, limitations L-1 through L-7 |
| `cross-session-read-model-and-continuity-attribution.md` | S495 | Read model architecture, 7-step flow, HTTP contract, invariants |
| `carryover-read-surfaces-resolved-vs-unresolved-attribution-and-limitations.md` | S495 | Continuity state taxonomy, resolution metrics, limitations L-S495-1 through L-S495-7 |
| `continuity-review-and-cross-session-reconciliation.md` | S496 | Review surface contract, reconciliation flags, carryover reliability, limitations L-S496-1 through L-S496-5 |
| `carryover-boundary-fees-pairing-result-reconciliation-semantics-and-limitations.md` | S496 | Carryover fragment taxonomy, fee reconciliation, effectiveness reconciliation, limitations L1–L6 |
| `cross-session-continuity-evidence-gate.md` | S497 | Formal gate evaluation, verdict |
| `cross-session-continuity-evidence-matrix-residual-gaps-and-next-ceremony.md` | S497 | This document |

---

## 2. Residual Gaps

### 2.1 Acknowledged Limitations (Non-Blocking)

| ID | Description | Severity | Status | Disposition |
|----|-------------|----------|--------|-------------|
| L-1 | No runtime carry-forward — sessions do not share state at runtime | LOW | **BY DESIGN** | Guard rail GR-7; retrospective model is the charter's explicit scope |
| L-2 | Strategy direction consistency required — if session N runs long and session N+1 runs short for same symbol, legs will not cross-pair | LOW | **DOCUMENTED** | Rare in practice; pairing is directional (buy entry needs sell exit) |
| L-3 | Non-terminal orders at session close are ineligible for carry-forward (R-CF3) | LOW | **DOCUMENTED** | Market orders reach terminal fast; edge case |
| L-4 | Lookback window finite (default 30 days/30 sessions) | LOW | **DOCUMENTED** | Operator can extend; no hard maximum enforced |
| L-5 | No cross-symbol or cross-segment pairing | LOW | **BY DESIGN** | Matches existing M2 invariant; out of scope |
| L-6 | Fee schedule changes between sessions reflected in recorded fees | LOW | **CORRECT BEHAVIOR** | P&L uses recorded fees, not inferred rates |
| L-S495-2 | Session time overlap risk — fill timestamps near session close boundary | LOW | **MITIGATED** | ±5 minute buffer (S485) |
| L-S495-3 | Deduplication across overlapping windows — same correlation_id in multiple windows | VERY LOW | **ACCEPTED** | Sessions non-overlapping by design |
| L-S495-4 | Attribution requires both chains persisted | LOW | **GRACEFUL DEGRADATION** | Pairing valid; attribution nil if chain missing |
| L-S495-5 | No intra-session pre-filtering — already-paired legs re-processed | VERY LOW | **ACCEPTED** | No correctness impact; minor cost |
| L-S495-6 | Fee data missing at session boundaries | LOW | **MITIGATED** | `cross_session_fee_gap` flag + `carryover_reliable` assessment |
| L-S495-7 | No real-time updates — point-in-time query | LOW | **BY DESIGN** | Retrospective model |
| L-S496-1 | No session-level audit bundle integration | LOW | **DEFERRED** | MAY capability; continuity-review is self-contained |
| L-S496-2 | Fill timestamp precision (DateTime64 vs time.Time) | VERY LOW | **MITIGATED** | ±5 minute buffer |
| L-S496-3 | Duplicate legs from improper session closure | LOW | **OPERATIONAL** | Requires operational discipline |
| L-S496-4 | Fee normalization across segments | LOW | **MITIGATED** | `fee_asset_mismatch` flag detects this |
| L-S496-5 | No aggregated position tracking | LOW | **BY DESIGN** | Scope is round-trip review, not position management |

### 2.2 Risk Registry Audit

| Risk (from Charter) | Materialized? | Outcome |
|---------------------|--------------|---------|
| R-1: MatchFIFO may not work across sessions | **NO** | Algorithm is session-agnostic; works without modification |
| R-2: ClickHouse query cost for multi-session lookback | **NO** | Bounded by `max_sessions` (default 30); acceptable performance |
| R-3: Carry-forward rules too restrictive (false negatives) | **NO** | R-CF5 captures all meaningful cases; rejected/cancelled correctly excluded |
| R-4: Continuity classification ambiguity | **NO** | Four-state model is deterministic; C-1 through C-6 rules are mutually exclusive |

### 2.3 Critical / High Gaps

**None.** All gaps are LOW or VERY LOW severity. No gap blocks wave closure.

---

## 3. Regression Summary

### 3.1 Full Test Suite Execution

| Package Group | Packages | Result |
|--------------|----------|--------|
| `internal/domain/...` | 14 packages | **ALL PASS** |
| `internal/application/...` | 20 packages | **ALL PASS** |
| `internal/actors/...` | 5 packages | **ALL PASS** |

### 3.2 Build Verification

| Binary | Result |
|--------|--------|
| `cmd/gateway` | **BUILD OK** |
| `cmd/execute` | **BUILD OK** |
| `cmd/writer` | **BUILD OK** |

### Regression Verdict: **ZERO REGRESSIONS**

---

## 4. Wave Verdict

| Dimension | Result |
|-----------|--------|
| MUST capabilities at FULL | **3/3** |
| SHOULD capabilities at FULL | **2/2** |
| MAY capabilities at SUBSTANTIAL+ | **1/1** |
| Governing questions YES | **5/5** |
| Guard rails compliant | **8/8** |
| Regressions | **ZERO** |
| Critical/High gaps | **NONE** |
| Non-goal compliance | **ALL RESPECTED** |
| Risk registry items materialized | **NONE** |

**VERDICT: FULL PASS — Wave closed.**

---

## 5. Next Ceremony Recommendation

### 5.1 Strategic Context

With S497, the Foundry has closed three consecutive analytical-layer waves:

1. **Operational History and Explainability** (S452a–S456a) — historical execution read model, list queries, session explainability
2. **Session Intelligence and Operational Automation** (S459–S492) — session metadata, PO automation, audit bundles, verification, decision quality, strategy effectiveness measurement, operational automation
3. **Cross-Session Position Continuity** (S493–S497) — cross-session pairing, continuity attribution, reconciliation

The analytical and operational read layer is now substantially complete. The system can:
- Query historical execution lifecycle with full lineage
- Pair entries and exits within and across sessions using FIFO
- Attribute P&L with effectiveness classification
- Review reconciliation flags for data quality
- Measure strategy effectiveness with cohort grouping
- Discover and resolve artificial unresolved outcomes at session boundaries
- Split cross-session vs intra-session performance

### 5.2 Recommended Next Direction

**Strategy Effectiveness Measurement Completion** — The S474 wave (Strategy Effectiveness) opened with S474–S476 delivering the charter and measurement read surfaces. This wave remains open and is the natural next focus. Completing it would close the effectiveness measurement loop with batch evaluation inputs, aggregation surfaces, and operator-facing comparison queries.

Alternatively, if the effectiveness wave is considered sufficiently addressed by the existing surfaces, the next macro-direction should focus on **operational hardening** — specifically, areas where the runtime pipeline has known limitations documented across multiple waves (e.g., session lifecycle edge cases at close, fee persistence gaps, writer stability under sustained load).

### 5.3 Preconditions Met

- All analytical read surfaces operational (pairing, effectiveness, reconciliation, continuity)
- Cross-session boundary problem solved for retrospective analysis
- Reconciliation framework extended with cross-session flags
- No open blockers from this wave

### 5.4 What This Recommendation Does NOT Do

- Does not open the next wave — that requires a separate charter ceremony
- Does not commit to a timeline or scope
- Does not prioritize between effectiveness completion and operational hardening — that is an operator decision based on current priorities
- Does not authorize any implementation work
