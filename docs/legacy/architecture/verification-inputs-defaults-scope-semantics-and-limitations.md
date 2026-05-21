# Verification Inputs, Defaults, Scope Semantics, and Limitations

Reference for all parameterizable verification inputs after S466.

## HTTP Query Surfaces

### Query Key Parameters (all families)

| Parameter | Required | Type | Default | Semantics |
|-----------|----------|------|---------|-----------|
| `source` | Yes | string | -- | Exchange source identifier (e.g., `binancef`, `binances`) |
| `symbol` | Yes | string | -- | Trading pair (e.g., `btcusdt`) |
| `timeframe` | Yes | int | -- | Candle interval in seconds (e.g., `60`, `300`) |

Missing any required parameter returns HTTP 400 with a message naming the field.

### Analytical Pagination Parameters

| Parameter | Required | Type | Default | Bounds |
|-----------|----------|------|---------|--------|
| `limit` | No | int | 50 | 1 -- 500 |
| `since` | No | int64 | 0 (no filter) | Unix timestamp |
| `until` | No | int64 | 0 (no filter) | Unix timestamp |

Constants exported as `AnalyticalDefaultLimit`, `AnalyticalMinLimit`, `AnalyticalMaxLimit`.

### Lifecycle List Filters

| Parameter | Required | Type | Default | Semantics |
|-----------|----------|------|---------|-----------|
| `source` | No | string | "" (all) | Narrows to entries matching source |
| `symbol` | No | string | "" (all) | Narrows to entries matching symbol |

Both filters are applied server-side in the query responder actor.

## Health Server Configuration

| Option | Default | Semantics |
|--------|---------|-----------|
| `WithIdleThreshold(d)` | 2 minutes | Duration after which idle trackers emit warnings |
| `WithHeartbeatInterval(d)` | 30 seconds | Interval between idle heartbeat checks |
| `WithStartingThreshold(d)` | 30 seconds | Grace period before "starting" phase ends |

Defaults are exported as `DefaultIdleThreshold`, `DefaultHeartbeatInterval`, `DefaultStartingThreshold`.

## Smoke Script Environment Variables

| Variable | Default | Used By |
|----------|---------|---------|
| `BASE_URL` | `http://127.0.0.1:8080` | All smoke scripts |
| `SMOKE_WAIT` | varies per script | Wait timeout override |
| `SMOKE_POLL_INTERVAL` | 5 | Poll interval in `smoke-first-slice.sh` |
| `HEALTH_WAIT_MAX` | 120 | Health check wait ceiling |
| `HEALTH_POLL_INTERVAL` | 5 | Health check poll interval |
| `CANDLE_WAIT_MAX` | 90 | Candle wait ceiling |
| `CANDLE_POLL_INTERVAL` | 5 | Candle poll interval |
| `DEFAULT_SYMBOL` | `btcusdt` | Default trading pair |
| `DEFAULT_SOURCE` | `binancef` | Default exchange source |
| `CLICKHOUSE_PORT` | `9000` | ClickHouse connection port |
| `CLICKHOUSE_USER` | `default` | ClickHouse username |
| `CLICKHOUSE_PASSWORD` | `clickhouse` | ClickHouse password |
| `CLICKHOUSE_DATABASE` | `market_foundry` | ClickHouse database name |

## Scope Semantics

**Source** identifies the exchange adapter instance (e.g., `binancef` for Binance
Futures, `binances` for Binance Spot). It is part of the partition key for both
NATS KV and ClickHouse. The mapping from MarketSegment to source is fixed in the
domain model and is NOT a verification parameter.

**Symbol** identifies the trading pair. It is always lowercase.

**Timeframe** is always in seconds. Common values: 60, 300, 900, 3600.

## Limitations

1. **No wildcard queries on required key params.** The evidence/signal/decision/
   strategy/risk/execution history endpoints require all three key params.
   The relaxed-filter `execution/list` and `execution/summary` endpoints accept
   partial filters.

2. **LifecycleList filtering is exact match only.** No prefix or pattern matching.

3. **Health thresholds apply globally, not per-tracker.** The idle threshold is
   the same for all trackers within a single health server.

4. **Smoke script env vars do not propagate into Docker containers.** ClickHouse
   credentials in smoke scripts are for the `docker compose exec` client command,
   not for the services themselves (services use their own config).

5. **ALL_TIMEFRAMES in lib.sh** is overridable but must be a space-separated
   list of integers, not a comma-separated list.
