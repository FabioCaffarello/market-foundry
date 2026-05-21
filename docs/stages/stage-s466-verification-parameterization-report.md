# S466: Verification Parameterization and Operator Ergonomics

**Status:** Complete
**Date:** 2026-03-24

## Objective

Reduce hardcoded verification parameters, improve operator ergonomics for
inspection flows, and make the verification pipeline more reusable across
environments.

## Deliverables

### Code Changes

1. **`parseQueryKeyParams` (evidence.go):** Added required validation for
   `source` and `symbol` query parameters. Previously only `timeframe` was
   validated; missing source/symbol silently produced empty results.

2. **Analytical limit constants (analytical.go):** Extracted magic numbers
   (default 50, min 1, max 500) into exported `AnalyticalDefaultLimit`,
   `AnalyticalMinLimit`, `AnalyticalMaxLimit`. Error messages now include bounds.

3. **`LifecycleListQuery` filtering (contracts.go, execution.go,
   query_responder_actor.go):** Added optional `Source`/`Symbol` fields to
   `LifecycleListQuery`. HTTP handler reads from query params. Query responder
   filters server-side. Backward compatible (empty struct = no filter).

4. **Health server options (healthz.go):** Extracted heartbeat interval (30s)
   and starting threshold (30s) from hardcoded values to configurable Options
   (`WithHeartbeatInterval`, `WithStartingThreshold`). Exported default
   constants.

5. **Smoke script parameterization (lib.sh, smoke-first-slice.sh,
   smoke-round-trip.sh):**
   - ClickHouse creds now env-overridable (`CLICKHOUSE_PORT`, `CLICKHOUSE_USER`,
     `CLICKHOUSE_PASSWORD`, `CLICKHOUSE_DATABASE`).
   - Poll interval overridable via `SMOKE_POLL_INTERVAL`.
   - Default source/symbol overridable via `DEFAULT_SOURCE`/`DEFAULT_SYMBOL`.

### Tests

- `s466_verification_parameterization_test.go` (handlers): 4 tests covering
  source/symbol/timeframe validation and constant values.
- `s466_healthz_options_test.go` (healthz): 3 tests covering default constants
  and new Options.
- `s466_lifecycle_filter_test.go` (executionclient): 2 tests covering
  serialization with/without filters.

### Documentation

- `docs/architecture/verification-parameterization-and-operator-ergonomics.md`
- `docs/architecture/verification-inputs-defaults-scope-semantics-and-limitations.md`

## What Remained Fixed

- Market segment names, phase strings, KV bucket names, NATS subjects, table
  schemas, venue validation bounds. These are domain model or protocol contracts,
  not verification parameters.

## Acceptance Criteria Evaluation

| Criterion | Met |
|-----------|-----|
| Verification less rigid and more reusable | Yes -- LifecycleList filters, env-overridable scripts |
| Operator ergonomics improved materially | Yes -- clear 400 errors, parameterized scripts |
| Reduced implicit convention dependency | Yes -- source/symbol validated, constants exported |
| Ready for batch audit/session evidence in S467 | Yes -- filtering and parameterization in place |

## Limitations

- No wildcard or pattern matching on lifecycle filters (exact match only).
- Health thresholds are global, not per-tracker.
- Smoke env vars don't propagate into Docker container service configs.

## Files Changed

| File | Change |
|------|--------|
| `internal/interfaces/http/handlers/evidence.go` | source/symbol validation |
| `internal/interfaces/http/handlers/analytical.go` | limit constants |
| `internal/interfaces/http/handlers/execution.go` | lifecycle list query params |
| `internal/application/executionclient/contracts.go` | LifecycleListQuery fields |
| `internal/actors/scopes/store/query_responder_actor.go` | server-side filtering |
| `internal/shared/healthz/healthz.go` | configurable thresholds |
| `scripts/utils/lib.sh` | env var defaults |
| `scripts/smoke-first-slice.sh` | poll interval parameterization |
| `scripts/smoke-round-trip.sh` | ClickHouse cred parameterization |
