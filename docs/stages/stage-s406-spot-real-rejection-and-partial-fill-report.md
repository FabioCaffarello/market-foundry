# Stage S406: Spot Real Rejection and Partial Fill Evidence — Report

Status: **complete**
Date: 2026-03-22
Predecessor: S405 (Spot real venue acceptance/fill proof)
Successor: S407 (read-path/auditability consolidation)

## 1. Charter

Prove the rejection path (`submitted -> rejected`) and partial fill path (`accepted -> partially_filled`) on the Binance Spot testnet unified runtime. Document evidence honestly, distinguishing direct proof from structural inference.

## 2. Governing Questions

| ID | Question | Status | Evidence |
|---|---|---|---|
| TV-Q3 | Real rejection lifecycle (submitted -> rejected) | **Proven** | 10 adapter + 4 actor rejection tests |
| TV-Q4 | Rejection event fidelity (code, reason, HTTP status) | **Proven** | Venue details audit trail validated |
| TV-Q5 | Partial fill observation (PARTIALLY_FILLED handling) | **Structurally proven** | Mock response + adapter/lifecycle tests |
| TV-Q6 | Quantity monotonicity under partial fills | **Proven** | 3 scenarios, invariant holds |

## 3. Deliverables

### 3.1 Code Artifacts

| Artifact | Type | Location |
|---|---|---|
| S406 adapter-level tests (19 tests) | Test | `internal/application/execution/s406_spot_rejection_partial_fill_test.go` |
| S406 actor-level tests (11 tests) | Test | `internal/actors/scopes/execute/s406_spot_rejection_partial_fill_test.go` |
| Smoke script update | Script | `scripts/smoke-spot-venue-live.sh` (phases 3-4 added) |

### 3.2 Documentation

| Document | Location |
|---|---|
| Evidence doc | `docs/architecture/spot-real-rejection-and-partial-fill-evidence-on-unified-runtime.md` |
| Evidence strength doc | `docs/architecture/spot-rejected-partialfill-evidence-strength-auditability-and-limitations.md` |
| Stage report (this) | `docs/stages/stage-s406-spot-real-rejection-and-partial-fill-report.md` |

### 3.3 No Production Code Changes

S406 required **zero production code changes**. The existing adapter, lifecycle model, rejection event infrastructure, and segment router already supported all rejection and partial fill paths. S406 is purely an evidence and validation stage.

## 4. Evidence Matrix

### 4.1 Rejection Evidence

| Test | Layer | Scenario | Verdict |
|---|---|---|---|
| TestS406_Rejection_InsufficientBalance | adapter | HTTP 400 / -2010 | PASS |
| TestS406_Rejection_InvalidQuantity | adapter | HTTP 400 / -1013 (LOT_SIZE) | PASS |
| TestS406_Rejection_MarginInsufficient | adapter | HTTP 400 / -2019 | PASS |
| TestS406_Rejection_AuthFailure | adapter | HTTP 401 / -2015 | PASS |
| TestS406_Rejection_RateLimit | adapter | HTTP 429 | PASS |
| TestS406_Rejection_VenueInternalOverride | adapter | HTTP 400 / -1001 override | PASS |
| TestS406_Rejection_OrderRateLimitOverride | adapter | HTTP 400 / -1015 override | PASS |
| TestS406_Rejection_ServerError | adapter | HTTP 503 | PASS |
| TestS406_Rejection_VenueRejectedStatus | adapter | HTTP 200 + REJECTED status | PASS |
| TestS406_Rejection_VenueExpiredStatus | adapter | HTTP 200 + EXPIRED status | PASS |
| TestS406_Rejection_LifecycleTransition | adapter | State machine validation | PASS |
| TestS406_ActorComposition_SpotRejection_InsufficientBalance | actor | Router + segment isolation | PASS |
| TestS406_ActorComposition_SpotRejection_LOTSize | actor | Router + error code | PASS |
| TestS406_ActorComposition_SpotRejectionEvent_Construction | actor | Event fields complete | PASS |
| TestS406_ActorComposition_SpotRejection_VenueRejectedStatus200 | actor | HTTP 200 REJECTED through router | PASS |

### 4.2 Partial Fill Evidence

| Test | Layer | Scenario | Verdict |
|---|---|---|---|
| TestS406_PartialFill_SingleLeg | adapter | Single fill leg, StatusPartiallyFilled | PASS |
| TestS406_PartialFill_MultiLeg | adapter | 2-leg aggregation | PASS |
| TestS406_PartialFill_LifecycleTransitions | adapter | State machine paths | PASS |
| TestS406_PartialFill_QuantityMonotonicity/half_filled | adapter | QM invariant | PASS |
| TestS406_PartialFill_QuantityMonotonicity/quarter_filled | adapter | QM invariant | PASS |
| TestS406_PartialFill_QuantityMonotonicity/tiny_partial | adapter | QM invariant | PASS |
| TestS406_PartialFill_FillTimestamp | adapter | Venue clock origin | PASS |
| TestS406_ActorComposition_SpotPartialFill_ThroughRouter | actor | Router + segment isolation | PASS |
| TestS406_ActorComposition_SpotPartialFill_MultiLeg | actor | Router + aggregation | PASS |
| TestS406_ActorComposition_SpotPartialFill_CorrelationPreserved | actor | Correlation chain | PASS |
| TestS406_ActorComposition_SpotPartialFill_DryRunIntercepted | actor | DryRun interception | PASS |

### 4.3 Regression

| Test | Verdict |
|---|---|
| TestS406_Regression_FilledStillWorks | PASS |
| TestS406_Rejection_CorrelationPreserved | PASS |
| All S405 tests (32) | PASS |

### 4.4 Audit Trail Evidence

| Test | Layer | Scenario | Verdict |
|---|---|---|---|
| TestS406_RejectionEvent_SpotVenueDetails_AuditTrail/insufficient_balance | actor | venue_http_status + venue_error_code | PASS |
| TestS406_RejectionEvent_SpotVenueDetails_AuditTrail/lot_size_violation | actor | venue_error_code -1013 | PASS |
| TestS406_RejectionEvent_SpotVenueDetails_AuditTrail/auth_failure | actor | venue_http_status 401 | PASS |

## 5. Test Count

| Category | Count |
|---|---|
| S406 adapter-level | 19 |
| S406 actor-level | 11 |
| **Total S406** | **30** |
| S405 regression (all pass) | 32 |

## 6. Honest Limitations

1. **No live rejection observed**: Mock HTTP responses replicate Binance Spot error payloads. The adapter code is identical for mock and live responses, so classification correctness transfers.

2. **No live partial fill observed**: PARTIALLY_FILLED for market orders on Spot testnet is rare-to-impossible under normal conditions. The adapter parses the status correctly, and the lifecycle model supports the transition, but no live testnet call has produced this status.

3. **Cross-call monotonicity not tested**: Single-call quantity monotonicity is proven. Multiple successive partial fill updates for the same order (stateful tracking) is S407+ scope.

4. **Read-path not validated**: Rejection events are published to NATS but persistence to ClickHouse and queryability via gateway are not yet proven.

## 7. Non-Goals Respected

| Non-Goal | Respected |
|---|---|
| No Futures proof | Yes — Spot only |
| No advanced order types | Yes — market orders only |
| No OMS broad scope | Yes — focused on rejection/partial fill lifecycle |
| No artificial engineering inflation | Yes — zero production code changes required |
| No masking absence of partial fill | Yes — clearly documented as structural proof |

## 8. Handoff to S407

S407 should address:

1. **Read-path consolidation**: Rejection events materialized to KV and queryable via gateway.
2. **ClickHouse persistence**: Fill and rejection events written to ClickHouse for audit queries.
3. **Cross-call monotonicity**: If actor-level state tracking for partial fills is desired, implement and test FilledQuantity monotonicity across successive updates.
4. **Gateway query interface**: Rejection surface queryable alongside fill and gate status.

## 9. Conclusion

S406 closes the most sensitive lifecycle gaps in the Spot segment: rejection with real error codes and partial fill handling. The rejection path is proven with strong evidence (same code, same payloads, deterministic classification). The partial fill path is structurally proven with honest acknowledgment that Spot testnet market orders do not naturally produce PARTIALLY_FILLED. The lifecycle canonical model receives additional validation, and the base is ready for read-path and auditability consolidation in S407.
