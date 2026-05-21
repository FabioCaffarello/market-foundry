# Live Trading Authorization -- Capabilities, Questions, Non-Goals, and Rollback Criteria

> Authority: S438 | Date: 2026-03-24 | Phase: 50 (Live Trading Authorization)

## Purpose

This document defines the governing questions, chartered capabilities, explicit non-goals, rollback criteria, and stop conditions for the Live Trading Authorization Wave (S438--S443). It is the operational companion to the [wave charter](live-trading-authorization-wave-charter-and-scope-freeze.md).

## Governing Questions

These questions must be answered with evidence by the evidence gate (S443). A question answered "No" or "Insufficient" blocks authorization.

### Secret Manager and Credential Lifecycle

| ID | Question | Expected Answer | Assigned Stage |
|----|----------|-----------------|----------------|
| GQ-1 | Is an external secret manager deployed and operational? | Yes -- with proof of credential retrieval | S439 |
| GQ-2 | Do mainnet adapters retrieve credentials exclusively from the external provider? | Yes -- env-var path disabled for mainnet | S439 |
| GQ-3 | Does the system fail-closed if the secret manager is unreachable? | Yes -- adapter refuses to boot | S439 |
| GQ-4 | Is the credential scope limited to trade-only (no withdrawal)? | Yes -- API key permissions documented | S439 |

### Backup and Data Protection

| ID | Question | Expected Answer | Assigned Stage |
|----|----------|-----------------|----------------|
| GQ-5 | Is ClickHouse backup automated on a schedule? | Yes -- cron or equivalent with proof of execution | S440 |
| GQ-6 | Are backup artifacts replicated to off-host storage? | Yes -- with proof of transfer and integrity check | S440 |
| GQ-7 | Can the system be restored from an off-host backup? | Yes -- restore test from off-host with data verification | S440 |
| GQ-8 | Is RTO/RPO documented for the automated procedure? | Yes -- updated from S435 baseline | S440 |

### Authenticated Connectivity and Sustained Operation

| ID | Question | Expected Answer | Assigned Stage |
|----|----------|-----------------|----------------|
| GQ-9 | Has an authenticated mainnet API call succeeded for Spot? | Yes -- read-only endpoint, valid response | S441 |
| GQ-10 | Has an authenticated mainnet API call succeeded for Futures? | Yes -- read-only endpoint, valid response | S441 |
| GQ-11 | Are credentials correctly scoped per segment? | Yes -- Spot key accesses Spot, Futures key accesses Futures | S441 |
| GQ-12 | Has a sustained soak test (>=4h) been completed? | Yes -- with uptime, reconnection, and resource metrics | S441 |
| GQ-13 | Were zero real orders placed during the soak? | Yes -- DryRunSubmitter active throughout | S441 |
| GQ-14 | Did the soak reveal any data loss or state corruption? | No -- KV and ClickHouse consistent throughout | S441 |

### Kill-Switch Operational Readiness

| ID | Question | Expected Answer | Assigned Stage |
|----|----------|-----------------|----------------|
| GQ-15 | Is a kill-switch runbook documented? | Yes -- canonical procedure with roles and SLAs | S442 |
| GQ-16 | Has the kill switch been tested under operational conditions? | Yes -- during soak, with halt/verify/resume cycle | S442 |
| GQ-17 | Does the kill switch halt execution within the documented SLA? | Yes -- with timestamped evidence | S442 |
| GQ-18 | Does the system recover cleanly after kill-switch resumption? | Yes -- no orphaned state, no duplicate intents | S442 |

### Authorization Gate

| ID | Question | Expected Answer | Assigned Stage |
|----|----------|-----------------|----------------|
| GQ-19 | Are all six conditions (C-1 through C-6) satisfied? | Yes -- each with individual evidence | S443 |
| GQ-20 | Have zero regressions been introduced? | Yes -- full test suite passes | S443 |
| GQ-21 | Are rollback criteria and stop conditions documented? | Yes -- in charter and evidence gate | S443 |
| GQ-22 | Is the minimum authorized scope explicitly defined and bounded? | Yes -- 1 exchange, 1 segment, 1 symbol, minimum size | S443 |

## Chartered Capabilities

### S439 -- External Secret Manager Deployment

| ID | Capability | Evidence Required |
|----|-----------|-------------------|
| C-SM-1 | External secret manager provider implementation | Concrete `CredentialProvider` backed by Vault/AWS SM/equivalent |
| C-SM-2 | Mainnet adapter credential wiring | Both Spot and Futures mainnet adapters retrieve from external provider |
| C-SM-3 | Fail-closed on unreachable secret manager | Adapter exit(1) on credential retrieval failure |
| C-SM-4 | Credential scope documentation | API key permission model documented (trade-only, no withdrawal) |

### S440 -- Automated Backup with Off-Host Replication

| ID | Capability | Evidence Required |
|----|-----------|-------------------|
| C-BK-1 | Automated backup schedule | Cron/equivalent running on configured interval with proof of execution |
| C-BK-2 | Off-host replication | Backup artifacts transferred to remote storage with integrity verification |
| C-BK-3 | Restore from off-host proof | Clean restore from remote backup with row count and schema verification |
| C-BK-4 | Updated RTO/RPO documentation | Revised estimates reflecting automated + off-host procedure |

### S441 -- Authenticated Mainnet API Proof and Sustained Soak

| ID | Capability | Evidence Required |
|----|-----------|-------------------|
| C-AP-1 | Spot authenticated API call | `GET /api/v3/account` success with valid response (redacted credentials) |
| C-AP-2 | Futures authenticated API call | `GET /fapi/v2/account` success with valid response (redacted credentials) |
| C-AP-3 | Credential scoping verification | Each segment uses its own credential set; cross-segment access denied or isolated |
| C-AP-4 | Sustained soak execution (>=4h) | Continuous operation with real mainnet data, dry_run=true |
| C-AP-5 | Soak metrics documentation | Uptime, reconnections, data gaps, intent rate, memory/CPU usage |
| C-AP-6 | Zero real orders during soak | DryRunSubmitter interception verified throughout |

### S442 -- Kill-Switch Operational Runbook and Test

| ID | Capability | Evidence Required |
|----|-----------|-------------------|
| C-KS-1 | Canonical kill-switch runbook | Document with halt/verify/resume/authorization procedures |
| C-KS-2 | Live kill-switch test | Trigger during soak; halt verified with timestamps |
| C-KS-3 | Halt SLA verification | Execution stops within documented time bound |
| C-KS-4 | Clean recovery verification | Resume after halt with no orphaned state or duplicate intents |

### S443 -- Live Trading Authorization Evidence Gate

| ID | Capability | Evidence Required |
|----|-----------|-------------------|
| C-EG-1 | Condition evaluation matrix | All six conditions (C-1 through C-6) individually scored |
| C-EG-2 | Regression verification | Full test suite pass, both binaries compile |
| C-EG-3 | Residual gap profile | No new medium+ gaps |
| C-EG-4 | Authorization verdict | AUTHORIZED or NOT AUTHORIZED with explicit rationale |
| C-EG-5 | Minimum scope documentation | Authorized scope boundaries formally stated |
| C-EG-6 | Rollback and stop conditions | Final version in evidence gate document |

## Non-Goals

These items are explicitly excluded from the wave. Any attempt to include them constitutes scope inflation and must be rejected.

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-1 | Enable live trading during the wave | Wave authorizes; go-live is a separate operator decision |
| NG-2 | Authorize Futures live trading | Spot-first; Futures authorization requires its own evidence after Spot is proven in production |
| NG-3 | Authorize multi-symbol trading | Single-symbol scope; expansion requires separate ceremony |
| NG-4 | Support multi-exchange | Binance-only; adding exchanges is a structural change |
| NG-5 | Expand OMS (limit orders, amendments, cancel API) | Market-order-only lifecycle is frozen |
| NG-6 | Implement advanced order types | Out of wave scope |
| NG-7 | Build dashboards, UI, or alerting rules | Operational signals remain HTTP/JSON-based |
| NG-8 | Re-expand config or compose surfaces | Canonical 3+3 surface preserved |
| NG-9 | Implement portfolio risk management | Out of scope for execution engine |
| NG-10 | Implement credential hot-swap without restart | RG-21 accepted; restart-based rotation sufficient for minimum scope |
| NG-11 | Resolve non-blockers NB-1 through NB-10 | Deferred to post-authorization hardening |
| NG-12 | Restructure documentation or governance | Separate concern |
| NG-13 | Implement per-segment kill switch | Global kill switch sufficient for single-segment scope |
| NG-14 | Add OTEL tracing or advanced observability | Deferred |
| NG-15 | Redesign runtime, adapter, or actor architecture | Architecture is proven and stable |
| NG-16 | Implement position tracking or PnL calculation | Out of scope |
| NG-17 | Implement fee optimization or rebate tracking | Fee model is sufficient as-is |
| NG-18 | Deploy to cloud infrastructure | Deployment topology is operator's choice; wave proves local/on-prem authorization |

## Rollback Criteria

### Wave-Level Rollback

These conditions halt the wave and require re-evaluation before resuming:

| ID | Trigger | Action | Severity |
|----|---------|--------|----------|
| RC-1 | Real order placed on mainnet during any wave stage | IMMEDIATE HALT. Full 5-layer defense audit. Wave cannot resume until root cause identified and fixed. | Critical |
| RC-2 | Secret manager deployment introduces persistent boot failure | Revert to env-var provider. Re-evaluate deployment strategy. Resume only after fix is proven. | High |
| RC-3 | Automated backup causes ClickHouse performance degradation >10% | Revert to manual backup. Adjust schedule or method. | Medium |
| RC-4 | Soak test reveals data loss, state corruption, or unrecoverable errors | HALT soak. Investigate root cause. Do not proceed to evidence gate. | High |
| RC-5 | Kill-switch test fails to halt execution within SLA | Investigate control plane reliability. Fix before evidence gate. | High |
| RC-6 | Any test regression introduced by wave changes | Fix regression before proceeding. Zero-regression policy is inviolable. | Medium |
| RC-7 | Credential leak or exposure during any stage | IMMEDIATE HALT. Rotate all affected credentials. Audit access logs. | Critical |

### Post-Authorization Stop Conditions

These conditions apply to any live trading session authorized by the evidence gate. They must be documented in S443 and accepted by the operator before enabling `dry_run=false`:

| ID | Condition | Required Action | Resumption Criteria |
|----|-----------|----------------|---------------------|
| SC-1 | Kill switch triggered | All execution halts. No new orders. | Operator investigation complete. Root cause documented. Explicit resume decision. |
| SC-2 | Credential expiration or rotation failure | Halt execution. | Credentials re-verified via external secret manager. |
| SC-3 | ClickHouse write failure | Halt execution. | Audit trail integrity restored. Backup verified. |
| SC-4 | NATS connectivity loss >30s | Halt execution. | Control plane health verified. |
| SC-5 | Unexpected order state | IMMEDIATE HALT. | Full state audit. Root cause identified. May require re-authorization. |
| SC-6 | Exchange API error rate >10% for >5 minutes | Halt execution. | Exchange health verified. |
| SC-7 | Operator decision | Halt at any time. | Operator decides. No minimum commitment. |
| SC-8 | Resource exhaustion (memory, disk, CPU) | Halt execution. | Resources restored. Root cause addressed. |
| SC-9 | Unexplained fill price deviation >5% from market | HALT. Investigate venue behavior. | Market conditions verified. Adapter behavior audited. |

## Condition-to-Stage Mapping

| Condition | Description | Stage | Capabilities | Governing Questions |
|-----------|-------------|-------|-------------|---------------------|
| C-1 | Authenticated mainnet API call | S441 | C-AP-1, C-AP-2 | GQ-9, GQ-10 |
| C-2 | External secret manager deployed | S439 | C-SM-1 to C-SM-4 | GQ-1 to GQ-4 |
| C-3 | Automated backup + off-host replication | S440 | C-BK-1 to C-BK-4 | GQ-5 to GQ-8 |
| C-4 | Sustained mainnet soak | S441 | C-AP-4 to C-AP-6 | GQ-12 to GQ-14 |
| C-5 | Kill-switch runbook tested | S442 | C-KS-1 to C-KS-4 | GQ-15 to GQ-18 |
| C-6 | dry_run=false rejection removal documented | S443 | C-EG-1 | GQ-19 |

## Stage Ordering and Rationale

| Order | Stage | Block | Dependencies | Rationale |
|-------|-------|-------|-------------|-----------|
| 1 | S438 | Charter | None | Scope freeze must precede execution |
| 2 | S439 | Secret Manager | S438 | Credentials must be available before authenticated API calls |
| 3 | S440 | Backup Automation | S438 | Can overlap with S439; no credential dependency |
| 4 | S441 | API Proof + Soak | S439 | Requires credentials from external provider |
| 5 | S442 | Kill-Switch | S441 | Test occurs during or immediately after soak |
| 6 | S443 | Evidence Gate | S439-S442 | Evaluates complete wave |

S439 and S440 can execute in parallel if capacity allows, but are sequenced for focus. S441 has a hard dependency on S439 (needs external credentials). S442 has a soft dependency on S441 (kill-switch test benefits from soak infrastructure).

## Links

- Wave charter: [live-trading-authorization-wave-charter-and-scope-freeze.md](live-trading-authorization-wave-charter-and-scope-freeze.md)
- Predecessor evidence gate: [mainnet-authorization-evidence-gate.md](mainnet-authorization-evidence-gate.md)
- Predecessor evidence matrix: [mainnet-authorization-evidence-matrix-blockers-residual-gaps-and-next-ceremony.md](mainnet-authorization-evidence-matrix-blockers-residual-gaps-and-next-ceremony.md)
- Predecessor charter: [mainnet-enablement-wave-charter-and-scope-freeze.md](mainnet-enablement-wave-charter-and-scope-freeze.md)
