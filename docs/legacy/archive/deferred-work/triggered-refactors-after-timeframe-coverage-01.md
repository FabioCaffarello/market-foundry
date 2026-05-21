# Triggered Refactors After TC-01 — Timeframe Coverage

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S135 (Triggered Refactors)
> **Date:** 2026-03-19
> **Source:** S134 Friction Capture (Prioritized Matrix)

---

## 1. Purpose

This document records every refactor executed in S135, each with an explicit trigger from the S134 friction capture. No refactor was performed without a concrete trigger from the timeframe expansion evidence.

---

## 2. Executed Refactors

### R-01: Timeframe Config Validation (`ValidateTimeframes`)

**Trigger:** F-02 (Operational Fragility — no timeframe validation in config)
**Priority:** P1 (fix before TC-02 planning)
**Classification:** Targeted enhancement, not architectural refactor

**Problem:**
`PipelineConfig.TimeframeDurations()` silently dropped values ≤ 0 and fell back to `[60s]` if empty. No validation existed for:
- Duplicate timeframes (e.g., `[60, 60, 300]`)
- Values below 10s (meaningless for candle accumulation)
- Values above 86400s (beyond daily, no operational support yet)

A typo in `derive.jsonc` could silently spawn actors for meaningless timeframes, producing correct but useless candles.

**Change:**
Added `ValidateTimeframes()` method to `PipelineConfig` in `schema.go`:
- Rejects duplicate timeframe values
- Rejects timeframes < 10s (minimum meaningful candle window)
- Rejects timeframes > 86400s (maximum supported in TC-01/TC-02 scope)
- Integrated into `ValidatePipeline()` call chain, which is already called at startup via `AppConfig.Validate()`

**Files changed:**
| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added `ValidateTimeframes()` method (~25 lines); integrated into `ValidatePipeline()` |
| `internal/shared/settings/settings_test.go` | Added 5 test cases covering valid range, below-minimum, above-maximum, duplicates, and empty list |

**Effort:** Small (~30 lines code + tests)

**Structural gain:**
- Prevents silent misconfiguration at startup — same fail-fast pattern used for family validation
- The 86400s upper bound is a deliberate scope guard: values beyond daily require explicit TC-02+ evaluation
- The 10s lower bound prevents accidental sub-second or trivially small candles that would overwhelm KV writes

---

### R-02: Post-Crash Recovery Expectations per Timeframe

**Trigger:** F-17 (Operational Fragility — no runbook for post-crash recovery at high TFs)
**Priority:** P1 (fix before TC-02 planning)
**Classification:** Documentation enhancement, zero code change

**Problem:**
No documented procedure existed for "derive crashed at minute 45 of a 1h candle — what do I do?" An operator facing this scenario had no guidance on expected data loss or recovery timeline.

**Change:**
Added Section 9 ("Post-Crash Recovery Expectations") to the TC-01 Validation Procedure document. The section documents:
- Data loss expectations per timeframe on crash
- Recovery timeline per timeframe
- What operators should expect after a restart
- Why this is inherent to the current design (no WAL/snapshots)

**Files changed:**
| File | Change |
|------|--------|
| `docs/architecture/timeframe-coverage-01-validation-procedure.md` | Added §9: Post-Crash Recovery Expectations per Timeframe |

**Effort:** Small (documentation only)

**Structural gain:**
- Operators now have explicit guidance for crash recovery scenarios
- Data loss expectations are quantified per timeframe, reducing ambiguity
- The section explicitly documents that F-13 (state persistence) is a TC-02 prerequisite for 4h+ timeframes

---

## 3. What Was NOT Refactored (and Why)

See companion document: `refactors-still-deferred-after-timeframe-coverage-01.md`

---

## 4. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| Every refactor has an explicit S134 trigger | PASS — R-01 triggered by F-02, R-02 triggered by F-17 |
| No horizontal refactoring opened | PASS — changes are localized to 2 files + 1 doc |
| No new framework or pattern introduced | PASS — follows existing `ValidatePipeline()` pattern |
| No refactor justified by aesthetics | PASS — both address operational fragility |
| Changes are small and high-value | PASS — ~30 lines code, ~40 lines doc |
