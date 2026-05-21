# Supervised Live Session Proof

> Authority: S446 | Date: 2026-03-24 | Wave: Live Trading Enablement Ceremony (S444-S448)

## Purpose

This document proves that the market-foundry system is operationally ready for a supervised live trading session under the minimum authorized scope. It specifies the exact session protocol, the pre-session verification evidence, the runtime control path, and the post-session verification steps.

This is Block 2 of the enablement ceremony defined in the [S444 charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md).

## Session Scope (Frozen)

| Dimension | Value | Enforcement |
|-----------|-------|-------------|
| Exchange | Binance | Config: `binance_spot_mainnet` adapter |
| Segment | Spot | Config: only `spot` segment enabled |
| Symbol | BTCUSDT | Pipeline config: single symbol |
| Order type | Market | Domain model: `MARKET` type only |
| Order size | Minimum exchange quantity | Config: quantity field at exchange floor |
| Order count | Exactly 1 | Operator discipline + kill-switch after first fill |
| Credentials | Trade-only API key (no withdrawal) | Operator attestation (PS-5) |
| Credential provider | File-based | Config: `credential_provider: "file"` |
| Operator | Present throughout | Ceremony protocol |
| Kill-switch | Active and tested | Pre-session check PS-1 |

This scope is inherited from the [S443 authorization verdict](live-trading-authorization-evidence-gate.md) and frozen by the [S444 charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md). No expansion is permitted.

## Authorization Chain

```
S437 (Mainnet Authorization Evidence Gate)
  -> S438 (Live Trading Authorization Wave Charter)
    -> S439-S442 (Conditions C-1 through C-5 closed)
      -> S443 (Evidence Gate: AUTHORIZED -- CONDITIONAL)
        -> S444 (Enablement Ceremony Charter)
          -> S445 (C-6 Executed -- dry_run=false now valid)
            -> S446 (This document -- Supervised Live Session)
```

## Pre-Session Checklist Evidence

The pre-session checklist is implemented in `scripts/smoke-supervised-live-session.sh pre-session`. All 7 checks must pass before the session begins.

### PS-1: Kill-Switch Cycle Test

| Item | Detail |
|------|--------|
| Command | `./scripts/kill-switch-ops.sh cycle` |
| What it tests | Full halt -> verify-halted -> resume -> verify-active cycle |
| SLA | Gate transitions must complete within 2 seconds |
| Pass criteria | All 4 steps succeed, gate returns to `active` |
| Evidence | Script output logged to session log file |

The kill-switch is the primary safety mechanism. If it fails to respond, the session MUST NOT proceed (SC-5).

### PS-2: Automated Backup (Pre-Session)

| Item | Detail |
|------|--------|
| Command | `./scripts/clickhouse-scheduled-backup.sh` |
| What it tests | ClickHouse backup completes for all MergeTree tables |
| Off-host | Replication to configured target if `BACKUP_OFFHOST_TARGET` is set |
| Pass criteria | All tables backed up, zero failures |
| Evidence | Backup log in `backups/logs/` |

A pre-session backup ensures that the system state before the live session is preserved and recoverable.

### PS-3: Credential File Mount

| Item | Detail |
|------|--------|
| Path | `<CREDENTIAL_PATH>/binance_spot_mainnet/API_KEY` and `API_SECRET` |
| What it tests | Files exist, are non-empty, and are readable |
| Pass criteria | Both files present and non-empty |
| Evidence | File existence and size logged |

File-based credential provider (S439) is required for mainnet. Environment variables are NOT authorized for mainnet credentials.

### PS-4: Config Audit

| Item | Detail |
|------|--------|
| File | `deploy/configs/execute-mainnet-live.jsonc` |
| What it tests | Config matches minimum authorized scope exactly |
| Verified fields | `dry_run=false`, `credential_provider=file`, `spot.enabled=true`, `spot.adapter=binance_spot_mainnet`, `futures.enabled` absent or false |
| Pass criteria | All fields match expected values |
| Evidence | Parsed config fields logged |

The config is the primary scope containment mechanism at the config layer.

### PS-5: API Key Permission (Operator Attestation)

| Item | Detail |
|------|--------|
| What it tests | Operator confirms in Binance console: trade-only, no withdrawal |
| Mechanism | `OPERATOR_ATTESTS_TRADE_ONLY=true` environment variable |
| Pass criteria | Operator explicitly attests trade-only permissions |
| Evidence | Attestation logged with operator name |

This is a human-in-the-loop check. The system cannot verify API key permissions programmatically without making an authenticated call to the account endpoint.

### PS-6: Kill-Switch Initial State

| Item | Detail |
|------|--------|
| Command | `GET /execution/control` via gateway |
| What it tests | Gate is in `active` state before session |
| Pass criteria | Gate status = `active` |
| Evidence | Gate state logged |

After the PS-1 cycle test, the gate should be in `active` state. This check confirms.

### PS-7: System Boot Verification

| Item | Detail |
|------|--------|
| What it tests | Gateway and execute binaries are reachable and healthy |
| Endpoints | `GET /readyz` on both gateway and execute |
| Pass criteria | HTTP 200 from both endpoints |
| Additional | Execute `/statusz` and gateway `/execution/control` logged for audit |

## Session Protocol

### Order Lifecycle Path

The system follows this exact path for a live market order:

```
1. Ingest:    WebSocket receives BTCUSDT market data from Binance
2. Derive:    Pipeline processes candles, signals, decisions, risk assessments
3. Execute:   Risk assessment produces ExecutionIntent (side=buy or sell, qty=minimum)
4. Safety:    VenueAdapterActor checks:
              a. Segment source guard (S401)
              b. Kill-switch gate via SafetyGate (must be active)
              c. Staleness guard (intent must be recent)
5. Submit:    Real HTTP POST to https://api.binance.com/api/v3/order
              - HMAC-SHA256 signed with file-based credentials
              - Parameters: symbol=BTCUSDT, side=BUY/SELL, type=MARKET,
                quantity=<minimum>, newOrderRespType=FULL
6. Response:  Binance returns order response with fills[] array
7. Parse:     BinanceSpotMainnetAdapter parses response into VenueOrderReceipt
8. Publish:   VenueAdapterActor publishes VenueOrderFilledEvent to NATS
9. Persist:   Writer binary consumes fill event, writes to ClickHouse
10. KV:       NATS KV stores order lifecycle state
```

### What Happens With `dry_run=false`

With `dry_run=false` in the config (S445 C-6):
- `IsDryRun()` returns `false`
- DryRunSubmitter is NOT wrapped around the venue adapter (line 87-96 of `cmd/execute/run.go`)
- The real `BinanceSpotMainnetAdapter` receives the `SubmitOrder` call
- Real HTTP calls reach `api.binance.com`
- Real money is involved

### Safety Gates Active During Session

Even with `dry_run=false`, these gates remain active:

| Gate | Location | Effect |
|------|----------|--------|
| Kill-switch / SafetyGate | `venue_adapter_actor.go:246-285` | Blocks all orders if gate is `halted` |
| Staleness guard | SafetyGate | Rejects intents older than `staleness_max_age` |
| Segment source guard | `venue_adapter_actor.go:232-243` | Rejects intents from unauthorized sources |
| RetrySubmitter | Decorator pipeline | Retries transient failures with backoff |
| Post200Reconciler | Decorator pipeline | Recovers body-read-failure-after-200 |
| RateLimiter | `cmd/execute/run.go:317` | Rate-limits venue calls (10 per 100ms window) |
| Credential preflight | `preflight.go` | Binary exits at boot if credentials missing |

### Operator Responsibilities During Session

1. **Monitor system logs** continuously for any errors or anomalies.
2. **Watch for first order completion** (fill or reject) in logs.
3. **Immediately halt** after first order: `./scripts/kill-switch-ops.sh halt "s446-session-complete" "<name>"`
4. **Do not leave the session unattended** at any point.
5. **Activate kill-switch immediately** on any doubt or anomaly (SC-9: no penalty for false positives).

## Post-Session Verification

After the session ends (via kill-switch or operator decision), the following checks are performed by `scripts/smoke-supervised-live-session.sh post-session`:

### PO-1: Kill-Switch Halt

Verify gate is `halted`. If not, issue halt command.

### PO-2: Post-Session Backup

Run `clickhouse-scheduled-backup.sh` to capture the post-session state.

### PO-3: ClickHouse Intent Records

Query `execution_intents` table for BTCUSDT records. Expected: at least 1 record with correct symbol, side, quantity, and timestamp within the session window.

### PO-4: ClickHouse Venue Response Records

Query `venue_responses` table for BTCUSDT records. Expected: at least 1 record with venue order ID, status (filled or rejected), and fill details.

### PO-5: NATS KV State

Query execution control endpoint for current state. Expected: gate = halted, lifecycle state consistent with venue response.

### PO-6: System Status Summary

Capture execute `/statusz` for final health counters (processed, filled, rejected, errors).

## Stop Conditions

All stop conditions from the [S444 scope constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) are binding. Key conditions:

| ID | Condition | Action |
|----|-----------|--------|
| SC-1 | API error rate > 10% | Kill-switch halt |
| SC-3 | Unexpected order state | Kill-switch halt, full audit |
| SC-4 | Fill qty > requested qty | Kill-switch halt, investigate |
| SC-5 | Kill-switch fails SLA | DO NOT START / HALT ALL |
| SC-9 | Operator uncertainty | Kill-switch halt, no penalty |
| SC-12 | More than 1 order | Kill-switch halt, investigate |

## Operational Script

The canonical operational script for this session is:

```
scripts/smoke-supervised-live-session.sh
```

Commands:
- `pre-session` -- Run all 7 pre-session checks
- `monitor` -- Monitor the live session (polls every 10s)
- `post-session` -- Run post-session verification
- `full` -- Complete ceremony: pre-session + monitor + post-session

Required environment variables:
- `OPERATOR_NAME` -- Operator identity (required)
- `OPERATOR_ATTESTS_TRADE_ONLY=true` -- API key permission attestation
- `CREDENTIAL_PATH` -- Path to file-based credentials

## Evidence Artifacts Produced

| Artifact | Location | Content |
|----------|----------|---------|
| Session log | `backups/logs/sessions/live_<timestamp>.log` | Complete ceremony execution log |
| Pre-session backup | `backups/clickhouse/pre_session_*` | ClickHouse state before session |
| Post-session backup | `backups/clickhouse/post_session_*` | ClickHouse state after session |
| ClickHouse intent records | `execution_intents` table | ExecutionIntent with BTCUSDT |
| ClickHouse response records | `venue_responses` table | Venue response with order ID and fills |
| Kill-switch state log | Session log | Gate transitions throughout ceremony |

## Limitations

| # | Limitation | Impact |
|---|-----------|--------|
| 1 | Exact minimum quantity depends on exchange's `LOT_SIZE` filter at session time | Operator must verify via `GET /api/v3/exchangeInfo` before session |
| 2 | Order timing depends on pipeline generating an intent | Session length is indeterminate until first intent |
| 3 | Fill price depends on live market conditions | Exact fill price is not predictable |
| 4 | No automated halt after first fill | Operator must manually halt via kill-switch |
| 5 | ClickHouse persistence depends on writer binary health | Writer must be running and healthy |
| 6 | No push alerting | Operator observation is the detection mechanism |

## References

- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints and Stop Conditions](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [C-6 Controlled Removal](c6-controlled-dry-run-false-removal.md) (S445)
- [Scope Guards and Fail-Closed Behavior](live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md) (S445)
- [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md) (S443)
- [Kill-Switch Operational Runbook](kill-switch-operational-runbook.md) (S442)
