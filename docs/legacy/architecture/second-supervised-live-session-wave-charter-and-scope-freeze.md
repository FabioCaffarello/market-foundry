# Second Supervised Live Session Wave -- Charter and Scope Freeze

## Wave Identity

| Field | Value |
|---|---|
| Wave | Second Supervised Live Session |
| Phase | 53 |
| Charter Stage | S457 |
| Planned Stages | S458--S460 |
| Predecessor Wave | Operational History & Explainability (S452A--S456A) |
| Predecessor Verdict | WAVE CLOSED -- SUBSTANTIALLY COMPLETE |
| Predecessor Decision | S451 -- Live Session Stabilization AUTHORIZED |
| Date Opened | 2026-03-24 |

## Strategic Context

### Chain of Authority

1. **S448** closed the Live Trading Enablement Ceremony with verdict **LIVE TRADING ENABLED -- WITH RESTRICTIONS**. The system can submit real orders to Binance Spot mainnet under minimum scope.
2. **S449** executed the first supervised live session. The full pipeline ran in `venue_live` mode on Binance mainnet for ~15 minutes. No real order was submitted because the strategy returned `direction=flat` on all 16 evaluations. Six dimensions advanced from INFRASTRUCTURE to OBSERVED; three remain at INFRASTRUCTURE (real order, real fill, real fees).
3. **S450** reviewed the first session and identified 5 operational findings (F3 persistence gap, F4 type confusion, F5 status stuck, F7 incomplete PO, F10 infrastructure friction).
4. **S451** rendered the GO/NO-GO decision: **Spot Scope Expansion BLOCKED; Live Session Stabilization AUTHORIZED**. The stabilization wave must close the operational gaps and then execute a second session at the same minimum scope to observe the real order execution path.
5. **S452A--S456A** delivered the Operational History & Explainability wave: 5 new HTTP endpoints, 48 tests, cross-surface consistency audit, and structured explainability. Verdict: SUBSTANTIALLY COMPLETE. F4 and F5 CLOSED. F3 MITIGATED (detection improved). F7 PARTIALLY CLOSED. F10 MITIGATED.

### What This Wave Is

This is a **ceremony of stabilization** -- a controlled, supervised, auditable second live session that:
- Operates under the identical scope as the first session (no expansion).
- Targets the three INFRASTRUCTURE dimensions that S449 did not advance: real order submission, real fill parsing, real fees/commission.
- Leverages the observability improvements from S452A--S456A to produce richer post-session evidence.
- Includes a structured comparison between the first and second sessions.
- Ends with a go/no-go decision on Spot Scope Expansion.

### What This Wave Is NOT

This is NOT scope expansion. It does not introduce new capabilities, exchanges, segments, symbols, order types, sizing changes, or runtime behaviors. The system architecture is unchanged. The wave removes zero guards and adds zero production paths. It validates the existing minimum path under improved observability.

## Wave Objective

1. Execute mandatory pre-session operational verification (kill-switch cycle, backup, credential mount, config audit) with all S449 deviations corrected.
2. Execute a second supervised live session at identical minimum scope, targeting real order evidence.
3. Leverage S452A--S456A observability endpoints during post-session review:
   - `GET /analytical/execution/lifecycle` -- full lifecycle timeline.
   - `GET /analytical/execution/list` -- filtered execution list.
   - `GET /analytical/execution/summary` -- aggregated summary.
   - `GET /analytical/execution/lifecycle/list` -- lifecycle list.
   - `GET /analytical/execution/explain` -- cross-surface consistency + explanation.
4. Execute full post-session verification protocol (all 9 PO checks).
5. Compare second session evidence with first session evidence (S449) across a structured matrix.
6. Render a go/no-go decision on Spot Scope Expansion based on stabilization evidence.

## Scope Freeze

### Minimum Authorized Scope (Unchanged from S444/S449)

| Dimension | Authorized Value | Hard Limit | Source |
|-----------|-----------------|------------|--------|
| Exchange | Binance | 1 exchange | S443 verdict |
| Segment | Spot | 1 segment | S443 verdict |
| Symbol | BTCUSDT | 1 symbol | S443 verdict |
| Order size | Minimum exchange quantity | Exchange-defined floor | S443 verdict |
| Order type | Market | 1 type | S443 verdict |
| Credentials | Trade-only (no withdrawal) | API key permission | S443 verdict |
| Credential provider | File-based | No env vars for mainnet | S444 charter |
| Kill-switch | Active and tested before session | Must be available | S443 verdict |
| Backup | Before and after session | Automated with off-host | S443 verdict |
| Operator presence | Required throughout | No unattended operation | S443 verdict |
| Duration | Operator-controlled; minimum sufficient for strategy signal | Operator judgment | S451 scope freeze |

This scope is **not expandable** within this wave. Any expansion requires a new authorization ceremony.

### Deviations from S449 That Must Be Corrected

| S449 Deviation | Required Correction | Rationale |
|----------------|---------------------|-----------|
| Pre-session backup not executed (PS-2) | Mandatory. Must complete before session starts. | S451 U3: no recovery path without backup |
| credential_provider: env instead of file | File-based provider mandatory. | S444 charter requirement |
| Post-session backup not executed | Mandatory. Must complete after session ends. | S451 U3 |
| Infrastructure friction undocumented | Setup guide must be consulted pre-session. | S451 U2 |
| Only 2 of 9 PO checks executed | All 9 PO checks mandatory. | S451 U1 |

### Session Strategy for Real Order Evidence

S449 ran for ~15 minutes and no strategy signal triggered. For the second session, the operator must choose one of the following approaches (decision at session time):

| Option | Description | Risk | Evidence Value |
|--------|-------------|------|----------------|
| A: Extended duration | Run 1-4 hours during volatile market window | LOW -- same minimum scope | Medium -- depends on market conditions |
| B: Manual execution intent | One-shot manual intent forcing `side=buy, quantity=minimum` | LOW -- same minimum scope, bypasses strategy | HIGH -- directly tests venue path |
| C: Strategy parameter adjustment | Temporarily widen RSI threshold for signal generation | MEDIUM -- may trigger at non-ideal price | HIGH -- exercises full pipeline |

**Recommended**: Option B (manual intent) provides the highest evidence value with the lowest uncertainty. The operator retains full control. The real order path is tested directly.

## Ceremony Blocks

### Block 1: Second Supervised Live Session Execution (S458)

Execute the second live session with all S449 deviations corrected.

**Mandatory pre-session checklist (all must pass):**

| # | Check | Method | Pass Criteria |
|---|-------|--------|---------------|
| PS-1 | Kill-switch cycle test | `./scripts/kill-switch-ops.sh cycle` | Halt and resume within 2s SLA |
| PS-2 | Pre-session backup | `./scripts/clickhouse-scheduled-backup.sh` | Backup + off-host replication succeed |
| PS-3 | Credential file mount | `ls -la <credential_path>/spot/` | API key and secret files exist, non-empty |
| PS-4 | Config audit | Read `deploy/configs/execute-mainnet-live.jsonc` | 1 symbol, min size, Spot only, dry_run=false, credential_provider=file |
| PS-5 | API key permission | Operator checks Binance console | Trade-only, no withdrawal |
| PS-6 | Kill-switch state | `./scripts/kill-switch-ops.sh status` | Gate is `active` |
| PS-7 | System boot | Start execute binary with live config | Boot succeeds, preflight passes |

**Session execution:**

1. Operator confirms all 7 pre-session checks passed (zero deviations allowed).
2. System boots in `venue_live` mode with Binance Spot mainnet adapter.
3. Operator selects the order trigger approach (A, B, or C).
4. If Option B: operator issues a manual execution intent for BTCUSDT market buy at minimum quantity.
5. VenueAdapterActor submits order to Binance Spot via real adapter (HMAC-SHA256 signed).
6. Order lifecycle proceeds: submit -> accept -> fill (or submit -> reject).
7. Operator monitors throughout. Any anomaly triggers immediate kill-switch.
8. After first completed order lifecycle (or after documented failure): kill-switch halt.

**Session scope:** exactly ONE order. The session ends after the first order lifecycle completes.

**Exit criteria:** One live order submitted. Order lifecycle observed (accept+fill or reject). Operator present throughout. All 7 pre-session checks passed. Zero S449 deviations repeated.

### Block 2: Post-Second-Session Operational Review (S459)

Full post-session verification using both the S447 protocol and S452A--S456A observability.

**Post-session checklist (all 9 PO checks mandatory):**

| # | Check | Method | Expected |
|---|-------|--------|----------|
| PO-1 | Kill-switch halt | `./scripts/kill-switch-ops.sh halt` | Gate transitions to `halted` |
| PO-2 | Post-session backup | `./scripts/clickhouse-scheduled-backup.sh` | Backup + off-host succeed |
| PO-3 | ClickHouse intent record | Query execution_intents table | Record with BTCUSDT, correct timestamp |
| PO-4 | ClickHouse response record | Query venue_responses table | Order ID, status, fill details |
| PO-5 | NATS KV order state | Check KV store | Lifecycle state matches venue response |
| PO-6 | System shutdown | Stop all binaries | Clean shutdown |
| PO-7 | Fee/commission verification | Query fee fields from venue response | Populated from real data |
| PO-8 | Lifecycle consistency | Compare KV and ClickHouse records | No divergence |
| PO-9 | Evidence collection | Collect logs, queries, screenshots | Complete evidence set |

**Observability endpoints to exercise:**

| Endpoint | Purpose | Comparison Dimension |
|----------|---------|---------------------|
| `GET /analytical/execution/lifecycle?source=binances&symbol=btcusdt` | Full lifecycle timeline | Completeness vs S449 |
| `GET /analytical/execution/list` | All execution events | Record count vs S449 |
| `GET /analytical/execution/summary` | Aggregated statistics | Status distribution vs S449 |
| `GET /analytical/execution/explain?source=binances&symbol=btcusdt` | Cross-surface audit | Consistency vs S449 |

**Session comparison matrix:**

| Dimension | S449 Result | S458 Target | Verdict Criteria |
|-----------|-------------|-------------|------------------|
| Pre-session checks | 5/7 PASS | 7/7 PASS | All pass, zero deviations |
| Real order submitted | NO | YES | Order ID from Binance |
| Real fill received | NO | YES | Fill quantity, price, commission |
| Real fees parsed | NO | YES | Non-zero fee fields |
| Persistence gap | 50% (12/24) | 0% gap | All events persisted |
| PO checks executed | 2/9 | 9/9 | Full protocol |
| Type field | paper_order | venue_market_order | Correct type |
| Status field | submitted | accepted/filled | Correct lifecycle |
| Observability endpoints | Not available | 5 endpoints exercised | All return data |
| Infrastructure friction | 11 min, 5 issues | Zero undocumented issues | Setup guide followed |

**Exit criteria:** All 9 PO checks executed. Session comparison matrix filled with evidence. All observability endpoints exercised. Findings documented.

### Block 3: GO/NO-GO Decision Revisited for Spot Scope Expansion (S460)

Formal re-evaluation of the S451 decision using second session evidence.

**Decision criteria (from S451):**

The stabilization wave is complete when ALL of:
1. At least one real order submitted and filled on Binance Spot mainnet.
2. Fill response parsed and persisted to ClickHouse.
3. Fee/commission fields populated from real venue data.
4. Persistence completeness verified (no gaps).
5. Full post-session protocol executed (PO-1 through PO-9).
6. Infrastructure setup guide documented.
7. Pre and post session backups executed.

**Decision options:**

| Option | Condition |
|--------|-----------|
| Spot Scope Expansion AUTHORIZED | All 7 criteria met with concrete evidence |
| Additional Stabilization Required | Some criteria met, bounded gaps remain |
| Live Safety Closure | Safety incident or fundamental issue discovered |

**Exit criteria:** Decision rendered with evidence. Next macro-front identified. Residual gaps documented.

## Out of Scope (Frozen)

| ID | Exclusion | Rationale |
|----|-----------|-----------|
| NG-1 | Futures live trading | Spot-first; requires separate ceremony. |
| NG-2 | New symbols (beyond BTCUSDT) | Single-symbol scope frozen. |
| NG-3 | Multi-exchange support | Binance-only. |
| NG-4 | Limit orders, amendments, cancel path | Market-order-only lifecycle frozen. |
| NG-5 | Order sizing beyond minimum | Minimum quantity mandatory. |
| NG-6 | Multiple orders per session | Single order per ceremony. |
| NG-7 | Withdrawal-capable API keys | Trade-only is a hard constraint. |
| NG-8 | Automated or unmonitored trading | Human-in-the-loop mandatory. |
| NG-9 | OMS expansion | Order lifecycle model frozen. |
| NG-10 | Dashboard, UI, or alerting development | Not part of stabilization. |
| NG-11 | Config or compose surface changes | Canonical surfaces preserved. |
| NG-12 | Runtime or actor architecture redesign | Architecture proven and stable. |
| NG-13 | OTEL tracing or advanced observability | Deferred. |
| NG-14 | Portfolio risk or PnL calculation | Out of scope for execution engine. |
| NG-15 | Credential hot-swap without restart | Accepted limitation. |
| NG-16 | Per-segment kill-switch | Global kill-switch sufficient. |
| NG-17 | Scope expansion without evidence gate | Scope locked to minimum. |
| NG-18 | Broader sizing justification | This wave is same-scope stabilization. |

## Governing Questions

These questions must be answered with evidence by the wave closure (S460).

### Session Execution

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-1 | Did all 7 pre-session checks pass without deviation? | Yes -- zero deviations (correcting S449) | S458 |
| GQ-2 | Was a real order submitted to Binance Spot mainnet? | Yes -- with order ID | S458 |
| GQ-3 | Was the order accepted by the exchange? | Yes -- or reject with documented reason | S458 |
| GQ-4 | Was a real fill received and parsed? | Yes -- quantity, price, commission | S458 |
| GQ-5 | Was the operator present throughout? | Yes | S458 |
| GQ-6 | Were any stop conditions triggered? | No (or documented if yes) | S458 |

### Post-Session Verification

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-7 | Were all 9 PO checks executed? | Yes -- PO-1 through PO-9 | S459 |
| GQ-8 | Is the order lifecycle fully persisted in ClickHouse? | Yes -- intent + response + fill | S459 |
| GQ-9 | Are fee/commission fields populated from real data? | Yes -- non-zero values | S459 |
| GQ-10 | Is NATS KV state consistent with ClickHouse? | Yes -- verified by explain endpoint | S459 |
| GQ-11 | Was persistence completeness verified (no gap)? | Yes -- 100% events persisted | S459 |

### Observability Comparison

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-12 | Do the new observability endpoints return data for the second session? | Yes -- all 5 endpoints | S459 |
| GQ-13 | Does the session comparison matrix show improvement over S449? | Yes -- all target dimensions met | S459 |
| GQ-14 | Does the explain endpoint confirm cross-surface consistency? | Yes -- no divergence | S459 |

### Decision Gate

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-15 | Are all 7 S451 stabilization criteria met? | Yes -- with concrete evidence | S460 |
| GQ-16 | Is Spot Scope Expansion now justified? | Decision rendered with evidence | S460 |

## Dependency Chain

```
S457 (charter) --> S458 (second supervised live session)
                       |
                       +--> S459 (post-session operational review)
                               |
                               +--> S460 (GO/NO-GO decision revisited)
```

All stages are strictly sequential. No parallelism is possible.

## Success Criteria

The wave passes if:

1. All 7 pre-session checks pass without deviation.
2. A single live BTCUSDT market order is submitted at minimum quantity to Binance Spot.
3. The order lifecycle is observed to completion (accept+fill or reject).
4. The operator is present throughout.
5. All 9 PO checks are executed.
6. All S452A--S456A observability endpoints are exercised against session data.
7. The session comparison matrix shows improvement over S449 across all target dimensions.
8. The S460 decision gate renders a verdict on Spot Scope Expansion.

## Rollback Criteria

All rollback criteria from S444 remain in force. The following are emphasized for the second session:

| ID | Trigger | Action | Severity |
|----|---------|--------|----------|
| RC-1 | Pre-session check fails | DO NOT START SESSION. Fix and re-verify all checks. | High |
| RC-2 | Any stop condition (SC-1 through SC-14) during session | IMMEDIATE KILL-SWITCH. End session. Investigate. | Critical |
| RC-3 | Order submitted to wrong venue or wrong symbol | IMMEDIATE KILL-SWITCH. Full audit. | Critical |
| RC-4 | Fill quantity exceeds requested quantity | IMMEDIATE KILL-SWITCH. Investigate. | Critical |
| RC-5 | Credential exposure at any point | IMMEDIATE HALT. Rotate credentials. | Critical |
| RC-6 | Kill-switch fails to respond within SLA | HALT. Investigate control plane. | Critical |
| RC-7 | S449 deviation repeated (backup skip, env provider) | Session is INVALID for stabilization. Re-attempt required. | High |
| RC-8 | Scope inflation beyond charter | HALT WAVE. Charter violation. | Critical |

## Ceremony Rules

1. This wave inherits ALL ceremony rules from S444.
2. No S449 deviation may be repeated. Deviations that recur invalidate the session for stabilization purposes.
3. The observability endpoints (S452A--S456A) must be exercised during post-session review; skipping them degrades the session's evidence value.
4. The session comparison matrix is mandatory -- it is the primary instrument for measuring stabilization progress.
5. The GO/NO-GO decision at S460 must reference the S451 criteria explicitly and evaluate each one against evidence from S458/S459.

## References

- [S451 GO/NO-GO Decision](go-no-go-decision-for-spot-scope-expansion.md)
- [S451 Decision Matrix](post-first-live-session-decision-matrix-risks-and-next-ceremony.md)
- [S449 Execution Record](first-supervised-live-session-execution-record.md)
- [S448 Evidence Gate](live-trading-enablement-evidence-gate.md)
- [S444 Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md)
- [S444 Scope Constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md)
- [S456A Evidence Gate](operational-history-and-explainability-evidence-gate.md)
- [S456A Evidence Matrix](operational-history-and-explainability-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [S452A Charter](operational-history-and-explainability-wave-charter-and-scope-freeze.md)
