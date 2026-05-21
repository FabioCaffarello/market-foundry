# Post-Session Operational Verification

> Authority: S447 | Date: 2026-03-24 | Wave: Live Trading Enablement Ceremony (S444-S448)

## Purpose

This document defines the complete verification protocol applied after the S446 supervised live session. It covers every dimension the session was expected to produce evidence for: persistence, read-path queryability, fee/commission fields, backup integrity, lifecycle consistency, and scope containment.

S447 does NOT repeat or extend the live session. It rigorously examines what the session left in the system.

## Verification Scope

| Dimension | What S447 Verifies | Method |
|-----------|-------------------|--------|
| Persistence (ClickHouse) | ExecutionIntent and venue response rows exist | SQL query on `executions` table |
| Persistence (NATS KV) | Venue fill state stored in KV bucket | Gateway query route |
| Fee/commission fields | Fills JSON contains Fee, FeeAsset, CostBasis | SQL query + JSON inspection |
| Read-path queryability | All query routes return consistent data | Gateway API calls |
| Backup (post-session) | ClickHouse snapshot captured after session | `clickhouse-scheduled-backup.sh` |
| Lifecycle consistency | ClickHouse and NATS KV agree on terminal status | Cross-store comparison |
| Scope containment | No orders outside BTCUSDT, no Futures activity | SQL scope audit |
| Kill-switch state | Gate halted after session | Gateway /execution/control |

## Verification Checks

### PO-1: Kill-Switch Halt

| Item | Detail |
|------|--------|
| Purpose | Confirm gate is halted after session |
| Method | `GET /execution/control` via gateway |
| Pass criteria | `gate.status = "halted"` |
| If not halted | Issue halt command immediately, log the gap |

### PO-2: Post-Session Backup

| Item | Detail |
|------|--------|
| Purpose | Capture ClickHouse state after session |
| Method | `clickhouse-scheduled-backup.sh` with `BACKUP_NAME=post_session_<id>` |
| Pass criteria | All MergeTree tables backed up, zero failures |
| Off-host | Replication via rsync if `BACKUP_OFFHOST_TARGET` configured |
| Evidence | Backup log in `backups/logs/` |

### PO-3: ClickHouse Intent Records

| Item | Detail |
|------|--------|
| Purpose | Verify ExecutionIntent was persisted |
| Query | `SELECT * FROM executions WHERE symbol = 'BTCUSDT' ORDER BY created_at DESC LIMIT 5` |
| Expected | At least 1 record with correct symbol, side, quantity, timestamp within session window |
| Fields checked | `symbol`, `side`, `quantity`, `status`, `timestamp`, `source` |

### PO-4: ClickHouse Venue Response Records

| Item | Detail |
|------|--------|
| Purpose | Verify venue response (fill or rejection) was persisted |
| Query | `SELECT * FROM venue_responses WHERE symbol = 'BTCUSDT' ORDER BY created_at DESC LIMIT 5` |
| Expected | At least 1 record with venue order ID, status, and fill details |
| Fields checked | `status` (filled/rejected), `filled_quantity`, `fills` JSON |

### PO-5: NATS KV State

| Item | Detail |
|------|--------|
| Purpose | Verify KV stores reflect final lifecycle state |
| Method | `GET /execution/control` and `GET /execution/venue-market-order/latest?symbol=BTCUSDT` |
| Expected | Gate = halted; venue order state matches ClickHouse |
| KV buckets checked | `EXECUTION_VENUE_MARKET_ORDER_LATEST`, `EXECUTION_VENUE_REJECTION_LATEST` |

### PO-6: System Status Summary

| Item | Detail |
|------|--------|
| Purpose | Capture final health counters |
| Method | `GET /statusz` on execute binary |
| Expected | Counters show processed >= 1, filled or rejected >= 1, errors = 0 |

### PO-7: Fee/Commission Verification (S447)

| Item | Detail |
|------|--------|
| Purpose | Verify fee fields are populated in persisted fill records |
| Query | `SELECT event_id, symbol, side, status, filled_quantity, fills FROM executions WHERE symbol = 'BTCUSDT' AND status IN ('filled','partially_filled')` |
| Expected (Spot) | `fills` JSON contains `Fee` (non-zero string), `FeeAsset` (e.g., "BNB" or "USDT"), `CostBasis` (notional value) |
| Expected (Spot) | `Simulated = false` in fills |
| Known limitation | Fee field is aggregated across fill legs; individual leg fees are not stored separately |

#### Fee Field Semantics for Spot (Reference)

| Field | Meaning | Source |
|-------|---------|--------|
| `Fee` | Total commission charged by Binance across all fill legs | `SUM(fills[].commission)` from Binance response |
| `FeeAsset` | Denomination of fee | `fills[0].commissionAsset` (uniform across legs) |
| `CostBasis` | Total notional value of the order | `cumulativeQuoteQty` from Binance response |
| `Simulated` | `false` for real venue fills | Adapter sets based on venue type |

### PO-8: Lifecycle Consistency (S447)

| Item | Detail |
|------|--------|
| Purpose | Verify ClickHouse and NATS KV agree on terminal state |
| Method | Compare `status` and `filled_quantity` from ClickHouse query vs NATS KV query |
| Pass criteria | Both report same `status` (filled or rejected) and `final = true` |
| Tolerance | Timestamp may differ slightly (async persistence); status must match |

### PO-9: Scope Containment (S447)

| Item | Detail |
|------|--------|
| Purpose | Verify no scope leakage occurred during session |
| Queries | Count all `venue_market_order` executions in 24h; count non-BTCUSDT executions |
| Pass criteria | Non-BTCUSDT count = 0; total count matches operator expectation (1 order) |
| Scope dimensions | Symbol (BTCUSDT only), segment (Spot only), exchange (Binance only) |

## Evidence Artifacts Produced

| Artifact | Location | Content |
|----------|----------|---------|
| Post-session backup | `backups/clickhouse/post_session_*` | ClickHouse snapshot |
| Session log (with PO checks) | `backups/logs/sessions/live_*.log` | All PO-1 through PO-9 output |
| ClickHouse intent query result | Session log | Raw JSON rows |
| ClickHouse fills query result | Session log | Raw JSON with fee fields |
| NATS KV state snapshot | Session log | Gateway query JSON |
| Scope audit result | Session log | Execution counts |

## Verification Protocol (Operator Steps)

1. Confirm kill-switch is halted (PO-1). If not, halt immediately.
2. Run post-session backup (PO-2). Verify off-host replication if configured.
3. Query ClickHouse for intent and response records (PO-3, PO-4).
4. Query NATS KV for lifecycle state (PO-5).
5. Capture system health (PO-6).
6. Verify fee/commission fields in fills (PO-7).
7. Cross-check ClickHouse vs NATS KV for lifecycle consistency (PO-8).
8. Audit scope containment -- no unauthorized symbols or segments (PO-9).
9. Review session log for any anomalies, warnings, or unexpected behavior.
10. Archive session log and backup artifacts.

All steps are automated in `scripts/smoke-supervised-live-session.sh post-session`.

## Governing Questions Answered (S447 Scope)

| ID | Question | Verification Check |
|----|----------|-------------------|
| GQ-14 | Was the kill-switch activated after session? | PO-1 |
| GQ-15 | Was a post-session backup completed? | PO-2 |
| GQ-16 | Is the order lifecycle persisted in ClickHouse? | PO-3, PO-4, PO-7 |
| GQ-17 | Is NATS KV state consistent with the final venue response? | PO-5, PO-8 |

## What This Verification Does NOT Cover

| Exclusion | Reason |
|-----------|--------|
| Re-execution or session extension | S447 is verification only, not session continuation |
| PnL calculation or portfolio impact | Out of ceremony scope (NG-16) |
| Fee optimization analysis | Out of scope (NG-17) |
| Futures segment verification | Spot-only ceremony |
| Multi-symbol analysis | Single-symbol scope |
| Push alerting verification | Accepted limitation (NG-11) |

## Limitations

| # | Limitation | Impact |
|---|-----------|--------|
| 1 | Fee field is a JSON string in ClickHouse, not a numeric column | Requires JSON parsing for fee analysis; not directly aggregatable in SQL |
| 2 | NATS KV stores only latest state per key | Historical lifecycle transitions not queryable from KV; rely on ClickHouse for history |
| 3 | Lifecycle consistency check is eventually-consistent | Writer may lag behind KV; small window of inconsistency is expected |
| 4 | Scope audit uses 24h window | If session spans midnight UTC, adjust window accordingly |
| 5 | PO-7 fee check is pattern-based | Checks for presence of "Fee" string in JSON; does not validate numeric correctness |

## References

- [Supervised Live Session Proof](supervised-live-session-proof.md) (S446)
- [Audit Trail and Operational Findings](live-session-observed-behavior-audit-trail-and-operational-findings.md) (S446)
- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints and Stop Conditions](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [Fee Normalization Model](fee-normalization-model-and-cross-segment-consistency.md) (S428)
- [Fee/Commission Cross-Segment Semantics](fees-commission-assets-cross-segment-semantics-and-limitations.md) (S428)
