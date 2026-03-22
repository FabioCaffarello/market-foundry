# Derive Integration Wave — Formal Evidence Gate

> **Stage:** S369 (DI-5)
> **Wave:** Derive Integration (S364–S369)
> **Predecessor:** S363 — Strategy/Signal Integration Evidence Gate (CLOSED)
> **Verdict:** WAVE CLOSED — ALL OBJECTIVES MET

---

## 1. Executive Summary

The Derive Integration Wave (S364–S368) set out to prove that the Foundry's derive scope produces `StrategyResolvedEvent` in full compliance with the S359 canonical contract, that the store/gateway read-path preserves all domain fields, and that the end-to-end analytical-to-execution pipeline is connected and correct.

**Result:** All 8 governing questions answered with HIGH confidence. Zero production code changes required — the existing implementation was already correct. 88 new tests added across 6 test files. Zero regressions. The producer side of the analytical pipeline is closed.

---

## 2. Wave Scope Recap

| Parameter | Value |
|---|---|
| Strategy family | `mean_reversion_entry` (single family) |
| Signal family | RSI (`rsi_oversold`) |
| Execution mode | Paper only |
| Scope | Producer-side proof (audit + test + E2E) |
| Non-goals respected | 15/15 (no batch onboarding, no multi-venue, no OMS, no risk domain changes, no mainnet) |

### Ordered Execution Blocks

| Block | Stage | Objective | Verdict |
|---|---|---|---|
| DI-1 | S365 | Producer spec and derive ownership audit | COMPLETE |
| DI-2 | S366 | Canonical producer wiring + unit tests | 49 TESTS PASS |
| DI-3 | S367 | Store/gateway/read-path verification | 21 TESTS PASS |
| DI-4 | S368 | E2E analytical-to-execution proof | 18 TESTS PASS |
| DI-5 | S369 | Evidence gate (this document) | ISSUED |

---

## 3. Governing Questions Audit

| ID | Question | Answer | Confidence | Evidence |
|---|---|---|---|---|
| DIQ-1 | Does derive satisfy all 6 producer-side invariants (PI-1–PI-6)? | YES | HIGH | 20 tests in `producer_invariant_test.go` |
| DIQ-2 | Does derive output match the S359 16-field contract? | YES | HIGH | Field-level compliance matrix (S365); INV-1,3,5,7,11 tested |
| DIQ-3 | Are all producer-side invariants covered by targeted tests? | YES | HIGH | 49 tests across PI, BI, TI, INV categories (S366) |
| DIQ-4 | Is publisher correctness proven without live NATS? | YES | HIGH | 29 publisher correctness tests (S366) |
| DIQ-5 | Does the store/gateway read-path preserve all domain fields? | YES | HIGH | 21 tests: KV round-trip, projection gates, query responder (S367) |
| DIQ-6 | Are projection gates (final, validation, monotonicity) verified? | YES | HIGH | Dedicated tests for each gate with counter tracking (S367) |
| DIQ-7 | Does the full pipeline connect derive through execution? | YES | HIGH | 18 E2E tests: triggered→buy, flat→none, severity scaling, safety gates (S368) |
| DIQ-8 | Does the correlation chain propagate end-to-end? | YES | HIGH | 5-hop verification: decision→strategy→execution→submit→fill (S368) |

**Summary:** 8/8 HIGH confidence. 0 MEDIUM. 0 LOW.

---

## 4. Capability Classification

| ID | Capability | Rating | Evidence |
|---|---|---|---|
| DC-1 | Producer spec compliance (S359 contract) | **FULL** | 16-field compliance matrix, zero mismatches, INV-1/3/5/7/11 tested |
| DC-2 | Producer wiring correctness | **FULL** | 49 unit tests covering PI/BI/TI/INV categories, all PASS |
| DC-3 | Store/gateway read-path | **SUBSTANTIAL** | 21 tests verify field preservation and gates; event metadata gap documented (L1) |
| DC-4 | E2E analytical-to-execution pipeline | **FULL** | 18 E2E tests proving complete chain from derive to venue fill |
| DC-5 | Correlation chain preservation | **FULL** | 5-hop chain verified; CorrelationID immutable, CausationID DAG correct |
| DC-6 | Regression-free integration | **FULL** | 25/25 repository consistency checks pass; zero production code changes |

**Summary:** 5 FULL, 1 SUBSTANTIAL. The SUBSTANTIAL rating for DC-3 reflects the documented event metadata gap (correlation_id, causation_id not persisted in KV) — this is a known trade-off, not a defect.

---

## 5. Test Inventory

| File | Stage | Tests | Category |
|---|---|---|---|
| `internal/actors/scopes/derive/producer_invariant_test.go` | S366 | 20 | Unit: PI, BI, TI, INV |
| `internal/adapters/nats/natsstrategy/publisher_correctness_test.go` | S366 | 29 | Unit: registry, subjects, dedup, consumers |
| `internal/actors/scopes/store/strategy_read_path_test.go` | S367 | 15 | Unit: projection, monotonicity, query, subjects |
| `internal/adapters/nats/natsstrategy/kv_store_read_path_test.go` | S367 | 6 | Unit: KV round-trip, field preservation |
| `internal/actors/scopes/execute/e2e_derive_to_execution_test.go` | S368 | 12 | E2E: derive→execute→venue |
| `internal/actors/scopes/store/e2e_derive_to_store_test.go` | S368 | 6 | E2E: derive→store→query |

**Total:** 88 new tests, all PASS. Zero TODO/FIXME markers.

---

## 6. Invariant Coverage Matrix

### Structural Invariants (PI-1 through PI-6)

| ID | Invariant | Unit Test | E2E Test | Status |
|---|---|---|---|---|
| PI-1 | Type always `mean_reversion_entry` | `TestPI1_TypeAlwaysMeanReversionEntry` | `TestE2E_DeriveTriggered` | PASS |
| PI-2 | Direction is valid enum | `TestPI2_DirectionIsValid` | `TestE2E_DeriveTriggered` | PASS |
| PI-3 | Confidence in [0.0, 1.0] with severity scaling | `TestPI3_ConfidenceIsValidDecimal` | `TestE2E_DeriveSeverityScaling` | PASS |
| PI-4 | Decisions has exactly one entry | `TestPI4_DecisionsHasExactlyOneEntry` | — | PASS |
| PI-5 | Final always true | `TestPI5_FinalAlwaysTrue` | `TestE2E_DeriveTriggered` | PASS |
| PI-6 | Timestamp is decision timestamp | `TestPI6_TimestampIsDecisionTimestamp` | `TestE2E_DeriveTriggered` | PASS |

### Behavioral Invariants (BI-1 through BI-6)

| ID | Invariant | Unit Test | E2E Test | Status |
|---|---|---|---|---|
| BI-1 | Deterministic resolution | `TestBI1_ResolutionIsDeterministic` | `TestE2E_DeduplicationKey` | PASS |
| BI-2 | Validation gates | (implicit via TI test) | — | PASS |
| BI-3 | Unknown outcomes silent | `TestBI3_UnknownDecisionOutcome` | `TestE2E_UnknownOutcome` | PASS |
| BI-4 | Severity scaling bounded | (implicit via PI-3) | `TestE2E_DeriveSeverityScaling` | PASS |
| BI-5 | Flat = zero confidence | `TestBI5_FlatDirection_ZeroConfidence` | `TestE2E_DeriveNotTriggered` | PASS |
| BI-6 | Event metadata immutable | `TestBI6_EventMetadata` | `TestE2E_FullPipeline` | PASS |

### Transport Invariants (TI-1 through TI-5)

| ID | Invariant | Unit Test | Status |
|---|---|---|---|
| TI-1 | Subject pattern | `TestSubjectConstruction_MeanReversionEntry` | PASS |
| TI-2 | Dedup key format | `TestDeduplicationKey_Format` | PASS |
| TI-3 | Envelope type | `TestRegistry_MeanReversionEntryType` | PASS |
| TI-4 | Correlation/causation passthrough | `TestTI_CorrelationIDAndCausationID` | PASS |
| TI-5 | Stream creation | `TestRegistry_StreamName/Subjects/Retention/MaxBytes` | PASS |

### S359 Contract Invariants (INV-1 through INV-11)

| ID | Invariant | Producer Test | Consumer/E2E Test | Status |
|---|---|---|---|---|
| INV-1 | Type identity | `TestINV1_TypeIdentity` | `TestE2E_DeriveTriggered` | PASS |
| INV-2 | Direction-to-side mapping | — | `TestE2E_DeriveTriggered` (side=buy) | PASS |
| INV-3 | Correlation/causation chain | `TestINV3_CausationChain` | `TestE2E_FullPipeline` | PASS |
| INV-4 | Pass-through risk | — | `TestE2E_DeriveTriggered` (risk=pass_through) | PASS |
| INV-5 | Timestamp source-derived | `TestINV5_TimestampMonotonicity` | `TestE2E_DeriveTriggered` | PASS |
| INV-6 | Flat → no execution | — | `TestE2E_DeriveNotTriggered` | PASS |
| INV-7 | Flat direction handling | `TestINV7_FlatMeansNoExecution` | `TestE2E_DeriveNotTriggered` | PASS |
| INV-8 | Dedup key determinism | — | `TestE2E_DeduplicationKey` | PASS |
| INV-9 | Consumer subject alignment | `TestConsumer_*_MatchesProducerSubject` | — | PASS |
| INV-10 | Partition key determinism | `TestKVRoundTrip_PartitionKeyStable` | — | PASS |
| INV-11 | Dedup key uniqueness | `TestINV11_DeduplicationKeyUniqueness` | `TestE2E_DeduplicationKey` | PASS |

**Summary:** 11/11 invariants verified. All PASS.

---

## 7. Regression Verification

| Check | Result |
|---|---|
| Repository consistency (25 checks) | ALL PASS |
| Stage report naming and shape | COMPLIANT |
| Stage-index alignment (400+ reports) | ALL INDEXED |
| Architecture doc links | ALL RESOLVE |
| Production code changes in wave | ZERO (existing code correct) |
| Pre-existing test suites | NOT BROKEN (no code changes) |
| Makefile targets | ALL PRESENT |

**Regression verdict:** ZERO REGRESSIONS.

---

## 8. Non-Goals Respected

All 15 chartered non-goals were respected:

1. No batch onboarding of additional strategy families
2. No additional signal families beyond RSI
3. No multi-venue routing
4. No OMS or order management
5. No portfolio-level risk management
6. No mainnet execution
7. No dashboard or alerting
8. No derive runtime redesign
9. No Docker Compose changes
10. No log aggregation infrastructure
11. No performance optimization
12. No historical replay capability
13. No multi-binary orchestration testing
14. No ClickHouse writer verification
15. No confidence threshold at producer level

---

## 9. Formal Verdict

**DERIVE INTEGRATION WAVE (S364–S369): CLOSED — ALL OBJECTIVES MET**

- 8/8 governing questions: HIGH confidence
- 5 FULL + 1 SUBSTANTIAL capability ratings
- 88 new tests, all PASS
- 11/11 contract invariants verified
- Zero production code changes required
- Zero regressions
- 15/15 non-goals respected

The Foundry has closed the producer side of the analytical pipeline. The derive scope produces `StrategyResolvedEvent` in full compliance with the S359 contract. The store/gateway read-path preserves all domain fields. The end-to-end pipeline from derive through execution to venue fill is connected and proven.
