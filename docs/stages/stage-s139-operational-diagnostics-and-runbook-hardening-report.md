# Stage S139 — Operational Diagnostics and Runbook Hardening

**Objective:** Strengthen operational diagnostics and runbooks for the current Foundry baseline, making it more observable, more legible, and easier to operate consistently.

**Directive:** Harden what exists. No new features. No heavyweight observability. No ClickHouse implementation.

## 1. Executive Summary

S139 reviewed the entire diagnostic surface of the Foundry baseline and applied targeted improvements to reduce operational ambiguity. The core change is adding **operational phase classification** (`starting`, `warming`, `active`, `idle`, `stalled`) to `/statusz` and `/diagz` endpoints, giving operators an immediate aggregate answer to "is this service healthy?" without parsing individual tracker states. Runtime metadata (`go_version`, `num_goroutines`) was added to `/diagz` for debugging. A lightweight `diag-check.sh` script was created for quick diagnostic snapshots. Comprehensive runbook documentation was written, covering lifecycle, diagnosis, and recovery procedures.

## 2. Hardening Applied

### 2.1 Operational Phase Classification (Code Change)

**Problem:** `/statusz` exposed per-tracker event counts and idle warnings, but the operator had to mentally aggregate these into an operational state. There was no single field answering "is this service working normally?"

**Solution:** Added `phase` field to `/statusz` and `/diagz` responses with five states:

| Phase      | Condition                                           |
|------------|-----------------------------------------------------|
| `starting` | Uptime < 30s, no tracker has events                 |
| `warming`  | At least one tracker awaiting first event            |
| `active`   | All trackers receiving events, none idle             |
| `idle`     | At least one tracker exceeds idle threshold          |
| `stalled`  | All active trackers exceed idle threshold            |

The phase is computed on each request from tracker state — no additional state storage. The algorithm handles edge cases: services with no trackers (configctl), services where only long-timeframe trackers are naturally idle (store), and the initial startup window.

### 2.2 Runtime Metadata in /diagz (Code Change)

**Problem:** When debugging a running service, basic runtime information (Go version, goroutine count) required shell access to the container.

**Solution:** Added `go_version` and `num_goroutines` to `/diagz` response. Zero overhead (uses `runtime.Version()` and `runtime.NumGoroutine()`).

### 2.3 Diagnostic Check Script (New Tooling)

**Problem:** `live-pipeline-activate.sh` combines stack startup, seeding, and diagnostics into one long flow. Checking an already-running stack required `--check-only` mode, which still runs all 8 phases.

**Solution:** Created `scripts/diag-check.sh` — a focused, read-only diagnostic script that:
- Probes `/readyz` on all runtimes.
- Reads `/statusz` for phase and tracker summary.
- Reads `/diagz` for readiness checks, goroutines, and Go version.
- Scans compose logs for error-level entries.
- Supports `--local` mode for direct HTTP (no compose exec).
- Runs in < 10 seconds on a healthy stack.

### 2.4 Pipeline Script Enhancement

Updated `live-pipeline-activate.sh` Phase 5 and Phase 8 to display the new `phase` field, making operational state immediately visible during pipeline activation.

## 3. Files Changed

### Modified

| File | Change |
|------|--------|
| `internal/shared/healthz/healthz.go` | Added `phase` field to `/statusz`, `phase`+`go_version`+`num_goroutines` to `/diagz`, added `computePhase()` method |
| `internal/shared/healthz/healthz_test.go` | Added 3 tests: Phase_Warming, Phase_Stalled, Diagz_RuntimeMetadata; updated Statusz test to verify phase |
| `scripts/live-pipeline-activate.sh` | Phase 5: display `phase` in /statusz output; Phase 8: display `phase` in tracker summary |

### Created

| File | Purpose |
|------|---------|
| `scripts/diag-check.sh` | Lightweight diagnostic snapshot script |
| `docs/architecture/current-baseline-operational-diagnostics.md` | Assessment of all diagnostic signals: what's sufficient, what's minimal, where gaps exist |
| `docs/architecture/current-baseline-runbook.md` | Operational runbook covering lifecycle, phases, diagnosis, and recovery |
| `docs/architecture/future-analytics-signals-candidates-for-clickhouse.md` | Disciplined catalog of signals that would benefit from ClickHouse persistence |
| `docs/stages/stage-s139-operational-diagnostics-and-runbook-hardening-report.md` | This report |

## 4. Operability Gains

| Before S139 | After S139 |
|-------------|------------|
| Operator reads per-tracker counts and mentally aggregates state | Single `phase` field gives immediate aggregate answer |
| No runtime metadata in diagnostic endpoints | `go_version` and `num_goroutines` in `/diagz` |
| Diagnostic checks bundled with pipeline activation | Standalone `diag-check.sh` for quick snapshots |
| Operational procedures scattered across stage reports | Consolidated runbook with lifecycle, diagnosis, and recovery |
| Diagnostic sufficiency undocumented | Formal assessment of what's sufficient vs minimal |
| ClickHouse signals undefined | Prioritized catalog with pre-conditions and anti-patterns |
| Idle tracker on 3600s family triggers confusion | Runbook documents expected idle intervals per timeframe |
| Post-crash data loss unclear | Recovery table with per-timeframe data loss estimates |

## 5. Limits Maintained

- **No new features** — all changes improve operability of existing capabilities.
- **No heavyweight observability** — no Prometheus, no OpenTelemetry, no new dependencies.
- **No ClickHouse implementation** — future signals cataloged, pre-conditions defined, no code written.
- **No new endpoints** — existing `/statusz` and `/diagz` enriched with computed fields.
- **No new runtime dependencies** — only `runtime` standard library package added.
- **No stack inflation** — zero new services, zero new NATS subjects, zero new KV buckets.

## 6. Test Results

All 12 healthz tests pass (3 new):
- `TestHealthServer_Statusz_Phase_Warming` — verifies `warming` when one tracker awaits first event.
- `TestHealthServer_Statusz_Phase_Stalled` — verifies `stalled` when all trackers exceed idle threshold.
- `TestHealthServer_Diagz_RuntimeMetadata` — verifies `go_version`, `num_goroutines`, and `phase` in `/diagz`.

## 7. Preparation for S140

S139 leaves the baseline in a hardened operational state. Recommended directions for S140:

1. **Product wave** (recommended by S136): exercise the pipeline with a concrete use case (paper trading dashboard, alert triggers, or backtesting). The operational foundation is now solid enough to support product-level validation.

2. **ClickHouse integration** (if product wave deferred): implement P1 signals (event flow metrics) following the catalog in `future-analytics-signals-candidates-for-clickhouse.md`. Pre-conditions CH-01 through CH-03 are already satisfied.

3. **Gateway diagnostic enrichment** (optional): add `/statusz` to gateway if product wave reveals the need. Currently not justified since gateway is a stateless proxy.

The key decision is whether the next wave exercises the pipeline for product value or deepens infrastructure. S136 recommended product wave — S139's hardening supports that recommendation.
