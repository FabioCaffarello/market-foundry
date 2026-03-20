# Family 03 — Runtime and Operability Notes

**Family**: Strategies (`mean_reversion_entry`)
**Stage**: S176
**Date**: 2026-03-19

---

## 1. Endpoint specification

```
GET /analytical/strategy/history
```

### Parameters

| Parameter   | Required | Type   | Default | Constraint     |
|-------------|----------|--------|---------|----------------|
| `type`      | yes      | string | —       | non-empty      |
| `source`    | yes      | string | —       | non-empty      |
| `symbol`    | yes      | string | —       | non-empty      |
| `timeframe` | yes      | int    | —       | > 0            |
| `direction` | no       | string | (all)   | empty OK       |
| `limit`     | no       | int    | 50      | 1–500          |
| `since`     | no       | int64  | 0       | unix seconds   |
| `until`     | no       | int64  | 0       | unix seconds   |

### Response

```json
{
  "strategies": [
    {
      "type": "mean_reversion_entry",
      "source": "binancef",
      "symbol": "btcusdt",
      "timeframe": 60,
      "direction": "long",
      "confidence": "0.85",
      "decisions": [
        {"type": "rsi_oversold", "outcome": "triggered", "confidence": "0.85", "timeframe": 60}
      ],
      "parameters": {"entry_threshold": "30"},
      "metadata": {"version": "1"},
      "final": true,
      "timestamp": "2026-03-19T10:00:00Z"
    }
  ],
  "source": "clickhouse",
  "meta": {"query_ms": 12, "row_count": 1}
}
```

### Headers

- `Server-Timing: total;dur=15, query;dur=12`

### Error codes

| Code | Condition |
|------|-----------|
| 400  | Missing required param, invalid limit, invalid since/until format |
| 503  | ClickHouse not configured, reader unavailable, query failed |

---

## 2. Data flow

```
Write path (pre-existing, zero changes):
  NATS subject: strategy.resolved.mean_reversion_entry
    → writer consumer (mean_reversion_entry pipeline)
      → mapStrategyRow() (15 values)
        → INSERT INTO strategies

Read path (new in S176):
  GET /analytical/strategy/history?...
    → GetStrategyHistory handler
      → GetStrategyHistoryUseCase.Execute()
        → StrategyReader.QueryStrategyHistory()
          → BuildStrategyQuery() → SELECT 11 columns
            → Scan + ParseDecisionInputsJSON + ParseMetadataJSON ×2 + FormatFloat
              → []strategy.Strategy → StrategyHistoryReply → JSON
```

---

## 3. Observability

| Signal | Source | Notes |
|--------|--------|-------|
| `events_received` counter | Writer consumer | Pre-existing |
| Inserter batch timing | Writer inserter actor | Pre-existing |
| `query completed` log | Strategy reader adapter | New — includes rows, elapsed_ms |
| `query failed` log | Strategy reader adapter | New — includes error |
| `analytical strategy query completed` log | Use case | New — includes rows, query_ms |
| `analytical strategy query failed` log | Use case | New — includes error |
| `analytical strategy request failed` log | HTTP handler | New — includes problem code |
| `Server-Timing` header | HTTP response | New |
| statusz/diagz pipeline visibility | Writer tracker | Pre-existing |

---

## 4. Diagnostic queries

```sql
-- Verify strategies table has data
SELECT count() FROM strategies;

-- Check recent strategy events
SELECT type, source, symbol, timeframe, direction, confidence, timestamp
FROM strategies
ORDER BY timestamp DESC
LIMIT 10;

-- Check distribution by direction
SELECT direction, count() FROM strategies GROUP BY direction;

-- Check distribution by type
SELECT type, count() FROM strategies GROUP BY type;

-- Verify JSON column content
SELECT decisions, parameters, metadata
FROM strategies
ORDER BY timestamp DESC
LIMIT 5;
```

---

## 5. Failure modes and recovery

| Symptom | Likely cause | Recovery |
|---------|-------------|----------|
| 503 on endpoint | ClickHouse not configured | Verify `clickhouse.addr` in gateway config |
| 503 on endpoint | ClickHouse connection failed | Check ClickHouse container health |
| Empty results for known events | Writer pipeline disabled | Check `IsStrategyFamilyEnabled("mean_reversion_entry")` in writer config |
| Empty results for known events | Events not yet flushed | Wait for writer batch flush interval |
| JSON fields return `[]` or `{}` | Serialization mismatch | Check `marshalJSON` output in writer logs |
| Slow queries (>1s) | Table growth beyond retention | Verify TTL is active, check partition count |
| Direction filter returns nothing | Invalid direction value | Expected behavior — invalid values return empty |

### Recovery actions

- **Writer restart**: Supervisor handles automatic reconnection. No manual intervention needed.
- **Reader degradation**: Gateway returns 503 for strategy endpoint only; other analytical endpoints unaffected.
- **Schema mismatch**: Compare DDL 15 columns vs `mapStrategyRow()` 15 values vs `Scan()` 11 variables.

---

## 6. Known limits

1. **No pagination beyond 500**: Sufficient for operational debugging queries
2. **TTL 90 days**: Strategies older than 90 days are automatically dropped by ClickHouse
3. **No direction validation**: Invalid direction values silently return empty results
4. **No confidence filtering**: Confidence is returned but not filterable
5. **No decision drill-down**: The `decisions` JSON array is returned as-is; no join to decisions table
6. **No cross-family queries**: Strategies are queried independently
7. **JSON parsed client-side**: No ClickHouse JSON functions used; string scan + Go unmarshal
