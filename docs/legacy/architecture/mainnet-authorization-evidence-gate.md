# Mainnet Authorization Evidence Gate

> Authority: S437 | Date: 2026-03-24 | Phase: 49 (Mainnet Enablement) | Verdict: see below

## Purpose

This document renders the formal evidence gate for the Mainnet Enablement Wave (S432-S436). It evaluates whether the wave closed the three explicit mainnet blockers, proved dry-run execution on real mainnet endpoints, preserved fail-closed semantics, and authorizes (or does not authorize) a future mainnet trading ceremony.

This gate does NOT authorize live trading. It evaluates whether the structural, operational, and safety prerequisites for a future live-trading authorization ceremony have been met.

## Wave Charter Recap

The Mainnet Enablement Wave opened after 12 consecutive wave passes with zero regressions (since S370). It was chartered in S432 to resolve three explicit mainnet blockers identified in the S430 readiness audit:

| Blocker | Description | Assigned Stage |
|---------|-------------|----------------|
| B-1 | No mainnet adapter implementation | S433 |
| B-2 | No mainnet credential management | S434 |
| B-3 | No ClickHouse backup/restore strategy | S435 |

After blocker closure, S436 was chartered to prove dry-run execution against real mainnet endpoints. S437 (this gate) evaluates the wave.

## Evaluation Criteria

The gate evaluates five dimensions:

1. **Blocker closure** -- all three blockers resolved with evidence
2. **Operational proof** -- dry-run execution proven on real mainnet endpoints
3. **Safety invariant** -- zero real orders placed; fail-closed semantics intact
4. **Regression status** -- no regressions introduced in the wave
5. **Residual gap profile** -- no new high/medium-severity gaps

## Blocker Resolution Audit

### B-1: Mainnet Adapter Implementation (S433)

**Status: CLOSED -- FULL EVIDENCE**

| Evidence | Detail |
|----------|--------|
| Spot mainnet adapter | `binance_spot_mainnet_adapter.go` -- type alias over testnet adapter, mainnet base URL (`api.binance.com`) |
| Futures mainnet adapter | `binance_futures_mainnet_adapter.go` -- type alias over testnet adapter, mainnet base URL (`fapi.binance.com`) |
| Rate limiter | `rate_limiter.go` -- token-bucket decorator (10 burst, 100ms refill) |
| VenuePort interface compliance | Both adapters satisfy `VenuePort` and `VenueQueryPort` (test-proven) |
| Config-driven selection | `cmd/execute/run.go` wires adapters by VenueType; `schema.go` adds mainnet VenueType constants |
| Config enforcement | `dry_run=false` + mainnet adapter rejected at validation (5 test cases) |
| Test count | 20 tests (10 adapter + 10 config), all PASS |

### B-2: Mainnet Credential Management (S434)

**Status: CLOSED -- FULL EVIDENCE**

| Evidence | Detail |
|----------|--------|
| CredentialProvider interface | `credentials.go` -- `Resolve(venueType, key) string` + `Name() string` |
| EnvCredentialProvider | Default provider reads `MF_VENUE_{TYPE}_{KEY}` env vars |
| Pluggable backend | `SetCredentialProvider()` allows Vault/AWS SM/file-based substitution |
| Format validation | Mainnet Binance: min 16 chars, no whitespace; testnet unrestricted |
| Preflight check | `bootstrap.MainnetCredentialCheck()` fails-fast at Phase 0 (before I/O) |
| Backward compatibility | Existing env-var path unchanged; all testnet tests pass |
| Test count | 20 tests (13 provider + 7 preflight), all PASS |

### B-3: ClickHouse Backup/Restore (S435)

**Status: CLOSED -- FULL EVIDENCE**

| Evidence | Detail |
|----------|--------|
| Backup mechanism | Native ClickHouse `BACKUP TABLE ... TO Disk()` (zero external dependencies) |
| Restore mechanism | Native `RESTORE TABLE ... FROM Disk()` with drop-before-restore |
| Tables covered | All 7 MergeTree tables (executions, evidence_candles, signals, decisions, strategies, risk_assessments, _migrations) |
| Automated proof | `smoke-clickhouse-backup-restore.sh` -- 9 steps, 33 checks, 33/33 PASS |
| Schema preservation | DDL, TTL, partitioning, order keys verified post-restore |
| Data integrity | Row counts match; proof marker rows survive full cycle |
| RTO estimate | <1s current; <35s at 1M rows; <190s at 10M rows |
| Makefile integration | `make ch-backup`, `make ch-restore`, `make ch-backup-list` |

## Operational Proof Audit (S436)

**Status: PROVEN -- FULL EVIDENCE**

### Mainnet Connectivity

| Check | Spot | Futures |
|-------|------|---------|
| DNS resolution | PASS (`api.binance.com`) | PASS (`fapi.binance.com`) |
| TCP port 443 | PASS | PASS |
| TLS handshake (1.2+) | PASS | PASS |
| `/ping` HTTP 200 | PASS | PASS |

### DryRunSubmitter Interception

| Check | Result |
|-------|--------|
| Spot mainnet adapter wrapped by DryRunSubmitter | PASS -- inner adapter never invoked |
| Futures mainnet adapter wrapped by DryRunSubmitter | PASS -- inner adapter never invoked |
| Full pipeline chain (Adapter -> RateLimiter -> DryRunSubmitter) | PASS -- all intents intercepted |

### Audit Trail Markers

All dry-run receipts carry unambiguous markers:
- `VenueOrderID`: `dryrun-{16-char hex}`
- `Simulated: true`
- `Fee: "0"`

### Config and Compose

| Artifact | Purpose |
|----------|---------|
| `execute-mainnet-dry-run.jsonc` | Explicit `dry_run: true`, both segments enabled with mainnet adapters |
| `docker-compose.mainnet-dry-run.yaml` | Compose overlay with credential env vars (never embedded) |
| `smoke-mainnet-dry-run.sh` | 4-phase smoke: config validation, connectivity, interception, audit markers |

## Safety Invariant Audit

### Five-Layer Fail-Closed Defense

| Layer | Mechanism | Bypass Requires |
|-------|-----------|-----------------|
| 1. Config validation | `dry_run=false` + mainnet rejected at parse time | Modifying Go source (`schema.go`) |
| 2. Fail-closed default | Omitted `dry_run` defaults to `true` | Explicit `dry_run: false` (caught by Layer 1) |
| 3. DryRunSubmitter | Intercepts all `SubmitOrder` calls at runtime | Removing decorator from `run.go` |
| 4. Credential preflight | Fails-fast if credentials missing/malformed | N/A (defense-in-depth) |
| 5. Kill switch | `EXECUTION_CONTROL` KV halts execution independently | Direct NATS KV manipulation |

**Assessment:** Any single layer is sufficient to prevent real order submission. All five are active simultaneously. Accidental real order submission requires modifying source code in at least two locations and recompiling -- this is not a configuration-reachable state.

### Zero Real Orders Confirmation

- No authenticated API calls reach mainnet endpoints (DryRunSubmitter intercepts first)
- No order-related HTTP requests observed in any test or smoke execution
- All receipts carry `dryrun-` prefix and `Simulated: true`

## Regression Verification

### Test Suite Results (2026-03-24)

| Package | Result | Duration |
|---------|--------|----------|
| `internal/shared/settings` | PASS | 0.190s |
| `internal/shared/bootstrap` | PASS | 0.170s |
| `internal/application/execution` | PASS | 32.090s |
| `internal/actors/scopes/execute` | PASS | 1.418s |
| `internal/domain/execution` | PASS | cached |
| `internal/adapters/clickhouse/writerpipeline` | PASS | cached |
| `internal/adapters/nats/natsexecution` | PASS | cached |

### Binary Compilation

| Binary | Result |
|--------|--------|
| `cmd/execute` | BUILDS CLEAN |
| `cmd/writer` | BUILDS CLEAN |

### Regression Verdict

**ZERO REGRESSIONS.** All packages pass. Both binaries compile. No test failures introduced by the wave.

## Residual Gap Profile

### New Gaps Introduced by This Wave

| Gap | Severity | Description |
|-----|----------|-------------|
| RG-20 | LOW | Mainnet credentials via env vars (no Vault/AWS SM in production yet) |
| RG-21 | LOW | No credential rotation without restart |
| RG-22 | LOW | ClickHouse backup is manual trigger only (no automated schedule) |
| RG-23 | LOW | ClickHouse backup stored on same host (no off-host replication) |
| RG-24 | LOW | No authenticated mainnet API call proven (DryRunSubmitter intercepts first) |
| RG-25 | LOW | No sustained mainnet soak (point-in-time proof only) |

### Severity Assessment

- **Critical gaps:** 0
- **High gaps:** 0
- **Medium gaps:** 0
- **Low gaps:** 6 (all new, all acceptable for dry-run authorization scope)

No gap escalation from prior waves. The 18 pre-existing LOW gaps from S431 remain unchanged.

## Formal Verdict

### Wave Evaluation

| Dimension | Result |
|-----------|--------|
| Blocker closure (B-1, B-2, B-3) | ALL CLOSED with full evidence |
| Operational proof (mainnet dry-run) | PROVEN on real endpoints |
| Safety invariant (zero real orders) | INTACT -- 5-layer fail-closed defense |
| Regression status | ZERO regressions |
| Residual gap profile | 6 new LOW gaps, 0 medium+ |

### Authorization Decision

**VERDICT: AUTHORIZED -- CONDITIONAL**

The Mainnet Enablement Wave (S432-S436) is **authorized to close** with the following determination:

1. **All three mainnet blockers are resolved** with concrete, test-backed evidence.
2. **Mainnet dry-run execution is proven** against real Binance endpoints for both Spot and Futures segments.
3. **Zero real orders were placed** on any exchange during the wave.
4. **Zero regressions** were introduced.
5. **No critical, high, or medium-severity gaps** were introduced.

### Conditions for Future Live-Trading Authorization Ceremony

The wave does NOT authorize live trading. A future live-trading authorization ceremony must satisfy:

| Condition | Rationale |
|-----------|-----------|
| C-1: Authenticated mainnet API call proven | DryRunSubmitter prevented this in S436 by design; must be proven before real submission |
| C-2: Credential provider upgraded to external secret manager | RG-20: env vars are insufficient for production mainnet credentials |
| C-3: ClickHouse backup automated and off-host | RG-22/RG-23: manual same-host backup is insufficient for production |
| C-4: Sustained mainnet soak (endurance proof) | RG-25: point-in-time proof must be extended to sustained operation |
| C-5: Kill-switch operational procedure documented and tested | Currently structural; needs operational runbook |
| C-6: Explicit removal of `dry_run=false` config rejection for mainnet | Requires deliberate source-code change in `schema.go` |

These conditions are **non-negotiable prerequisites** for any future ceremony that enables `dry_run=false` on mainnet adapters.

## Links

- Charter: [mainnet-enablement-wave-charter-and-scope-freeze.md](mainnet-enablement-wave-charter-and-scope-freeze.md)
- Capabilities: [mainnet-enablement-capabilities-questions-and-non-goals.md](mainnet-enablement-capabilities-questions-and-non-goals.md)
- Risk register: [mainnet-blockers-non-blockers-kv-history-decision-and-risk-register.md](mainnet-blockers-non-blockers-kv-history-decision-and-risk-register.md)
- Evidence matrix: [mainnet-authorization-evidence-matrix-blockers-residual-gaps-and-next-ceremony.md](mainnet-authorization-evidence-matrix-blockers-residual-gaps-and-next-ceremony.md)
- Stage report: [../stages/stage-s437-mainnet-authorization-evidence-gate-report.md](../stages/stage-s437-mainnet-authorization-evidence-gate-report.md)
