# Live Trading Enablement -- Scope Constraints, Stop Conditions, and Non-Goals

> Authority: S444 | Date: 2026-03-24 | Phase: 51 (Live Trading Enablement Ceremony)

## Purpose

This document is the operational companion to the [ceremony charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md). It defines with binding precision:

1. The minimum authorized scope for live trading.
2. The stop conditions that trigger immediate halt.
3. The rollback criteria for each ceremony block.
4. The explicit non-goals that bound the ceremony.

Every constraint in this document is inherited from the S443 authorization verdict and the S438 wave charter. No constraint has been relaxed. Several have been tightened for the ceremony context.

## Minimum Authorized Scope

### Scope Table

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
| Backup | Before and after session | Automated with off-host | Pre-session PS-2, post-session |

### What "Minimum Exchange Quantity" Means

For BTCUSDT on Binance Spot, the minimum order quantity is defined by the exchange's `LOT_SIZE` filter. As of the charter date, this is approximately 0.00001 BTC. The exact minimum must be confirmed by the operator from the exchange's `GET /api/v3/exchangeInfo` endpoint before the session.

The config must specify this exact minimum. Any quantity above the exchange-defined minimum is a scope violation.

### Scope Enforcement Summary

| Layer | What It Prevents |
|-------|-----------------|
| Config validation (schema.go) | Invalid adapter combinations, missing fields |
| Config profile (execute-mainnet-live.jsonc) | Scope locked to 1 symbol, 1 segment, min size |
| DryRunSubmitter removal | Only in live config; all other profiles retain it |
| SafetyGate / kill-switch | Operator can halt at any moment |
| Operator presence | Human judgment on any anomaly |
| Session protocol | Exactly 1 order, then halt |

## Stop Conditions

### Inherited from S438/S443 (Binding)

These stop conditions were established by the authorization wave and are mandatory for any live session. Any trigger causes **immediate kill-switch activation**.

| ID | Condition | Detection Method | Required Action |
|----|-----------|-----------------|----------------|
| SC-1 | API error rate exceeds 10% | System logs, operator observation | Kill-switch halt. Do not resume until exchange health confirmed. |
| SC-2 | Latency exceeds 5x baseline | System logs, operator observation | Kill-switch halt. Investigate network/exchange conditions. |
| SC-3 | Unexpected order state (neither Accepted nor Rejected) | Venue response parsing | Kill-switch halt. Full state audit required. May require re-authorization. |
| SC-4 | Fill quantity exceeds requested quantity | Venue response validation | Kill-switch halt. Investigate exchange behavior. Report to exchange if confirmed. |
| SC-5 | Kill-switch fails to respond within SLA (2s) | Operator test (PS-1) | DO NOT START SESSION if detected in pre-session. HALT ALL if detected during session. |
| SC-6 | Credential error during session | System logs | Kill-switch halt. Verify credential state. |
| SC-7 | ClickHouse write failure | System logs | Kill-switch halt. Audit trail integrity is non-negotiable. |
| SC-8 | NATS connectivity loss | System health monitoring | Kill-switch halt. Control plane must be reliable. |
| SC-9 | Operator uncertainty about system behavior | Operator judgment | Kill-switch halt. No penalty for false positives. Any doubt = halt. |

### Ceremony-Specific Stop Conditions (New)

| ID | Condition | Required Action |
|----|-----------|----------------|
| SC-10 | Order submitted to wrong symbol | Kill-switch halt. Full config audit. Ceremony cannot resume without re-verification. |
| SC-11 | Order submitted to wrong segment (Futures instead of Spot) | Kill-switch halt. Critical failure. Requires full re-authorization. |
| SC-12 | More than 1 order observed | Kill-switch halt. Investigate intent generation. Session protocol violation. |
| SC-13 | System boots with unexpected config values | DO NOT START SESSION. Re-audit config. |
| SC-14 | Pre-session check fails | DO NOT START SESSION. Fix and re-verify all checks from the beginning. |

### Stop Condition Escalation

| Severity | Conditions | Resumption Path |
|----------|-----------|----------------|
| Session-ending (no re-attempt) | SC-3, SC-4, SC-5, SC-11, SC-12 | Requires new authorization ceremony |
| Session-ending (re-attempt possible) | SC-1, SC-2, SC-6, SC-7, SC-8, SC-10 | Fix root cause, re-verify all pre-session checks, operator decision |
| Session-pausing | SC-9, SC-13, SC-14 | Resolve concern, re-verify, operator decision |

## Rollback Criteria

### Block 1 (C-6 Execution) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-1 | Any test regression after removing dry_run=false rejection | `git revert` the commit. Investigate. Do not proceed. |
| RB-2 | Safety invariant SI-2 through SI-12 broken | `git revert`. Full safety audit. May require re-authorization. |
| RB-3 | DryRunSubmitter behavior changes for non-live configs | `git revert`. The removal must be scoped to mainnet+live only. |

### Block 2 (Live Session) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-4 | Any stop condition triggered | Kill-switch halt. Session ends. Evidence collected as-is. |
| RB-5 | System fails to boot with live config | Do not proceed. Debug boot failure. May require Block 1 revert. |
| RB-6 | Pre-session check failure | Do not start session. Fix and re-verify. |

### Block 3 (Post-Session) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-7 | ClickHouse persistence missing or inconsistent | Document as ceremony finding. Does not invalidate session if order was observed live. |
| RB-8 | NATS KV state inconsistent | Document as ceremony finding. Investigate root cause. |

### Block 4 (Evidence Gate) Rollback

| ID | Trigger | Action |
|----|---------|--------|
| RB-9 | Evidence insufficient for verdict | Render CEREMONY INCOMPLETE. Document gaps. Operator decides on re-attempt. |

## Non-Goals

### Scope Expansion Non-Goals

| ID | Non-Goal | Why It Is Excluded |
|----|----------|-------------------|
| NG-1 | Futures live trading | Spot-first mandate from S443. Futures requires separate ceremony after Spot is proven. |
| NG-2 | Multi-symbol trading | Single-symbol scope from S443. Cross-symbol interference risk must be zero for first session. |
| NG-3 | Multi-exchange support | Binance-only from S443. No other exchange has proven adapters. |
| NG-4 | Limit orders, amendments, cancel path | Market-order-only lifecycle is frozen. OMS expansion is a separate concern. |
| NG-5 | Order sizing beyond minimum | Minimum quantity limits financial exposure. Sizing expansion requires its own evidence. |
| NG-6 | Multiple orders per session | First ceremony is single-order. Repeated execution requires evidence from first success. |
| NG-7 | Withdrawal-capable API keys | Trade-only is a hard constraint. Withdrawal capability is never authorized for execution keys. |

### Operational Non-Goals

| ID | Non-Goal | Why It Is Excluded |
|----|----------|-------------------|
| NG-8 | Automated or unmonitored trading | Human-in-the-loop is mandatory for the enablement ceremony. |
| NG-9 | Per-segment kill-switch | Global kill-switch is sufficient for single-segment scope. |
| NG-10 | Credential hot-swap without restart | Accepted limitation from S439. Restart-based rotation is sufficient. |
| NG-11 | Push alerting or automated halt triggers | Operator observation is the detection mechanism for the ceremony. |
| NG-12 | Dashboard, UI, or visualization | Operational signals remain HTTP/JSON and log-based. |

### Architecture Non-Goals

| ID | Non-Goal | Why It Is Excluded |
|----|----------|-------------------|
| NG-13 | Runtime, adapter, or actor redesign | Architecture is proven across 15 consecutive waves. No changes. |
| NG-14 | Config or compose surface re-expansion | Canonical surfaces preserved. |
| NG-15 | OTEL tracing or advanced observability | Deferred to post-enablement hardening. |
| NG-16 | Portfolio risk management or PnL calculation | Out of scope for execution engine. |
| NG-17 | Fee optimization or rebate tracking | Fee model is sufficient as-is. |
| NG-18 | Cloud deployment or infrastructure changes | Deployment topology is operator's choice. |
| NG-19 | Documentation governance or restructuring | Separate concern. |
| NG-20 | Resolution of accepted LOW gaps (RG-S439-* through RG-S442-*) | These are accepted for minimum scope. Post-enablement concern. |

## Pre-Session Checklist (Consolidated)

This is the complete, ordered checklist that must be executed before the live session begins. Every item must pass. Any failure aborts the session.

| Order | ID | Check | Method | Pass Criteria |
|-------|-----|-------|--------|---------------|
| 1 | PS-1 | Kill-switch cycle test | `./scripts/kill-switch-ops.sh cycle` | Halt and resume both succeed, SLA met (2s) |
| 2 | PS-2 | Automated backup (pre-session) | `./scripts/clickhouse-scheduled-backup.sh` | Backup + off-host replication succeed |
| 3 | PS-3 | Credential file mount | `ls -la <credential_path>/spot/` | API key and secret files exist, non-empty |
| 4 | PS-4 | Config audit | Read `deploy/configs/execute-mainnet-live.jsonc` | 1 symbol (BTCUSDT), min size, Spot only, dry_run=false, credential_provider=file |
| 5 | PS-5 | API key permission | Operator checks Binance console | Trade-only, no withdrawal, IP restriction recommended |
| 6 | PS-6 | Kill-switch state | `./scripts/kill-switch-ops.sh status` | Gate is `active` |
| 7 | PS-7 | System boot | Start execute binary with live config | Boot succeeds, preflight passes, logs show live adapter wired |

## Post-Session Checklist (Consolidated)

| Order | ID | Check | Method | Expected |
|-------|-----|-------|--------|----------|
| 1 | PO-1 | Kill-switch halt | `./scripts/kill-switch-ops.sh halt -r "session complete" -b <operator>` | Gate transitions to `halted` |
| 2 | PO-2 | Automated backup (post-session) | `./scripts/clickhouse-scheduled-backup.sh` | Backup + off-host replication succeed |
| 3 | PO-3 | ClickHouse intent record | Query execution_intents table | Record with BTCUSDT, correct timestamp |
| 4 | PO-4 | ClickHouse response record | Query venue_responses table | Record with order ID, status, fill details |
| 5 | PO-5 | NATS KV order state | Check KV store | Lifecycle state matches venue response |
| 6 | PO-6 | System shutdown | Stop all binaries | Clean shutdown, no orphaned processes |

## Accepted Limitations

These limitations are inherited from the authorization wave and acknowledged for the ceremony:

| # | Limitation | Origin | Impact on Ceremony |
|---|-----------|--------|-------------------|
| 1 | Credential rotation requires restart | RG-S439-1 | None -- single session, no rotation needed |
| 2 | No multi-provider fallback | RG-S439-2 | None -- single provider is sufficient |
| 3 | No push alerting on backup failure | RG-S440-1 | None -- operator runs backup manually as checklist step |
| 4 | No S3/GCS backup integration | RG-S440-2 | None -- rsync off-host is sufficient |
| 5 | No WebSocket authenticated streams | RG-S441-3 | None -- REST execution path is the proven path |
| 6 | No per-segment kill-switch | RG-S442-1 | None -- single segment, global kill-switch is sufficient |
| 7 | No automated halt triggers | RG-S442-2 | None -- operator is the detection mechanism |
| 8 | No HTTP auth on gateway | RG-S442-3 | None -- localhost binding |
| 9 | Fail-open on NATS KV unavailability | RG-S442-5 | NATS loss is a stop condition (SC-8); operator monitors |

## References

- [Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md) (S443)
- [Live Trading Authorization Evidence Matrix](live-trading-authorization-evidence-matrix-blockers-conditions-and-next-ceremony.md) (S443)
- [Kill-Switch Operational Runbook](kill-switch-operational-runbook.md) (S442)
- [Kill-Switch Procedures](kill-switch-trigger-verification-rollback-and-recovery-procedure.md) (S442)
