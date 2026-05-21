# Live Trading Enablement Ceremony -- Charter and Scope Freeze

## Wave Identity

| Field | Value |
|---|---|
| Wave | Live Trading Enablement Ceremony |
| Phase | 51 |
| Charter Stage | S444 |
| Planned Stages | S445--S448 |
| Predecessor Wave | Live Trading Authorization (S438--S443) |
| Predecessor Verdict | AUTHORIZED -- CONDITIONAL FOR FUTURE LIVE TRADING CEREMONY |
| Date Opened | 2026-03-24 |

## Strategic Context

The Live Trading Authorization Wave (S438--S443) closed with verdict **AUTHORIZED -- CONDITIONAL FOR FUTURE LIVE TRADING CEREMONY**. Five of six authorization conditions were closed with concrete evidence. Condition C-6 (removal of `dry_run=false` config rejection for mainnet) was explicitly deferred to a dedicated enablement ceremony.

This wave IS the dedicated enablement ceremony. It transforms the conditional authorization into a controlled, supervised, reversible live trading session under the minimum authorized scope.

### What This Wave Is

This is a **ceremony of enablement** -- a controlled operational event with:
- Explicit pre-conditions that must be verified before the session begins
- A single, atomic source-code change (C-6)
- A supervised live trading session with human operator present
- Post-session verification and evidence collection
- An evidence gate that evaluates whether the session succeeded

### What This Wave Is NOT

This is NOT scope expansion. It does not introduce new capabilities, new exchanges, new segments, new order types, or new runtime behaviors. The system is architecturally complete. This wave only removes the last administrative guard (`dry_run=false` rejection) and proves the result under supervision.

## Wave Objective

1. Execute condition C-6: remove the `dry_run=false` config rejection in `schema.go:517-524` under controlled conditions.
2. Create a production config profile for the minimum authorized scope.
3. Conduct mandatory pre-session operational verification (kill-switch cycle, backup, credential mount, config audit).
4. Execute a single supervised live trading session: one BTCUSDT market order at minimum exchange quantity on Binance Spot.
5. Verify the full order lifecycle (submit, accept, fill) with persistence evidence.
6. Conduct post-session operational verification (backup, halt, evidence collection).
7. Close with an evidence gate that renders a final verdict on live trading enablement.

## Scope Freeze

### Minimum Authorized Scope (Inherited from S443)

| Dimension | Authorized Value | Source |
|-----------|-----------------|--------|
| Exchange | Binance only | S443 verdict |
| Segment | Spot only | S443 verdict |
| Symbol | BTCUSDT | S443 verdict |
| Order size | Minimum exchange quantity | S443 verdict |
| Order type | Market order only | S443 verdict |
| Credentials | Trade-only API key (no withdrawal) | S443 verdict |
| Kill-switch | Must be tested (cycle) before session | S443 verdict |
| Backup | Automated backup before and after session | S443 verdict |
| Monitoring | Operator must monitor throughout session | S443 verdict |
| Duration | Operator-controlled; no minimum commitment | S443 verdict |

This scope is **not expandable** within this wave. Any expansion requires a new authorization ceremony.

### Ceremony Blocks

The ceremony is organized into four sequential blocks:

#### Block 1: C-6 Controlled Execution (S445)

Execute the deferred condition C-6 under version control.

- Remove the `dry_run=false` config rejection in `schema.go:517-524`.
- The removal must be a single, isolated, reviewable commit with no other behavioral changes.
- Create `deploy/configs/execute-mainnet-live.jsonc` for minimum authorized scope:
  - `dry_run: false`
  - Binance Spot adapter only
  - Symbol: BTCUSDT
  - Minimum exchange quantity
  - Credential provider: file
  - Kill-switch: active (default)
- Verify that all existing tests pass after the removal (zero regressions).
- Verify that the DryRunSubmitter, SafetyGate, and kill-switch remain fully intact for all other config profiles.
- Document the exact diff and its safety analysis.

**Exit criteria:** C-6 executed. Config created. Zero regressions. All safety invariants intact except SI-1 (intentionally modified).

#### Block 2: Supervised Live Session Proof (S446)

Conduct the first live trading session under full supervision.

**Mandatory pre-session checklist (all must pass before session starts):**

| # | Check | Command/Method | Pass Criteria |
|---|-------|---------------|---------------|
| PS-1 | Kill-switch cycle test | `kill-switch-ops.sh cycle` | Halt and resume both succeed within SLA |
| PS-2 | Automated backup | `clickhouse-scheduled-backup.sh` | Backup completes, off-host replication succeeds |
| PS-3 | Credential mount verification | Verify file-based credentials exist at configured path | Files present, non-empty, correct segment |
| PS-4 | Config audit | Read `execute-mainnet-live.jsonc` | Exactly 1 symbol, minimum size, Spot only, dry_run=false |
| PS-5 | API key permission check | Operator confirms trade-only (no withdrawal) in Binance console | Screenshot or operator attestation |
| PS-6 | Kill-switch initial state | `kill-switch-ops.sh status` | Gate is `active` |
| PS-7 | System boot with live config | Start execute binary with live config | Boot succeeds, preflight passes, no errors |

**Session execution:**

1. Operator confirms all pre-session checks passed.
2. System generates execution intent from live market data for BTCUSDT.
3. VenueAdapterActor submits market order to Binance Spot via real adapter.
4. Order lifecycle proceeds: submit -> accept -> fill (or submit -> reject).
5. Operator monitors throughout. Any anomaly triggers immediate kill-switch.
6. After first successful fill (or after reject with documented reason): session ends.

**Session scope:** exactly ONE order. The session ends after the first order lifecycle completes.

**Exit criteria:** One live order submitted. Order lifecycle observed (accept+fill or reject). Operator present throughout. No stop conditions triggered.

#### Block 3: Post-Session Operational Verification (S447)

Verify system state and collect evidence after the live session.

- Trigger kill-switch halt immediately after session completion.
- Execute automated backup (post-session).
- Verify ClickHouse persistence:
  - ExecutionIntent record exists with correct symbol, side, quantity.
  - Venue response record exists with order ID, status, fill details.
  - Timestamps are consistent with session window.
- Verify NATS KV state:
  - Order lifecycle state reflects final venue response.
  - Kill-switch gate is halted (post-session).
- Collect evidence artifacts:
  - Order receipt from Binance (order ID, status, fills).
  - ClickHouse query results showing persisted lifecycle.
  - Kill-switch state before/during/after session.
  - System logs covering session window.
- Document any anomalies, latencies, or unexpected behaviors.

**Exit criteria:** Post-session backup complete. ClickHouse persistence verified. NATS state verified. Evidence artifacts collected and documented.

#### Block 4: Evidence Gate Final (S448)

Evaluate the complete ceremony and render the final verdict.

- Score each block's exit criteria with evidence.
- Verify zero regressions across the full test suite.
- Evaluate the live order lifecycle evidence:
  - Was the order submitted to the correct venue?
  - Was the order accepted by the exchange?
  - Was the fill received and persisted?
  - Did the lifecycle match the canonical model?
- Evaluate operational controls:
  - Did pre-session checks all pass?
  - Was the operator present throughout?
  - Did post-session verification complete?
  - Were stop conditions respected?
- Render verdict:
  - **LIVE TRADING ENABLED** -- ceremony completed successfully, live trading proven under minimum scope.
  - **CEREMONY INCOMPLETE** -- with explicit remediation path for re-attempt.
- Document residual gaps and limitations.
- State next-ceremony direction (Futures enablement, scope expansion, etc.).

**Exit criteria:** Evidence matrix produced. All blocks evaluated. Verdict rendered. Next-ceremony direction stated.

### Out of Scope (Frozen)

| Exclusion | Rationale |
|---|---|
| Futures live trading | Spot-first; requires separate ceremony after Spot is proven. NG-1. |
| Multi-symbol trading | Single-symbol scope is frozen. NG-2. |
| Multi-exchange support | Binance-only. NG-3. |
| Limit orders, amendments, cancel API | Market-order-only lifecycle is frozen. NG-4. |
| Advanced order types | Out of ceremony scope. NG-5. |
| Dashboard, UI, or alerting development | Not part of enablement. NG-6. |
| Config or compose surface re-expansion | Canonical surfaces preserved. NG-7. |
| Portfolio risk management or PnL tracking | Out of scope. NG-8. |
| Credential hot-swap or rotation without restart | Accepted limitation. NG-9. |
| Per-segment kill-switch | Global kill-switch sufficient for Spot-only. NG-10. |
| OTEL tracing or advanced observability | Deferred. NG-11. |
| OMS expansion | Order lifecycle model is frozen. NG-12. |
| Runtime, adapter, or actor architecture redesign | Architecture is proven and stable. NG-13. |
| Automated or unmonitored trading sessions | Human-in-the-loop is mandatory. NG-14. |
| Scope expansion without new evidence gate | Scope is locked to minimum authorized surface. NG-15. |
| Multiple orders per session | Single order per ceremony. NG-16. |
| Sizing beyond minimum exchange quantity | Minimum quantity is mandatory. NG-17. |
| Withdrawal-capable API keys | Trade-only is mandatory. NG-18. |

## Governing Questions

These questions must be answered with evidence by the evidence gate (S448).

### C-6 Execution

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-1 | Has the `dry_run=false` rejection been removed from schema.go? | Yes -- single commit, isolated diff | S445 |
| GQ-2 | Does the live config specify exactly the minimum authorized scope? | Yes -- 1 symbol, min size, Spot only | S445 |
| GQ-3 | Do all existing tests pass after the removal? | Yes -- zero regressions | S445 |
| GQ-4 | Are all safety invariants (except SI-1) intact? | Yes -- DryRunSubmitter, SafetyGate, kill-switch unchanged | S445 |

### Pre-Session Verification

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-5 | Did the kill-switch cycle test pass? | Yes -- halt and resume within SLA | S446 |
| GQ-6 | Was an automated backup completed before session? | Yes -- with off-host replication | S446 |
| GQ-7 | Were credentials mounted via file provider? | Yes -- verified at configured path | S446 |
| GQ-8 | Did the operator confirm trade-only API key permissions? | Yes -- documented | S446 |

### Live Session

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-9 | Was a live order submitted to Binance Spot? | Yes -- BTCUSDT market order | S446 |
| GQ-10 | Was the order accepted by the exchange? | Yes -- with order ID | S446 |
| GQ-11 | Was a fill received? | Yes -- or reject with documented reason | S446 |
| GQ-12 | Was the operator present throughout the session? | Yes | S446 |
| GQ-13 | Were any stop conditions triggered? | No (or documented if yes) | S446 |

### Post-Session Verification

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-14 | Was the kill-switch activated after session? | Yes -- system halted | S447 |
| GQ-15 | Was a post-session backup completed? | Yes -- with off-host replication | S447 |
| GQ-16 | Is the order lifecycle persisted in ClickHouse? | Yes -- intent + response records | S447 |
| GQ-17 | Is NATS KV state consistent with the final venue response? | Yes | S447 |

### Evidence Gate

| ID | Question | Expected Answer | Stage |
|----|----------|-----------------|-------|
| GQ-18 | Are all 17 governing questions answered with evidence? | Yes | S448 |
| GQ-19 | Were zero regressions introduced? | Yes | S448 |
| GQ-20 | Did the live order lifecycle match the canonical model? | Yes | S448 |

## Dependency Chain

```
S444 (charter) --> S445 (C-6 execution)
                       |
                       +--> S446 (supervised live session)
                               |
                               +--> S447 (post-session verification)
                                       |
                                       +--> S448 (evidence gate)
```

All stages are strictly sequential. No parallelism is possible -- each block depends on the output of the previous block.

## Success Criteria

The ceremony passes if:

1. C-6 is executed with zero regressions and all safety invariants (except SI-1) intact.
2. A live config is created for exactly the minimum authorized scope.
3. All pre-session checks pass.
4. A single live BTCUSDT market order is submitted at minimum quantity.
5. The order lifecycle is observed to completion (accept+fill or reject).
6. The operator is present throughout.
7. Post-session verification confirms persistence and state consistency.
8. The evidence gate renders **LIVE TRADING ENABLED**.

## Rollback Criteria

| ID | Trigger | Action | Severity |
|----|---------|--------|----------|
| RC-1 | Test regression after C-6 removal | HALT. Revert commit. Investigate. | High |
| RC-2 | Safety invariant broken (SI-2 through SI-12) | HALT. Revert commit. Full audit. | Critical |
| RC-3 | Pre-session check fails | DO NOT START SESSION. Fix and re-verify. | High |
| RC-4 | Any stop condition (SC-1 through SC-9) during session | IMMEDIATE KILL-SWITCH. End session. Investigate. | Critical |
| RC-5 | Order submitted to wrong venue or wrong symbol | IMMEDIATE KILL-SWITCH. Full audit. May require re-authorization. | Critical |
| RC-6 | Fill quantity exceeds requested quantity | IMMEDIATE KILL-SWITCH. Investigate venue behavior. | Critical |
| RC-7 | Credential exposure at any point | IMMEDIATE HALT. Rotate credentials. Audit. | Critical |
| RC-8 | Kill-switch fails to respond within SLA | HALT. Investigate control plane. | Critical |
| RC-9 | Scope inflation beyond charter | HALT WAVE. Charter violation. | Critical |
| RC-10 | Post-session persistence verification fails | Document discrepancy. May require re-attempt. | High |

## Ceremony Rules

- This is a single-attempt ceremony. If the live session succeeds, the ceremony closes. If it fails, the failure is documented and a re-attempt requires explicit operator decision.
- No stage may expand beyond its block definition.
- The live config must specify exactly the minimum authorized scope -- no more.
- The operator must be present and attentive for the entire live session (Block 2).
- Any uncertainty triggers the kill-switch. There is no penalty for false positives.
- The kill-switch is the primary safety mechanism. It must be tested before the session and available throughout.
- After the ceremony, `dry_run=false` remains a valid configuration option for the minimum authorized scope. Expansion requires a new ceremony.

## References

- [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md) (S443)
- [Live Trading Authorization Evidence Matrix](live-trading-authorization-evidence-matrix-blockers-conditions-and-next-ceremony.md) (S443)
- [Live Trading Authorization Wave Charter](live-trading-authorization-wave-charter-and-scope-freeze.md) (S438)
- [Live Trading Authorization Capabilities, Questions, Non-Goals](live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md) (S438)
- [Kill-Switch Operational Runbook](kill-switch-operational-runbook.md) (S442)
- [Kill-Switch Procedures](kill-switch-trigger-verification-rollback-and-recovery-procedure.md) (S442)
- [Mainnet Authorization Evidence Gate](mainnet-authorization-evidence-gate.md) (S437)
