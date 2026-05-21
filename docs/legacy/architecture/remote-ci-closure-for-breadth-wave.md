# Remote CI Closure for Breadth Wave

**Date:** 2026-03-21
**Charter:** BREADTH-WAVE-1 (S240–S244)
**Stage:** S245
**Purpose:** Close debt D3 — provide remote CI evidence for the accumulated breadth wave

---

## 1. Context

The BREADTH-WAVE-1 charter (S240–S244) delivered three new evaluator/resolver types across Decision, Strategy, and Risk domains. The charter gate (S244) passed all 9 exit criteria but left D3 as high-severity debt: remote CI verification of the accumulated changes.

This document records the closure of D3 through real remote CI execution.

## 2. What Was Validated Remotely

### 2.1 Commits Under Test

| Commit | Description | Scope |
|--------|-------------|-------|
| `95c7cc2` | Breadth wave S241–S244 delivery | 113 files: 3 new families, full pipeline wiring, tests, golden snapshots, docs |
| `516236d` | Migration fix (multi-statement → single ALTER) | 1 file: deploy/migrations/007 |

### 2.2 CI Pipeline Jobs

| Job | Duration | Result | What It Proves |
|-----|----------|--------|----------------|
| Unit Tests | 1m31s | PASS | All domain, application, actor, adapter, and interface tests pass |
| Codegen Golden Equivalence | 30s | PASS | All 10 families (including 3 new) validate and match golden snapshots |
| Integration Tests | 1m34s | PASS | Actor chain integration with embedded NATS works for all types |
| Smoke Analytical E2E | 7m23s | PASS | Full compose stack boots, migrations apply, seed runs, HTTP endpoints respond |

### 2.3 CI Run Reference

- **Run ID:** 23375533952
- **URL:** https://github.com/FabioCaffarello/market-foundry/actions/runs/23375533952
- **Branch:** main
- **Trigger:** push
- **Final commit:** `516236d`

## 3. Defect Found and Fixed

The first CI attempt (run `23375415266`) exposed a real defect invisible to local testing:

- **Problem:** Migration `007_add_decision_severity_rationale.sql` contained two separate `ALTER TABLE` statements. ClickHouse rejects multi-statement queries in a single migration file.
- **Root cause:** Local ClickHouse testing either didn't run migrations or used a different execution path.
- **Fix:** Combined the two `ADD COLUMN` statements into a single comma-separated `ALTER TABLE` statement.
- **Commit:** `516236d`

This validates the project's stance that remote CI catches defects invisible to local execution.

## 4. D3 Debt Status

| Before S245 | After S245 |
|-------------|------------|
| D3: Remote CI verification — HIGH severity, unresolved | **CLOSED** — full pipeline green on run 23375533952 |

## 5. Scope Boundaries

- S245 did NOT expand test coverage beyond what S241–S244 delivered.
- S245 did NOT modify CI pipeline design.
- S245 fixed exactly one defect (migration syntax) discovered by the remote run.
- The smoke E2E covers the rsi_oversold chain; breadth-specific smoke coverage (ema_crossover, trend_following_entry, drawdown_limit) remains D1 debt for future hardening.
