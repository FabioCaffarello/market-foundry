# Live Trading Authorization Evidence Matrix, Blockers, Conditions, and Next Ceremony

Authority: S443. Companion to [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md).

## Evidence Matrix

### Condition Closure Matrix

| # | Condition | Source | Stage | Evidence | Rating | Closed |
|---|-----------|--------|-------|----------|--------|--------|
| C-1 | Authenticated mainnet API call | RG-24 | S441 | AMP-1 (Spot HTTP 200), AMP-2 (Futures HTTP 200), HMAC-SHA256 signing | FULL | YES |
| C-2 | External secret manager | RG-20 | S439 | FileCredentialProvider, config validation, preflight, 19 tests | FULL | YES |
| C-3 | Automated off-host backup | RG-22, RG-23 | S440 | 4-phase pipeline, rsync replication, recovery proof, retention | FULL | YES |
| C-4 | Sustained mainnet soak | RG-25 | S441 | AMP-5 (5-min soak, both segments, 5% tolerance), AMP-6 (DryRunSubmitter 100%) | FULL | YES |
| C-5 | Kill-switch operational runbook | -- | S442 | Runbook, procedures, script, SLA 2s, HTTP surface verified | FULL | YES |
| C-6 | Remove dry_run=false rejection | -- | S443 | Authorized but NOT executed (by design; requires enablement ceremony) | AUTHORIZED | DEFERRED |

**5/6 conditions CLOSED. 1/6 AUTHORIZED for future execution.**

### Capability Evidence Matrix

| # | Capability | Stage | Test/Script Evidence | Lines | Rating |
|---|-----------|-------|---------------------|-------|--------|
| 1 | FileCredentialProvider implementation | S439 | s439_external_secret_manager_test.go (11 tests) | 203 | FULL |
| 2 | Config-driven provider selection | S439 | s439_credential_provider_config_test.go (8 tests) | 141 | FULL |
| 3 | Credential preflight validation | S439 | s434_mainnet_credential_preflight_test.go (8 tests) | 135 | FULL |
| 4 | Automated backup orchestration | S440 | clickhouse-scheduled-backup.sh (4-phase) | -- | FULL |
| 5 | Off-host replication via rsync | S440 | smoke-automated-backup-offhost.sh (full cycle) | -- | FULL |
| 6 | Post-replication verification | S440 | File-count parity check in scheduled script | -- | FULL |
| 7 | Automatic retention pruning | S440 | BACKUP_RETAIN_COUNT config + prune phase | -- | FULL |
| 8 | Recovery from off-host backup | S440 | Destroy local, restore from off-host, verify markers | -- | FULL |
| 9 | Authenticated Spot mainnet call | S441 | AMP-1: GET /api/v3/account, HTTP 200, HMAC-SHA256 | 440 | FULL |
| 10 | Authenticated Futures mainnet call | S441 | AMP-2: GET /fapi/v2/account, HTTP 200, HMAC-SHA256 | -- | FULL |
| 11 | DryRunSubmitter post-auth integrity | S441 | AMP-3: 100% interception after authenticated calls | -- | FULL |
| 12 | Pipeline chain with auth adapters | S441 | AMP-4: adapter + RateLimiter + DryRunSubmitter | -- | FULL |
| 13 | Sustained soak (Spot + Futures) | S441 | AMP-5: 5 min, both segments, 5% tolerance | -- | FULL |
| 14 | DryRunSubmitter soak stability | S441 | AMP-6: 100% interception throughout soak | -- | FULL |
| 15 | Kill-switch runbook | S442 | kill-switch-operational-runbook.md | -- | FULL |
| 16 | Kill-switch procedures | S442 | kill-switch-trigger-verification-rollback-and-recovery-procedure.md | -- | FULL |
| 17 | Kill-switch operational script | S442 | kill-switch-ops.sh (status/halt/resume/verify/cycle) | 289 | FULL |
| 18 | Kill-switch SLA definition | S442 | 2s bounded by gateReadTimeout in safety_gate.go:40 | -- | FULL |
| 19 | Rate limiter integration | S441 | Token-bucket in rate_limiter.go, AMP-4 pipeline | 92 | FULL |
| 20 | Segment health registry | S429 | segment_health.go + segment_health_test.go (9 tests) | 391 | FULL |

**20/20 capabilities at FULL rating.**

### Safety Invariant Matrix

| # | Invariant | File:Line | Status |
|---|-----------|-----------|--------|
| SI-1 | dry_run=false + mainnet rejected by config | schema.go:517-524 | INTACT |
| SI-2 | DryRunSubmitter intercepts all SubmitOrder | dry_run_submitter.go:77-133 | INTACT |
| SI-3 | DryRunSubmitter has zero bypass paths | dry_run_submitter.go:30,35,73 | INTACT |
| SI-4 | SafetyGate checked before venue calls | venue_adapter_actor.go:246 | INTACT |
| SI-5 | Kill-switch enforcement via IsHalted() | safety_gate.go:52-60 | INTACT |
| SI-6 | gateReadTimeout = 2s | safety_gate.go:40 | INTACT |
| SI-7 | MainnetCredentialCheck at preflight | preflight.go:74-96 | INTACT |
| SI-8 | CredentialPathCheck at preflight | preflight.go:105-129 | INTACT |
| SI-9 | Phase -1 credential provider wiring | run.go:27-38 | INTACT |
| SI-10 | HTTP PUT /execution/control wired | execution_control.go:51-75 | INTACT |
| SI-11 | HTTP GET /execution/control wired | execution_control.go:30-48 | INTACT |
| SI-12 | Gateway composition connects control | compose.go:115-118 | INTACT |

**12/12 safety invariants INTACT.**

## Regression Assessment

| Package | Tests | Status | Notes |
|---------|-------|--------|-------|
| internal/shared/settings | Config validation, segment enablement | PASS | credential_provider fields added cleanly |
| internal/shared/bootstrap | Preflight checks | PASS | New checks added to chain |
| internal/application/execution | Credential providers, adapters, DryRunSubmitter | PASS | No existing behavior modified |
| internal/actors/scopes/execute | SafetyGate, VenueAdapterActor | PASS | Kill-switch enforcement unchanged |
| internal/domain/execution | ControlGate model | PASS | No changes in wave |
| internal/shared/healthz | Health checks | PASS | SegmentHealthRegistry added |
| cmd/execute | Bootstrap sequence | PASS | Phase -1 added without disruption |
| cmd/gateway | HTTP routing | PASS | Control endpoints pre-existing |

**Zero regressions. Zero new medium+ severity gaps.**

## Blockers

### Critical Blockers: NONE

No critical blockers remain. All six S437 conditions have been addressed:
five closed with evidence, one authorized for future execution.

### Non-Blockers (Low Severity, Pre-Existing or Accepted)

| ID | Severity | Description | Origin | Status |
|----|----------|-------------|--------|--------|
| RG-S439-1 | LOW | No credential rotation without restart | S439 | ACCEPTED (RG-21, restart acceptable) |
| RG-S439-2 | LOW | No multi-provider fallback | S439 | ACCEPTED (single provider per process) |
| RG-S439-3 | LOW | No hot-reload of credentials | S439 | ACCEPTED (restart model sufficient) |
| RG-S440-1 | LOW | No push alerting on backup failure | S440 | ACCEPTED (operator checks logs) |
| RG-S440-2 | LOW | No S3/GCS integration | S440 | ACCEPTED (rsync sufficient) |
| RG-S440-3 | LOW | No point-in-time recovery | S440 | ACCEPTED (full snapshots sufficient) |
| RG-S441-1 | LOW | AccountStatus() is read-only proof | S441 | ACCEPTED (SubmitOrder blocked by DryRunSubmitter) |
| RG-S441-2 | LOW | Soak window is 5 minutes | S441 | ACCEPTED (configurable, sufficient for proof) |
| RG-S441-3 | LOW | No WebSocket authenticated streams | S441 | ACCEPTED (REST sufficient for execution path) |
| RG-S442-1 | LOW | No per-segment kill-switch | S442 | ACCEPTED (global sufficient for Spot-only scope) |
| RG-S442-2 | LOW | No automated halt triggers | S442 | ACCEPTED (human-in-the-loop by design) |
| RG-S442-3 | LOW | No HTTP auth on gateway | S442 | ACCEPTED (localhost binding) |
| RG-S442-4 | LOW | No historical audit log | S442 | ACCEPTED (KV entry carries timestamp/reason/by) |
| RG-S442-5 | LOW | Fail-open on NATS KV unavailability | S442 | ACCEPTED (design since S335; NATS loss is halt trigger) |

**14 low-severity gaps. Zero medium+ severity gaps. All documented and accepted.**

## Conditions for Future Live Trading

### Mandatory Conditions (from this gate)

1. **C-6 execution requires a dedicated enablement ceremony** with:
   - Source-code change to `schema.go` removing dry_run=false rejection for mainnet
   - Production config profile for minimum authorized scope
   - Kill-switch cycle test before session
   - Automated backup before and after session

2. **Scope is frozen to minimum authorized surface:**
   - Binance Spot only (Futures requires separate ceremony)
   - 1 symbol (BTCUSDT) at minimum exchange quantity
   - Market orders only
   - Trade-only credentials (no withdrawal)

3. **Post-authorization stop conditions are binding:**
   - Any of SC-1 through SC-9 triggers immediate kill-switch halt
   - Operator must monitor throughout session
   - Session duration bounded by operator presence

4. **Rollback criteria are binding:**
   - RC-1: Any regression in safety invariants halts wave
   - RC-2: Kill-switch failure halts wave
   - RC-3: Credential exposure halts wave
   - RC-4: Backup failure blocks session
   - RC-5: Config validation bypass halts wave
   - RC-6: DryRunSubmitter bypass detected halts wave
   - RC-7: Scope inflation beyond charter halts wave

### Residual Limitations (Acknowledged, Not Blocking)

1. Credential rotation requires process restart
2. Kill-switch is global (no per-segment granularity)
3. Backup alerting is log-based (no push notification)
4. Mainnet soak proven at 5 minutes (not production endurance)
5. AccountStatus() proof is read-only (SubmitOrder proof deferred to enablement ceremony)

## Governing Questions Resolution

| # | Question | Answer | Stage |
|---|----------|--------|-------|
| GQ-1 | Can credentials be managed via external secret manager? | YES (FileCredentialProvider) | S439 |
| GQ-2 | Does config validation reject unknown providers? | YES (fail-closed) | S439 |
| GQ-3 | Does preflight validate credential path? | YES (CredentialPathCheck) | S439 |
| GQ-4 | Is backup automated? | YES (4-phase pipeline) | S440 |
| GQ-5 | Does backup replicate off-host? | YES (rsync to configurable target) | S440 |
| GQ-6 | Is recovery from off-host proven? | YES (destroy + restore + verify) | S440 |
| GQ-7 | Is retention automated? | YES (BACKUP_RETAIN_COUNT pruning) | S440 |
| GQ-8 | Can the system authenticate against mainnet Spot? | YES (AMP-1, HTTP 200) | S441 |
| GQ-9 | Can the system authenticate against mainnet Futures? | YES (AMP-2, HTTP 200) | S441 |
| GQ-10 | Is HMAC-SHA256 signing correct for mainnet? | YES (AMP-3) | S441 |
| GQ-11 | Does DryRunSubmitter survive authenticated calls? | YES (AMP-3, AMP-6: 100%) | S441 |
| GQ-12 | Is sustained soak stable? | YES (AMP-5: 5 min, 5% tolerance) | S441 |
| GQ-13 | Are both segments covered in soak? | YES (Spot + Futures) | S441 |
| GQ-14 | Is endpoint selection correct? | YES (api.binance.com, fapi.binance.com) | S441 |
| GQ-15 | Is kill-switch runbook documented? | YES (runbook + procedures) | S442 |
| GQ-16 | Is kill-switch operationally testable? | YES (kill-switch-ops.sh cycle) | S442 |
| GQ-17 | Is kill-switch SLA bounded? | YES (2s after PUT response) | S442 |
| GQ-18 | Is recovery from halt clean? | YES (resume + verify-active) | S442 |
| GQ-19 | Are all 5 safety layers intact? | YES (12 invariants verified) | S443 |
| GQ-20 | Are there zero regressions? | YES (all packages clean) | S443 |
| GQ-21 | Is scope frozen to minimum? | YES (charter enforces) | S438 |
| GQ-22 | Are stop conditions defined? | YES (SC-1 through SC-9) | S438 |

**22/22 governing questions answered with evidence.**

## Next Ceremony Recommendation

### Recommended: Live Trading Enablement Ceremony

**Objective:** Execute C-6 and conduct first supervised live trading session.

**Proposed scope:**
1. Source-code change: remove `dry_run=false` rejection for mainnet in `schema.go`
2. Create `execute-mainnet-live.jsonc` config for minimum authorized scope
3. Pre-session: kill-switch cycle, backup, credential mount verification
4. Session: single BTCUSDT market order at minimum quantity
5. Post-session: verify order lifecycle (submit, accept, fill), backup, halt
6. Evidence: order receipt, fill confirmation, ClickHouse persistence, audit log

**Pre-conditions:**
- This evidence gate (S443) accepted by repository owner
- Valid trade-only API key for Binance Spot mainnet
- Off-host backup target configured
- Operator available for full session monitoring

**Risk profile:** LOW. All safety layers verified. Kill-switch provides 2s halt.
Scope is irreducibly minimal. Single market order at minimum size represents
negligible financial exposure.

## References

- [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md)
- [Mainnet Authorization Evidence Gate (S437)](mainnet-authorization-evidence-gate.md)
- [Mainnet Authorization Evidence Matrix](mainnet-authorization-evidence-matrix-blockers-residual-gaps-and-next-ceremony.md)
- [Live Trading Authorization Wave Charter](live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md)
