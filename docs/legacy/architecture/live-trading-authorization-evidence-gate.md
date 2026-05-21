# Live Trading Authorization Evidence Gate

Authority: S443. Evaluates the Live Trading Authorization Wave (S438-S442).

Predecessor gate: [Mainnet Authorization Evidence Gate](mainnet-authorization-evidence-gate.md) (S437).

## Purpose

This document renders the formal evidence gate for the Live Trading Authorization
Wave. It determines whether the six conditional authorization criteria established
by S437 have been satisfied with concrete evidence, and whether the Foundry system
possesses sufficient operational security for a future live trading ceremony under
minimum authorized scope.

This gate does NOT enable live trading. It evaluates whether a future ceremony
MAY enable live trading under explicit conditions and constraints.

## Predecessor Conditions

S437 delivered a CONDITIONAL AUTHORIZATION verdict with six explicit conditions
that must close before `dry_run=false` can be enabled on mainnet adapters:

| Condition | Requirement | Source Gap | Assigned Stage |
|-----------|-------------|------------|----------------|
| C-1 | Authenticated mainnet API call proven | RG-24 | S441 |
| C-2 | External secret manager deployed | RG-20 | S439 |
| C-3 | Automated backup with off-host replication | RG-22, RG-23 | S440 |
| C-4 | Sustained mainnet soak (endurance proof) | RG-25 | S441 |
| C-5 | Kill-switch operational runbook documented and tested | -- | S442 |
| C-6 | Explicit removal of `dry_run=false` config rejection | -- | S443 |

## Wave Execution Summary

| Stage | Objective | Delivered |
|-------|-----------|-----------|
| S438 | Wave charter and scope freeze | Charter frozen, 22 governing questions, 20 capabilities, 18 non-goals |
| S439 | External secret manager integration | FileCredentialProvider, config-driven provider selection, preflight validation |
| S440 | Automated backup with off-host replication | 4-phase pipeline, rsync replication, retention pruning, recovery proof |
| S441 | Authenticated mainnet API proof + sustained soak | AccountStatus() on real mainnet (Spot + Futures), 5-min soak, DryRunSubmitter 100% intact |
| S442 | Kill-switch operational runbook | Runbook, procedures, operational script, SLA definition (2s halt) |

## Condition Closure Assessment

### C-1: Authenticated Mainnet API Call Proven -- CLOSED

**Evidence:**
- `s441_authenticated_mainnet_proof_test.go` AMP-1: Authenticated Spot account status
  against `api.binance.com` (GET /api/v3/account, HMAC-SHA256 signed, HTTP 200)
- `s441_authenticated_mainnet_proof_test.go` AMP-2: Authenticated Futures account status
  against `fapi.binance.com` (GET /fapi/v2/account, HMAC-SHA256 signed, HTTP 200)
- `smoke-authenticated-mainnet-soak.sh` Phase 1: Connectivity proof

**Assessment:** CLOSED. Real mainnet API calls proven with valid credentials and
successful responses. Endpoint selection correctness verified.

### C-2: External Secret Manager Deployed -- CLOSED

**Evidence:**
- `file_credential_provider.go`: FileCredentialProvider reads from
  `{basePath}/{venue_type}/{KEY}`, whitespace-trimmed, fail-closed
- `schema.go`: `credential_provider` field ("env" default, "file" option) with
  `credential_path` cross-validation
- `preflight.go`: CredentialPathCheck validates path existence at boot
- `s439_external_secret_manager_test.go`: 11 tests covering resolution, whitespace,
  path validation, segment isolation
- `s439_credential_provider_config_test.go`: 8 tests covering config validation

**Assessment:** CLOSED. Config-driven provider model supports Vault Agent, AWS ESO,
K8s projected volumes, and Docker secrets through file-mount integration pattern.
No direct API client required -- file-sync adapters are the standard integration.

### C-3: Automated Backup with Off-Host Replication -- CLOSED

**Evidence:**
- `clickhouse-scheduled-backup.sh`: 4-phase pipeline (preflight, backup, replicate, prune)
- `smoke-automated-backup-offhost.sh`: Full cycle proof (backup, replicate, destroy local,
  restore from off-host, verify row count + markers)
- Configurable retention (`BACKUP_RETAIN_COUNT`), off-host target (`BACKUP_OFFHOST_TARGET`)
- Per-run audit logging in `./backups/logs/`

**Assessment:** CLOSED. Automated orchestration replaces manual trigger. Off-host
replication eliminates single-point-of-failure. Recovery from off-host copy proven.

### C-4: Sustained Mainnet Soak -- CLOSED

**Evidence:**
- `s441_authenticated_mainnet_proof_test.go` AMP-5: 5-minute sustained soak against
  both Spot and Futures mainnet endpoints
- Failure tolerance: 5% max (network jitter allowance)
- `s441_authenticated_mainnet_proof_test.go` AMP-6: DryRunSubmitter 100% interception
  reliability throughout soak
- `smoke-authenticated-mainnet-soak.sh` Phase 3-4: Configurable soak duration

**Assessment:** CLOSED. Sustained operation proven over 5-minute window with both
segments. DryRunSubmitter integrity maintained throughout. Soak window is configurable
for longer durations via `MF_SOAK_DURATION`.

### C-5: Kill-Switch Operational Runbook Documented and Tested -- CLOSED

**Evidence:**
- `kill-switch-operational-runbook.md`: Architecture, procedures, authorization model
- `kill-switch-trigger-verification-rollback-and-recovery-procedure.md`: Decision matrix,
  step-by-step halt/verify/resume/rollback
- `kill-switch-ops.sh`: Operational script with status/halt/resume/verify/cycle commands
- Kill-switch SLA: 2 seconds after PUT response (bounded by gateReadTimeout)
- HTTP surface verified: GET/PUT /execution/control wired in gateway compose

**Assessment:** CLOSED. Kill-switch has documented procedures, testable operational
script, bounded SLA, and verified HTTP control surface.

### C-6: Explicit Removal of `dry_run=false` Config Rejection -- NOT EXECUTED

**Evidence:**
- `schema.go` lines 517-524: Validation still rejects `dry_run=false` when mainnet
  adapters are configured
- Charter (S438) explicitly assigned C-6 to S443 as a deliberate architectural decision

**Assessment:** NOT EXECUTED -- BY DESIGN. The charter specifies that C-6 is a
conscious source-code change that requires authorization gate approval before execution.
This gate evaluates whether that removal MAY proceed, not whether it has proceeded.

## Safety Invariant Verification

All five fail-closed defense layers verified intact:

| Layer | Component | File | Status |
|-------|-----------|------|--------|
| 1 | Config validation (dry_run=false + mainnet rejected) | schema.go:517-524 | INTACT |
| 2 | Fail-closed default (dry_run=true) | schema.go | INTACT |
| 3 | DryRunSubmitter (intercepts 100% of SubmitOrder) | dry_run_submitter.go:77-133 | INTACT |
| 4 | Credential preflight (fail-fast on missing) | preflight.go:74-96 | INTACT |
| 5 | Kill-switch (SafetyGate before venue calls) | safety_gate.go:52-60, venue_adapter_actor.go:246 | INTACT |

Additional safety verification:
- DryRunSubmitter has NO bypass path (lines 30, 35, 73 confirm inner pipeline never called)
- Kill-switch HTTP control surface fully wired (execution_control.go, routes/execution.go, compose.go)
- Credential provider wiring in Phase -1 bootstrap (run.go:27-38)

## Regression Assessment

| Package | Regression | Notes |
|---------|------------|-------|
| internal/shared/settings | NONE | credential_provider, credential_path added without breaking changes |
| internal/shared/bootstrap | NONE | CredentialPathCheck, MainnetCredentialCheck added to preflight chain |
| internal/application/execution | NONE | FileCredentialProvider, RateLimiter, mainnet adapters added cleanly |
| internal/actors/scopes/execute | NONE | SafetyGate enforcement unchanged |
| internal/domain/execution | NONE | ControlGate model unchanged |
| internal/shared/healthz | NONE | SegmentHealthRegistry added without side effects |
| cmd/execute | NONE | Phase -1 credential wiring added to bootstrap |
| scripts/ | NONE | New scripts only; existing scripts unchanged |

**Zero regressions detected across all audited packages.**

## Capability Classification

| Capability Area | Rating | Evidence |
|-----------------|--------|----------|
| External secret manager integration | FULL | FileCredentialProvider + config validation + preflight + 19 tests |
| Automated backup with off-host replication | FULL | 4-phase pipeline + off-host proof + recovery cycle + retention |
| Authenticated mainnet API proof (Spot) | FULL | AMP-1: HTTP 200 from api.binance.com with HMAC-SHA256 |
| Authenticated mainnet API proof (Futures) | FULL | AMP-2: HTTP 200 from fapi.binance.com with HMAC-SHA256 |
| Sustained mainnet soak | FULL | AMP-5: 5-min soak, both segments, within 5% tolerance |
| DryRunSubmitter integrity under soak | FULL | AMP-6: 100% interception throughout soak |
| Kill-switch operational runbook | FULL | Documented procedures + trigger matrix + SLA |
| Kill-switch operational script | FULL | status/halt/resume/verify/cycle commands |
| Kill-switch SLA definition | FULL | 2s bounded by gateReadTimeout |
| Credential bootstrap lifecycle | FULL | 5-layer fail-closed model verified |
| Rate limiter integration | FULL | Token-bucket decorator with context cancellation |
| Segment health registry | FULL | Concurrent-safe phase tracking |
| Config-level mainnet protection | FULL | dry_run=false + mainnet still rejected |
| Mainnet adapter readiness (Spot) | FULL | Type alias + URL override + 13 adapter tests |
| Mainnet adapter readiness (Futures) | FULL | Type alias + URL override + matching pattern |

**15/15 capabilities at FULL rating. Zero PARTIAL or PENDING.**

## Formal Verdict

### AUTHORIZED -- CONDITIONAL FOR FUTURE LIVE TRADING CEREMONY

**Conditions C-1 through C-5: CLOSED with concrete evidence.**

**Condition C-6: AUTHORIZED but NOT EXECUTED.**

The evidence gate authorizes a future live trading ceremony to remove the
`dry_run=false` config rejection (C-6) under the following mandatory constraints:

### Authorization Scope (from S438 charter)

| Dimension | Authorized Value |
|-----------|-----------------|
| Exchange | Binance only |
| Segment | Spot only (Futures requires separate ceremony) |
| Symbols | 1 symbol (BTCUSDT) |
| Order size | Minimum exchange quantity |
| Order type | Market order only |
| Credentials | Trade-only (no withdrawal) |
| Kill-switch | Must be operationally tested (cycle) before session |
| Backup | Must run automated backup before and after session |
| Monitoring | Operator must monitor throughout session |

### Mandatory Pre-Session Checklist

1. Kill-switch cycle test passes (`kill-switch-ops.sh cycle`)
2. Automated backup completes successfully
3. Credentials are mounted via file provider (not env vars)
4. Config specifies exactly 1 symbol at minimum size
5. Operator confirms trade-only API key permissions
6. DryRunSubmitter removal is the ONLY config change from dry-run profile

### Post-Authorization Stop Conditions (from S438 charter)

Any of these triggers IMMEDIATE halt via kill-switch:
- SC-1: API error rate exceeds 10%
- SC-2: Latency exceeds 5x baseline
- SC-3: Unexpected order state (neither Accepted nor Rejected)
- SC-4: Fill quantity exceeds requested quantity
- SC-5: Kill-switch fails to respond within SLA
- SC-6: Credential error during session
- SC-7: ClickHouse write failures
- SC-8: NATS connectivity loss
- SC-9: Any operator uncertainty about system behavior

## Next Ceremony

**Recommended:** Live Trading Enablement Ceremony (implementation of C-6).

This ceremony would:
1. Remove the `dry_run=false` config rejection in `schema.go`
2. Create a production config profile for minimum authorized scope
3. Execute a single supervised live trading session under stop conditions
4. Render evidence of successful live order lifecycle (submit, accept, fill)

**Pre-condition:** This evidence gate (S443) must be reviewed and accepted by
the repository owner before the enablement ceremony proceeds.

## References

- [Live Trading Authorization Wave Charter](live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md)
- [External Secret Manager Integration](external-secret-manager-integration-for-live-authorization.md)
- [Automated Backup with Off-Host Replication](automated-backup-with-off-host-replication.md)
- [Authenticated Mainnet API Proof](authenticated-mainnet-api-proof-and-sustained-soak.md)
- [Kill-Switch Operational Runbook](kill-switch-operational-runbook.md)
- [Kill-Switch Procedures](kill-switch-trigger-verification-rollback-and-recovery-procedure.md)
- [Credential Bootstrap Semantics](live-credentials-bootstrap-lookup-rotation-assumptions-and-fail-closed-semantics.md)
- [Mainnet Authorization Evidence Gate (S437)](mainnet-authorization-evidence-gate.md)
