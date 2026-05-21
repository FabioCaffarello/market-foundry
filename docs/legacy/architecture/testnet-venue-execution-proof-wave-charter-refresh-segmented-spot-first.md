# Testnet Venue Execution Proof Wave -- Charter Refresh (Segmented, Spot-First)

**Wave:** Testnet Venue Execution Proof (refreshed)
**Charter refresh stage:** S396
**Original charter:** S389
**Date frozen:** 2026-03-22
**Predecessor:** Binance Spot/Futures Segmentation Foundation Wave (S390--S395, PASSED)

---

## 1. Refresh Rationale

S389 opened the Testnet Venue Execution Proof Wave with 12 governing
questions, 10 capability targets, and 22 non-goals. The wave assumed a
single venue adapter (`BinanceFuturesTestnetAdapter`) operating in isolation.

Before execution of TV-Q1 could begin, an architectural gap was identified:
the system lacked a segmented venue model that could support both Binance Spot
and Binance Futures as orthogonal market segments. The Binance Spot/Futures
Segmentation Foundation Wave (S390--S395) was opened as an interleaving wave
and closed with PASS, delivering:

- Canonical venue model with `MarketSegment` dimension (C1 -- FULL).
- `BinanceSpotTestnetAdapter` implementing `VenuePort` + `VenueQueryPort` (C2 -- FULL).
- Config-driven segment enablement with fail-closed semantics (C3 -- FULL).
- Compose-level segment isolation (C4 -- SUBSTANTIAL).
- Mainnet extensibility proof (C5 -- FULL structural).

The original S389 charter is now stale in three dimensions:

1. **Venue target.** S389 assumed Futures-only. The system now supports both
   segments. The wave must adopt a Spot-first execution strategy.
2. **Stage numbering.** S390--S395 were consumed by the segmentation wave.
   The refreshed execution stages begin at S397.
3. **Infrastructure preconditions.** The segmentation wave introduced new
   prerequisites (Spot ingest seed, dual-instance compose) that the original
   charter did not anticipate.

This document refreshes the charter without redesigning it. The 12 governing
questions remain structurally intact but are resequenced and targeted at the
Spot segment first.

---

## 2. Spot-First Strategy

### 2.1 Why Spot First

| Factor | Rationale |
|---|---|
| Adapter freshness | `BinanceSpotTestnetAdapter` was just built in S394; proving it first closes the gap fastest |
| API simplicity | Spot REST API (`/api/v3/order`) is simpler than Futures (`/fapi/v1/order`) -- fewer edge cases for first proof |
| Fill model | Spot returns `fills[]` array directly in the order response; Futures returns aggregated fields -- Spot exercises the multi-fill aggregation path |
| Risk isolation | Spot testnet has no leverage/margin complexity -- cleaner proof of lifecycle fidelity |
| Segmentation validation | Proving Spot first validates the segmentation architecture under real venue load, not just structural tests |

### 2.2 What Spot-First Means

1. All capability targets are first proven against the Binance Spot testnet.
2. Futures proof is deferred to a follow-on wave or inline extension after the
   Spot evidence gate passes.
3. Dual-instance compose is proven early to confirm segmentation holds under
   concurrent runtime, but governing questions are answered Spot-only.
4. Credentials, configs, and smoke scripts are created for Spot first.

### 2.3 What Spot-First Does NOT Mean

- Futures is not abandoned. The `BinanceFuturesTestnetAdapter` remains intact
  and regression-tested.
- The lifecycle state machine is segment-agnostic. Proof on Spot is proof of
  the lifecycle; Futures adds only venue-specific HTTP/response mapping coverage.
- No Spot-specific lifecycle states or domain model changes are introduced.

---

## 3. Revised Wave Blocks and Stage Order

### 3.1 Block Map

| Block | Stage | Description | Governing Qs |
|---|---|---|---|
| B0 | S396 | Charter refresh (this document) | -- |
| B1 | S397 | Spot ingest binding seed and runtime projection closure | Precondition |
| B2 | S398 | Dual-instance compose proof for segmented runtime | Precondition, TV-Q9 (partial) |
| B3 | S399 | Spot real acceptance/fill/rejection lifecycle proof | TV-Q1, TV-Q2, TV-Q3, TV-Q4, TV-Q11, TV-Q12 |
| B4 | S400 | Spot OMS read-path, auditability, and compose E2E proof | TV-Q5, TV-Q6, TV-Q7, TV-Q8, TV-Q9, TV-Q10 |
| B5 | S401 | Evidence gate (final) | All |

### 3.2 Block Descriptions

**B1 -- S397: Spot Ingest Binding Seed and Runtime Projection Closure**

Close S395 residual gap G3 (Spot ingest not seeded). This stage:

- Seeds `binances` source bindings in NATS for Spot market data ingestion.
- Verifies that the Spot execute binary can boot with real ingest data flowing.
- Confirms runtime projection (activation surface logs segment identity with
  live data present).
- Produces `make seed-spot` or extends `make seed` to include Spot bindings.

This is a precondition stage. No governing question is answered, but the
infrastructure is required for all subsequent stages.

**B2 -- S398: Dual-Instance Compose Proof for Segmented Runtime**

Close S395 residual gap G1 (concurrent multi-instance compose not proven). This
stage:

- Creates `docker-compose.dual.yaml` running both `execute-futures` and
  `execute-spot` as separate compose services against their respective testnets.
- Proves that both services boot, report segment identity, and process
  independently without NATS subject or KV collision.
- Produces a smoke script (`smoke-dual-compose.sh`) that validates concurrent
  healthy operation.
- Partially addresses TV-Q9 by proving compose-level infrastructure under
  segmented runtime.

**B3 -- S399: Spot Real Acceptance/Fill/Rejection Lifecycle Proof**

The core proof stage. Against the Binance Spot testnet:

- TV-Q1: `venue_live` write-path produces correct `submitted -> accepted -> filled`.
- TV-Q2: Fill records carry real `price`, `qty`, `commission` from Spot testnet.
- TV-Q3: Lifecycle transitions to `rejected` on real Spot rejection (insufficient
  balance, invalid params).
- TV-Q4: `VenueOrderRejectedEvent` carries real HTTP status, error code, and
  reason from Spot testnet.
- TV-Q11: Correlation chain intact through real Spot venue interactions.
- TV-Q12: Post-200 reconciliation confirmed structurally sound for Spot adapter.

Partial fill (TV-Q5, TV-Q6) is deferred to B4 because Spot market orders
typically fill immediately. B4 will address partial fill as structural proof or
limit-order exploration if feasible.

**B4 -- S400: Spot OMS Read-Path, Auditability, and Compose E2E Proof**

Consolidates persistence, read-path, and E2E concerns:

- TV-Q5: Partial fill observation or structural proof from Spot testnet.
- TV-Q6: Quantity monotonicity under partial fills (structural or observed).
- TV-Q7: KV, HTTP, and ClickHouse agree on terminal state after real Spot
  interactions.
- TV-Q8: ClickHouse rejection writer wired and producing correct rows (RG-1
  closure).
- TV-Q9: Full compose pipeline (derive -> execute -> store) in `venue_live`
  mode against Spot testnet (builds on B2 dual-compose).
- TV-Q10: Sustained correct behavior over multiple consecutive Spot order cycles.

**B5 -- S401: Evidence Gate (Final)**

Wave closure ceremony:

- Evaluate all 12 governing questions against Spot evidence.
- Classify all 10 capabilities.
- Regression verification against all prior waves.
- Non-goal compliance check.
- Verdict: PASSED, PASSED -- CONDITIONAL, or FAILED.
- Recommendation for next ceremony (Futures proof, mainnet activation, or
  operational refinement).

---

## 4. Revised Governing Questions

The 12 original governing questions (TV-Q1 through TV-Q12) are preserved
verbatim but retargeted to Spot:

| ID | Question (unchanged from S389) | Revised target | Original target |
|---|---|---|---|
| TV-Q1 | Does `venue_live` produce correct lifecycle transitions on real acceptance + fill? | S399 (Spot) | S390 |
| TV-Q2 | Do fill records carry accurate real prices, quantities, and fees? | S399 (Spot) | S390 |
| TV-Q3 | Does the lifecycle transition to `rejected` on real venue rejection? | S399 (Spot) | S391 |
| TV-Q4 | Does `VenueOrderRejectedEvent` carry real venue rejection code and reason? | S399 (Spot) | S391 |
| TV-Q5 | Can partial fill be observed or structurally proven from testnet? | S400 (Spot) | S392 |
| TV-Q6 | Does quantity monotonicity hold under real partial fills? | S400 (Spot) | S392 |
| TV-Q7 | Do KV, HTTP, and ClickHouse agree on terminal state after real interactions? | S400 (Spot) | S393 |
| TV-Q8 | Is the ClickHouse rejection writer wired and producing correct rows? | S400 (Spot) | S393 |
| TV-Q9 | Does the full compose pipeline work in `venue_live` against testnet? | S398 (partial), S400 (full) | S394 |
| TV-Q10 | Does the system sustain correct behavior over multiple order cycles? | S400 (Spot) | S394 |
| TV-Q11 | Is the correlation chain intact through real venue interactions? | S399 (Spot) | S390, S391 |
| TV-Q12 | Does post-200 reconciliation work under real conditions? | S399 (Spot) | S390 |

No questions are added, removed, or modified. Only the target stages changed to
reflect the Spot-first sequencing and the consumed S390--S395 stage numbers.

---

## 5. Revised Capability Targets

The 10 original capability targets are preserved and retargeted to Spot:

| ID | Capability | Revised stage | Notes |
|---|---|---|---|
| TV-C1 | Real venue acceptance lifecycle | S399 | Spot testnet |
| TV-C2 | Real venue fill record fidelity | S399 | Spot `fills[]` array |
| TV-C3 | Real venue rejection lifecycle | S399 | Spot testnet |
| TV-C4 | Real venue rejection event fidelity | S399 | Spot testnet |
| TV-C5 | Real venue partial fill lifecycle | S400 | Structural or observed |
| TV-C6 | Lifecycle invariant fidelity under real data | S399, S400 | All 8 categories |
| TV-C7 | Persistence consistency under real data | S400 | KV + HTTP + ClickHouse |
| TV-C8 | Post-200 reconciliation under real conditions | S399 | Structural for Spot |
| TV-C9 | Compose E2E with real testnet | S398 (infra), S400 (full) | Dual-instance |
| TV-C10 | OMS read-path auditability under real data | S400 | Composite status query |

---

## 6. Entry Preconditions (Revised)

All preconditions from S389 remain met. New preconditions from the segmentation
wave:

| Precondition | Status | Source |
|---|---|---|
| Seven-state lifecycle proven | MET | S383 |
| Write-path per mode proven | MET | S385 |
| Rejection event path proven | MET | S386 |
| Persistence read-path proven | MET | S387 |
| BinanceFuturesTestnetAdapter exists | MET | Pre-segmentation |
| BinanceSpotTestnetAdapter exists | MET | S394 (NEW) |
| Segmented venue model | MET | S391 (NEW) |
| Config-driven segment enablement | MET | S393 (NEW) |
| Compose-level segment isolation | MET | S394 (NEW) |
| Spot ingest seeded | NOT MET | S397 will close |
| Dual-instance compose proven | NOT MET | S398 will close |

---

## 7. Scope Freeze Rules (Revised)

### What enters the wave

1. Real Spot testnet HTTP interactions via `BinanceSpotTestnetAdapter`.
2. Spot ingest binding seed (`binances` source in NATS).
3. Dual-instance compose proving segmented runtime concurrency.
4. Lifecycle state transitions validated against real Spot venue responses.
5. Fill record fidelity: real prices, quantities, fees from Spot testnet.
6. Rejection event fidelity: real Spot venue rejection codes and reasons.
7. Partial fill observation or structural feasibility proof (Spot).
8. Persistence surface consistency under real Spot event data.
9. Compose-level E2E proof with `venue_live` mode against Spot testnet.
10. ClickHouse rejection writer wiring (RG-1 closure).

### What does NOT enter the wave

See companion document:
[`testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md`](testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md)

### Scope amendment protocol

Unchanged from S389. Any scope addition requires:

1. Written justification linking to a governing question.
2. Proof that the addition does not violate any non-goal.
3. Explicit acknowledgment in the stage report that introduced it.

---

## 8. Risk Register (Revised)

Original risks from S389 remain applicable. New risks from segmented context:

| Risk | Likelihood | Mitigation |
|---|---|---|
| Spot testnet API unavailability | Medium | Same mitigation as Futures: idempotent tests, retriable smoke scripts |
| Spot testnet insufficient balance | Medium | Verify test account funding before S399; document top-up procedure |
| Partial fills hard to trigger on Spot | High | Accept structural proof (same policy as S389 for Futures); Spot market orders fill instantly |
| Dual-compose resource contention | Low | Lightweight single-symbol execution; compose resource limits if needed |
| Spot API rate limits stricter than Futures | Low | Single-symbol, low-frequency; retry policy handles 429 |
| Segmentation regression during wave | Low | Existing 39 segmentation tests provide guardrail |

---

## 9. Success Criteria for Wave Closure (Revised)

The evidence gate (S401) will evaluate:

1. **10/10 capabilities at FULL or SUBSTANTIAL** -- no PARTIAL or PENDING.
2. **12/12 governing questions ANSWERED or SUBSTANTIAL** -- no UNANSWERED.
3. **Zero non-goal violations.**
4. **Zero regressions** in existing test suites (including segmentation tests).
5. **All residual gaps from OMS Foundation (RG-1) closed** or explicitly
   deferred with justification.
6. **S395 residual gaps G1 (dual compose) and G3 (Spot ingest) closed.**

Verdict options: `PASSED`, `PASSED -- CONDITIONAL`, `FAILED`.

---

## 10. Links

- Original charter: [`testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](testnet-venue-execution-proof-wave-charter-and-scope-freeze.md)
- Original capabilities and non-goals: [`testnet-venue-execution-capabilities-questions-and-non-goals.md`](testnet-venue-execution-capabilities-questions-and-non-goals.md)
- Revised capabilities and non-goals: [`testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md`](testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md)
- Segmentation evidence gate: [`binance-spot-futures-segmentation-evidence-gate.md`](binance-spot-futures-segmentation-evidence-gate.md)
- Segmentation evidence matrix: [`binance-segmentation-evidence-matrix-residual-gaps-and-next-ceremony.md`](binance-segmentation-evidence-matrix-residual-gaps-and-next-ceremony.md)
- Canonical order model: [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md)
- Stage report: [`../stages/stage-s396-testnet-venue-execution-charter-refresh-report.md`](../stages/stage-s396-testnet-venue-execution-charter-refresh-report.md)
