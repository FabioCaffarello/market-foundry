# Carryover Read Surfaces: Resolved vs Unresolved Attribution and Limitations

> S495 — Cross-Session Position Continuity Wave (S493–S497)

## Purpose

This document defines the semantics of carryover read surfaces — the distinction between **resolved**, **open**, **genuine unresolved**, and **artificial unresolved** outcomes when viewing execution data across session boundaries. It documents what is queryable, what remains ambiguous, and what limitations exist.

## Continuity State Taxonomy

### Resolved (`continuity: "resolved"`)

**Definition**: The round-trip has both an entry and exit leg. The pair was matched by FIFO rules across one or more sessions.

**Attribution**: Full `ClassifyPair` attribution is available — outcome (win/loss/breakeven), net P&L, gross P&L, fees, cost basis.

**Subtypes**:
- **Intra-session resolved**: Both legs from the same session. `cross_session: false`.
- **Cross-session resolved**: Entry and exit from different sessions. `cross_session: true`. These are the primary value of this surface — they were previously hidden as `unresolved` in intra-session views.

**Queryability**: `continuity=resolved` filter. `cross_only=true` to see only cross-session pairs.

### Open (`continuity: "open"`)

**Definition**: An entry leg exists without a matching exit across the entire lookback window. This represents a genuinely open position — it may resolve in a future session.

**Attribution**: No P&L attribution is possible. Outcome is structurally `unresolved`.

**Causes**:
- `UnmatchedReason: no_exit_found` — no opposite-side fill exists
- `UnmatchedReason: quantity_mismatch_remainder` — partial fill residue awaiting closure

**Queryability**: `continuity=open` filter.

**Operator guidance**: Open legs are informational. They may resolve when new sessions produce counterpart fills. The operator can extend the lookback window (`max_sessions`, `since`) to search further.

### Genuine Unresolved (`continuity: "genuine_unresolved"`)

**Definition**: The leg cannot resolve due to a structural condition — not because data is missing, but because the execution itself failed or is incomplete.

**Attribution**: No P&L attribution. Permanently unresolved.

**Causes**:
- `UnmatchedReason: rejected_leg` — execution was rejected by the venue
- `UnmatchedReason: cancelled_leg` — execution was cancelled before fill
- `StateUnmatchedExit` — orphan exit with no entry (data gap or error)

**Queryability**: `continuity=genuine_unresolved` filter.

**Operator guidance**: Genuine unresolved legs are permanent conditions. They indicate rejected/cancelled orders or orphan exits. They should be reviewed for operational health but will never resolve through cross-session matching.

### Artificial Unresolved (`continuity: "artificial_unresolved"`)

**Definition**: The leg was classified as unresolved solely because it sits at a session boundary. A counterpart may exist in an adjacent session but was not found within the current lookback window.

**Attribution**: No P&L attribution yet. May become resolved with a wider window.

**Causes**:
- `UnmatchedReason: session_boundary` — the intra-session matching flagged this as a boundary artifact

**Queryability**: `continuity=artificial_unresolved` filter.

**Operator guidance**: Artificial unresolved is the primary target for cross-session resolution. If this count is non-zero after cross-session matching, it means either:
1. The lookback window is too narrow (extend `since` or `max_sessions`)
2. The counterpart exit genuinely hasn't occurred yet (the position is still open but was misclassified at the intra-session level)

## Resolution Metrics

### CarryForwardResolutionRate

```
CarryForwardResolutionRate = cross_session_pairs / (cross_session_pairs + artificial_unresolved_count)
```

This measures how effective cross-session matching was at resolving boundary artifacts.

| Value | Interpretation |
|-------|---------------|
| 1.0 | All boundary artifacts resolved — perfect cross-session matching |
| 0.5–0.99 | Most boundary artifacts resolved; remaining may need wider window |
| 0.0 | No boundary artifacts resolved — counterparts may exist beyond window |
| N/A (0/0) | No boundary artifacts existed — intra-session matching was sufficient |

### CrossSessionResolutionRate

```
CrossSessionResolutionRate = cross_session_paired_count / (cross_session_paired_count + artificial_unresolved_count)
```

Computed in `ContinuitySummary`. Same metric, available in the summary block.

### ResolutionRate

```
ResolutionRate = resolved_count / total
```

Overall resolution rate across all round-trips in the response.

## Carry-Forward Rules (from S494)

Only `CarryEligible` intents are included in cross-session discovery:

| Rule | Condition | Eligible? | Rationale |
|------|-----------|-----------|-----------|
| R-CF1 | Status = rejected | No | No fills → nothing to carry |
| R-CF2 | Cancelled before fill | No | No fill data exists |
| R-CF3 | Non-terminal status | No | Lifecycle incomplete |
| R-CF4 | Terminal, zero fills | No | No leg data |
| R-CF5 | Filled with records | **Yes** | Valid leg for carry-forward |

Intents excluded by these rules are counted in `meta.legs_excluded`.

## Data Flow and Source Authority

| Data | Source | Authority |
|------|--------|-----------|
| Session metadata (ID, start, close, status) | NATS KV | Authoritative |
| Execution chains (fills, status, side, symbol) | ClickHouse | Authoritative |
| Leg direction (entry/exit) | Derived from side + strategy direction | Computed |
| Carry-forward eligibility | S494 `ClassifyCarryForward` | Computed |
| Continuity state | S494 `ClassifyContinuity` | Computed |
| Effectiveness attribution | S476 `ClassifyPair` | Computed |
| Session provenance | S494 `AnnotateRoundTrips` | Computed |

## Limitations

### L-S495-1: Lookback Window Bounds

Cross-session resolution is bounded by the lookback window (`since`/`until`, `max_sessions`). Legs whose counterparts exist beyond the window will appear as `open` or `artificial_unresolved`.

**Mitigation**: Operator can extend the window. Default 30 sessions / 30 days balances cost vs relevance.

### L-S495-2: Session Time Overlap with ClickHouse

Chains are queried per session using `session.StartedAt`/`session.ClosedAt` as time bounds. If an intent's fill timestamp falls slightly outside the session bounds (race condition at close), it may be missed.

**Impact**: Low. Fill timestamps are typically within session bounds. Market orders reach terminal state quickly.

**Mitigation**: None required at this stage. Can be addressed in S496 with overlap margins.

### L-S495-3: No Deduplication Across Session Boundaries

If the same `correlation_id` appears in multiple session time windows (unlikely but possible with time overlap), the leg will appear twice.

**Impact**: Very low. Session boundaries are non-overlapping by design.

**Mitigation**: Future improvement: deduplicate by `correlation_id` after aggregation.

### L-S495-4: Effectiveness Attribution Requires Both Chains

Cross-session round-trip attribution requires both the entry and exit chain to be available in ClickHouse. If one chain was not persisted (e.g., writer failure), attribution will be nil.

**Impact**: Degrades gracefully — round-trip is still paired and classified, only P&L attribution is missing.

### L-S495-5: No Intra-Session Pre-Filtering

The use case queries all `CarryEligible` legs per session, not just unmatched ones. This means legs that were already paired intra-session will be re-paired by the cross-session FIFO run.

**Impact**: None on correctness — FIFO is deterministic and will produce the same intra-session pairs plus any new cross-session pairs. Minor impact on query cost for sessions with many paired legs.

**Mitigation**: Acceptable at current scale. Can be optimized in S496 with pre-paired exclusion.

### L-S495-6: Strategy Direction Consistency Requirement

Cross-session pairing assumes consistent strategy direction across sessions. If session 1 runs a long strategy and session 2 runs a short strategy on the same symbol, legs will not cross-pair correctly (buy→sell and sell→buy directions conflict).

**Impact**: Low in practice. Strategy direction is typically consistent per symbol/source.

### L-S495-7: No Real-Time Updates

The cross-session read model is a point-in-time query. It does not update in real-time as new sessions close or fills arrive.

**Impact**: By design. This is a retrospective read model, not a live feed.

## Trade-Offs

| Decision | Trade-off | Rationale |
|----------|-----------|-----------|
| Query all eligible legs (not just unmatched) | Higher query cost | Simpler logic; deduplication adds complexity for marginal gain |
| Session boundary detection from KV metadata | Depends on session lifecycle correctness | Session metadata is already the authority for session identity |
| FIFO across sessions without pre-filtering | Re-pairs intra-session legs | Deterministic; simplifies reasoning; M1-M7 invariants unchanged |
| Single HTTP endpoint (not split by continuity state) | Larger response for mixed queries | Filters (`continuity`, `cross_only`) narrow results without multiple endpoints |

## Preparation for S496

S496 (Review/Reconciliation) can build on this surface to:

1. Add `FlagCrossSession` to `ReconciliationResult` for cross-session pairs
2. Compare intra-session-only vs cross-session resolution rates
3. Surface "improvement delta" — how many previously unresolved legs became resolved
4. Integrate cross-session continuity into session audit bundles
