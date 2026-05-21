# Wave B Family 02 — Decisions (RSI Oversold): Runtime and Operability Notes

## Overview

Runtime behavior, diagnostic signals, failure modes, and recovery actions for the Decisions analytical family added in S169.

## Runtime Activation Rules

The decision analytical endpoint activates when ALL conditions are met:

1. **ClickHouse configured** — `gateway.jsonc` contains a valid `clickhouse` block with addr, database, username, password.
2. **ClickHouse reachable** — connection succeeds during gateway startup.
3. **decisions table exists** — migration 003 was applied (by cmd/migrate or manually).
4. **rsi_oversold pipeline enabled** — `writer.jsonc` has `rsi_oversold` in the decision family list.

If any condition is unmet, the endpoint returns 503. The gateway remains healthy — ClickHouse is NOT in the readiness check.

## Diagnostic Signals

### Writer Side

```bash
# Check if writer is consuming decision events
docker compose -f deploy/compose/docker-compose.yaml exec writer \
  wget -q -O - http://127.0.0.1:8085/statusz | python3 -c "
import sys, json
d = json.load(sys.stdin)
for t in d.get('trackers', []):
    if 'decision' in t['name'].lower():
        print(f'{t[\"name\"]}: events={t.get(\"event_count\",0)} errors={t.get(\"error_count\",0)}')
        c = t.get('counters', {})
        print(f'  flushed={c.get(\"events_flushed\",0)} dropped={c.get(\"events_dropped\",0)}')
"
```

### ClickHouse Side

```sql
-- Row count for rsi_oversold decisions
SELECT count() FROM decisions WHERE type = 'rsi_oversold';

-- Sample recent decisions
SELECT type, source, symbol, timeframe, outcome, confidence, final, timestamp
FROM decisions
WHERE type = 'rsi_oversold'
ORDER BY timestamp DESC
LIMIT 5
FORMAT Pretty;

-- Outcome distribution
SELECT outcome, count() as cnt
FROM decisions
WHERE type = 'rsi_oversold'
GROUP BY outcome
ORDER BY cnt DESC;
```

### Gateway Side

```bash
# Basic endpoint check
curl -s "http://127.0.0.1:8080/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&limit=5" | python3 -m json.tool

# Check Server-Timing
curl -s -D - -o /dev/null "http://127.0.0.1:8080/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60"

# Outcome filter
curl -s "http://127.0.0.1:8080/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&outcome=triggered&limit=5" | python3 -m json.tool
```

## Failure Modes

| Failure | Symptom | Impact | Recovery |
|---------|---------|--------|----------|
| ClickHouse down | 503 on decision endpoint | Decision reads unavailable; write path buffers in writer | Restart ClickHouse; writer will retry |
| Writer decision consumer degraded | 0 new rows in ClickHouse | Historical queries return stale data | Check writer logs, restart writer if needed |
| decisions table missing | Query error on read path | 503 on endpoint | Run migrations: `make migrate-up` |
| JSON deserialization failure | signals field returns `[]` | Data loss in read path (signals array empty) | Check writer-side marshalJSON output |
| Confidence parse error | confidence returns "0" | Misleading value | Check writer-side parseFloat logs |

## Performance Notes

- Decision queries follow the same ORDER BY (source, symbol, timeframe, type, timestamp) index as the DDL.
- The `outcome` filter maps to a LowCardinality column — ClickHouse optimizes this efficiently.
- Default limit (50) and max limit (500) are identical to candles and signals — no family-specific tuning needed.
- Batch size, flush interval, and retry behavior are shared with all other writer pipelines.

## Observability Checklist

| Signal | Location | What to Watch |
|--------|----------|---------------|
| Writer event count | `/statusz` trackers | Decision tracker receiving events |
| Writer error count | `/statusz` trackers | Non-zero error_count on decision tracker |
| Writer degradation | `/diagz` trackers | pipeline_degraded counter > 0 |
| CH row count | `SELECT count() FROM decisions` | Growing over time |
| HTTP response time | Server-Timing header | query;dur=N stays reasonable |
| HTTP error rate | Gateway logs | Warn-level entries for decision queries |

## Interaction with Other Families

- **No cross-family coupling** — the decision endpoint reads only from the `decisions` table.
- **Writer independence** — the `rsi_oversold` consumer-inserter pair is independent of candle and signal pipelines.
- **Gateway independence** — the decision reader is constructed independently of candle/signal readers.
- **Smoke independence** — Phase 5c tests decisions independently; existing Phase 5/5b are unchanged.

## Configuration Reference

### writer.jsonc (relevant section)
```jsonc
"pipeline": {
  "decision": {
    "families": ["rsi_oversold"]
  }
}
```

### gateway.jsonc (relevant section)
```jsonc
"clickhouse": {
  "addr": "clickhouse:9000",
  "database": "default",
  "username": "default",
  "password": "clickhouse"
}
```

No additional configuration is needed — the decision endpoint activates automatically when ClickHouse is configured and the `decisions` table exists.
