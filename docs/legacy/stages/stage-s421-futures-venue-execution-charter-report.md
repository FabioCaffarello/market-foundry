# Stage S421: Futures Venue Execution Proof Wave Charter Report

> Wave: Futures Venue Execution Proof (Post-Simplification, Phase 47)
> Stage: S421 -- Charter and Scope Freeze
> Date: 2026-03-23
> Predecessor: S420 -- Runtime Simplification Evidence Gate (PASS -- FULL DELIVERY)

---

## 1. Executive Summary

S421 opens the Futures Venue Execution Proof Wave (Phase 47) with a formal charter, scope freeze, and canonical surface contract. This wave proves that the canonical OMS lifecycle behaves correctly when exercised against real Binance Futures testnet responses on the unified runtime, using the consolidated config/compose surface established by the Runtime Simplification wave.

**Key decisions:**

1. **Wave authorized.** The S420 evidence gate closed with PASS -- FULL DELIVERY and formally authorized this wave with 5 explicit conditions. All conditions are encoded in the charter and surface contract.

2. **Scope frozen.** 10 capabilities, 12 governing questions, 5 execution blocks (S422--S426). No scope inflation permitted without written justification and non-goal compliance proof.

3. **Canonical surface contract established.** 3 configs, 3 compose files, frozen runtime topology. No new per-segment or "temporary" artifacts. This is the first wave to carry an explicit surface contract, inheriting the simplification discipline from Phase 46.

4. **55 non-goals locked.** 40 inherited from prior waves, 8 preserved from runtime simplification, 7 new surface constraints. Total coverage: venue, OMS, risk, infrastructure, architecture, segmentation, runtime, Futures-specific, and surface discipline.

5. **5 stages ordered.** S422 (connectivity/fill) -> S423 (rejection/partial fill) -> S424 (read-path/parity) -> S425 (compose E2E) -> S426 (evidence gate).

---

## 2. Stage Purpose

S421 is a charter-only stage. It produces no code changes. Its deliverables are:

1. Wave charter with canonical surface contract.
2. Capabilities, questions, non-goals, and surface constraints document.
3. This stage report.

The stage serves three functions:

- **Authorization gate:** Formalizes the S420 authorization into actionable scope.
- **Entropy guard:** Freezes the canonical surface to prevent reintroduction of the operational entropy that Phase 46 eliminated.
- **Execution blueprint:** Orders stages with explicit dependencies and config/compose assignments.

---

## 3. Wave Structure

### 3.1 Blocks and Stages

| Stage | Block | Title | Governing Qs |
|---|---|---|---|
| S421 | B0 | Charter and scope freeze | -- |
| S422 | B1 | Futures real venue connectivity and acceptance/fill proof | FV-Q1, FV-Q2, FV-Q11, FV-Q12 |
| S423 | B2 | Futures real rejection and partial-fill evidence | FV-Q3, FV-Q4, FV-Q5, FV-Q6 |
| S424 | B3 | Unified runtime read-path, auditability, and segment parity | FV-Q7, FV-Q8, FV-Q10 |
| S425 | B4 | Unified compose E2E proof with Futures live execution path | FV-Q9 |
| S426 | B5 | Evidence gate (final) | All |

### 3.2 Dependencies

```
S421 (charter) -> S422 (fill) -> S423 (rejection) -> S424 (read-path) -> S425 (E2E) -> S426 (gate)
```

Each stage depends on its predecessor. No parallel execution between stages.

### 3.3 Config/Compose per Stage

All proof stages (S422--S425) use:
- **Config:** `deploy/configs/execute-venue-live.jsonc`
- **Compose:** `deploy/compose/docker-compose.yaml` + `deploy/compose/docker-compose.venue-live.yaml`

No deviations. No per-stage config/compose variants.

---

## 4. Governing Questions

12 questions organized by domain:

| ID | Question | Stage |
|---|---|---|
| FV-Q1 | Does `venue_live` write-path produce correct lifecycle on real Futures acceptance/fill? | S422 |
| FV-Q2 | Do fill records carry real `avgPrice`, `executedQty`, `cumQuote`, commission? | S422 |
| FV-Q3 | Does lifecycle transition to `rejected` on real Futures rejection? | S423 |
| FV-Q4 | Does `VenueOrderRejectedEvent` carry real Futures error code and reason? | S423 |
| FV-Q5 | Can `partially_filled` be observed or structurally proven from Futures? | S423 |
| FV-Q6 | Does quantity monotonicity hold under real Futures partial fills? | S423 |
| FV-Q7 | Do KV, HTTP, and ClickHouse agree on terminal state after Futures? | S424 |
| FV-Q8 | Does ClickHouse rejection writer handle Futures rejection events? | S424 |
| FV-Q9 | Does full compose pipeline operate with Futures `venue_live`? | S425 |
| FV-Q10 | Does system sustain correct behavior over multiple Futures cycles? | S424 |
| FV-Q11 | Does correlation chain remain intact through Futures interactions? | S422 |
| FV-Q12 | Does post-200 reconciliation work under Futures conditions? | S422 |

---

## 5. Canonical Surface (Frozen)

### 5.1 Config

| File | Purpose | Status |
|---|---|---|
| `execute.jsonc` | Paper simulator, standalone | Canonical (not used in proof) |
| `execute-unified.jsonc` | Both segments, dry_run=true | Canonical (testing) |
| `execute-venue-live.jsonc` | Both segments, dry_run=false | **Canonical (proof primary)** |

No new config files permitted (NG-41, NG-50).

### 5.2 Compose

| File | Purpose | Status |
|---|---|---|
| `docker-compose.yaml` | Base stack (9 services) | Canonical (always present) |
| `docker-compose.unified.yaml` | Dry-run overlay | Canonical (testing) |
| `docker-compose.venue-live.yaml` | Real testnet overlay | **Canonical (proof primary)** |

No new compose overlays permitted (NG-46, NG-47, NG-49).

### 5.3 Runtime Topology (Frozen)

- Single execute binary with `SegmentRouter` dispatch.
- Source-prefix routing: `binances` -> Spot, `binancef` -> Futures.
- NATS: `execution.intents.>`, `execution.events.>` with segment-scoped consumers.
- KV: `execution-lifecycle` (shared, both segments).
- ClickHouse: `executions` (20-column, segment-agnostic).
- Lifecycle: 7 states, 49/49 transitions proven.
- Adapters: `BinanceSpotTestnetAdapter`, `BinanceFuturesTestnetAdapter`.

---

## 6. Non-Goals (55 Total)

### Summary by Domain

| Domain | IDs | Count |
|---|---|---|
| Venue and market scope | NG-1 -- NG-5 | 5 |
| OMS and lifecycle scope | NG-6 -- NG-10 | 5 |
| Risk, portfolio, strategy | NG-11 -- NG-13 | 3 |
| Infrastructure and operations | NG-14 -- NG-18 | 5 |
| Architectural scope | NG-19 -- NG-22 | 4 |
| Segmentation scope | NG-23 -- NG-27 | 5 |
| Runtime scope | NG-28 -- NG-32 | 5 |
| Futures-specific exclusions | NG-33 -- NG-40 | 8 |
| Runtime simplification preserved | NG-41 -- NG-48 | 8 |
| Surface constraints (new) | NG-49 -- NG-55 | 7 |
| **Total** | | **55** |

### Critical Non-Goals for This Wave

| ID | Non-goal | Why critical |
|---|---|---|
| NG-1 | No mainnet | Safety boundary |
| NG-2 | No multi-exchange | Scope boundary |
| NG-6 | No full OMS | Scope boundary |
| NG-11 | No portfolio risk | Scope boundary |
| NG-41 | No new configs | Surface discipline |
| NG-46 | No per-segment compose | Surface discipline |
| NG-49 | No temporary compose | Surface discipline |
| NG-50 | No Futures-only config | Surface discipline |

Full enumeration in companion document: [`futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md`](../architecture/futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md).

---

## 7. Residual Gaps Carried Forward

| Gap | Severity | Disposition |
|---|---|---|
| G-4: Fee semantic divergence | Medium | Monitored; flag if it affects fill fidelity |
| G-1: No parallel dual-segment live proof | Low | Deferred |
| G-2/RG-4: Segment-scoped list queries | Low | Deferred |
| G-3: Rejection code in JSON metadata | Low | Carried |
| G-5: No per-segment health check | Low | Carried |
| RG-2: Partial fill live observation | Low | Elevated for Futures (S423) |
| RG-3: Latest-only KV semantics | Low | Deferred |
| RG-16: 97 untracked docs | Low | Deferred |
| RG-17: Smoke script naming | Low | Deferred |
| RG-18: Doc suitability not assessed | Low | Deferred |

1 Medium, 9 Low. Zero blocking.

---

## 8. Success Criteria

The evidence gate (S426) evaluates:

1. 10/10 capabilities at FULL or SUBSTANTIAL.
2. 12/12 governing questions ANSWERED or SUBSTANTIAL.
3. Zero non-goal violations (55 total).
4. Zero regressions (S370--S420 full chain).
5. Canonical surface contract respected (no unauthorized deviations).
6. Segment parity demonstrated (Futures + Spot in unified read surfaces).
7. G-4 fee divergence assessed under real Futures data.

Verdict options: `PASS`, `PASS -- CONDITIONAL`, `FAIL`.

---

## 9. Deliverables

| # | Artifact | Path |
|---|----------|------|
| 1 | Wave charter and canonical surface contract | `docs/architecture/futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md` |
| 2 | Capabilities, questions, non-goals, and surface constraints | `docs/architecture/futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md` |
| 3 | Stage report (this document) | `docs/stages/stage-s421-futures-venue-execution-charter-report.md` |

---

## 10. Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| Wave formally opened with scope frozen | DONE -- charter frozen 2026-03-23 |
| Canonical surface explicit and auditable | DONE -- Section 5 of charter, Section 4 of capabilities doc |
| Non-goals clear (55 total) | DONE -- Section 3 of capabilities doc |
| Next stages ordered with rigor | DONE -- S422 -> S423 -> S424 -> S425 -> S426 |
| No new "temporary" compose | ENFORCED -- NG-49 |
| No new parallel config "for Futures" | ENFORCED -- NG-50 |
| No multi-exchange, mainnet, OMS, portfolio risk | ENFORCED -- NG-1, NG-2, NG-6, NG-11 |
| No runtime redesign | ENFORCED -- NG-28, NG-42 |

**Stage verdict: COMPLETE. Wave is formally open.**

---

## 11. Next Stage

**S422: Futures Real Venue Connectivity and Acceptance/Fill Proof**

- Prove `submitted -> accepted -> filled` against real Binance Futures testnet.
- Validate fill record fidelity (`avgPrice`, `executedQty`, `cumQuote`, commission).
- Verify correlation chain integrity.
- Config: `execute-venue-live.jsonc`. Compose: base + venue-live overlay.
- Precondition: Futures testnet credentials provisioned.

---

## 12. Links

| Reference | Link |
|---|---|
| Charter and canonical surface contract | [`../architecture/futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md`](../architecture/futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md) |
| Capabilities, questions, non-goals | [`../architecture/futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md`](../architecture/futures-venue-execution-capabilities-questions-non-goals-and-surface-constraints.md) |
| S420 evidence gate | [`stage-s420-runtime-simplification-evidence-gate-report.md`](stage-s420-runtime-simplification-evidence-gate-report.md) |
| S420 authorization | [`../architecture/runtime-simplification-evidence-gate-and-futures-proof-authorization.md`](../architecture/runtime-simplification-evidence-gate-and-futures-proof-authorization.md) |
| S420 evidence matrix | [`../architecture/runtime-simplification-evidence-matrix-residual-gaps-and-next-ceremony.md`](../architecture/runtime-simplification-evidence-matrix-residual-gaps-and-next-ceremony.md) |
| Cumulative gate history | 10 consecutive PASS verdicts across Phases 38--46 |
