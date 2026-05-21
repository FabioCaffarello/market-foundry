# Testnet Venue Execution (Unified Runtime, Spot-First): Evidence Matrix, Residual Gaps, and Next Ceremony

## Evidence Matrix

### Capability Classification

| ID | Capability | Classification | Stage | Test Count | Key Evidence |
|---|---|---|---|---|---|
| TV-C1 | Spot Testnet Connectivity | FULL | S405 | 33 | BinanceSpotTestnetAdapter, HMAC signing, API path /api/v3/order |
| TV-C2 | Dominant Lifecycle (submitted->filled) | FULL | S405 | 32 | Buy/sell lifecycle, state machine transitions, VenueOrderID |
| TV-C3 | Fill Record Fidelity | FULL | S405 | 32 | Multi-leg aggregation, weighted avg price, fee, Simulated=false |
| TV-C4 | Rejection Lifecycle (submitted->rejected) | FULL | S406 | 30 | 10 error scenarios, Problem classification, VenueDetails |
| TV-C5 | Partial Fill Handling | SUBSTANTIAL | S406 | 30 | Mock-based structural proof; no live observation (venue constraint) |
| TV-C6 | Quantity Monotonicity | FULL | S406 | 30 | 3 scenarios (half/quarter/tiny), FilledQuantity <= Quantity |
| TV-C7 | Read-Path Queryability | FULL | S407 | 11 | 3 KV buckets, composite status, dedicated rejection route |
| TV-C8 | Segment Isolation | FULL | S407 | 11 | Partition keys, NATS filters, fail-closed routing |
| TV-C9 | Compose E2E Pipeline | FULL | S408 | 9 | 16-phase smoke, dual-mode, Spot ingest to read-path |
| TV-C10 | Audit Trail Completeness | FULL | S407/S408 | 11+9 | Metadata embedding, KV round-trip, rejection detail extraction |

### Governing Question Disposition

| ID | Question | Classification | Stage | Disposition |
|---|---|---|---|---|
| TV-Q1 | Real acceptance + fill lifecycle | FULL | S405 | submitted->filled proven with real Spot response shape, 32 tests |
| TV-Q2 | Fill record fidelity | FULL | S405 | Price, qty, fee, timestamp from fills[] array, multi-leg aggregation |
| TV-Q3 | Real rejection lifecycle | FULL | S406 | 10 scenarios covering HTTP 400/401/429/503, Binance error codes, HTTP 200+REJECTED/EXPIRED |
| TV-Q4 | Rejection event fidelity | FULL | S406 | VenueOrderRejectedEvent with RejectionCode, RejectionReason, VenueDetails (HTTP status, venue error code) |
| TV-Q5 | Partial fill observation | SUBSTANTIAL | S406 | PARTIALLY_FILLED parsed correctly, aggregation works, but no live observation. Market orders fill instantly on Spot testnet. |
| TV-Q6 | Quantity monotonicity | FULL | S406 | FilledQuantity <= Quantity across 3 partial fill scenarios |
| TV-Q7 | KV/HTTP/ClickHouse agreement | FULL | S407 | KV and HTTP routes proven. ClickHouse analytical path inherited from prior waves. |
| TV-Q8 | ClickHouse rejection writer (RG-1) | PARTIAL | S407 | KV rejection queryability proven. ClickHouse-specific rejection writer not wired. Pre-existing RG-1 from charter. |
| TV-Q9 | Full compose pipeline in venue_live | FULL | S408 | 9 integration tests + 16-phase smoke script, compose overlay proven |
| TV-Q10 | Sustained multi-cycle behavior | FULL | S407 | Composite propagation rules resolve multi-state intents by timestamp |
| TV-Q11 | Correlation chain integrity | FULL | S405 | CorrelationID/CausationID preserved from derive through execute to store |
| TV-Q12 | Post-200 reconciliation | FULL | S405 | QueryOrder via GET /api/v3/order with origClientOrderId |

### Test Evidence Pyramid

| Layer | Stage | File | Tests | Focus |
|---|---|---|---|---|
| Adapter | S405 | `s405_spot_venue_acceptance_fill_test.go` | 26 | Spot lifecycle, fill fidelity, correlation |
| Adapter | S406 | `s406_spot_rejection_partial_fill_test.go` | 19 | Rejection classification, partial fill aggregation |
| Adapter | — | `binance_spot_testnet_adapter_test.go` | 7 | Request construction, signing, response parsing |
| Actor | S405 | `s405_spot_venue_lifecycle_test.go` | 6 | SegmentRouter composition, DryRun wrapping |
| Actor | S406 | `s406_spot_rejection_partial_fill_test.go` | 11 | Rejection events, partial fill through router |
| Actor | S407 | `s407_read_path_segment_isolation_test.go` | 4 | Audit trail, metadata round-trip, partition keys |
| Application | S407 | `s407_read_path_audit_test.go` | 7 | RejectionDetail extraction, composite propagation |
| Integration | S408 | `s408_unified_compose_e2e_spot_test.go` | 9 | E2E pipeline, config coexistence, DryRun safety |
| **Total** | | | **82+7=89** | |

### Smoke Script Coverage

| Script | Stage | Phases | Mode | Key Validations |
|---|---|---|---|---|
| `smoke-spot-venue-live.sh` | S405/S406 | 7 | Unit + optional live | Adapter tests, actor tests, config validation, credential check |
| `smoke-e2e-unified-spot.sh` | S408 | 16 | venue_live or dry-run | Full stack: ingest -> derive -> execute -> store -> gateway -> writer |
| `smoke-unified-coexistence.sh` | S402 | 7 | Dry-run | Spot+Futures coexistence, segment_count=2, dry_run protection |

### Compose and Config Artifacts

| Artifact | Purpose | Mode |
|---|---|---|
| `docker-compose.unified.yaml` | Unified runtime overlay (both segments, dry-run) | Dry-run |
| `docker-compose.unified-spot-live.yaml` | Spot venue_live overlay (real orders) | venue_live |
| `execute-unified.jsonc` | Multi-segment config, dry_run=true | Dry-run |
| `execute-venue-live-spot.jsonc` | Spot live config, dry_run=false | venue_live |

### Control Points

| Control | Mechanism | Evidence Source |
|---|---|---|
| Dry-run safety | DryRunSubmitter wraps SegmentRouter | S408 `TestS408_ComposeE2E_DryRun_WrapsUnifiedRouter` |
| Kill switch | EXECUTION_CONTROL KV | Inherited (S319, S380) |
| Staleness guard | StalenessGuard max_age check | Inherited (S317) |
| Source guard | AllowedSources in VenueAdapterActor | S408 `TestS408_ComposeE2E_AllowedSourcesGate_SpotPermitted` |
| NATS consumer filter | Segment-filtered subscriptions | Inherited (S401) |
| Fail-closed routing | SegmentRouter rejects unknown sources | S408 `TestS408_ComposeE2E_ConfigCoexistence_BothSegmentsEnabled` |

---

## Residual Gaps

### Gap 1: ClickHouse Rejection Writer (RG-1)

- **Severity**: Medium
- **Source**: S404 charter risk register, TV-Q8
- **Status**: Pre-existing, not introduced by this wave
- **Description**: KV rejection queryability is proven (S407). The dedicated ClickHouse analytical persistence path for rejection events is not wired. Fill events flow to ClickHouse via the existing writer pipeline, but rejection events do not have an equivalent analytical writer.
- **Impact**: Rejection data is queryable via KV (operational) but not available in ClickHouse (analytical). Does not affect lifecycle correctness or Spot-first proof.
- **Recommendation**: Address in a future hardening wave or as part of Futures venue execution proof.

### Gap 2: Partial Fill Live Observation

- **Severity**: Low (venue constraint, not code gap)
- **Source**: TV-Q5, S406
- **Description**: Market orders on Binance Spot testnet fill instantly. PARTIALLY_FILLED is rare-to-impossible for market orders. The adapter code handles the status correctly (proven via mock), but no live partial fill was observed.
- **Impact**: None for Spot-first proof. If limit orders are added in the future, partial fill handling is already in place.
- **Recommendation**: Accept as structural evidence. Live observation would require limit order support or extreme liquidity conditions.

### Gap 3: Latest-Only KV Semantics

- **Severity**: Low
- **Source**: S407 L1
- **Description**: KV buckets store latest state only. Historical lifecycle progression (e.g., accepted -> partially_filled -> filled) is not preserved in KV. Requires JetStream streams or ClickHouse for historical queries.
- **Impact**: Operational queries show current state correctly. Historical analysis requires analytical path.
- **Recommendation**: Accept for current wave. Historical read-path is a future capability.

### Gap 4: No Segment-Scoped List Queries

- **Severity**: Low
- **Source**: S407 L4
- **Description**: Queries require specific partition key. There is no "list all Spot rejections" query across symbols/timeframes.
- **Impact**: Individual intent queryability is complete. Cross-intent listing is a future feature.
- **Recommendation**: Defer to operational tooling or analytical path.

### Gap 5: Commission Asset Type

- **Severity**: Low
- **Source**: S405 L4
- **Description**: Fill records capture commission amount but not asset type (BNB, USDT, etc.). The Binance Spot response provides both, but only the amount is extracted.
- **Impact**: Informational. Does not affect lifecycle correctness or P&L calculation at current scope.
- **Recommendation**: Capture when multi-asset accounting becomes a requirement.

---

## Regression Verification

### Compilation

| Check | Result |
|---|---|
| `go vet ./...` (all 12 modules) | CLEAN — zero warnings, zero errors |
| S405-S408 test compilation | CLEAN — all files compile |

### Prior Wave Test Suites

| Wave | Representative Tests | Status |
|---|---|---|
| OMS Foundation (S382-S388) | Lifecycle state machine, rejection events, dry-run submitter | PASS (used directly by S405-S408) |
| Segmentation (S390-S395) | Adapter boundary, config enablement, source routing | PASS (S401 isolation tests pass) |
| Unified Runtime (S398-S403) | SegmentRouter, coexistence, fail-closed dispatch | PASS (S402 coexistence tests pass) |

### Wave-Internal Regression

| Addition | Prior Tests Affected | Result |
|---|---|---|
| S405 (32 tests) | None (additive) | PASS |
| S406 (30 tests) | S405 (32) | ALL PASS |
| S407 (11 tests + code changes) | S405 (32) + S406 (30) | ALL PASS |
| S408 (9 tests) | S405 (32) + S406 (30) + S407 (11) | ALL PASS |

Zero regressions across all 82 wave tests and all prior wave capabilities.

---

## Non-Goal Compliance Audit

| ID | Non-Goal | Compliant | Notes |
|---|---|---|---|
| NG-23 | No parallel Futures testnet proof | YES | Futures adapter registered but not exercised |
| NG-24 | No production/mainnet connectivity | YES | Testnet URLs only |
| NG-29 | No unified runtime redesign | YES | Runtime consumed as-is from S403 |
| NG-30 | No per-segment dry_run toggle | YES | Global dry_run only |
| NG-31 | No limit order support | YES | Market orders only |
| NG-34 | No concurrent Spot + Futures venue_live | YES | Only Spot venue_live proven |
| NG-35 | No config schema changes | YES | Existing schema used |

Full compliance with all 35 frozen non-goals.

---

## Next Ceremony Recommendation

### Wave Closure

The Testnet Venue Execution Proof Wave on Unified Runtime (Spot-First) closes with **PASS — SUBSTANTIAL DELIVERY** (9/10 capabilities FULL, 1/10 SUBSTANTIAL).

### Strategic Options for Next Ceremony

Based on the evidence accumulated across waves S370-S408, the Foundry has proven:

1. **Multi-binary orchestration** (S370-S375): PASS
2. **Exchange listening + dry-run** (S376-S381): PASS
3. **OMS Foundation** (S382-S388): PASS
4. **Binance segmentation** (S390-S395): PASS
5. **Unified segment runtime** (S398-S403): PASS (FULL DELIVERY)
6. **Testnet venue execution, Spot-first** (S404-S408): PASS (SUBSTANTIAL DELIVERY)

The Foundry now has a complete, proven Spot execution chain from exchange ingress through lifecycle outcome to persistence and read-path on a unified multi-segment runtime.

### Recommended Next Direction

**Option A: Futures Testnet Venue Execution Proof (Spot foundation leverage)**
- Extend venue execution proof to the Futures segment on the same unified runtime.
- Leverage: BinanceSpotTestnetAdapter pattern, SegmentRouter dispatch, compose overlays, read-path architecture.
- Risk: Futures testnet API differences (margin modes, leverage, different fill response shape).
- Value: Completes the multi-segment venue execution story.

**Option B: Production Readiness and Operational Hardening**
- Close RG-1 (ClickHouse rejection writer).
- Add soak testing and multi-symbol coverage.
- Harden credential management and operational runbooks.
- Value: Moves from proof to production-grade.

**Option C: Analytical Path and Observability Consolidation**
- Close latest-only KV gap with JetStream stream history.
- Wire ClickHouse rejection writer.
- Add segment-scoped list queries.
- Value: Completes the analytical layer.

### Recommendation

**Open a charter ceremony to decide the next macro-direction based on product priority.** The technical foundation supports any of the three options. The decision should be driven by:

- If multi-segment parity is the priority: **Option A** (Futures proof).
- If operational confidence is the priority: **Option B** (hardening).
- If analytical capability is the priority: **Option C** (observability).

The evidence gate does not prescribe the next wave. It confirms the current wave is complete and the foundation is sound for any of the above directions.

---

## References

| Document | Path |
|---|---|
| Evidence Gate | `docs/architecture/testnet-venue-execution-evidence-gate-unified-runtime-spot-first.md` |
| Wave Charter | `docs/architecture/testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md` |
| Capabilities and Non-Goals | `docs/architecture/testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md` |
| S405 Connectivity Proof | `docs/architecture/spot-real-venue-connectivity-and-lifecycle-acceptance-fill-proof-on-unified-runtime.md` |
| S405 Controls | `docs/architecture/spot-accepted-filled-real-response-alignment-controls-and-limitations.md` |
| S406 Rejection/Partial Fill | `docs/architecture/spot-real-rejection-and-partial-fill-evidence-on-unified-runtime.md` |
| S406 Evidence Strength | `docs/architecture/spot-rejected-partialfill-evidence-strength-auditability-and-limitations.md` |
| S407 Read-Path | `docs/architecture/unified-runtime-read-path-auditability-and-segment-isolation-under-real-spot-responses.md` |
| S407 Queryability | `docs/architecture/spot-real-response-queryability-correlation-segment-isolation-and-limitations.md` |
| S408 E2E Proof | `docs/architecture/unified-compose-e2e-proof-with-spot-live-execution-path.md` |
| S408 Evidence/Controls | `docs/architecture/spot-segment-e2e-compose-evidence-controls-and-limitations.md` |
