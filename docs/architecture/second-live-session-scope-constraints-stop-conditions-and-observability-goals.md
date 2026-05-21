# Second Live Session -- Scope Constraints, Stop Conditions, and Observability Goals

> Authority: S457 | Date: 2026-03-24 | Phase: 53 (Second Supervised Live Session Wave)

## Purpose

This document is the operational companion to the [wave charter](second-supervised-live-session-wave-charter-and-scope-freeze.md). It defines with binding precision:

1. The authorized scope for the second supervised live session.
2. The stop conditions that trigger immediate halt.
3. The rollback criteria for each ceremony block.
4. The explicit non-goals that bound the wave.
5. The observability goals that differentiate this session from the first.

Every constraint in this document is inherited from the S444 charter, the S443 authorization verdict, and the S451 stabilization decision. No constraint has been relaxed. Several have been tightened based on S449 findings.

## Minimum Authorized Scope

### Scope Table (Identical to S444/S449)

| Dimension | Authorized Value | Hard Limit | Enforcement Point |
|-----------|-----------------|------------|-------------------|
| Exchange | Binance | 1 exchange | Config validation (schema.go) |
| Segment | Spot | 1 segment | Config: single adapter block |
| Symbol | BTCUSDT | 1 symbol | Config: `symbols` array length = 1 |
| Order size | Minimum exchange quantity | Exchange-defined floor | Config: quantity field |
| Order type | Market | 1 type | Domain model: only market in lifecycle |
| Order count per session | 1 | Exactly 1 | Operator discipline + session protocol |
| Credential scope | Trade-only (no withdrawal) | API key permission | Operator verification (PS-5) |
| Credential provider | File-based | No env vars for mainnet | Config: `credential_provider: "file"` |
| Operator presence | Required throughout | No unattended operation | Ceremony protocol |
| Kill-switch | Active and tested before session | Must be available | Pre-session check PS-1 |
| Backup | Before and after session | Automated with off-host | Pre-session PS-2, post-session PO-2 |

### S449 Deviation Corrections (Mandatory)

The following deviations from the first session protocol are **binding corrections** for the second session. Any repeat of these deviations invalidates the session for stabilization purposes.

| Deviation in S449 | S457 Correction | Enforcement |
|--------------------|-----------------|-------------|
| PS-2: pre-session backup not executed | Backup must complete before session starts | Checklist gate: session cannot begin until PS-2 passes |
| PS-3: credential_provider was `env` not `file` | File-based provider is mandatory | Config audit PS-4 verifies `credential_provider: "file"` |
| Post-session backup not executed | PO-2 must complete after session ends | Checklist gate: review cannot begin until PO-2 passes |
| Only 2 of 9 PO checks executed | All 9 PO checks mandatory | Review protocol: session evidence is incomplete without full PO |
| Infrastructure friction undocumented | Setup guide from S449 findings must be followed | Operator preparation: review S449 friction log before session |

### What "Minimum Exchange Quantity" Means

For BTCUSDT on Binance Spot, the minimum order quantity is defined by the exchange's `LOT_SIZE` filter. The exact minimum must be confirmed by the operator from `GET /api/v3/exchangeInfo` before the session. The config must specify this exact minimum. Any quantity above the exchange-defined minimum is a scope violation.

## Stop Conditions

### Inherited from S444 (Binding)

All stop conditions from the first ceremony remain in force without modification:

| ID | Condition | Detection Method | Required Action |
|----|-----------|-----------------|----------------|
| SC-1 | API error rate exceeds 10% | System logs, operator observation | Kill-switch halt |
| SC-2 | Latency exceeds 5x baseline | System logs, operator observation | Kill-switch halt |
| SC-3 | Unexpected order state (neither Accepted nor Rejected) | Venue response parsing | Kill-switch halt. Full state audit. |
| SC-4 | Fill quantity exceeds requested quantity | Venue response validation | Kill-switch halt. Investigate. |
| SC-5 | Kill-switch fails to respond within SLA (2s) | Pre-session test / runtime | DO NOT START or HALT ALL |
| SC-6 | Credential error during session | System logs | Kill-switch halt |
| SC-7 | ClickHouse write failure | System logs | Kill-switch halt |
| SC-8 | NATS connectivity loss | System health monitoring | Kill-switch halt |
| SC-9 | Operator uncertainty about system behavior | Operator judgment | Kill-switch halt. No penalty for false positives. |
| SC-10 | Order submitted to wrong symbol | Config audit | Kill-switch halt. Full audit. |
| SC-11 | Order submitted to wrong segment (Futures instead of Spot) | Segment guard | Kill-switch halt. Requires re-authorization. |
| SC-12 | More than 1 order observed | Intent generation monitoring | Kill-switch halt. Protocol violation. |
| SC-13 | System boots with unexpected config values | Pre-session audit | DO NOT START SESSION |
| SC-14 | Pre-session check fails | Pre-session checklist | DO NOT START SESSION |

### Second-Session-Specific Stop Conditions (New)

| ID | Condition | Required Action |
|----|-----------|----------------|
| SC-15 | S449 deviation repeated (backup skip, env provider, incomplete PO) | Session is INVALID. Halt and correct before re-attempt. |
| SC-16 | Observability endpoints unreachable during post-session | Session evidence is degraded. Document gap. May require additional review. |
| SC-17 | Explain endpoint reports cross-surface divergence during PO | Investigate before rendering session verdict. |

### Stop Condition Escalation

| Severity | Conditions | Resumption Path |
|----------|-----------|----------------|
| Session-ending (no re-attempt) | SC-3, SC-4, SC-5, SC-11, SC-12 | Requires new authorization ceremony |
| Session-ending (re-attempt possible) | SC-1, SC-2, SC-6, SC-7, SC-8, SC-10, SC-15 | Fix root cause, re-verify all checks |
| Session-pausing | SC-9, SC-13, SC-14, SC-16, SC-17 | Resolve concern, re-verify, operator decision |

## Rollback Criteria

### Block 1 (Session Execution -- S458) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-1 | Pre-session check failure | DO NOT START. Fix and re-verify from PS-1. |
| RB-2 | Any stop condition triggered | Kill-switch halt. Session ends. Evidence collected as-is. |
| RB-3 | System fails to boot with live config | Debug boot failure. Do not proceed. |
| RB-4 | S449 deviation repeated | Session INVALID. Correct deviation. Re-attempt with full pre-session. |

### Block 2 (Post-Session Review -- S459) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-5 | ClickHouse persistence missing | Document as finding. Investigate root cause. Does not invalidate session if order was observed live. |
| RB-6 | NATS KV state inconsistent | Document as finding. Investigate. |
| RB-7 | Observability endpoints return no data | Document as gap. Exercise alternative query paths (direct ClickHouse). |
| RB-8 | Explain endpoint reports divergence | Investigate before rendering verdict. May require additional data collection. |

### Block 3 (GO/NO-GO Decision -- S460) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-9 | Evidence insufficient for Scope Expansion | Render ADDITIONAL STABILIZATION REQUIRED. Document gaps. Operator decides. |
| RB-10 | Safety incident during session | Render LIVE SAFETY CLOSURE. Full audit. |

## Non-Goals

### Scope Expansion Non-Goals (Inherited from S444)

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-1 | Futures live trading | Spot-first. Separate ceremony required. |
| NG-2 | New symbols (beyond BTCUSDT) | Single-symbol scope frozen. |
| NG-3 | Multi-exchange support | Binance-only. |
| NG-4 | Limit orders, amendments, cancel path | Market-order-only. |
| NG-5 | Order sizing beyond minimum | Minimum quantity mandatory. |
| NG-6 | Multiple orders per session | Single order per ceremony. |
| NG-7 | Withdrawal-capable API keys | Trade-only is hard constraint. |

### Operational Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-8 | Automated or unmonitored trading | Human-in-the-loop mandatory. |
| NG-9 | Per-segment kill-switch | Global kill-switch sufficient for Spot-only. |
| NG-10 | Credential hot-swap without restart | Accepted limitation. |
| NG-11 | Push alerting or automated halt triggers | Operator observation is detection mechanism. |
| NG-12 | Dashboard, UI, or visualization | Operational signals remain HTTP/JSON and log-based. |

### Architecture Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-13 | Runtime, adapter, or actor redesign | Architecture proven across 17+ waves. |
| NG-14 | Config or compose surface changes | Canonical surfaces preserved. |
| NG-15 | OTEL tracing or advanced observability | S452A--S456A endpoints are the observability layer. |
| NG-16 | Portfolio risk or PnL calculation | Out of scope. |
| NG-17 | Fee optimization or rebate tracking | Fee model is sufficient as-is. |
| NG-18 | OMS expansion or new order states | Order lifecycle model frozen. |

## Observability Goals

The second session has an explicit observability mandate that the first session did not. The S452A--S456A wave delivered 5 HTTP endpoints and a cross-surface consistency framework. The second session must exercise these capabilities and produce evidence that is structurally richer than the first.

### Observability Objectives

| ID | Objective | How to Verify | Comparison with S449 |
|----|-----------|---------------|---------------------|
| OBS-1 | Full lifecycle timeline available via HTTP | `GET /analytical/execution/lifecycle?source=binances&symbol=btcusdt` returns events | S449: endpoint did not exist |
| OBS-2 | Execution list shows real order with correct type | `GET /analytical/execution/list` shows `type=venue_market_order` | S449: type was `paper_order` (F4) |
| OBS-3 | Execution summary shows non-zero fill statistics | `GET /analytical/execution/summary` returns fill count > 0 | S449: all noop fills |
| OBS-4 | Explain endpoint confirms cross-surface consistency | `GET /analytical/execution/explain` shows no divergence | S449: endpoint did not exist |
| OBS-5 | ClickHouse record count matches KV event count | Compare CH query count with KV enumeration | S449: 50% gap (F3) |
| OBS-6 | Status field reflects real lifecycle state | Query shows `status=accepted` or `status=filled` | S449: status stuck at `submitted` (F5) |
| OBS-7 | Fee/commission fields populated from real data | Query shows non-zero commission, commission_asset | S449: noop fills, zero fees |
| OBS-8 | Pre and post session backups verifiable | Backup files exist with timestamps bracketing session | S449: neither backup executed |
| OBS-9 | All 9 PO checks produce documented evidence | PO-1 through PO-9 each have recorded output | S449: only PO-1 and PO-6 attempted |

### Observability Success Matrix

The second session is observationally complete when ALL of the following are true:

| Criterion | Required Evidence |
|-----------|-------------------|
| Lifecycle timeline | At least one `venue_market_order` event in lifecycle response |
| Type disambiguation | Zero `paper_order` records for the live session window |
| Status progression | At least one record with `status=accepted` or `status=filled` |
| Fee population | At least one record with `commission > 0` |
| Cross-surface consistency | Explain endpoint reports `consistent: true` for the session's partition key |
| Persistence completeness | CH record count = KV event count for session window |
| Backup bracket | Pre-session backup timestamp < session start < session end < post-session backup timestamp |

### What These Observability Goals Enable

If the observability matrix is fully satisfied, the system transitions from "can execute but cannot explain" (S452A problem statement) to "can execute AND explain what it executed." This transition is the prerequisite for the S460 GO/NO-GO decision to authorize Spot Scope Expansion.

## Pre-Session Checklist (Consolidated)

This is the complete, ordered checklist that must be executed before the second live session begins. Every item must pass. Any failure aborts the session. **No deviations are permitted.**

| Order | ID | Check | Method | Pass Criteria |
|-------|-----|-------|--------|---------------|
| 1 | PS-1 | Kill-switch cycle test | `./scripts/kill-switch-ops.sh cycle` | Halt and resume within 2s SLA |
| 2 | PS-2 | Pre-session backup | `./scripts/clickhouse-scheduled-backup.sh` | Backup + off-host replication succeed |
| 3 | PS-3 | Credential file mount | `ls -la <credential_path>/spot/` | Files exist, non-empty |
| 4 | PS-4 | Config audit | Read `deploy/configs/execute-mainnet-live.jsonc` | 1 symbol, min size, Spot, dry_run=false, credential_provider=file |
| 5 | PS-5 | API key permission | Operator checks Binance console | Trade-only, no withdrawal |
| 6 | PS-6 | Kill-switch state | `./scripts/kill-switch-ops.sh status` | Gate is `active` |
| 7 | PS-7 | System boot | Start execute binary with live config | Boot succeeds, preflight passes |

## Post-Session Checklist (Consolidated)

| Order | ID | Check | Method | Expected |
|-------|-----|-------|--------|----------|
| 1 | PO-1 | Kill-switch halt | `./scripts/kill-switch-ops.sh halt -r "s458 session complete" -b <operator>` | Gate transitions to `halted` |
| 2 | PO-2 | Post-session backup | `./scripts/clickhouse-scheduled-backup.sh` | Backup + off-host succeed |
| 3 | PO-3 | ClickHouse intent record | Query `execution_intents` table | BTCUSDT, correct timestamp |
| 4 | PO-4 | ClickHouse response record | Query `venue_responses` table | Order ID, status, fill details |
| 5 | PO-5 | NATS KV order state | Check KV store | Lifecycle state matches venue response |
| 6 | PO-6 | System shutdown | Stop all binaries | Clean shutdown |
| 7 | PO-7 | Fee/commission verification | Query fee fields | Populated from real data |
| 8 | PO-8 | Lifecycle consistency | Compare KV and ClickHouse + explain endpoint | No divergence |
| 9 | PO-9 | Evidence collection | Logs, queries, endpoint responses, screenshots | Complete evidence set |

## Accepted Limitations

Inherited from S444 and S452A--S456A. Unchanged.

| # | Limitation | Origin | Impact on Second Session |
|---|-----------|--------|--------------------------|
| 1 | Credential rotation requires restart | RG-S439-1 | None -- single session |
| 2 | No multi-provider fallback | RG-S439-2 | None -- single provider |
| 3 | No push alerting on backup failure | RG-S440-1 | None -- operator runs backup |
| 4 | No S3/GCS backup integration | RG-S440-2 | None -- rsync sufficient |
| 5 | No WebSocket authenticated streams | RG-S441-3 | None -- REST path proven |
| 6 | No per-segment kill-switch | RG-S442-1 | None -- single segment |
| 7 | No automated halt triggers | RG-S442-2 | None -- operator is mechanism |
| 8 | No HTTP auth on gateway | RG-S442-3 | None -- localhost binding |
| 9 | No first-class session metadata entity | G1 from S456A | Explain endpoint provides session-scoped view |
| 10 | No batch KV-to-CH consistency audit | G2 from S456A | Explain endpoint provides per-key audit |
| 11 | No automated PO check harness | G3 from S456A | Operator executes PO checks manually per checklist |
| 12 | No cursor-based pagination | G6 from S456A | Minimum scope: single order, pagination unnecessary |

## References

- [Wave Charter](second-supervised-live-session-wave-charter-and-scope-freeze.md) (S457)
- [S444 Scope Constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md)
- [S444 Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md)
- [S449 Execution Record](first-supervised-live-session-execution-record.md)
- [S451 GO/NO-GO Decision](go-no-go-decision-for-spot-scope-expansion.md)
- [S456A Evidence Gate](operational-history-and-explainability-evidence-gate.md)
- [Kill-Switch Runbook](kill-switch-operational-runbook.md)
- [Kill-Switch Procedures](kill-switch-trigger-verification-rollback-and-recovery-procedure.md)
