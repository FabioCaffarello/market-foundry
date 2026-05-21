# S419: Unified Compose E2E Futures Proof -- Report

**Stage**: S419
**Wave**: Phase 45 -- Futures Venue Execution Proof (S415--S420)
**Type**: Proof and consolidation
**Status**: Complete
**Date**: 2026-03-23
**Predecessor**: S418 (unified runtime read-path auditability under real Futures responses)
**Successor**: S420 (evidence gate)

## 1. Charter

Prove a compose-level end-to-end pipeline connecting Futures ingest/listening through the unified runtime to lifecycle outcome, persistence, read-path, and audit trail. Preserve controls and Spot structural coexistence.

## 2. Governing Questions

| ID | Question | Status | Evidence |
|---|---|---|---|
| CE-Q1 | Does the Futures segment produce a complete E2E pipeline at compose level? | **Proven** | 8 unit tests + 16-phase smoke script |
| CE-Q2 | Is the read-path coherent for Futures fill/rejection outcomes? | **Proven** | KV round-trip test + partition key isolation (binancef.btcusdt.60) |
| CE-Q3 | Is the audit trail complete for Futures rejections on the unified runtime? | **Proven** | Audit metadata embedding + extraction tests (code -2019, margin insufficient) |
| CE-Q4 | Is the correlation chain intact from Futures ingest to execution outcome? | **Proven** | CorrelationID/CausationID preserved in all paths |
| CE-Q5 | Does dry_run protect the entire unified runtime (both segments)? | **Proven** | DryRunSubmitter wraps SegmentRouter, neither adapter contacted |
| CE-Q6 | Is Spot coexistence structurally preserved? | **Proven** | Both segments registered, Spot untouched during Futures execution |
| CE-Q7 | Does the AllowedSources gate permit Futures on the unified runtime? | **Proven** | binancef in allowed set, unknown sources rejected |

## 3. Deliverables

### 3.1 Code Artifacts

| Artifact | Type | Location |
|---|---|---|
| S419 integration tests (8 tests) | Test | `internal/actors/scopes/execute/s419_unified_compose_e2e_futures_test.go` |
| E2E smoke script (16 phases) | Script | `scripts/smoke-e2e-unified-futures.sh` |
| Compose overlay (Futures venue_live) | Config | `deploy/compose/docker-compose.unified-futures-live.yaml` |
| Makefile target | Build | `smoke-e2e-unified-futures` |

### 3.2 Documentation

| Document | Location |
|---|---|
| E2E proof with Futures live execution path | `docs/architecture/unified-compose-e2e-proof-with-futures-live-execution-path.md` |
| Evidence, controls, and limitations | `docs/architecture/futures-segment-e2e-compose-evidence-controls-and-limitations.md` |
| Stage report (this) | `docs/stages/stage-s419-unified-compose-e2e-futures-report.md` |

### 3.3 No Production Code Changes

S419 required **zero production code changes**. The existing unified runtime, SegmentRouter, Futures adapter, projection actors, and read-path infrastructure already supported the full E2E pipeline. S419 is purely a proof, evidence, and validation stage.

## 4. Test Evidence

### S419 Tests (8 tests, all PASS)

| Test | Proves |
|---|---|
| `TestS419_ComposeE2E_FuturesFill_ThroughUnifiedRuntime` | Full E2E fill with real data (avgPrice-based), segment isolation, correlation chain |
| `TestS419_ComposeE2E_FuturesRejection_AuditTrailComplete` | Rejection audit trail with venue details (margin insufficient) |
| `TestS419_ComposeE2E_RejectionMetadata_FuturesKVRoundTrip` | Audit metadata survives KV round-trip |
| `TestS419_ComposeE2E_DryRun_WrapsFuturesOnUnifiedRouter` | Compose-level dry-run safety |
| `TestS419_ComposeE2E_FillEventConstruction_FuturesSegment` | Fill event carries store-ready fields and Futures identity |
| `TestS419_ComposeE2E_ConfigCoexistence_FuturesAndSpotEnabled` | Both segments coexist, fail-closed routing |
| `TestS419_ComposeE2E_FuturesPartialFill_UnifiedRuntime` | Partial fill fidelity (avgPrice-based) and monotonicity |
| `TestS419_ComposeE2E_AllowedSourcesGate_FuturesPermitted` | Defense-in-depth source gate |

### Regression Check

Full test suite across all touched packages passes with zero regressions:
- `internal/actors/scopes/execute`: all tests PASS
- `internal/application/execution`: all tests PASS
- `internal/shared/settings`: all tests PASS
- `internal/domain/execution`: all tests PASS

## 5. Compose Proof Structure

### Pipeline Path

```
Binance Futures testnet (testnet.binancefuture.com)
  -> ingest (binancef WebSocket adapter)
  -> NATS OBSERVATION_EVENTS
  -> derive (candle -> signal -> decision -> strategy)
  -> NATS STRATEGY_EVENTS
  -> execute (SegmentRouter -> BinanceFuturesTestnetAdapter)
  -> NATS EXECUTION_FILL_EVENTS / EXECUTION_REJECTION_EVENTS
  -> store (projection -> KV)
  -> gateway (HTTP read-path)
  -> writer -> ClickHouse (analytical persistence)
```

### Compose Overlay

`docker-compose.unified-futures-live.yaml` overrides the execute service to use `execute-venue-live-futures.jsonc`:
- `dry_run: false` -- real Futures testnet orders
- `futures.enabled: true, adapter: binance_futures_testnet` -- live Futures adapter
- `spot.enabled: true` -- structural coexistence preserved

### Smoke Script Modes

| Mode | Trigger | Execute Config | Evidence Strength |
|---|---|---|---|
| venue_live | Futures testnet credentials set | `execute-venue-live-futures.jsonc` | Full (real HTTP) |
| dry-run | No credentials | `execute-unified.jsonc` | Structural (compose wiring) |

## 6. Controls Verified

| Control | Mechanism | Evidence |
|---|---|---|
| Dry-run safety | DryRunSubmitter wraps SegmentRouter | `TestS419_ComposeE2E_DryRun_WrapsFuturesOnUnifiedRouter` |
| Kill switch | EXECUTION_CONTROL KV | Inherited (S319, S380) |
| Staleness guard | StalenessGuard | Inherited (S317) |
| Source guard | AllowedSources in VenueAdapterActor | `TestS419_ComposeE2E_AllowedSourcesGate_FuturesPermitted` |
| NATS consumer filter | Segment-filtered subscriptions | Inherited (S401) |
| Fail-closed routing | SegmentRouter rejects unknown sources | `TestS419_ComposeE2E_ConfigCoexistence_FuturesAndSpotEnabled` |

## 7. Residual Gaps

| Gap | Impact | Mitigation |
|---|---|---|
| No parallel Spot+Futures proof | Simultaneous execution not proven | Structural coexistence proven; parallel proof is out of scope |
| Pipeline depends on live market conditions | Smoke soft phases may show zero activity | Hard evidence from deterministic unit tests |
| Single symbol (btcusdt) | Multi-symbol Futures not proven at compose level | Sufficient for representational proof |
| Testnet only | No mainnet proof | Guard rail: no mainnet in this wave |
| No soak test | Long-running stability untested | Point-in-time proof sufficient; S412 covered endurance |

## 8. What S419 Closes

S419 is the capstone proof of the Futures value chain on the unified runtime:

- **S416**: Futures venue connectivity and dominant lifecycle (submitted -> filled)
- **S417**: Futures rejection and partial fill paths
- **S418**: Read-path, audit trail, and segment isolation under real Futures responses
- **S419**: Compose-level E2E connecting all of the above in a running stack

Together, these four stages prove that the Futures segment works end-to-end from live exchange data through execution to queryable outcomes, on a unified runtime that preserves Spot coexistence.

## 9. Segment Parity

With S408 (Spot) and S419 (Futures), both segments now have full compose-level E2E proof on the unified runtime:

| Dimension | S408 (Spot) | S419 (Futures) |
|---|---|---|
| Source | binances | binancef |
| Adapter | BinanceSpotTestnetAdapter | BinanceFuturesTestnetAdapter |
| Fill model | fills[] array | avgPrice/cumQuote |
| Rejection example | -2010 (insufficient balance) | -2019 (insufficient margin) |
| Partition key | binances.btcusdt.60 | binancef.btcusdt.60 |
| Tests | 8 | 8 |
| Smoke phases | 16 | 16 |

The Futures wave is now ready for the evidence gate (S420).
