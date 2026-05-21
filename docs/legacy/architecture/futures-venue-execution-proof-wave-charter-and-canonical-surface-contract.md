# Futures Venue Execution Proof Wave -- Charter and Canonical Surface Contract

**Wave:** Futures Venue Execution Proof (Post-Simplification)
**Phase:** 47
**Charter stage:** S421
**Date frozen:** 2026-03-23
**Predecessor gate:** S420 -- Runtime Simplification Evidence Gate (PASS -- FULL DELIVERY)
**Authority:** This document freezes wave scope AND canonical surface. Changes require a new stage.

---

## 1. Strategic Context

### 1.1 Consolidated Baseline

Ten consecutive passing gates across 46 phases have proven the Foundry's execution architecture:

| Phase | Wave | Range | Verdict |
|---|---|---|---|
| 38 | Multi-binary orchestration | S370--S375 | PASS |
| 39 | Exchange listening + dry-run | S376--S381 | PASS |
| 40 | OMS Foundation | S382--S388 | PASS |
| 41 | Binance segmentation | S390--S395 | PASS |
| 42 | Unified segment runtime | S398--S403 | PASS -- FULL |
| 43 | Testnet venue execution (Spot-first) | S404--S409 | PASS -- FULL |
| 44 | Production readiness hardening | S410--S414 | PASS -- FULL |
| 45 | Futures venue execution proof | S415--S420 | PASS -- SUBSTANTIAL |
| 46 | Runtime simplification | S416--S420 | PASS -- FULL |

The Runtime Simplification wave (Phase 46) specifically eliminated operational entropy:

- Config surface: 6 variants reduced to 3 canonical (50% reduction).
- Compose surface: 7 overlays reduced to 3 canonical (57% reduction).
- Taxonomy: "legacy" labels corrected to "standalone" (100% cleanup).
- Transitional test debt: 3 files removed with explicit supersession mapping.
- Zero production code changes. Zero regressions. All 62 non-goals respected.

### 1.2 Why This Wave Now

The S420 evidence gate formally **AUTHORIZED** the Futures Venue Execution Proof Wave under explicit conditions:

1. Use the consolidated config/compose surface.
2. Monitor G-4 (fee divergence) during proof.
3. Respect all 62 cumulative non-goals.
4. Define wave-specific charter with capabilities, questions, and non-goals.
5. Evidence gate must verify zero regressions against the full S370--S420 chain.

All 10 Futures preconditions validated at S419:

| # | Precondition | Status |
|---|---|---|
| 1 | Futures segment enabled in unified config | Ready |
| 2 | Futures segment enabled in venue-live config | Ready |
| 3 | Futures adapter (`BinanceFuturesTestnetAdapter`) exists | Ready |
| 4 | `SegmentRouter` dispatches `binancef` | Proven |
| 5 | Compose overlays declare Futures credentials | Ready |
| 6 | Futures E2E smoke script exists | Ready |
| 7 | Futures venue acceptance/fill tests pass | Proven |
| 8 | Futures rejection/audit tests pass | Proven |
| 9 | Source-to-segment mapping bijective | Proven |
| 10 | Fail-closed validation holds | Proven |

### 1.3 Key Differences from Spot Proof

| Dimension | Spot (proven at S409) | Futures (to prove) |
|---|---|---|
| Base URL | `testnet.binance.vision` | `testnet.binancefuture.com` |
| API path | `/api/v3/order` | `/fapi/v1/order` |
| Response model | `fills[]` array with per-leg price/qty/commission | Top-level `avgPrice`, `cumQuote`, `executedQty` |
| Partial fill | Rare (market orders fill atomically on Spot) | Common (large orders on Futures) |
| Margin | Not applicable | Required; insufficient margin is a common rejection |
| Position semantics | No position tracking | Futures positions exist but are NOT in scope |
| Leverage | Not applicable | Default leverage applies; NOT configurable |

### 1.4 Discipline: No Entropy Reintroduction

The Runtime Simplification wave explicitly consolidated the operational surface. This wave **MUST NOT** reintroduce parallel artifacts. The canonical surface is frozen (see Section 5). Any deviation requires written justification, non-goal compliance proof, and explicit acknowledgment in the stage report.

---

## 2. Wave Objective

Prove that the canonical OMS lifecycle behaves correctly when exercised against **real Binance Futures testnet responses** on the **unified runtime**, using the **canonical config/compose surface** established by the Runtime Simplification wave. This includes acceptance, fill, rejection, and partial fill paths -- confirming lifecycle fidelity, persistence consistency, compose-level E2E operation, and segment parity for the Futures segment.

---

## 3. Capability Target

At wave closure, the system must demonstrate:

| ID | Capability | Acceptance Criterion |
|---|---|---|
| **FV-C1** | Real Futures venue acceptance lifecycle | `submitted -> accepted -> filled` observed with real Futures testnet response |
| **FV-C2** | Real Futures venue fill record fidelity | Fill records carry real `avgPrice`, `executedQty`, `cumQuote` from Futures testnet |
| **FV-C3** | Real Futures venue rejection lifecycle | `submitted -> rejected` observed with real Futures rejection (insufficient margin, invalid params) |
| **FV-C4** | Real Futures venue rejection event fidelity | `VenueOrderRejectedEvent` carries real HTTP status, Futures error code, and reason |
| **FV-C5** | Real Futures venue partial fill lifecycle | `submitted -> accepted -> partially_filled -> filled` observed or structurally proven |
| **FV-C6** | Lifecycle invariant fidelity under real Futures data | All 8 invariant categories (ST, TERM, FR, QM, SM, SAFE, CORR, FINAL) hold |
| **FV-C7** | Persistence consistency under real Futures data | KV projection, HTTP query, and ClickHouse row agree on terminal state |
| **FV-C8** | Read-path auditability and segment parity | Unified read surfaces return correct Futures results alongside existing Spot data |
| **FV-C9** | Compose E2E with real Futures testnet | Full pipeline (derive -> execute -> store) runs in unified compose with Futures `venue_live` |
| **FV-C10** | Segment isolation under dual-segment operation | Futures orders do not leak to Spot adapter; Spot orders do not leak to Futures adapter |

---

## 4. Wave Blocks and Stage Order

| Stage | Block | Title | Governing Qs | Dependencies |
|---|---|---|---|---|
| **S421** | B0 -- Charter | Futures Venue Execution Proof Wave charter and canonical surface contract (this document) | -- | S420 PASS |
| **S422** | B1 -- Connectivity and fill | Futures real venue connectivity and acceptance/fill proof on unified runtime | FV-Q1, FV-Q2, FV-Q11, FV-Q12 | S421 |
| **S423** | B2 -- Rejection and partial fill | Futures real rejection and partial-fill evidence on unified runtime | FV-Q3, FV-Q4, FV-Q5, FV-Q6 | S422 |
| **S424** | B3 -- Read-path and segment parity | Unified runtime read-path, auditability, and segment parity under real Futures responses | FV-Q7, FV-Q8, FV-Q10 | S423 |
| **S425** | B4 -- Compose E2E | Unified compose E2E proof with Futures live execution path | FV-Q9 | S424 |
| **S426** | B5 -- Evidence gate | Futures Venue Execution Proof evidence gate (final) | All | S425 |

### Block Details

#### B1 -- S422: Futures Real Venue Connectivity and Acceptance/Fill Proof

Core connectivity and happy-path proof against Binance Futures testnet via the unified runtime:

- **FV-Q1:** `venue_live` write-path produces correct `submitted -> accepted -> filled` lifecycle transitions on a real Futures market order.
- **FV-Q2:** Fill records carry real `avgPrice`, `executedQty`, `cumQuote`, `commission`, `commissionAsset` from Futures testnet response.
- **FV-Q11:** Correlation chain (`CorrelationID`, `CausationID`) remains intact through real Futures HTTP request/response cycle.
- **FV-Q12:** Post-200 reconciliation path confirmed structurally sound under Futures conditions.

**Config:** `execute-venue-live.jsonc` (canonical, both segments, `dry_run=false`).
**Compose:** `docker-compose.yaml` + `docker-compose.venue-live.yaml`.

Deliverables:
- Real Futures testnet order submission via `BinanceFuturesTestnetAdapter` in `venue_live` mode.
- Evidence log showing lifecycle transitions with real venue data.
- Fill record audit showing real prices and quantities.
- Correlation chain trace through the pipeline.
- Smoke script or test harness for repeatable proof.

#### B2 -- S423: Futures Real Rejection and Partial-Fill Evidence

Rejection and edge-case proof against Binance Futures testnet:

- **FV-Q3:** Lifecycle correctly transitions to `rejected` on real Futures venue rejection (insufficient margin, invalid quantity, invalid symbol, auth failure).
- **FV-Q4:** `VenueOrderRejectedEvent` carries real Futures HTTP status, error code (`-2019`, `-1111`, `-4003`, etc.), and human-readable reason.
- **FV-Q5:** Partial fill observation from Futures testnet. Unlike Spot, Futures market orders can produce real partial fills on testnet.
- **FV-Q6:** Quantity monotonicity holds under real Futures partial fills.

**Config:** `execute-venue-live.jsonc` (canonical).
**Compose:** `docker-compose.yaml` + `docker-compose.venue-live.yaml`.

Deliverables:
- Real Futures rejection triggered and captured with venue error details.
- `VenueOrderRejectedEvent` audit with real venue payload.
- Partial fill evidence (real observation preferred; structural proof acceptable if testnet cannot reliably trigger partial fills).
- Monotonicity invariant verification.

#### B3 -- S424: Unified Runtime Read-Path, Auditability, and Segment Parity

Persistence and observability proof with real Futures data:

- **FV-Q7:** KV projections, HTTP query responses, and ClickHouse rows agree on terminal state after real Futures venue interactions.
- **FV-Q8:** Existing ClickHouse rejection writer (closed in S411) produces correct rows for Futures rejection events without code changes.
- **FV-Q10:** Sustained correct behavior over multiple consecutive Futures order cycles (minimum 3 cycles with real venue).

Additional parity verification:
- Unified read surfaces (lifecycle list, composite status) return correct results for both Spot and Futures data coexisting in the same KV/ClickHouse stores.
- Segment filtering prevents cross-contamination in query results.

**Config:** `execute-venue-live.jsonc` (canonical).
**Compose:** `docker-compose.yaml` + `docker-compose.venue-live.yaml`.

Deliverables:
- Persistence consistency audit across KV, HTTP, and ClickHouse after real Futures orders.
- ClickHouse row verification for Futures fill and rejection events.
- Multi-cycle sustained operation evidence.
- Segment parity audit showing dual-segment coexistence.

#### B4 -- S425: Unified Compose E2E Proof with Futures Live Execution Path

Full pipeline proof in compose with real Futures venue:

- **FV-Q9:** Full compose pipeline (`derive -> execute -> store`) operates correctly in `venue_live` mode against Futures testnet on the unified runtime with both Spot and Futures segments enabled.

**Config:** `execute-venue-live.jsonc` (canonical).
**Compose:** `docker-compose.yaml` + `docker-compose.venue-live.yaml`.

Deliverables:
- Compose stack boots with unified config: both segments enabled, `dry_run=false`.
- End-to-end flow: market data ingress -> signal derivation -> execution intent -> real Futures order -> fill/rejection -> persistence -> query.
- Smoke script proving the full pipeline with real Futures venue responses.
- Evidence of segment isolation under dual-segment compose operation.

#### B5 -- S426: Evidence Gate (Final)

Wave closure ceremony:

- Evaluate all 12 governing questions against Futures evidence on unified runtime.
- Classify all 10 capabilities.
- Regression verification against all prior waves (S370--S420).
- Non-goal compliance check (62 cumulative + wave-specific).
- Residual gap assessment.
- Verdict: `PASS`, `PASS -- CONDITIONAL`, or `FAIL`.
- Recommendation for next ceremony.

---

## 5. Canonical Surface Contract

This section freezes the operational surface for the entire wave. All execution stages (S422--S425) MUST use only the artifacts listed below.

### 5.1 Canonical Config Files

| File | Purpose | Mode | Segments | `dry_run` |
|---|---|---|---|---|
| `deploy/configs/execute.jsonc` | Paper simulator, standalone mode | Development | None (standalone) | `true` |
| `deploy/configs/execute-unified.jsonc` | Segmented execution, dry-run | Testing | Spot + Futures | `true` |
| `deploy/configs/execute-venue-live.jsonc` | Real testnet execution | **Proof** | Spot + Futures | `false` |

**Wave primary config:** `execute-venue-live.jsonc` -- this is the config used for all Futures venue proof stages.

**Rules:**
- No new execute config files may be created during this wave.
- To run Futures-only, disable the Spot segment in `execute-venue-live.jsonc` at runtime; do not create a per-segment config file.
- If a config change is needed, modify the canonical file with justification in the stage report.

### 5.2 Canonical Compose Files

| File | Purpose | Usage |
|---|---|---|
| `deploy/compose/docker-compose.yaml` | Base stack (9 services) | Always present |
| `deploy/compose/docker-compose.unified.yaml` | Segmented dry-run overlay | Testing only |
| `deploy/compose/docker-compose.venue-live.yaml` | Real testnet overlay | **Proof execution** |

**Wave primary compose:** `docker-compose.yaml` + `docker-compose.venue-live.yaml`.

**Rules:**
- No new compose overlay files may be created during this wave.
- No per-segment compose overlays (NG-46 from prior waves).
- Overlay only touches the `execute` service definition.
- Base always present; at most one overlay active at a time.

### 5.3 Canonical Smoke Scripts

Existing scripts may be used and extended. New smoke scripts may be created **only** if they reference canonical compose/config artifacts and follow the established naming convention.

| Existing Script | Purpose |
|---|---|
| `scripts/smoke-e2e-unified-futures.sh` | Futures E2E smoke |
| `scripts/smoke-futures-venue-live.sh` | Futures venue live smoke |
| `scripts/smoke-futures-rejection-partial-fill.sh` | Futures rejection/partial fill |
| `scripts/smoke-unified-runtime-preflight.sh` | Runtime preflight validation |

### 5.4 Canonical Runtime Topology

| Component | Canonical Form | Frozen? |
|---|---|---|
| Execute binary | Single binary, unified config, `SegmentRouter` dispatch | Yes |
| Segment routing | Source-prefix dispatch: `binances` -> Spot, `binancef` -> Futures | Yes |
| NATS subjects | `execution.intents.>`, `execution.events.>` with segment-scoped consumers | Yes |
| KV bucket | `execution-lifecycle` (both segments, shared) | Yes |
| ClickHouse table | `executions` (20-column, segment-agnostic) | Yes |
| Lifecycle state machine | 7 states, 49/49 transitions proven | Yes |
| Adapters | `BinanceSpotTestnetAdapter`, `BinanceFuturesTestnetAdapter` | Yes |

### 5.5 Surface Deviation Protocol

Any deviation from the canonical surface requires:

1. Written justification in the stage report that introduces it.
2. Proof that the deviation does not violate any non-goal.
3. Explicit plan for removing the deviation before the evidence gate (S426) unless the deviation becomes the new canonical form.
4. Approval from the stage author (recorded in the stage report).

---

## 6. Entry Preconditions

All preconditions are met as of S420:

| Precondition | Status | Evidence |
|---|---|---|
| Seven-state lifecycle proven | MET | S383: 49/49 transitions, 8 invariant categories |
| Write-path per execution mode proven | MET | S385: 19 integration tests |
| Rejection event path proven | MET | S386: 19 tests; S411: ClickHouse wiring |
| Persistence read-path proven | MET | S387 + S413: lifecycle list consolidated |
| OMS Foundation evidence gate | MET | S388 PASS |
| `BinanceFuturesTestnetAdapter` exists | MET | `internal/application/execution/binance_futures_testnet_adapter.go` |
| Segment routing proven | MET | S401: `SegmentRouter` with fail-closed dispatch |
| Config-driven segment enablement proven | MET | S393, S399 unified config |
| Unified runtime proven | MET | S403 PASS -- FULL |
| Spot venue execution proven | MET | S409 PASS -- FULL |
| Production hardening proven | MET | S414 PASS -- FULL |
| ClickHouse rejection writer wired | MET | S411: RG-1 CLOSED |
| Endurance evidence baseline | MET | S412: 2,000+ cycles, zero drift |
| Operational queryability | MET | S413: lifecycle list, composite status |
| Runtime simplification | MET | S420 PASS -- FULL DELIVERY |
| Config/compose canonical surface | MET | S416--S417: 3 configs, 3 compose files |
| Futures testnet credentials | REQUIRED | Must be provisioned before S422 |

---

## 7. Scope Freeze Rules

### What enters the wave

1. Real Futures testnet HTTP interactions via `BinanceFuturesTestnetAdapter`.
2. Lifecycle state transitions validated against real Futures venue responses.
3. Fill record fidelity: real `avgPrice`, `executedQty`, `cumQuote`, fees from Futures testnet.
4. Rejection event fidelity: real Futures venue error codes and reasons.
5. Partial fill observation (real preferred) or structural feasibility proof.
6. Persistence surface consistency under real Futures event data.
7. Read-path segment parity: Futures and Spot coexist correctly in unified read surfaces.
8. Compose-level E2E proof with `venue_live` mode against Futures testnet.
9. Segment isolation verification under dual-segment operation.

### What does NOT enter the wave

See companion document:
[`futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md`](futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md)

### Scope amendment protocol

Any scope addition requires:

1. Written justification linking to a governing question.
2. Proof that the addition does not violate any non-goal.
3. Proof that the addition does not violate the canonical surface contract (Section 5).
4. Explicit acknowledgment in the stage report that introduced it.

---

## 8. Carried Forward Items

### 8.1 Residual Gaps from S420

| Gap | Severity | Disposition in This Wave |
|---|---|---|
| G-4: Fee semantic divergence (Spot commission vs Futures cumQuote) | Medium | **MONITORED**: Flag if it affects Futures fill record fidelity. Normalization deferred to production analytics. |
| G-1: No parallel Spot+Futures live proof | Low | DEFERRED: Each segment proven independently. Parallel is soak concern. |
| G-2/RG-4: Segment-scoped list queries | Low | DEFERRED: Operational listing sufficient. |
| G-3: Rejection code in JSON metadata | Low | CARRIED: Queryable via JSON extraction. |
| G-5: No per-segment health check | Low | CARRIED: `/execution/activation/surface` provides visibility. |
| RG-2: Partial fill live observation | Low | **ELEVATED**: Futures partial fills are more likely; S423 attempts real observation. |
| RG-3: Latest-only KV semantics | Low | DEFERRED: By design; ClickHouse covers history. |
| RG-16: 97 untracked docs | Low | DEFERRED: Separate governance ceremony. |
| RG-17: Smoke script naming | Low | DEFERRED: Cosmetic. |
| RG-18: Doc suitability not assessed | Low | DEFERRED: Separate review. |

### 8.2 Non-Goals Lifted

| Former NG | Disposition |
|---|---|
| NG-36 (no Futures testnet proof) from S410 | **LIFTED**: Becomes this wave's primary goal. |

All other cumulative non-goals remain in force.

---

## 9. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Futures testnet API unavailability | Medium | Medium | Idempotent tests, retriable smoke scripts |
| Futures testnet insufficient margin | Medium | High | Verify test account funding and leverage before S422 |
| Partial fills hard to trigger on testnet | Medium | Low | More likely than Spot; structural proof acceptable |
| Futures API response format divergence | Low | Medium | Adapter already coded and unit-tested; S422 validates against real responses |
| Cross-segment leakage under dual `venue_live` | Low | High | S401 isolation proven; S424 re-verifies under real data |
| Futures-specific error codes unmapped | Medium | Low | Adapter error classification extensible; S423 maps observed codes |
| Credential management | Low | Low | Env vars only; never committed |
| Entropy reintroduction via "temporary" artifacts | Low | Medium | Canonical surface contract (Section 5) prevents creation of parallel artifacts |

---

## 10. Success Criteria for Wave Closure

The evidence gate (S426) will evaluate:

1. **10/10 capabilities at FULL or SUBSTANTIAL** -- no PARTIAL or PENDING.
2. **12/12 governing questions ANSWERED or SUBSTANTIAL** -- no UNANSWERED.
3. **Zero non-goal violations** (62 cumulative + wave-specific).
4. **Zero regressions** in all prior test suites (S370--S420).
5. **Canonical surface contract respected** -- no unauthorized deviations.
6. **Segment parity demonstrated**: Futures and Spot coexist in unified read surfaces.
7. **G-4 (fee divergence) assessed** under real Futures data.
8. **All residual gaps remain non-blocking** under real Futures venue load.

Verdict options: `PASS`, `PASS -- CONDITIONAL`, `FAIL`.

---

## 11. Links

| Reference | Link |
|---|---|
| Companion: capabilities, questions, non-goals, surface constraints | [`futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md`](futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md) |
| Predecessor gate: S420 (Runtime Simplification) | [`../stages/stage-s420-runtime-simplification-evidence-gate-report.md`](../stages/stage-s420-runtime-simplification-evidence-gate-report.md) |
| Authorization document | [`runtime-simplification-evidence-gate-and-futures-proof-authorization.md`](runtime-simplification-evidence-gate-and-futures-proof-authorization.md) |
| Config reference | [`../../deploy/configs/CONFIG-REFERENCE.md`](../../deploy/configs/CONFIG-REFERENCE.md) |
| Prior Futures charter (S415) | [`futures-venue-execution-proof-wave-charter-and-scope-freeze.md`](futures-venue-execution-proof-wave-charter-and-scope-freeze.md) |
| Prior Futures capabilities/non-goals | [`futures-venue-execution-capabilities-questions-and-non-goals.md`](futures-venue-execution-capabilities-questions-and-non-goals.md) |
| Spot proof charter (S404) | [`testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md`](testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md) |
| Canonical order model | [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md) |
| Stage report | [`../stages/stage-s421-futures-venue-execution-charter-report.md`](../stages/stage-s421-futures-venue-execution-charter-report.md) |
