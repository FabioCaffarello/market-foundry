# Testnet Venue Execution Proof (Unified Runtime, Spot-First) -- Capabilities, Questions, and Non-Goals

**Wave:** Testnet Venue Execution Proof (unified runtime)
**Charter stage:** S404
**Prior versions:** S389 (original), S396 (segmented refresh)
**Date frozen:** 2026-03-22

---

## 1. Governing Questions (Retargeted to Unified Runtime)

The 12 governing questions are preserved verbatim from S389. The only change
across charter refreshes has been the target stage. This version retargets to
the unified runtime architecture established by S398--S403.

### 1.1 Acceptance and Fill (S405 -- Spot on Unified Runtime)

| ID | Question | Target |
|---|---|---|
| **TV-Q1** | Does the `venue_live` write-path produce correct lifecycle transitions when the Spot venue accepts and fills a real order? | S405 |
| **TV-Q2** | Do fill records from a real Spot testnet response carry accurate prices, quantities, and fees? | S405 |

**Unified runtime context:** The Spot adapter is one of potentially multiple
adapters projected by the unified binary. `venue_live` mode means `dry_run=false`
in the unified config. The DryRunSubmitter is bypassed; the real
`BinanceSpotTestnetAdapter.Submit()` is called. The decorator pipeline
(Post200Reconciler, RetrySubmitter) wraps the Spot adapter as configured.

### 1.2 Rejection (S406 -- Spot on Unified Runtime)

| ID | Question | Target |
|---|---|---|
| **TV-Q3** | Does the lifecycle correctly transition to `rejected` when the Spot venue returns a real rejection (insufficient balance, invalid params, auth failure)? | S406 |
| **TV-Q4** | Does the `VenueOrderRejectedEvent` carry the real Spot venue rejection code, reason, and HTTP status? | S406 |

**Unified runtime context:** Rejection events flow through the unified
pipeline. The `VenueAdapterRouter` dispatches to the Spot adapter; rejection
from Spot HTTP response propagates back through the same decorator chain and
event publishing path as in dry-run mode, but with real venue error payloads.

### 1.3 Partial Fill (S406 -- Spot on Unified Runtime)

| ID | Question | Target |
|---|---|---|
| **TV-Q5** | Can the system observe or structurally prove the `partially_filled` transition from a real Spot venue response? | S406 |
| **TV-Q6** | Does quantity monotonicity hold when real partial fills arrive from the Spot venue? | S406 |

**Feasibility policy (preserved from S396):**

Spot market orders on testnet typically fill instantly. The wave accepts:

- **Preferred:** Real partial fill observation from Spot testnet.
- **Acceptable:** Structural proof combining:
  1. S383 domain-level transition evidence (49/49 including `partially_filled`).
  2. S394 `BinanceSpotTestnetAdapter` multi-fill aggregation unit tests.
  3. `mapBinanceStatus("PARTIALLY_FILLED")` -> correct lifecycle transition.
  4. S384 quantity monotonicity invariant tests.

### 1.4 Persistence and Read-Path (S407 -- Spot on Unified Runtime)

| ID | Question | Target |
|---|---|---|
| **TV-Q7** | Do KV projections, HTTP queries, and ClickHouse rows agree on terminal state after real Spot venue interactions? | S407 |
| **TV-Q8** | Is the ClickHouse rejection writer wired and producing correct rows (RG-1 closure)? | S407 |

**Unified runtime context:** The store binary reads events from NATS subjects
that are source-scoped (`binances.*`). The unified runtime's source routing
(S401) ensures events land on the correct subjects. The read-path queries KV
projections and HTTP endpoints that were proven in S387 under simulated data;
S407 proves them under real data.

**RG-1 status:** The ClickHouse rejection writer was identified as a gap in
S388 (OMS Foundation evidence gate). It has not been wired in any subsequent
stage. S407 explicitly targets its closure or documents a justified deferral.

### 1.5 Compose and Sustained Operation (S407--S408 -- Unified Runtime)

| ID | Question | Target |
|---|---|---|
| **TV-Q9** | Does the full compose pipeline (derive -> execute -> store) operate correctly in `venue_live` mode against the Spot testnet? | S408 |
| **TV-Q10** | Does the system sustain correct behavior over multiple consecutive order cycles against the Spot testnet? | S407 |

**Unified runtime context:** S402 proved compose coexistence under dry-run.
S408 elevates this to `venue_live` mode with real Spot venue interaction. The
compose stack uses `execute-unified.jsonc` with Spot enabled and `dry_run=false`.

### 1.6 Cross-Cutting (S405 -- Spot on Unified Runtime)

| ID | Question | Target |
|---|---|---|
| **TV-Q11** | Does the correlation chain (CorrelationID + CausationID) remain intact through real Spot venue interactions? | S405 |
| **TV-Q12** | Does the post-200 reconciliation path recover correctly when body-read fails after a real Spot 200? | S405 |

**TV-Q12 evidence policy (preserved from S389/S396):** Structural proof
(code inspection + existing `httptest` coverage) is acceptable if the scenario
cannot be reliably triggered against the real Spot testnet.

---

## 2. Capability Map (Retargeted to Unified Runtime)

Each capability links to governing questions and target stages on the unified
runtime.

| ID | Capability | Questions | Stage | Predecessor evidence |
|---|---|---|---|---|
| **TV-C1** | Real Spot venue acceptance lifecycle | TV-Q1 | S405 | S385: write-path; S394: Spot adapter; S400: multi-adapter projection |
| **TV-C2** | Real Spot venue fill record fidelity | TV-Q2 | S405 | S384: PriceSource; S394: Spot multi-fill; S399: unified config |
| **TV-C3** | Real Spot venue rejection lifecycle | TV-Q3 | S406 | S386: rejection path; S394: Spot error mapping; S401: source routing |
| **TV-C4** | Real Spot venue rejection event fidelity | TV-Q4 | S406 | S386: VenueOrderRejectedEvent; S394: Spot auth errors |
| **TV-C5** | Real Spot venue partial fill lifecycle | TV-Q5, TV-Q6 | S406 | S383: domain transitions; S394: multi-fill aggregation |
| **TV-C6** | Lifecycle invariant fidelity under real data | TV-Q1--TV-Q6 | S405--S407 | S383: 49/49; S384: 8 invariant categories; S403: unified runtime stable |
| **TV-C7** | Persistence consistency under real data | TV-Q7, TV-Q8 | S407 | S387: KV + HTTP; RG-1 open |
| **TV-C8** | Post-200 reconciliation under real conditions | TV-Q12 | S405 | S322: Post200Reconciler; structural proof policy |
| **TV-C9** | Compose E2E with real Spot testnet | TV-Q9, TV-Q10 | S407--S408 | S402: unified compose dry-run; S380: E2E dry-run |
| **TV-C10** | OMS read-path auditability under real data | TV-Q7 | S407 | S387: composite status query; S401: source-scoped subjects |

---

## 3. Non-Goals

These items are explicitly excluded from the wave. The 28 non-goals from S396
are preserved. Additional non-goals are added to reflect the post-unified-runtime
context.

### 3.1 Venue and Market Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-1** | Mainnet execution | Testnet only. Mainnet requires separate activation ceremony. |
| **NG-2** | Multi-venue support | Only Binance testnet. No second exchange. |
| **NG-3** | Multi-symbol execution | Single symbol per execution cycle. No symbol multiplexing. |
| **NG-4** | Advanced order types | Market orders only. No limit, stop-loss, OCO. |
| **NG-5** | WebSocket streaming fills | Synchronous REST-based fill retrieval only. |

### 3.2 OMS and Lifecycle Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-6** | Full OMS | No order book, no amendment, no cross-symbol aggregation. |
| **NG-7** | Cancel-order API | `cancelled` status mapped but no cancel HTTP call. |
| **NG-8** | Order amendment or replace | Fire-and-forget model. |
| **NG-9** | Lifecycle state machine extension | Seven-state machine is frozen. |
| **NG-10** | Multi-fill accumulation from WebSocket | REST-only partial fill observation. |

### 3.3 Risk, Portfolio, and Strategy Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-11** | Portfolio risk management | No position tracking, exposure limits, or margin management. |
| **NG-12** | P&L computation | No profit/loss. Fee realism deferred. |
| **NG-13** | Strategy optimization | Strategies produce intents; this wave validates execution. |

### 3.4 Infrastructure and Operations Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-14** | Dashboard or UI | No operational dashboards, no web UI. |
| **NG-15** | Alerting or paging | Observability via structured logs and NATS streams. |
| **NG-16** | CI/CD pipeline for testnet | Manual execution. |
| **NG-17** | Performance benchmarking | Correctness, not throughput. |
| **NG-18** | Credential rotation or vault integration | Env vars only. |

### 3.5 Architectural Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-19** | Lifecycle redesign | State machine proven and frozen (S383). |
| **NG-20** | Actor topology changes | No new actors or supervision changes beyond RG-1. |
| **NG-21** | New NATS streams or KV buckets | No new infra beyond RG-1 wiring. |
| **NG-22** | Domain model extension | `ExecutionIntent` is frozen. |

### 3.6 Segmentation Scope (preserved from S396)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-23** | Parallel Futures testnet proof | Futures proof is deferred to a follow-on wave. This wave answers TV-Q1--TV-Q12 for Spot only. |
| **NG-24** | Reopen segmentation wave | Segmentation is closed (S395 PASS). No venue model, adapter boundary, or config enablement redesign. |
| **NG-25** | Per-segment control gate | S395 G2 / S403: global gate is conservative and sufficient. |
| **NG-26** | Shared core extraction | S395 G5: ~120 lines duplication acceptable. Extract when third adapter justifies. |
| **NG-27** | Activation surface segment queryability | S395 G4: startup logging is sufficient. API/KV exposure is observability enhancement. |
| **NG-28** | Multi-exchange adapters | No second exchange. Segmentation is Binance-internal. |

### 3.7 Unified Runtime Scope (NEW -- post-S403)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-29** | Unified runtime redesign | The unified runtime (S398--S403) is a consumed foundation. This wave proves venue execution on it; it does not modify it. |
| **NG-30** | Per-segment dry_run toggle | `dry_run` remains global and top-level (S403 invariant, USR-Q11 FULL). Config for `venue_live` disables Futures or runs Spot-only. |
| **NG-31** | Per-segment kill switch | S403 residual: deferred, not needed for Spot testnet proof. |
| **NG-32** | Segmented metrics | S403 residual: deferred, observability concern. |
| **NG-33** | Multi-consumer per binary | S403 residual: single consumer sufficient for proof. |
| **NG-34** | Concurrent Spot + Futures `venue_live` | This wave proves Spot `venue_live` only. Futures remains dry-run or disabled. No concurrent real trading across segments. |
| **NG-35** | Config schema changes to unified model | Unified config model is frozen (S399). No new segment fields, no new validation rules. |

---

## 4. Boundary Conditions

### 4.1 Spot Testnet Credential Requirements

- `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY` and
  `MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET` must be provisioned before S405.
- Credentials are never committed to the repository.
- Spot testnet account must have sufficient test balance for market order
  execution.
- Top-up procedure must be documented before S405 begins.

### 4.2 Unified Config for `venue_live` Spot-Only

The unified config for this wave will use one of these patterns:

**Pattern A -- Spot-only config:**
```jsonc
{
  "venue": {
    "dry_run": false,
    "segments": {
      "spot": {
        "enabled": true,
        "type": "binance_spot_testnet",
        "source": "binances"
      }
    }
  }
}
```

**Pattern B -- Spot live + Futures disabled:**
```jsonc
{
  "venue": {
    "dry_run": false,
    "segments": {
      "spot": {
        "enabled": true,
        "type": "binance_spot_testnet",
        "source": "binances"
      },
      "futures": {
        "enabled": false,
        "type": "binance_futures_testnet",
        "source": "binancef"
      }
    }
  }
}
```

The exact pattern will be established in S405 based on validation behavior.
Both patterns must satisfy the global `dry_run=false` invariant: when
`dry_run=false`, ALL enabled segment adapters submit real orders. Therefore,
only Spot should be enabled for this wave.

### 4.3 Partial Fill Feasibility (Spot)

Same policy as S396 Section 4.3. Structural proof is acceptable.

### 4.4 Post-200 Reconciliation

Same policy as S389/S396. Structural proof is acceptable.

### 4.5 RG-1 (ClickHouse Rejection Writer)

This gap was identified in S388 and has persisted through S389--S403. S407
targets its explicit closure. If closure requires infrastructure changes beyond
the scope boundary (NG-20, NG-21), a justified deferral with mitigation plan
is acceptable for the evidence gate.

---

## 5. Differences from Prior Charters

| Dimension | S389 Original | S396 Refresh | S404 This Charter |
|---|---|---|---|
| Runtime model | Single binary, single adapter | Multi-binary, per-segment | **Unified: single binary, multi-adapter** |
| Venue target | Futures only | Spot-first | **Spot-first (preserved)** |
| Adapter | `BinanceFuturesTestnetAdapter` | `BinanceSpotTestnetAdapter` | **`BinanceSpotTestnetAdapter` via unified router** |
| Config model | Single `venue.type` | Per-segment files | **Unified `venue.segments.*`** |
| Compose model | Single instance | Dual-instance | **Unified single compose** |
| Stage range | S390--S395 | S397--S401 | **S405--S409** |
| Non-goals | 22 (NG-1--NG-22) | 28 (NG-1--NG-28) | **35 (NG-1--NG-35)** |
| Governing questions | 12 (unchanged) | 12 (retargeted) | **12 (retargeted again)** |
| Capabilities | 10 (unchanged) | 10 (retargeted) | **10 (retargeted again)** |
| Foundation waves | OMS only | OMS + Segmentation | **OMS + Segmentation + Unified Runtime** |

---

## 6. Links

| Reference | Link |
|---|---|
| Charter | [`testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md`](testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md) |
| S396 charter refresh (superseded) | [`testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md`](testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md) |
| S389 original charter (superseded) | [`testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](testnet-venue-execution-proof-wave-charter-and-scope-freeze.md) |
| S396 capabilities/non-goals (superseded) | [`testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md`](testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md) |
| S389 capabilities/non-goals (superseded) | [`testnet-venue-execution-capabilities-questions-and-non-goals.md`](testnet-venue-execution-capabilities-questions-and-non-goals.md) |
| Unified runtime evidence gate | [`unified-segment-runtime-evidence-gate.md`](unified-segment-runtime-evidence-gate.md) |
| Unified runtime evidence matrix | [`unified-segment-runtime-evidence-matrix-residual-gaps-and-next-ceremony.md`](unified-segment-runtime-evidence-matrix-residual-gaps-and-next-ceremony.md) |
| Stage report | [`../stages/stage-s404-testnet-venue-execution-unified-runtime-charter-report.md`](../stages/stage-s404-testnet-venue-execution-unified-runtime-charter-report.md) |
