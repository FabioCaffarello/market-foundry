# Runtime, Config, and Operational Closure

> S208 — Consolidation of startup, configuration, diagnostics, and health
> across all market-foundry runtimes.

## Purpose

This document records the operational closure of runtime configuration,
startup validation, health/diagnostics, and recovery semantics as of S208.
It is not a redesign; it is a factual snapshot of what is closed, what remains
limited, and what the next phase can rely on.

---

## 1. Runtime Inventory

| Service     | Port  | NATS | ClickHouse | Pipeline cfg | /statusz | /diagz | /readyz |
|-------------|-------|------|------------|-------------|----------|--------|---------|
| configctl   | 8080  | req  | —          | —           | yes      | yes    | yes     |
| gateway     | 8080  | req  | optional   | —           | —        | —      | yes     |
| ingest      | 8082  | req  | —          | —           | yes      | yes    | yes     |
| derive      | 8083  | req  | —          | yes          | yes      | yes    | yes     |
| store       | 8081  | req  | —          | yes          | yes      | yes    | yes     |
| execute     | 8084  | req  | —          | yes (+ venue)| yes      | yes    | yes     |
| writer      | 8085  | req  | required   | yes          | yes      | yes    | yes     |
| migrate     | CLI   | —    | required   | —           | —        | —      | —       |

### Key observations

- Gateway is the only HTTP-facing service without `/statusz`/`/diagz`.
  This is by design: gateway is a stateless proxy. It reports readiness
  via `/readyz` and ClickHouse availability via HTTP 503 on analytical endpoints.
- Writer is the only service that **requires** ClickHouse at startup.
  Gateway degrades gracefully (503 on analytical endpoints) if ClickHouse
  is unreachable.
- Migrate is a one-shot CLI tool, not a long-running service.

---

## 2. Configuration Validation at Startup

### Shared validation (all services via `bootstrap.Main`)

- Log level: must be `debug|info|warn|error` (default: `info`)
- Log format: must be `json|text` (default: `text`)
- HTTP addr: must not be empty (default: `:8080`)
- HTTP timeouts: read/write/idle/shutdown must be > 0
- NATS: if `enabled`, URL must match `nats://host:port`

### Writer-specific validation (`ValidateForWriter`)

Phase 0 fails fast on:
1. `nats.enabled` must be `true`
2. ClickHouse `addr` and `database` must not be empty
3. ClickHouse `username` must not be empty
4. Batching fields (`batch_size`, `max_pending`, `max_retries`) must not be negative
5. Duration fields (`flush_interval`, `initial_backoff`) must parse correctly
6. Pipeline config must have at least one enabled family
7. All family names must be in the known set
8. Cross-layer dependency rules must be satisfied

### Gateway analytical validation

- If ClickHouse config is present, `addr` and `database` must not be empty
- If ClickHouse is unreachable at startup, gateway starts anyway (log warning)
- Analytical endpoints return 503 until ClickHouse connection is established

### Pipeline dependency enforcement

```
evidence (candle, tradeburst, volume)
  └── signal (rsi, ema, ema_crossover) — depends on candle
       └── decision (rsi_oversold) — depends on rsi
            └── strategy (mean_reversion_entry) — depends on rsi_oversold
                 └── risk (position_exposure) — depends on mean_reversion_entry
                      └── execution (paper_order, venue_market_order) — depends on position_exposure
```

Validation ensures that enabling a downstream family without its upstream
dependency fails at startup with an actionable error message.

---

## 3. Health and Diagnostics

### Endpoints

| Endpoint   | Purpose                          | Available on         |
|------------|----------------------------------|---------------------|
| `/healthz` | Liveness (process alive)         | All services        |
| `/readyz`  | Readiness (dependencies checked) | All services        |
| `/statusz` | Phase + tracker activity         | Pipeline services   |
| `/diagz`   | Machine-readable diagnostics     | Pipeline services   |

### Phase computation

- `starting` — uptime < 30s and no tracker events
- `warming` — at least one tracker awaiting first event
- `active` — all trackers receiving events, none idle
- `idle` — at least one tracker exceeds idle threshold (2 min)
- `stalled` — all trackers exceed idle threshold
- `degraded` — any tracker has `pipeline_degraded` counter > 0 (writer only)

### Readiness checks

- NATS: TCP dial with 2s timeout
- ClickHouse: `Ping()` (writer only)
- Configctl gateway: availability probe (gateway only)

---

## 4. Recovery Semantics

### Shutdown

- SIGTERM triggers graceful shutdown (15s budget)
- Order: poison actor → stop health server → cleanup

### Restart recovery

- NATS durable consumers resume from last acknowledged position
- ClickHouse client reconnects automatically
- Writer supervisor restarts failed families with exponential backoff:
  2s → 4s → 8s → 16s → 30s (capped), 5 attempts before `degraded`
- Other services recover NATS subscriptions transparently

### Time-to-healthy

- Single service restart: < 30s
- Full stack restart: < 2 min (with health check polling)

---

## 5. Configuration Files

| File                        | Service   | Key settings                          |
|-----------------------------|-----------|---------------------------------------|
| `deploy/configs/gateway.jsonc`   | gateway   | NATS, ClickHouse (optional), HTTP     |
| `deploy/configs/writer.jsonc`    | writer    | NATS, ClickHouse (required), pipeline |
| `deploy/configs/derive.jsonc`    | derive    | NATS, pipeline (4 timeframes)         |
| `deploy/configs/store.jsonc`     | store     | NATS, pipeline (full family set)      |
| `deploy/configs/execute.jsonc`   | execute   | NATS, venue, pipeline                 |
| `deploy/configs/configctl.jsonc` | configctl | NATS (minimal)                        |
| `deploy/configs/ingest.jsonc`    | ingest    | NATS (minimal)                        |

### Cross-runtime consistency

There is no automated validation that config files across services are
consistent (e.g., that store's pipeline families are a superset of derive's).
This is a known limitation. Mismatches cause silent failures — events
published but not projected.

**Mitigation:** The `diag-check.sh` script and `/statusz` endpoints make
family mismatches observable through event count discrepancies across services.

---

## 6. What Is Operationally Closed

| Area                          | Status | Evidence                                    |
|-------------------------------|--------|---------------------------------------------|
| Startup validation            | Closed | All services fail fast on invalid config    |
| Writer-specific validation    | Closed | `ValidateForWriter` covers all fields       |
| Health endpoints              | Closed | All services expose `/healthz` + `/readyz`  |
| Diagnostics                   | Closed | Pipeline services expose `/statusz`+`/diagz`|
| Phase computation             | Closed | 6 phases cover all operational states       |
| Recovery (restart)            | Closed | Durable consumers + supervisor backoff      |
| Graceful degradation          | Closed | Gateway 503 when ClickHouse unavailable     |
| Config file coverage          | Closed | All 7 services have deploy configs          |
| Docker health checks          | Closed | All services have compose health checks     |

## 7. What Remains Limited (by Design)

| Area                              | Status       | Rationale                                  |
|-----------------------------------|--------------|--------------------------------------------|
| Gateway `/statusz`/`/diagz`       | Not planned  | Stateless proxy; no trackers to report     |
| Cross-service config validation   | Not planned  | Would require centralized config registry  |
| NATS JetStream readiness check    | TCP-only     | Sufficient for current scale; JetStream    |
|                                   |              | readiness would add startup complexity     |
| ClickHouse reconnection (gateway) | No heartbeat | 503 on analytical endpoints is sufficient  |
| Performance thresholds            | Not enforced | No latency assertions in health checks     |
| Distributed tracing               | Deferred     | Premature for current operational scope    |
| Alerting rules                    | Deferred     | No external monitoring integration yet     |

---

## 8. S208 Changes Applied

1. **`scripts/utils/lib.sh`** — Added `writer` to `PIPELINE_SERVICES` array.
   Writer has `/statusz` and `/diagz` but was missing from the shared constant.
2. **`Makefile`** — Fixed `make up` help text to reflect full stack
   (was listing only original 7 services, missing clickhouse/migrations/writer).
