# Analytical Runtime Activation Rules and Failure Modes

## Purpose

This document defines the activation rules, failure modes, and operational
semantics of the analytical runtime — the writer service and the gateway's
analytical endpoints. It serves as a reference for operators deploying,
diagnosing, or modifying the analytical layer.

## Activation Model

The analytical runtime is **optional and additive**. It activates only when
explicitly configured and does not affect the baseline pipeline (ingest → derive
→ store → gateway operational path).

### Writer Activation

The writer binary activates when:

1. A config file is provided with `clickhouse.addr` set.
2. `nats.enabled` is `true` with a valid `nats.url`.
3. At least one pipeline family is enabled.

If any of these preconditions are not met, the writer **exits immediately** with
a structured error message identifying the missing or invalid config field.

### Gateway Analytical Endpoints

The gateway activates analytical endpoints when:

1. `clickhouse.addr` is configured in the gateway config.
2. The ClickHouse config passes structural validation.
3. A connection to ClickHouse succeeds.

If any condition fails, the gateway **continues normally** without analytical
endpoints. Baseline readiness is not affected.

## Failure Modes

### F-01: Writer Exits on Missing ClickHouse Addr

| Property | Value |
|----------|-------|
| Trigger | `clickhouse.addr` is empty in writer config |
| Behavior | Writer logs error and exits with code 1 |
| Message | `writer startup blocked: clickhouse config validation failed` |
| Resolution | Set `clickhouse.addr` in `deploy/configs/writer.jsonc` |

### F-02: Writer Exits on Empty Database

| Property | Value |
|----------|-------|
| Trigger | `clickhouse.database` is empty when addr is set |
| Behavior | Writer logs error with field-level detail and exits |
| Resolution | Set `clickhouse.database` (typically "default") |

### F-03: Writer Exits on Empty Username

| Property | Value |
|----------|-------|
| Trigger | `clickhouse.username` is empty when addr is set |
| Behavior | Writer logs error with field-level detail and exits |
| Resolution | Set `clickhouse.username` (typically "default") |

### F-04: Writer Exits on Invalid Batching Config

| Property | Value |
|----------|-------|
| Trigger | Negative batch_size, max_pending, or max_retries; invalid duration strings |
| Behavior | Writer logs all issues and exits |
| Resolution | Fix the offending fields; zero values silently default to safe values |

### F-05: Writer Exits on NATS Not Enabled

| Property | Value |
|----------|-------|
| Trigger | `nats.enabled` is false or missing |
| Behavior | Writer logs error and exits |
| Message | `writer startup blocked: nats.enabled must be true` |
| Resolution | Set `nats.enabled: true` and provide `nats.url` |

### F-06: Writer Exits on Empty Pipeline

| Property | Value |
|----------|-------|
| Trigger | No families configured in any layer |
| Behavior | Writer logs error and exits |
| Resolution | Enable at least one family (e.g., `"families": ["candle"]`) |

### F-07: Writer Exits on Pipeline Dependency Violation

| Property | Value |
|----------|-------|
| Trigger | Signal/decision/strategy/risk/execution family enabled without its upstream dependency |
| Behavior | Writer logs all dependency issues and exits |
| Resolution | Enable the required upstream family or remove the dependent family |

### F-08: Writer Exits on Connection Failure

| Property | Value |
|----------|-------|
| Trigger | `clickhouse.Open()` fails (network, auth, etc.) |
| Behavior | Writer logs error with addr and exits |
| Resolution | Verify ClickHouse is running and reachable; check credentials |

### F-09: Gateway Degrades on Invalid ClickHouse Config

| Property | Value |
|----------|-------|
| Trigger | ClickHouse addr is set but config is structurally invalid |
| Behavior | Gateway logs warning, disables analytical endpoints, continues |
| Impact | `/analytical/*` endpoints are not registered; baseline routes unaffected |

### F-10: Gateway Degrades on Connection Failure

| Property | Value |
|----------|-------|
| Trigger | ClickHouse connection fails at gateway startup |
| Behavior | Gateway logs warning with addr, disables analytical endpoints, continues |
| Impact | Same as F-09 |

### F-11: Writer Pipeline Family Degrades at Runtime

| Property | Value |
|----------|-------|
| Trigger | Consumer startup failure exceeds restart budget (5 attempts) |
| Behavior | Family marked degraded; other families continue; health phase → "degraded" |
| Observable | `/statusz` shows `pipeline_degraded` counter > 0 for affected trackers |

## Validation Order Guarantee

The writer validates configuration in this strict order:

```
NATS enabled → ClickHouse config → Pipeline config → Log summary → Open connections
```

This means:
- A NATS error is reported before ClickHouse errors.
- All config errors are reported before any connection attempt.
- The operator sees the maximum number of actionable issues per restart cycle.

## Observability

### Writer Startup

On successful validation, the writer logs:

```
writer config validated  clickhouse_addr=clickhouse:9000  clickhouse_database=default  batch_size=1000  flush_interval=5s  max_pending=10000  max_retries=5  nats_url=nats://nats:4222
```

### Writer Runtime

| Endpoint | Purpose |
|----------|---------|
| `/healthz` | Liveness — always 200 |
| `/readyz` | Readiness — checks NATS + ClickHouse ping |
| `/statusz` | Phase + per-pipeline tracker counters |
| `/diagz` | Full diagnostic snapshot |

### Gateway Analytical Activation

On activation:
```
clickhouse connected, analytical endpoints enabled  addr=clickhouse:9000  database=default
```

On skip:
```
clickhouse not configured, analytical endpoints disabled
```

On validation failure:
```
clickhouse config invalid, analytical endpoints disabled  error=...
```

## ClickHouse Optionality Rules

| Binary | ClickHouse | Behavior |
|--------|-----------|----------|
| writer | Required | Hard exit on missing or invalid config |
| gateway | Optional | Graceful degradation; analytical endpoints disabled |
| derive | Not used | ClickHouse config section ignored |
| store | Not used | ClickHouse config section ignored |
| ingest | Not used | ClickHouse config section ignored |
| execute | Not used | ClickHouse config section ignored |
| configctl | Not used | ClickHouse config section ignored |

## What Remains Out of Scope

- Schema migration validation at startup (separate `cmd/migrate` concern).
- Runtime ClickHouse reconnection (handled by the ClickHouse driver).
- Writer auto-recovery after degradation (requires process restart).
- Cross-service config coherence checks (writer families vs store projections).
