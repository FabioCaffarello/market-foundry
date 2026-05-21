# Futures Venue Execution Proof Wave -- Charter and Scope Freeze

**Wave:** Futures Venue Execution Proof
**Charter stage:** S415
**Date frozen:** 2026-03-23
**Predecessor wave:** Production Readiness Hardening (S410--S414, PASSED -- FULL DELIVERY)
**Authority:** This document freezes wave scope. Changes require a new stage.

---

## 1. Strategic Context

### 1.1 Consolidated State

The Foundry has completed seven consecutive wave passes since S370:

| Wave | Range | Verdict |
|---|---|---|
| Multi-binary orchestration | S370--S375 | PASS |
| Exchange listening + dry-run | S376--S381 | PASS |
| OMS Foundation | S382--S388 | PASS |
| Binance segmentation | S390--S395 | PASS |
| Unified segment runtime | S398--S403 | PASS -- FULL DELIVERY |
| Testnet venue execution (Spot-first) | S404--S409 | PASS -- SUBSTANTIAL DELIVERY |
| Production readiness hardening | S410--S414 | PASS -- FULL DELIVERY |

The Spot execution path is now production-hardened: real venue connectivity,
complete lifecycle proof (acceptance, fill, rejection, partial fill structural),
persistence to KV and ClickHouse, operational queryability, endurance evidence
(2,000+ cycles), and zero open medium-severity gaps.

### 1.2 Why Futures Now

The Futures segment is the highest-value, lowest-risk next expansion:

| Factor | Rationale |
|---|---|
| **Adapter exists** | `BinanceFuturesTestnetAdapter` built in S389, exercised in structural tests through S403 |
| **Segment routing proven** | `SegmentRouter` dispatches by source; Futures routing (`binancef`) tested in S401 |
| **Config enablement proven** | Futures segment enablement validated in S393, unified config in S399 |
| **Persistence schema ready** | 20-column `executions` table accommodates all event types without DDL changes |
| **Read surfaces segment-agnostic** | Lifecycle list, composite status, and history queries work across segments |
| **Non-goal lifted** | NG-36 (no Futures proof) from S410 becomes this wave's primary goal |
| **Partial fill relevance** | Futures markets commonly produce partial fills, elevating RG-2 |

The alternative directions (OMS expansion, analytics consolidation) both
benefit from having complete dual-segment venue coverage first.

### 1.3 Key Differences from Spot Proof

The Futures execution proof exercises the same architecture but with
Futures-specific API and response semantics:

| Dimension | Spot (proven) | Futures (to prove) |
|---|---|---|
| Base URL | `testnet.binance.vision` | `testnet.binancefuture.com` |
| API path | `/api/v3/order` | `/fapi/v1/order` |
| Response model | `fills[]` array with per-leg price/qty/commission | Top-level `avgPrice`, `cumQuote`, `executedQty` |
| Partial fill | Rare (market orders fill atomically on Spot) | Common (large orders on Futures) |
| Margin | Not applicable | Required; insufficient margin is a common rejection |
| Position semantics | No position tracking | Futures positions exist but are NOT in scope |
| Leverage | Not applicable | Default leverage applies; NOT configurable in this wave |

---

## 2. Wave Objective

Prove that the canonical OMS lifecycle behaves correctly when exercised against
**real Binance Futures testnet responses** on the unified runtime. This includes
acceptance, fill, rejection, and partial fill paths -- confirming lifecycle
fidelity, persistence consistency, and compose-level E2E operation for the
Futures segment.

---

## 3. Capability Target

At wave closure, the system must demonstrate:

| ID | Capability | Acceptance |
|---|---|---|
| **FV-C1** | Real Futures venue acceptance lifecycle | `submitted -> accepted -> filled` observed with real Futures testnet response |
| **FV-C2** | Real Futures venue fill record fidelity | Fill records carry real `avgPrice`, `executedQty`, `cumQuote` from Futures testnet |
| **FV-C3** | Real Futures venue rejection lifecycle | `submitted -> rejected` observed with real Futures rejection (insufficient margin, invalid params) |
| **FV-C4** | Real Futures venue rejection event fidelity | `VenueOrderRejectedEvent` carries real HTTP status, Futures error code, and reason |
| **FV-C5** | Real Futures venue partial fill lifecycle | `submitted -> accepted -> partially_filled -> filled` observed or structurally proven with Futures testnet |
| **FV-C6** | Lifecycle invariant fidelity under real Futures data | All 8 invariant categories (ST, TERM, FR, QM, SM, SAFE, CORR, FINAL) hold |
| **FV-C7** | Persistence consistency under real Futures data | KV projection, HTTP query, and ClickHouse row agree on terminal state |
| **FV-C8** | Read-path auditability and segment parity | Unified read surfaces return correct Futures results alongside existing Spot data |
| **FV-C9** | Compose E2E with real Futures testnet | Full pipeline (derive -> execute -> store) runs in unified compose with Futures `venue_live` |
| **FV-C10** | Segment isolation under dual-segment live execution | Futures orders do not leak to Spot adapter; Spot orders do not leak to Futures adapter |

---

## 4. Wave Blocks and Stage Order

| Stage | Block | Title | Governing Qs |
|---|---|---|---|
| **S415** | B0 -- Charter | Futures Venue Execution Proof Wave charter and scope freeze (this document) | -- |
| **S416** | B1 -- Connectivity and fill | Futures real venue connectivity, acceptance, and fill proof | FV-Q1, FV-Q2, FV-Q11, FV-Q12 |
| **S417** | B2 -- Rejection and partial fill | Futures real rejection and partial-fill evidence | FV-Q3, FV-Q4, FV-Q5, FV-Q6 |
| **S418** | B3 -- Read-path and parity | Unified runtime read-path, auditability, and segment parity under real Futures responses | FV-Q7, FV-Q8, FV-Q10 |
| **S419** | B4 -- Compose E2E | Unified compose E2E proof with Futures live execution path | FV-Q9 |
| **S420** | B5 -- Evidence gate | Futures Venue Execution Proof evidence gate (final) | All |

### Block Details

#### B1 -- S416: Futures Real Venue Connectivity, Acceptance, and Fill Proof

Core connectivity and happy-path proof against Binance Futures testnet via the
unified runtime:

- **FV-Q1:** `venue_live` write-path produces correct `submitted -> accepted -> filled`
  lifecycle transitions on a real Futures market order.
- **FV-Q2:** Fill records carry real `avgPrice`, `executedQty`, `cumQuote`,
  `commission`, `commissionAsset` from Futures testnet response.
- **FV-Q11:** Correlation chain (`CorrelationID`, `CausationID`) remains intact
  through real Futures HTTP request/response cycle.
- **FV-Q12:** Post-200 reconciliation path confirmed structurally sound under
  Futures conditions.

Deliverables:
- Real Futures testnet order submission via `BinanceFuturesTestnetAdapter` in
  `venue_live` mode on the unified runtime.
- Evidence log showing lifecycle transitions with real venue data.
- Fill record audit showing real prices and quantities.
- Correlation chain trace through the pipeline.
- Smoke script or test harness for repeatable proof.

#### B2 -- S417: Futures Real Rejection and Partial-Fill Evidence

Rejection and edge-case proof against Binance Futures testnet:

- **FV-Q3:** Lifecycle correctly transitions to `rejected` on real Futures venue
  rejection (insufficient margin, invalid quantity, invalid symbol, auth failure).
- **FV-Q4:** `VenueOrderRejectedEvent` carries real Futures HTTP status, error
  code (`-2019`, `-1111`, `-4003`, etc.), and human-readable reason.
- **FV-Q5:** Partial fill observation from Futures testnet. Unlike Spot, Futures
  market orders can produce real partial fills on testnet.
- **FV-Q6:** Quantity monotonicity holds under real Futures partial fills.

Deliverables:
- Real Futures rejection triggered and captured with venue error details.
- `VenueOrderRejectedEvent` audit with real venue payload.
- Partial fill evidence (real observation preferred; structural proof acceptable
  if testnet cannot reliably trigger partial fills).
- Monotonicity invariant verification.

#### B3 -- S418: Unified Runtime Read-Path, Auditability, and Segment Parity

Persistence and observability proof with real Futures data:

- **FV-Q7:** KV projections, HTTP query responses, and ClickHouse rows agree on
  terminal state after real Futures venue interactions.
- **FV-Q8:** Existing ClickHouse rejection writer (closed in S411) produces
  correct rows for Futures rejection events without code changes.
- **FV-Q10:** Sustained correct behavior over multiple consecutive Futures order
  cycles (minimum 3 cycles with real venue).

Additional parity verification:
- Unified read surfaces (lifecycle list, composite status) return correct
  results for both Spot and Futures data coexisting in the same KV/ClickHouse
  stores.
- Segment filtering prevents cross-contamination in query results.

Deliverables:
- Persistence consistency audit across KV, HTTP, and ClickHouse after real
  Futures orders.
- ClickHouse row verification for Futures fill and rejection events.
- Multi-cycle sustained operation evidence.
- Segment parity audit showing dual-segment coexistence.

#### B4 -- S419: Unified Compose E2E Proof with Futures Live Execution Path

Full pipeline proof in compose with real Futures venue:

- **FV-Q9:** Full compose pipeline (`derive -> execute -> store`) operates
  correctly in `venue_live` mode against Futures testnet on the unified runtime
  with both Spot and Futures segments enabled.

Deliverables:
- Compose stack boots with unified config: Spot (`dry_run` or disabled) +
  Futures (`venue_live`).
- End-to-end flow: market data ingress -> signal derivation -> execution intent
  -> real Futures order -> fill/rejection -> persistence -> query.
- Smoke script proving the full pipeline with real Futures venue responses.
- Evidence of segment isolation under dual-segment compose operation.

#### B5 -- S420: Evidence Gate (Final)

Wave closure ceremony:

- Evaluate all 12 governing questions against Futures evidence on unified runtime.
- Classify all 10 capabilities.
- Regression verification against all prior waves.
- Non-goal compliance check.
- Verdict: `PASSED`, `PASSED -- CONDITIONAL`, or `FAILED`.
- Recommendation for next ceremony.

---

## 5. Entry Preconditions

All preconditions are met:

| Precondition | Status | Evidence |
|---|---|---|
| Seven-state lifecycle proven | MET | S383: 49/49 transitions, 8 invariant categories |
| Write-path per execution mode proven | MET | S385: 19 integration tests |
| Rejection event path proven | MET | S386: 19 tests; S411: ClickHouse wiring |
| Persistence read-path proven | MET | S387 + S413: lifecycle list consolidated |
| OMS Foundation evidence gate | MET | S388 PASS |
| BinanceFuturesTestnetAdapter exists | MET | `internal/application/execution/binance_futures_testnet_adapter.go` |
| Segment routing proven | MET | S401: `SegmentRouter` with fail-closed dispatch |
| Config-driven segment enablement proven | MET | S393, S399 unified config |
| Unified runtime proven | MET | S403 PASS -- FULL DELIVERY |
| Spot venue execution proven | MET | S409 PASS -- SUBSTANTIAL |
| Production hardening proven | MET | S414 PASS -- FULL DELIVERY |
| ClickHouse rejection writer wired | MET | S411: RG-1 CLOSED |
| Endurance evidence baseline | MET | S412: 2,000+ cycles, zero drift |
| Operational queryability | MET | S413: lifecycle list, composite status |
| Futures testnet credentials | REQUIRED | Must be provisioned before S416 |

---

## 6. Scope Freeze Rules

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
[`futures-venue-execution-capabilities-questions-and-non-goals.md`](futures-venue-execution-capabilities-questions-and-non-goals.md)

### Scope amendment protocol

Any scope addition requires:

1. Written justification linking to a governing question.
2. Proof that the addition does not violate any non-goal.
3. Explicit acknowledgment in the stage report that introduced it.

---

## 7. Architectural Assumptions

The unified runtime, persistence pipeline, and read surfaces are treated as a
**stable foundation**. This wave **consumes** them, not **modifies** them.

| Assumption | Basis |
|---|---|
| Single binary boots Futures adapter from unified config | S400, S402 proven |
| Source-based routing dispatches `binancef` intents to Futures adapter | S401 proven |
| `dry_run=false` activates `venue_live` write-path for Futures | S385, S403 proven |
| NATS consumer filtering prevents cross-segment delivery | S401 proven |
| ClickHouse writer handles Futures events without schema changes | S411: 20-column schema is segment-agnostic |
| Compose stack supports dual-segment `venue_live` | S402 structural; S419 proves under real venue |
| Lifecycle list and composite status queries are segment-agnostic | S413 proven |

**Key design decision:** For `venue_live` Futures execution, the unified config
sets `futures.enabled=true` with `dry_run=false`. Spot segment may be kept
enabled with `dry_run=true` or disabled entirely. The exact config pattern will
be established in S416 and documented.

---

## 8. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Futures testnet API unavailability | Medium | Medium | Idempotent tests, retriable smoke scripts |
| Futures testnet insufficient margin | Medium | High | Verify test account funding and leverage before S416 |
| Partial fills hard to trigger on Futures testnet | Medium | Low | More likely than Spot but not guaranteed; structural proof acceptable |
| Futures API response format divergence from docs | Low | Medium | Adapter already coded and unit-tested; S416 validates against real responses |
| Cross-segment leakage under dual `venue_live` | Low | High | S401 isolation proven; S418 re-verifies under real data |
| Futures-specific error codes unmapped | Medium | Low | Adapter error classification extensible; S417 maps observed codes |
| Credential management | Low | Low | Env vars only; never committed; documented in setup instructions |

---

## 9. Carried Forward Items

### Residual Gaps from S414

| Gap | Severity | Disposition in This Wave |
|---|---|---|
| RG-2: Partial fill live observation | Low | **ELEVATED**: Futures partial fills are more likely; S417 attempts real observation |
| RG-3: Latest-only KV semantics | Low | DEFERRED: by design; ClickHouse covers history |
| RG-4: Segment-scoped list queries (partial) | Low | DEFERRED: S413 operational listing sufficient |
| RG-6--RG-11 | Low | CARRIED: not addressed in this wave |

### Non-Goals Lifted

| Former NG | Disposition |
|---|---|
| NG-36 (no Futures testnet proof) | **LIFTED**: becomes this wave's primary goal |

All other 49 non-goals from S414 remain in force unless explicitly superseded
in the companion non-goals document.

---

## 10. Success Criteria for Wave Closure

The evidence gate (S420) will evaluate:

1. **10/10 capabilities at FULL or SUBSTANTIAL** -- no PARTIAL or PENDING.
2. **12/12 governing questions ANSWERED or SUBSTANTIAL** -- no UNANSWERED.
3. **Zero non-goal violations.**
4. **Zero regressions** in all prior test suites.
5. **Segment parity demonstrated**: Futures and Spot coexist in unified read surfaces.
6. **All S414 residual gaps remain non-blocking** under real Futures venue load.

Verdict options: `PASSED`, `PASSED -- CONDITIONAL`, `FAILED`.

---

## 11. Links

| Reference | Link |
|---|---|
| Companion: capabilities, questions, non-goals | [`futures-venue-execution-capabilities-questions-and-non-goals.md`](futures-venue-execution-capabilities-questions-and-non-goals.md) |
| Predecessor gate: S414 | [`../stages/stage-s414-production-readiness-hardening-evidence-gate-report.md`](../stages/stage-s414-production-readiness-hardening-evidence-gate-report.md) |
| Predecessor evidence matrix | [`production-readiness-hardening-evidence-matrix-residual-gaps-and-next-ceremony.md`](production-readiness-hardening-evidence-matrix-residual-gaps-and-next-ceremony.md) |
| Spot venue execution charter (S404) | [`testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md`](testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md) |
| Spot venue execution gate (S409) | [`../stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md`](../stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md) |
| Unified runtime charter | [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Canonical order model | [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md) |
| Lifecycle invariants | [`order-lifecycle-invariants-transitions-and-boundaries.md`](order-lifecycle-invariants-transitions-and-boundaries.md) |
| Stage report | [`../stages/stage-s415-futures-venue-execution-proof-charter-report.md`](../stages/stage-s415-futures-venue-execution-proof-charter-report.md) |
