# Stage S368 — End-to-End Analytical-to-Execution Report

> Derive Integration Wave — DI-4: Capstone proof of the complete
> analytical pipeline from derive through execution.

## Executive Summary

S368 proves the complete connected pipeline: derive → StrategyResolvedEvent →
store/gateway/read-path → source-driven execution path → venue-active behavior.
18 new end-to-end tests exercise the **real derive resolver** output flowing
through **real consumer actors**, validating all 11 contract invariants, the
complete correlation chain, safety gates, severity scaling, and store
materialization. Zero production code changes required. Zero regressions.

The wave's strategic goal — "prove derive produces StrategyResolvedEvent that
drives execution end-to-end" — is concretely demonstrated.

## Scope

- **In scope**: End-to-end pipeline proof with real derive resolver output
- **Out of scope**: Multi-binary orchestration, ClickHouse writer, new families, multi-venue

## Pipeline Validated

```
Decision (rsi_oversold, triggered, high, 0.8500)
  ↓ MeanReversionEntryResolver.Resolve()
Strategy (mean_reversion_entry, long, 0.8500, final=true)
  ↓ StrategyResolvedEvent via NATS
  ├─→ StrategyProjectionActor → KV → QueryResponder → HTTP
  └─→ StrategyConsumerActor → PaperOrderEvaluator
       ↓ ExecutionIntent (buy, 0.01, pass_through)
       VenueAdapterActor → SafetyGate → PaperVenueAdapter
       ↓ VenueOrderFilledEvent (simulated fill)
```

**Verdict**: Pipeline operational end-to-end. All segments connected. No broken links.

## Files Changed

### New Test Files

| File | Tests | Purpose |
|------|-------|---------|
| `internal/actors/scopes/execute/e2e_derive_to_execution_test.go` | 12 | Derive→execute→venue full pipeline proof |
| `internal/actors/scopes/store/e2e_derive_to_store_test.go` | 6 | Derive→store→query read-path proof |

### New Documentation

| File | Purpose |
|------|---------|
| `docs/architecture/end-to-end-analytical-to-execution-proof.md` | Complete pipeline architecture and invariant coverage |
| `docs/architecture/derive-to-venue-canonical-pipeline-evidence-and-limitations.md` | Evidence matrix, limitations, controls, auditability |
| `docs/stages/stage-s368-end-to-end-analytical-to-execution-report.md` | This report |

### No Production Code Changes

Zero production code changes were required. The connected pipeline was already
correctly wired from prior stages (S358–S367).

## Key Evidence

### E1: Triggered Decision → Buy Execution (full chain)

A real derive resolver produces a long strategy from an RSI oversold triggered
decision. The strategy consumer actor evaluates it into a buy execution intent.
The paper venue adapter submits and fills. The fill event preserves the complete
correlation chain.

**Test**: `TestE2E_FullPipeline_DeriveToVenueFill`

### E2: Flat Decision → No Execution

A not-triggered decision produces a flat strategy with zero confidence. The
execute consumer produces side=none, quantity=0. The intent is forwarded for
observability but produces no venue action.

**Tests**: `TestE2E_DeriveNotTriggered_ProducesNoExecution`, `TestE2E_DeriveInsufficientData_ProducesNoExecution`

### E3: Correlation Chain Unbroken

The 5-hop correlation chain (decision → strategy → execute intent → submit event → fill event)
is verified with concrete assertions at every boundary.

**Test**: `TestE2E_FullPipeline_DeriveToVenueFill`

### E4: Severity Scaling Flows End-to-End

Three severity levels (high, moderate, low) produce correctly scaled confidence
values that flow from the derive resolver through to execution risk metadata.

**Test**: `TestE2E_DeriveSeverityScaling_FlowsToExecution` (3 subtests)

### E5: Safety Gates Accept/Reject Derive Events

Fresh derive-produced events pass the staleness guard. Replayed events with
old timestamps are correctly blocked. The confidence threshold gate correctly
filters low-confidence derive events while passing high-confidence ones.

**Tests**: `TestE2E_SafetyGate_AcceptsFreshDeriveEvent`,
`TestE2E_SafetyGate_RejectsStaleReplayedDeriveEvent`,
`TestE2E_ConfidenceThreshold_FiltersDeriveLowConfidence`,
`TestE2E_ConfidenceThreshold_PassesDeriveHighConfidence`

### E6: Store Materialization with Real Derive Output

All 16 strategy fields survive KV round-trip. Decision inputs, parameters, and
domain metadata are preserved. Monotonicity guard correctly rejects stale events
and accepts newer ones.

**Tests**: `TestE2E_Store_DeriveTriggered_Materializes`,
`TestE2E_Store_NewerDeriveEventOverwrites`,
`TestE2E_Store_MonotonicityRejectsStale`

### E7: Tracker Metrics Across Scopes

Health tracker counters correctly accumulate across actionable and flat derive
events, proving observability works with real pipeline output.

**Test**: `TestE2E_TrackerMetrics_CrossScope`

## Remaining Limitations

| ID | Limitation | Impact | Mitigation |
|----|-----------|--------|------------|
| L1 | Event metadata not in KV | No HTTP-visible trace | Log aggregation or NATS replay |
| L2 | No multi-binary orchestration test | Pipeline proven in-process only | Live smoke scripts exist separately |
| L3 | ClickHouse writer path not verified | Analytical completeness gap | Separate verification scope |
| L4 | Only mean_reversion_entry tested E2E | Other families not proven | Pattern is mechanical; same wiring |
| L5 | No backpressure from execute to derive | Potential lag under extreme load | NATS consumer buffering |

## Test Results

```
ok  internal/actors/scopes/execute   0.861s   (27 tests — 15 existing + 12 new)
ok  internal/actors/scopes/store     0.171s   (33 tests — 27 existing + 6 new)
ok  internal/actors/scopes/derive    (all existing pass — zero regressions)
ok  internal/adapters/nats/natsstrategy (all existing pass — zero regressions)
ok  internal/application/execution   (all existing pass — zero regressions)
ok  internal/application/strategy    (all existing pass — zero regressions)
```

Zero regressions across all packages. 18 new tests, all PASS.

## Preparation for S369

S369 is the evidence gate closing the Derive Integration Wave. Based on S368 findings:

1. **All DI-4 deliverables complete**: E2E integration tests, correlation chain
   verification, invariant verification across the full path, and safety gate
   validation with derive-produced events.

2. **Remaining evaluation for S369**:
   - Formal evidence matrix with capability classifications (FULL/SUBSTANTIAL/PARTIAL)
   - Audit of all 5 charter blocks (DI-1 through DI-5) against success criteria
   - Regression verification across all 8 binary targets
   - Residual gaps catalog with explicit severity ratings
   - Next ceremony recommendation

3. **Known gaps to document in S369**:
   - L1 (event metadata loss in KV) — accepted as known limitation
   - L2 (no multi-binary orchestration test) — deferred to operational wave
   - L3 (ClickHouse writer) — deferred to analytical wave
   - L4 (other families) — mechanical extension, not blocking

4. **Wave success criteria status**:
   - ✅ Derive resolver satisfies all 11 invariants (S365 + S366)
   - ✅ Derive publisher produces correct NATS messages (S366)
   - ✅ Store materializes derive-produced events (S367 + S368)
   - ✅ Gateway returns derive-produced state (S367)
   - ✅ Full pipeline test: derive → strategy → execute → fill (S368)
   - ✅ Zero regressions (verified S368)
   - ⏳ Evidence gate with all questions at HIGH confidence (S369)

## Conclusion

Stage S368 closes the capstone proof of the Derive Integration Wave. The
analytical pipeline from decision evaluation through venue-active execution is
proven with real derive resolver output, complete correlation chain preservation,
and all contract invariants verified end-to-end. The codebase is ready for the
formal evidence gate in S369.
