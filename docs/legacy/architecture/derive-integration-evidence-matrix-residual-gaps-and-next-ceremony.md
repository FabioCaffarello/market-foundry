# Derive Integration Wave — Evidence Matrix, Residual Gaps, and Next Ceremony

> **Stage:** S369 (DI-5)
> **Wave:** Derive Integration (S364–S369)
> **Status:** CLOSED

---

## 1. Objective Evidence Matrix

### Stage-by-Stage Delivery

| Stage | Block | Deliverable | Tests Added | Code Changes | Verdict |
|---|---|---|---|---|---|
| S364 | Charter | Wave charter + scope freeze + capabilities doc | 0 | 0 | CHARTERED |
| S365 | DI-1 | Producer spec + ownership model + field compliance matrix | 0 | 0 | AUDIT COMPLETE |
| S366 | DI-2 | Producer invariant tests + publisher correctness tests | 49 | 0 | ALL PASS |
| S367 | DI-3 | KV round-trip tests + read-path tests + store verification | 21 | 0 | ALL PASS |
| S368 | DI-4 | E2E derive-to-execution + derive-to-store tests | 18 | 0 | ALL PASS |
| S369 | DI-5 | Evidence gate (this document) | 0 | 0 | ISSUED |

**Wave totals:** 88 tests added, 0 production code changes, 10 architecture documents, 6 stage reports.

### Governing Question Resolution

| ID | Question | Stage | Confidence | Method |
|---|---|---|---|---|
| DIQ-1 | Producer invariant compliance | S365+S366 | HIGH | Code audit + 20 unit tests |
| DIQ-2 | S359 16-field contract match | S365 | HIGH | Field-level compliance matrix |
| DIQ-3 | Producer invariant test coverage | S366 | HIGH | 49 tests across PI/BI/TI/INV |
| DIQ-4 | Publisher correctness without live NATS | S366 | HIGH | 29 structural tests |
| DIQ-5 | Read-path field preservation | S367 | HIGH | 21 tests: KV, projection, query |
| DIQ-6 | Projection gate verification | S367 | HIGH | Dedicated gate tests with counters |
| DIQ-7 | Full pipeline connectivity | S368 | HIGH | 18 E2E tests |
| DIQ-8 | Correlation chain E2E | S368 | HIGH | 5-hop chain verification |

### Capability Ratings

| ID | Capability | Rating | Justification |
|---|---|---|---|
| DC-1 | Producer spec compliance | FULL | 16 fields verified, zero mismatches, INV-1/3/5/7/11 tested in unit + E2E |
| DC-2 | Producer wiring correctness | FULL | 49 unit tests cover all structural, behavioral, transport invariants |
| DC-3 | Store/gateway read-path | SUBSTANTIAL | 21 tests prove field preservation and projection gates; event metadata gap (L1) is documented trade-off |
| DC-4 | E2E analytical-to-execution | FULL | 18 E2E tests prove complete chain: derive→execute→venue fill + derive→store→query |
| DC-5 | Correlation chain preservation | FULL | CorrelationID immutable across 5 hops; CausationID DAG linkage verified |
| DC-6 | Regression-free integration | FULL | 25/25 consistency checks pass; zero production code changes; no test breakage |

### Pipeline Segments Verified

| Segment | Producer | Consumer | Evidence |
|---|---|---|---|
| Decision → Strategy | DecisionEvaluatorActor | MeanReversionEntryResolverActor | PI-1–6, BI-1–6 unit tests |
| Strategy → NATS | StrategyPublisherActor | JetStream STRATEGY_EVENTS | Publisher correctness tests (29) |
| NATS → Store | JetStream | StrategyProjectionActor | Read-path tests (15) + KV tests (6) |
| Store → Query | KV bucket | QueryResponder | Query use case tests |
| Query → Gateway | QueryResponder | HTTP `/strategy/:type/latest` | Subject alignment tests |
| NATS → Execute | JetStream | StrategyConsumerActor | E2E tests (12) |
| Execute → Venue | PaperOrderEvaluator | VenueAdapterActor | Full pipeline test |

---

## 2. Residual Gaps

### Wave-Scoped Gaps (discovered during wave execution)

| ID | Gap | Severity | Stage Found | Mitigation | Impact |
|---|---|---|---|---|---|
| DG-W1 | Event metadata (correlation_id, causation_id) not persisted in KV | LOW | S367 | ClickHouse analytical path + NATS replay + structured logs carry full provenance | HTTP surface cannot show event trace; operational concern only |
| DG-W2 | BI-2 and BI-4 covered implicitly, not by dedicated tests | LOW | S366 | BI-2 implicit in TI validation test; BI-4 implicit in PI-3 severity scaling | Full coverage exists; naming gap only |

### Explicitly Deferred Gaps (per charter non-goals)

| ID | Gap | Priority | Rationale |
|---|---|---|---|
| DG-D1 | `squeeze_breakout_entry` and `trend_following_entry` not E2E tested | LOW | Pattern proven mechanical; registry already has specs for all 3 families |
| DG-D2 | ClickHouse writer path not verified for strategy events | LOW | Separate scope (analytical storage); operational path works |
| DG-D3 | Multi-binary orchestration not tested | MEDIUM | Pipeline proven in-process; separate operational proof via smoke scripts |
| DG-D4 | No backpressure from execute to derive | LOW | NATS buffering prevents data loss; acceptable for current throughput |
| DG-D5 | No rate limiting at producer or KV reads | LOW | Single-family, single-signal constraint limits volume |
| DG-D6 | No push-based cache invalidation on HTTP | LOW | KV reflects latest at query time; acceptable for analytical use |
| DG-D7 | No cross-partition ordering guarantee | LOW | Within-partition FIFO sufficient; multi-symbol independent by design |
| DG-D8 | No at-most-once delivery | LOW | JetStream at-least-once with dedup key idempotency |
| DG-D9 | No short direction for RSI signal | LOW | RSI oversold produces long only; short requires different signal |
| DG-D10 | Gateway SourcePathConfigProvider not wired in compose.go | LOW | Inherited from S363 (WG-1); explain endpoint incomplete |
| DG-D11 | Confidence threshold E2E not exercised at wave level | LOW | Inherited from S363 (WG-2); unit-tested in execute scope |
| DG-D12 | Risk domain is pass-through only | BY DESIGN | Intentional for current wave; risk assessment is separate macro-front |

### Gap Summary

| Category | Count | Blocking? |
|---|---|---|
| Wave-scoped | 2 | NO (both LOW) |
| Deferred | 12 | NO (all per charter) |
| Inherited from S363 | 2 | NO (both LOW, unchanged) |

**No blocking gaps.** All gaps are either LOW severity or explicitly deferred by charter.

---

## 3. Regression Audit

### Verification Performed

| Check | Scope | Result |
|---|---|---|
| Repository consistency | 25 invariant checks | ALL PASS |
| Stage report naming | 400+ reports | COMPLIANT |
| Architecture doc links | Phase 37 section | ALL RESOLVE |
| Production code integrity | derive/store/execute/adapter | ZERO CHANGES (no regressions possible) |
| Test suite integrity | Pre-existing tests | NO BREAKAGE (no code changes) |
| Index alignment | stages/INDEX.md, architecture/README.md | CURRENT |
| Makefile targets | test, check, verify | ALL PRESENT |

### Binary Build Status

Since zero production code changes were made during S364–S368, all binaries remain in their pre-wave state. The wave added only:
- 6 test files (not compiled into binaries)
- 10 architecture documents
- 6 stage reports

**Regression verdict:** ZERO REGRESSIONS. No production code was modified.

---

## 4. Cumulative Wave Progress

| Wave | Stages | Status | Key Achievement |
|---|---|---|---|
| Venue Activation | S341–S345 | CLOSED | Paper venue adapter, safety gates |
| Production Readiness | S346–S351 | CLOSED | Binary builds, Docker Compose, health checks |
| Operational Foundation | S352–S357 | CLOSED | Consistency checks, quality gates, observability |
| Strategy/Signal Integration | S358–S363 | CLOSED | Consumer-side wiring, contract invariants, explainability |
| **Derive Integration** | **S364–S369** | **CLOSED** | **Producer-side proof, E2E pipeline, correlation chain** |

After 5 closed waves (29 stages), the Foundry has a proven end-to-end pipeline from signal ingestion through analytical processing to paper execution, with both producer and consumer sides verified.

---

## 5. Next Ceremony Recommendation

### Assessment Criteria

The next macro-front must be selected based on:
1. What gaps remain after 5 closed waves
2. What delivers the highest value for the least complexity
3. What the codebase is ready for

### Candidate Macro-Fronts

| ID | Front | Value | Complexity | Prerequisites Met? | Notes |
|---|---|---|---|---|---|
| NW-1 | Strategy Family Expansion | MEDIUM | LOW | YES | Pattern proven mechanical; registry ready for all 3 families |
| NW-2 | Risk Domain Integration | HIGH | HIGH | PARTIAL | Pass-through risk sufficient; real risk requires position/portfolio modeling |
| NW-3 | Multi-Binary Orchestration | HIGH | MEDIUM | YES | In-process proven; Docker Compose exists; smoke scripts available |
| NW-4 | Observability & Alerting | MEDIUM | MEDIUM | YES | Structured logs exist; metrics exist; dashboards/alerting not wired |
| NW-5 | ClickHouse Analytical Path | MEDIUM | MEDIUM | YES | Schema exists; writer actors exist; verification not done for strategy events |
| NW-6 | Mainnet Preparation | HIGH | VERY HIGH | NO | Requires risk domain, OMS, multi-venue — premature |

### Recommendation

**Primary:** NW-3 — Multi-Binary Orchestration Proof

**Rationale:**
- The biggest unverified assumption is that the in-process proven pipeline works identically when split across separate binaries (`cmd/derive`, `cmd/store`, `cmd/execute`, `cmd/gateway`) communicating through real NATS.
- Docker Compose and smoke scripts already exist but have not been formally verified as a wave.
- This closes the gap between "unit/E2E tests prove correctness" and "the system actually runs as deployed."
- Complexity is bounded: the actors and wiring are proven; only the operational envelope needs verification.

**Alternative:** NW-1 — Strategy Family Expansion (if the team prefers breadth over depth). This would be a short wave (2–3 stages) to activate `trend_following_entry` and `squeeze_breakout_entry` using the proven mechanical pattern.

**Not recommended yet:**
- NW-2 (Risk Domain): pass-through risk is sufficient until position/portfolio modeling is designed.
- NW-6 (Mainnet): premature without risk domain and OMS.

### Next Steps

1. Charter the selected macro-front with frozen scope and governing questions.
2. Do NOT expand the derive integration scope further — the wave is closed.
3. Carry forward DG-W1 (event metadata gap) and DG-D3 (multi-binary) as tracked items for the appropriate future wave.
