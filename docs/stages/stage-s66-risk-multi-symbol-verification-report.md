# Stage S66 — Risk Multi-Symbol Verification

| Field | Value |
|-------|-------|
| Stage | S66 |
| Title | Risk Multi-Symbol Verification |
| Status | Complete |
| Date | 2026-03-18 |

## 1. Executive Summary

Validated that the `risk` domain behaves correctly under a controlled multi-symbol scenario (2 symbols x 2 timeframes), proving activation coherence, partition key isolation, deduplication key uniqueness, projection independence, query surface isolation, and absence of cross-symbol bleed.

No new risk families were opened. No new domains were introduced. No structural workarounds were applied. All existing tests continue to pass alongside the new multi-symbol verification tests.

The `risk` domain is now proven multi-symbol-safe — a prerequisite for any serious `execution` readiness discussion.

---

## 2. Multi-Symbol Scenario Validated

| Dimension | Values |
|-----------|--------|
| Source | `binancef` |
| Symbols | `btcusdt`, `ethusdt` |
| Timeframes | `60`, `300` |
| Risk family | `position_exposure` |
| KV entries expected | 4 (2 symbols x 2 timeframes) |

### Verification Matrix

| Verification | Method | Result |
|-------------|--------|--------|
| Partition key isolation (3 symbols x 2 TFs) | Domain unit test | 6 unique keys, zero collisions |
| Deduplication key isolation (same timestamp) | Domain unit test | Unique per symbol at identical timestamp |
| No ownership bleed between assessments | Domain unit test | Independent field values, shared source/type |
| Independent materialization (2 sym x 2 TFs) | Projection actor test | 4 put calls, 4 materialized, 4 tracker events |
| Projection partition key no-bleed (3 sym x 2 TFs) | Projection actor test | 6 unique keys, zero collisions |
| Projection dedup key isolation | Projection actor test | Unique per symbol at same timestamp |
| Evaluator multi-symbol independence | Application test | 4 unique results, correct symbol/timeframe per result |
| Evaluator no-ownership-bleed | Application test | BTC long / ETH short maintain independent symbols, directions, partition keys, dedup keys |
| KV store key isolation (pre-existing) | NATS adapter test | 6 unique keys across 3 symbols x 2 TFs |
| E2E smoke: risk endpoint per symbol/TF | Smoke test (Steps 11-12) | Validates HTTP 200, structure, symbol field, disposition, constraints |
| E2E smoke: cross-symbol isolation | Smoke test (Step 12) | Detects COLLISION, BLEED_A, BLEED_B |
| E2E smoke: risk error handling | Smoke test (Step 13) | Unknown type -> 400, missing timeframe -> 400 |

---

## 3. Files Changed

### Tests (3 files, modified)

| File | Change |
|------|--------|
| `internal/domain/risk/risk_test.go` | +3 tests: `MultiSymbol_PartitionKeyIsolation`, `MultiSymbol_DeduplicationKeyIsolation`, `MultiSymbol_NoOwnershipBleed` |
| `internal/actors/scopes/store/risk_projection_actor_test.go` | +3 tests: `MultiSymbol_IndependentMaterialization`, `MultiSymbol_NoBleed_PartitionKeys`, `MultiSymbol_DeduplicationKeys` |
| `internal/application/risk/position_exposure_evaluator_test.go` | +2 tests: `MultiSymbol_IndependentEvaluation`, `MultiSymbol_NoOwnershipBleed` |

### Smoke Test (1 file, modified)

| File | Change |
|------|--------|
| `scripts/smoke-multi-symbol.sh` | +Steps 11-12: risk validation and cross-symbol isolation; +2 error handling checks in Step 13; updated header, summary |

### Documentation (1 file, new)

| File | Content |
|------|---------|
| `docs/stages/stage-s66-risk-multi-symbol-verification-report.md` | This report |

---

## 4. Test Results

| Package | Tests | Result |
|---------|-------|--------|
| `internal/domain/risk` | 15 (was 12) | PASS |
| `internal/application/risk` | 12 (was 10) | PASS |
| `internal/actors/scopes/store` (risk) | 14 (was 11) | PASS |
| `internal/adapters/nats` (risk) | 9 | PASS (pre-existing multi-symbol test confirmed) |
| `internal/application/riskclient` | 5 | PASS (unchanged) |
| `internal/interfaces/http/handlers` (risk) | 4 | PASS (unchanged) |
| `internal/interfaces/http/routes` (risk) | 3 | PASS (unchanged) |

Build: clean (`go build internal/...` passes).

---

## 5. Problems Found or Discarded

| Concern | Status | Detail |
|---------|--------|--------|
| Partition key collision | Discarded | Format `{source}.{symbol}.{timeframe}` produces unique keys for all tested combinations |
| Deduplication key collision at same timestamp | Discarded | Symbol is embedded in key: `risk:{type}:{source}:{symbol}:{tf}:{unix}` |
| Cross-symbol bleed via shared evaluator state | Discarded | Each evaluator is instantiated per symbol — no shared mutable state |
| Projection actor mixing symbols | Discarded | Actor is stateless per event; KV key is derived from event's own PartitionKey |
| KV bucket cross-contamination | Discarded | Single bucket, but keys are symbol-scoped; Get/Put always use partition key |
| Strategy input leaking between assessments | Discarded | StrategyInput is constructed fresh per evaluation; no shared references |

No structural problems were found. The multi-symbol isolation is sound at all layers.

---

## 6. Limitations Remaining

| Limitation | Impact | Path Forward |
|-----------|--------|-------------|
| Single risk family (position_exposure) | Multi-family projection not yet proven | Add second family (e.g., drawdown_guard) in future stage if needed |
| No concurrent actor stress test | Isolation proven structurally, not under load | Run sustained load test before execution readiness |
| Smoke test requires full pipeline warm-up | Risk results may be NULL if pipeline is young | Expected behavior; smoke handles gracefully |
| No history query surface | Cannot query past assessments by symbol | Stream retention (72h) available; history bucket deferred |

---

## 7. Impact on Readiness for S67/S68

### Strengthened

- **Execution readiness**: `risk` now has explicit multi-symbol proof. Any `execution` layer consuming risk assessments can trust symbol isolation.
- **Pipeline confidence**: The full chain (evidence -> signal -> decision -> strategy -> risk) is now validated multi-symbol end-to-end via smoke test.
- **Projection integrity**: Independent materialization and partition key isolation are proven — same patterns will apply to any future domain.

### Recommended Before Execution Layer

- [ ] Sustained load test: >1000 risk events/min across multiple symbols
- [ ] Graceful degradation: NATS interruption mid-stream with recovery validation
- [ ] Correlation-ID end-to-end tracing from strategy through risk
- [ ] Evaluate whether execution needs risk history or latest-only suffices

### Guard Rails Respected

- [x] No new risk families opened
- [x] No history projection opened
- [x] No `execution` layer implemented
- [x] No workarounds to mask structural failures
- [x] Limitations documented honestly
- [x] Existing tests unchanged and passing
