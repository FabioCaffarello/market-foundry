# Unified Segment Runtime Foundation -- Evidence Gate

**Stage:** S403
**Wave:** Unified Segment Runtime Foundation (S398--S403)
**Date:** 2026-03-22
**Predecessor:** [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md)
**Companion:** [`unified-segment-runtime-evidence-matrix-residual-gaps-and-next-ceremony.md`](unified-segment-runtime-evidence-matrix-residual-gaps-and-next-ceremony.md)

---

## 1. Gate Purpose

This document evaluates whether the Unified Segment Runtime Foundation Wave
(S398--S402) delivered its chartered capabilities with sufficient evidence to
close the wave and unblock the next strategic ceremony.

The wave was opened to resolve four structural debts inherited from the
Binance Segmentation Foundation Wave (S390--S395):

| Debt | Problem | Resolution block |
|------|---------|-----------------|
| D1 | Sequential seed semantics -- Spot and Futures cannot seed concurrently | B2 (S400) |
| D2 | One config per binary -- doubled maintenance surface | B1 (S399) |
| D3 | Compose overlay per segment -- operator picks one, no dual | B4 (S402) |
| D4 | Distributed source routing -- leakage risk grows linearly | B3 (S401) |

---

## 2. Capability Evaluation

Each of the 10 chartered capabilities is classified using the scale defined
in [`unified-segment-runtime-capabilities-questions-and-non-goals.md`](unified-segment-runtime-capabilities-questions-and-non-goals.md):

| Classification | Definition |
|---|---|
| **FULL** | All evidence present, no exceptions, all invariant tests pass |
| **SUBSTANTIAL** | Primary evidence present, minor gaps that do not compromise safety |
| **PARTIAL** | Some evidence present but key questions remain open |
| **NONE** | No evidence or evidence contradicts the claim |

### C1: Unified Config Model -- FULL

**Claim:** Single config file expresses multiple market segments with per-segment type, source, and enablement.

**Evidence:**
- `internal/shared/settings/schema.go`: `Segments map[MarketSegment]*SegmentVenueConfig` replaces flat `venue.type` scalar.
- Helpers: `HasUnifiedSegments()`, `EnabledSegments()`, `IsSegmentEnabled()`, `AdapterForSegment()`.
- `deploy/configs/execute-unified.jsonc`: working config with both `spot` and `futures` segments enabled.
- 26 validation tests in `s393_segment_enablement_test.go`: all pass.
- 7 source-segment mapping tests in `s400_source_segment_test.go`: all pass.

**Gaps:** None.

### C2: Multi-Segment Validation -- FULL

**Claim:** Config validation rejects incomplete, contradictory, or disabled-only segment declarations at startup.

**Evidence:**
- `schema.go` validation rules: unknown segment keys, adapter/segment mismatch, enabled without adapter, paper as segment adapter, segments map with nothing enabled -- all rejected.
- 26 tests in `s393_segment_enablement_test.go` cover positive and negative validation paths.
- Fail-closed: zero enabled segments from a segments map triggers startup error.

**Gaps:** None.

### C3: Backward-Compatible Migration -- FULL

**Claim:** Legacy single-segment configs (with `venue.type`) boot correctly with deprecation warning.

**Evidence:**
- `cmd/execute/run.go`: `buildVenueAdapter()` dispatches to `buildVenueAdapterFromType()` when `HasUnifiedSegments()` is false.
- Legacy configs (`execute.jsonc`, `execute-futures.jsonc`, `execute-spot.jsonc`) all migrated to unified format but retain `venue.type` path.
- Validation tests confirm `Type`-only configs pass validation.
- Smoke restore phase (phase 7 in `smoke-unified-coexistence.sh`) boots with default paper config after unified test.

**Gaps:** None.

### C4: Merged Binding Seed -- FULL

**Claim:** Single seed activation produces NATS bindings for all enabled segments.

**Evidence:**
- `scripts/seed-configctl.sh`: `--merge` flag and `SOURCES` env var.
- Makefile targets: `make seed-unified`, `make seed-unified-multi`.
- Merged seed produces `binancef.*` and `binances.*` bindings in a single configctl activation.
- Seed is idempotent per source.

**Gaps:** None.

### C5: Multi-Adapter Runtime Projection -- FULL

**Claim:** Execute binary boots one adapter instance per enabled segment from a single config.

**Evidence:**
- `cmd/execute/run.go`: `buildVenueAdapterFromSegments()` iterates `EnabledSegments()`, builds one adapter per entry, registers into `SegmentRouter`.
- Each adapter instantiated independently with dedicated credentials.
- Startup logs show `type=multi_segment`, `segment_count=2`.
- 8 coexistence tests in `s402_unified_coexistence_test.go` confirm dual boot.
- 3 structural tests in `s400_multi_segment_test.go` confirm adapter distinctness.

**Gaps:** None.

### C6: Source-Based Intent Routing -- FULL

**Claim:** SegmentRouter dispatches intents to segment adapter matching `intent.Source`.

**Evidence:**
- `internal/application/execution/segment_router.go`: `SubmitOrder` parses `intent.Source` -> `SegmentForSource()` -> registered adapter.
- Source-segment mapping is bijective: `binancef` <-> `futures`, `binances` <-> `spot`.
- 8 routing tests in `s400_segment_router_test.go`: single-segment isolation, multi-segment dispatch, cross-segment rejection.
- Source-segment round-trip tested in `s400_source_segment_test.go`.

**Gaps:** None.

### C7: Fail-Closed Unknown Source Rejection -- FULL

**Claim:** Intents with source values not matching any enabled segment are rejected, not dropped.

**Evidence:**
- `SegmentRouter.SubmitOrder`: unknown source -> `SegmentForSource()` returns empty -> structured `Problem` returned.
- `VenueAdapterActor`: `AllowedSources` gate rejects unregistered sources with `rejected_source` counter.
- Test: `s400_segment_router_test.go` -- unknown source `kraken` triggers rejection.
- Test: `s402_unified_coexistence_test.go` -- cross-segment rejection for `kraken` source.
- Test: `s401_segment_isolation_test.go` -- unknown sources (`kraken`, `bybit`, empty) return empty segment.

**Gaps:** None.

### C8: Cross-Segment Leakage Prevention -- FULL

**Claim:** NATS consumer subject filtering and source validation prevent Spot/Futures data mixing.

**Evidence:**
- `internal/adapters/nats/natsexecution/registry.go`: `ExecuteVenueIntakeConsumerForSegments(sources)` constructs `FilterSubjects` scoped to enabled segments.
- `internal/actors/scopes/execute/venue_adapter_actor.go`: Gate 0 -- `AllowedSources` check before kill switch.
- Defense-in-depth model with 7 layers (L0--L6) documented in [`segment-safe-routing-and-leakage-hardening.md`](segment-safe-routing-and-leakage-hardening.md).
- 6 consumer filter tests in `s401_segment_consumer_test.go`.
- 7 isolation invariant tests in `s401_segment_isolation_test.go` proving bijection, injectivity, and partition completeness.

**Gaps:** None.

### C9: Single-Compose Coexistence -- FULL

**Claim:** Both segments run concurrently in one compose stack with one binary and one config.

**Evidence:**
- `deploy/compose/docker-compose.unified.yaml`: single compose overlay for dual-segment.
- `deploy/configs/execute-unified.jsonc`: both segments enabled.
- `scripts/smoke-unified-coexistence.sh`: 7-phase compose-level proof.
- 8 coexistence invariant tests in `s402_unified_coexistence_test.go`.
- S402 introduced zero production code changes -- pure validation of S399-S401 foundation.

**Gaps:** None.

### C10: Global dry_run Preservation -- FULL

**Claim:** `dry_run=true` wraps ALL segment adapters uniformly; no per-segment override.

**Evidence:**
- `cmd/execute/run.go`: `DryRunSubmitter` wraps the entire `SegmentRouter` as outermost decorator.
- `schema.go`: `dry_run` is top-level in `VenueConfig`, not per-segment. NG-7 freezes this permanently.
- Test: `s402_unified_coexistence_test.go/DryRunWrapsCoexistentRouterUniformly` proves both segments receive dry-run treatment.
- Config validation: no per-segment `dry_run` field exists.

**Gaps:** None.

---

## 3. Governing Question Disposition

| ID | Question | Answer | Classification |
|---|---|---|---|
| USR-Q1 | Can a single config express enablement for multiple segments? | Yes -- `execute-unified.jsonc` with both segments enabled; 26 validation tests pass | FULL |
| USR-Q2 | Does validation reject contradictory/incomplete multi-segment declarations? | Yes -- 26 tests cover missing source, unknown type, disabled-only, duplicate source | FULL |
| USR-Q3 | Does backward-compatible migration accept legacy configs? | Yes -- `venue.type` path preserved; smoke phase 7 validates restore | FULL |
| USR-Q4 | Can a single seed produce bindings for all enabled segments? | Yes -- `make seed-unified` with `--merge` flag produces both source bindings | FULL |
| USR-Q5 | Does execute boot multiple adapters from single config? | Yes -- `buildVenueAdapterFromSegments` iterates segments; logs show `segment_count=2` | FULL |
| USR-Q6 | Does intent routing dispatch to correct segment adapter by source? | Yes -- SegmentRouter dispatch tested for spot and futures isolation | FULL |
| USR-Q7 | Is an unknown/disabled source rejected fail-closed? | Yes -- structured Problem returned; `rejected_source` counter incremented | FULL |
| USR-Q8 | Can a Spot intent never reach Futures adapter? | Yes -- bijective mapping + consumer filtering + actor guard; 7 invariant tests | FULL |
| USR-Q9 | Does NATS consumer filtering prevent cross-segment delivery? | Yes -- `FilterSubjects` scoped per source; 6 consumer tests prove partitioning | FULL |
| USR-Q10 | Can both segments run concurrently in single compose? | Yes -- unified compose file; 7-phase smoke proof | FULL |
| USR-Q11 | Does unified runtime preserve global dry_run fail-closed? | Yes -- DryRunSubmitter wraps entire router; no per-segment override path | FULL |
| USR-Q12 | Are per-segment compose overrides still valid? | Yes -- legacy compose files preserved; backward-compatible boot path maintained | FULL |

**Result:** 12/12 questions answered at FULL.

---

## 4. Structural Debt Resolution

| Debt | Status | Evidence |
|------|--------|----------|
| D1: Sequential seed | **RESOLVED** | `--merge` flag in seed script; `make seed-unified` target |
| D2: One config per binary | **RESOLVED** | Unified segments map in single config file |
| D3: Compose overlay per segment | **RESOLVED** | `docker-compose.unified.yaml` with single binary |
| D4: Distributed source routing | **RESOLVED** | SegmentRouter + 7-layer defense-in-depth model |

---

## 5. Regression Audit

**Method:** Full `go test` execution across all workspace modules.

**Result:** All tests pass. Zero failures across:
- `internal/shared` (9 packages)
- `internal/application` (17 packages)
- `internal/domain` (8 packages)
- `internal/actors` (6 packages)
- `internal/adapters/nats` (9 packages)
- `internal/adapters/exchanges` (2 packages)
- `cmd/execute` (builds clean, no test files)

**Production code regressions:** None detected.

**Build verification:** All 7 modules touched by S398--S402 compile without errors.

---

## 6. Non-Goal Compliance

All 15 frozen non-goals (NG-1 through NG-15) remain respected. No stage
reopened any exclusion. Verified:

- NG-1, NG-2: No separate compose/config per segment created as permanent model.
- NG-3: No multi-exchange support added.
- NG-6: All work on testnet only.
- NG-7: No per-segment `dry_run` toggle introduced.
- NG-9: Ingest binary not unified (already source-aware from S397).
- NG-13: No platform-wide actor topology changes outside execute scope.
- NG-15: No real trading activation; dry-run throughout.

---

## 7. Verdict

### Classification Summary

| Capability | Classification |
|---|---|
| C1: Unified config model | **FULL** |
| C2: Multi-segment validation | **FULL** |
| C3: Backward-compatible migration | **FULL** |
| C4: Merged binding seed | **FULL** |
| C5: Multi-adapter runtime projection | **FULL** |
| C6: Source-based intent routing | **FULL** |
| C7: Fail-closed unknown source rejection | **FULL** |
| C8: Cross-segment leakage prevention | **FULL** |
| C9: Single-compose coexistence | **FULL** |
| C10: Global dry_run preservation | **FULL** |

**10/10 capabilities at FULL.**

### Wave Pass Threshold

The charter requires all 10 capabilities at FULL or SUBSTANTIAL with none
at PARTIAL or NONE.

**Result: PASS.**

### Formal Verdict

The Unified Segment Runtime Foundation Wave is **CLOSED WITH FULL
DELIVERY**. All 10 chartered capabilities are evidenced at FULL
classification. All 12 governing questions are answered at FULL. All 4
structural debts are resolved. Zero regressions detected. All 15 frozen
non-goals remain respected.

The Foundry runtime now supports Spot and Futures in the same binary, with
a single config, a single compose stack, source-based routing with
defense-in-depth, and global dry_run fail-closed preservation.

The project is ready to resume the Testnet Venue Execution Proof Wave on
the unified runtime foundation.

---

## 8. References

| Reference | Link |
|---|---|
| Wave charter | [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Capabilities, questions, non-goals | [`unified-segment-runtime-capabilities-questions-and-non-goals.md`](unified-segment-runtime-capabilities-questions-and-non-goals.md) |
| Evidence matrix and residual gaps | [`unified-segment-runtime-evidence-matrix-residual-gaps-and-next-ceremony.md`](unified-segment-runtime-evidence-matrix-residual-gaps-and-next-ceremony.md) |
| Config model architecture | [`unified-config-model-and-segment-enablement-refactor.md`](unified-config-model-and-segment-enablement-refactor.md) |
| Binding merge architecture | [`binding-merge-and-multi-segment-runtime-projection.md`](binding-merge-and-multi-segment-runtime-projection.md) |
| Leakage hardening architecture | [`segment-safe-routing-and-leakage-hardening.md`](segment-safe-routing-and-leakage-hardening.md) |
| Coexistence proof | [`single-compose-coexistence-proof-for-spot-and-futures.md`](single-compose-coexistence-proof-for-spot-and-futures.md) |
| S398 report | [`../stages/stage-s398-unified-segment-runtime-charter-report.md`](../stages/stage-s398-unified-segment-runtime-charter-report.md) |
| S399 report | [`../stages/stage-s399-unified-config-model-report.md`](../stages/stage-s399-unified-config-model-report.md) |
| S400 report | [`../stages/stage-s400-binding-merge-and-runtime-projection-report.md`](../stages/stage-s400-binding-merge-and-runtime-projection-report.md) |
| S401 report | [`../stages/stage-s401-segment-safe-routing-report.md`](../stages/stage-s401-segment-safe-routing-report.md) |
| S402 report | [`../stages/stage-s402-single-compose-coexistence-proof-report.md`](../stages/stage-s402-single-compose-coexistence-proof-report.md) |
