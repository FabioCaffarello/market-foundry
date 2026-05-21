# Analytical Config and Startup Validation Hardening

## Purpose

This document defines the config validation rules and startup semantics for the
analytical layer (writer + gateway analytical runtime), as hardened in S161.

The goal is **not** to expand functionality but to ensure misconfiguration fails
early, explicitly, and with actionable messages — before any I/O or connection
attempt.

## Design Principles

1. **Fail fast, fail loud.** Every config invariant is checked before opening
   connections. Errors name the offending field and explain what is expected.
2. **Aggregate all issues.** Validation collects every problem before returning,
   so operators fix everything in one pass rather than chasing one-at-a-time
   errors through restart cycles.
3. **Writer-specific vs generic validation.** `Validate()` remains lenient
   (returns nil when addr is empty — optional for gateway/other binaries).
   `ValidateForWriter()` is strict: addr, database, and username are hard
   requirements.
4. **ClickHouse remains optional for the baseline.** Services that do not
   participate in the analytical layer (derive, store, ingest, execute) are
   completely unaffected.
5. **No hidden defaults.** When batching fields default to fallback values
   (batch_size → 1000, flush_interval → 5s), the startup log emits the
   effective values so the operator knows exactly what the writer will use.

## Writer Startup Validation Sequence

The writer `Run()` validates in this order:

| Step | Check | Failure |
|------|-------|---------|
| 1 | `nats.enabled == true` | Hard exit — writer consumes from JetStream |
| 2 | `ClickHouseConfig.ValidateForWriter()` | Hard exit — checks addr, database, username, batching fields |
| 3 | `PipelineConfig.ValidateForWriter()` | Hard exit — standard pipeline rules + at least one family enabled |
| 4 | Log validated config summary | — |
| 5 | `clickhouse.Open()` | Hard exit — connection failure |
| 6 | `buildTrackers()` | Hard exit — no enabled families |
| 7 | Spawn supervisor | — |

All config checks (steps 1–3) complete before any connection is attempted (step 5).

## ClickHouseConfig.ValidateForWriter()

Validates all fields the writer depends on:

| Field | Rule |
|-------|------|
| `addr` | Must not be empty |
| `database` | Must not be empty |
| `username` | Must not be empty |
| `password` | Not validated (empty password may be intentional in dev) |
| `batch_size` | Must not be negative |
| `max_pending` | Must not be negative |
| `max_retries` | Must not be negative |
| `flush_interval` | Must be valid Go duration if set |
| `initial_backoff` | Must be valid Go duration if set |

All issues are aggregated into a single `problem.Problem` response.

## PipelineConfig.ValidateForWriter()

Extends the standard `ValidatePipeline()` with:

- At least one family must be configured across all layers (families,
  signal_families, decision_families, strategy_families, risk_families,
  execution_families). An empty pipeline config is valid for other binaries
  but invalid for the writer.

## Gateway Analytical Client Validation

`buildAnalyticalClient()` now validates ClickHouse config before attempting to
connect:

1. If `addr` is empty → log info, return nil (analytical disabled).
2. If `addr` is set but config is invalid → log warning with problem details,
   return nil (analytical disabled, gateway continues).
3. If config is valid but connection fails → log warning with addr, return nil.
4. If connection succeeds → log success with addr and database.

The gateway never hard-exits on ClickHouse problems. Analytical endpoints are
additive and their absence does not affect baseline readiness.

## Startup Config Summary Log

The writer emits a structured log entry after validation passes:

```
writer config validated  clickhouse_addr=... clickhouse_database=... batch_size=1000 flush_interval=5s max_pending=10000 max_retries=5 nats_url=...
```

This makes the effective configuration visible in logs without requiring a
separate diagnostics call, aiding post-mortem analysis and deploy verification.

## What Is NOT Validated

| Item | Reason |
|------|--------|
| ClickHouse schema existence | Validated at query time; migrations are a separate concern |
| NATS stream/consumer existence | Created at runtime by the consumer actor |
| Network reachability | Validated by `clickhouse.Open()` and readiness checks |
| Password correctness | Validated by `clickhouse.Open()` at connection time |
| Pipeline family ↔ ClickHouse table alignment | Enforced by the pipeline catalog (compile-time mapping) |

## Files Changed

- `internal/shared/settings/schema.go` — added `ValidateForWriter()` for both ClickHouseConfig and PipelineConfig; refactored internal validation helpers
- `internal/shared/settings/settings_test.go` — 11 new tests for writer validation
- `cmd/writer/run.go` — consolidated validation with fail-fast ordering and config summary log
- `cmd/gateway/compose.go` — added ClickHouse config validation before connection attempt
