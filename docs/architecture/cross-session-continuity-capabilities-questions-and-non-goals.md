# Cross-Session Continuity — Capabilities, Questions, and Non-Goals

**Wave**: Cross-Session Position Continuity (S493–S497)
**Date**: 2026-03-26
**Companion to**: [cross-session-position-continuity-wave-charter-and-scope-freeze.md](cross-session-position-continuity-wave-charter-and-scope-freeze.md)

---

## 1. Capabilities

### 1.1 MUST — Required for wave pass

| ID | Capability | Stage | Description |
|----|-----------|-------|-------------|
| C-CS1 | Cross-session leg discovery query | S494 | Query that finds unmatched legs (entries with `ReasonSessionBoundary`) across a range of prior sessions for a given symbol and segment. Returns legs with their originating session metadata. |
| C-CS2 | Multi-session FIFO pairing execution | S495 | Invocation of the existing `MatchFIFO` algorithm on a combined leg set drawn from multiple sessions. Produces `RoundTrip` instances where entry and exit originate from different sessions. Temporal ordering (M4 invariant) ensures correctness across session boundaries. |
| C-CS3 | Cross-session P&L attribution | S495 | Application of `ClassifyPair()` to cross-session round-trips, producing win/loss/breakeven outcomes with correct entry price, exit price, fees, and cost basis. Lineage traceability preserved via CorrelationID/CausationID from each leg's originating chain. |

### 1.2 SHOULD — Delivered if guard rails hold

| ID | Capability | Stage | Description |
|----|-----------|-------|-------------|
| C-CS4 | Cross-session pairing HTTP query surface | S495 | HTTP endpoint that accepts symbol, segment, and optional time range, returning cross-session pairing results: paired round-trips with session origin for each leg, unmatched legs with reason, summary statistics (resolved rate, P&L). |
| C-CS5 | Continuity reconciliation flags | S496 | Extension of existing `ReconcileRoundTrip` to produce data-quality flags specific to cross-session pairs: `FlagCrossSession`, `FlagSessionGap` (time gap between sessions exceeds threshold), `FlagSegmentMismatch` (defensive — should not occur). |

### 1.3 MAY — Only if trivial

| ID | Capability | Stage | Description |
|----|-----------|-------|-------------|
| C-CS6 | Continuity integration with audit bundle | S496 | Optional section in the existing session audit bundle that summarizes cross-session continuity: how many legs were resolved by pairing with other sessions, how many remain unmatched, and the net P&L impact of cross-session resolution. |

---

## 2. Governing Questions

| ID | Question | Answered by | Success = YES |
|----|----------|-------------|---------------|
| Q-CS1 | Can the system discover unmatched entry legs from prior sessions for a given symbol and segment? | S494 test + S495 integration | Query returns legs filtered by `ReasonSessionBoundary`, scoped to symbol + segment + lookback window |
| Q-CS2 | Can the system pair an entry from session N with an exit from session N+k using existing FIFO rules? | S495 test | `MatchFIFO` produces `RoundTrip` with entry.session ≠ exit.session and correct temporal ordering |
| Q-CS3 | Does cross-session pairing produce accurate P&L attribution with full lineage traceability? | S495 test | `ClassifyPair()` returns correct outcome; attribution carries both entry and exit CorrelationIDs |
| Q-CS4 | Can an operator query cross-session pairing results via HTTP? | S495 HTTP test | Endpoint returns structured response with paired round-trips and per-leg session origin |
| Q-CS5 | Are cross-session pairs distinguishable from intra-session pairs in reconciliation? | S496 test | `ReconcileRoundTrip` output includes `FlagCrossSession` for cross-session pairs |

---

## 3. Non-Goals — Explicit Exclusions

The following are **permanently out of scope** for this wave. Any pressure to
include them triggers a scope-freeze violation and must be deferred to a
future charter.

### 3.1 Position engine and portfolio model

| Exclusion | Rationale |
|-----------|-----------|
| Live position tracking in the write path | This wave is read-side only. No runtime state carry-forward. |
| Position accumulator or inventory model | Positions are computed retrospectively from fills, not maintained as live state. |
| Portfolio-level aggregation across symbols | Pairing is per-symbol, per-segment. No cross-symbol P&L or exposure. |
| Risk engine or exposure limits | No risk model, no margin tracking, no position sizing engine. |
| Net asset value (NAV) computation | No portfolio valuation surface. |

### 3.2 OMS expansion

| Exclusion | Rationale |
|-----------|-----------|
| Limit orders, cancel, or modify lifecycle | Order model unchanged. Market orders only. |
| Multi-order strategy execution | Single-order decision chains only. |
| Order routing or smart order routing | Venue-direct execution unchanged. |
| Bracket orders, OCO, or conditional orders | Not in scope for any near-term wave. |

### 3.3 Multi-exchange and instrument scope

| Exclusion | Rationale |
|-----------|-----------|
| Multi-exchange support | Binance-only. No cross-exchange pairing. |
| New instrument families | Existing spot + futures segments only. |
| Cross-exchange arbitrage or correlation | Not a trading system concern at this stage. |

### 3.4 Dashboards, analytics platform, and observability

| Exclusion | Rationale |
|-----------|-----------|
| Dashboard or visualization UI | HTTP JSON endpoints only. No frontend. |
| Real-time WebSocket surfaces | Request/response HTTP only. |
| Prometheus/Grafana integration | No metrics in this wave. |
| ML-based analytics or backtesting | No predictive modeling. |
| Statistical significance testing | No hypothesis testing on effectiveness cohorts. |
| Temporal trend analysis across sessions | Out of scope. Trend analysis is a separate concern. |

### 3.5 Execution model and runtime changes

| Exclusion | Rationale |
|-----------|-----------|
| Runtime state carry-forward between sessions | Sessions remain isolated at runtime. |
| Write-path modifications to fills, orders, or lifecycle | Read-side only. |
| Session orchestration or auto-restart | No session lifecycle automation beyond what exists. |
| Decision replay or simulation | Not in scope. |
| Strategy feedback loop from effectiveness | Separate concern. |

### 3.6 Infrastructure

| Exclusion | Rationale |
|-----------|-----------|
| New databases or message brokers | Reuses ClickHouse + NATS KV. |
| New ClickHouse tables or materialized views | Queries existing execution tables. |
| Schema migrations | No DDL changes. |
| New NATS streams or consumers | Read-only queries. |

---

## 4. Boundary Clarifications

### 4.1 Cross-session pairing vs position tracking

Cross-session pairing is **retrospective**: given a set of fills across
sessions, match entries with exits using FIFO rules. It does not imply or
require live position tracking. The system does not "know" it has an open
position — it discovers, after the fact, that an unmatched entry from a prior
session was resolved by an exit in a later session.

### 4.2 Multi-session window vs unbounded lookback

The cross-session discovery query operates within a bounded lookback window
(configurable, default: 30 days or 30 sessions). It does not scan the entire
history. This is a performance and relevance constraint, not a domain
limitation.

### 4.3 Intra-session pairing unchanged

Existing intra-session pairing (S479–S483) is not modified. Cross-session
pairing is an additional query path that operates on a superset of the
intra-session leg set. When an entry and exit occur in the same session, they
are paired by intra-session logic. Cross-session pairing only resolves legs
that intra-session logic left unmatched.

### 4.4 Reconciliation flag semantics

The `FlagCrossSession` reconciliation flag is informational. It does not
change the reliability assessment of the round-trip. A cross-session pair
with valid prices and fees is equally reliable as an intra-session pair.
The flag enables the operator to distinguish the two for audit purposes.
