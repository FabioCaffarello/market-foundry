# Execution Operational Validation Matrix

> Stage: S79 | Date: 2026-03-18 | Scope: paper_order execution family

This matrix documents every validated operational behavior of the `execution` domain,
the test that proves it, and any known gaps.

---

## 1. Domain Model Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| ExecutionIntent required field validation (type, source, symbol, timeframe, side, status, quantity, risk, timestamp) | `domain/execution/execution_test.go` — 14 tests | PROVEN |
| Side enum validation (buy, sell, none) | `domain/execution/execution_test.go::TestExecutionIntent_Validate_AllSides` | PROVEN |
| Status enum validation (6 values) | `domain/execution/execution_test.go::TestExecutionIntent_Validate_AllStatuses` | PROVEN |
| Invalid side/status rejection | `domain/execution/execution_test.go` — 3 tests | PROVEN |
| Lifecycle transitions (7 valid, 5 invalid, terminal enforcement) | `domain/execution/execution_test.go` — 10 tests | PROVEN |
| Terminal state immutability (filled, rejected, cancelled) | `domain/execution/execution_test.go::TestValidTransition_TerminalStatesCannotTransition` | PROVEN |
| FillRecord in filled intent passes validation | `domain/execution/execution_test.go::TestFillRecord_FilledIntentValidation` | PROVEN |
| PartitionKey format: `{source}.{symbol}.{timeframe}` | `domain/execution/execution_test.go::TestExecutionIntent_PartitionKey` | PROVEN |
| DeduplicationKey format: `exec:{type}:{source}:{symbol}:{timeframe}:{unix}` | `domain/execution/execution_test.go::TestExecutionIntent_DeduplicationKey` | PROVEN |
| Multi-symbol partition key isolation (3 sym x 2 tf = 6 unique) | `domain/execution/execution_test.go::TestExecutionIntent_MultiSymbol_PartitionKeyIsolation` | PROVEN |
| Multi-symbol dedup key isolation | `domain/execution/execution_test.go::TestExecutionIntent_MultiSymbol_DeduplicationKeyIsolation` | PROVEN |
| Cross-timeframe no collision (same symbol, different timeframes) | `domain/execution/execution_test.go::TestExecutionIntent_MultiSymbol_CrossTimeframe_NoCollision` | PROVEN |
| No ownership bleed between symbols | `domain/execution/execution_test.go::TestExecutionIntent_MultiSymbol_NoOwnershipBleed` | PROVEN |

## 2. Control Gate Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| GateActive valid | `domain/execution/control_test.go::TestValidGateStatus_Active` | PROVEN |
| GateHalted valid | `domain/execution/control_test.go::TestValidGateStatus_Halted` | PROVEN |
| Unknown gate status invalid | `domain/execution/control_test.go::TestValidGateStatus_Unknown` | PROVEN |
| Empty gate status invalid | `domain/execution/control_test.go::TestValidGateStatus_Empty` | PROVEN |
| IsHalted=true when halted | `domain/execution/control_test.go::TestControlGate_IsHalted_WhenHalted` | PROVEN |
| IsHalted=false when active | `domain/execution/control_test.go::TestControlGate_IsHalted_WhenActive` | PROVEN |
| Zero-value gate fail-open (not halted) | `domain/execution/control_test.go::TestControlGate_IsHalted_ZeroValue` | PROVEN |
| DefaultControlGate is active | `domain/execution/control_test.go::TestDefaultControlGate_IsActive` | PROVEN |
| DefaultControlGate has timestamp | `domain/execution/control_test.go::TestDefaultControlGate_HasTimestamp` | PROVEN |
| Halt → resume cycle | `domain/execution/control_test.go::TestControlGate_HaltAndResume` | PROVEN |
| Halt preserves audit fields | `domain/execution/control_test.go::TestControlGate_HaltPreservesAuditFields` | PROVEN |
| Control gate HTTP GET/PUT (smoke) | `scripts/smoke-multi-symbol.sh` — Step 15 | REPEATABLE |
| Control gate halt→verify→resume→verify (smoke) | `scripts/smoke-multi-symbol.sh` — Step 15a-15e | REPEATABLE |

## 3. Evaluator (Derive Path) Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Approved long → SideBuy | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_ApprovedLong_ProducesBuy` | PROVEN |
| Approved short → SideSell | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_ApprovedShort_ProducesSell` | PROVEN |
| Rejected risk → SideNone | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_Rejected_ProducesNone` | PROVEN |
| Flat strategy → SideNone | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_FlatStrategy_ProducesNone` | PROVEN |
| Modified disposition → works like approved | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_MultiSymbol_ModifiedDisposition` | PROVEN |
| Intent is Final=true and StatusSubmitted | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_IntentIsFinalAndValid` | PROVEN |
| Multi-symbol independent evaluation (3 sym x 2 tf) | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_MultiSymbol_IndependentEvaluation` | PROVEN |
| Multi-symbol different dispositions produce correct sides | `application/execution/paper_order_evaluator_test.go::TestPaperOrderEvaluator_MultiSymbol_DifferentDispositions` | PROVEN |

## 4. Fill Simulator Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Buy order → StatusFilled with simulated fill | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_BuyOrder_ProducesFilled` | PROVEN |
| Sell order → StatusFilled | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_SellOrder_ProducesFilled` | PROVEN |
| No-action (SideNone) → StatusSubmitted, no fills | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_NoAction_StaysSubmitted` | PROVEN |
| Non-submitted status → returns false | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_NonSubmittedStatus_ReturnsFalse` | PROVEN |
| Original fields preserved through simulation | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_PreservesOriginalFields` | PROVEN |
| Filled intent passes domain validation | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_FilledIntentPassesValidation` | PROVEN |
| Multi-symbol independent fills | `application/execution/paper_fill_simulator_test.go::TestPaperFillSimulator_MultiSymbol_IndependentFills` | PROVEN |

## 5. End-to-End Pipeline (Evaluate → Simulate → Emit) Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Full pipeline: risk primitives → evaluated intent → filled intent → valid event | `application/execution/pipeline_integration_test.go::TestPipeline_EvaluateSimulateEmit_BuyOrder` | PROVEN |
| Pipeline: rejected risk → no-action intent → no fills → valid event | `application/execution/pipeline_integration_test.go::TestPipeline_EvaluateSimulateEmit_RejectedRisk_NoFill` | PROVEN |
| Pipeline multi-symbol: 3 symbols × 2 timeframes, full isolation | `application/execution/pipeline_integration_test.go::TestPipeline_MultiSymbol_FullIsolation` | PROVEN |
| Trace fields (correlation_id, causation_id) survive pipeline | `pipeline_integration_test.go::TestPipeline_EvaluateSimulateEmit_BuyOrder` | PROVEN |
| Event metadata (ID, correlation, causation) correctly populated | `pipeline_integration_test.go::TestPipeline_EvaluateSimulateEmit_BuyOrder` | PROVEN |
| Partition key format verified end-to-end | `pipeline_integration_test.go::TestPipeline_EvaluateSimulateEmit_BuyOrder` | PROVEN |

## 6. Projection (Store Path) Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Final gate: non-final intents skipped | `store/execution_projection_actor_test.go::TestExecutionProjection_FinalGate_SkipsNonFinal` | PROVEN |
| Validation gate: malformed intents rejected | `store/execution_projection_actor_test.go::TestExecutionProjection_ValidationGate_RejectsMalformed` | PROVEN |
| Validation gate: invalid side rejected | `store/execution_projection_actor_test.go::TestExecutionProjection_ValidationGate_RejectsInvalidSide` | PROVEN |
| PutWritten → materialized counter incremented | `store/execution_projection_actor_test.go::TestExecutionProjection_PutWritten_Materializes` | PROVEN |
| PutSkippedStale → skippedStale counter | `store/execution_projection_actor_test.go::TestExecutionProjection_PutSkippedStale` | PROVEN |
| PutSkippedDuplicate → skippedDedup counter | `store/execution_projection_actor_test.go::TestExecutionProjection_PutSkippedDuplicate` | PROVEN |
| Put error → errors counter, tracker.RecordError | `store/execution_projection_actor_test.go::TestExecutionProjection_PutError` | PROVEN |
| Put error → tracker does NOT record success | `store/execution_projection_actor_test.go::TestExecutionProjection_PutError_TrackerRecordsError` | PROVEN |
| No tracker → no panic | `store/execution_projection_actor_test.go::TestExecutionProjection_NoTracker_DoesNotPanic` | PROVEN |
| All side values pass validation gate | `store/execution_projection_actor_test.go::TestExecutionProjection_AllSideValues_PassValidation` | PROVEN |
| Multiple events → stats accumulate | `store/execution_projection_actor_test.go::TestExecutionProjection_MultipleEvents_StatsAccumulate` | PROVEN |
| Stats invariant: received = sum of outcomes | `store/execution_projection_actor_test.go::TestExecutionProjection_StatsInvariant_ReceivedEqualsSum` | PROVEN |
| Multi-symbol independent materialization (2 sym × 2 tf) | `store/execution_projection_actor_test.go::TestExecutionProjection_MultiSymbol_IndependentMaterialization` | PROVEN |
| Multi-symbol partition key isolation (3 sym × 2 tf) | `store/execution_projection_actor_test.go::TestExecutionProjection_MultiSymbol_NoBleed_PartitionKeys` | PROVEN |
| Multi-symbol dedup key isolation | `store/execution_projection_actor_test.go::TestExecutionProjection_MultiSymbol_DeduplicationKeys` | PROVEN |
| Multi-symbol mixed outcomes (materialized + skipped) | `store/execution_projection_actor_test.go::TestExecutionProjection_MultiSymbol_MixedOutcomes` | PROVEN |

## 7. Trace Persistence Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Trace fields survive full gate pipeline to materialization | `store/execution_projection_actor_test.go::TestExecutionProjection_TracePersistence_FieldsSurviveMaterialization` | PROVEN |
| Empty trace still materializes (trace is optional) | `store/execution_projection_actor_test.go::TestExecutionProjection_TracePersistence_EmptyTraceStillMaterializes` | PROVEN |
| Multi-symbol independent traces (no bleed) | `store/execution_projection_actor_test.go::TestExecutionProjection_TracePersistence_MultiSymbol_IndependentTraces` | PROVEN |
| Trace fields in pipeline event construction | `application/execution/pipeline_integration_test.go::TestPipeline_EvaluateSimulateEmit_BuyOrder` | PROVEN |
| Trace fields in smoke HTTP response | `scripts/smoke-multi-symbol.sh` — Step 13 | REPEATABLE |

## 8. Lifecycle Fields Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Filled intent with fill records passes projection gates | `store/execution_projection_actor_test.go::TestExecutionProjection_LifecycleFields_FilledIntentMaterializes` | PROVEN |
| Submitted no-action intent (side=none) materializes | `store/execution_projection_actor_test.go::TestExecutionProjection_LifecycleFields_SubmittedNoActionMaterializes` | PROVEN |
| Smoke validates actionable orders are filled with fill records | `scripts/smoke-multi-symbol.sh` — Step 13 (lifecycle assertions) | REPEATABLE |
| Smoke validates no-action orders remain submitted | `scripts/smoke-multi-symbol.sh` — Step 13 (lifecycle assertions) | REPEATABLE |

## 9. Query Surface Validation

| Behavior | Test Location | Status |
|----------|--------------|--------|
| GET /execution/:type/latest returns 200 with execution_intent | `scripts/smoke-multi-symbol.sh` — Step 13 | REPEATABLE |
| Missing timeframe → 400 | `scripts/smoke-multi-symbol.sh` — Step 16 | REPEATABLE |
| Unknown execution type → 400 | `scripts/smoke-multi-symbol.sh` — Step 16 | REPEATABLE |
| Null response when no data yet | `scripts/smoke-multi-symbol.sh` — Step 13 | REPEATABLE |
| GET /execution/control → gate object | `scripts/smoke-multi-symbol.sh` — Step 15a | REPEATABLE |
| PUT /execution/control → update gate | `scripts/smoke-multi-symbol.sh` — Step 15b-15e | REPEATABLE |

## 10. Multi-Symbol Isolation (E2E)

| Behavior | Test Location | Status |
|----------|--------------|--------|
| Cross-symbol candle isolation | `scripts/smoke-multi-symbol.sh` — Step 4 | REPEATABLE |
| Cross-symbol signal isolation | `scripts/smoke-multi-symbol.sh` — Step 6 | REPEATABLE |
| Cross-symbol decision isolation | `scripts/smoke-multi-symbol.sh` — Step 8 | REPEATABLE |
| Cross-symbol strategy isolation | `scripts/smoke-multi-symbol.sh` — Step 10 | REPEATABLE |
| Cross-symbol risk isolation | `scripts/smoke-multi-symbol.sh` — Step 12 | REPEATABLE |
| Cross-symbol execution isolation | `scripts/smoke-multi-symbol.sh` — Step 14 | REPEATABLE |

---

## Summary

| Category | Tests | Status |
|----------|-------|--------|
| Domain model | 30 unit tests | ALL PASS |
| Control gate | 11 unit tests + 5 smoke steps | ALL PASS |
| Evaluator (derive path) | 9 unit tests | ALL PASS |
| Fill simulator | 7 unit tests | ALL PASS |
| Pipeline integration | 3 unit tests | ALL PASS |
| Projection (store path) | 21 unit tests | ALL PASS |
| Trace persistence | 4 unit tests + 1 smoke step | ALL PASS |
| Lifecycle fields | 2 unit tests + 2 smoke assertions | ALL PASS |
| Query surface | 6 smoke assertions | REPEATABLE |
| Multi-symbol isolation | 6 smoke steps | REPEATABLE |

**Total: 87 unit tests + 16-step E2E smoke script**

---

## Known Gaps (Accepted for Phase 1)

| Gap | Reason | Impact | When |
|-----|--------|--------|------|
| No NATS integration test for KV monotonicity | Requires running NATS server | Medium — KV adapter untested with real NATS | S80 |
| No publisher gate integration test | Requires NATS + control KV | Medium — gate check logic proven by design, not live test | S80 |
| No consumer redelivery/termination test | Requires NATS JetStream | Low — counters exist, behavior documented | S80 |
| No concurrent write test | Single-writer invariant by design | Low — no concurrent writers in paper mode | S81+ |
| No per-symbol gate control test | Global gate only (by design) | Low — per-symbol is future scope | S81+ |
| No fill consistency enforcement in Validate() | Deferred to keep gate simple | Low — paper fills are always consistent | S81+ |
| No cross-domain trace correlation query | No endpoint for full causal chain | Low — trace fields present, query is future | S81+ |
