# Carryover, Boundary, Fees, Pairing Result — Reconciliation Semantics and Limitations

> S496 — Reconciliation semantics for cross-session carryover fragments.

## Purpose

This document formalizes the reconciliation semantics that apply when execution legs cross session boundaries. It defines how carryover, boundary artifacts, fees, pairing results, and effectiveness outcomes interact — and where the reconciliation model reaches its limits.

## Carryover Fragment Taxonomy

A **carryover fragment** is a filled entry leg from session N that has no matching exit within session N. It may resolve in session N+1, N+2, or never.

### Eligibility Rules (S494)

| Rule | Condition | Carry-Forward? |
|------|-----------|----------------|
| R-CF1 | Status = rejected | No |
| R-CF2 | Status = cancelled, zero fills | No |
| R-CF3 | Status = non-terminal | No |
| R-CF4 | Status = terminal, zero fills | No |
| R-CF5 | Status = terminal, has fills | **Yes** |

Only R-CF5 legs enter the cross-session matching pool.

### Resolution Outcomes

After cross-session FIFO matching, each carryover fragment has one of four continuity states:

| State | Meaning | Reconciliation Impact |
|-------|---------|----------------------|
| `resolved` | Paired with exit in a later session | Full P&L and fee reconciliation possible |
| `open` | No exit found across all sessions in window | P&L unresolvable; fees partial (entry only) |
| `genuine_unresolved` | Structurally unpaired (rejected, cancelled, orphan) | Permanently unresolvable |
| `artificial_unresolved` | Unmatched at session boundary, may resolve with wider window | May resolve with broader lookback |

## Boundary Semantics

A **session boundary** is the temporal gap between session N's close and session N+1's open. Legs at boundaries create reconciliation challenges:

### Time Gap

The idle period between sessions means:
- Entry price reflects market conditions at session N's close.
- Exit price reflects market conditions at session N+1's open (or later).
- The P&L includes the gap — this is real, not an artifact.

### Fee Accumulation

Fees are recorded per fill, per session. When a round-trip spans sessions:
- Entry fees belong to session N.
- Exit fees belong to session M.
- The reconciliation surface aggregates both for the round-trip.

### Flag: `boundary_carryover`

Applied to all resolved cross-session round-trips. Signals that the round-trip includes idle time between sessions.

## Fee Reconciliation Across Sessions

### Normal Case

Both entry and exit have non-zero fees → `fee_reliable = true`.

### Cross-Session Fee Gap

One or both legs have zero fees → `fee_gap` (base) + `cross_session_fee_gap` (cross-session specific).

Causes:
- Venue did not report fees in the fill (e.g., maker rebate credited separately).
- Fee data was not persisted during session close.
- Segment-specific fee behavior (futures vs spot).

### Fee Asset Mismatch

Entry and exit fees use different assets (e.g., BNB for entry, USDT for exit) → `fee_asset_mismatch`.

This is possible across sessions if fee configuration changed between sessions.

## Pairing Result Reconciliation

### Intra-Session Pairs

Standard reconciliation (S482) applies. No cross-session flags added.

### Cross-Session Pairs

Standard reconciliation applies first, then cross-session flags are added:

1. `cross_session` — always present for cross-session pairs.
2. `boundary_carryover` — present for resolved cross-session pairs.
3. `cross_session_fee_gap` — present if fee data is missing on either leg.

### Carryover Reliability Assessment

```
carryover_reliable = fee_reliable AND pnl_reliable AND NOT cross_session_fee_gap
```

This provides a single flag for operators to decide whether to trust the cross-session P&L.

## Effectiveness Reconciliation

### Attribution Rules

Effectiveness attribution uses the same `ClassifyPair` function regardless of session provenance. The P&L computation is:

- Long: `gross_pnl = exit_cost - entry_cost`
- Short: `gross_pnl = entry_cost - exit_cost`
- Net: `net_pnl = gross_pnl - total_fees`

Cross-session pairs may have:
- Entry cost basis from session N's market price.
- Exit cost basis from session M's market price.
- The gap between sessions is reflected in the P&L.

### Split Accounting

The continuity review surface provides split effectiveness summaries:

| Metric | Scope | Purpose |
|--------|-------|---------|
| `cross_session_wins` | Cross-session pairs only | Measure carryover success |
| `cross_session_losses` | Cross-session pairs only | Measure carryover cost |
| `cross_session_pnl` | Cross-session pairs only | Net carryover impact |
| `intra_session_wins` | Same-session pairs only | Baseline comparison |
| `intra_session_losses` | Same-session pairs only | Baseline comparison |
| `intra_session_pnl` | Same-session pairs only | Baseline comparison |

## Limitations

### L1: Lookback Window Bounds

Carryover fragments can only resolve within the lookback window defined by `since`, `until`, and `max_sessions`. A fragment from session 1 will appear as `artificial_unresolved` if its counterpart exists in session 35 but `max_sessions=30`.

**Mitigation**: Increase `max_sessions` or widen the time window. Maximum supported: 50 sessions.

### L2: No Real-Time Carryover Visibility

The reconciliation surface is retrospective. During an active session, you cannot see which legs are accumulating as potential carryover.

**Mitigation**: Use intra-session pairing review (`/analytical/composite/pairing/review`) for the current session.

### L3: Fill Timestamp Precision

ClickHouse stores fill timestamps as DateTime64. NATS KV stores session timestamps as time.Time. At session close boundaries, a fill may be recorded in ClickHouse with a timestamp that falls outside the session's `started_at`/`closed_at` range by a few milliseconds.

**Mitigation**: The use case applies the same ±5 minute buffer as session verification (S485).

### L4: Duplicate Legs Across Session Overlap

If two sessions have overlapping time ranges (e.g., a session was not properly closed before a new one started), the same chain may appear in both sessions' legs, producing duplicate entries.

**Mitigation**: Operational discipline — ensure sessions are closed before starting new ones. The system does not deduplicate across sessions.

### L5: Fee Normalization Across Segments

Spot and futures segments may have different fee models. Cross-session pairs where entry is in a different segment context than exit may have inconsistent fee semantics.

**Mitigation**: The `fee_asset_mismatch` flag detects this. Operators should review cross-segment pairs manually.

### L6: No Aggregated Position Tracking

The reconciliation surface shows individual round-trips, not aggregated positions. Multiple open legs in the same symbol are not netted.

**Mitigation**: This is by design — the scope is round-trip review, not position management.

## Guard Rails

- No position engine or portfolio model.
- No write-path changes.
- No new ClickHouse tables or NATS subjects.
- No runtime carry-forward — sessions remain isolated at runtime.
- No dashboard — operator review surface only.
