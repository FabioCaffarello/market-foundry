# Final State, Boundary, and Carryover Resolution: Semantics and Limitations

**Stage**: S500
**Status**: Complete

## 1. Purpose

This document defines the canonical semantics for final states, session boundary conditions, and carryover resolution in the market-foundry lifecycle. It consolidates rules from S480 (pairing), S494 (continuity), S496 (continuity reconciliation), S499 (fee provenance), and S500 (lifecycle close hardening) into a single reference.

## 2. Final State Semantics

### 2.1 Execution Intent Final States

An execution intent reaches a final state when its `Status.IsTerminal()` returns true:

| Status | Terminal | Fills Expected | Carry-Forward Eligible |
|--------|----------|---------------|----------------------|
| `submitted` | No | No | No (R-CF3) |
| `sent` | No | No | No (R-CF3) |
| `accepted` | No | No | No (R-CF3) |
| `partially_filled` | No | Yes (partial) | No (R-CF3) |
| `filled` | Yes | Yes | Yes if fills > 0 (R-CF5) |
| `rejected` | Yes | No | No (R-CF1) |
| `cancelled` | Yes | Maybe | Yes if fills > 0 (R-CF5), No if fills == 0 (R-CF2) |

**Key invariant**: Only terminal intents with actual fill records produce legs eligible for pairing. Non-terminal intents are excluded from carry-forward regardless of whether fills exist, because their lifecycle is incomplete.

### 2.2 Session Final States

| Status | Terminal | Description |
|--------|----------|-------------|
| `open` | No | Session is actively processing |
| `closed` | Yes | Graceful shutdown with complete counters |
| `halted` | Yes | Forced shutdown with reason (kill-switch, error) |

**S500 invariants**:
- A terminal session cannot be closed or halted again (idempotency guard)
- `ClosedAt` must not precede `StartedAt` (temporal ordering)
- `HaltReason` must be set when status is `halted`

### 2.3 Pairing Final States

| State | Meaning | Continuity |
|-------|---------|-----------|
| `paired` | Both entry and exit legs matched | `resolved` |
| `unmatched_entry` | Entry without matching exit | Depends on reason |
| `unmatched_exit` | Exit without matching entry (orphan) | `genuine_unresolved` |

## 3. Session Boundary Conditions

### 3.1 The Boundary Gap

When an entry fill occurs in session N and the corresponding exit fill occurs in session N+1, intra-session pairing cannot see both legs. The entry is marked `unmatched_entry` with `ReasonSessionBoundary` and the exit may appear as `unmatched_exit` with `ReasonNoEntryFound`.

Cross-session pairing resolves this by collecting legs from multiple sessions and running FIFO matching on the combined set.

### 3.2 Non-Terminal Orders at Boundary

Orders in `submitted`, `sent`, `accepted`, or `partially_filled` state when a session closes are:
- **Not carried forward** (R-CF3): Their lifecycle is incomplete
- **Not cancelled by the system**: The venue may still fill them
- **Invisible to next session**: Execute binary starts clean
- **S500**: Surfaced via `InFlight` counter in `SessionSegmentCounters`
- **S500**: Flagged via `FlagNonTerminalAtClose` in reconciliation when `LifecycleCloseContext` is provided

### 3.3 Halted Session Boundary

A halted session (kill-switch, error) introduces additional uncertainty:
- Orders may have been in-flight when halt occurred
- Counters may be incomplete if the tracker was not fully flushed
- **S500**: Flagged via `FlagHaltedSessionOrigin` in reconciliation
- **S500**: Carryover reliability is automatically degraded

### 3.4 Timestamp Boundary Cases

| Scenario | Behavior |
|----------|---------|
| Entry and exit same timestamp | M4 allows (entry <= exit satisfied) |
| Fill timestamp == session close timestamp | Fill belongs to current session |
| Fill timestamp after session close | Not observable (execute binary has stopped) |

## 4. Carryover Resolution Rules

### 4.1 Carry-Forward Eligibility (R-CF1 through R-CF5)

Classification is a pure function of the execution intent:

1. **R-CF1**: Rejected → `ineligible_rejected`
2. **R-CF2**: Cancelled, zero fills → `ineligible_cancelled`
3. **R-CF3**: Non-terminal → `ineligible_non_terminal`
4. **R-CF4**: Terminal, zero fills → `ineligible_no_fills`
5. **R-CF5**: Terminal with fills → `eligible`

After R-CF1–5, a second filter removes legs already paired within the originating session (`ineligible_already_paired`).

### 4.2 Cross-Session Matching

Eligible legs from multiple sessions are combined into a `CrossSessionLegSet`, sorted by timestamp, and matched with FIFO:

- **M1**: Same symbol
- **M2**: Same source/segment (no cross-segment pairing)
- **M3**: Opposite side
- **M4**: Temporal ordering (entry ≤ exit)
- **M5**: FIFO priority
- **M6**: One-to-one (no double counting)
- **M7**: Deterministic

### 4.3 Continuity Classification (C-1 through C-6)

| Rule | Condition | Continuity | Actionability |
|------|-----------|-----------|---------------|
| C-1 | Paired | `resolved` | Closed |
| C-2 | Unmatched entry, session boundary | `artificial_unresolved` | Resolvable |
| C-3 | Unmatched entry, rejected/cancelled leg | `genuine_unresolved` | Permanent |
| C-4 | Unmatched entry, no exit found | `open` | May resolve |
| C-5 | Unmatched exit (orphan) | `genuine_unresolved` | Investigate |
| C-6 | Unmatched entry, quantity remainder | `open` | May resolve |

### 4.4 Carryover Reliability

A cross-session round-trip is `carryover_reliable` when ALL of:
1. `fee_reliable = true` on both legs
2. `pnl_reliable = true` (outcome classifiable, non-zero cost basis)
3. No `cross_session_fee_gap` flag
4. **S500**: No `halted_session_origin` flag
5. **S500**: No `non_terminal_at_close` flag

## 5. Reconciliation Flags at Lifecycle Close

### 5.1 Standard Flags (S482/S499)

| Flag | Trigger |
|------|---------|
| `fee_gap` | Zero fees on either leg |
| `cost_basis_zero` | Zero cost basis on either leg |
| `simulated` | Paper/dry-run fill |
| `partial_remainder` | Quantity split remainder |
| `unmatched_open` | Entry without exit |
| `orphan_exit` | Exit without entry |
| `fee_asset_mismatch` | Different fee assets on legs |
| `outcome_unresolved` | Paired but outcome unclassifiable |
| `fee_ratio_anomaly` | Fee > 10% of cost basis |
| `fee_source_fallback` | Unexpected fallback fee path |

### 5.2 Cross-Session Flags (S496)

| Flag | Trigger |
|------|---------|
| `cross_session` | Entry and exit from different sessions |
| `boundary_carryover` | Resolved after crossing session boundary |
| `cross_session_fee_gap` | Fee gap on cross-session pair |

### 5.3 Lifecycle Close Flags (S500)

| Flag | Trigger | Effect |
|------|---------|--------|
| `non_terminal_at_close` | Leg from intent that was non-terminal at session close | Degrades carryover reliability |
| `halted_session_origin` | Leg from halted session | Degrades carryover reliability |

## 6. Limitations

| ID | Limitation | Impact | Mitigation |
|----|-----------|--------|-----------|
| L-1 | No runtime carry-forward | System doesn't know open positions at session start | Read-path continuity is retrospective only |
| L-2 | Strategy direction consistency | Mixed long/short for same symbol breaks pairing | Operator responsibility |
| L-3 | Non-terminal orders at close | May be filled post-session, invisible to system | S500 InFlight counter + flag |
| L-4 | Finite lookback window (30 days) | Long-held positions may appear unresolved | Operator can increase window |
| L-5 | No cross-symbol pairing | By design | Per-symbol analysis only |
| L-6 | Fee schedule changes between sessions | Entry/exit at different fee rates | Correct; operator awareness |
| L-7 | No post-session fill ingestion | Abandoned orders may be filled by venue | By design; no retrospective ingestion |
| L-8 | LifecycleCloseContext is caller-provided | Not automatically derived from session data | Read-path callers must supply metadata |
| L-9 | Halted session data may be incomplete | Counters depend on tracker flush state | Advisory only; flag degrades reliability |

## 7. Interaction with Existing Systems

### 7.1 Effectiveness (S484/S486)

Effectiveness analysis (`Attribution`) depends on paired round-trips. S500 hardening:
- Does not change how attributions are computed
- Flags round-trips with potentially unreliable data
- `CarryoverReliable` gates whether effectiveness metrics should be included in batch analysis

### 7.2 Verification Trigger (S490)

The event-driven verification trigger fires on session close/halt. S500 hardening:
- Close/Halt now return errors for double-invocation
- Execute supervisor handles this by logging and skipping
- Verification trigger is unaffected (fires on published lifecycle event)

### 7.3 Fee Provenance (S499)

S500 flags compose with S499 fee provenance:
- `FeeSourceUnavailable` (Futures) → `isFeeReliableLeg` still returns true
- `FlagFeeSourceFallback` (unexpected) → independent of lifecycle close flags
- Both S499 and S500 flags can appear on the same round-trip
