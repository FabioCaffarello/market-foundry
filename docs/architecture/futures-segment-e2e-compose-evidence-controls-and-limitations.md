# Futures Segment E2E Compose Evidence, Controls, and Limitations

**Stage**: S425 (supersedes S419)
**Wave**: Phase 47 -- Futures Venue Execution Proof, Post-Simplification (S421--S426)
**Status**: Active
**Date**: 2026-03-23

## 1. Evidence Matrix

### 1.1 Unit/Integration Tests (10 tests, all PASS)

| Test | What It Proves |
|---|---|
| `TestS425_ComposeE2E_FuturesFill_ValidatedLifecycle` | Full E2E fill with explicit ValidTransition chain (submitted -> accepted -> filled), avgPrice-based fill fidelity, segment isolation, correlation chain |
| `TestS425_ComposeE2E_FuturesRejection_ValidatedLifecycleAndAudit` | Rejection with ValidTransition (submitted -> rejected), full audit metadata (code=-2019, venue HTTP 400), terminal state verification |
| `TestS425_ComposeE2E_RejectionMetadata_CanonicalKVRoundTrip` | 5 audit metadata keys survive JSON serialize/deserialize (KV persistence guarantee) |
| `TestS425_ComposeE2E_DryRun_CanonicalSurface` | DryRunSubmitter intercepts Futures intents before SegmentRouter; neither adapter contacted |
| `TestS425_ComposeE2E_FillEvent_CanonicalStorePipeline` | VenueOrderFilledEvent carries Futures segment identity, store-ready fields, real venue order ID, partition key |
| `TestS425_ComposeE2E_ConfigCoexistence_CanonicalSurface` | Both segments registered on canonical config; correct routing; unknown sources rejected (fail-closed) |
| `TestS425_ComposeE2E_FuturesPartialFill_ValidatedLifecycle` | Partial fill with ValidTransition chain (submitted -> accepted -> partially_filled), quantity monotonicity, avgPrice-based fill |
| `TestS425_ComposeE2E_AllowedSourcesGate_CanonicalSurface` | AllowedSources gate permits binancef and binances; unknown sources rejected |
| `TestS425_ComposeE2E_MultiCycle_SustainedConnectivity` | 5 sequential orders through same router: unique VenueOrderIDs, per-order correlation chain, stable segment identity |
| `TestS425_ComposeE2E_ReadPathSegmentParity` | Futures and Spot fills serialize to same LifecycleEntry structure; partition keys structurally isolated |

### 1.2 Prerequisite Evidence (S422--S424, 54 tests)

| Stage | Tests | Coverage |
|---|---|---|
| S422 | 19 | Real venue acceptance/fill, multi-cycle connectivity, ValidTransition |
| S423 | 19 | 6 rejection scenarios, partial fill, terminal state exhaustion |
| S424 | 16 + 20 sub | Read-path consolidation, 10/10 segment parity dimensions |

### 1.3 Compose-Level Evidence (16-Phase Smoke Script)

| Phase | Gate | What It Proves |
|---|---|---|
| 1. Stack Readiness | Hard | All 9 services healthy |
| 2. Unit Tests | Hard | S425 tests (10) + S422-S424 prerequisites pass |
| 3. Credential Detection | -- | Selects venue_live or dry-run mode |
| 4. Unified Compose Boot | Hard | Execute boots with Futures on canonical surface |
| 5. Active Bindings | Hard | Futures bindings (source=binancef) present |
| 6. Live Exchange Data | Hard | OBSERVATION_EVENTS growing from Futures feed |
| 7. Derive Pipeline | Hard | STRATEGY_EVENTS produced from Futures data |
| 8. Execute Consumption | Soft | Strategy events consumed by execute |
| 9. Venue Adapter | Soft | Futures execution activity detected |
| 10. Fill/Rejection Stream | Soft | EXECUTION_FILL_EVENTS populated |
| 11. Store Read-Path | Soft | Futures evidence queryable via HTTP |
| 12. Analytical Persistence | Soft | Futures data in ClickHouse |
| 13. Correlation Chain | Soft | Composite chains with source=binancef |
| 14. Segment Isolation | Soft | Spot has zero execution activity |
| 15. Config Restore | Hard | Default paper config restored |
| 16. Stream Delta Summary | Info | End-to-end traceability summary |

Hard gates must pass for the proof to succeed. Soft phases provide additional evidence that depends on live market conditions producing actionable strategies.

## 2. Safety Controls

### C1: Dry-Run Protection

DryRunSubmitter wraps the entire SegmentRouter. When `dry_run=true` (the compose default config), ALL intents -- both Futures and Spot -- are intercepted before reaching any real adapter.

**Evidence**: `TestS425_ComposeE2E_DryRun_CanonicalSurface` proves neither adapter is contacted.

### C2: Kill Switch

The EXECUTION_CONTROL KV gate can halt all execution at runtime. This is inherited from S319/S380 and applies equally to Futures intents.

### C3: Staleness Guard

StalenessGuard rejects intents older than `staleness_max_age` (120s in config). This prevents stale Futures strategies from reaching the venue.

### C4: Segment Source Guard

AllowedSources in VenueAdapterActor filters intents by source. Only sources from enabled segments (`binancef`, `binances`) pass through.

**Evidence**: `TestS425_ComposeE2E_AllowedSourcesGate_CanonicalSurface`

### C5: NATS Consumer Filter

NATS consumers use segment-aware subject filters, preventing cross-segment event leakage at the transport layer.

### C6: Fail-Closed Routing

SegmentRouter rejects intents with unrecognized sources. There is no default or fallback adapter.

**Evidence**: `TestS425_ComposeE2E_ConfigCoexistence_CanonicalSurface`

## 3. Persistence and Read-Path Coherence

### 3.1 KV Round-Trip

Futures rejection metadata (5 audit keys) survives JSON serialize/deserialize, matching the store projection path on the canonical surface.

**Evidence**: `TestS425_ComposeE2E_RejectionMetadata_CanonicalKVRoundTrip`

### 3.2 Partition Key Isolation

All Futures data uses partition key `binancef.btcusdt.60`, structurally isolated from Spot data at `binances.btcusdt.60`. Read-path queries filter by source, ensuring no cross-segment contamination.

**Evidence**: `TestS425_ComposeE2E_ReadPathSegmentParity`

### 3.3 Analytical Persistence

ClickHouse tables store Futures data with `source = 'binancef'`, enabling segment-scoped analytical queries.

## 4. Spot Coexistence

Spot structural coexistence is preserved during the Futures E2E proof:

- Both segments remain registered in SegmentRouter
- Spot adapter is NOT contacted during Futures execution (proven by guard server)
- Spot configuration remains enabled in the canonical config
- AllowedSources gate permits both `binancef` and `binances`
- Read-path parity confirmed: same LifecycleEntry structure, different partition keys

This is the inverse of S408, which proved Spot E2E while Futures remained structurally present.

## 5. Canonical Surface Compliance

S425 operates exclusively on the canonical surface frozen by S421:

- **Config**: `execute-venue-live.jsonc` (shared by both segments)
- **Compose**: `docker-compose.yaml` + `docker-compose.venue-live.yaml`
- **No per-segment overlays** (NG-46, NG-47, NG-49)
- **No new configs** (NG-41, NG-50)
- **Zero production code changes** -- all infrastructure pre-existed on the unified runtime

## 6. Limitations

| ID | Limitation | Impact | Mitigation |
|---|---|---|---|
| L1 | Pipeline coverage depends on live market conditions | Soft smoke phases may show zero activity if no actionable strategies are produced | Hard evidence from deterministic unit tests covers all paths |
| L2 | Single symbol (btcusdt) | Multi-symbol Futures not proven at compose level | Sufficient for representational proof; multi-symbol proven at unit level |
| L3 | Testnet only | No mainnet proof | Guard rail: no mainnet in this wave |
| L4 | Point-in-time proof | Long-running stability not assessed | Soak testing out of scope; S412 covered endurance |
| L5 | Credential dependency | venue_live mode requires real testnet credentials | Smoke script degrades gracefully to dry-run mode |
| L6 | No Spot parallel proof | Spot is not exercised simultaneously | Structural coexistence proven; parallel proof is not a requirement |
| L7 | Partial fill not observed on testnet | Market orders fill instantly on Futures testnet | Structural proof provided (same as Spot, accepted since S423) |

## 7. Evidence Strength Assessment

| Dimension | Strength | Notes |
|---|---|---|
| Futures lifecycle (fill, rejection, partial fill) | **Strong** | ValidTransition chain assertions on all paths |
| Correlation chain | **Strong** | Preserved through all E2E paths, multi-cycle uniqueness proven |
| Segment isolation | **Strong** | Guard servers prove no cross-segment calls |
| Dry-run safety | **Strong** | DryRunSubmitter wraps entire router on canonical surface |
| Persistence/read-path | **Strong** | KV round-trip + partition key isolation + segment parity |
| Compose wiring | **Moderate to Strong** | Depends on market conditions for full pipeline evidence |
| Analytical persistence | **Moderate** | Depends on pipeline producing data during smoke window |
| Multi-cycle connectivity | **Strong** | 5 sequential orders with unique IDs and preserved correlation |
| Canonical surface compliance | **Strong** | Zero deviations from S421-frozen surface |

## 8. Conclusion

S425 establishes the Futures segment as a fully proven E2E pipeline at the compose level on the canonical surface. Combined with S422 (venue connectivity), S423 (rejection/partial fill), and S424 (read-path consolidation), the Futures wave value chain is complete on the post-simplification canonical surface.

The wave is ready for the evidence gate (S426).
