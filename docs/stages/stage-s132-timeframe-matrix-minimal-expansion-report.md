# Stage S132: Timeframe Matrix Minimal Expansion — Report

> **Stage:** S132
> **Type:** Implementation (Minimal Expansion)
> **Wave:** TC-01 (Timeframe Coverage)
> **Predecessor:** S131 (Strategic Timeframe Coverage Definition)
> **Status:** Complete

---

## 1. Executive Summary

Stage S132 implements the TC-01 timeframe matrix defined in S131: expanding from 2 timeframes (60s, 300s) to 4 timeframes (60s, 300s, 900s, 3600s).

**The entire implementation is a single config change.** Zero Go source files were modified. This confirms the core S15/S131 thesis: the architecture treats timeframe as a first-class config-driven dimension that scales without code changes.

Supporting changes were limited to aligning smoke tests, HTTP test files, and live pipeline validation scripts to cover the new timeframes.

---

## 2. Timeframe Matrix Implemented

| Timeframe | Seconds | Status | Role |
|-----------|---------|--------|------|
| 1-minute | 60 | Existing (unchanged) | Regression baseline |
| 5-minute | 300 | Existing (unchanged) | Regression baseline |
| **15-minute** | **900** | **New** | Validates 3× step from 5m; standard intraday |
| **1-hour** | **3600** | **New** | Validates order-of-magnitude jump; tests long accumulation |

---

## 3. Files Changed

### 3.1 Configuration (1 file)

| File | Change |
|------|--------|
| `deploy/configs/derive.jsonc` | `"timeframes": [60, 300]` → `"timeframes": [60, 300, 900, 3600]` |

### 3.2 Smoke Tests and Scripts (3 files)

| File | Change |
|------|--------|
| `scripts/smoke-first-slice.sh` | Added Steps 6b/6c: 900s and 3600s candle endpoint validation; updated summary |
| `scripts/smoke-multi-symbol.sh` | Default `TIMEFRAMES` expanded from `60 300` to `60 300 900 3600`; header KV key counts updated from 4 to 8 per domain |
| `scripts/live-pipeline-activate.sh` | Gateway query surface evidence validation loops over all 4 timeframes per symbol |

### 3.3 HTTP Test Files (5 files)

| File | Queries Added |
|------|--------------|
| `tests/http/evidence.http` | 900s/3600s candle (btcusdt, ethusdt), 900s/3600s trade burst |
| `tests/http/signal.http` | 900s/3600s RSI signal |
| `tests/http/decision.http` | 900s/3600s RSI Oversold decision |
| `tests/http/strategy.http` | 900s/3600s mean_reversion_entry |
| `tests/http/risk.http` | 900s/3600s position_exposure |

### 3.4 Documentation (3 files, new)

| File | Purpose |
|------|---------|
| `docs/architecture/timeframe-coverage-01-implementation-notes.md` | Implementation details, simplifications, limits |
| `docs/architecture/timeframe-coverage-01-runtime-activation-and-query-surface.md` | Actor hierarchy, NATS subjects, KV keys, query surface, diagnostics |
| `docs/stages/stage-s132-timeframe-matrix-minimal-expansion-report.md` | This report |

### 3.5 Go Source Code (0 files)

**No Go files were modified.** This is the central validation of TC-01.

---

## 4. Simplifications Adopted

| # | Simplification | Rationale |
|---|---------------|-----------|
| S1 | Global timeframe list (all sources share `[60, 300, 900, 3600]`) | Per-binding overrides not triggered; accepted limit from S131 (L1) |
| S2 | Smoke tests validate endpoint reachability, not candle data for 900s/3600s | Extended runtime required for finalization; 200 + valid structure is sufficient activation proof |
| S3 | Live pipeline validation checks evidence at all 4 TFs, downstream at 60s only | Downstream domains need long warmup at higher TFs; evidence reachability proves wiring |
| S4 | No separate TC-01 smoke script | Existing scripts expanded in-place; minimal change principle |
| S5 | RSI warmup documented as comments, not enforced in tests | 900s RSI needs 225min, 3600s needs 15h; diagnostic criteria D3/D5 |

---

## 5. Triggers Observed

### 5.1 No Refactor Triggers Activated

TC-01 at 4 timeframes creates no evidence-based trigger for any refactoring:

| Potential Trigger | Status | Evidence |
|-------------------|--------|----------|
| Fan-out refactoring (CF-08 threshold: ~10 TF) | NOT TRIGGERED | 4 TF is well within safe range |
| Generic sampler (CF-08 threshold: N=3 signal families) | NOT TRIGGERED | TC-01 adds no families |
| Map-based registry (CF-11) | NOT TRIGGERED | No new family types |
| Per-binding timeframe overrides | NOT TRIGGERED | Single source sufficient |

### 5.2 Natural Signals Observed

| Signal | Observation | Classification |
|--------|------------|----------------|
| Smoke test duration pressure | Higher TFs need extended runtime for full validation; existing wait times sufficient for activation proof | Operational awareness, not a trigger |
| Signal warmup at high TFs | 3600s RSI needs 15h warmup for first meaningful signal; this is by design, not a gap | Known limit (L5) |
| KV key cardinality doubling | 2× growth is exactly as predicted; linear and bounded | Expected behavior |
| Test ergonomics | HTTP test files now have 900s/3600s queries but manual execution requires patience | Minor, not a trigger |

### 5.3 Diagnostic Ergonomics Note

The `smoke-multi-symbol.sh` script already supports `SMOKE_TIMEFRAMES` as an environment variable. This means validation can be scoped:

```bash
# Quick regression (existing TFs only)
SMOKE_TIMEFRAMES="60 300" ./scripts/smoke-multi-symbol.sh

# Full TC-01 validation (all 4 TFs)
./scripts/smoke-multi-symbol.sh
```

This emerged naturally from the existing design — no new mechanism needed.

---

## 6. Acceptance Criteria Verification

### 6.1 S132-Specific Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| New matrix implemented in minimal scope | PASS | 1 config change, 0 Go changes, 8 support file updates |
| Activation/wiring/query surfaces coherent | PASS | Config propagation path verified end-to-end in docs; all surfaces parameterized by timeframe |
| Existing architecture reused disciplinarily | PASS | Zero new abstractions, families, or domains |
| System ready for E2E temporal validation | PASS | Smoke tests, HTTP tests, and live pipeline scripts cover all 4 TFs |
| Simplifications and limits documented | PASS | 5 simplifications + 5 limits explicitly documented |

### 6.2 S131 Mandatory Criteria Readiness (M1–M13)

All 13 criteria from S131 are now **verifiable** but require live runtime for PASS/FAIL:

| # | Criterion | Verification Ready | Requires |
|---|-----------|-------------------|----------|
| M1 | Config activates 4 TFs without code change | YES | Deploy |
| M2 | Derive spawns correct actor count | YES | Startup log inspection |
| M3 | Evidence events for all 4 TFs | YES | NATS monitoring |
| M4 | KV entries for all 4 TFs | YES | KV bucket inspection |
| M5 | HTTP query for all 4 TFs | YES | Smoke test |
| M6 | NATS request/reply for all 4 TFs | YES | NATS request test |
| M7 | Signal pipeline for all 4 TFs | YES | Extended runtime (~225min for 900s) |
| M8 | Full pipeline for all 4 TFs | YES | Extended runtime (~15h for 3600s) |
| M9 | 15-minute candle correct | YES | ~15 min runtime |
| M10 | 1-hour candle correct | YES | ~60 min runtime |
| M11 | 1m/5m regression | YES | Smoke test |
| M12 | No duplicate events | YES | Dedup key inspection |
| M13 | Historical candle store for all 4 TFs | YES | Extended runtime |

---

## 7. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No new families opened | PASS — zero family additions |
| Scope within S131 definition | PASS — exactly the matrix defined in S131 |
| No new abstractions without trigger | PASS — zero new abstractions |
| Boundaries not broken for acceleration | PASS — all changes within existing config/test boundaries |
| Refactor triggers documented | PASS — Section 5 explicitly documents all triggers as NOT ACTIVATED |

---

## 8. Preparation for S133

S133 should be **Operational Validation and Friction Capture**:

### 8.1 Recommended S133 Scope

1. **Deploy and run for ≥ 90 minutes** to validate M1–M6, M9–M11
2. **Walk through all 13 mandatory criteria** with live evidence
3. **Capture diagnostic signals** D1–D5
4. **Document any friction** found during validation
5. **Extended run (optional):** 15+ hours for full 3600s signal pipeline validation (M7, M8)

### 8.2 S133 Exit Condition

All 13 mandatory criteria PASS with live evidence. Any failure documented as architectural finding.

### 8.3 Post-S133 Direction

| Outcome | Next Step |
|---------|-----------|
| All 13 criteria PASS | TC-01 complete; assess TC-02 readiness (4h, daily) |
| Code change required somewhere | Document as architectural gap; reassess S15 claims |
| 1h candle data quality poor | Expected (D5 is diagnostic); document finding |
| Memory concern at 1h window | Investigate sampler lifecycle; document as finding |

---

## 9. Strategic Position

**What S132 proves:**
- Timeframe expansion from 2→4 is a pure config change
- The architecture designed in S10–S15 genuinely scales along the temporal dimension
- Test infrastructure, query surfaces, and operational tooling accommodate new timeframes with minimal alignment
- The system is ready for live E2E validation of temporal depth

**What S132 does NOT prove (deferred to S133):**
- Runtime correctness of 900s and 3600s candles
- Memory behavior under 1-hour accumulation windows
- Signal quality at lower temporal frequencies
- Full pipeline completion at all 4 timeframes

**TC-01 remains a consolidation wave.** Zero new capabilities were introduced. The existing capability was extended to cover more temporal ground.
