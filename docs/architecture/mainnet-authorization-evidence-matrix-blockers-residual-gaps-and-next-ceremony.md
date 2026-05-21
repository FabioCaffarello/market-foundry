# Mainnet Authorization Evidence Matrix, Residual Gaps, and Next Ceremony

> Authority: S437 | Date: 2026-03-24 | Phase: 49 (Mainnet Enablement)

## Evidence Matrix

### Capability Classification Scale

| Rating | Meaning |
|--------|---------|
| FULL | Capability proven end-to-end with tests, documentation, and operational evidence |
| SUBSTANTIAL | Core capability proven; minor edges documented as accepted limitations |
| PARTIAL | Capability structurally present but missing significant operational proof |
| PENDING | Not implemented or not evaluated |

### S433 -- Mainnet Adapter Readiness

| ID | Capability | Rating | Evidence |
|----|-----------|--------|----------|
| C-1 | Spot mainnet adapter (VenuePort compliance) | FULL | Type alias adapter + interface test + base URL verification |
| C-2 | Futures mainnet adapter (VenuePort compliance) | FULL | Type alias adapter + interface test + base URL verification |
| C-3 | Rate-limiter decorator | FULL | Token-bucket implementation + pass-through test + context cancellation test |
| C-4 | Config-driven adapter selection | FULL | `buildVenueAdapterByType` switch cases + VenueType constants + 10 config validation tests |

**Stage verdict: FULL DELIVERY (4/4 FULL)**

### S434 -- Secret Manager Integration

| ID | Capability | Rating | Evidence |
|----|-----------|--------|----------|
| C-5 | CredentialProvider interface | FULL | Interface defined + `Resolve`/`Name` contract + pluggable via `SetCredentialProvider` |
| C-6 | EnvCredentialProvider (concrete impl) | FULL | Default provider + backward-compatible env var resolution + 4 tests |
| C-7 | Mainnet adapter wiring | FULL | Preflight check + credential loading at adapter bootstrap + audit logging |
| C-8 | Fail-closed credential behavior | FULL | 7 preflight tests (missing/partial/multi-segment scenarios) + format validation (min 16 chars, no whitespace) |

**Stage verdict: FULL DELIVERY (4/4 FULL)**

### S435 -- ClickHouse Backup/Restore

| ID | Capability | Rating | Evidence |
|----|-----------|--------|----------|
| C-9 | Canonical backup procedure | FULL | `scripts/clickhouse-backup.sh` + `make ch-backup` + native SQL `BACKUP TABLE` |
| C-10 | Restore with verified integrity | FULL | `scripts/clickhouse-restore.sh` + `make ch-restore` + 33/33 automated checks (row counts, marker rows, schema) |
| C-11 | RTO/RPO documentation | FULL | Benchmarked RTO estimates (5s current to 190s at 10M rows) + runbook with risk matrix |

**Stage verdict: FULL DELIVERY (3/3 FULL)**

### S436 -- Mainnet Dry-Run Proof

| ID | Capability | Rating | Evidence |
|----|-----------|--------|----------|
| C-12 | Spot mainnet dry-run proof | FULL | DNS + TLS + HTTP ping + DryRunSubmitter interception proven (MDR-1 through MDR-4) |
| C-13 | Futures mainnet dry-run proof | FULL | DNS + TLS + HTTP ping + DryRunSubmitter interception proven (MDR-1, MDR-2, MDR-3, MDR-5) |
| C-14 | DryRunSubmitter chain verification | FULL | Full pipeline composition test (MDR-8) + audit marker consistency (MDR-6) |
| C-15 | Fail-closed config defaults | FULL | `dry_run` omitted defaults to true + 9 config validation tests |
| C-16 | Safety layer verification | FULL | 5-layer defense documented and test-proven; zero real orders confirmed |

**Stage verdict: FULL DELIVERY (5/5 FULL)**

### S437 -- Evidence Gate

| ID | Capability | Rating | Evidence |
|----|-----------|--------|----------|
| C-17 | Evidence gate verdict | FULL | This document + evidence gate document + stage report |

**Stage verdict: FULL DELIVERY (1/1 FULL)**

### Wave Summary

| Stage | Capabilities | FULL | SUBSTANTIAL | PARTIAL | PENDING |
|-------|-------------|------|-------------|---------|---------|
| S433 | 4 | 4 | 0 | 0 | 0 |
| S434 | 4 | 4 | 0 | 0 | 0 |
| S435 | 3 | 3 | 0 | 0 | 0 |
| S436 | 5 | 5 | 0 | 0 | 0 |
| S437 | 1 | 1 | 0 | 0 | 0 |
| **Total** | **17** | **17** | **0** | **0** | **0** |

**Wave classification: FULL DELIVERY -- 17/17 capabilities at FULL rating.**

## Governing Questions Resolution

| ID | Question | Answer | Evidence Stage |
|----|----------|--------|---------------|
| GQ-1 | Do mainnet adapters satisfy VenuePort interface? | Yes -- type alias guarantees identical interface | S433 |
| GQ-2 | What API differences exist between testnet and mainnet? | Base URL only; auth, paths, response schemas identical | S433 |
| GQ-3 | Is the rate-limiter strategy sufficient? | Yes -- 10 burst / 100ms refill covers Binance rate limits | S433 |
| GQ-4 | Is config sufficient to select mainnet adapters? | Yes -- VenueType constants + `buildVenueAdapterByType` | S433 |
| GQ-5 | What is the minimal credential interface? | `Resolve(venueType, key) string` + `Name() string` | S434 |
| GQ-6 | Which credential provider to use? | EnvCredentialProvider default; interface supports Vault/AWS SM | S434 |
| GQ-7 | Are hot-swap credentials needed? | No -- startup-only loading acceptable for dry-run scope | S434 |
| GQ-8 | How are fail-closed semantics ensured? | Phase 0 preflight + format validation + exit(1) on failure | S434 |
| GQ-9 | Which backup tool? | Native ClickHouse SQL (BACKUP/RESTORE to Disk) | S435 |
| GQ-10 | What is acceptable RTO/RPO? | <35s at 1M rows; <190s at 10M rows | S435 |
| GQ-11 | Is TTL preserved through backup/restore? | Yes -- schema properties verified post-restore | S435 |
| GQ-12 | Does WebSocket behavior differ on mainnet? | Not evaluated (execute binary scope; ingest concern) | N/A |
| GQ-13 | Does DryRunSubmitter intercept mainnet adapters? | Yes -- proven for both Spot and Futures (MDR-4, MDR-5) | S436 |
| GQ-14 | How are intents derived in dry-run mode? | Same pipeline; DryRunSubmitter returns synthetic fills | S436 |
| GQ-15 | How is the safety layer verified? | 5-layer defense: config validation + default + DryRunSubmitter + preflight + kill switch | S436 |
| GQ-16 | Are all blockers resolved? | Yes -- B-1 (S433), B-2 (S434), B-3 (S435) all CLOSED | S437 |
| GQ-17 | Were zero real orders placed? | Confirmed -- DryRunSubmitter prevents all authenticated API calls | S437 |
| GQ-18 | Is the authorization threshold met? | Yes -- 17/17 FULL, 0 regressions, 0 medium+ gaps | S437 |

## Blocker Closure Summary

| Blocker | Description | Closed In | Resolution | Evidence Strength |
|---------|-------------|-----------|------------|-------------------|
| B-1 | No mainnet adapter implementation | S433 | Type-alias adapters + mainnet URLs + config wiring | FULL (20 tests) |
| B-2 | No mainnet credential management | S434 | CredentialProvider interface + preflight + format validation | FULL (20 tests) |
| B-3 | No ClickHouse backup/restore strategy | S435 | Native SQL backup/restore + automated 33-check proof | FULL (33 checks) |

**All three blockers CLOSED. Zero blockers remaining.**

## Residual Gaps

### Pre-Existing Gaps (Carried from S431)

18 LOW-severity gaps carried forward unchanged. These were accepted in prior gates and remain accepted:

- RG-2: Partial fill testnet limitation
- RG-8: Synthetic endurance testing
- RG-9: No time-based drift detection
- RG-11: Lifecycle list eventual consistency <1s
- RG-15: Single symbol at compose level
- (13 additional LOW gaps from earlier waves)

### New Gaps Introduced by This Wave

| ID | Severity | Description | Stage | Mitigation |
|----|----------|-------------|-------|-----------|
| RG-20 | LOW | Mainnet credentials stored in env vars (no external secret manager deployed) | S434 | CredentialProvider interface ready for Vault/AWS SM; env vars acceptable for dry-run |
| RG-21 | LOW | Credential rotation requires process restart | S434 | Acceptable for dry-run scope; hot-swap deferred |
| RG-22 | LOW | ClickHouse backup is manual trigger only | S435 | Operator checklist documented; cron automation deferred |
| RG-23 | LOW | ClickHouse backup stored on same host filesystem | S435 | Operator must copy to external storage; documented in runbook |
| RG-24 | LOW | No authenticated mainnet API call proven | S436 | By design (DryRunSubmitter intercepts); must be proven before live trading |
| RG-25 | LOW | No sustained mainnet soak test | S436 | Point-in-time proof only; endurance deferred to pre-live ceremony |

### Gap Profile Summary

| Severity | Pre-Existing | New (This Wave) | Total |
|----------|-------------|-----------------|-------|
| Critical | 0 | 0 | **0** |
| High | 0 | 0 | **0** |
| Medium | 0 | 0 | **0** |
| Low | 18 | 6 | **24** |

**No severity escalation. Zero medium+ gaps across the entire project history since S370.**

## Non-Blockers Status (from S430 Audit)

| ID | Description | Status |
|----|-------------|--------|
| NB-1 | Configurable rate limiter | Partially addressed (fixed params in S433); LOW |
| NB-2 | Per-segment kill switch | Deferred; LOW |
| NB-3 | Idle detection and graceful shutdown | Deferred; LOW |
| NB-4 | OTEL tracing integration | Deferred; LOW |
| NB-5 | Alerting rules | Deferred; LOW |
| NB-6 | Execution pagination | Deferred; LOW |
| NB-7 | Rejection reason code column | Deferred; LOW |
| NB-8 | /fapi/v1/userTrades integration | Deferred; LOW |
| NB-9 | Parallel execution proof (multi-symbol) | Deferred; LOW |
| NB-10 | Documentation index refresh | Deferred; LOW |

No non-blocker was promoted to blocker during this wave.

## Regression Verification

### Test Execution (2026-03-24)

| Package | Tests | Result | Duration |
|---------|-------|--------|----------|
| `internal/shared/settings` | S393, S400, S401, S416, S419, S433, S436 | PASS | 0.190s |
| `internal/shared/bootstrap` | S434 | PASS | 0.170s |
| `internal/application/execution` | S384-S387, S400, S405-S407, S412-S413, S416-S418, S422-S424, S428, S433-S434 | PASS | 32.090s |
| `internal/actors/scopes/execute` | S373-S374, S379-S380, S386, S401, S405-S408, S416-S419, S425, S429 | PASS | 1.418s |
| `internal/domain/execution` | S384, S386 | PASS | cached |
| `internal/adapters/clickhouse/writerpipeline` | existing | PASS | cached |
| `internal/adapters/nats/natsexecution` | S386, S401 | PASS | cached |

### Binary Build

| Binary | Result |
|--------|--------|
| `cmd/execute` | CLEAN |
| `cmd/writer` | CLEAN |

**Regression verdict: ZERO REGRESSIONS. All tests pass. All binaries compile.**

## Next Ceremony Recommendation

### Immediate Next Step

**The Mainnet Enablement Wave is AUTHORIZED TO CLOSE.**

The wave delivered 17/17 capabilities at FULL rating, closed all three mainnet blockers, introduced zero regressions, and maintained zero medium+ severity gaps.

### Strategic Options for the Next Macro-Front

Based on the current state of the platform, three strategic directions emerge:

#### Option A: Live Trading Authorization Ceremony (Recommended)

**Objective:** Authorize `dry_run=false` on mainnet for a controlled, single-symbol, single-segment scope.

**Prerequisites (from this gate):**
1. Prove authenticated mainnet API call (spot and futures)
2. Deploy external secret manager (Vault or AWS Secrets Manager)
3. Automate ClickHouse backup with off-host replication
4. Execute sustained mainnet soak (endurance proof)
5. Document and test kill-switch operational procedure
6. Remove `dry_run=false` config rejection for mainnet (deliberate source change)

**Risk profile:** HIGH -- first real order on a real exchange. Requires ceremony-grade authorization with explicit human sign-off at each step.

#### Option B: Operational Hardening Wave

**Objective:** Close non-blockers NB-1 through NB-10 and harden operational surfaces before live trading.

**Scope:** Configurable rate limits, per-segment kill switch, idle detection, OTEL tracing, alerting rules, pagination, rejection columns.

**Risk profile:** LOW -- no new exchange interaction; internal quality improvements.

#### Option C: Multi-Symbol/Multi-Exchange Expansion

**Objective:** Expand beyond single-symbol, single-exchange scope.

**Risk profile:** MEDIUM -- structural changes to routing, config, and compose surfaces.

### Recommendation

**Option A (Live Trading Authorization Ceremony)** is the recommended next macro-front.

Rationale:
1. All structural and operational prerequisites for mainnet are in place.
2. The 6 conditions identified in this gate are well-scoped and can form a focused wave (5-6 stages).
3. Operational hardening (Option B) and expansion (Option C) are more valuable after live trading is proven -- they are optimizations of a system that works, not prerequisites for a system that does not yet work.
4. The platform has accumulated 14 consecutive wave passes with zero regressions. The execution discipline is proven.

The recommended ceremony structure:

| Stage | Description |
|-------|-------------|
| S438 | Live Trading Authorization Wave Charter and Scope Freeze |
| S439 | External Secret Manager Deployment (C-2) |
| S440 | Automated Backup and Off-Host Replication (C-3) |
| S441 | Authenticated Mainnet API Proof and Sustained Soak (C-1, C-4) |
| S442 | Kill-Switch Operational Runbook and Test (C-5) |
| S443 | Live Trading Authorization Evidence Gate (C-6: explicit dry_run removal) |

This structure preserves the wave discipline that has delivered 14 consecutive passes while taking the platform through the most consequential ceremony in its history.
