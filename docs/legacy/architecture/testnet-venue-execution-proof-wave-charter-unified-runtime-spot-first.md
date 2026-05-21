# Testnet Venue Execution Proof Wave -- Charter on Unified Runtime (Spot-First)

**Wave:** Testnet Venue Execution Proof (unified runtime)
**Charter stage:** S404
**Prior charters:** S389 (original), S396 (segmented refresh)
**Date frozen:** 2026-03-22
**Predecessor wave:** Unified Segment Runtime Foundation (S398--S403, PASSED -- FULL DELIVERY)
**Authority:** This document freezes wave scope. Changes require a new stage.

---

## 1. Strategic Context

### 1.1 What Changed Since S396

S396 opened the Testnet Venue Execution Proof Wave under a **multi-binary per
segment** architecture: one execute binary per segment, one compose overlay per
segment, one config per segment. That model was correct at the time.

The Unified Segment Runtime Foundation Wave (S398--S403) then intervened and
resolved four structural debts, delivering:

| Capability | Evidence |
|---|---|
| Unified config model (`venue.segments.*`) | S399: 26 validation tests, `execute-unified.jsonc` |
| Merged binding seed (single `make seed` for all segments) | S400: `seed-configctl.sh --merge` |
| Multi-adapter runtime projection (single binary, N adapters) | S400: `VenueAdapterRouter`, intent-to-segment dispatch |
| Source-based routing with fail-closed rejection | S401: NATS consumer filtering, leakage invariant tests |
| Single-compose coexistence (Spot + Futures in one stack) | S402: `docker-compose.unified.yaml`, 7-phase smoke |
| Global `dry_run` preservation across all segments | S403: 10/10 FULL, 12/12 FULL |

The S396 charter is now **architecturally stale** in two dimensions:

1. **Runtime model.** S396 assumed dual-instance compose with per-segment
   binaries. The runtime is now unified: single binary, single config, single
   compose, multi-adapter. All venue execution proof must target this model.
2. **Stage numbering.** S398--S403 consumed the stage numbers that S396
   allocated for venue execution. The refreshed stages begin at S405.

The 12 governing questions (TV-Q1 through TV-Q12) and 10 capability targets
(TV-C1 through TV-C10) from S396 remain **structurally valid**. They are
retargeted to the unified runtime and resequenced in this charter.

### 1.2 Why This Wave Now

The unified runtime is proven but only under **dry-run and structural test**
conditions. No stage has yet submitted a real order to a real venue. The
platform's credibility requires proving that:

- The `venue_live` write-path produces correct lifecycle transitions against a
  real testnet.
- Fill records carry accurate prices, quantities, and fees from real venue
  responses.
- Rejections from real venue errors propagate correctly through the lifecycle.
- The unified runtime's multi-adapter routing, source dispatch, and compose
  orchestration hold under real HTTP interactions with non-deterministic latency
  and venue-side validation.

Until this wave passes, the system is provably correct only in simulation.

### 1.3 Spot-First Strategy (Preserved from S396)

The rationale from S396 Section 2 remains valid and is incorporated by
reference:

| Factor | Rationale |
|---|---|
| Adapter freshness | `BinanceSpotTestnetAdapter` built in S394, extended through S402 |
| API simplicity | Spot REST `/api/v3/order` is simpler than Futures `/fapi/v1/order` |
| Fill model | Spot returns `fills[]` array directly -- exercises multi-fill aggregation |
| Risk isolation | No leverage/margin complexity on Spot testnet |
| Segmentation validation | Spot-first validates unified runtime under real load |

**Spot-first means:** all TV-Q1 through TV-Q12 are answered against Binance
Spot testnet. Futures proof is a separate follow-on wave.

---

## 2. Entry Preconditions

All preconditions are met. This is the first wave where every prerequisite is
satisfied at charter time.

| Precondition | Status | Source |
|---|---|---|
| Seven-state lifecycle proven | MET | S383 (49/49 transitions) |
| Write-path per execution mode proven | MET | S385 |
| Rejection event path proven | MET | S386 |
| Persistence read-path proven | MET | S387 |
| OMS Foundation evidence gate | MET | S388 (PASS) |
| BinanceSpotTestnetAdapter exists | MET | S394 |
| Segmented venue model | MET | S391 |
| Config-driven segment enablement | MET | S393, S399 (unified) |
| Unified config model | MET | S399 (10/10 FULL) |
| Multi-adapter runtime projection | MET | S400 |
| Source-based routing + leakage hardening | MET | S401 |
| Single-compose coexistence | MET | S402 |
| Unified runtime evidence gate | MET | S403 (PASS -- FULL DELIVERY) |
| Spot ingest bindings seeded | MET | S397 (`binances` source) |
| Spot testnet credentials provisioned | REQUIRED | Must be provisioned before S405 |

---

## 3. Wave Blocks (Ordered)

| Block | Stage | Title | Governing Qs |
|---|---|---|---|
| B0 | S404 | Charter and scope freeze (this document) | -- |
| B1 | S405 | Spot real venue connectivity, acceptance, and fill proof | TV-Q1, TV-Q2, TV-Q11, TV-Q12 |
| B2 | S406 | Spot real rejection and partial-fill evidence | TV-Q3, TV-Q4, TV-Q5, TV-Q6 |
| B3 | S407 | Unified runtime read-path and auditability under real responses | TV-Q7, TV-Q8, TV-Q10 |
| B4 | S408 | Unified compose E2E proof against real Spot testnet | TV-Q9 |
| B5 | S409 | Evidence gate: Testnet Venue Execution Proof (final) | All |

### Block Details

#### B1 -- S405: Spot Real Venue Connectivity, Acceptance, and Fill Proof

The core connectivity and happy-path proof. Against the Binance Spot testnet
via the unified runtime:

- **TV-Q1:** `venue_live` write-path produces correct `submitted -> accepted -> filled`
  lifecycle transitions on a real Spot market order.
- **TV-Q2:** Fill records carry real `price`, `qty`, `commission`, `commissionAsset`
  from Spot testnet response `fills[]` array.
- **TV-Q11:** Correlation chain (`CorrelationID`, `CausationID`) remains intact
  through real Spot HTTP request/response cycle.
- **TV-Q12:** Post-200 reconciliation path confirmed structurally sound under
  real conditions (structural proof acceptable per S396 policy).

Deliverables:
- Real Spot testnet order submission via `BinanceSpotTestnetAdapter` in
  `venue_live` mode on unified runtime.
- Evidence log showing lifecycle transitions with real venue data.
- Fill record audit showing real prices and fees.
- Correlation chain trace through the pipeline.
- Smoke script or test harness for repeatable proof.

#### B2 -- S406: Spot Real Rejection and Partial-Fill Evidence

The rejection and edge-case proof. Against the Binance Spot testnet:

- **TV-Q3:** Lifecycle correctly transitions to `rejected` on real Spot venue
  rejection (insufficient balance, invalid params, auth failure).
- **TV-Q4:** `VenueOrderRejectedEvent` carries real HTTP status, Spot error
  code (`-1013`, `-2010`, etc.), and human-readable reason.
- **TV-Q5:** Partial fill observation or structural proof from Spot testnet.
  Policy: Spot market orders typically fill instantly; structural proof
  combining S383 domain evidence + S394 multi-fill aggregation tests +
  `mapBinanceStatus("PARTIALLY_FILLED")` mapping is acceptable.
- **TV-Q6:** Quantity monotonicity under partial fills (structural or observed).

Deliverables:
- Real Spot rejection triggered and captured with venue error details.
- `VenueOrderRejectedEvent` audit with real venue payload.
- Partial fill evidence (real or structural, per policy).
- Monotonicity invariant verification.

#### B3 -- S407: Unified Runtime Read-Path and Auditability Under Real Responses

Persistence and observability proof with real data flowing:

- **TV-Q7:** KV projections, HTTP query responses, and ClickHouse rows agree on
  terminal state after real Spot venue interactions.
- **TV-Q8:** ClickHouse rejection writer wired and producing correct rows under
  real rejection events (RG-1 closure from OMS Foundation wave).
- **TV-Q10:** Sustained correct behavior over multiple consecutive order cycles
  (minimum 3 cycles with real Spot venue).

Deliverables:
- Persistence consistency audit across KV, HTTP, and ClickHouse after real orders.
- ClickHouse rejection row verification with real venue rejection data.
- Multi-cycle sustained operation evidence.
- Read-path query results matching write-path events.

#### B4 -- S408: Unified Compose E2E Proof Against Real Spot Testnet

Full pipeline proof in compose with real venue:

- **TV-Q9:** Full compose pipeline (`derive -> execute -> store`) operates
  correctly in `venue_live` mode against Spot testnet on the unified runtime.
  This builds on S402's dry-run coexistence proof and elevates it to real venue
  interaction.

Deliverables:
- Compose stack boots with unified config in `venue_live` mode (Spot enabled,
  Futures disabled or dry-run).
- End-to-end flow: market data ingress -> signal derivation -> execution intent
  -> real Spot order -> fill/rejection -> persistence -> query.
- Smoke script proving the full pipeline with real venue responses.
- Evidence that the unified runtime's source routing, adapter projection, and
  decorator pipeline function correctly under real conditions.

#### B5 -- S409: Evidence Gate (Final)

Wave closure ceremony:

- Evaluate all 12 governing questions against Spot evidence on unified runtime.
- Classify all 10 capabilities.
- Regression verification against all prior waves (OMS, Segmentation, Unified
  Runtime, Exchange Listening, Activation).
- Non-goal compliance check.
- Verdict: `PASSED`, `PASSED -- CONDITIONAL`, or `FAILED`.
- Recommendation for next ceremony.

---

## 4. Governing Questions (Retargeted to Unified Runtime)

The 12 governing questions are preserved verbatim from S389/S396. The only
change is the target stage, reflecting execution on the unified runtime.

| ID | Question | S396 target | S404 target |
|---|---|---|---|
| TV-Q1 | Does `venue_live` produce correct lifecycle transitions on real acceptance + fill? | S399 | **S405** |
| TV-Q2 | Do fill records carry accurate real prices, quantities, and fees? | S399 | **S405** |
| TV-Q3 | Does the lifecycle correctly transition to `rejected` on real venue rejection? | S399 | **S406** |
| TV-Q4 | Does `VenueOrderRejectedEvent` carry real venue rejection code and reason? | S399 | **S406** |
| TV-Q5 | Can partial fill be observed or structurally proven from testnet? | S400 | **S406** |
| TV-Q6 | Does quantity monotonicity hold under real partial fills? | S400 | **S406** |
| TV-Q7 | Do KV, HTTP, and ClickHouse agree on terminal state after real interactions? | S400 | **S407** |
| TV-Q8 | Is the ClickHouse rejection writer wired and producing correct rows? | S400 | **S407** |
| TV-Q9 | Does the full compose pipeline work in `venue_live` against testnet? | S400 | **S408** |
| TV-Q10 | Does the system sustain correct behavior over multiple order cycles? | S400 | **S407** |
| TV-Q11 | Is the correlation chain intact through real venue interactions? | S399 | **S405** |
| TV-Q12 | Does post-200 reconciliation work under real conditions? | S399 | **S405** |

---

## 5. Capability Targets (Retargeted to Unified Runtime)

The 10 capability targets are preserved from S396 and retargeted:

| ID | Capability | Stage | Notes |
|---|---|---|---|
| TV-C1 | Real Spot venue acceptance lifecycle | S405 | Unified runtime, `venue_live` mode |
| TV-C2 | Real Spot venue fill record fidelity | S405 | Spot `fills[]` array, real prices |
| TV-C3 | Real Spot venue rejection lifecycle | S406 | Real Spot rejection codes |
| TV-C4 | Real Spot venue rejection event fidelity | S406 | Real HTTP status + error payload |
| TV-C5 | Real Spot venue partial fill lifecycle | S406 | Structural or observed |
| TV-C6 | Lifecycle invariant fidelity under real data | S405--S407 | All 8 invariant categories |
| TV-C7 | Persistence consistency under real data | S407 | KV + HTTP + ClickHouse |
| TV-C8 | Post-200 reconciliation under real conditions | S405 | Structural proof acceptable |
| TV-C9 | Compose E2E with real Spot testnet | S408 | Unified compose, single binary |
| TV-C10 | OMS read-path auditability under real data | S407 | Composite status query |

---

## 6. What Enters (Scope Boundary)

1. Real Spot testnet HTTP interactions via `BinanceSpotTestnetAdapter` on the
   unified runtime (single binary, single config, `execute-unified.jsonc`).
2. Lifecycle state transitions validated against real Spot venue responses.
3. Fill record fidelity: real prices, quantities, fees from Spot testnet.
4. Rejection event fidelity: real Spot venue error codes and reasons.
5. Partial fill observation or structural feasibility proof (Spot).
6. Persistence surface consistency (KV, HTTP, ClickHouse) under real event data.
7. ClickHouse rejection writer wiring (RG-1 closure).
8. Compose-level E2E proof with `venue_live` mode against Spot testnet.
9. Correlation chain integrity through real venue interactions.
10. Multi-cycle sustained operation proof.

---

## 7. What Does NOT Enter (Scope Freeze)

See companion document:
[`testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md`](testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md)

Summary of frozen exclusions (full list in companion):

| Category | Key exclusions |
|---|---|
| Venue scope | Futures proof, multi-exchange, mainnet |
| Order types | Limit, stop-loss, OCO (market only) |
| OMS scope | Full OMS, cancel API, amendment, order book |
| Risk | Portfolio risk, P&L, margin management |
| Infrastructure | Dashboards, CI/CD, performance benchmarks |
| Architecture | Lifecycle redesign, actor topology changes, domain model extension |
| Segmentation | Reopen segmentation wave, per-segment dry_run, per-segment kill switch |
| Runtime | Redesign unified runtime, per-segment control gates |

### Scope Amendment Protocol

Unchanged from S389/S396. Any scope addition requires:

1. Written justification linking to a governing question.
2. Proof that the addition does not violate any non-goal.
3. Explicit acknowledgment in the stage report that introduced it.

---

## 8. Architectural Assumptions

The unified runtime is treated as a stable foundation. This wave **consumes**
it, not **modifies** it.

| Assumption | Basis |
|---|---|
| Single binary boots Spot adapter from unified config | S400, S402 proven |
| Source-based routing dispatches intents correctly | S401 proven (79 tests, 0 regressions) |
| `dry_run=false` activates `venue_live` write-path | S385 proven at domain/application level |
| DryRunSubmitter bypass under `dry_run=false` | S379 decorator pipeline contract |
| NATS consumer filtering prevents cross-segment delivery | S401 proven |
| Global `dry_run` flag wraps all adapters uniformly | S403 evidence gate FULL |
| Compose stack supports `venue_live` mode | S402 structural; S408 proves under real venue |

**Key design decision:** For `venue_live` Spot-only execution, the unified
config sets `spot.enabled=true` with `dry_run=false`. Futures segment may be
disabled entirely or kept enabled with the global `dry_run=true` override
replaced by a Spot-only config. The exact config pattern will be established in
S405 and documented.

---

## 9. Risk Register

Risks from S396 remain applicable. Additional risks from unified runtime context:

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Spot testnet API unavailability | Medium | Medium | Idempotent tests, retriable smoke scripts |
| Spot testnet insufficient balance | Medium | High | Verify test account funding before S405; document top-up procedure |
| Partial fills hard to trigger on Spot | High | Low | Structural proof acceptable (same policy as S389/S396) |
| Unified runtime `venue_live` mode exposes untested path | Medium | High | S405 is specifically scoped to prove this path incrementally |
| `dry_run=false` on unified binary activates ALL segment adapters | Low | High | Config must enable only Spot; Futures disabled or not present in unified config for `venue_live` |
| Spot API rate limits | Low | Low | Single-symbol, low-frequency execution |
| ClickHouse rejection writer (RG-1) not wired | Known gap | Medium | S407 closes explicitly |
| Regression in unified runtime under real latency | Low | Medium | S403 evidence gate provides baseline; S409 regression check |

---

## 10. Success Criteria for Wave Closure

The evidence gate (S409) will evaluate:

1. **10/10 capabilities at FULL or SUBSTANTIAL** -- no PARTIAL or PENDING.
2. **12/12 governing questions ANSWERED or SUBSTANTIAL** -- no UNANSWERED.
3. **Zero non-goal violations.**
4. **Zero regressions** in all prior test suites (OMS, Segmentation, Unified
   Runtime, Exchange Listening, Activation).
5. **RG-1 (ClickHouse rejection writer) closed** or explicitly deferred with
   justification.
6. **All S403 residual limitations remain non-blocking** under real venue load.

Verdict options: `PASSED`, `PASSED -- CONDITIONAL`, `FAILED`.

---

## 11. Relationship to Prior Charters

| Charter | Status | Relationship |
|---|---|---|
| S389 (original) | Superseded by S396 | Founded TV-Q1--TV-Q12, TV-C1--TV-C10, NG-1--NG-22 |
| S396 (segmented refresh) | **Superseded by this charter (S404)** | Retargeted to Spot-first; assumed multi-binary architecture |
| S398 (unified runtime) | CLOSED (S403 PASS) | Foundation this wave builds on |
| S404 (this charter) | **ACTIVE** | Retargets TV-Q1--TV-Q12 to unified runtime |

The chain of charter authority:

```
S389 (original) -> S396 (segmented refresh) -> S404 (unified runtime refresh)
                                                 ^-- ACTIVE CHARTER
```

S389 and S396 are historical. Only S404 governs execution from this point.

---

## 12. References

| Reference | Link |
|---|---|
| Companion: capabilities, questions, non-goals | [`testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md`](testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md) |
| S403 evidence gate report | [`../stages/stage-s403-unified-segment-runtime-evidence-gate-report.md`](../stages/stage-s403-unified-segment-runtime-evidence-gate-report.md) |
| S396 charter refresh (superseded) | [`testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md`](testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md) |
| S389 original charter (superseded) | [`testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](testnet-venue-execution-proof-wave-charter-and-scope-freeze.md) |
| Unified runtime charter | [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Unified runtime evidence gate | [`unified-segment-runtime-evidence-gate.md`](unified-segment-runtime-evidence-gate.md) |
| Canonical order model | [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md) |
| Stage INDEX | [`../stages/INDEX.md`](../stages/INDEX.md) |
