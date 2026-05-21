# CI Smoke-Analytical Integration

> How the analytical smoke test is integrated into CI, what it covers, and how to maintain it.

## Motivation

Constraint C-3 (from S162) requires CI integration of smoke-analytical before the second Wave B family. S164 and S165 confirmed this as the sole hard blocker (PF-6). This document describes the integration.

## CI Architecture

### Workflow: `.github/workflows/ci.yml`

The CI pipeline has two jobs:

```
┌──────────────┐     ┌──────────────────────┐
│  unit-tests  │────▶│  smoke-analytical    │
│              │     │  (needs: unit-tests) │
└──────────────┘     └──────────────────────┘
```

**Job 1: unit-tests**
- Runs `make test` across all Go workspace modules.
- Fast (~30s). Catches compilation errors and unit test regressions.
- Must pass before smoke-analytical runs.

**Job 2: smoke-analytical**
- Depends on `unit-tests` passing.
- Builds service binaries (`make build`).
- Starts the full compose stack (`make up`).
- Seeds configctl (`make seed`).
- Waits for writer flush (120s — writer batch interval).
- Runs `make smoke-analytical` (the full `smoke-analytical-e2e.sh` script).
- Collects compose logs on failure for debugging.
- Tears down the stack on completion.

### Trigger Events

- **Push to `main`**: Validates that the main branch remains green.
- **Pull request targeting `main`**: Gates merges — no PR merges without CI passing.

### Failure Diagnostics

When smoke-analytical fails in CI:

1. The workflow collects compose logs (all services, last 200 lines) and uploads them as a `compose-logs` artifact retained for 7 days.
2. Writer and gateway logs are printed inline in the job output.
3. The smoke script's own output includes per-phase PASS/FAIL indicators with context.

### Timeout Budget

| Phase | Budget |
|---|---|
| Infrastructure readiness (ClickHouse health) | 120s |
| Seed configctl | ~5s |
| Writer flush wait | 120s |
| Smoke script execution | ~60s |
| **Total estimated** | **~5 min** |

The GitHub Actions job has no explicit timeout override, using the default 6-hour limit. The compose readiness check has its own 120s timeout via the `timeout` command.

## What the Smoke Test Validates

The smoke script (`scripts/smoke-analytical-e2e.sh`) covers 7 phases:

| Phase | What It Proves |
|---|---|
| 1. Infrastructure Readiness | ClickHouse, writer, gateway are healthy |
| 2. Migration Status | All 6 core tables exist, migrations applied |
| 3. Writer Pipeline Health | NATS → writer consumption active |
| 4. ClickHouse Data Verification | Rows exist in evidence_candles and signals tables |
| 5a. Candle Read Path | GET /analytical/evidence/candles → 200 with correct structure |
| 5b. Signal Read Path | GET /analytical/signal/history → 200 with correct structure, Server-Timing |
| 6. Error Handling | Invalid params return 400 for both families |
| 7. Writer Observability | diagz shows healthy pipelines, no degraded state |

## How to Extend for New Families

When adding a new analytical family in Wave B:

1. Add the family's write path to `cmd/writer/pipeline.go`.
2. Add the family's read path (reader + use case + handler + route).
3. Extend `smoke-analytical-e2e.sh` with a new phase section following the Phase 5b pattern.
4. The CI workflow requires no changes — it runs the same `make smoke-analytical` target.

The smoke script grows per family. At family 3, extract a `validate_analytical_family()` bash function (per PF-5 hardening commitment).

## Local vs CI Execution

The same `make smoke-analytical` target runs in both environments:

- **Local:** Developer starts the stack (`make up`), seeds (`make seed`), waits, then runs `make smoke-analytical`.
- **CI:** The workflow automates the same sequence. No CI-specific scripts or wrappers.

This ensures parity between local validation and CI — what passes locally passes in CI, and vice versa.

## Maintenance Rules

1. **Never skip the smoke job.** If it's too slow, optimize the script — don't disable the job.
2. **Every new family extends the smoke script.** A family without a smoke phase does not pass the gate review.
3. **Log artifacts are diagnostic only.** Do not parse them programmatically or build alerting on CI artifacts.
4. **Flush wait is configurable.** If writer batch intervals change, update the `sleep 120` in the workflow and the `--wait` default in the script.
5. **Unit tests gate smoke.** Smoke never runs if unit tests fail. This prevents long CI runs when compilation is broken.
