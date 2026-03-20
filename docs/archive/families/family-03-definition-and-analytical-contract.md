# Family 03 Definition and Analytical Contract

**Family**: Strategies (`mean_reversion_entry`)
**Layer**: 4 (evidence → signal → decision → **strategy**)
**Status**: Defined — ready for S176 implementation
**Predecessor**: Family 02 (Decisions / `rsi_oversold`)
**Selection rationale**: See `family-03-selection-rationale-and-deferred-candidates.md`

---

## 1. Analytical contract

Family 03 makes strategy resolution events queryable through the analytical read path.
The contract is: **given a strategy type, source, symbol, and timeframe, return historical strategy resolutions ordered newest-first, with optional direction and time-range filtering.**

### 1.1 Domain type (source of truth)

```
internal/domain/strategy/strategy.go → Strategy struct
```

| Field        | Go type             | JSON            | Description                                      |
|------------- |-------------------- |---------------- |------------------------------------------------- |
| Type         | `string`            | `type`          | Strategy family (e.g., `mean_reversion_entry`)   |
| Source       | `string`            | `source`        | Data source identifier                           |
| Symbol       | `string`            | `symbol`        | Trading pair (e.g., `BTCUSD`)                    |
| Timeframe    | `int`               | `timeframe`     | Candle period in seconds                         |
| Direction    | `Direction(string)` | `direction`     | Positional intent: `long`, `short`, `flat`       |
| Confidence   | `string`            | `confidence`    | Decimal string (e.g., `"0.85"`)                  |
| Decisions    | `[]DecisionInput`   | `decisions`     | Contributing decision snapshots (JSON array)     |
| Parameters   | `map[string]string` | `parameters`    | Strategy-specific parameters (JSON object)       |
| Metadata     | `map[string]string` | `metadata`      | Arbitrary key-value metadata (JSON object)       |
| Final        | `bool`              | `final`         | Whether this is a final (not interim) resolution |
| Timestamp    | `time.Time`         | `timestamp`     | Resolution timestamp                             |

### 1.2 Payload shape — DecisionInput (nested type)

```go
type DecisionInput struct {
    Type       string `json:"type"`
    Outcome    string `json:"outcome"`
    Confidence string `json:"confidence"`
    Timeframe  int    `json:"timeframe"`
}
```

This is a strategy-owned type. It does not import from the decision domain.

### 1.3 JSON column inventory

Family 03 introduces **3 JSON columns** — the highest count so far in the analytical surface:

| Column       | Go source type      | ClickHouse type | JSON shape           | Parser needed             |
|------------- |-------------------- |---------------- |--------------------- |-------------------------- |
| `decisions`  | `[]DecisionInput`   | `String`        | Array of objects     | `ParseDecisionInputsJSON` (new) |
| `parameters` | `map[string]string` | `String`        | Object of strings    | `ParseMetadataJSON` (reuse)     |
| `metadata`   | `map[string]string` | `String`        | Object of strings    | `ParseMetadataJSON` (reuse)     |

**Pattern pressure**: This tests whether the read path scales to 3 JSON columns (up from 2 in decisions). The new parser `ParseDecisionInputsJSON` mirrors `ParseSignalInputsJSON` in structure — deserializing a JSON array of structs.

### 1.4 Enum-like filter: direction

Family 03 adds `direction` as an optional query filter, analogous to `outcome` in Family 02.

| Value   | Meaning                                  |
|-------- |----------------------------------------- |
| `long`  | Strategy resolved with bullish intent    |
| `short` | Strategy resolved with bearish intent    |
| `flat`  | Strategy resolved with neutral intent    |
| (empty) | No filter — return all directions        |

**Validation**: The reader does NOT validate direction values. ClickHouse handles the filter as a string equality check. Invalid direction values return empty results, which is correct behavior.

---

## 2. Schema mapping

### 2.1 DDL (migration 004_create_strategies.sql — already applied)

```sql
CREATE TABLE IF NOT EXISTS strategies (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from Strategy)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    direction      LowCardinality(String),
    confidence     Float64,
    decisions      String,        -- JSON: []DecisionInput
    parameters     String,        -- JSON: map[string]string
    metadata       String,        -- JSON: map[string]string
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
```

**Column count**: 15 domain columns (vs 14 in decisions) — healthy complexity increment.

### 2.2 Schema coherence table

| # | Domain field | DDL column     | DDL type                 | Writer value             | Reader scan type | Reader domain mapping     |
|---|------------- |--------------- |------------------------- |------------------------- |----------------- |-------------------------- |
| 1 | Type         | `type`         | `LowCardinality(String)` | `s.Type`                 | `string`         | direct                    |
| 2 | Source       | `source`       | `LowCardinality(String)` | `s.Source`               | `string`         | direct                    |
| 3 | Symbol       | `symbol`       | `LowCardinality(String)` | `s.Symbol`               | `string`         | direct                    |
| 4 | Timeframe    | `timeframe`    | `UInt32`                 | `uint32(s.Timeframe)`    | `uint32`         | `int(tf)`                 |
| 5 | Direction    | `direction`    | `LowCardinality(String)` | `string(s.Direction)`    | `string`         | `strategy.Direction(dir)` |
| 6 | Confidence   | `confidence`   | `Float64`                | `parseFloat(s.Confidence)` | `float64`     | `FormatFloat(confidence)` |
| 7 | Decisions    | `decisions`    | `String`                 | `marshalJSON(s.Decisions)` | `string`       | `ParseDecisionInputsJSON` |
| 8 | Parameters   | `parameters`   | `String`                 | `marshalJSON(s.Parameters)` | `string`     | `ParseMetadataJSON`       |
| 9 | Metadata     | `metadata`     | `String`                 | `marshalJSON(s.Metadata)` | `string`       | `ParseMetadataJSON`       |
| 10| Final        | `final`        | `Bool`                   | `s.Final`                | `bool`           | direct                    |
| 11| Timestamp    | `timestamp`    | `DateTime64(3)`          | `s.Timestamp`            | `time.Time`      | direct                    |

All 11 domain columns are type-aligned across DDL, writer mapper, and reader adapter.

---

## 3. Query contract

### 3.1 Request: StrategyHistoryQuery

```go
type StrategyHistoryQuery struct {
    Type      string `json:"type"`               // required — strategy type (e.g., "mean_reversion_entry")
    Source    string `json:"source"`              // required
    Symbol    string `json:"symbol"`              // required
    Timeframe int    `json:"timeframe"`           // required, positive
    Direction string `json:"direction,omitempty"` // optional filter (long, short, flat)
    Limit     int    `json:"limit"`               // default 50, max 500
    Since     int64  `json:"since,omitempty"`     // unix seconds, inclusive lower bound (0 = unset)
    Until     int64  `json:"until,omitempty"`     // unix seconds, inclusive upper bound (0 = unset)
}
```

### 3.2 Response: StrategyHistoryReply

```go
type StrategyHistoryReply struct {
    Strategies []strategy.Strategy `json:"strategies"`
    Source     string              `json:"source"` // always "clickhouse"
    Meta       QueryMeta           `json:"meta"`
}
```

### 3.3 HTTP endpoint

```
GET /analytical/strategy/history?type=...&source=...&symbol=...&timeframe=...&direction=...&since=...&until=...&limit=...
```

| Parameter   | Required | Type   | Default | Constraint     |
|------------ |--------- |------- |-------- |--------------- |
| `type`      | yes      | string | —       | non-empty      |
| `source`    | yes      | string | —       | non-empty      |
| `symbol`    | yes      | string | —       | non-empty      |
| `timeframe` | yes      | int    | —       | > 0            |
| `direction` | no       | string | (all)   | empty OK       |
| `limit`     | no       | int    | 50      | 1–500          |
| `since`     | no       | int64  | 0       | unix seconds   |
| `until`     | no       | int64  | 0       | unix seconds   |

**Response shape**:
```json
{
  "strategies": [ ... ],
  "source": "clickhouse",
  "meta": { "query_ms": 12, "row_count": 5 }
}
```

**Headers**: `Server-Timing: total;dur=15, query;dur=12`

**Error responses**: Standard `problem` JSON format, same codes as candle/signal/decision endpoints.

---

## 4. Simplifications and explicit limits

1. **No direction validation at query level** — invalid direction values return empty results. This is consistent with outcome handling in Family 02.
2. **No aggregation** — no count-by-direction, no confidence distributions, no time bucketing.
3. **No drill-down** — the `decisions` JSON array is returned as-is; there is no join to the decisions table.
4. **No write-path changes** — `mapStrategyRow()` and pipeline entry already exist and are active. Zero writer modifications.
5. **No cross-family queries** — strategies are queried independently; no evidence→signal→decision→strategy chain queries.
6. **No confidence filtering** — confidence is returned but not filterable (matches decision family behavior).
7. **JSON columns are scanned as strings** — parsed client-side by the reader adapter. No ClickHouse JSON functions.
8. **TTL 90 days** — inherited from migration 004; not configurable at query time.
