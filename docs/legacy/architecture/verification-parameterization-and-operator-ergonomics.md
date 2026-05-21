# Verification Parameterization and Operator Ergonomics

S466 reduces hardcoded parameters in the verification and operational inspection
surfaces, making them more reusable across environments and less dependent on
implicit conventions.

## Changes

### 1. HTTP Query Key Validation (parseQueryKeyParams)

**Before:** `source` and `symbol` were silently accepted as empty strings, passing
through to KV/ClickHouse lookups that would return no results without any
indication of why. Only `timeframe` was validated.

**After:** All three parameters (`source`, `symbol`, `timeframe`) are validated
as required. Missing any one returns a 400 with a specific message naming the
missing field.

**Ergonomic impact:** Operators no longer see empty 200 responses when they
forget a query parameter. The error message tells them exactly what is missing.

### 2. Analytical Query Limit Constants

**Before:** Default limit (50), min (1), and max (500) were magic numbers
inlined in `parseAnalyticalParams`.

**After:** Exported as `AnalyticalDefaultLimit`, `AnalyticalMinLimit`,
`AnalyticalMaxLimit`. The error message for out-of-bounds limits now includes
the actual bounds.

**Ergonomic impact:** Operators and tests can reference the canonical bounds.
Error messages are self-documenting.

### 3. LifecycleListQuery Filtering

**Before:** `LifecycleListQuery` was an empty struct. The `/execution/lifecycle/list`
endpoint always returned all entries across all partition keys with no filtering.

**After:** `LifecycleListQuery` accepts optional `Source` and `Symbol` fields.
The HTTP handler reads these from query parameters. The query responder actor
filters entries server-side before returning.

**Ergonomic impact:** Operators can narrow lifecycle list to a specific source
or symbol without client-side filtering. Backward compatible: omitting the
params returns all entries as before.

### 4. Health Server Configurable Thresholds

**Before:** Heartbeat interval (30s), starting threshold (30s), and idle
threshold (2m) were hardcoded. Only idle threshold had an Option.

**After:** All three thresholds are exported as named constants
(`DefaultIdleThreshold`, `DefaultHeartbeatInterval`, `DefaultStartingThreshold`)
and configurable via `WithHeartbeatInterval` and `WithStartingThreshold` Options.

**Ergonomic impact:** Different environments (CI, soak, production) can tune
health timing without code changes.

### 5. Smoke Script Parameterization

**Before:** ClickHouse credentials (port, user, password) hardcoded in
`smoke-round-trip.sh`. Poll interval hardcoded in `smoke-first-slice.sh`.
Default symbol and source hardcoded in `lib.sh`.

**After:**
- `CLICKHOUSE_PORT`, `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`,
  `CLICKHOUSE_DATABASE` are environment-overridable in `lib.sh` and used
  by `smoke-round-trip.sh`.
- `SMOKE_POLL_INTERVAL` overrides the poll interval in `smoke-first-slice.sh`.
- `DEFAULT_SOURCE` and `DEFAULT_SYMBOL` are overridable in `lib.sh`.

**Ergonomic impact:** Smoke scripts work against non-default ClickHouse
credentials and different symbols without editing script source.

## What Remains Fixed (and Why)

| Item | Reason |
|------|--------|
| Market segment names (`binancef`/`binances`) | Domain model identity, not tuning |
| Phase strings (`starting`, `active`, etc.) | Fixed operational semantics |
| KV bucket names | Protocol contracts between services |
| NATS subject patterns | Infrastructure wiring, not verification |
| ClickHouse table name (`executions`) | Schema contract, not environment param |
| HTTP server timeouts (5s/5s/30s) | Reasonable defaults, rarely need tuning |
| Venue validation bounds (staleness 30-600s, submit 1-60s) | Safety invariants |

## Backward Compatibility

All changes are backward compatible:

- `parseQueryKeyParams`: source/symbol were already expected by callers; smoke
  scripts always provided them. The new validation catches only invalid usage.
- `LifecycleListQuery`: empty struct is still valid (zero-value fields = no filter).
- Health options: new options are additive; defaults match prior behavior.
- Smoke scripts: env var defaults match prior hardcoded values.
