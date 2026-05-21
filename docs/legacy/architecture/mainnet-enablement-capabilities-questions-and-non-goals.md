# Mainnet Enablement Wave -- Capabilities, Governing Questions, and Non-Goals

> Wave: Mainnet Enablement (Phase 49) | Charter: S432 | Date: 2026-03-23

---

## 1. Capabilities

Each capability maps to a blocker resolution or a proof requirement from the S430 mainnet readiness audit.

| ID | Capability | Block | Target Blocker |
|---|---|---|---|
| C-1 | Spot mainnet adapter implemented and interface-compliant | S433 | B-1 |
| C-2 | Futures mainnet adapter implemented and interface-compliant | S433 | B-1 |
| C-3 | Rate-limiter decorator integrated in mainnet adapter call chain | S433 | B-1 / NB-1 |
| C-4 | Config-driven adapter selection supports mainnet variants | S433 | B-1 |
| C-5 | CredentialProvider interface defined and abstracted | S434 | B-2 |
| C-6 | At least one concrete credential provider implemented | S434 | B-2 |
| C-7 | Mainnet adapters wired to CredentialProvider (no raw env vars) | S434 | B-2 |
| C-8 | Fail-closed on missing or invalid credentials | S434 | B-2 |
| C-9 | ClickHouse backup procedure defined and documented | S435 | B-3 |
| C-10 | ClickHouse restore tested with verified data integrity | S435 | B-3 |
| C-11 | RTO/RPO documented for chosen backup strategy | S435 | B-3 |
| C-12 | Mainnet Spot dry-run execution proven with real market data | S436 | Proof |
| C-13 | Mainnet Futures dry-run execution proven with real market data | S436 | Proof |
| C-14 | DryRunSubmitter intercepts all venue calls on mainnet config | S436 | Safety |
| C-15 | Fail-closed: omitted dry_run defaults to true on mainnet | S436 | Safety |
| C-16 | Kill-switch and staleness guard verified on mainnet pipeline | S436 | Safety |
| C-17 | Evidence gate verdict rendered with authorization recommendation | S437 | Gate |

---

## 2. Governing Questions

These questions guide execution decisions across the wave. Each must be answered with evidence before the evidence gate can render a verdict.

### Adapter Readiness (S433)

| ID | Question | Target |
|---|---|---|
| GQ-1 | Can mainnet adapters be instantiated from the same `VenueAdapter` interface without modifying the execution pipeline? | S433 |
| GQ-2 | What are the concrete differences between testnet and mainnet Binance endpoints (base URL, auth scheme, rate limits, response schema)? | S433 |
| GQ-3 | Is a token-bucket rate limiter sufficient for Binance mainnet API limits, or is a more complex strategy needed? | S433 |
| GQ-4 | Does config-driven adapter selection support mainnet variants without adding new config keys (i.e., the existing `venue_adapter` field is sufficient)? | S433 |

### Credential Management (S434)

| ID | Question | Target |
|---|---|---|
| GQ-5 | What is the minimal `CredentialProvider` interface that satisfies adapter needs without over-abstracting? | S434 |
| GQ-6 | Which concrete provider is the right first choice for the project's deployment model (Vault, AWS SM, file-based, or other)? | S434 |
| GQ-7 | Does the credential provider need to support hot-swap (mid-execution rotation), or is boot-time fetch sufficient? | S434 |
| GQ-8 | What is the correct fail-closed behavior when credentials become unavailable after boot (crash, degrade, or retry)? | S434 |

### Backup/Restore (S435)

| ID | Question | Target |
|---|---|---|
| GQ-9 | Which ClickHouse backup mechanism is the simplest that meets RTO/RPO requirements (native SQL, clickhouse-backup, volume snapshot)? | S435 |
| GQ-10 | What is an acceptable RTO and RPO for the executions table given 90-day TTL and current operational model? | S435 |
| GQ-11 | Does restore preserve ClickHouse TTL settings, or must they be re-applied post-restore? | S435 |

### Dry-Run Proof (S436)

| ID | Question | Target |
|---|---|---|
| GQ-12 | Does mainnet WebSocket connection require different handshake or keepalive behavior compared to testnet? | S436 |
| GQ-13 | Does the DryRunSubmitter correctly intercept mainnet adapter calls (same decorator chain as testnet)? | S436 |
| GQ-14 | Does mainnet market data produce valid execution intents through the existing derive pipeline without modification? | S436 |
| GQ-15 | Are the 4 safety layers (dry_run flag, DryRunSubmitter, kill-switch, staleness guard) independently verifiable on a mainnet deployment? | S436 |

### Evidence Gate (S437)

| ID | Question | Target |
|---|---|---|
| GQ-16 | Are all three blockers (B-1, B-2, B-3) resolved with evidence? | S437 |
| GQ-17 | Were zero real orders placed on mainnet during the entire wave? | S437 |
| GQ-18 | Does the system meet the authorization threshold for mainnet dry-run operations? | S437 |

---

## 3. Non-Goals

These items are explicitly excluded from the Mainnet Enablement Wave. Attempting to address any of these within this wave constitutes scope violation.

| ID | Non-Goal | Rationale |
|---|---|---|
| NG-1 | Live trading on mainnet (real order submission with dry_run=false) | This wave authorizes dry-run operations only. Live trading requires a separate, future authorization ceremony after operational confidence is established through sustained mainnet dry-run operation. |
| NG-2 | OMS expansion (limit orders, order amendments, cancel API) | The lifecycle model is frozen and proven for market orders. OMS expansion is a separate capability wave that should follow mainnet validation of the current model. |
| NG-3 | Multi-exchange support (beyond Binance) | The architecture supports extension, but this wave targets Binance-only. Multi-exchange is a separate initiative. |
| NG-4 | Advanced order types (stop-loss, trailing stop, OCO) | Out of scope. Market-order-only execution model is the proven baseline. |
| NG-5 | Dashboard, UI, or wide alerting rule development | Operational signals are exposed via HTTP/JSON endpoints (/statusz, /diagz). Dashboard development is an operational tooling concern, not a mainnet enablement prerequisite. |
| NG-6 | Config or compose surface re-expansion | The 3+3 canonical config and compose surface (S416--S418) is preserved. Mainnet-targeted configs are additive entries within the canonical model, not a surface expansion. |
| NG-7 | Large structural refactoring or architecture redesign | Mainnet adapters follow the proven testnet pattern. The credential provider is a new abstraction but is narrowly scoped. No pipeline redesign. |
| NG-8 | Portfolio risk management, position sizing, or capital controls | Capital controls were assessed in S430 and determined to be an operational layer concern, not an execution engine prerequisite. |
| NG-9 | Multi-tenant or multi-operator deployment model | The system is single-operator. Multi-tenancy is a separate architectural expansion. |
| NG-10 | /fapi/v1/userTrades integration for Futures commission | Deferred per NB-8 from S430. The canonical Fee/FeeAsset/CostBasis model is sufficient. |
| NG-11 | Resolution of non-blockers NB-2 through NB-10 | All non-blockers have existing mitigations. NB-1 (rate limiter) is partially addressed by C-3 within this wave. Remaining non-blockers are backlog items. |
| NG-12 | Documentation governance or index cleanup | 97 untracked docs (RG-16) have no runtime impact. Documentation ceremony is a separate concern. |

---

## 4. Capability-to-Question Mapping

| Capability | Governing Questions |
|---|---|
| C-1, C-2 | GQ-1, GQ-2 |
| C-3 | GQ-3 |
| C-4 | GQ-4 |
| C-5 | GQ-5, GQ-7 |
| C-6 | GQ-6 |
| C-7 | GQ-5, GQ-8 |
| C-8 | GQ-8 |
| C-9 | GQ-9, GQ-10 |
| C-10 | GQ-11 |
| C-11 | GQ-10 |
| C-12 | GQ-12, GQ-14 |
| C-13 | GQ-12, GQ-14 |
| C-14 | GQ-13 |
| C-15 | GQ-15 |
| C-16 | GQ-15 |
| C-17 | GQ-16, GQ-17, GQ-18 |

---

## 5. Blocker Resolution Traceability

| Blocker | Required Capabilities | Stage |
|---|---|---|
| B-1 (mainnet adapters) | C-1, C-2, C-3, C-4 | S433 |
| B-2 (credential management) | C-5, C-6, C-7, C-8 | S434 |
| B-3 (ClickHouse backup/restore) | C-9, C-10, C-11 | S435 |

The mainnet dry-run proof (S436) and evidence gate (S437) are not blocker resolutions; they are validation and authorization stages that depend on all blockers being resolved.

---

## 6. Limitations

- This document defines capabilities and questions for the Mainnet Enablement Wave only. It does not define capabilities for a future live-trading authorization wave.
- Governing questions may be refined during execution if mainnet behavior reveals unexpected constraints. Any question refinement must be documented in the relevant stage report and evaluated by the evidence gate.
- Non-goals are frozen. Reclassification of a non-goal as in-scope requires a charter amendment with explicit justification.
