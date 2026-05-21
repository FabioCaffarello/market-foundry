# Stage S366: Canonical Derive Producer Wiring Report

> Phase 37 — Derive Integration Wave, Block DI-2
> Predecessor: [S365](stage-s365-strategy-resolved-event-producer-spec-report.md)
> Charter: [S364](stage-s364-derive-integration-charter-report.md)

## Executive Summary

S366 validated the canonical producer wiring of `StrategyResolvedEvent` in the derive binary through targeted unit tests and publisher correctness tests. The S365 compliance audit found zero blocking mismatches, so S366 required no code changes to the production path — only test additions and documentation. All governing questions for DI-2 are answered with HIGH confidence.

## Governing Questions

| Question | Answer | Confidence |
|---|---|---|
| DIQ-3: Do unit tests prove each invariant on producer side? | YES — 20 tests covering PI-1 through PI-6, BI-1/3/5/6, TI-2/4, INV-1/3/5/7/11 | HIGH |
| DIQ-4: Does publisher produce correct NATS messages? | YES — 29 tests covering registry contracts, subject construction, dedup keys, consumer alignment, error paths, event construction | HIGH |

## Deliverables

### Tests Added

| File | Tests | Coverage |
|---|---|---|
| `internal/actors/scopes/derive/producer_invariant_test.go` | 20 | Structural (PI-1..6), behavioral (BI-1,3,5,6), transport (TI-2,4), S359 contract (INV-1,3,5,7,11) |
| `internal/adapters/nats/natsstrategy/publisher_correctness_test.go` | 29 | Registry contracts, specForType routing, subject construction, dedup key format/uniqueness, error paths, consumer-producer alignment, event interface, LatestSpecByType |

**Total new tests: 49, all PASS.**

### Documentation Added

| Document | Purpose |
|---|---|
| [`docs/architecture/canonical-derive-producer-wiring.md`](../architecture/canonical-derive-producer-wiring.md) | Canonical wiring path, integration points, contracts preserved |
| [`docs/architecture/derive-to-strategy-event-wiring-order-controls-and-limitations.md`](../architecture/derive-to-strategy-event-wiring-order-controls-and-limitations.md) | Ordering guarantees, control mechanisms, 11 explicit limitations |
| This report | Stage evidence and closure |

### No Code Changes Required

S365 found zero blocking mismatches. The derive binary's `MeanReversionEntryResolverActor` already:

- Produces events with correct type, direction, confidence, decisions, final flag, and timestamp
- Propagates CorrelationID and CausationID correctly
- Validates strategy before publication
- Constructs deterministic dedup keys
- Publishes to correct NATS subjects matching all consumer filters

## Invariant Coverage Matrix

### Structural Invariants (PI)

| Invariant | Description | Test | Status |
|---|---|---|---|
| PI-1 | Type always `mean_reversion_entry` | `TestPI1_TypeAlwaysMeanReversionEntry` | PASS |
| PI-2 | Direction in {long, short, flat} | `TestPI2_DirectionIsValid` | PASS |
| PI-3 | Confidence valid decimal [0.0, 1.0] | `TestPI3_ConfidenceIsValidDecimal` | PASS |
| PI-4 | Decisions has exactly one entry | `TestPI4_DecisionsHasExactlyOneEntry` | PASS |
| PI-5 | Final always true | `TestPI5_FinalAlwaysTrue` | PASS |
| PI-6 | Timestamp is decision timestamp | `TestPI6_TimestampIsDecisionTimestamp` | PASS |

### Behavioral Invariants (BI)

| Invariant | Description | Test | Status |
|---|---|---|---|
| BI-1 | Deterministic resolution | `TestBI1_ResolutionIsDeterministic` | PASS |
| BI-3 | Unknown outcome never produces event | `TestBI3_UnknownDecisionOutcome_NeverProducesEvent` | PASS |
| BI-5 | Flat direction has zero confidence | `TestBI5_FlatDirection_ZeroConfidence` | PASS |
| BI-6 | Metadata constructed once, immutable | `TestBI6_EventMetadata_ConstructedOnceImmutable` | PASS |

### Transport-Readiness Invariants (TI)

| Invariant | Description | Test | Status |
|---|---|---|---|
| TI-2 | Dedup key deterministic from strategy | `TestTI_DeduplicationKey_DeterministicFromStrategy` | PASS |
| TI-4 | CorrelationID/CausationID in event | `TestTI_CorrelationIDAndCausationID_PassedToEvent` | PASS |

### S359 Contract Invariants (INV)

| Invariant | Description | Test | Status |
|---|---|---|---|
| INV-1 | Type identity | `TestINV1_TypeIdentity` | PASS |
| INV-3 | Causation chain | `TestINV3_CausationChain` | PASS |
| INV-5 | Timestamp monotonicity | `TestINV5_TimestampMonotonicity` | PASS |
| INV-7 | Flat = no execution | `TestINV7_FlatMeansNoExecution` | PASS |
| INV-11 | Dedup key uniqueness | `TestINV11_DeduplicationKeyUniqueness` | PASS |

### Publisher Correctness

| Category | Tests | Status |
|---|---|---|
| Registry contract (subjects, types, stream config) | 7 | ALL PASS |
| specForType routing (3 types + unknown) | 4 | ALL PASS |
| Subject construction and consumer alignment | 2 | ALL PASS |
| Dedup key format and uniqueness | 3 | ALL PASS |
| Error paths (nil publisher, nil JS, unknown type) | 3 | ALL PASS |
| Consumer spec alignment (writer, store, execute) | 5 | ALL PASS |
| Event interface implementation | 1 | ALL PASS |
| LatestSpecByType routing | 2 | ALL PASS |
| Consumer AckWait and MaxDeliver | 2 | ALL PASS |

## Regression Check

All existing derive tests continue to pass:

```
ok  internal/actors/scopes/derive  10.136s
ok  internal/adapters/nats/natsstrategy  0.200s
```

## Files Changed

| File | Change |
|---|---|
| `internal/actors/scopes/derive/producer_invariant_test.go` | NEW — 20 producer invariant tests |
| `internal/adapters/nats/natsstrategy/publisher_correctness_test.go` | NEW — 29 publisher correctness tests |
| `docs/architecture/canonical-derive-producer-wiring.md` | NEW — wiring documentation |
| `docs/architecture/derive-to-strategy-event-wiring-order-controls-and-limitations.md` | NEW — order/controls/limits |
| `docs/stages/stage-s366-canonical-derive-producer-wiring-report.md` | NEW — this report |
| `docs/architecture/README.md` | UPDATED — S366 links |
| `docs/stages/INDEX.md` | UPDATED — S366 entry |

## Guard Rails Compliance

| Guard rail | Status |
|---|---|
| No multiple parallel producers opened | COMPLIANT — single StrategyPublisherActor per SourceScopeActor |
| No broad derive pipeline redesign | COMPLIANT — zero production code changes |
| No multi-binary orchestration inflation | COMPLIANT — tests are single-binary unit tests |
| No consolidated contracts broken | COMPLIANT — all existing tests pass |

## Residual Limitations

1. Publisher correctness tests validate logic paths without a live NATS server. Integration-level publishing (NATS round-trip) is deferred to S367.
2. BI-2 (validation failure never produces event) is covered implicitly by the resolver rejecting invalid confidence, but not by a dedicated S366 test (already covered in existing `TestMeanReversionResolverActor_InvalidConfidence_NoPublish`).
3. BI-4 (severity scaling bounded) is covered by existing `severity_scaling_test.go` in the application layer.
4. TI-1 (subject matches registry), TI-3 (envelope type from registry), TI-5 (stream created before first publish) are structural properties verified by the publisher correctness tests at the registry level.

## Preparation for S367

S367 (DI-3: Store/gateway/read-path verification) should:

1. Write an integration test that publishes a `StrategyResolvedEvent` via the real NATS adapter and consumes it with the store projection actor.
2. Verify that the store materializes the event to a KV bucket with correct partition key.
3. Verify that the gateway HTTP endpoint returns the materialized strategy state.
4. Prove that the monotonicity guard rejects stale strategy timestamps.
5. Validate the full read-path: derive → NATS → store → KV → gateway → HTTP response.

Prerequisites for S367:
- S366 tests all PASS (confirmed)
- Store projection actor exists and is wired for strategy events (confirmed in S360)
- Gateway strategy endpoint exists (confirmed in S361)
- NATS test infrastructure is proven (confirmed by `natsexecution` integration tests)
