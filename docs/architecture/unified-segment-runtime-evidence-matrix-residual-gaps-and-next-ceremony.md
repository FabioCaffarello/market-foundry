# Unified Segment Runtime Foundation -- Evidence Matrix, Residual Gaps, and Next Ceremony

**Stage:** S403
**Wave:** Unified Segment Runtime Foundation (S398--S403)
**Date:** 2026-03-22
**Companion:** [`unified-segment-runtime-evidence-gate.md`](unified-segment-runtime-evidence-gate.md)

---

## 1. Evidence Matrix

### 1.1 Capability x Stage Matrix

| Capability | S399 | S400 | S401 | S402 | Classification |
|---|---|---|---|---|---|
| C1: Unified config model | 26 validation tests, schema types, config examples | Source-segment helpers added | -- | Coexistence test confirms | **FULL** |
| C2: Multi-segment validation | 26 tests: positive + negative paths, fail-closed | -- | -- | -- | **FULL** |
| C3: Backward-compatible migration | Type-based path preserved, validation passes | -- | -- | Smoke phase 7: restore to paper | **FULL** |
| C4: Merged binding seed | -- | `--merge` flag, `make seed-unified` target | -- | -- | **FULL** |
| C5: Multi-adapter runtime projection | -- | SegmentRouter, `buildVenueAdapterFromSegments`, 3 structural tests | -- | 8 coexistence tests | **FULL** |
| C6: Source-based intent routing | -- | 8 router tests, bijective mapping | -- | Coexistence dispatch test | **FULL** |
| C7: Fail-closed unknown source | -- | Router rejects unknown source with Problem | Actor gate rejects + counter | Cross-segment rejection test | **FULL** |
| C8: Cross-segment leakage prevention | -- | -- | 7 layers, 6 consumer tests, 7 invariant tests | Unified consumer test | **FULL** |
| C9: Single-compose coexistence | -- | Unified compose file | -- | 8 tests, 7-phase smoke | **FULL** |
| C10: Global dry_run preservation | NG-7 frozen; no per-segment field | DryRunSubmitter wraps router | -- | Uniform dry-run test | **FULL** |

### 1.2 Governing Question x Evidence Matrix

| Question | Evidence artifacts | Verdict |
|---|---|---|
| USR-Q1: Multi-segment config expression | `execute-unified.jsonc`, `s393_segment_enablement_test.go` (26 tests) | FULL |
| USR-Q2: Validation rejects contradictions | `s393_segment_enablement_test.go`: missing source, unknown type, disabled-only, duplicates | FULL |
| USR-Q3: Legacy config backward compat | `buildVenueAdapterFromType()` path, smoke phase 7 restore | FULL |
| USR-Q4: Merged seed activation | `seed-configctl.sh --merge`, `make seed-unified` | FULL |
| USR-Q5: Multi-adapter boot | `buildVenueAdapterFromSegments()`, `segment_count=2` in logs | FULL |
| USR-Q6: Intent dispatch by source | `s400_segment_router_test.go` (8 tests), round-trip in `s400_source_segment_test.go` | FULL |
| USR-Q7: Unknown source rejection | Router Problem, actor `rejected_source` counter, test coverage | FULL |
| USR-Q8: No cross-segment delivery | `s401_segment_isolation_test.go` (7 tests): bijection, injectivity, partition | FULL |
| USR-Q9: NATS consumer filtering | `ExecuteVenueIntakeConsumerForSegments`, `s401_segment_consumer_test.go` (6 tests) | FULL |
| USR-Q10: Concurrent compose coexistence | `docker-compose.unified.yaml`, `smoke-unified-coexistence.sh` (7 phases) | FULL |
| USR-Q11: Global dry_run invariant | `DryRunSubmitter` outermost decorator, `DryRunWrapsCoexistentRouterUniformly` test | FULL |
| USR-Q12: Per-segment overrides valid | Legacy compose files preserved, backward-compatible boot path | FULL |

### 1.3 Test Artifact Summary

| Test file | Count | Scope | Result |
|---|---|---|---|
| `s393_segment_enablement_test.go` | 26 | Config validation, segment helpers | Pass |
| `s394_segmented_compose_test.go` | 8 | Segmented compose config | Pass |
| `s400_segment_router_test.go` | 8 | Router dispatch, isolation, rejection | Pass |
| `s400_source_segment_test.go` | 7 | Source-segment mapping, round-trip | Pass |
| `s400_multi_segment_test.go` | 3 | Structural validation, adapter distinctness | Pass |
| `s401_segment_sources_test.go` | 6 | EnabledSegmentSources helpers | Pass |
| `s401_segment_consumer_test.go` | 6 | NATS consumer filter subjects | Pass |
| `s401_segment_isolation_test.go` | 7 | Isolation invariants (bijection, partition) | Pass |
| `s402_unified_coexistence_test.go` | 8 | Coexistence proof (all layers) | Pass |
| **Total** | **79** | | **All pass** |

### 1.4 Architecture Documents Produced

| Document | Stage | Scope |
|---|---|---|
| `unified-segment-runtime-wave-charter-and-scope-freeze.md` | S398 | Charter, 12 questions, 15 non-goals, risk registry |
| `unified-segment-runtime-capabilities-questions-and-non-goals.md` | S398 | 10 capabilities, classification targets |
| `unified-config-model-and-segment-enablement-refactor.md` | S399 | Schema design, helpers, validation rules |
| `binding-merge-and-multi-segment-runtime-projection.md` | S400 | Router design, seed merge, source mapping |
| `segment-safe-routing-and-leakage-hardening.md` | S401 | 7-layer defense model, leakage vectors |
| `single-compose-coexistence-proof-for-spot-and-futures.md` | S402 | Compose proof, runtime topology |
| `unified-runtime-compose-behavior-isolation-and-limitations.md` | S402 | Boot sequence, credential handling, isolation |

---

## 2. Residual Gaps

### 2.1 Acknowledged Limitations (Non-Blocking)

These limitations are documented and accepted. None compromise safety or
block the next wave.

| ID | Limitation | Severity | Status | Disposition |
|---|---|---|---|---|
| L1 | No per-segment `dry_run` toggle | Low | Frozen (NG-7) | Global safety invariant; no current need |
| L2 | No per-segment kill switch | Low | Deferred | Binary-wide kill switch sufficient for testnet |
| L3 | No per-segment staleness override | Low | Deferred | Binary-wide staleness check sufficient |
| L4 | Metrics not segmented | Low | Deferred | Aggregate counters across segments; per-segment observability is future enhancement |
| L5 | QueryOrder sequential across segments | Low | Accepted | Reconciliation is infrequent; sequential iteration acceptable |
| L6 | Source-segment mapping hardcoded for Binance | Low | Accepted (NG-3) | No second exchange exists; mapping extension is trivial when needed |
| L7 | Single consumer per binary | Low | Deferred | SegmentRouter + consumer filter sufficient; per-segment consumer isolation is optimization |

### 2.2 Prior Wave Gap Disposition

| Gap | Origin | Status |
|---|---|---|
| G1 (S395): Concurrent multi-instance compose | Segmentation wave | **Resolved** by S402: single compose, both segments |
| G2 (S395): Per-segment control gate | Segmentation wave | **Non-goal** (NG-7): global dry_run invariant |
| G3 (S395): Spot ingest not seeded | Segmentation wave | **Closed** by S397: spot ingest binding seed |
| G4 (S395): Activation surface segment query | Segmentation wave | **Partial**: health reports both segments; full query API is future |
| G5 (S395): Shared core extraction | Segmentation wave | **Non-goal** for this wave; preserved |
| G1 (S399): Multi-adapter routing | Config wave | **Resolved** by S400: SegmentRouter |
| G2 (S399): Per-segment dry_run override | Config wave | **Non-goal** (NG-7) |

### 2.3 Risk Registry Audit

| Risk (from charter) | Materialized? | Outcome |
|---|---|---|
| Config migration breaks workflows | No | Backward-compatible path preserved; legacy configs boot |
| Multi-adapter boot complexity | No | Fail-closed: segment failure stops binary |
| Source routing single point of failure | No | 7-layer defense; invariant tests prove correctness |
| Merged seed ordering dependency | No | Seed is idempotent per source |
| Scope creep into per-segment dry_run | No | NG-7 held firm throughout wave |

---

## 3. Regression Summary

### 3.1 Full Test Suite Execution

| Module | Packages tested | Result |
|---|---|---|
| `internal/shared` | 9 | All pass |
| `internal/application` | 17 | All pass |
| `internal/domain` | 8 | All pass |
| `internal/actors` | 6 | All pass |
| `internal/adapters/nats` | 9 | All pass |
| `internal/adapters/exchanges` | 2 | All pass |
| `cmd/execute` | 1 (build only) | Compiles clean |

**Regressions detected:** Zero.

### 3.2 Build Verification

All 7 workspace modules touched by S398--S402 compile without errors:
`cmd/execute`, `internal/shared`, `internal/application`, `internal/domain`,
`internal/actors`, `internal/adapters/nats`, `internal/adapters/exchanges`.

---

## 4. Wave Verdict

**PASS -- FULL DELIVERY.**

- 10/10 capabilities at FULL classification.
- 12/12 governing questions answered at FULL.
- 4/4 structural debts resolved.
- 0 regressions.
- 15/15 non-goals respected.
- 0 critical gaps.
- 7 acknowledged non-blocking limitations (all documented and accepted).

---

## 5. Next Ceremony Recommendation

### 5.1 Strategic Context

With the unified segment runtime foundation closed, the Foundry has:

1. A single execute binary supporting Spot and Futures concurrently.
2. A single config model with segment enablement and fail-closed validation.
3. Source-based routing with 7-layer defense-in-depth.
4. A single compose stack for dual-segment operation.
5. Global dry_run fail-closed preservation across all segments.

The Testnet Venue Execution Proof Wave (originally S389, refreshed in S396
as segmented Spot-first) was blocked pending unified runtime. That blocker
is now resolved.

### 5.2 Recommended Next Wave

**Testnet Venue Execution Proof Wave (S404+)**

Resume the venue execution proof on the unified runtime. The 12 testnet
venue questions (TV-Q1 through TV-Q12) from S396 remain valid and
unanswered. The wave should:

1. Prove Spot testnet execution end-to-end on the unified runtime.
2. Prove Futures testnet execution end-to-end on the unified runtime.
3. Validate fill capture, order lifecycle, and reconciliation paths.
4. Exercise the SegmentRouter under real testnet traffic (dry-run protected).

### 5.3 Preconditions Met for Next Wave

| Precondition | Status |
|---|---|
| Unified config model with segment enablement | Ready |
| Multi-adapter runtime projection | Ready |
| Source-based routing with leakage prevention | Ready |
| Single compose for dual-segment operation | Ready |
| Spot adapter (`binance_spot_testnet`) implemented | Ready (S392) |
| Futures adapter (`binance_futures_testnet`) implemented | Ready (S390) |
| Spot ingest binding seed | Ready (S397) |
| Global dry_run fail-closed | Ready |

### 5.4 What This Recommendation Does NOT Do

- Does not open the next wave (that requires a charter ceremony).
- Does not commit to a timeline.
- Does not change the scope of the venue execution wave.
- Does not reopen any frozen non-goal.

---

## 6. References

| Reference | Link |
|---|---|
| Evidence gate | [`unified-segment-runtime-evidence-gate.md`](unified-segment-runtime-evidence-gate.md) |
| Wave charter | [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Capabilities and non-goals | [`unified-segment-runtime-capabilities-questions-and-non-goals.md`](unified-segment-runtime-capabilities-questions-and-non-goals.md) |
| S395 evidence gate | [`binance-spot-futures-segmentation-evidence-gate.md`](binance-spot-futures-segmentation-evidence-gate.md) |
| S396 charter refresh | [`testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md`](testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md) |
| S403 stage report | [`../stages/stage-s403-unified-segment-runtime-evidence-gate-report.md`](../stages/stage-s403-unified-segment-runtime-evidence-gate-report.md) |
