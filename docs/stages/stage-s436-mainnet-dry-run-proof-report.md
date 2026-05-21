# Stage S436: Mainnet Dry-Run Proof

> Date: 2026-03-24 | Phase: 49 (Mainnet Enablement) | Prepares: S437 evidence gate

## Objective

Prove the platform can operate against Binance mainnet endpoints (Spot + Futures) in strict dry-run mode, with zero risk of real order submission. Transform abstract mainnet readiness (S433-S435) into operationally verified readiness.

## Context

After S433 (mainnet adapters, B-1), S434 (secret manager, B-2), and S435 (ClickHouse backup, B-3), all structural blockers for the Mainnet Enablement Wave are closed. S436 is the operational proof that ties these capabilities together against real mainnet endpoints without authorizing any real trading.

## Strategy

Exercise real Binance mainnet DNS, TLS, and HTTP reachability while proving that the DryRunSubmitter decorator chain prevents any order submission. Validate credential format enforcement, config fail-closed semantics, audit trail markers, and pipeline composition.

## Deliverables

### Code and Tests

| Artifact | Type | Description |
|----------|------|-------------|
| `internal/application/execution/live_mainnet_dryrun_test.go` | New | 8 test functions (MDR-1 through MDR-8) with `livemainnet` build tag |
| `internal/shared/settings/s436_mainnet_dryrun_config_test.go` | New | Config validation tests: mainnet dry_run enforcement, fail-closed defaults, VenueType helpers |
| `scripts/smoke-mainnet-dry-run.sh` | New | 4-phase smoke: config validation, connectivity, DryRunSubmitter interception, audit markers |

### Documentation

| Document | Description |
|----------|-------------|
| [`mainnet-dry-run-proof.md`](../architecture/mainnet-dry-run-proof.md) | Connectivity proof, DryRunSubmitter interception chain, endpoint selection, what was/was not proven |
| [`mainnet-dry-run-controls-endpoints-auditability-and-limitations.md`](../architecture/mainnet-dry-run-controls-endpoints-auditability-and-limitations.md) | 5-layer fail-closed chain, endpoint map, credential flow, audit markers, pipeline decorator diagram |

## Proof Results

### Config Validation Tests (Phase 1)

| Test | What It Proves | Result |
|------|---------------|--------|
| `TestS436_MainnetDryRunFalse_Rejected` (5 cases) | `dry_run=false` + mainnet → rejected | PASS |
| `TestS436_MainnetDryRunTrue_Accepted` (3 cases) | `dry_run=true` + mainnet → accepted | PASS |
| `TestS436_MainnetDryRunOmitted_DefaultsToTrue` | `dry_run` nil → defaults to true | PASS |
| `TestS436_VenueTypeMainnetHelpers` (5 cases) | IsMainnet, Environment, Segment helpers | PASS |
| `TestS436_IsDryRun_FailClosed` (3 cases) | Nil/true/false semantics | PASS |

### Live Mainnet Connectivity Tests (Phases 2-4, `livemainnet` tag)

| Test | What It Proves | Endpoint |
|------|---------------|----------|
| MDR-1: DNS + TCP | `api.binance.com` and `fapi.binance.com` resolve and accept TCP:443 | Both |
| MDR-2: TLS | Valid certificate chain, TLS 1.2+ handshake | Both |
| MDR-3: /ping | Public endpoints return HTTP 200 | Both |
| MDR-4: Spot interception | DryRunSubmitter intercepts Spot mainnet adapter | Spot |
| MDR-5: Futures interception | DryRunSubmitter intercepts Futures mainnet adapter | Futures |
| MDR-6: Audit markers | `dryrun-` prefix, `Simulated: true`, buy/sell/noop coverage | Spot |
| MDR-7: Credential format | Short/whitespace rejected, valid accepted (mainnet only) | N/A |
| MDR-8: Pipeline chain | adapter → RateLimiter → DryRunSubmitter, 5 sequential intents | Spot |

## What Was Proven

1. **Network reachability**: Both Binance mainnet endpoints (Spot + Futures) are DNS-resolvable, TCP-connectable, TLS-negotiable, and HTTP-responsive.
2. **DryRunSubmitter interception**: The decorator chain prevents any HTTP call to mainnet order endpoints. The inner adapter is composed but never invoked.
3. **Config enforcement**: No config combination can activate `dry_run=false` with mainnet adapters. Omitting `dry_run` defaults to `true` (fail-closed).
4. **Audit trail**: Every dry-run receipt carries unambiguous markers (`dryrun-` prefix, `Simulated: true`) that cannot be confused with real venue fills.
5. **Credential format validation**: Mainnet credentials undergo stricter validation (min-length, no whitespace) than testnet.
6. **Pipeline composition**: The full decorator chain (adapter → RateLimiter → DryRunSubmitter) composes and functions correctly.

## What Was NOT Proven

| Gap | Reason | Risk |
|-----|--------|------|
| Authenticated API call to mainnet | DryRunSubmitter intercepts first | LOW — adapter code identical to testnet |
| Rate limiter under real mainnet load | DryRunSubmitter intercepts first | LOW — proven in S433 unit tests |
| Extended soak against mainnet | Out of scope for S436 | ACCEPTED |
| Real order fill on mainnet | Prohibited by design | BY DESIGN |

## Limitations

1. Credentials are format-validated but not network-tested against mainnet auth.
2. Rate limiter is dormant during dry-run (proven in isolation).
3. Point-in-time proof, not sustained soak.
4. TLS certificates may rotate independently.

## Gate Readiness Assessment

S436 closes the operational proof gap between structural mainnet readiness (S433-S435) and the S437 evidence gate. The platform is now:

- **Structurally ready**: Adapters, credentials, config, and backup are in place.
- **Operationally verified**: Real mainnet endpoints are reachable, dry-run interception is proven.
- **Audit-clean**: Every dry-run receipt is unambiguously marked.
- **Fail-closed**: Five independent layers prevent accidental real order submission.

The wave is ready for S437 evidence gate evaluation.
