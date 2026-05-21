# First Live Session -- Preflight, Observed Behavior, and Stop Condition Log

> Authority: S449 | Date: 2026-03-24 | Operator: fabio

## Purpose

This document records every preflight check result, observed runtime behavior, and stop condition evaluation from the first supervised live session. It is the honest operational log of the S449 ceremony.

## Preflight Checklist Results

### PS-1: Kill-Switch Cycle Test

| Step | Time (UTC) | Result |
|------|-----------|--------|
| HALT | 14:42:14 | PASS -- gate set to halted |
| VERIFY HALTED | 14:42:16 | PASS -- gate confirmed halted |
| RESUME | 14:42:16 | PASS -- gate set to active |
| VERIFY ACTIVE | 14:42:18 | PASS -- gate confirmed active |

**Result: PASS.** Full 4-step cycle completed in ~4 seconds.

**Observation**: `Cannot reach execute at http://127.0.0.1:8084/statusz -- skipping counter verification` logged during PS-1 because the execute service port was not yet mapped to the host at that point. This was a compose overlay gap corrected before session start.

### PS-2: Pre-Session Backup

**Result: NOT EXECUTED.**

The `clickhouse-scheduled-backup.sh` was not run as part of the preflight. This is a deviation from the S446 protocol. Mitigation: the session produced no records with financial impact (all noop intents). The ClickHouse data from before the session remains intact.

### PS-3: Credential File Mount

**Result: N/A -- DEVIATED to env provider.**

Credentials were provided via environment variables (`MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY`, `MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET`) injected through `deploy/envs/local.env` and the compose `env_file` directive.

The file-based credential provider path (`/run/secrets/market-foundry`) was not used. The config was changed to `credential_provider: "env"` in the S449-specific config file.

Execute boot log confirms: `credential provider set to env`.

### PS-4: Config Audit

**Result: PASS.**

Config: `deploy/configs/execute-mainnet-live-s449.jsonc`

| Field | Expected | Actual | Match |
|-------|---------|--------|-------|
| dry_run | false | false | YES |
| credential_provider | file | env (DEVIATION) | DEVIATION |
| spot.enabled | true | true | YES |
| spot.adapter | binance_spot_mainnet | binance_spot_mainnet | YES |
| futures.enabled | false/absent | absent | YES |
| staleness_max_age | 120s | 120s | YES |
| submit_timeout | 10s | 10s | YES |

Confirmed by execute boot log: `type=binance_spot_mainnet enabled_segments=spot dry_run=false`.

### PS-5: Operator Attestation

**Result: PASS.**

Operator `fabio` confirmed:
- API key has TRADE permission
- API key does NOT have WITHDRAWAL permission
- Operator is present and ready to monitor

### PS-6: Kill-Switch Initial State

**Result: PASS.**

```
Gate Status:  active
Reason:       cycle-resume-after-s449-pre-session
Updated By:   fabio
```

### PS-7: System Boot Verification

**Result: PASS.**

| Endpoint | Response |
|----------|----------|
| Gateway /readyz | `{"status":"ready"}` |
| Execute /readyz | `{"status":"ready"}` |
| Execute /statusz | phase=starting, segment=spot, adapter=binance_spot_mainnet |

## Infrastructure Issues Encountered During Preflight

### Issue 1: Mainnet Credentials Not Present

**Discovery**: The `deploy/envs/local.env` contained only testnet credential keys (`MF_BINANCE_SPOT_TESTNET_API_KEY`, `MF_BINANCE_SPOT_TESTNET_SECRET`). No mainnet-formatted keys existed.

**Resolution**: Mainnet env var entries (`MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY`, `MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET`) were added with values sourced from the existing (mainnet-created) credentials stored under testnet names.

**Operator clarification**: The credentials were confirmed as created on `binance.com` (mainnet), not `testnet.binance.vision`. The testnet key names were a naming convention, not a credential source indicator.

### Issue 2: Execute Crash Loop -- Credential Preflight Failure

**Discovery**: After initial stack start, the execute service entered a restart loop with:
```
preflight check "mainnet-credentials" failed: mainnet credential missing:
segment=spot adapter=binance_spot_mainnet key=API_KEY
```

**Root cause**: The compose overlay used `environment: ${VAR:-}` syntax which requires the variable to exist in the host shell environment. The credentials were in `local.env` (loaded by ClickHouse's `env_file` directive) but not exposed to the execute service.

**Resolution**: Changed the compose overlay from explicit `environment:` entries to `env_file: ../envs/local.env`.

### Issue 3: NATS Consumer Conflict

**Discovery**: Execute started successfully but the supervisor failed:
```
start venue consumer: create durable consumer: nats: multiple consumer filter subjects not supported by nats-server
```

**Root cause**: A durable consumer `execute-venue-market-order-intake` existed in NATS JetStream from a previous run with no filter subjects. The S401 segment-scoped consumer code attempted to recreate it with `FilterSubjects: ["execution.events.paper_order.submitted.binances.>"]`, which NATS rejected as a configuration change to an existing consumer.

**Resolution**: Deleted the stale consumer via NATS CLI:
```bash
nats -s nats://127.0.0.1:4222 consumer rm EXECUTION_EVENTS execute-venue-market-order-intake --force
```
The consumer was automatically recreated with the correct segment filter on next execute restart.

### Issue 4: Execute Port Not Exposed

**Discovery**: After execute started successfully, `curl http://127.0.0.1:8084/readyz` returned UNREACHABLE. The base compose file does not map port 8084 to the host.

**Resolution**: Added `ports: ["127.0.0.1:8084:8084"]` to the mainnet-live compose overlay.

### Issue 5: Binding Not Seeded

**Discovery**: Ingest showed `total_bindings=0, activated=0`. No exchange binding for BTCUSDT existed in configctl.

**Resolution**: Ran the seed script:
```bash
SOURCE=binances SYMBOLS=btcusdt ./scripts/seed-configctl.sh
```
This created and activated the `binances.btcusdt` binding, which ingest discovered via the binding-watcher event subscription.

## Observed Runtime Behavior

### Data Ingestion

| Metric | Observed |
|--------|----------|
| WebSocket endpoint | wss://stream.binance.com:9443/ws/btcusdt@aggTrade |
| Connection time | ~1.3 seconds (14:45:14 to 14:45:16) |
| Trades per minute | 1,500 - 4,007 |
| BTCUSDT price range | $69,582 - $69,835 |
| WebSocket disconnections | 0 |
| Data gaps | None observed |

### Pipeline Processing

| Stage | Frequency | Observation |
|-------|-----------|------------|
| Candle finalization | Every 60s | Consistent, no missed candles |
| Volume finalization | Every 60s | Consistent, VWAP computed correctly |
| Trade burst detection | Every 60s | All returned burst=false |
| RSI signal | Not logged explicitly | RSI values not above oversold threshold |
| RSI oversold decision | Implicit in strategy | No oversold trigger |
| Strategy resolution | Every 60s per timeframe | direction=flat for all 16 evaluations |
| Execution intent | Every 60s per timeframe | side=none, quantity=0 |

### Execute Processing

| Metric | Value |
|--------|-------|
| Total intents processed | 28 |
| Noop fills | 24 |
| Rejections | 0 |
| Skipped (halt) | 4 |
| Skipped (stale) | 0 |
| Errors | 0 |
| Real API calls to Binance order endpoint | 0 |

### Venue Order ID Pattern

All fills used the pattern `binance-spot-noop-{timestamp_nanos}`, confirming they went through the noop path in the adapter (no HTTP call to the exchange).

## Stop Condition Evaluation

| SC | Condition | Triggered | Evidence |
|----|-----------|-----------|----------|
| SC-1 | API error rate > 10% | NO | 0 errors across 28 intents |
| SC-2 | Latency > 5x baseline | NO | WebSocket connected in ~1.3s, no latency anomalies |
| SC-3 | Unexpected order state | NO | All states were noop/flat — expected |
| SC-4 | Fill qty > requested qty | NO | All qty=0, filled_qty=0 |
| SC-5 | Kill-switch fails SLA | NO | Cycle test PASS; halt at session end PASS |
| SC-6 | Credential error | NO | Preflight PASS, no credential errors in logs |
| SC-7 | ClickHouse write failure | NO | 12 records written successfully |
| SC-8 | NATS connectivity loss | NO | All consumers ran without disconnection |
| SC-9 | Operator uncertainty | NO | Operator confirmed throughout |
| SC-10 | Wrong symbol | NO | All records symbol=btcusdt |
| SC-11 | Wrong segment | NO | Only spot segment enabled and observed |
| SC-12 | More than 1 real order | NO | 0 real orders submitted |
| SC-13 | Unexpected config | NO | Config audit PASS |
| SC-14 | Pre-session check fails | PARTIAL | PS-2 (backup) not executed; PS-3 deviated to env |

**No stop condition was triggered during the session.**

SC-14 is marked PARTIAL because PS-2 (backup) was not executed and PS-3 was deviated. These are documented deviations, not runtime failures. They did not impact session safety because no real financial operations occurred.

## Honest Assessment

### What Was Proven

1. The system CAN run in `venue_live` mode on Binance mainnet
2. Real market data flows through the full pipeline
3. Strategy evaluation works on live data
4. The kill-switch functions correctly in a live stack
5. The noop path correctly prevents API calls for flat signals
6. ClickHouse persistence works for session records
7. The operational tooling (kill-switch-ops.sh, seed-configctl.sh) works against a live stack

### What Was NOT Proven

1. A real order can be submitted via `POST /api/v3/order`
2. A real fill response can be parsed from the venue
3. The HMAC-SHA256 signing works against mainnet (credentials may or may not be valid)
4. Fee/commission fields from a real response
5. End-to-end lifecycle from real submission to persistence

### What Remains Unknown

1. Whether the mainnet API credentials are actually valid for trading (they authenticate successfully for WebSocket data, but order submission was not tested)
2. The exact minimum order quantity at session time (LOT_SIZE filter)
3. How long a session must run before a real signal is generated (depends entirely on market conditions)

## References

- [Execution Record](first-supervised-live-session-execution-record.md) (S449)
- [S446 Supervised Live Session Proof](supervised-live-session-proof.md)
- [S444 Scope Constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md)
- [Kill-Switch Runbook](kill-switch-operational-runbook.md)
