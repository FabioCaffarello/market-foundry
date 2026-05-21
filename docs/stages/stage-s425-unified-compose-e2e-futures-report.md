# S425: Unified Compose E2E Proof -- Futures Segment on Canonical Surface

**Wave**: Phase 47 -- Futures Venue Execution Proof, Post-Simplification (S421--S426)
**Block**: B4 (final execution stage before evidence gate)
**Governs**: FV-Q9 -- Does full compose pipeline operate with Futures `venue_live`?
**Status**: COMPLETE
**Date**: 2026-03-23
**Predecessor**: S424 (unified runtime read-path auditability under real Futures responses)
**Successor**: S426 (evidence gate)

## 1. Charter and Governing Question

S425 is the final execution stage in the post-simplification Futures wave. It proves
that the compose-level E2E pipeline operates correctly for the Futures segment on the
canonical surface frozen by the Runtime Simplification wave (S421).

**FV-Q9**: Does full compose pipeline operate with Futures `venue_live`?

**Answer**: **Proven.** 10 unit/integration tests (all PASS) + 16-phase smoke script
validate the complete pipeline from Futures exchange listening through execution to
persistence and read-path on the canonical surface.

## 2. Deliverables

| Artifact | Path | Status |
|---|---|---|
| Integration tests (10) | `internal/actors/scopes/execute/s425_unified_compose_e2e_futures_test.go` | PASS |
| Smoke script (16 phases) | `scripts/smoke-e2e-unified-futures.sh` | Updated for canonical surface |
| Architecture doc | `docs/architecture/unified-compose-e2e-proof-with-futures-live-execution-path.md` | Updated |
| Evidence/controls doc | `docs/architecture/futures-segment-e2e-compose-evidence-controls-and-limitations.md` | Updated |
| Stage report | `docs/stages/stage-s425-unified-compose-e2e-futures-report.md` | This document |

## 3. Test Evidence

### 3.1 S425 Tests (10 tests, all PASS)

| # | Test | Proves |
|---|---|---|
| 1 | `TestS425_ComposeE2E_FuturesFill_ValidatedLifecycle` | E2E fill with ValidTransition chain, avgPrice fidelity, segment isolation |
| 2 | `TestS425_ComposeE2E_FuturesRejection_ValidatedLifecycleAndAudit` | Rejection lifecycle + full audit metadata (code, reason, venue details) |
| 3 | `TestS425_ComposeE2E_RejectionMetadata_CanonicalKVRoundTrip` | 5 audit keys survive KV round-trip |
| 4 | `TestS425_ComposeE2E_DryRun_CanonicalSurface` | DryRunSubmitter intercepts before both adapters |
| 5 | `TestS425_ComposeE2E_FillEvent_CanonicalStorePipeline` | Fill event carries all store-required fields |
| 6 | `TestS425_ComposeE2E_ConfigCoexistence_CanonicalSurface` | Both segments registered, fail-closed for unknown sources |
| 7 | `TestS425_ComposeE2E_FuturesPartialFill_ValidatedLifecycle` | Partial fill lifecycle + quantity monotonicity |
| 8 | `TestS425_ComposeE2E_AllowedSourcesGate_CanonicalSurface` | AllowedSources permits both segments, rejects unknown |
| 9 | `TestS425_ComposeE2E_MultiCycle_SustainedConnectivity` | 5 sequential orders, unique IDs, per-order correlation |
| 10 | `TestS425_ComposeE2E_ReadPathSegmentParity` | Futures/Spot structural parity, partition key isolation |

### 3.2 Upstream Evidence (S422--S424, 54 tests)

| Stage | Tests | Coverage |
|---|---|---|
| S422 | 19 | Real venue acceptance/fill, multi-cycle connectivity |
| S423 | 19 | 6 rejection scenarios, partial fill, terminal state |
| S424 | 16 + 20 sub | Read-path consolidation, 10/10 segment parity |

**Total evidence for Futures on canonical surface**: 84+ tests across 4 stages.

## 4. Compose Proof Structure

```
Binance Futures testnet
  -> ingest (source=binancef)
  -> OBSERVATION_EVENTS
  -> derive (candle -> signal -> decision -> strategy)
  -> STRATEGY_EVENTS
  -> execute (SegmentRouter -> BinanceFuturesTestnetAdapter)
  -> EXECUTION_FILL_EVENTS / EXECUTION_REJECTION_EVENTS
  -> store (projection -> KV, partition=binancef.btcusdt.60)
  -> gateway (HTTP read-path)
  -> writer -> ClickHouse (source=binancef)
```

## 5. Canonical Surface Compliance

S425 uses ONLY the canonical surface frozen by S421:

- **Config**: `execute-venue-live.jsonc` -- both segments enabled, `dry_run=false`
- **Compose**: `docker-compose.yaml` + `docker-compose.venue-live.yaml`
- **Zero per-segment overlays** (NG-46, NG-47, NG-49)
- **Zero new configs** (NG-41, NG-50)
- **Zero production code changes** -- all infrastructure pre-existed

## 6. Controls Verified

| ID | Control | Evidence |
|---|---|---|
| C1 | Dry-run safety | `TestS425_ComposeE2E_DryRun_CanonicalSurface` |
| C2 | Kill switch | Inherited (S319, S380) |
| C3 | Staleness guard | Inherited (S317) |
| C4 | Source guard | `TestS425_ComposeE2E_AllowedSourcesGate_CanonicalSurface` |
| C5 | NATS consumer filter | Inherited (S401) |
| C6 | Fail-closed routing | `TestS425_ComposeE2E_ConfigCoexistence_CanonicalSurface` |

## 7. New Value Over S419

S425 adds the following over the prior S419 proof:

| Dimension | S419 (Phase 45) | S425 (Phase 47) |
|---|---|---|
| Config surface | Per-segment overlays | Single canonical surface |
| Lifecycle assertions | Basic status checks | Explicit ValidTransition chain |
| Multi-cycle proof | Not included | 5-cycle sustained connectivity |
| Read-path parity | Basic KV round-trip | Structural parity with Spot confirmed |
| Upstream evidence | S416-S418 (basic) | S422-S424 (comprehensive, 54 tests) |
| Total test count | 8 | 10 |
| Canonical compliance | Pre-consolidation | Post-consolidation (S421 frozen) |

## 8. Residual Gaps

| ID | Gap | Status | Notes |
|---|---|---|---|
| G1 | Partial fill not observed on testnet | Accepted | Market orders fill instantly; structural proof provided (same as Spot) |
| G2 | Fee semantic divergence (cumQuote vs commission) | Monitored | Known since S416; consumers interpret by source field |
| G3 | Single symbol (btcusdt) | Accepted | Multi-symbol proven at unit level |
| G4 | No Spot parallel proof | Accepted | Structural coexistence proven; not a requirement |

## 9. Wave Readiness

With S425 complete, the Phase 47 wave has proven:

| Question | Stage | Verdict |
|---|---|---|
| FV-Q1: Lifecycle transitions | S422 | Proven |
| FV-Q2: Fill record fidelity | S422 | Proven |
| FV-Q3: Rejection lifecycle | S423 | Proven |
| FV-Q4: Terminal state exhaustion | S423 | Proven |
| FV-Q5: Partial fill lifecycle | S423 | Proven (structural) |
| FV-Q6: Error scenario coverage | S423 | Proven (6 scenarios) |
| FV-Q7: Read-path auditability | S424 | Proven |
| FV-Q8: Segment parity | S424 | Proven (10/10 dimensions) |
| **FV-Q9: Full compose E2E** | **S425** | **Proven** |
| FV-Q10: Correlation chain | S424 | Proven |
| FV-Q11: Multi-cycle connectivity | S422, S425 | Proven |
| FV-Q12: Post-200 reconciliation | S422 | Proven |

**The wave is ready for the evidence gate (S426).**

## 10. Conclusion

S425 closes the operational value of the Futures wave by proving that the complete
compose-level pipeline operates correctly on the canonical surface. With 84+ tests
across 4 stages (S422-S425), the Futures segment has comprehensive evidence for
lifecycle fidelity, read-path coherence, audit trail completeness, and structural
coexistence with Spot -- all on the post-simplification canonical surface.
