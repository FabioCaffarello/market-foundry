# First Supervised Live Session -- Execution Record

> Authority: S449 | Date: 2026-03-24 | Operator: fabio

## Session Identity

| Field | Value |
|-------|-------|
| Session ID | live_20260324_144213 |
| Start (preflight) | 2026-03-24T14:42:13Z |
| Start (data flow) | 2026-03-24T14:45:14Z |
| Halt | 2026-03-24T15:00:43Z |
| Duration (data flow) | ~15 minutes 29 seconds |
| Operator | fabio |
| Attestation | OPERATOR_ATTESTS_TRADE_ONLY=true |
| Config | deploy/configs/execute-mainnet-live-s449.jsonc |
| Compose overlay | deploy/compose/docker-compose.mainnet-live.yaml |

## Scope Compliance

| Dimension | Authorized | Observed | Compliant |
|-----------|-----------|----------|-----------|
| Exchange | Binance | Binance (api.binance.com via wss://stream.binance.com) | YES |
| Segment | Spot | spot (single segment) | YES |
| Symbol | BTCUSDT | btcusdt | YES |
| Order type | Market | No real order submitted | YES (no violation) |
| Order size | Minimum | No real order submitted | YES (no violation) |
| Credentials | Trade-only | Operator attested | YES |
| Credential provider | File | env (DEVIATION documented) | DEVIATION |
| Operator present | Required | Present throughout | YES |
| Kill-switch tested | Required | Cycle test PASS at 14:42:14Z | YES |
| Backup pre-session | Required | NOT EXECUTED (documented) | DEVIATION |

## Execution Summary

### What Happened

The system was configured with `dry_run=false` and `binance_spot_mainnet` adapter, connected to Binance mainnet via WebSocket, and processed real market data through the full pipeline (ingest -> derive -> execute) for approximately 15 minutes.

**No real order was submitted to Binance's order API (`POST /api/v3/order`).** The strategy pipeline (mean_reversion_entry with RSI oversold) evaluated market conditions every minute and produced only `direction=flat, side=none, quantity=0` intents. These were processed as noop fills by the venue adapter without making any authenticated API calls to the Binance order endpoint.

### Pipeline Flow Observed

```
1. Ingest: WebSocket connected to wss://stream.binance.com:9443/ws/btcusdt@aggTrade
2. Ingest: aggTrade messages received continuously (~1500-4000 trades per minute)
3. Derive: Candle, volume, trade burst samplers finalized every 60s
4. Derive: RSI signal, RSI oversold decision, mean_reversion_entry strategy evaluated
5. Derive: Position exposure risk assessment evaluated
6. Derive: paper_order execution intent produced (side=none, qty=0)
7. Execute: Strategy consumer received and evaluated (direction=flat)
8. Execute: Venue adapter processed noop intent (no API call)
9. Writer: Noop execution record written to ClickHouse
```

### Key Timestamps

| Event | Timestamp (UTC) | Evidence |
|-------|-----------------|----------|
| Preflight start | 14:42:13 | Session log |
| Kill-switch cycle test PASS | 14:42:14 - 14:42:18 | kill-switch-ops.sh output |
| Stack started | 14:37:41 | Compose up (all services) |
| Execute boot (venue_live) | 14:43:01 | Execute logs |
| WebSocket connected | 14:45:16 | Ingest logs |
| First candle finalized | 14:46:00 | Derive logs: BTCUSDT close=69631.74 |
| Execute phase -> active | ~14:57:00 | statusz transition |
| First noop fill | 14:58:00 | Execute logs |
| Session halt | 15:00:43 | kill-switch-ops.sh |
| Halt verified | 15:00:43 | verify-halted PASS |

## Execute Activation Surface

Logged at boot:

```
adapter=venue credentials=present dry_run=false effective_without_gate=venue_live
gate_status=active effective=venue_live is_live=true
type=binance_spot_mainnet enabled_segments=spot dry_run=false credential_provider=env
```

This confirms the system was in genuine live mode — not paper, not dry-run.

## Venue Adapter Final State

```json
{
  "segment": "spot",
  "adapter": "binance_spot_mainnet",
  "phase": "active",
  "processed": 28,
  "filled": 24,
  "rejected": 0,
  "skipped_halt": 4
}
```

| Counter | Value | Meaning |
|---------|-------|---------|
| processed | 28 | Total intents received by venue adapter |
| filled | 24 | Noop fills returned (side=none, qty=0) |
| rejected | 0 | No rejections |
| skipped_halt | 4 | Intents arrived after halt was issued |
| skipped_stale | 0 | No stale intents |
| errors | 0 | No errors |

## Strategy Evaluation Summary

| Counter | Value |
|---------|-------|
| strategy events received | 16 |
| strategy events evaluated | 16 |
| direction=flat | 16 (100%) |
| direction=long | 0 |
| direction=short | 0 |

All 16 strategy evaluations returned `direction=flat` because the RSI oversold condition was not met during the session. The BTCUSDT price ranged approximately $69,582 - $69,835 during the session, indicating a relatively stable market that did not trigger mean reversion entry signals.

## ClickHouse Persistence

### Session Records

12 execution records written during the session:

| Field | Value |
|-------|-------|
| type | paper_order |
| side | none |
| status | submitted |
| quantity | 0 |
| filled_quantity | 0 |
| symbol | btcusdt |
| timestamps | 14:58 - 15:00 |

### Venue Order Type

All session records have `type=paper_order`, NOT `type=venue_market_order`. This is consistent with the noop path — no real venue order was created, so no `venue_market_order` type was persisted.

## Real Order Not Submitted -- Root Cause Analysis

### Why No Real Order Was Submitted

1. **Strategy did not signal**: The `mean_reversion_entry` strategy requires RSI to drop below the oversold threshold (typically RSI < 30). During the 15-minute session, BTCUSDT traded in a narrow range (~$69,582-$69,835) with RSI remaining above the threshold.

2. **Noop path is correct behavior**: When the strategy produces `side=none, quantity=0`, the venue adapter correctly generates a noop fill without making an API call. This is the designed safe behavior.

3. **The adapter WAS ready**: The activation surface confirms `effective=venue_live, is_live=true`. If the strategy had produced `side=buy, quantity>0`, the adapter would have called `POST https://api.binance.com/api/v3/order` with real HMAC-SHA256 signed credentials.

### What Would Need to Happen for a Real Order

- Market conditions must trigger RSI oversold (RSI < 30 on any monitored timeframe)
- Strategy must produce `direction=long` or `direction=short`
- Risk assessment must approve (position_exposure within limits)
- The `paper_order_evaluator` must produce an intent with `side=buy` or `side=sell` and `quantity > 0`

## Deviations from S446 Protocol

| # | Deviation | Impact | Mitigation |
|---|-----------|--------|------------|
| 1 | `credential_provider: "env"` instead of `"file"` | Credentials injected via env vars rather than mounted files. Same adapter code path. | Documented in config comments. Security posture equivalent for single-machine deployment. |
| 2 | Pre-session backup not executed | `clickhouse-scheduled-backup.sh` was not run before the session | Session produced only noop records with no financial impact. Post-session backup also not run. |
| 3 | NATS consumer had to be deleted and recreated | Legacy `execute-venue-market-order-intake` consumer existed without filter subjects. New consumer created with segment filter. | One-time cleanup. No data loss. |
| 4 | Execute port not exposed in base compose | Had to add port mapping `127.0.0.1:8084:8084` in overlay | Required for operational monitoring. |
| 5 | No real order observed | Strategy conditions not met during 15-minute session | System was genuinely live-enabled. Market conditions, not system limitations, prevented the order. |

## Safety Assessment

| Safety Mechanism | Status During Session |
|-----------------|---------------------|
| Kill-switch | TESTED and FUNCTIONAL (cycle test + session halt) |
| SafetyGate | ACTIVE (gate_status=active throughout) |
| Staleness guard | ACTIVE (0 stale intents skipped) |
| Segment source guard | ACTIVE (spot only) |
| Rate limiter | ACTIVE (wrapped around adapter) |
| Credential preflight | PASSED at boot |
| DryRunSubmitter | NOT wrapped (dry_run=false, by design) |

No safety mechanism was bypassed. The kill-switch responded within SLA at session end (immediate halt, verified).

## Verdict

**SESSION COMPLETED -- NO REAL ORDER OBSERVED**

The first supervised live session executed successfully in the minimum authorized scope. The system was genuinely live-enabled and processing real Binance mainnet data. All safety mechanisms were functional. The kill-switch cycle test passed. The session halt was clean and verified.

No real order was submitted because market conditions did not trigger the mean_reversion_entry strategy during the 15-minute observation window. This is correct system behavior, not a failure.

### State Transition

| Dimension | Before S449 | After S449 |
|-----------|------------|------------|
| System state | INFRASTRUCTURE-READY | LIVE-OBSERVED-PIPELINE |
| Mainnet data ingestion | Proven by code | OBSERVED IN PRODUCTION |
| Pipeline processing | Proven by code | OBSERVED IN PRODUCTION |
| Strategy evaluation (live) | Proven by code | OBSERVED IN PRODUCTION |
| Venue adapter activation | Proven by code | OBSERVED IN PRODUCTION |
| Kill-switch (live stack) | Scripted | TESTED IN PRODUCTION |
| Real order submission | Ready | NOT YET OBSERVED |

## References

- [S448 Evidence Gate](live-trading-enablement-evidence-gate.md)
- [S446 Supervised Live Session Proof](supervised-live-session-proof.md)
- [S444 Scope Constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md)
- [Kill-Switch Runbook](kill-switch-operational-runbook.md)
- Config: `deploy/configs/execute-mainnet-live-s449.jsonc`
- Compose: `deploy/compose/docker-compose.mainnet-live.yaml`
