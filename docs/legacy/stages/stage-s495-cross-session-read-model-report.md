# Stage S495 — Cross-Session Read Model and Continuity Attribution Report

**Stage**: S495
**Type**: Read Model Implementation
**Status**: COMPLETE
**Date**: 2026-03-26
**Wave**: Cross-Session Position Continuity (S493–S497)
**Predecessor**: S494 (Canonical Cross-Session Continuity Model — COMPLETE)

---

## 1. Executive Summary

S495 delivers the cross-session pairing read model — the application-layer orchestration that makes S494's continuity model queryable via HTTP. It composes data from two sources (NATS KV sessions + ClickHouse execution chains) into a unified cross-session pairing response with continuity classification, session provenance, and effectiveness attribution.

**Key contributions**:
- `GetCrossSessionPairingUseCase` orchestrates multi-session leg discovery, FIFO matching, and continuity annotation
- `ContinuitySummary` provides aggregate metrics for resolution rates and cross-session matching effectiveness
- HTTP endpoint `GET /analytical/composite/pairing/cross-session` exposes the complete read surface
- Integration with `ClassifyPair` provides full P&L attribution for cross-session round-trips
- 17 new tests (80 total in pairing domain), zero regressions

---

## 2. Problem Addressed

### 2.1 Before S495

- S494 defined domain types and classification rules but had no query implementation
- Cross-session continuity was not accessible through any HTTP surface
- Operators had no way to discover which legs resolved across sessions
- Pairing and effectiveness surfaces were intra-session only
- No aggregate metrics for cross-session resolution rates

### 2.2 After S495

- Cross-session pairing is queryable via a single HTTP endpoint
- Continuity classification (resolved, open, genuine unresolved, artificial unresolved) is surfaced per round-trip
- Session provenance is included for audit traceability
- Effectiveness attribution is computed for all paired round-trips (intra and cross-session)
- `ContinuitySummary` provides resolution rate and cross-session resolution rate metrics
- Filters by continuity state and cross-session flag enable targeted queries

---

## 3. Capability Delivery

| Capability | Charter Target | Status | Evidence |
|-----------|---------------|--------|----------|
| C-CS1 (cross-session leg discovery) | S494 model + S495 query | **COMPLETE** | `GetCrossSessionPairingUseCase` fetches sessions from KV, chains from ClickHouse, applies `ClassifyCarryForward`, builds `CrossSessionLegSet` |
| C-CS2 (cross-session matching) | S495 | **COMPLETE** | `MatchFIFO` applied to multi-session leg set; `AnnotateRoundTrips` adds provenance |
| C-CS3 (continuity read surface) | S495 | **COMPLETE** | `GET /analytical/composite/pairing/cross-session` with filters |
| C-CS4 (effectiveness integration) | S495 | **COMPLETE** | `ClassifyPair` attribution for all paired round-trips |
| C-CS5 (continuity summary) | S495 | **COMPLETE** | `ContinuitySummary` with resolution rates |

---

## 4. Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/domain/pairing/continuity_summary.go` | `ContinuitySummary` type and `SummarizeContinuity` pure function |
| `internal/domain/pairing/s495_continuity_summary_test.go` | 17 tests for summary, leg set, annotation, and E2E cross-session |
| `internal/application/analyticalclient/cross_session_pairing_contracts.go` | Query/reply contracts for cross-session pairing |
| `internal/application/analyticalclient/get_cross_session_pairing.go` | Use case: orchestration, discovery, matching, attribution |
| `docs/architecture/cross-session-read-model-and-continuity-attribution.md` | Architecture document |
| `docs/architecture/carryover-read-surfaces-resolved-vs-unresolved-attribution-and-limitations.md` | Semantics, limitations, trade-offs |

### Modified Files

| File | Change |
|------|--------|
| `internal/interfaces/http/handlers/composite.go` | Added `getCrossSessionPairingUseCase` interface, field, deps, and `GetCrossSessionPairing` handler |
| `internal/interfaces/http/routes/analytical.go` | Added `GetCrossSessionPairing` to `AnalyticalFamilyDeps`, registered route |
| `cmd/gateway/compose.go` | Wired `GetCrossSessionPairingUseCase` when both ClickHouse and session gateway are available |
| `cmd/gateway/session_reader.go` | Added `crossSessionSessionAdapter` bridging ListSessions to `CrossSessionSessionReader` |

---

## 5. Test Coverage

| Category | Tests | Status |
|----------|-------|--------|
| ContinuitySummary — empty | 1 | PASS |
| ContinuitySummary — all resolved intra-session | 1 | PASS |
| ContinuitySummary — cross-session pairs | 1 | PASS |
| ContinuitySummary — mixed states | 1 | PASS |
| ContinuitySummary — all open | 1 | PASS |
| ContinuitySummary — full cross-session resolution | 1 | PASS |
| CrossSessionLegSet — extract legs preserves order | 1 | PASS |
| CrossSessionLegSet — session leg index | 1 | PASS |
| AnnotateRoundTrips — cross-session provenance | 1 | PASS |
| AnnotateRoundTrips — artificial unresolved | 1 | PASS |
| AnnotateRoundTrips — genuine unresolved (rejected) | 1 | PASS |
| AnnotateRoundTrips — open (no exit found) | 1 | PASS |
| E2E — two sessions FIFO matching | 1 | PASS |
| E2E — three sessions mixed outcomes | 1 | PASS |
| **S495 new tests** | **14** | **ALL PASS** |
| **Existing S494 tests** | **63** | **ALL PASS** |
| **Other pairing tests** | **3** | **ALL PASS** |
| **Total pairing domain** | **80** | **ALL PASS** |

Zero regressions.

---

## 6. Guard Rail Compliance

| Guard Rail | Status | Evidence |
|-----------|--------|----------|
| GR-1: No write-path changes | COMPLIANT | All code is read-path; no mutations to session, execution, or KV |
| GR-2: No position engine | COMPLIANT | No position accumulator or portfolio model |
| GR-3: No OMS expansion | COMPLIANT | No new order types or cancel/modify |
| GR-4: No multi-exchange scope | COMPLIANT | Binance-only, existing segments |
| GR-5: No new infrastructure | COMPLIANT | No new DB tables, streams, or consumers; queries existing data |
| GR-6: No dashboards | COMPLIANT | No UI, metrics, or Grafana; HTTP API only |
| GR-7: No runtime carry-forward | COMPLIANT | Sessions remain isolated at runtime; continuity is retrospective |
| GR-8: Stage closes independently | COMPLIANT | S496 can build on this without further changes |
| GR-9: No analytics platform inflation | COMPLIANT | Minimal read model, not BI/dashboard |
| GR-10: No generic matching engine | COMPLIANT | Uses existing FIFO; no new matching algorithms |

---

## 7. Limitations

| ID | Limitation | Impact | Mitigation |
|----|-----------|--------|-----------|
| L-S495-1 | Lookback window bounds | Legs beyond window appear open/unresolved | Operator can extend window |
| L-S495-2 | Session boundary time overlap risk | Fill timestamps at session close edge | Low impact; market orders are fast |
| L-S495-3 | No deduplication across session boundaries | Duplicate legs if time overlap | Very low risk; sessions non-overlapping |
| L-S495-4 | Attribution requires both chains | Missing chain → nil attribution | Degrades gracefully |
| L-S495-5 | No intra-session pre-filtering | Re-pairs already-paired legs | No correctness impact; minor cost |
| L-S495-6 | Strategy direction consistency | Mixed long/short won't cross-pair | Rare; documented |
| L-S495-7 | No real-time updates | Point-in-time query | By design; retrospective model |

---

## 8. Architecture Documents

| Document | Purpose |
|----------|---------|
| [`cross-session-read-model-and-continuity-attribution.md`](../architecture/cross-session-read-model-and-continuity-attribution.md) | Read model architecture, data flow, HTTP surface, integration, invariants |
| [`carryover-read-surfaces-resolved-vs-unresolved-attribution-and-limitations.md`](../architecture/carryover-read-surfaces-resolved-vs-unresolved-attribution-and-limitations.md) | Continuity state semantics, resolution metrics, carry-forward rules, limitations |

---

## 9. Preparation for S496

S496 (Review/Reconciliation) can build on S495 to:

1. **Add `FlagCrossSession` to `ReconciliationResult`** — mark cross-session pairs in round-trip review
2. **Compare intra-session vs cross-session resolution rates** — quantify improvement delta
3. **Integrate cross-session continuity into session audit bundles** — enrich session audit with carry-forward status
4. **Surface "improvement delta"** — how many previously `unresolved` legs became `resolved` via cross-session matching

All types and interfaces introduced by S495 are stable and ready for extension.

---

## 10. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Continuity has a minimal read surface | **MET** | `GET /analytical/composite/pairing/cross-session` with full query contract |
| Pairing/effectiveness gain cross-session integration | **MET** | `ClassifyPair` attribution for cross-session round-trips; `ContinuitySummary` |
| Stage reduces dependency on manual reconstruction | **MET** | Automated discovery from KV+ClickHouse; no manual session correlation needed |
| Wave ready for review/reconciliation in S496 | **MET** | Types, interfaces, and HTTP surface are stable; S496 additive only |
