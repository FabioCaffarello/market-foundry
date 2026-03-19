# Execute Actor Critical Test Coverage

> Maps the safety-critical behaviors of the execute actor to their test evidence.

## Test Inventory

### SafetyGate Tests (`internal/application/execution/safety_gate_test.go`)

Tests the combined pre-submit gate logic independently from infrastructure.

| Test | Gate | Proves |
|------|------|--------|
| `TestSafetyGate_KillSwitch_Halted_BlocksSubmission` | Gate 1 | Halted gate blocks intent, reason = `kill_switch` |
| `TestSafetyGate_KillSwitch_Active_AllowsSubmission` | Gate 1 | Active gate allows intent |
| `TestSafetyGate_KillSwitch_Nil_FailOpen` | Gate 1 | Nil gate checker = fail-open |
| `TestSafetyGate_KillSwitch_Timeout_FailOpen` | Gate 1 | Slow gate read = fail-open (50ms timeout) |
| `TestSafetyGate_Staleness_StaleIntent_Blocked` | Gate 2 | 5min-old intent blocked with 2min guard |
| `TestSafetyGate_Staleness_FreshIntent_Allowed` | Gate 2 | 30s-old intent passes |
| `TestSafetyGate_Staleness_ExactBoundary_Allowed` | Gate 2 | Exact boundary = NOT stale |
| `TestSafetyGate_Staleness_NilGuard_SkipsCheck` | Gate 2 | Nil staleness guard = check skipped |
| `TestSafetyGate_KillSwitchBlocksBeforeStaleness` | Priority | Kill switch blocks even for fresh intents |
| `TestSafetyGate_KillSwitchHalted_StaleIntent_ReportsKillSwitch` | Priority | Both would block → kill_switch reason reported |
| `TestSafetyGate_AllGatesPass` | Combined | Fresh intent + active gate = allowed |
| `TestSafetyGate_FutureTimestamp_Allowed` | Edge | Future timestamp = not stale |
| `TestSafetyGate_ZeroTimestamp_Stale` | Edge | Zero timestamp = stale |
| `TestSafetyGate_DefaultGateReadTimeout` | Config | Zero timeout defaults to 2s |
| `TestSafetyGate_BothNil_FullyOpen` | Degraded | No gate checker + no staleness = fully open |

**Total: 15 tests**

### StalenessGuard Tests (`internal/application/execution/staleness_guard_test.go`)

Tests the staleness calculation in isolation.

| Test | Proves |
|------|--------|
| `TestStalenessGuard_Fresh` | 30s-old intent with 2min max = not stale |
| `TestStalenessGuard_Stale` | 3min-old intent with 2min max = stale |
| `TestStalenessGuard_ExactBoundary` | At exact max age = NOT stale (`>` not `>=`) |
| `TestStalenessGuard_FutureTimestamp` | Future timestamp = not stale |
| `TestStalenessGuard_ZeroMaxAge_EverythingStale` | Zero max age: past = stale, exact now = not stale |
| `TestStalenessGuard_ZeroTimestamp` | Zero-value timestamp = stale |
| `TestStalenessGuard_LargeClockSkew` | 1h future = not stale, 24h past = stale |
| `TestStalenessGuard_JustOverBoundary` | 1ns past boundary = stale |
| `TestStalenessGuard_JustUnderBoundary` | 1ns under boundary = not stale |

**Total: 9 tests**

### ControlGate Domain Tests (`internal/domain/execution/control_test.go`)

Tests the kill switch domain model.

| Test | Proves |
|------|--------|
| `TestValidGateStatus_Active` | `active` is valid |
| `TestValidGateStatus_Halted` | `halted` is valid |
| `TestValidGateStatus_Unknown` | Unknown values rejected |
| `TestValidGateStatus_Empty` | Empty string rejected |
| `TestControlGate_IsHalted_WhenHalted` | IsHalted=true when status=halted |
| `TestControlGate_IsHalted_WhenActive` | IsHalted=false when status=active |
| `TestControlGate_IsHalted_ZeroValue` | Zero-value gate = not halted (fail-open) |
| `TestDefaultControlGate_IsActive` | Default = active |
| `TestDefaultControlGate_HasTimestamp` | Default has valid timestamp |
| `TestDefaultControlGate_NoReason` | Default has no reason |
| `TestDefaultControlGate_NoUpdatedBy` | Default has no updated_by |
| `TestControlGate_HaltAndResume` | Full halt → resume cycle works |
| `TestControlGate_HaltPreservesAuditFields` | Audit fields preserved after halt |

**Total: 13 tests**

### PaperVenueAdapter Tests (`internal/application/execution/paper_venue_adapter_test.go`)

Tests the paper (simulated) venue adapter.

| Test | Proves |
|------|--------|
| `TestPaperVenueAdapter_SubmitOrder_Buy` | Buy order fills correctly |
| `TestPaperVenueAdapter_SubmitOrder_Sell` | Sell order fills correctly |
| `TestPaperVenueAdapter_SubmitOrder_NoAction` | SideNone = accepted, no fill |
| `TestPaperVenueAdapter_UniqueVenueOrderIDs` | 10 orders = 10 unique IDs |
| `TestPaperVenueAdapter_ImplementsVenuePort` | Compile-time interface check |
| `TestPaperVenueAdapter_SubmitOrder_CancelledContext` | Paper adapter ignores cancelled context (documented) |
| `TestPaperVenueAdapter_FillDelay_RespectsDelay` | Fill delay configuration works |

**Total: 7 tests**

### Pipeline Integration Tests (`internal/application/execution/pipeline_integration_test.go`)

Tests end-to-end chains without infrastructure.

| Test | Proves |
|------|--------|
| `TestPipeline_EvaluateSimulateEmit_BuyOrder` | Full derive-side pipeline |
| `TestPipeline_EvaluateSimulateEmit_RejectedRisk_NoFill` | Rejected risk = no fill |
| `TestPipeline_MultiSymbol_FullIsolation` | Multi-symbol partition isolation |
| `TestPipeline_VenueAdapter_FullChain_DeriveToFill` | derive → execute → fill chain with trace |
| `TestPipeline_VenueAdapter_NoAction_NoFillRecord` | No-action through venue = no fill |
| `TestPipeline_StalenessGuard_Integration` | Staleness + venue chain integration |
| `TestPipeline_StatusPropagation_IntentAndResult` | Composite status derivation |
| `TestPipeline_MultiSymbol_FillIsolation` | Multi-symbol fill isolation |

**Total: 8 tests**

### ExecutionIntent Domain Tests (`internal/domain/execution/execution_test.go`)

30+ tests covering validation, lifecycle transitions, partition keys, deduplication.

## Coverage Matrix

| Safety Behavior | Unit Test | Integration Test | Edge Cases |
|----------------|-----------|------------------|------------|
| Kill switch blocks when halted | SafetyGate | - | Timeout fail-open, nil fail-open, zero-value fail-open |
| Kill switch fail-open on KV failure | SafetyGate | - | Nil checker, slow checker |
| Staleness blocks old intents | StalenessGuard + SafetyGate | Pipeline | Zero ts, boundary, 1ns boundary, future, clock skew, zero maxAge |
| Submit timeout terminates hanging calls | BinanceAdapter | - | Context deadline exceeded (Binance test) |
| Gate evaluation order | SafetyGate | - | Kill switch before staleness |
| Paper fills are simulated | PaperVenueAdapter | Pipeline | Fill records marked simulated=true |
| Trace chain preservation | - | Pipeline | CorrelationID + CausationID flow |
| Multi-symbol isolation | - | Pipeline | Partition keys, dedup keys, venue IDs |

## Not Covered (Known Gaps)

| Gap | Reason | Risk |
|-----|--------|------|
| Fill publish failure after successful submit | Requires NATS mock; application layer has no retry | Medium — fills can be lost |
| Actor startup with unavailable NATS | Requires running NATS; tested manually | Low — fail-fast on startup |
| Real venue adapter timeout via context expiry | Covered by Binance adapter test, not by actor | Low — context propagation is stdlib |
| Kill switch read via real KV store | Integration test territory; covered by contract | Low — `IsHalted` is a thin wrapper |
| Concurrent intent processing | Actor framework serializes messages | N/A — not applicable |
