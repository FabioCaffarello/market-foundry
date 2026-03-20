# Stage S208 — Runtime, Config, and Operational Closure Report

## Executive Summary

S208 consolidates the operational layer of market-foundry: startup, config,
diagnostics, smoke tests, CI pipeline, and runbooks. The objective is not to
open new capabilities but to close gaps that would otherwise become blockers
during the next refactoring phase.

Four concrete fixes were applied. Two architecture documents record the
operational closure. All existing runbooks were reviewed and confirmed
accurate. The system's operational surface is now consistent, reproducible,
and documented.

---

## Operational Closure Applied

### Fix 1: `scripts/utils/lib.sh` — Writer in PIPELINE_SERVICES

**Problem:** The `PIPELINE_SERVICES` shared constant listed 5 services but
omitted `writer`, despite writer exposing `/statusz` and `/diagz`.

**Fix:** Added `writer` to the array.

**Impact:** Any future script using `PIPELINE_SERVICES` from `lib.sh` will
correctly include writer in diagnostic iterations.

### Fix 2: `scripts/smoke-analytical-e2e.sh` — Error Log Scanning

**Problem:** The analytical smoke test validated data paths and observability
but did not scan for error-level log entries. Both `diag-check.sh` and
`live-pipeline-activate.sh` already performed this scan, creating an
inconsistency — the CI-facing script had the weakest error visibility.

**Fix:** Added Phase 8 (Error Log Scan) before the summary. Scans compose
logs for `"level":"error"` entries and reports count + last 5 entries.

**Impact:** Error-level noise is now surfaced in every analytical E2E run,
including CI.

### Fix 3: `.github/workflows/ci.yml` — CI Error Log Scanning + Configurable Flush

**Problem:** CI captured logs only on failure. Error-level entries in passing
runs were invisible. The 120s writer flush wait was hardcoded.

**Fix:**
- Added an `always()` step that scans compose logs for error-level entries
  regardless of pass/fail.
- Made the flush wait configurable via `WRITER_FLUSH_WAIT` env var
  (default: 120).

**Impact:** CI runs now surface error noise even on green builds. Flush
timeout is adjustable without code changes.

### Fix 4: `Makefile` — Corrected `make up` Help Text

**Problem:** `make up` help text listed only the original 7 services,
omitting clickhouse, migrations, and writer.

**Fix:** Changed to "start the full stack (nats + clickhouse + migrations + all services)".

**Impact:** Developer-facing documentation now matches reality.

---

## Files Changed

| File                                   | Change                                    |
|----------------------------------------|-------------------------------------------|
| `scripts/utils/lib.sh`                 | Added `writer` to `PIPELINE_SERVICES`     |
| `scripts/smoke-analytical-e2e.sh`      | Added Phase 8: error log scan             |
| `.github/workflows/ci.yml`             | Added error log scan step; configurable flush wait |
| `Makefile`                             | Fixed `make up` help text                 |
| `docs/architecture/runtime-config-and-operational-closure.md` | New |
| `docs/architecture/operational-smoke-ci-and-runbook-closure.md` | New |

---

## Items Closed vs Limits Maintained

### Closed

| Item                                          | Evidence                           |
|-----------------------------------------------|------------------------------------|
| Writer included in pipeline service constants | `lib.sh` PIPELINE_SERVICES updated |
| Error log scanning in analytical smoke        | Phase 8 added                      |
| Error log scanning in CI                      | `always()` step added              |
| CI flush wait configurable                    | `WRITER_FLUSH_WAIT` env var        |
| Makefile accuracy                             | Help text corrected                |
| Startup validation completeness               | Reviewed; no gaps found            |
| Health endpoint coverage                      | All services expose `/readyz`      |
| Diagnostics coverage                          | All pipeline services expose `/statusz`+`/diagz` |
| Runbook accuracy                              | All 6 runbooks reviewed; all current |
| Recovery semantics                            | Documented and confirmed           |

### Limits Maintained (by design)

| Limit                                  | Rationale                              |
|----------------------------------------|----------------------------------------|
| No gateway `/statusz`/`/diagz`         | Stateless proxy; no trackers           |
| No cross-service config validation     | Would require config registry          |
| NATS readiness is TCP-only             | JetStream readiness check not worth startup complexity |
| No ClickHouse heartbeat in gateway     | 503 on analytical endpoints is sufficient signal |
| No multi-symbol analytical E2E in CI   | Single symbol proves the path          |
| No performance assertions              | No SLAs defined yet                    |
| No smoke-multi-symbol in CI            | Too slow for CI; available locally     |
| No external alerting integration       | Premature; no monitoring infrastructure |
| No distributed tracing                 | Deferred by S205 scope freeze          |

---

## Impact for Next Phase

The next phase (refactoring/architecture/documentation) can rely on:

1. **Reproducible startup** — All services validate config and fail fast.
   No silent misconfiguration can reach runtime.

2. **Observable health** — Every service exposes at least `/readyz`. Pipeline
   services expose `/statusz` and `/diagz` with phase classification.

3. **Automated CI gate** — Unit tests + codegen validation + analytical E2E
   with error log scanning run on every push/PR to main.

4. **Documented recovery** — Restart semantics, durable consumers, and
   supervisor backoff are documented and match actual behavior.

5. **Diagnostic toolchain** — `make diag`, `make live-check`, and
   `make smoke-analytical` provide layered operational snapshots.

The next phase does **not** need to:
- Fix broken startup validation (it works)
- Add missing health endpoints (they exist)
- Backfill diagnostic capabilities (they are sufficient)
- Repair CI coverage (it covers the critical path)

---

## Preparation Recommended for S209

If S209 begins the refactoring/architecture phase, the following preparation
is recommended:

1. **Snapshot baseline** — Run `make snapshot` before any refactoring to
   establish a code intelligence baseline for drift detection.

2. **Lock CI green** — Verify CI is green on main before starting
   structural changes. Run `make live` locally to confirm full stack health.

3. **Review S205 scope freeze** — The `stabilization-scope-freeze-and-must-finish-matrix.md`
   defines deferred items. Some may become relevant once refactoring begins.

4. **Identify refactoring boundaries** — Use `make arch-guard` and
   `make drift-detect` (raccoon-cli) to establish current architectural
   boundaries before moving code.

5. **Preserve operational invariants** — Any refactoring must preserve:
   - Config validation fail-fast behavior
   - Health endpoint contracts
   - NATS durable consumer semantics
   - Writer supervisor recovery model
   - Gateway graceful degradation (503 when ClickHouse unavailable)

---

## Stage Classification

- **Type:** Operational closure
- **Scope:** Config, startup, diagnostics, smoke, CI, runbooks
- **New capabilities:** None
- **Architectural changes:** None
- **Risk:** Minimal (fixes are additive, no behavior changes)
