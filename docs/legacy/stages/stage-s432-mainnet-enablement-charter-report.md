# Stage S432 -- Mainnet Enablement Wave Charter Report

## Stage Identity

| Field | Value |
|---|---|
| Stage | S432 |
| Type | Charter and scope freeze |
| Wave | Mainnet Enablement (Phase 49) |
| Predecessor | S431 (Production Hardening and Mainnet Readiness Audit -- PASS, FULL DELIVERY) |
| Date | 2026-03-23 |

## Executive Summary

S432 opens the Mainnet Enablement Wave as the 13th macro-front in the Foundry's development sequence. This wave transforms the three explicit mainnet blockers identified in S430 into a formal, scope-frozen execution plan with five blocks: mainnet adapter readiness, secret manager integration, ClickHouse backup/restore proof, mainnet dry-run proof, and a mainnet authorization evidence gate.

The wave does NOT enable live trading on mainnet. It produces the infrastructure, credentials management, data protection, and dry-run validation necessary for a future live-trading authorization ceremony.

## Rationale

The S431 evidence gate closed 12 consecutive wave passes with:

- Zero medium-severity or higher residual gaps (first time since gap tracking began).
- Complete Spot + Futures testnet execution proven on unified runtime.
- Formal mainnet readiness audit with 21 dimensions assessed.
- All deferred decisions rendered (KV history, fee normalization, capital controls).
- Three explicit blockers (B-1, B-2, B-3) as the only items between the current state and mainnet readiness.

Further testnet hardening offers diminishing returns. The shortest path to production value is resolving the three blockers and proving mainnet connectivity in dry-run mode.

## Deliverables

| Artifact | Path | Status |
|---|---|---|
| Wave charter and scope freeze | [`mainnet-enablement-wave-charter-and-scope-freeze.md`](../architecture/mainnet-enablement-wave-charter-and-scope-freeze.md) | Delivered |
| Capabilities, questions, and non-goals | [`mainnet-enablement-capabilities-questions-and-non-goals.md`](../architecture/mainnet-enablement-capabilities-questions-and-non-goals.md) | Delivered |
| Stage report | This document | Delivered |

## Wave Structure

### Blocks and Stages

| Stage | Block | Objective | Resolves |
|---|---|---|---|
| S432 | Charter | Scope freeze, capabilities, questions, non-goals | Wave authorization |
| S433 | Mainnet Adapter Readiness | Spot + Futures mainnet adapters with rate limiter | B-1 |
| S434 | Secret Manager Integration | CredentialProvider interface + concrete implementation | B-2 |
| S435 | ClickHouse Backup/Restore | Backup procedure, restore test, RTO/RPO | B-3 |
| S436 | Mainnet Dry-Run Proof | E2E dry-run on real mainnet endpoints, both segments | Proof |
| S437 | Mainnet Authorization Evidence Gate | Wave verdict + authorization recommendation | Gate |

### Dependency Chain

```
S432 (charter) --> S433 (mainnet adapters)
                       |
                       +--> S434 (secret manager)
                       |       |
                       +-------+--> S435 (ClickHouse backup)
                                       |
                                       +--> S436 (mainnet dry-run proof)
                                               |
                                               +--> S437 (evidence gate)
```

### Capabilities Summary

17 capabilities chartered across 5 blocks:

- S433: 4 capabilities (C-1 through C-4) -- adapter implementation and config
- S434: 4 capabilities (C-5 through C-8) -- credential abstraction and fail-closed
- S435: 3 capabilities (C-9 through C-11) -- backup, restore, RTO/RPO
- S436: 5 capabilities (C-12 through C-16) -- dry-run proof and safety verification
- S437: 1 capability (C-17) -- evidence gate verdict

### Governing Questions Summary

18 governing questions across 5 areas:

- Adapter readiness: GQ-1 through GQ-4
- Credential management: GQ-5 through GQ-8
- Backup/restore: GQ-9 through GQ-11
- Dry-run proof: GQ-12 through GQ-15
- Evidence gate: GQ-16 through GQ-18

### Non-Goals Summary

12 non-goals frozen:

| ID | Non-Goal |
|---|---|
| NG-1 | Live trading on mainnet |
| NG-2 | OMS expansion |
| NG-3 | Multi-exchange support |
| NG-4 | Advanced order types |
| NG-5 | Dashboard/UI/alerting rules |
| NG-6 | Config/compose surface re-expansion |
| NG-7 | Large structural refactoring |
| NG-8 | Portfolio risk management |
| NG-9 | Multi-tenant deployment |
| NG-10 | /fapi/v1/userTrades integration |
| NG-11 | Non-blocker resolution (NB-2 through NB-10) |
| NG-12 | Documentation governance |

## Blocker Traceability

| Blocker | Source | Severity | Resolution Stage | Capabilities |
|---|---|---|---|---|
| B-1 | S430 audit | Critical | S433 | C-1, C-2, C-3, C-4 |
| B-2 | S430 audit | Critical | S434 | C-5, C-6, C-7, C-8 |
| B-3 | S430 audit | High | S435 | C-9, C-10, C-11 |

## Residual State Entering This Wave

### Residual Gaps (18 total, all LOW)

Carried from S431. No new gaps introduced by this charter stage.

| Category | Count |
|---|---|
| Medium+ severity | 0 |
| Low severity (carried) | 13 |
| Low severity (from S427-S431 wave) | 5 |

### Mainnet Blockers (3 total)

All three are targeted for resolution in this wave.

### Non-Blockers (10 total, NB-1 through NB-10)

NB-1 (rate limiter) is partially addressed by C-3 in this wave. Remaining non-blockers are deferred.

## Inviolable Safety Constraint

**No real orders may be placed on mainnet during any stage of this wave.**

The entire wave operates under dry_run=true for any mainnet-targeted deployment. The 4-layer safety defense (dry_run config flag, DryRunSubmitter decorator, kill-switch gate, staleness guard) must be independently verified in S436 before the evidence gate can render a verdict.

## Success Criteria

The wave passes if:

1. All three blockers (B-1, B-2, B-3) are resolved with evidence.
2. Mainnet dry-run execution is proven for both Spot and Futures with real market data.
3. Zero real orders are placed on mainnet.
4. Zero high-severity or medium-severity gaps are introduced.
5. The evidence gate renders a formal mainnet authorization recommendation.

## Preparation for S433

The next stage (S433: Mainnet Adapter Readiness) should:

1. Read the existing testnet adapter implementations:
   - `internal/application/execution/binance_spot_testnet_adapter.go`
   - `internal/application/execution/binance_futures_testnet_adapter.go`
2. Identify the concrete differences between testnet and mainnet Binance API:
   - Base URLs (api.binance.com vs testnet.binance.vision)
   - Authentication headers (same HMAC scheme, different key scoping)
   - Rate limits (testnet lenient, mainnet strict)
   - Response schema (expected to be identical)
3. Design the rate-limiter decorator (token-bucket recommended).
4. Implement both mainnet adapters following the testnet pattern.
5. Write interface compliance and structural tests.
6. Do NOT connect to real mainnet endpoints in S433. Structural proof only.
