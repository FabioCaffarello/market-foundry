# Stage S135: Triggered Refactors After Timeframe Expansion — Report

> **Stage:** S135
> **Type:** Targeted Refactoring
> **Wave:** TC-01 (Timeframe Coverage)
> **Predecessor:** S134 (Timeframe-Driven Friction Capture)
> **Status:** Complete

---

## 1. Executive Summary

S135 closes the TC-01 wave by executing the two P1 items identified in S134's friction capture. Both changes are small, localized, and directly triggered by evidence from the 2→4 timeframe expansion.

**Result:** Two targeted enhancements applied. Zero horizontal refactoring. The monorepo gains startup-time validation of timeframe configuration and documented crash recovery expectations — the only operational fragilities that the timeframe expansion actually exposed.

TC-01 is now complete: stages S131–S135 form a closed loop from definition through validation to friction capture and resolution.

---

## 2. Refactors Executed

### R-01: Timeframe Config Validation (F-02 → P1)

| Dimension | Detail |
|-----------|--------|
| **Trigger** | F-02: No timeframe validation in config |
| **S134 classification** | Operational Fragility |
| **Change** | Added `ValidateTimeframes()` to `PipelineConfig` |
| **Rules enforced** | Reject duplicates, reject < 10s, reject > 86400s |
| **Integration** | Called inside `ValidatePipeline()` → `Validate()` → startup |
| **Tests added** | 5 test cases (valid range, below-min, above-max, duplicates, empty) |
| **Lines changed** | ~25 lines production code, ~35 lines test code |

**Why this was triggered:** The expansion to 4 timeframes made configuration the primary activation surface. A typo (`[60, 60, 300, 3600]`) would silently spawn duplicate actors. This class of misconfiguration is now caught at startup with a clear validation error.

### R-02: Post-Crash Recovery Runbook (F-17 → P1)

| Dimension | Detail |
|-----------|--------|
| **Trigger** | F-17: No runbook for post-crash recovery at high TFs |
| **S134 classification** | Operational Fragility |
| **Change** | Added §9 to `timeframe-coverage-01-validation-procedure.md` |
| **Content** | Data loss table per TF, post-restart behavior, operator actions, design rationale |
| **Code changes** | Zero |

**Why this was triggered:** The 3600s timeframe introduced a crash scenario (F-13) where up to 60 minutes of accumulated state can be lost. Without documented expectations, an operator would not know whether the post-restart behavior is correct or broken.

---

## 3. Files Changed

### 3.1 Go Source Code (2 files)

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added `ValidateTimeframes()` method; integrated into `ValidatePipeline()` |
| `internal/shared/settings/settings_test.go` | Added 5 test cases for timeframe validation |

### 3.2 Documentation (3 files, new)

| File | Purpose |
|------|---------|
| `docs/architecture/triggered-refactors-after-timeframe-coverage-01.md` | Detailed record of executed refactors with triggers |
| `docs/architecture/refactors-still-deferred-after-timeframe-coverage-01.md` | Deferred items with rationale, triggers for action, and TC-02 gate summary |
| `docs/stages/stage-s135-triggered-refactors-after-timeframe-expansion-report.md` | This report |

### 3.3 Documentation (1 file, modified)

| File | Change |
|------|--------|
| `docs/architecture/timeframe-coverage-01-validation-procedure.md` | Added §9: Post-Crash Recovery Expectations per Timeframe |

---

## 4. Structural Gains

| Gain | Source |
|------|--------|
| **Fail-fast on misconfigured timeframes** | R-01 — startup rejects duplicates, out-of-range values |
| **Quantified crash recovery expectations** | R-02 — data loss per TF is explicit and documented |
| **TC-02 gate criteria documented** | Deferred-refactors doc specifies hard gates for next expansion |
| **Friction capture → resolution traceability** | Every change traces to an S134 finding ID |

---

## 5. Items Still Deferred

7 friction items remain deferred with explicit rationale. Full details in `refactors-still-deferred-after-timeframe-coverage-01.md`.

| ID | Friction | Priority | Trigger for Action |
|----|----------|----------|--------------------|
| F-01 | Per-binding timeframes | P2 | TC-02 requires heterogeneous TF sets |
| F-04 | Single tracker for evidence | P2 | TC-02 adds 8+ TFs |
| F-05 | No per-TF idle detection | P2 | TC-02 adds 4h+ TFs |
| F-07 | No "list timeframes" endpoint | P3 | External consumers query the system |
| F-08 | Null response ambiguity | P3 | Non-expert consumers exposed |
| F-13+F-15 | Window state loss / no interim snapshots | P2 | **TC-02 hard gate** — must resolve before 4h+ TFs |
| F-19 | No aggregate gateway view | P3 | Symbol count > 5 or dashboard needed |

9 items (F-03, F-06, F-09–F-12, F-14, F-16, F-18) are permanently accepted as P4.

---

## 6. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No horizontal refactoring opened | PASS — 2 localized changes |
| No "while we're at it" improvements | PASS — scope limited to P1 items |
| No refactor justified by aesthetics | PASS — both address operational fragility |
| No new framework or pattern introduced | PASS — follows existing `ValidatePipeline()` pattern |
| Consciously deferred items documented | PASS — 7 items with triggers |
| Permanently accepted items documented | PASS — 9 P4 items |

---

## 7. TC-01 Wave Completion

With S135 complete, the TC-01 wave is closed:

| Stage | Purpose | Status |
|-------|---------|--------|
| S131 | Strategic Timeframe Coverage Definition | Complete |
| S132 | Timeframe Matrix Minimal Expansion | Complete |
| S133 | End-to-End Timeframe Coverage Validation | Complete |
| S134 | Timeframe-Driven Friction Capture | Complete |
| S135 | Triggered Refactors After Timeframe Expansion | **Complete** |

**TC-01 outcome:** The pipeline operates correctly at 4 timeframes [60, 300, 900, 3600]. The architecture absorbs temporal expansion as a configuration concern. Two operational fragilities were closed. Seven structural debts are tracked with explicit triggers. The system is ready for normal operation at TC-01 scope.

---

## 8. Preparation for S136

### 8.1 Recommended Next Directions

TC-01 is complete. The next stage should be chosen based on the highest-value work available, not by continuing temporal expansion. Options:

| Direction | Prerequisites | Value |
|-----------|--------------|-------|
| **TC-02 planning** | Evaluate F-13 (state persistence) cost | Extends temporal coverage to 4h/daily |
| **Symbol expansion** | None — architecture already supports N symbols | Broadens market coverage |
| **Other product wave** | Depends on wave definition | New capabilities |

### 8.2 TC-02 Entry Conditions (if chosen)

Before TC-02 execution:
1. **Resolve F-13/F-15 decision:** Accept 4h state loss risk OR implement WAL/snapshots
2. **Implement F-04 + F-05:** Per-timeframe tracker split + timeframe-aware idle detection
3. **Evaluate F-01:** Per-binding timeframes needed only if heterogeneous TF sets required

### 8.3 What S135 Does NOT Change

- The derive runtime behavior is unchanged (no new code paths at runtime)
- The query surface is unchanged
- The config format is unchanged (validation is additive, not breaking)
- All existing tests pass without modification
