# Configuration Reference

> Quick reference for all service configuration fields.
> Each service uses a JSONC file in `deploy/configs/`.

---

## Common Fields (all services)

### `log`

| Field    | Type   | Default  | Values                          |
|----------|--------|----------|---------------------------------|
| `level`  | string | `"info"` | `debug`, `info`, `warn`, `error`|
| `format` | string | `"text"` | `json`, `text`                  |

### `http`

| Field              | Type     | Default  | Constraint          |
|--------------------|----------|----------|---------------------|
| `addr`             | string   | `":8080"`| Go listen address   |
| `read_timeout`     | duration | `"10s"`  | > 0                 |
| `write_timeout`    | duration | `"15s"`  | > 0                 |
| `idle_timeout`     | duration | `"60s"`  | > 0                 |
| `shutdown_timeout` | duration | `"10s"`  | > 0                 |

### `nats`

| Field             | Type     | Default  | Notes                                    |
|-------------------|----------|----------|------------------------------------------|
| `enabled`         | bool     | `false`  | Must be `true` for pipeline services      |
| `url`             | string   | —        | Required when enabled. Docker: `nats://nats:4222`, local: `nats://localhost:4222` |
| `request_timeout` | duration | `"2s"`   | Max wait for request/reply                |

---

## Service-Specific Ports

| Service    | Internal Port | External (host) |
|------------|--------------|-----------------|
| configctl  | 8080         | —               |
| store      | 8081         | —               |
| ingest     | 8082         | —               |
| derive     | 8083         | —               |
| execute    | 8084         | —               |
| gateway    | 8080         | 8080            |

Only gateway is exposed to the host. Other services are reached via `docker compose exec` or through the gateway proxy.

---

## Pipeline Config (`derive`, `store`, `execute`)

### `pipeline.timeframes`

Integer array of candle window durations in seconds.

| Constraint | Value  |
|------------|--------|
| Minimum    | 10     |
| Maximum    | 86400  |
| Duplicates | Rejected |
| Default (if empty) | `[60]` |

### Family Lists

All family lists are opt-in arrays. Absent or empty means **no activation** (except `families` which defaults to all when empty).

| Field                | Known Values                          | Default behavior     |
|----------------------|---------------------------------------|----------------------|
| `families`           | `candle`, `tradeburst`, `volume`      | Empty = all enabled  |
| `signal_families`    | `rsi`, `ema_crossover`                | Empty = none enabled |
| `decision_families`  | `rsi_oversold`                        | Empty = none enabled |
| `strategy_families`  | `mean_reversion_entry`                | Empty = none enabled |
| `risk_families`      | `position_exposure`                   | Empty = none enabled |
| `execution_families` | `paper_order`, `venue_market_order`   | Empty = none enabled |

### Cross-Layer Dependencies

Enabling a downstream family requires its upstream dependency to also be enabled:

```
evidence (candle) ← signal (rsi, ema_crossover)
                         ← decision (rsi_oversold ← rsi)
                              ← strategy (mean_reversion_entry ← rsi_oversold)
                                   ← risk (position_exposure ← mean_reversion_entry)
                                        ← execution (paper_order ← position_exposure)
                                                     (venue_market_order ← position_exposure)
```

Validation rejects configs that violate these dependencies at startup.

---

## Venue Config (`execute` only)

### `venue`

| Field              | Type     | Default  | Range        | Notes                              |
|--------------------|----------|----------|--------------|------------------------------------|
| `type`             | string   | —        | —            | `paper_simulator` or `binance_futures_testnet` |
| `staleness_max_age`| duration | `"120s"` | 30s – 600s   | Reject intents older than this     |
| `submit_timeout`   | duration | `"10s"`  | 1s – 60s     | Max wait for venue SubmitOrder     |

Only `paper_simulator` is currently approved. `binance_futures_testnet` requires activation gate ceremony.

---

## Diagnostic Endpoints (all services)

| Endpoint   | Purpose                                | Notes                        |
|------------|----------------------------------------|------------------------------|
| `/healthz` | Liveness probe                         | Always 200 if process alive  |
| `/readyz`  | Readiness probe                        | 200 when all checks pass     |
| `/statusz` | Activity status, tracker state, phase  | Phase: starting/warming/active/idle/stalled |
| `/diagz`   | Diagnostic summary (goroutines, checks)| Machine-readable overview    |
