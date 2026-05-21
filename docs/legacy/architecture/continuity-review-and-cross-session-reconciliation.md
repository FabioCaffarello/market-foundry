# Continuity Review and Cross-Session Reconciliation

> S496 — Continuity review surface and cross-session reconciliation.

## Purpose

This document defines the continuity review surface: a unified, operator-facing read model that answers **"what was carried between sessions, what resolved, what remains open, and how reliable is the data?"**

The surface combines four existing capabilities into a single query:

1. **Cross-session pairing** (S494/S495) — multi-session FIFO matching with continuity classification.
2. **Reconciliation** (S482) — data-quality flags per round-trip.
3. **Effectiveness attribution** (S476) — P&L classification for paired round-trips.
4. **Continuity classification** (S494) — resolved, open, genuine_unresolved, artificial_unresolved.

## Surface Contract

### Endpoint

```
GET /analytical/composite/pairing/continuity-review
    ?source=binance_spot
    &symbol=BTCUSDT
    &timeframe=60
    &since=1711234567
    &until=1711320967       (optional, defaults to now)
    &max_sessions=30        (optional, default 30, max 50)
    &continuity=resolved    (optional: resolved|open|genuine_unresolved|artificial_unresolved)
    &cross_only=true        (optional: only cross-session round-trips)
    &flagged=true           (optional: only round-trips with reconciliation flags)
    &outcome=win            (optional: win|loss|breakeven|unresolved)
```

### Response Structure

```json
{
  "reviews": [
    {
      "entry": { "...leg..." },
      "exit": { "...leg..." },
      "state": "paired",
      "symbol": "BTCUSDT",
      "source": "binance_spot",
      "entry_session_id": "session_20260320_100000",
      "exit_session_id": "session_20260321_100000",
      "cross_session": true,
      "continuity": "resolved",
      "attribution": {
        "outcome": "win",
        "net_pnl": 95.0,
        "gross_pnl": 100.0,
        "total_fees": 5.0
      },
      "reconciliation": {
        "flags": ["cross_session", "boundary_carryover"],
        "clean": false,
        "fee_reliable": true,
        "pnl_reliable": true,
        "continuity": "resolved",
        "cross_session": true,
        "entry_session_id": "session_20260320_100000",
        "exit_session_id": "session_20260321_100000",
        "carryover_reliable": true
      }
    }
  ],
  "continuity": {
    "resolved_count": 5,
    "open_count": 1,
    "genuine_unresolved_count": 1,
    "artificial_unresolved_count": 0,
    "cross_session_paired_count": 3,
    "intra_session_paired_count": 2,
    "resolution_rate": 0.714,
    "cross_session_resolution_rate": 1.0
  },
  "reconciliation": {
    "total": 7,
    "clean_count": 2,
    "flagged_count": 5,
    "cross_session_count": 3,
    "boundary_carryover_count": 3,
    "flag_counts": {
      "cross_session": 3,
      "boundary_carryover": 3,
      "fee_gap": 1
    },
    "carryover_reliable_count": 2,
    "fee_reliable_count": 5,
    "pnl_reliable_count": 5
  },
  "effectiveness": {
    "total_paired": 5,
    "win_count": 3,
    "loss_count": 2,
    "total_net_pnl": 45.50,
    "total_fees": 12.00,
    "cross_session_wins": 2,
    "cross_session_losses": 1,
    "cross_session_pnl": 30.00,
    "intra_session_wins": 1,
    "intra_session_losses": 1,
    "intra_session_pnl": 15.50
  },
  "source": "clickhouse+kv",
  "meta": {
    "total_ms": 342,
    "sessions_fetched": 8,
    "chains_scanned": 156,
    "legs_produced": 42,
    "round_trips": 7,
    "reviewed": 7
  }
}
```

## What the Surface Answers

| Question | Where to Look |
|----------|---------------|
| How many legs resolved across sessions? | `continuity.cross_session_paired_count` |
| How many legs remain open/unresolved? | `continuity.open_count`, `continuity.artificial_unresolved_count` |
| Is the cross-session P&L data reliable? | `reconciliation.carryover_reliable_count` vs `reconciliation.cross_session_count` |
| What is the P&L contributed by cross-session pairs? | `effectiveness.cross_session_pnl` |
| Are there fee gaps in carried legs? | `reconciliation.flag_counts["cross_session_fee_gap"]` |
| Which round-trips have data quality issues? | Query with `flagged=true` |
| What outcome did cross-session pairs produce? | `effectiveness.cross_session_wins`, `cross_session_losses` |

## Cross-Session Reconciliation Flags

S496 introduces three new reconciliation flags specific to cross-session context:

| Flag | Meaning | Operator Action |
|------|---------|-----------------|
| `cross_session` | Entry and exit come from different sessions. | Informational — verify session continuity. |
| `boundary_carryover` | The round-trip resolved after crossing a session boundary. | Check that the idle gap did not affect price assumptions. |
| `cross_session_fee_gap` | Fees missing on one or both legs of a cross-session pair. | Investigate whether fee data was lost at session boundary. |

These are additive to the existing S482 flags (`fee_gap`, `cost_basis_zero`, `simulated`, `partial_remainder`, `unmatched_open`, `orphan_exit`, `fee_asset_mismatch`, `outcome_unresolved`).

## Carryover Reliability

A cross-session round-trip is classified as `carryover_reliable` when:

1. Both legs have non-zero fees (`fee_reliable = true`).
2. Both legs have non-zero cost basis and classifiable outcome (`pnl_reliable = true`).
3. No `cross_session_fee_gap` flag is present.

This provides operators a single boolean to assess whether the cross-session P&L can be trusted for effectiveness analysis.

## Architecture

### Data Flow

```
Sessions (NATS KV)
    ↓ ListSessions + time filter
Chains per session (ClickHouse)
    ↓ CarryForwardEligibility filter
SessionLegs (sorted by timestamp)
    ↓ MatchFIFO
CrossSessionRoundTrips
    ↓ AnnotateRoundTrips (session provenance + continuity)
    ↓ ClassifyPair (effectiveness attribution)
    ↓ ReconcileCrossSessionRoundTrip (reconciliation flags)
ContinuityReviewItems
    ↓ Filters (continuity, cross_only, flagged, outcome)
ContinuityReviewReply
```

### Use Case: `GetContinuityReviewUseCase`

- **Location**: `internal/application/analyticalclient/get_continuity_review.go`
- **Dependencies**: `CrossSessionSessionReader` (KV), `CompositeReader` (ClickHouse)
- **Reuses**: `MatchFIFO`, `AnnotateRoundTrips`, `ClassifyPair`, `ReconcileCrossSessionRoundTrip`
- **No new infrastructure**: no new ClickHouse tables, NATS subjects, or streams

## Alignment with Existing Surfaces

| Surface | Stage | Relationship |
|---------|-------|-------------|
| Pairing read model | S481 | Continuity review extends pairing with cross-session context |
| Round-trip review | S482 | Base reconciliation is embedded; cross-session flags are added |
| Effectiveness | S476 | Attribution is computed identically; summary splits by cross/intra |
| Cross-session pairing | S495 | Underlying data flow is reused; review adds reconciliation layer |

## Limitations

- **L-S496-1**: The review surface is retrospective — it cannot show real-time carryover state during an active session.
- **L-S496-2**: Lookback window bounds from S495 apply — counterparts beyond `since`/`until` or `max_sessions` will appear unresolved.
- **L-S496-3**: The stub composite reader returns the same chains for all sessions in tests, which may produce duplicated legs. Production code queries per-session time bounds.
- **L-S496-4**: Carryover reliability assessment is based on fill data completeness, not venue confirmation. If a venue returned fills without fees (e.g., maker rebate scenarios), `carryover_reliable` may be false despite a valid round-trip.
- **L-S496-5**: This is not a dashboard or position tracker. It answers historical review questions only.

## Guard Rails

- No new ClickHouse tables.
- No write-path changes.
- No position engine or portfolio model.
- No runtime carry-forward — sessions remain isolated.
- No dashboard — minimal review surface only.
- Additive only — zero changes to existing pairing, reconciliation, or effectiveness types.
