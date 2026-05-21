# Stage S494 — Canonical Cross-Session Continuity Model Report

**Stage**: S494
**Type**: Canonical Model Definition
**Status**: COMPLETE
**Date**: 2026-03-26
**Wave**: Cross-Session Position Continuity (S493–S497)
**Predecessor**: S493 (Charter and Scope Freeze — COMPLETE)

---

## 1. Executive Summary

S494 delivers the canonical model for cross-session continuity. It defines
domain types, carry-forward rules, continuity classification, and session
boundary semantics that reduce the ambiguity identified in G-RT4 (round-trip
pairing evidence gate, S483).

The model is read-side only. It does not modify the write path, carry state
at runtime, or introduce position tracking. All types are additive —
existing pairing, effectiveness, and session types are unchanged.

**Key contribution**: The four-state continuity model (`resolved`, `open`,
`genuine_unresolved`, `artificial_unresolved`) formalizes the distinction
between legs that are unresolved due to session boundaries (fixable) and legs
that are structurally unresolvable (permanent). This distinction is critical
for accurate effectiveness measurement across sessions.

---

## 2. Problem Addressed

### 2.1 Before S494

- Unmatched legs at session close were all classified as `unresolved`.
- No formal distinction between "unresolved because session ended" and
  "unresolved because order was rejected".
- No domain types for cross-session leg discovery or multi-session windows.
- No canonical rules for what constitutes valid carry-forward.
- Ambiguity about what can or should cross a session boundary.

### 2.2 After S494

- Four-state continuity model distinguishes actionable from non-actionable
  unresolved outcomes.
- `SessionLeg` wraps `Leg` with session provenance without modifying existing types.
- `CrossSessionWindow` and `CrossSessionLegSet` define the discovery scope.
- Five carry-forward rules (R-CF1 through R-CF5) are explicit and testable.
- Six continuity classification rules (C-1 through C-6) are deterministic.
- `AnnotateRoundTrips` bridges `MatchFIFO` output to cross-session provenance.

---

## 3. Capability Delivery

| Capability | Charter Target | Status | Evidence |
|-----------|---------------|--------|----------|
| C-CS1 (cross-session leg discovery) | S494 | PARTIAL — model and types delivered; query implementation deferred to S495 | Domain types define the contract. `CrossSessionWindow`, `CrossSessionLegSet`, and `ClassifyCarryForward` provide the foundation. S495 implements the actual ClickHouse query. |

**Note**: C-CS1 is split across S494 (model) and S495 (implementation). S494
delivers the type contract and classification logic. S495 wires the query.

---

## 4. Governing Questions

| ID | Question | Answer | Evidence |
|----|----------|--------|----------|
| Q-CS1 (partial) | Can the system discover unmatched entry legs from prior sessions? | YES (model) — types and classification are defined; query deferred to S495 | `ClassifyCarryForward` determines eligibility; `CrossSessionWindow` defines scope; `SessionLeg` carries provenance |

---

## 5. Domain Types Introduced

| Type | Location | Purpose |
|------|----------|---------|
| `ContinuityState` | `continuity.go` | Four-state lifecycle resolution enum |
| `SessionLeg` | `continuity.go` | `Leg` + session provenance (ID, start, close) |
| `CrossSessionWindow` | `continuity.go` | Temporal and filtering scope for discovery |
| `CrossSessionLegSet` | `continuity.go` | Ordered collection of session-annotated legs |
| `CarryForwardEligibility` | `continuity.go` | Six-value enum for intent eligibility |
| `CrossSessionRoundTrip` | `continuity.go` | `RoundTrip` + session provenance + continuity |

### 5.1 Pure functions

| Function | Input | Output |
|----------|-------|--------|
| `ClassifyCarryForward(intent)` | `ExecutionIntent` | `CarryForwardEligibility` |
| `ClassifyContinuity(rt)` | `RoundTrip` | `ContinuityState` |
| `IsCrossSession(a, b)` | Two `SessionLeg` | `bool` |
| `AnnotateRoundTrips(rts, idx)` | `[]RoundTrip`, `map[string]SessionLeg` | `[]CrossSessionRoundTrip` |

---

## 6. Tests

All tests in `internal/domain/pairing/s494_continuity_test.go`:

| Test | Rule | What it validates |
|------|------|-------------------|
| `TestValidContinuityState` | — | Enum validation |
| `TestValidCarryForwardEligibility` | — | Enum validation |
| `TestClassifyCarryForward_RejectedIsIneligible` | R-CF1 | Rejected → ineligible |
| `TestClassifyCarryForward_NonTerminalIsIneligible` | R-CF3 | Submitted/sent/accepted → ineligible |
| `TestClassifyCarryForward_CancelledNoFillsIsIneligible` | R-CF2 | Cancelled no fills → ineligible |
| `TestClassifyCarryForward_FilledNoFillsIsIneligible` | R-CF4 | Terminal no fills → ineligible |
| `TestClassifyCarryForward_FilledWithFillsIsEligible` | R-CF5 | Filled + fills → eligible |
| `TestClassifyCarryForward_PartiallyFilledIsIneligible` | R-CF3 | Partially filled (non-terminal) → ineligible |
| `TestClassifyContinuity_PairedIsResolved` | C-1 | Paired → resolved |
| `TestClassifyContinuity_SessionBoundaryIsArtificialUnresolved` | C-2 | Session boundary → artificial |
| `TestClassifyContinuity_RejectedLegIsGenuineUnresolved` | C-3 | Rejected → genuine |
| `TestClassifyContinuity_CancelledLegIsGenuineUnresolved` | C-3 | Cancelled → genuine |
| `TestClassifyContinuity_NoExitFoundIsOpen` | C-4 | No exit → open |
| `TestClassifyContinuity_QuantityResidueIsOpen` | C-6 | Partial remainder → open |
| `TestClassifyContinuity_OrphanExitIsGenuineUnresolved` | C-5 | Orphan exit → genuine |
| `TestIsCrossSession_*` | INV-2 | Session provenance detection |
| `TestCrossSessionWindow_*` | — | Window validation |
| `TestCrossSessionLegSet_*` | — | Leg set operations |
| `TestAnnotateRoundTrips_CrossSessionPair` | INV-2 | Cross-session annotation |
| `TestAnnotateRoundTrips_IntraSessionPair` | INV-3 | Intra-session not flagged |
| `TestAnnotateRoundTrips_UnmatchedEntrySessionBoundary` | C-2 | Boundary leg continuity |
| `TestMatchFIFO_CrossSessionLegsProducePairedRoundTrip` | INV-1 | MatchFIFO works on cross-session legs |
| `TestMatchFIFO_CrossSessionPreservesTemporalOrdering` | M4, M5 | FIFO ordering across sessions |

**Result**: 63 tests pass (26 existing + 37 new). Zero regressions.

---

## 7. Guard Rail Compliance

| Guard Rail | Status | Evidence |
|-----------|--------|----------|
| GR-1: No write-path changes | COMPLIANT | All types are read-side. No changes to execution, session, or event models. |
| GR-2: No position engine | COMPLIANT | No position accumulator, no inventory, no portfolio model. |
| GR-3: No OMS expansion | COMPLIANT | No new order types, no cancel/modify. |
| GR-4: No multi-exchange scope | COMPLIANT | Binance-only, existing segments. |
| GR-5: No new infrastructure | COMPLIANT | No new databases, streams, or consumers. |
| GR-6: No dashboards | COMPLIANT | No UI, no metrics, no Grafana. |
| GR-7: No runtime carry-forward | COMPLIANT | Sessions remain isolated. Continuity is retrospective. |
| GR-8: Stage closes independently | COMPLIANT | S494 delivers complete model; S495 can consume without further modeling. |

---

## 8. Limitations and Residual Ambiguity

| ID | Limitation | Impact | Mitigation |
|----|-----------|--------|-----------|
| L-1 | No runtime carry-forward | System does not "know" about open positions at session start | Accepted; retrospective model by design |
| L-2 | Strategy direction must be consistent across sessions | Mixed long/short for same symbol won't cross-pair | Document for operators; rare in practice |
| L-3 | Non-terminal orders at session close are ineligible | Post-session fills for prior orders not captured | Market orders reach terminal fast; edge case |
| L-4 | Lookback window is finite (30 days default) | Very long holds may not resolve | Operator can extend window |
| L-5 | No cross-symbol or cross-segment pairing | M2 invariant enforced | By design |
| L-6 | Fee schedule changes between sessions | P&L uses recorded fees, not current | Correct behavior; documented |

**No CRITICAL or HIGH gaps introduced by this stage.**

---

## 9. Artifacts Produced

| Artifact | Type | Location |
|----------|------|----------|
| Canonical continuity model | Architecture | `docs/architecture/canonical-cross-session-continuity-model.md` |
| Open fragments and carry-forward rules | Architecture | `docs/architecture/open-fragments-session-boundaries-carry-forward-rules-and-limitations.md` |
| Domain types | Code | `internal/domain/pairing/continuity.go` |
| Tests | Code | `internal/domain/pairing/s494_continuity_test.go` |
| Stage report | Stage | This document |

---

## 10. Preparation for S495

S495 (Cross-Session Read Model and Continuity Attribution) should:

1. Implement the cross-session discovery query:
   - Accept `CrossSessionWindow` as input.
   - Query session metadata from KV to enumerate sessions in the window.
   - Query ClickHouse for execution intents per session.
   - Apply `ClassifyCarryForward` to filter eligible intents.
   - Convert to `SessionLeg` via `IntentToLeg` + session metadata.
   - Assemble `CrossSessionLegSet` (sorted by timestamp).

2. Execute cross-session matching:
   - `MatchFIFO(legSet.ExtractLegs(), DefaultMatchingConfig())`
   - `AnnotateRoundTrips(roundTrips, legSet.SessionLegIndex())`

3. Compute P&L attribution:
   - `ClassifyPair(entryIntent, exitIntent)` for cross-session pairs.

4. Expose HTTP surface:
   - `GET /analytical/pairing/cross-session` (or similar).
   - Response includes `CrossSessionRoundTrip[]` with provenance and P&L.

**Key files to extend**:
- `internal/application/analyticalclient/` — new use case
- `internal/adapters/clickhouse/composite_reader.go` — multi-session query
- `cmd/gateway/analytical_reader.go` — HTTP handler

---

## 11. Conclusion

S494 delivers a minimal, auditable, and well-tested canonical model for
cross-session continuity. The four-state continuity classification, five
carry-forward rules, and session-annotated types reduce the ambiguity that
inflated `unresolved` counts in effectiveness measurement. All invariants
are tested. All guard rails are respected. The model is ready for S495 to
implement the read path and attribution.

Next stage: **S495 — Cross-Session Read Model and Continuity Attribution**.
