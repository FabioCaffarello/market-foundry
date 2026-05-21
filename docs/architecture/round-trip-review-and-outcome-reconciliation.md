# Round-Trip Review and Outcome Reconciliation

> S482 — round-trip review surface and outcome reconciliation between pairing, fills, fees, and effectiveness.

## Purpose

This surface lets an operator review individual round-trips with confidence by answering:

1. **What closed?** — paired round-trips with entry/exit legs and realized P&L.
2. **What didn't close?** — unmatched entries (open positions) and orphan exits (data gaps).
3. **Is the data reliable?** — reconciliation flags that identify fee gaps, zero cost basis, simulated fills, fee-asset mismatches, and unresolved outcomes on structurally paired trades.
4. **How does this affect evaluation?** — direct link between reconciliation flags and effectiveness attribution reliability.

## HTTP Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/analytical/composite/pairing/review` | Batch round-trip review with reconciliation |
| GET | `/analytical/composite/pairing/review/chain` | Single-chain round-trip review |

### Query Parameters

**Batch** (same partition key as pairing):
- `source` (required), `symbol` (required), `timeframe` (required)
- `since`, `until` (unix seconds), `limit` (default 50, max 200)

**Single**:
- `correlation_id` (required), `symbol` (required)

**Filters** (batch and single):
- `state`: `paired`, `unmatched_entry`, `unmatched_exit`
- `side`: `buy`, `sell`
- `outcome`: `win`, `loss`, `breakeven`, `unresolved` — filters on effectiveness outcome
- `flagged`: `true` — only return round-trips with reconciliation flags

### Response Shape

```json
{
  "reviews": [
    {
      "entry": { "direction": "entry", "side": "buy", "price": "50000", ... },
      "exit": { "direction": "exit", "side": "sell", "price": "51000", ... },
      "state": "paired",
      "matched_quantity": "0.10000000",
      "attribution": {
        "outcome": "win",
        "gross_pnl": 100.0,
        "net_pnl": 99.0,
        "total_fees": 1.0,
        ...
      },
      "reconciliation": {
        "flags": [],
        "clean": true,
        "fee_reliable": true,
        "pnl_reliable": true
      }
    }
  ],
  "summary": {
    "paired_count": 5,
    "unmatched_entries": 2,
    "resolved_rate": 0.714,
    "win_count": 3,
    "loss_count": 2,
    "total_pnl": 150.0,
    "clean_count": 4,
    "flagged_count": 3,
    "flag_counts": { "fee_gap": 1, "unmatched_open": 2 },
    "fee_reliable_count": 4,
    "pnl_reliable_count": 3
  },
  "source": "clickhouse",
  "meta": { "total_ms": 45, "chains_scanned": 12, "round_trips": 7, "reviewed": 7 }
}
```

## Reconciliation Flags

| Flag | When Raised | Impact |
|------|-------------|--------|
| `fee_gap` | One or both legs have zero fees (common on futures segment) | Fee data unreliable; net P&L may overstate actual return |
| `cost_basis_zero` | One or both legs have zero cost basis (paper/dry-run) | P&L not classifiable; outcome forced to unresolved |
| `simulated` | At least one leg is from paper/dry-run execution | Real-money attribution not applicable |
| `unmatched_open` | Entry without exit — position is still open | No realized P&L; outcome is unresolved |
| `orphan_exit` | Exit without entry — data gap or orphan | Cannot attribute this exit to any known entry |
| `fee_asset_mismatch` | Entry and exit legs have different fee denomination | Fee comparison across legs is not apples-to-apples |
| `outcome_unresolved` | Paired round-trip but effectiveness outcome is unresolved | Usually indicates both legs have zero cost basis |

## Reliability Signals

- **`clean`**: No flags present — the round-trip has reliable data for all dimensions.
- **`fee_reliable`**: Paired, and both legs have non-zero fees.
- **`pnl_reliable`**: Paired, non-zero cost basis on both legs, and outcome is win/loss/breakeven.

## Relationship to Existing Surfaces

| Surface | What It Answers | How Review Extends It |
|---------|----------------|----------------------|
| `/analytical/composite/pairing` (S481) | Raw pairing with FIFO matching | Review adds reconciliation flags and reliability signals |
| `/analytical/composite/decision/effectiveness` (S476) | Per-chain effectiveness attribution | Review links attribution to paired round-trips with data quality context |
| `/analytical/composite/decision/effectiveness/summary` (S477) | Cohort-level win rate and P&L | Review summary adds flag_counts and reliability counts for cohort data quality |

## Architecture

- **No new ClickHouse tables.** Reconciliation is computed at read time from existing fill data.
- **No write-path changes.** Additive read-path computation only.
- **Reuses existing infrastructure:** CompositeReader, MatchFIFO, ClassifyPair, IntentToLeg.
- **Domain layer:** `internal/domain/pairing/reconciliation.go` — pure functions, no I/O.
- **Application layer:** `internal/application/analyticalclient/get_roundtrip_review.go`.

## Limitations

1. **Reconciliation is structural, not causal.** Flags detect data-quality issues but do not diagnose root causes (e.g., why fees are zero on futures).
2. **No cross-session reconciliation.** Round-trips are scoped to the query window; a position opened in one session and closed in another requires both to be within the time range.
3. **Fee-asset mismatch does not adjust P&L.** When entry fees are in BNB and exit fees are in USDT, the system sums them numerically without currency conversion.
4. **Futures fee gap is a known platform limitation** (S428). The flag surfaces the issue but cannot resolve it without upstream venue changes.
5. **No position-level aggregation.** Each round-trip is reviewed independently; there is no net-position or portfolio view.
