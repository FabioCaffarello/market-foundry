# Stage S493 — Cross-Session Position Continuity Charter Report

**Stage**: S493
**Type**: Charter and Scope Freeze
**Status**: COMPLETE
**Date**: 2026-03-26
**Wave**: Cross-Session Position Continuity (S493–S497)
**Predecessor**: S492 (Operational Automation Closure Evidence Gate — PASS)

---

## 1. Executive Summary

S493 formally opens the Cross-Session Position Continuity wave. The system's
intra-session capabilities — session intelligence, decision quality,
effectiveness measurement, round-trip pairing, and operational automation —
are consolidated and proven through S492. The next structural bottleneck is
the session boundary itself: unmatched legs at session close produce artificial
`unresolved` outcomes that degrade effectiveness measurement and pairing
accuracy.

This charter defines a short, disciplined wave (4 execution stages + gate)
that extends existing read-side pairing and effectiveness infrastructure to
operate across session boundaries. No write-path changes, no position engine,
no portfolio model.

**Key decision**: this wave is explicitly read-side only. Cross-session
continuity is computed retrospectively from existing fill data, not tracked
as live runtime state.

---

## 2. Problem Analysis

### 2.1 Current state (post-S492)

| Capability | Status | Limitation |
|-----------|--------|-----------|
| Round-trip pairing (FIFO) | FULL | Scoped to single session |
| Effectiveness classification | FULL | `unresolved` for unmatched legs |
| P&L attribution | FULL | Only for intra-session pairs |
| Session metadata | FULL | Queryable but not linked across sessions |
| Composite reader | FULL | Supports arbitrary time ranges |
| Reconciliation flags | FULL | No cross-session awareness |
| Operational automation | FULL | Session-scoped verification and reporting |

### 2.2 Gap analysis

The gap G-RT4 (identified in S483, round-trip pairing evidence gate) states:

> Sessions are isolated; no carry-forward of positions/P&L across sessions.
> Unmatched entries at session close receive `ReasonSessionBoundary` and are
> not resolved even when a matching exit exists in a subsequent session.

**Impact quantification**:

- Every session that closes with open positions contributes unmatched entries.
- These entries inflate the `unresolved` count in effectiveness measurement.
- The operator cannot determine true win/loss rates for strategies that span
  session boundaries (e.g., overnight holds, multi-day positions).
- Audit accuracy degrades because the true position outcome is known but not
  computed.

### 2.3 Why existing infrastructure is sufficient

| Infrastructure | How it helps |
|---------------|-------------|
| `MatchFIFO` algorithm | Session-agnostic — accepts any leg list |
| `IntentToLeg` converter | Works with any `ExecutionIntent` regardless of session |
| `ClassifyPair()` | Requires only entry + exit intents |
| `CompositeReader.QueryChainsBatch` | Accepts arbitrary `since`/`until` |
| Session metadata KV | Provides session windows for multi-session construction |
| `ReasonSessionBoundary` | Explicit marker for discoverable unmatched legs |
| Existing HTTP route patterns | Extensible for new analytical endpoints |

No new databases, streams, or infrastructure dependencies are required.

---

## 3. Wave Charter

### 3.1 Promoted documents

| Document | Location |
|----------|----------|
| Charter and scope freeze | [`docs/architecture/cross-session-position-continuity-wave-charter-and-scope-freeze.md`](../architecture/cross-session-position-continuity-wave-charter-and-scope-freeze.md) |
| Capabilities, questions, and non-goals | [`docs/architecture/cross-session-continuity-capabilities-questions-and-non-goals.md`](../architecture/cross-session-continuity-capabilities-questions-and-non-goals.md) |

### 3.2 Wave structure

| Stage | Role | Deliverable |
|-------|------|-------------|
| S493 | Charter and scope freeze | This report + 2 architecture documents |
| S494 | Canonical cross-session continuity model | Domain types, discovery query, multi-session window |
| S495 | Cross-session read model and continuity attribution | Use case, FIFO matching across sessions, P&L attribution, HTTP surface |
| S496 | Continuity review and reconciliation | Reconciliation flags, audit bundle integration, review surface |
| S497 | Evidence gate | Formal evaluation against charter |

### 3.3 Capabilities summary

| ID | Capability | Priority | Stage |
|----|-----------|----------|-------|
| C-CS1 | Cross-session leg discovery query | MUST | S494 |
| C-CS2 | Multi-session FIFO pairing execution | MUST | S495 |
| C-CS3 | Cross-session P&L attribution | MUST | S495 |
| C-CS4 | Cross-session pairing HTTP query surface | SHOULD | S495 |
| C-CS5 | Continuity reconciliation flags | SHOULD | S496 |
| C-CS6 | Continuity integration with audit bundle | MAY | S496 |

---

## 4. Governing Questions

| ID | Question | Success = YES |
|----|----------|---------------|
| Q-CS1 | Can the system discover unmatched entry legs from prior sessions? | Query returns legs with `ReasonSessionBoundary` from prior sessions |
| Q-CS2 | Can the system pair entries and exits across session boundaries? | Cross-session `RoundTrip` with correct temporal ordering |
| Q-CS3 | Does cross-session pairing produce accurate P&L with lineage? | Correct win/loss/breakeven with both CorrelationIDs |
| Q-CS4 | Can the operator query cross-session pairing via HTTP? | Structured response with per-leg session origin |
| Q-CS5 | Are cross-session pairs distinguishable in reconciliation? | `FlagCrossSession` present on cross-session pairs |

**Pass criteria**: Q-CS1 through Q-CS3 all YES → PASS. Q-CS4 and Q-CS5 also
YES → FULL PASS.

---

## 5. Non-Goals — Explicit Exclusions

| Category | What stays out |
|----------|---------------|
| **OMS expansion** | Limit orders, cancel/modify, multi-order strategies, order routing |
| **Position engine** | Live position tracking, inventory model, position accumulator |
| **Portfolio model** | Cross-symbol aggregation, NAV, risk engine, exposure limits |
| **Multi-exchange** | Cross-exchange pairing, new venue adapters |
| **Dashboards** | UI, visualization, Grafana, real-time WebSocket |
| **Observability platform** | Prometheus, alerting, distributed tracing |
| **Runtime changes** | Write-path modifications, session state carry-forward, auto-restart |
| **Infrastructure** | New databases, tables, materialized views, schema migrations |
| **Analytics** | ML/backtesting, statistical significance, trend analysis, Sharpe/Sortino |

Full non-goals list in companion document.

---

## 6. Guard Rails

| # | Rule |
|---|------|
| GR-1 | No write-path changes |
| GR-2 | No position engine or portfolio model |
| GR-3 | No OMS expansion |
| GR-4 | No multi-exchange scope |
| GR-5 | No new infrastructure dependencies |
| GR-6 | No dashboards or observability platform |
| GR-7 | No runtime state carry-forward |
| GR-8 | Each stage closes independently |

---

## 7. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Cross-session queries expensive for long ranges | MEDIUM | LOW | Bounded lookback window (30 days/sessions) |
| Ambiguous pairing with multiple unmatched entries | LOW | MEDIUM | FIFO temporal ordering resolves deterministically |
| Scope inflation toward position engine | MEDIUM | HIGH | GR-2 + GR-7 enforced |
| Fee inconsistency across sessions | LOW | LOW | Fees recorded at fill time |

---

## 8. Preparation for S494

S494 should:

1. Define `CrossSessionWindow` type (list of session IDs or time range with
   segment + symbol filter).
2. Define `CrossSessionLegSet` — ordered collection of legs from multiple
   sessions, each annotated with originating session ID.
3. Implement discovery query: find all `ExecutionIntent` records with
   `ReasonSessionBoundary` unmatched status within the lookback window.
4. Add session origin metadata to `Leg` type (or a wrapper) so that
   cross-session `RoundTrip` results carry provenance.
5. Test: discovery query returns correct legs across 2+ sessions.
6. Test: leg ordering preserves temporal invariant (M4) across sessions.

**Key files to extend**:
- `internal/domain/pairing/pairing.go` — `Leg` type extension or wrapper
- `internal/application/analyticalclient/` — new or extended use case
- `internal/adapters/clickhouse/composite_reader.go` — multi-session query

---

## 9. Artifacts Produced

| Artifact | Type | Location |
|----------|------|----------|
| Wave charter | Architecture | `docs/architecture/cross-session-position-continuity-wave-charter-and-scope-freeze.md` |
| Capabilities and non-goals | Architecture | `docs/architecture/cross-session-continuity-capabilities-questions-and-non-goals.md` |
| Stage report | Stage | This document |

---

## 10. Conclusion

The Cross-Session Position Continuity wave is formally open with scope frozen.
The wave is short (4 stages + gate), read-side only, and builds entirely on
proven infrastructure. It closes the most impactful residual gap (G-RT4) from
the round-trip pairing wave without introducing position tracking, portfolio
models, or runtime changes.

Next stage: **S494 — Canonical Cross-Session Continuity Model**.
