# Futures Venue Execution Proof -- Capabilities, Questions, Non-Goals, and Surface Constraints

**Wave:** Futures Venue Execution Proof (Post-Simplification)
**Phase:** 47
**Charter stage:** S421
**Date frozen:** 2026-03-23
**Canonical surface contract:** [`futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md`](futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md)

---

## 1. Governing Questions

### 1.1 Acceptance and Fill (S422)

| ID | Question | Target |
|---|---|---|
| **FV-Q1** | Does the `venue_live` write-path produce correct lifecycle transitions when the Futures venue accepts and fills a real order? | S422 |
| **FV-Q2** | Do fill records from a real Futures testnet response carry accurate `avgPrice`, `executedQty`, `cumQuote`, and commission data? | S422 |

### 1.2 Rejection (S423)

| ID | Question | Target |
|---|---|---|
| **FV-Q3** | Does the lifecycle correctly transition to `rejected` when the Futures venue returns a real rejection (insufficient margin, invalid quantity, invalid symbol, auth failure)? | S423 |
| **FV-Q4** | Does the `VenueOrderRejectedEvent` carry the real Futures venue error code, reason, and HTTP status? | S423 |

### 1.3 Partial Fill (S423)

| ID | Question | Target |
|---|---|---|
| **FV-Q5** | Can the system observe or structurally prove the `partially_filled` transition from a real Futures venue response? | S423 |
| **FV-Q6** | Does quantity monotonicity hold when real partial fills arrive from the Futures venue? | S423 |

### 1.4 Persistence and Read-Path (S424)

| ID | Question | Target |
|---|---|---|
| **FV-Q7** | Do KV projections, HTTP queries, and ClickHouse rows agree on terminal state after real Futures venue interactions? | S424 |
| **FV-Q8** | Does the ClickHouse rejection writer (wired in S411) produce correct rows for Futures rejection events without code changes? | S424 |

### 1.5 Compose and Sustained Operation (S424, S425)

| ID | Question | Target |
|---|---|---|
| **FV-Q9** | Does the full compose pipeline (derive -> execute -> store) operate correctly in `venue_live` mode against the Futures testnet on the unified runtime? | S425 |
| **FV-Q10** | Does the system sustain correct behavior over multiple consecutive Futures order cycles? | S424 |

### 1.6 Cross-Cutting (S422)

| ID | Question | Target |
|---|---|---|
| **FV-Q11** | Does the correlation chain (CorrelationID + CausationID) remain intact through real Futures venue interactions? | S422 |
| **FV-Q12** | Does the post-200 reconciliation path recover correctly when body-read fails after a real Futures 200? | S422 |

---

## 2. Capability Map

Each capability links to governing questions, target stage, and predecessor evidence.

| ID | Capability | Questions | Stage | Predecessor Evidence |
|---|---|---|---|---|
| **FV-C1** | Real Futures venue acceptance lifecycle | FV-Q1 | S422 | S385: venue_live write-path; S389: Futures adapter |
| **FV-C2** | Real Futures venue fill record fidelity | FV-Q2 | S422 | S384: PriceSource; Futures adapter unit tests |
| **FV-C3** | Real Futures venue rejection lifecycle | FV-Q3 | S423 | S386: rejection event path; adapter error mapping |
| **FV-C4** | Real Futures venue rejection event fidelity | FV-Q4 | S423 | S386: VenueOrderRejectedEvent contract |
| **FV-C5** | Real Futures venue partial fill lifecycle | FV-Q5, FV-Q6 | S423 | S383: partially_filled at domain level |
| **FV-C6** | Lifecycle invariant fidelity under real Futures data | FV-Q1--FV-Q6 | S422--S423 | S383: 49/49 transitions; S384: 8 invariant categories |
| **FV-C7** | Persistence consistency under real Futures data | FV-Q7, FV-Q8 | S424 | S387: KV + HTTP; S411: ClickHouse rejection writer |
| **FV-C8** | Read-path auditability and segment parity | FV-Q7 | S424 | S413: lifecycle list, composite status |
| **FV-C9** | Compose E2E with real Futures testnet | FV-Q9, FV-Q10 | S425 | S408: unified compose E2E Spot; S402: coexistence proof |
| **FV-C10** | Segment isolation under dual-segment live execution | FV-Q9 | S425 | S401: segment isolation; S408: compose E2E |

---

## 3. Non-Goals

These items are explicitly excluded from the wave. All 62 cumulative non-goals from prior waves are preserved. This section enumerates the full set, organized by domain, with new surface constraints marked.

### 3.1 Venue and Market Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-1** | Mainnet execution | Testnet only. Mainnet requires separate activation ceremony. |
| **NG-2** | Multi-venue support | Only Binance testnet. No second exchange, no venue routing beyond Binance. |
| **NG-3** | Multi-symbol execution | Single symbol per segment instance. No symbol multiplexing. |
| **NG-4** | Advanced order types | Market orders only. No limit, stop-loss, stop-market, OCO, trailing stop. |
| **NG-5** | WebSocket streaming fills | Synchronous REST-based fill retrieval only. |

### 3.2 OMS and Lifecycle Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-6** | Full OMS | No order book, no amendment, no cross-symbol aggregation. |
| **NG-7** | Cancel-order API | `cancelled` status mapped but no cancel HTTP call. |
| **NG-8** | Order amendment or replace | Fire-and-forget model. |
| **NG-9** | Lifecycle state machine extension | Seven-state machine is frozen. |
| **NG-10** | Multi-fill accumulation from WebSocket | REST-only fill observation. |

### 3.3 Risk, Portfolio, and Strategy Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-11** | Portfolio risk management | No position tracking, exposure limits, or margin management. |
| **NG-12** | P&L computation | No profit/loss. Fee realism deferred. |
| **NG-13** | Strategy optimization | Strategies produce intents; this wave validates execution. |

### 3.4 Infrastructure and Operations Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-14** | Dashboard or UI | No operational dashboards, no web UI. |
| **NG-15** | Alerting or paging | Observability via structured logs and NATS streams. |
| **NG-16** | CI/CD pipeline for testnet | Manual execution. |
| **NG-17** | Performance benchmarking | Correctness, not throughput. |
| **NG-18** | Credential rotation or vault integration | Env vars only. |

### 3.5 Architectural Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-19** | Lifecycle redesign | State machine proven and frozen. |
| **NG-20** | Actor topology changes | No new actors or supervision restructuring. |
| **NG-21** | New NATS streams or KV buckets | No new infrastructure beyond what exists. |
| **NG-22** | Domain model extension | `ExecutionIntent` is frozen. |

### 3.6 Segmentation Scope (preserved from S404)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-23** | Re-open segmentation wave | Segmentation is closed (S395 PASS). No venue model, adapter boundary, or config redesign. |
| **NG-24** | Per-segment control gate | Global gate is conservative and sufficient. |
| **NG-25** | Shared core extraction | ~120 lines duplication between adapters is acceptable. Extract when third adapter justifies. |
| **NG-26** | Activation surface segment queryability | Startup logging is sufficient. |
| **NG-27** | Multi-exchange adapters | No second exchange. |

### 3.7 Runtime Scope (preserved from S404)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-28** | Runtime architecture redesign | Unified runtime is stable. This wave consumes it, not modifies it. |
| **NG-29** | Per-segment dry_run control | Global `dry_run` flag applies uniformly. |
| **NG-30** | Per-segment kill switch | Global kill switch sufficient. |
| **NG-31** | Concurrent `venue_live` on both segments simultaneously | Proving Futures `venue_live` with Spot in `dry_run` or disabled is sufficient. Simultaneous dual-segment `venue_live` is a follow-on concern. |
| **NG-32** | Schema changes to ClickHouse | 20-column schema is segment-agnostic. No DDL changes. |

### 3.8 Futures-Specific Exclusions

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-33** | Leverage configuration | Default leverage applies. No leverage adjustment API calls. |
| **NG-34** | Position mode switching | Default one-way mode. No hedge mode. |
| **NG-35** | Margin type management | Default cross margin. No margin type switching API calls. |
| **NG-36** | Funding rate impact analysis | Funding rates have no bearing on order lifecycle proof. |
| **NG-37** | Liquidation handling | Liquidation events are venue-initiated. Out of scope. |
| **NG-38** | Mark price or index price tracking | Not relevant to order lifecycle proof. |
| **NG-39** | Multi-asset margin mode | Default single-asset mode. |
| **NG-40** | Income/trade history API | Execution proof uses order API only. |

### 3.9 Runtime Simplification Scope (preserved from S416--S420)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-41** | Re-open config surface | Config consolidated at S416. No new execute configs. |
| **NG-42** | Production code changes to execution runtime | Proof wave consumes runtime, not modifies it. |
| **NG-43** | Settings schema structural refactor | Schema is stable. Only comment/label changes permitted. |
| **NG-44** | Broad smoke script refactoring | Scripts consolidated at S417. Use canonical scripts. |
| **NG-45** | Segment routing logic changes | `SegmentRouter` is frozen. |
| **NG-46** | Per-segment compose overlays | Unified compose model is canonical. No per-segment overlays. |
| **NG-47** | Re-open compose surface | Compose consolidated at S417. No new overlays. |
| **NG-48** | Broad test refactoring or parameterization | Test consolidation deferred. Stage-prefixed tests retained for unique invariants. |

### 3.10 Surface Constraints (NEW -- post-simplification discipline)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-49** | New "temporary" compose file | No temporary compose files without fortissimo justification and removal plan before S426. |
| **NG-50** | New "Futures-only" config file | No per-segment config file. Use canonical `execute-venue-live.jsonc` with segment toggle. |
| **NG-51** | New NATS subjects or KV buckets for Futures | All infrastructure is segment-agnostic. No Futures-specific infra. |
| **NG-52** | Taxonomy regression | No reintroduction of "legacy" labels or deprecated terminology. |
| **NG-53** | Fee normalization during proof | G-4 monitored but not resolved. Normalization is a production-readiness concern. |
| **NG-54** | Parallel segment live proof | Each segment proven independently. Simultaneous dual `venue_live` is deferred. |
| **NG-55** | Documentation governance ceremony | 97 untracked docs remain deferred. Not addressed in this wave. |

**Total non-goals: 55 (40 inherited + 8 simplification-preserved + 7 new surface constraints).**

---

## 4. Surface Constraints Detail

The canonical surface contract in the companion charter document (Section 5) establishes the following constraints that every stage in this wave MUST respect:

### 4.1 Config Constraint

```
ALLOWED configs for this wave:
  deploy/configs/execute.jsonc           -- paper simulator (not used in proof)
  deploy/configs/execute-unified.jsonc   -- dry-run testing
  deploy/configs/execute-venue-live.jsonc -- PROOF EXECUTION

FORBIDDEN:
  Any new execute config file.
  Any per-segment config file.
  Any "temporary" config file.
```

### 4.2 Compose Constraint

```
ALLOWED compose files for this wave:
  deploy/compose/docker-compose.yaml           -- base stack (always)
  deploy/compose/docker-compose.unified.yaml   -- dry-run overlay (testing)
  deploy/compose/docker-compose.venue-live.yaml -- venue-live overlay (PROOF)

FORBIDDEN:
  Any new compose overlay file.
  Any per-segment compose overlay.
  Any "temporary" compose file.
```

### 4.3 Runtime Constraint

```
FROZEN (no modifications):
  Execute binary topology
  SegmentRouter dispatch logic
  NATS subject structure
  KV bucket structure
  ClickHouse table schema
  Lifecycle state machine (7 states)
  Adapter interfaces

ALLOWED:
  Bug fixes in existing adapter implementations.
  New test files validating Futures behavior.
  New smoke scripts referencing canonical artifacts.
  Log-level or config-comment adjustments.
```

### 4.4 Deviation Protocol

If a stage encounters a genuine need to deviate from these constraints:

1. Document the need in the stage report with concrete justification.
2. Verify no non-goal violation.
3. Propose a removal plan (deviation must not persist past S426).
4. Record the deviation as a residual gap for the evidence gate.

---

## 5. Boundary Conditions

### 5.1 Futures Testnet Credential Requirements

- `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET` must be provisioned before S422.
- Credentials are never committed to the repository.
- Futures testnet account must have sufficient USDT balance for market order execution.

### 5.2 Futures Testnet Account State

- Account leverage: default (20x for most symbols). No modification in this wave (NG-33).
- Account margin type: cross margin (default). No modification (NG-35).
- Account position mode: one-way (default). No modification (NG-34).
- Sufficient USDT balance must be verified before each execution stage.

### 5.3 Partial Fill Feasibility

Futures market orders may produce partial fills when order size exceeds available liquidity at the best price level. This is more likely on Futures testnet than Spot testnet. The wave accepts:

- **Preferred:** Real partial fill observation from Futures testnet.
- **Acceptable:** Structural proof combining domain-level evidence (S383) with Futures adapter response parsing tests and `mapBinanceStatus("PARTIALLY_FILLED")` mapping demonstration.

### 5.4 Post-200 Reconciliation

Structural proof (code inspection + existing httptest coverage via `Post200Reconciler`) is acceptable if the scenario cannot be reliably triggered against the Futures testnet.

---

## 6. Relationship to Prior Waves

### 6.1 Spot Proof Parity (S404--S409)

This wave mirrors the Spot proof wave structurally but targets the Futures segment:

| Dimension | Spot Proof (S404--S409) | Futures Proof (S421--S426) |
|---|---|---|
| Venue target | Binance Spot testnet | Binance Futures testnet |
| Adapter | `BinanceSpotTestnetAdapter` | `BinanceFuturesTestnetAdapter` |
| Response model | `fills[]` array | Top-level `avgPrice`/`cumQuote` |
| Partial fill likelihood | Low | Medium |
| Margin relevance | None | Rejection source |
| New: canonical surface contract | Not applicable (pre-simplification) | Required (Section 4) |
| New: segment parity | Not applicable (single segment) | Required (FV-C8) |

### 6.2 Runtime Simplification Inheritance (S416--S420)

This wave inherits and enforces the consolidation achieved by the Runtime Simplification wave:

| Simplification Result | Inheritance in This Wave |
|---|---|
| 3 canonical configs | NG-41, NG-50: no new configs |
| 3 canonical compose files | NG-46, NG-47, NG-49: no new overlays |
| "Standalone" taxonomy | NG-52: no taxonomy regression |
| Zero deprecated references | Verified at gate |
| Fail-closed validation | Consumed, not modified |

---

## 7. Links

| Reference | Link |
|---|---|
| Charter and canonical surface contract | [`futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md`](futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md) |
| S420 authorization | [`runtime-simplification-evidence-gate-and-futures-proof-authorization.md`](runtime-simplification-evidence-gate-and-futures-proof-authorization.md) |
| Prior Futures charter (S415) | [`futures-venue-execution-proof-wave-charter-and-scope-freeze.md`](futures-venue-execution-proof-wave-charter-and-scope-freeze.md) |
| Prior Futures capabilities/non-goals | [`futures-venue-execution-capabilities-questions-and-non-goals.md`](futures-venue-execution-capabilities-questions-and-non-goals.md) |
| Config reference | [`../../deploy/configs/CONFIG-REFERENCE.md`](../../deploy/configs/CONFIG-REFERENCE.md) |
| Canonical order model | [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md) |
| Stage report | [`../stages/stage-s421-futures-venue-execution-charter-report.md`](../stages/stage-s421-futures-venue-execution-charter-report.md) |
