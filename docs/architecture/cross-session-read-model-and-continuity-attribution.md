# Cross-Session Read Model and Continuity Attribution

> S495 — Cross-Session Position Continuity Wave (S493–S497)

## Purpose

This document defines the read model that makes cross-session continuity queryable via HTTP. It builds on the S494 canonical continuity model (domain types, classification rules, pure functions) and adds the application-layer orchestration, data discovery, and HTTP surface needed to answer:

- **What legs carried forward across session boundaries?**
- **Which ones resolved, which remain open, and which are genuinely unresolved?**
- **What is the effectiveness attribution for cross-session round-trips?**

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌──────────────────┐
│  NATS KV     │────▶│  Session     │────▶│                  │
│  (sessions)  │     │  Discovery   │     │  Cross-Session   │
└─────────────┘     └──────────────┘     │  Pairing         │
                                          │  Use Case        │
┌─────────────┐     ┌──────────────┐     │                  │
│  ClickHouse  │────▶│  Chain       │────▶│  (orchestration) │
│  (chains)    │     │  Fetcher     │     │                  │
└─────────────┘     └──────────────┘     └────────┬─────────┘
                                                   │
                                          ┌────────▼─────────┐
                                          │  MatchFIFO       │
                                          │  (session-agnostic│
                                          │   FIFO matching)  │
                                          └────────┬─────────┘
                                                   │
                                          ┌────────▼─────────┐
                                          │  AnnotateRound-  │
                                          │  Trips + Summary │
                                          │  + Effectiveness │
                                          └────────┬─────────┘
                                                   │
                                          ┌────────▼─────────┐
                                          │  HTTP Response    │
                                          │  (cross-session   │
                                          │   pairing reply)  │
                                          └──────────────────┘
```

## Data Sources

### Session Metadata (NATS KV)

- **Bucket**: `EXECUTION_SESSION`
- **Query**: `ListSessions()` → all sessions, filtered by time window
- **Fields used**: `SessionID`, `StartedAt`, `ClosedAt`, `Status`
- **Purpose**: Determine which sessions to scan and provide provenance metadata

### Execution Chains (ClickHouse)

- **Tables**: Five domain tables queried via `CompositeReader.QueryChainsBatch`
- **Query**: Per session, bounded by `session.StartedAt` to `session.ClosedAt`
- **Fields used**: `ExecutionIntent` (status, fills, side, symbol, source, timeframe)
- **Purpose**: Extract legs for FIFO matching

## Discovery Flow (Steps 1–7)

| Step | Action | Input | Output |
|------|--------|-------|--------|
| 1 | Fetch sessions from KV | `CrossSessionPairingQuery` | `[]Session` (filtered, sorted) |
| 2 | Query chains per session from ClickHouse | Session time bounds | `[]CompositeExecutionChain` per session |
| 3 | Apply `ClassifyCarryForward` per intent | `ExecutionIntent` | Eligible legs only |
| 4 | Build `CrossSessionLegSet` (sorted by timestamp) | All eligible `SessionLeg`s | Ordered multi-session leg collection |
| 5 | Run `MatchFIFO` on extracted legs | Plain `[]Leg` | `[]RoundTrip` |
| 6 | Annotate with `AnnotateRoundTrips` | Round-trips + session index | `[]CrossSessionRoundTrip` |
| 7 | Compute effectiveness + summaries | Annotated round-trips + chain map | `CrossSessionPairingReply` |

## HTTP Surface

### Endpoint

```
GET /analytical/composite/pairing/cross-session
```

### Query Parameters

| Parameter | Required | Type | Description |
|-----------|----------|------|-------------|
| `source` | Yes | string | Venue/segment (e.g., `binance_spot`) |
| `symbol` | Yes | string | Instrument (e.g., `BTCUSDT`) |
| `timeframe` | Yes | int | Candle interval |
| `since` | Yes | int64 | Unix seconds, inclusive |
| `until` | No | int64 | Unix seconds (default: now) |
| `max_sessions` | No | int | Session cap (default 30, max 50) |
| `continuity` | No | string | Filter: `resolved`, `open`, `genuine_unresolved`, `artificial_unresolved` |
| `cross_only` | No | bool | When `true`, only cross-session round-trips |

### Response Structure

```json
{
  "round_trips": [
    {
      "entry": { "...leg..." },
      "exit": { "...leg..." },
      "state": "paired",
      "entry_session_id": "session_20260320_100000",
      "exit_session_id": "session_20260321_100000",
      "cross_session": true,
      "continuity": "resolved",
      "attribution": { "outcome": "win", "net_pnl": 95.0, "..." }
    }
  ],
  "summary": {
    "paired_count": 5,
    "unmatched_entries": 2,
    "cross_session_pairs": 3,
    "intra_session_pairs": 2,
    "carry_forward_resolution_rate": 0.75,
    "sessions_scanned": 8
  },
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
  "source": "clickhouse+kv",
  "meta": {
    "total_ms": 342,
    "sessions_fetched": 8,
    "chains_scanned": 156,
    "legs_produced": 42,
    "legs_carried": 42,
    "legs_excluded": 114,
    "round_trips": 7
  }
}
```

## Integration with Pairing/Effectiveness

### Pairing Integration

The cross-session read model reuses the existing `MatchFIFO` algorithm unchanged. Cross-session pairing is achieved by widening the input (legs from multiple sessions) rather than modifying the matching rules. All FIFO invariants (M1–M7) hold across session boundaries.

### Effectiveness Integration

Paired cross-session round-trips receive full `ClassifyPair` attribution (win/loss/breakeven/unresolved). The attribution chain enrichment (`enrichFromChain`) provides decision type, strategy type, and severity context.

### Continuity Summary

The `ContinuitySummary` type provides aggregate metrics that allow operators to assess:

1. **ResolutionRate** — fraction of total round-trips that resolved (paired)
2. **CrossSessionResolutionRate** — fraction of boundary artifacts that cross-session matching resolved
3. **State counts** — breakdown by resolved, open, genuine unresolved, artificial unresolved

## Invariants

| ID | Invariant | Enforcement |
|----|-----------|-------------|
| INV-S495-1 | Session legs are sorted by timestamp before matching | `sort.Slice` in use case |
| INV-S495-2 | Only `CarryEligible` intents produce legs | `ClassifyCarryForward` filter |
| INV-S495-3 | Cross-session flag is set iff entry and exit session IDs differ | `AnnotateRoundTrips` logic |
| INV-S495-4 | Continuity classification is deterministic per round-trip | `ClassifyContinuity` pure function |
| INV-S495-5 | No write-path changes | Architecture constraint |
| INV-S495-6 | No new ClickHouse tables | Uses existing execution data |

## Non-Goals

- Runtime carry-forward (sessions remain isolated at runtime)
- Position engine or portfolio tracking
- Cross-symbol or cross-segment pairing
- Dashboards or BI surfaces
- Generic matching engine beyond FIFO
