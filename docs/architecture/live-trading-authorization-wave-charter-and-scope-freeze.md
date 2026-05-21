# Live Trading Authorization Wave -- Charter and Scope Freeze

## Wave Identity

| Field | Value |
|---|---|
| Wave | Live Trading Authorization |
| Phase | 50 |
| Charter Stage | S438 |
| Planned Stages | S439--S443 |
| Predecessor Wave | Mainnet Enablement (S432--S437) |
| Predecessor Verdict | AUTHORIZED -- CONDITIONAL (17/17 FULL, 0 regressions, 6 conditions) |
| Date Opened | 2026-03-24 |

## Strategic Context

The Foundry has completed 14 consecutive wave passes since S370 with zero regressions. The Mainnet Enablement Wave (S432--S437) resolved all three structural mainnet blockers, proved dry-run execution against real Binance mainnet endpoints, and closed with verdict **AUTHORIZED -- CONDITIONAL**.

The conditional authorization identified exactly six conditions (C-1 through C-6) that must be satisfied before `dry_run=false` can be enabled on mainnet adapters. These six conditions map directly to the six LOW gaps introduced in the Mainnet Enablement Wave:

| Condition | Gap | Description |
|-----------|-----|-------------|
| C-1 | RG-24 | No authenticated mainnet API call proven |
| C-2 | RG-20 | Mainnet credentials via env vars (no external secret manager deployed) |
| C-3 | RG-22/RG-23 | ClickHouse backup manual and same-host only |
| C-4 | RG-25 | No sustained mainnet soak test |
| C-5 | -- | Kill-switch operational procedure not documented/tested |
| C-6 | -- | `dry_run=false` config rejection still active in `schema.go` |

This wave exists to resolve these six conditions and close with a formal **Live Trading Authorization Evidence Gate**. The evidence gate will render one of two verdicts:

- **AUTHORIZED FOR LIVE TRADING** -- all conditions met, `dry_run=false` may be enabled for the minimum authorized scope.
- **NOT AUTHORIZED** -- one or more conditions unmet, with explicit remediation path.

### What This Wave Is NOT

This wave is NOT a go-live event. It is a **ceremony of authorization**. The difference:

- **Authorization** = proving that the system meets all preconditions for live trading and that operational controls are in place to contain risk.
- **Go-live** = actually enabling live trading in production.

Authorization is a prerequisite for go-live, but go-live is a separate operational decision that requires explicit human sign-off after the evidence gate passes.

## Wave Objective

1. Deploy an external secret manager for mainnet credential lifecycle, replacing env-var credential passing for mainnet adapters.
2. Automate ClickHouse backup with off-host replication, closing the manual/same-host gaps.
3. Prove an authenticated mainnet API call (account info or similar read-only endpoint) for both Spot and Futures, then execute a sustained soak test with real mainnet connectivity.
4. Document and test a kill-switch operational runbook that demonstrates controlled halt of execution under real conditions.
5. Close with an evidence gate that evaluates all six conditions and renders a formal live-trading authorization verdict.

## Scope Freeze

### In Scope (Frozen)

The wave is organized into five execution blocks:

#### Block 1: External Secret Manager Deployment (S439)

Resolve C-2. Deploy an external secret manager and wire mainnet adapters to use it.

- Implement a concrete `CredentialProvider` backed by an external secret manager (HashiCorp Vault, AWS Secrets Manager, or equivalent).
- Mainnet adapters must resolve credentials via the external provider, not env vars.
- Fail-closed behavior: adapter refuses to boot if secret manager is unreachable or returns empty credentials.
- Prove credential retrieval for both Spot and Futures mainnet adapters.
- Env-var provider remains available for testnet/development use only.
- Document the deployment procedure and access control model.

**Exit Criteria**: External secret manager deployed. Mainnet adapters wired to retrieve credentials from it. Fail-closed behavior proven. RG-20 closed.

#### Block 2: Automated Backup with Off-Host Replication (S440)

Resolve C-3. Automate ClickHouse backup and add off-host replication.

- Implement automated backup on a configurable schedule (cron or equivalent).
- Backup artifacts must be replicated to off-host storage (remote filesystem, S3-compatible, or equivalent).
- Prove automated backup execution with at least one successful off-host replication cycle.
- Prove restore from off-host backup to a clean ClickHouse instance.
- Update RTO/RPO documentation to reflect the automated procedure.

**Exit Criteria**: Backup automated on schedule. Off-host replication proven. Restore from off-host backup proven. RG-22 and RG-23 closed.

#### Block 3: Authenticated Mainnet API Proof and Sustained Soak (S441)

Resolve C-1 and C-4. Prove authenticated mainnet connectivity and sustained operation.

- Execute an authenticated read-only API call (e.g., `GET /api/v3/account` for Spot, `GET /fapi/v2/account` for Futures) against real Binance mainnet using credentials from the external secret manager.
- Confirm successful authentication, valid response, and correct credential scoping.
- Execute a sustained soak test (minimum 4 hours continuous operation) with:
  - Real mainnet WebSocket market data ingestion.
  - Execution intent generation from real price feeds.
  - DryRunSubmitter interception (dry_run=true throughout soak).
  - KV and ClickHouse persistence operating normally.
  - Kill-switch and staleness guards active.
- Document soak results: uptime, reconnections, data gaps, intent generation rate, resource consumption.

**Exit Criteria**: Authenticated mainnet API call proven for both segments. Sustained soak executed and documented. RG-24 and RG-25 closed.

#### Block 4: Kill-Switch Operational Runbook and Test (S442)

Resolve C-5. Document and test the kill-switch procedure under operational conditions.

- Write a canonical kill-switch runbook covering:
  - How to halt all execution (EXECUTION_CONTROL KV).
  - How to halt a single segment.
  - How to verify execution has stopped.
  - How to resume execution after investigation.
  - Who is authorized to trigger the kill switch.
  - Communication protocol during a halt.
- Execute a live kill-switch test during the S441 soak window:
  - Trigger the kill switch while the system is actively generating intents.
  - Verify execution halts within the documented SLA.
  - Verify no new intents or venue calls are generated after the halt.
  - Resume execution and verify recovery.
- Document the test results with timestamps and evidence.

**Exit Criteria**: Runbook documented. Kill-switch test executed during soak. Halt, verification, and recovery proven. C-5 satisfied.

#### Block 5: Live Trading Authorization Evidence Gate (S443)

Evaluate the wave and render the formal authorization verdict.

- Score each block's exit criteria with evidence grades.
- Verify zero regressions across the full test suite.
- Evaluate residual gaps and classify by severity.
- Evaluate all six conditions (C-1 through C-6) individually.
- If all conditions are satisfied:
  - Render **AUTHORIZED FOR LIVE TRADING** for the minimum authorized scope (see below).
  - Document the explicit source change required to remove `dry_run=false` config rejection (C-6).
  - The source change is NOT executed in this stage. It is documented as a deliberate action for the operator to perform after authorization.
- If any condition remains unsatisfied:
  - Render **NOT AUTHORIZED** with explicit remediation path.
- Regardless of verdict, document rollback criteria and stop conditions for any future live trading session.

**Exit Criteria**: Evidence matrix produced. All six conditions evaluated. Verdict rendered. Rollback criteria and stop conditions documented. Next-ceremony direction stated.

### Minimum Authorized Scope

If the evidence gate authorizes live trading, authorization is limited to the following minimum scope:

| Dimension | Authorized Scope | Rationale |
|-----------|-----------------|-----------|
| Exchange | Binance only | Only exchange with proven adapters |
| Segment | Spot only (initial) | Lower risk; margin isolation; simpler failure mode |
| Symbol | BTCUSDT only (or operator-chosen single symbol) | Single symbol eliminates cross-symbol interference |
| Order type | Market order only | Only order type in the current lifecycle model |
| Size | Minimum exchange-allowed quantity | Limits financial exposure to smallest possible unit |
| Mode | `dry_run=false` with kill-switch active | Reversible via kill-switch at any moment |
| Duration | Operator-controlled; no minimum commitment | Operator may halt at any time |
| Credential scope | Trade-only API key (no withdrawal permission) | Limits blast radius of credential compromise |

This scope is **not expandable** without a separate authorization ceremony. Expansion to Futures, multi-symbol, or larger size requires its own evidence gate.

### Out of Scope (Frozen)

| Exclusion | Rationale |
|---|---|
| Live trading enablement in this wave | Authorization only; go-live is an operator decision post-gate. NG-1. |
| Futures live trading authorization | Spot-first; Futures requires separate evidence after Spot is proven. NG-2. |
| Multi-symbol authorization | Single-symbol scope; expansion requires its own ceremony. NG-3. |
| Multi-exchange support | Binance-only scope. NG-4. |
| OMS expansion (limit orders, amendments, cancel API) | Market-order-only lifecycle is frozen. NG-5. |
| Advanced order types | Out of wave scope. NG-6. |
| Dashboard, UI, or alerting rule development | Operational signals remain HTTP/JSON-based. NG-7. |
| Config or compose surface re-expansion | Canonical 3+3 surface preserved. NG-8. |
| Portfolio risk management | Out of scope for execution engine. NG-9. |
| Credential hot-swap / rotation without restart | RG-21 accepted; restart-based rotation is sufficient for minimum scope. NG-10. |
| Non-blocker resolution (NB-1 through NB-10) | Deferred to post-authorization hardening wave. NG-11. |
| Documentation governance or restructuring | Separate concern. NG-12. |
| Per-segment kill switch | Global kill switch is sufficient for single-segment authorization. NG-13. |
| OTEL tracing or advanced observability | Deferred. NG-14. |

## Dependency Chain

```
S438 (charter) --> S439 (external secret manager)
                       |
                       +--> S440 (automated backup + off-host)
                       |
                       +--> S441 (authenticated API + soak)
                               |
                               +--> S442 (kill-switch runbook + test)
                                       |
                                       +--> S443 (evidence gate)
```

S439 must complete before S441 because authenticated API calls require credentials from the external secret manager. S440 can proceed in parallel with S439 but is sequenced before S441 for operational readiness. S441 and S442 are tightly coupled (kill-switch test occurs during soak). S443 evaluates the complete wave.

## Success Criteria

The wave passes if:

1. C-1 is resolved: authenticated mainnet API call proven for both Spot and Futures.
2. C-2 is resolved: external secret manager deployed and wired to mainnet adapters.
3. C-3 is resolved: ClickHouse backup automated with off-host replication.
4. C-4 is resolved: sustained mainnet soak executed and documented.
5. C-5 is resolved: kill-switch runbook documented and tested under operational conditions.
6. C-6 is documented: explicit source change to remove `dry_run=false` config rejection is identified and ready (not executed).
7. No real orders are placed on mainnet at any point during the wave.
8. The evidence gate renders a verdict with zero high-severity or medium-severity residual gaps introduced by this wave.

## Rollback Criteria

If at any point during the wave the following conditions are observed, the wave must halt and the affected stage must be re-evaluated:

| Trigger | Action |
|---------|--------|
| Real order placed on mainnet during any stage | IMMEDIATE HALT. Investigate root cause. Wave cannot resume until 5-layer defense is re-verified. |
| External secret manager introduces boot-time failure rate >5% | Revert to env-var provider. Re-evaluate deployment strategy. |
| Automated backup causes ClickHouse performance degradation >10% | Revert to manual backup. Re-evaluate schedule and method. |
| Soak test reveals data loss, state corruption, or unrecoverable errors | HALT soak. Investigate. Do not proceed to evidence gate until resolved. |
| Kill-switch test fails to halt execution within documented SLA | Investigate control plane reliability. Do not proceed to evidence gate. |
| Any test regression introduced by wave changes | Fix regression before proceeding. Zero-regression streak must be maintained. |

## Stop Conditions for Future Live Trading Sessions

These conditions must be documented in the evidence gate (S443) and apply to any live trading session authorized by this wave:

| Condition | Required Action |
|-----------|----------------|
| Kill switch triggered | All execution halts. No new orders. Investigate before resuming. |
| Credential expiration or rotation failure | Halt execution. Resolve credential state. Resume only after re-verification. |
| ClickHouse write failure | Halt execution. Audit trail integrity is non-negotiable. |
| NATS connectivity loss >30s | Halt execution. Control plane must be reliable. |
| Unexpected order state (fill without submission, unknown venue order ID) | IMMEDIATE HALT. Investigate state integrity. |
| Exchange API error rate >10% sustained over 5 minutes | Halt execution. Evaluate exchange health before resuming. |
| Operator decision | Operator may halt at any time for any reason. No minimum commitment. |

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| Secret manager adds single point of failure | Fail-closed design: adapter refuses to start without credentials. Credential caching at boot reduces runtime dependency. |
| Off-host backup increases operational complexity | Automated script with health checks. Restore proof required before gate. |
| Authenticated API call reveals credential issues | Read-only endpoints first. No order-related calls until evidence gate passes. |
| Soak test discovers latent bugs under sustained load | Soak runs with dry_run=true. No financial risk. Bugs are evidence for the gate. |
| Kill-switch test disrupts ongoing proof work | Kill-switch test is part of the soak window, not a separate disruption. |
| First real order fails or behaves unexpectedly | Minimum scope (1 symbol, minimum size) limits blast radius. Kill switch provides immediate halt. |

## Ceremony Rules

- No stage may expand beyond its block definition without a charter amendment.
- Charter amendments require explicit justification and a documented decision in the stage report.
- The evidence gate (S443) must evaluate against this frozen scope, not against any informally expanded scope.
- All test evidence must be reproducible from the committed codebase.
- No real orders may be placed on mainnet during any stage of this wave (S438--S442). The evidence gate (S443) may authorize future live trading but does not enable it.
- The authorization verdict from S443 is a necessary but not sufficient condition for go-live. The operator must independently decide to enable live trading after reviewing the evidence gate.
- This is the most consequential ceremony in the project's history. Every condition must be satisfied with evidence, not assertion.
