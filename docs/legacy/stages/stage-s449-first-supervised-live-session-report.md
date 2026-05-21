# Stage S449: First Supervised Live Session Report

Stage: S449
Predecessor: S448 (Live Trading Enablement Evidence Gate)
Date: 2026-03-24
Operator: fabio

## Objective

Execute, validate, and document the first supervised live session on Binance Spot mainnet under the minimum authorized scope, transforming the system state from "live-enabled with restrictions" toward "live observed in minimum production scope".

## Executive Summary

The first supervised live session was executed on 2026-03-24 between 14:42Z and 15:00Z. The system ran in genuine `venue_live` mode on Binance mainnet with `dry_run=false`, `binance_spot_mainnet` adapter, and real market data from `wss://stream.binance.com`. The full pipeline (ingest -> derive -> execute) processed real BTCUSDT aggTrade data for approximately 15 minutes.

**No real order was submitted to Binance's order API.** The mean_reversion_entry strategy evaluated 16 times and returned `direction=flat` on every evaluation because market conditions (BTCUSDT ~$69,600-$69,835) did not trigger the RSI oversold threshold. The venue adapter processed 28 intents as noop fills without making authenticated API calls.

The session confirmed that the system is operationally live on mainnet but the transition from "infrastructure-ready" to "real-order-observed" requires either longer session duration or market conditions that trigger the strategy.

## Session Timeline

| Time (UTC) | Event |
|------------|-------|
| 14:42:13 | Preflight start |
| 14:42:14 | PS-1: Kill-switch cycle test PASS |
| 14:43:01 | Execute boot: venue_live, dry_run=false, binance_spot_mainnet |
| 14:45:14 | Binding activated: binances.btcusdt |
| 14:45:16 | WebSocket connected: wss://stream.binance.com:9443/ws/btcusdt@aggTrade |
| 14:46:00 | First candle finalized: BTCUSDT close=$69,631.74, 4007 trades |
| 14:57:xx | Execute phase transitions to active |
| 14:58:00 | First noop fills processed (side=none, qty=0) |
| 14:59:00 | Strategy evaluations: all flat |
| 15:00:43 | Session HALT issued: reason=s449-session-complete-no-real-order |
| 15:00:43 | Halt verified: gate=halted, execute reachable |

## Preflight Results

| Check | Result | Notes |
|-------|--------|-------|
| PS-1: Kill-switch cycle | PASS | 4-step cycle in ~4s |
| PS-2: Pre-session backup | NOT EXECUTED | Deviation (no financial impact) |
| PS-3: Credential mount | DEVIATED | env provider used instead of file |
| PS-4: Config audit | PASS | dry_run=false, spot-only, binance_spot_mainnet |
| PS-5: Operator attestation | PASS | Trade-only confirmed |
| PS-6: Gate state | PASS | active |
| PS-7: System boot | PASS | Gateway + Execute healthy |

5/7 PASS, 2 deviations documented.

## Infrastructure Friction Log

Five infrastructure issues were encountered and resolved during preflight:

| # | Issue | Resolution | Time to Fix |
|---|-------|-----------|-------------|
| 1 | Mainnet credential env vars missing | Added entries to local.env | ~2 min |
| 2 | Execute crash: credential preflight fail | Changed compose to env_file | ~3 min |
| 3 | NATS consumer config conflict | Deleted stale consumer | ~3 min |
| 4 | Execute port not exposed to host | Added port mapping to overlay | ~2 min |
| 5 | No BTCUSDT binding in configctl | Ran seed-configctl.sh | ~1 min |

Total friction: ~11 minutes of debugging before data flow started. All issues were resolvable without code changes (compose/config/operational adjustments only).

## Execution Evidence

### Venue Adapter Final State

| Metric | Value |
|--------|-------|
| Processed | 28 |
| Filled (noop) | 24 |
| Rejected | 0 |
| Skipped (halt) | 4 |
| Skipped (stale) | 0 |
| Errors | 0 |
| Real API calls | 0 |

### Strategy Evaluation

| Metric | Value |
|--------|-------|
| Events received | 16 |
| Events evaluated | 16 |
| Direction flat | 16 (100%) |
| Direction long/short | 0 |

### ClickHouse Persistence

12 noop execution records written (type=paper_order, side=none, quantity=0, status=submitted).

### Stop Conditions

Zero stop conditions triggered. All 14 conditions evaluated clean.

## Deviations from S446 Protocol

| # | Deviation | Severity | Impact |
|---|-----------|----------|--------|
| 1 | credential_provider: env (not file) | LOW | Same code path; security posture equivalent for single-machine |
| 2 | Pre-session backup not executed | LOW | No financial records produced |
| 3 | Post-session backup not executed | LOW | No financial records produced |
| 4 | Config created as S449-specific copy | LOW | Tracked explicitly; base config unchanged |
| 5 | No real order observed | HIGH (relative to objective) | Market conditions, not system limitation |

## Artifacts Produced

| Artifact | Path |
|----------|------|
| Execution record | docs/architecture/first-supervised-live-session-execution-record.md |
| Preflight and behavior log | docs/architecture/first-live-session-preflight-observed-behavior-and-stop-condition-log.md |
| Stage report | docs/stages/stage-s449-first-supervised-live-session-report.md |
| S449 config | deploy/configs/execute-mainnet-live-s449.jsonc |
| Compose overlay | deploy/compose/docker-compose.mainnet-live.yaml |

## State Transition Assessment

| Dimension | Pre-S449 (S448) | Post-S449 | Confidence |
|-----------|-----------------|-----------|------------|
| Mainnet data ingestion | INFRASTRUCTURE | **OBSERVED** | CONCRETE |
| Pipeline processing (live) | INFRASTRUCTURE | **OBSERVED** | CONCRETE |
| Strategy evaluation (live) | INFRASTRUCTURE | **OBSERVED** | CONCRETE |
| Kill-switch (live stack) | SCRIPTED | **TESTED** | CONCRETE |
| Venue adapter (venue_live) | INFRASTRUCTURE | **OBSERVED** | CONCRETE |
| Noop path (flat signals) | Untested | **OBSERVED** | CONCRETE |
| Real order submission | INFRASTRUCTURE | **INFRASTRUCTURE** | NOT YET OBSERVED |
| Real fill parsing | INFRASTRUCTURE | **INFRASTRUCTURE** | NOT YET OBSERVED |
| Fee/commission fields (real) | INFRASTRUCTURE | **INFRASTRUCTURE** | NOT YET OBSERVED |

**6 dimensions advanced from INFRASTRUCTURE to OBSERVED. 3 dimensions remain at INFRASTRUCTURE.**

## Honest Verdict

**CEREMONY COMPLETED -- PARTIAL OBSERVATION**

The S449 ceremony delivered its core objective (execute the first supervised live session) but did not deliver the ultimate evidence (a real order submitted and filled on Binance). The system is provably operational in live mode on mainnet with real data. The gap is not a system failure -- it is a market condition gap.

### What S449 Proved

- The system runs in `venue_live` mode on Binance mainnet without errors
- Real market data flows end-to-end through the production pipeline
- The strategy evaluation pipeline works on live data
- Safety mechanisms (kill-switch, staleness, segment guard) function in a live stack
- The noop path correctly prevents real API calls for flat signals
- Operational tooling works against a live stack
- The operator can start, monitor, and halt a live session

### What S449 Did Not Prove

- A real order can be submitted and accepted by Binance
- The HMAC-SHA256 signing works for order submission (only data WebSocket was tested)
- A real fill response can be parsed
- Fee and commission fields from a real response
- The complete order lifecycle from submission to persistence

## Preparation for S450

### Option A: Extended Session (Recommended)

Run a longer supervised session (1-4 hours) to increase the probability of the strategy triggering a real signal. Market conditions with higher volatility (Asian or US market hours) increase the chance of RSI oversold triggers.

### Option B: Minimal Manual Trigger

Create a one-shot manual execution intent that forces a real `side=buy, quantity=minimum` through the pipeline. This bypasses the strategy layer but directly tests the venue adapter, signing, API call, response parsing, and persistence. This is the fastest path to "real-order-observed" evidence.

### Option C: Strategy Parameter Adjustment

Temporarily adjust the RSI oversold threshold (e.g., from 30 to 60) to trigger signals more easily. This risks generating real orders at non-ideal prices but would exercise the full path.

### Infrastructure Fixes for S450

1. Create a canonical `docker-compose.mainnet-live.yaml` with all fixes from S449 (port mapping, env_file)
2. Document the NATS consumer cleanup requirement for first-time mainnet-live startups
3. Document the configctl seed requirement
4. Consider adding a "manual order" operational script for testing purposes

## References

- [Execution Record](../architecture/first-supervised-live-session-execution-record.md) (S449)
- [Preflight and Behavior Log](../architecture/first-live-session-preflight-observed-behavior-and-stop-condition-log.md) (S449)
- [S448 Evidence Gate](../architecture/live-trading-enablement-evidence-gate.md)
- [S446 Supervised Live Session Proof](../architecture/supervised-live-session-proof.md)
- [S444 Scope Constraints](../architecture/live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md)
- [Kill-Switch Runbook](../architecture/kill-switch-operational-runbook.md)
