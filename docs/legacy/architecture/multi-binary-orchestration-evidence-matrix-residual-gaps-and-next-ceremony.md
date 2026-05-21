# Multi-Binary Orchestration: Evidence Matrix, Residual Gaps, and Next Ceremony

Companion document to the evidence gate. Contains the detailed evidence matrix,
honest gap inventory, and fact-based recommendation for the next strategic ceremony.

---

## 1. Evidence Matrix

### 1.1 Structural Audit Evidence (S371)

| Item | Evidence | Quality |
|------|----------|---------|
| Binary inventory (7 operational + 2 infra) | Registry audit, compose topology | VERIFIED |
| Stream ownership (9 streams, single-writer) | Code audit of registries | VERIFIED |
| Handoff mapping (11 handoffs, H1–H11) | Subject/payload/invariant audit | VERIFIED |
| Correlation chain (8 UUID hops) | Static trace through code | VERIFIED |
| KV bucket ownership (7 buckets) | Store-only write access confirmed | VERIFIED |
| Domain isolation (no cross-binary imports) | raccoon-cli arch-guard | VERIFIED |
| Envelope contract (CBOR, dedup keys) | Code audit | VERIFIED |
| Risk register (7 risks: 2 MEDIUM, 5 LOW) | Documented with mitigations | VERIFIED |

### 1.2 Compose Wiring Evidence (S372)

| Item | Evidence | Quality |
|------|----------|---------|
| 9 services boot to healthy | smoke-compose-wiring phase 1 | AUTOMATED |
| NATS JetStream operational | smoke-compose-wiring phase 2 | AUTOMATED |
| 9 streams created | smoke-compose-wiring phase 3 | AUTOMATED |
| 44 durable consumers bound | smoke-compose-wiring phase 4 | AUTOMATED |
| Cross-binary request/reply | smoke-compose-wiring phase 5 | AUTOMATED |
| Service isolation (PID namespace) | smoke-compose-wiring phase 6 | AUTOMATED |
| Port allocation correct | smoke-compose-wiring phase 7 | AUTOMATED |
| Dependency chain integrity | smoke-compose-wiring phase 8 | AUTOMATED |
| Boot order documented (5 phases) | Architecture doc | DOCUMENTED |
| 10 limitations catalogued | Architecture doc | DOCUMENTED |

### 1.3 End-to-End Pipeline Evidence (S373)

| Item | Evidence | Quality |
|------|----------|---------|
| INV-1: Strategy type identity preserved | Structural + integration test | AUTOMATED |
| INV-2: Direction→side deterministic (3 dirs) | Structural + integration test | AUTOMATED |
| INV-3: Correlation chain derive→fill | Integration test (separate NATS connections) | AUTOMATED |
| INV-4: Risk type/disposition explicit | Structural test | AUTOMATED |
| INV-5: Strategy timestamp (not time.Now) | Structural test | AUTOMATED |
| INV-6: Wrong type filtered by subject | Structural test | AUTOMATED |
| INV-7: Flat→side=none, qty=0 | Structural + integration test | AUTOMATED |
| CTRL-1: Gate halt blocks fills | Integration test | AUTOMATED |
| CTRL-2: Gate resume enables fills | Integration test | AUTOMATED |
| MB-1: No shared Go state between binaries | Integration test (separate NATS conns) | AUTOMATED |
| KV readable from separate binary | Integration test S373-MB-3 | AUTOMATED |
| ClickHouse persistence E2E | Smoke phase 9 | AUTOMATED |
| Analytical endpoints return data | Smoke phase 9 | AUTOMATED |
| Composite chains with correlation | Smoke phase 10 | AUTOMATED |
| 12-phase compose smoke | smoke-e2e-multi-binary | AUTOMATED |
| 7 limitations catalogued | Architecture doc | DOCUMENTED |

### 1.4 Failure Isolation Evidence (S374)

| Item | Evidence | Quality |
|------|----------|---------|
| FI-1: Derive restart → others unaffected | Smoke + structural test | AUTOMATED |
| FI-2: Execute restart → derive continues | Smoke + structural test | AUTOMATED |
| FI-3: Store restart → derive/execute safe | Smoke + structural test | AUTOMATED |
| FI-4: Pipeline resumes after restart cycle | Smoke phase 4 | AUTOMATED |
| FI-5: Stream counts non-decreasing | Smoke phase 5 | AUTOMATED |
| FI-6: Tracker metrics independent | Smoke phase 6 + structural test | AUTOMATED |
| Durable consumer spec stable | TestS374 DurableConsumerSpecStable | AUTOMATED |
| Redelivery produces identical output | TestS374 ActorHandlesRedelivery | AUTOMATED |
| Staleness guard protects after restart | TestS374 StalenessGuardProtectsAfterRestart | AUTOMATED |
| Tracker survives actor recreation | TestS374 TrackerSurvivesActorRecreation | AUTOMATED |
| Gate safety on restart (KV unavailable) | TestS374 GateSafetyOnRestart | AUTOMATED |
| 6 limitations catalogued | Architecture doc | DOCUMENTED |

### 1.5 Regression Evidence

| Module | Packages tested | Result |
|--------|----------------|--------|
| internal/actors | 4 | ALL PASS |
| internal/domain | 8 | ALL PASS |
| internal/shared | 7 | ALL PASS |
| internal/application | 8 | ALL PASS |
| **Total** | **27 packages** | **ZERO FAILURES** |

---

## 2. Residual Gaps

Honest inventory of what the wave did NOT prove, with severity and recommendation.

### 2.1 Acknowledged Gaps (Not Blocking)

| ID | Gap | Severity | Why Not Blocking | Recommendation |
|----|-----|----------|------------------|----------------|
| G1 | Single strategy family tested (mean_reversion_entry) | LOW | Architecture is family-generic; routing by NATS subject is parametric | Will be covered when new families are added |
| G2 | Single symbol/timeframe (binancef/btcusdt/60) | LOW | Multi-symbol proven in earlier waves (S364–S369) | No action needed |
| G3 | Paper venue only | LOW | Venue is a leaf node; real venue integration is a separate wave | Covered by future venue wave |
| G4 | Writer flush timing warn-not-fail | LOW | Batch flush is a performance trade-off; data eventually persists | Monitor in production |
| G5 | NATS as infrastructure failure point not tested | MEDIUM | Shared infrastructure failure affects all binaries equally; not a wiring concern | Separate operational resilience wave if needed |
| G6 | Only sequential restarts tested | LOW | Concurrent failures are chaos engineering scope | Separate operational concern |
| G7 | Only graceful restart (SIGTERM), not crash (SIGKILL) | LOW | Redelivery determinism proven structurally; JetStream handles crash via durable consumers | Acceptable for proof wave |
| G8 | No back-pressure or load testing | LOW | Orthogonal to correctness; performance is a separate concern | Separate performance wave if needed |
| G9 | No TLS between services | LOW | Development environment; production hardening is a separate concern | Covered by production hardening wave |
| G10 | No long-duration endurance testing | LOW | Separate operational concern | Not needed for proof wave |
| G11 | Writer buffer loss on crash | KNOWN | Documented trade-off since S280; batch performance vs. crash safety | Acceptable; stream retention enables replay |
| G12 | No cross-stream ordering guarantee | BY DESIGN | Pipeline tolerates by design; each stream is independently ordered | Not a gap |

### 2.2 Risk Register Final State

| Risk | Severity | Stage Introduced | Current Status |
|------|----------|-----------------|----------------|
| RISK-1: Transitional bridge in execute | LOW | S371 | Open — paper-mode only, well-scoped |
| RISK-2: Dual intake paths in execute | MEDIUM | S371 | Mitigated — staleness + activation gate |
| RISK-3: Event metadata loss in KV | LOW | S371 | Accepted — ClickHouse preserves full metadata |
| RISK-4: Stream creation timing | MEDIUM | S371 | Resolved — S372 smoke validates stream existence |
| RISK-5: Gateway depends on store | LOW | S371 | Verified — compose dependency chain enforced |
| RISK-6: Writer buffer eviction | LOW | S371 | Accepted — counter + stream retention |
| RISK-7: NATS reconnection | LOW | S371 | Mitigated — dedup + idempotent processing |

**No new risks introduced by the wave. RISK-4 resolved. Others stable.**

---

## 3. Wave Quantitative Summary

| Metric | Value |
|--------|-------|
| Stages executed | 5 (S370–S374) |
| Governing questions | 8 (7 fully answered, 1 partial) |
| Capabilities proven | 10 (9 FULL, 1 SUBSTANTIAL) |
| Go tests added | 14 (8 structural + 4 integration + 2 structural isolation) |
| Smoke scripts added | 3 (27 automated phases total) |
| Architecture docs produced | 10 |
| Stage reports produced | 5 |
| Invariants verified | 10 pipeline + 6 isolation = 16 total |
| Handoff points documented | 11 |
| Streams verified | 9 |
| Durable consumers verified | 44 |
| Regressions | 0 |
| Risks introduced | 0 new (7 inherited, 1 resolved) |

---

## 4. Next Ceremony Recommendation

### 4.1 What the Evidence Says

The wave proved that the Foundry's canonical pipeline operates correctly across
separate binaries. The system boots, events flow, correlation is preserved,
control gates propagate, failures are isolated, and the pipeline resumes after
restarts.

What the system does NOT yet have:

1. **OMS and order lifecycle** — execute has paper venue only; no real venue, no order state machine, no position tracking.
2. **Multi-strategy orchestration** — only mean_reversion_entry exercised; portfolio-level coordination absent.
3. **Production operational tooling** — no dashboards, no alerting, no runbooks beyond smoke scripts.
4. **Real venue integration** — paper adapter is the current execution boundary.

### 4.2 Recommended Next Macro-Front

Based on the evidence and the project's trajectory:

**Option A (Recommended): OMS and Execution Lifecycle Wave**
- The pipeline is proven end-to-end through paper venue. The next architectural gap is order lifecycle management: order state machine, position tracking, fill reconciliation, and the transition from paper to real venue adapters.
- This is the natural next vertical after proving the plumbing works.

**Option B: Multi-Strategy Orchestration Wave**
- Extend the pipeline to handle multiple strategy families and portfolio-level coordination.
- Depends on OMS for meaningful execution beyond paper.

**Option C: Production Hardening Wave**
- TLS, resource limits, clustering, alerting, dashboards.
- Valuable but less architecturally urgent than OMS.

### 4.3 Ceremony Format

The next ceremony should be a **Wave Charter and Scope Freeze** (same format as S370) for whichever macro-front the owner selects. The charter should:

1. Define the frozen scope and non-goals.
2. Formulate governing questions.
3. Establish the capability target.
4. Order the execution stages.

This evidence gate does NOT open the next wave. The owner decides the direction.

---

**Evaluated:** 2025-03-22
**Companion to:** `multi-binary-orchestration-evidence-gate.md`
