# Cross-Session Position Continuity Wave — Charter and Scope Freeze

**Wave**: Cross-Session Position Continuity (S493–S497)
**Status**: OPEN — Scope Frozen
**Date**: 2026-03-26
**Predecessor**: S492 (Operational Automation Closure Evidence Gate — PASS)

---

## 1. Strategic Context

The system has consolidated a robust intra-session stack: session intelligence
(S459–S463), decision quality (S469–S473), effectiveness measurement
(S474–S478), round-trip pairing (S479–S483), and operational automation
(S484–S492). Within a single session, order lifecycle, pairing, P&L
attribution, lineage, and verification are complete and auditable.

The structural bottleneck is no longer intra-session. It is the boundary
between sessions.

When a session closes with an unmatched entry leg, the next session that
produces the corresponding exit has no mechanism to:

1. Discover that the entry exists in a prior session.
2. Pair the exit with that entry to form a complete round-trip.
3. Attribute the resulting P&L to the correct decision chain.
4. Reconcile the continuity across session boundaries for audit.

This produces artificial `unresolved` outcomes in effectiveness measurement,
inflates unmatched-entry counts in pairing, and prevents the operator from
understanding true position continuity across trading days.

### 1.1 Why now

- **G-RT4** (round-trip pairing wave, S483): explicitly flagged cross-session
  position continuity as a MEDIUM gap. No new infrastructure or live session
  is required to close it.
- The pairing domain (`MatchFIFO`, `IntentToLeg`) is session-agnostic — it
  accepts any list of legs regardless of origin session.
- The effectiveness domain (`ClassifyPair`) requires only entry and exit
  intents — no session binding.
- The `CompositeReader` already supports arbitrary time-range queries, not
  limited to single-session windows.
- Session metadata (start/end timestamps, segment, symbol) is queryable via
  KV and ClickHouse, enabling multi-session window construction.

### 1.2 Why this shape

This wave is **read-side only**. It does not introduce position tracking in
the write path, does not carry state forward at runtime, and does not create
a portfolio or position engine. It extends existing pairing and effectiveness
infrastructure to operate across session boundaries using data that already
exists in ClickHouse and NATS KV.

---

## 2. Problem Statement

### 2.1 What exists today

- **Pairing**: FIFO matching within a single session's correlation-ID boundary.
  Entry and exit legs from the same session are paired. Unmatched entries at
  session close receive `ReasonSessionBoundary`.
- **Effectiveness**: Classification runs on paired round-trips within session
  scope. Single-leg fills → `OutcomeUnresolved`.
- **Session metadata**: Each session has UUID, timestamps, segment, symbol,
  execution mode. Queryable via `GET /analytical/session/list`.
- **Composite reader**: Can query execution chains by source, symbol, timeframe,
  and arbitrary `since`/`until` time range.

### 2.2 What is missing

1. **Cross-session leg discovery**: No query finds unmatched entries from prior
   sessions that could pair with exits in a later session.
2. **Cross-session pairing execution**: No use case runs FIFO matching across
   a multi-session window.
3. **Continuity attribution**: No mechanism attributes a cross-session round-trip
   to the originating decision chain and computes its P&L.
4. **Continuity review surface**: No HTTP endpoint exposes cross-session
   pairing results, continuity state, or reconciliation flags.
5. **Continuity reconciliation**: No quality assessment flags distinguish
   cross-session pairs from intra-session pairs for audit purposes.

### 2.3 What this wave delivers

A read-side continuity model that discovers unmatched legs across sessions,
pairs them using existing FIFO logic, attributes P&L, and exposes the results
through queryable review surfaces — all without modifying the write path or
runtime execution model.

---

## 3. Wave Structure

| Stage | Role | Deliverable |
|-------|------|-------------|
| **S493** | Charter and scope freeze | This document. Formally opens the wave. |
| **S494** | Canonical cross-session continuity model | Domain types for cross-session leg discovery, multi-session window, and continuity state. Extension of existing pairing model. |
| **S495** | Cross-session read model and continuity attribution | Use case that queries across session boundaries, executes FIFO matching on the combined leg set, and computes attributed P&L. HTTP surface for cross-session pairing results. |
| **S496** | Continuity review and reconciliation | Reconciliation flags for cross-session pairs, integration with existing audit bundle, and review surface for continuity state. |
| **S497** | Evidence gate | Formal gate against this charter. Evaluates all capabilities and governing questions. |

### 3.1 Stage dependencies

```
S493 (charter) → S494 (model) → S495 (read model + attribution) → S496 (review + reconciliation) → S497 (gate)
```

S494 must land before S495 because the read model depends on the continuity
types. S496 depends on S495 for reconciliation input. S497 evaluates the full
chain.

---

## 4. Capabilities

| ID | Capability | Target | Source |
|----|-----------|--------|--------|
| C-CS1 | Cross-session leg discovery query | S494 | G-RT4 |
| C-CS2 | Multi-session FIFO pairing execution | S495 | G-RT4 |
| C-CS3 | Cross-session P&L attribution | S495 | G-RT4 |
| C-CS4 | Cross-session pairing HTTP query surface | S495 | New |
| C-CS5 | Continuity reconciliation flags | S496 | New |
| C-CS6 | Continuity integration with audit bundle | S496 | New |

### 4.1 Priority tiers

| Tier | Capabilities | Rule |
|------|-------------|------|
| **MUST** | C-CS1, C-CS2, C-CS3 | Required for wave pass |
| **SHOULD** | C-CS4, C-CS5 | Delivered if guard rails hold |
| **MAY** | C-CS6 | Only if trivial composition of existing audit bundle |

---

## 5. Governing Questions

| ID | Question | Success = YES |
|----|----------|---------------|
| Q-CS1 | Can the system discover unmatched entry legs from prior sessions for a given symbol and segment? | Query returns legs with `ReasonSessionBoundary` from prior sessions |
| Q-CS2 | Can the system pair an entry from session N with an exit from session N+k using existing FIFO rules? | Cross-session round-trip produced with correct temporal ordering |
| Q-CS3 | Does cross-session pairing produce accurate P&L attribution with full lineage traceability? | `ClassifyPair()` returns win/loss/breakeven with correct entry/exit prices and fees |
| Q-CS4 | Can an operator query cross-session pairing results via HTTP? | Endpoint returns paired round-trips with session origin metadata |
| Q-CS5 | Are cross-session pairs distinguishable from intra-session pairs in reconciliation? | Reconciliation flags identify cross-session boundary and originating sessions |

### 5.1 Success criteria

- Q-CS1 through Q-CS3: all **YES** required for PASS.
- Q-CS4 and Q-CS5: **YES** for FULL PASS, **NO** acceptable for CONDITIONAL PASS.
- All MUST capabilities at FULL or SUBSTANTIAL.
- Zero regressions across affected packages.
- No CRITICAL or HIGH residual gaps.
- Existing intra-session pairing behavior unchanged.

---

## 6. Guard Rails

| # | Guard Rail | Rationale |
|---|-----------|-----------|
| GR-1 | No write-path changes to order lifecycle or execution model | This is a read-side wave. Runtime execution is not modified. |
| GR-2 | No position engine or portfolio model | Cross-session pairing is retrospective matching, not live position tracking. |
| GR-3 | No OMS expansion (limit orders, cancels, multi-order strategies) | Out of scope. Not related to continuity. |
| GR-4 | No multi-exchange or new instrument scope | Binance-only, existing segments only. |
| GR-5 | No new infrastructure dependencies | Reuses ClickHouse, NATS KV, existing HTTP framework. |
| GR-6 | No dashboard, analytics platform, or observability redesign | HTTP endpoints only. No UI, no Prometheus, no Grafana. |
| GR-7 | No carry-forward state in runtime | Sessions remain isolated at runtime. Continuity is computed after the fact. |
| GR-8 | Each stage closes independently | If S496 threatens guard rails, skip to S497. |

---

## 7. Scope Boundary — What Enters vs What Stays Out

### 7.1 IN scope

- Domain types for cross-session continuity: multi-session window, cross-session
  leg set, continuity state enum.
- Extension of `MatchFIFO` invocation to accept legs from multiple sessions.
- Use case that constructs a multi-session leg set from ClickHouse queries.
- P&L attribution via existing `ClassifyPair()` for cross-session round-trips.
- HTTP endpoint for cross-session pairing results.
- Reconciliation flags that mark cross-session pairs.
- Optional integration with existing audit bundle composition.

### 7.2 OUT of scope (Non-goals)

See companion document: [cross-session-continuity-capabilities-questions-and-non-goals.md](cross-session-continuity-capabilities-questions-and-non-goals.md)

---

## 8. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Cross-session queries become expensive for long time ranges | MEDIUM | LOW | Enforce maximum lookback window (e.g. 30 sessions or 30 days). Paginate results. |
| Ambiguous pairing when multiple sessions have unmatched entries for same symbol | LOW | MEDIUM | FIFO temporal ordering already handles this — oldest unmatched entry pairs first. |
| Scope inflation from "just add position tracking" | MEDIUM | HIGH | GR-2 and GR-7 enforced. Retrospective matching only. |
| Fee inconsistency across sessions with different fee schedules | LOW | LOW | Fees recorded at fill time; P&L uses recorded fees, not current schedule. |

---

## 9. Predecessor Artifacts Consumed

| Artifact | Source | Used By |
|----------|--------|---------|
| `MatchFIFO` pairing algorithm | S480 | S495 (cross-session matching) |
| `IntentToLeg` converter | S480 | S494 (leg construction from multi-session data) |
| `ClassifyPair()` effectiveness | S475 | S495 (cross-session P&L attribution) |
| `CompositeReader.QueryChainsBatch` | S297 | S495 (multi-session data retrieval) |
| `ReconcileRoundTrip` quality flags | S481 | S496 (cross-session reconciliation) |
| Session metadata KV store | S460 | S494 (session window discovery) |
| `GET /analytical/session/list` | S465 | S494 (session enumeration) |
| Pairing HTTP routes | S482 | S495 (extension point) |
| Audit bundle composition | S462, S491 | S496 (optional integration) |
| `ReasonSessionBoundary` unmatched reason | S480 | S494 (discovery filter) |

---

## 10. Definition of Done

The wave is **DONE** when:

1. S497 evidence gate returns PASS or CONDITIONAL PASS.
2. All MUST capabilities (C-CS1, C-CS2, C-CS3) at FULL or SUBSTANTIAL.
3. All required governing questions (Q-CS1, Q-CS2, Q-CS3) answered YES.
4. Zero regressions across all affected packages.
5. Existing intra-session pairing and effectiveness behavior unchanged.
6. All guard rails respected.
