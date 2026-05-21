# Mainnet Enablement Wave -- Charter and Scope Freeze

## Wave Identity

| Field | Value |
|---|---|
| Wave | Mainnet Enablement |
| Phase | 49 |
| Charter Stage | S432 |
| Planned Stages | S433--S438 |
| Predecessor Wave | Production Hardening and Mainnet Readiness Audit (S427--S431) |
| Predecessor Verdict | PASS -- FULL DELIVERY |
| Date Opened | 2026-03-23 |

## Strategic Context

The Foundry has completed 12 consecutive wave passes since S370 with zero regressions:

1. Multi-binary orchestration (S370--S375): PASS
2. Exchange listening + dry-run (S376--S381): PASS
3. OMS Foundation (S382--S388): PASS
4. Binance segmentation (S390--S395): PASS
5. Unified segment runtime (S398--S403): PASS, FULL DELIVERY
6. Testnet venue execution, Spot-first (S404--S409): PASS, SUBSTANTIAL DELIVERY
7. Production readiness hardening (S410--S414): PASS, FULL DELIVERY
8. Futures venue execution proof (S415--S420): PASS, SUBSTANTIAL DELIVERY
9. Runtime simplification and consolidation (S416--S420): PASS, FULL DELIVERY
10. Futures venue execution proof, post-simplification (S421--S426): PASS, FULL DELIVERY
11. Production hardening and mainnet readiness audit (S427--S431): PASS, FULL DELIVERY

The S430 mainnet readiness audit identified exactly three blockers preventing a future mainnet authorization ceremony:

| ID | Blocker | Severity |
|---|---|---|
| B-1 | No mainnet adapter implementation | Critical |
| B-2 | No mainnet credential management | Critical |
| B-3 | No ClickHouse backup/restore strategy | High |

The system has zero medium-severity or higher residual gaps. All architectural decisions are rendered. The execution model (market-order-only, Binance-only, Spot + Futures on unified runtime) is proven and stable. The only work standing between the current state and a mainnet authorization ceremony is the resolution of B-1, B-2, B-3 and a mainnet dry-run proof.

This wave exists to resolve those three blockers, prove dry-run execution against real mainnet endpoints, and close with a formal mainnet authorization evidence gate. It does NOT enable live trading on mainnet.

## Wave Objective

1. Implement mainnet adapter variants for both Spot and Futures segments, following the proven testnet adapter pattern.
2. Integrate an external secret manager for mainnet credential lifecycle (storage, retrieval, rotation boundary).
3. Define and test a ClickHouse backup/restore procedure that protects the audit trail against infrastructure failure.
4. Prove dry-run execution against real mainnet endpoints with dry_run=true, demonstrating that the pipeline connects, authenticates, receives market data, and generates execution intents without submitting real orders.
5. Close the wave with a mainnet authorization evidence gate that renders a formal authorization verdict.

## Scope Freeze

### In Scope (Frozen)

The wave is organized into five execution blocks:

#### Block 1: Mainnet Adapter Readiness -- Spot and Futures (S433)

Resolve B-1. Implement mainnet adapter variants for both segments.

- Implement `binance_spot_mainnet` adapter following the proven `binance_spot_testnet` pattern.
- Implement `binance_futures_mainnet` adapter following the proven `binance_futures_testnet` pattern.
- Mainnet adapters must use production base URLs, production authentication headers, and production rate-limit awareness.
- Add a rate-limiter decorator in the adapter call chain to respect Binance mainnet API rate limits.
- Adapter selection must remain config-driven: `venue_adapter` config field determines which adapter is instantiated at boot.
- Validate that mainnet adapters satisfy the same `VenueAdapter` interface contract as testnet adapters.
- Prove via unit/integration tests that mainnet adapters are structurally correct (endpoint construction, auth header injection, response parsing).
- Do NOT execute against real mainnet endpoints in this stage. Structural proof only.

**Exit Criteria**: `binance_spot_mainnet` and `binance_futures_mainnet` adapters exist, pass interface compliance tests, and include rate-limiter decoration. B-1 resolved.

#### Block 2: Mainnet Secret Manager Integration (S434)

Resolve B-2. Replace environment-variable credential passing with external secret manager integration.

- Define a `CredentialProvider` interface that abstracts credential retrieval from the adapter layer.
- Implement at least one concrete provider: HashiCorp Vault, AWS Secrets Manager, or file-based encrypted store (operator choice at deployment).
- Environment-variable provider remains as a fallback for development/testnet use.
- Mainnet adapters must retrieve credentials via `CredentialProvider`, never from raw environment variables.
- Prove credential retrieval, error handling (secret not found, access denied, rotation mid-flight), and fail-closed behavior (adapter refuses to start if credentials are unavailable).
- Credential rotation boundary: the provider must support re-fetch on demand; the adapter is not required to hot-swap credentials during execution.

**Exit Criteria**: `CredentialProvider` interface defined and implemented. Mainnet adapters wired to use it. Fail-closed on missing credentials proven. B-2 resolved.

#### Block 3: ClickHouse Backup and Restore Proof (S435)

Resolve B-3. Define and test a ClickHouse backup/restore procedure.

- Define a backup schedule policy (frequency, retention depth, storage target).
- Implement or document a backup procedure using ClickHouse native tooling (`clickhouse-backup`, `BACKUP`/`RESTORE` SQL, or volume-level snapshot).
- Execute a restore test: backup, destroy data, restore, verify row count and data integrity for the executions table.
- Document recovery time objective (RTO) and recovery point objective (RPO) achievable with the chosen strategy.
- The backup/restore scope is the `executions` table (and any future execution-related tables). NATS KV and JetStream are excluded (operational state, not audit trail).

**Exit Criteria**: Backup procedure documented and tested. Restore test executed with verified data integrity. RTO/RPO documented. B-3 resolved.

#### Block 4: Mainnet Dry-Run Proof (S436)

Prove end-to-end dry-run execution against real mainnet endpoints.

- Deploy a mainnet-targeted compose configuration with `dry_run=true`.
- Mainnet adapters connect to real Binance mainnet API endpoints using real (scoped) API credentials via the secret manager.
- Ingest binary receives real mainnet market data (aggTrades or similar).
- Execute binary derives execution intents from real market data.
- DryRunSubmitter intercepts all venue calls, logging what would have been submitted without actually placing orders.
- Prove for both Spot and Futures segments:
  - Mainnet WebSocket connection established and receiving market data.
  - Execution intents generated from real price feeds.
  - DryRunSubmitter intercepts and logs dry-run fills with realistic price data.
  - KV and ClickHouse persistence operate normally with mainnet-sourced data.
  - Kill-switch and staleness guards function correctly.
- Prove fail-closed: if `dry_run` is omitted or null in a mainnet config, the system defaults to dry_run=true and refuses to submit real orders.

**Exit Criteria**: Mainnet dry-run execution proven for both Spot and Futures. Real market data ingested, intents generated, dry-run fills logged. No real orders placed. Safety controls verified.

#### Block 5: Mainnet Authorization Evidence Gate (S437)

Evaluate the wave against its charter and render a formal mainnet authorization verdict.

- Score each block's exit criteria with evidence grades.
- Verify zero regressions across the full test suite.
- Evaluate residual gaps and classify by severity.
- Render a wave verdict (PASS/FAIL with delivery classification).
- Render a mainnet authorization recommendation:
  - If all blockers resolved and dry-run proven: **AUTHORIZED FOR MAINNET DRY-RUN OPERATIONS**.
  - If any blocker remains unresolved: **NOT AUTHORIZED**, with explicit remaining prerequisites.
- The authorization verdict does NOT enable live trading. It authorizes the system to operate in dry-run mode against mainnet endpoints in a production-like deployment. Live trading enablement requires a separate, future authorization ceremony.

**Exit Criteria**: Evidence matrix produced. Wave verdict rendered. Mainnet authorization recommendation documented. Next-ceremony direction stated.

### Out of Scope (Frozen)

The following are explicitly excluded from this wave and must not be opened under any circumstance:

| Exclusion | Rationale |
|---|---|
| Live trading on mainnet | This wave proves dry-run only. Live trading requires a separate authorization ceremony after operational confidence is established. NG-1. |
| OMS expansion (limit orders, amendments, cancel API) | Market-order-only scope. Lifecycle model is frozen. NG-2. |
| Multi-exchange support | Binance-only scope. NG-3. |
| Advanced order types | Out of wave scope. NG-4. |
| Dashboard, UI, or alerting rule development | Operational signals remain HTTP/JSON-based. NG-5. |
| Config or compose surface re-expansion | 3+3 canonical surface preserved. New mainnet configs are additive within the canonical model. NG-6. |
| Large structural refactoring | Adapter pattern is proven; mainnet is a new instantiation, not a redesign. NG-7. |
| Portfolio risk management | Out of scope for execution engine. NG-8. |
| Multi-tenant or multi-operator deployment | Single-operator only. NG-9. |
| /fapi/v1/userTrades integration | Deferred (NB-8). Fee model is sufficient. NG-10. |
| Non-blocker resolution (NB-1 through NB-10) | Non-blockers have mitigations. The rate limiter (NB-1) is addressed as part of B-1 adapter work; remaining non-blockers are deferred. NG-11. |
| Documentation governance | Separate concern, no runtime impact. NG-12. |

## Dependency Chain

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

S433 and S434 are sequentially ordered because mainnet adapters must exist before they can be wired to a credential provider. S435 can proceed in parallel with S434 but is sequenced after S433 for focus. S436 depends on all three blocker-resolution stages. S437 evaluates the complete wave.

## Success Criteria

The wave passes if:

1. B-1 is resolved: mainnet adapters for Spot and Futures exist and pass interface compliance tests.
2. B-2 is resolved: credential provider is integrated and fail-closed behavior is proven.
3. B-3 is resolved: ClickHouse backup/restore is tested with verified data integrity.
4. Mainnet dry-run execution is proven for both Spot and Futures segments with real market data.
5. No real orders are placed on mainnet at any point during the wave.
6. The evidence gate renders a verdict with zero high-severity or medium-severity residual gaps introduced by this wave.

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| Mainnet API behavior differs from testnet | Adapters share the same interface contract; differences are isolated to URL, auth, and rate limits. Dry-run proof (S436) validates actual mainnet connectivity. |
| Credential provider adds boot-time latency | Fail-closed design: adapter refuses to start if credentials unavailable. Boot-time credential fetch is a one-time operation. |
| ClickHouse backup tooling incompatibility | Use ClickHouse native backup (BACKUP/RESTORE SQL) or proven third-party tool (clickhouse-backup). Test restore before declaring B-3 resolved. |
| Accidental real order submission on mainnet | 4-layer defense: dry_run config flag (fail-closed), DryRunSubmitter decorator, kill-switch gate, staleness guard. S436 explicitly verifies all four layers. |
| Rate limit violation during dry-run proof | Rate limiter decorator added in S433; dry-run proof uses minimal request frequency. |

## Ceremony Rules

- No stage may expand beyond its block definition without a charter amendment.
- Charter amendments require explicit justification and a documented decision in the stage report.
- The evidence gate (S437) must evaluate against this frozen scope, not against any informally expanded scope.
- All test evidence must be reproducible from the committed codebase.
- No real orders may be placed on mainnet during any stage of this wave. This is an inviolable safety constraint.
