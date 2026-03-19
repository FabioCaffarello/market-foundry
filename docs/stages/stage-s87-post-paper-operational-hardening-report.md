# Stage S87: Post-Paper Operational Hardening Report

> Completed: 2026-03-19

## Objective

Transform `execute` from a functionally complete but operationally isolated binary into a first-class citizen of the market-foundry platform, eliminating infrastructure drift and closing operational gaps.

## Strategic Context

After S80-S86, the paper-integrated execution pipeline is architecturally formed: derive produces execution intents, execute consumes and fills them, store materializes results, and gateway exposes queries. However, `execute` was missing from core platform workflows (docker-compose, Makefile), its observability was limited, and stale build artifacts created drift.

S87 is hardening, not feature work. No new domain concepts, no venue real, no scope expansion.

## Changes Applied

### 1. Docker Compose Integration

**File:** `deploy/compose/docker-compose.yaml`

Added `execute` service definition:
- Build: shared `go-service.Dockerfile` with `SERVICE=execute`
- Port: `127.0.0.1:8084:8084`
- Config: `deploy/configs/execute.jsonc` (read-only mount)
- Dependencies: `nats` (service_healthy), `derive` (service_healthy)
- Health check: `GET /readyz` with grep "ready" (same pattern as all services)

`make up` now starts the complete stack including execute.

### 2. Makefile Build Integration

**File:** `Makefile`

- Added `execute` to `BUILDABLE_SERVICES` (alphabetical order)
- Updated help text to include `execute` in stack description
- `make build`, `make docker-build`, `make build SERVICE=execute` now work

### 3. Health/Observability Enhancement

**Files:** `internal/shared/healthz/healthz.go`, `internal/shared/healthz/healthz_test.go`

Extended the `Tracker` type with custom named counters:
- `Counter(name)` — returns an `atomic.Int64`, created on first access
- `Counters()` — returns a snapshot of all custom counters
- `/statusz` now includes a `counters` field in each tracker's JSON

This is a generic platform enhancement that any service can use — not execute-specific.

**Files:** `internal/actors/scopes/execute/venue_adapter_actor.go`

Migrated VenueAdapterActor from private atomic fields to tracker counters:
- `processed`, `filled`, `skipped_stale`, `skipped_halt` — all visible in `/statusz`
- Removed `sync/atomic` import (no longer needed as private fields)
- Shutdown stats log reads from tracker counters

### 4. Smoke Test Hardening

**File:** `scripts/smoke-multi-symbol.sh`

- Execute health checks (Step 16) promoted from soft warnings to hard pass/fail assertions
- Added `/statusz` validation (HTTP 200 check)
- Execute is now a mandatory stack member in the smoke test

### 5. Build Artifact Drift Cleanup

**File:** `.gitignore`

Added root-level compiled binary names to `.gitignore`:
- `/configctl`, `/derive`, `/execute`, `/gateway`, `/ingest`, `/store`
- Prevents accidental tracking of loose build artifacts at repo root

## Files Changed

| File | Change |
|------|--------|
| `deploy/compose/docker-compose.yaml` | Added execute service |
| `Makefile` | Added execute to BUILDABLE_SERVICES + help text |
| `internal/shared/healthz/healthz.go` | Added custom counter support to Tracker |
| `internal/shared/healthz/healthz_test.go` | Added counter and /statusz counter tests |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Migrated stats to tracker counters |
| `scripts/smoke-multi-symbol.sh` | Hardened execute health checks + added /statusz |
| `.gitignore` | Added root binary names |
| `docs/architecture/execute-operational-platform-integration.md` | New |
| `docs/architecture/execute-observability-and-runtime-health.md` | New |
| `docs/stages/stage-s87-post-paper-operational-hardening-report.md` | New (this file) |

## Drift/Legacy Removed

| Item | Before S87 | After S87 |
|------|-----------|-----------|
| execute in docker-compose | Absent | Present (first-class service) |
| execute in BUILDABLE_SERVICES | Absent | Present |
| execute smoke health checks | Soft warnings | Hard assertions |
| /statusz domain counters | Not available | Visible (processed/filled/skipped/halt) |
| Root binary artifacts in git | Untracked clutter | Gitignored |
| raccoon-cli binary-without-compose warning | Warning emitted | Resolved (compose service exists) |

## Guard Rail Compliance

| Guard rail | Status |
|------------|--------|
| No venue real opened | Compliant — paper_simulator only |
| No generic deployment framework | Compliant — standard compose pattern |
| No inflated telemetry | Compliant — counters are domain-relevant, not speculative |
| No vague documentation | Compliant — all docs reference concrete files and endpoints |

## Test Results

- `internal/shared/healthz` — all tests pass (including new counter tests)
- `internal/actors` — builds clean
- `cmd/execute` — builds clean

## Preparation for S88

The natural next steps after S87:

1. **Integration test harness with real NATS** — execute currently has unit tests for domain logic but no integration test that spins up a NATS container and validates the full consumer → gate → venue → publisher flow.

2. **Multi-instance execution guard** — if multiple execute instances are deployed, they'll duplicate fills. A NATS-level dedup or leader election mechanism is needed before any horizontal scaling.

3. **Venue activation gate ceremony design** — the architectural prerequisite for moving beyond paper_simulator. Requires: governance doc, risk review, adapter interface finalization.

4. **Prometheus /metrics endpoint** — when the operational team needs alerting and dashboards beyond structured logs and /statusz.

S88 recommendation: **Integration test harness with real NATS for execute**, validating the full paper execution lifecycle end-to-end in an isolated test environment.
