# CC-02 End-to-End Validation Procedure

**Stage:** S127 — CC-02 End-to-End Operational Validation
**Family:** `ema_crossover` (EMA Crossover Signal)
**Objective:** Validate that the CC-02 family operates correctly end-to-end in a controlled live environment.

---

## Prerequisites

- S126 implementation complete (3 new files, 7 modified).
- All unit tests pass (`make test`).
- Docker and docker compose available.
- No other stack running on ports 4222, 8080–8084.

---

## Phase 0: Clean Slate

```bash
make down
docker volume prune -f
```

Purpose: eliminate stale state from prior runs.

---

## Phase 1: Full Stack Activation

```bash
make live-multi
```

This orchestrates:
1. Build all service images.
2. Start compose stack (nats, configctl, gateway, ingest, derive, store, execute).
3. Wait for all services to become healthy (120s max, 5s poll).
4. Probe readiness on all runtimes (`/readyz`).
5. Seed configctl with multi-symbol bindings (btcusdt + ethusdt).
6. Validate gateway query surface — including `GET /signal/ema_crossover/latest`.
7. Wait for pipeline event flow (candle materialization).
8. Scan for error-level log entries.
9. Capture memory usage snapshot.

### Validation Checks (Phase 1)

| ID   | Check                              | Method                           | Pass Criteria                    |
|------|------------------------------------|----------------------------------|----------------------------------|
| A1   | All services healthy               | `docker compose ps`              | 7 services, all `healthy`        |
| A2   | All readiness probes pass          | `/readyz` on each service        | HTTP 200, `status: ready`        |
| A3   | Config activation succeeds         | `seed-configctl.sh --multi-symbol` | HTTP 200, 2 bindings visible   |
| A4   | EMA crossover endpoint reachable   | `GET /signal/ema_crossover/latest` | HTTP 200                       |

---

## Phase 2: Pipeline Proof — EMA Crossover Signal (PP-01, PP-02, PP-03)

### PP-01: Signal Published (derive → NATS)

**What to verify:** The derive runtime produces `ema_crossover` signal events on the NATS stream.

**Method:** Check `/statusz` on the derive runtime for the `signal-ema-crossover` tracker.

```bash
# Via compose exec (derive is not exposed on host):
docker compose -f deploy/compose/docker-compose.yaml exec -T derive \
  wget -q -O - http://127.0.0.1:8083/statusz
```

**Pass criteria:** Tracker named `signal-ema-crossover-*` present with `event_count > 0` after warm-up (21 candles at 60s = ~21 minutes).

**Note:** Before warm-up completes, `event_count` will be 0. This is expected — the EMA crossover sampler needs `slow_period` (21) candles before producing its first signal.

### PP-02: Signal Projected (store KV materialization)

**What to verify:** The store runtime materializes EMA crossover signals into the `SIGNAL_EMA_CROSSOVER_LATEST` KV bucket.

**Method:** Check `/statusz` on the store runtime for the `signal-ema-crossover-projection` tracker.

```bash
docker compose -f deploy/compose/docker-compose.yaml exec -T store \
  wget -q -O - http://127.0.0.1:8081/statusz
```

**Pass criteria:** Tracker named `signal-ema-crossover-projection` with `event_count > 0` after derive produces signals.

### PP-03: Signal Queryable (gateway HTTP)

**What to verify:** The gateway returns materialized EMA crossover signals via HTTP.

```bash
curl -s "http://127.0.0.1:8080/signal/ema_crossover/latest?source=binancef&symbol=btcusdt&timeframe=60"
```

**Pass criteria:**
- HTTP 200.
- Response contains `signal` key with non-null value (after warm-up).
- Signal fields: `type=ema_crossover`, `value` in `{bullish, bearish, neutral}`.
- Metadata contains: `fast_period`, `slow_period`, `fast_ema`, `slow_ema`, `spread`.

**Pre-warmup response (acceptable):**
```json
{"signal": null}
```

**Post-warmup response (expected):**
```json
{
  "signal": {
    "type": "ema_crossover",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "value": "bullish",
    "metadata": {
      "fast_period": "9",
      "slow_period": "21",
      "fast_ema": "67432.1500",
      "slow_ema": "67210.8800",
      "spread": "221.2700"
    },
    "final": true,
    "timestamp": "2025-03-19T10:35:00Z"
  }
}
```

---

## Phase 3: RSI Coexistence (PP-04)

**What to verify:** The existing RSI signal family continues to operate correctly alongside ema_crossover.

```bash
curl -s "http://127.0.0.1:8080/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60"
```

**Pass criteria:**
- RSI endpoint returns HTTP 200 with valid signal structure.
- RSI `event_count` on store trackers is non-zero.
- Both `SIGNAL_RSI_LATEST` and `SIGNAL_EMA_CROSSOVER_LATEST` KV buckets are populated.

---

## Phase 4: Diagnostic Surfaces (PP-05, PP-06)

### PP-05: `/statusz` includes ema_crossover trackers

Check each relevant runtime:

| Runtime | Expected Tracker(s)                                       |
|---------|----------------------------------------------------------|
| derive  | `signal-ema-crossover-btcusdt-60s`, `signal-ema-crossover-btcusdt-300s` (per symbol/timeframe) |
| store   | `signal-ema-crossover-projection`, `signal-ema-crossover-consumer` |

### PP-06: `/diagz` shows ema_crossover readiness

```bash
docker compose -f deploy/compose/docker-compose.yaml exec -T store \
  wget -q -O - http://127.0.0.1:8081/diagz
```

**Pass criteria:** `readiness_checks` all pass. Trackers section includes ema_crossover entries.

---

## Phase 5: E2E Smoke Test

```bash
make smoke-multi
```

The smoke test now includes:
- **Step 6a:** Signal EMA Crossover multi-symbol validation — verifies endpoint reachability, response structure, field validation, and metadata presence for all symbol × timeframe combinations.
- **Step 6b:** Cross-symbol EMA Crossover signal isolation — verifies independent data per symbol.

**Pass criteria:** All steps pass or show acceptable warm-up-pending status.

---

## Phase 6: Error and Stability Checks

| ID   | Check                          | Method                              | Pass Criteria               |
|------|--------------------------------|-------------------------------------|-----------------------------|
| S1   | No crashes                     | `docker compose ps`                 | All healthy, 0 restarts     |
| S2   | No error-level logs            | `make logs \| grep '"level":"error"'` | Zero matches              |
| S3   | Memory usage reasonable        | `docker stats --no-stream`          | No service > 200MB          |
| T1   | Unit tests pass                | `make test`                         | Exit 0                      |
| T2   | Smoke multi passes             | `make smoke-multi`                  | Exit 0                      |

---

## Phase 7: Graceful Shutdown

```bash
make down
```

All services should stop within 15 seconds without error-level log entries during shutdown.

---

## Validation Matrix Summary

| ID    | Check                              | Phase | Pass Criteria                              |
|-------|------------------------------------|---------|--------------------------------------------|
| A1    | All services healthy               | 1       | 7 services, all healthy                    |
| A2    | All readiness probes pass          | 1       | HTTP 200 on all /readyz                    |
| A3    | Config activation                  | 1       | 2 bindings visible                         |
| A4    | EMA crossover endpoint reachable   | 1       | HTTP 200                                   |
| PP-01 | Signal published (derive)          | 2       | Tracker event_count > 0                    |
| PP-02 | Signal projected (store)           | 2       | Projection tracker event_count > 0         |
| PP-03 | Signal queryable (gateway)         | 2       | HTTP 200, valid signal payload             |
| PP-04 | RSI coexistence                    | 3       | RSI still produces and queries correctly   |
| PP-05 | /statusz includes ema_crossover    | 4       | Tracker entries present                    |
| PP-06 | /diagz shows readiness             | 4       | All checks pass                            |
| S1    | No crashes                         | 6       | 0 restarts                                 |
| S2    | No error-level logs                | 6       | Zero matches                               |
| T1    | Unit tests pass                    | 6       | Exit 0                                     |
| T2    | Smoke multi passes                 | 5       | Exit 0                                     |

### Minimum Viable Success

A1–A4, PP-01–PP-06, S1, S2, T1, T2.

---

## Warm-Up Timing

| Signal Family  | Warm-Up Period              | At 60s Timeframe |
|---------------|-----------------------------|-------------------|
| RSI           | 15 candles (rsi_period=14)  | ~15 minutes       |
| EMA Crossover | 21 candles (slow_period=21) | ~21 minutes       |

For a fresh stack, full pipeline proof requires ~25 minutes of sustained operation to ensure both signal families have completed warm-up and produced at least one signal.
