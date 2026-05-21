# ClickHouse Core Schema Design

> **Stage:** S144 — Core Analytical Schema Design
> **Status:** Definitive
> **Scope:** Schema design only. No implementation.

---

## 1. Purpose

This document defines the core analytical schema for Market Foundry's ClickHouse entry. It specifies the 6 event tables that form the minimum viable analytical layer, their column mappings, ordering, partitioning, retention, and the rules that govern their design.

The schema is **event-oriented, not query-oriented**. Each table is a direct projection of a Go domain event struct into a ClickHouse MergeTree table. No materialized views, no pre-aggregations, no query-driven denormalization.

---

## 2. Design Principles

| # | Principle | Application |
|---|-----------|-------------|
| DP-01 | Schema follows events | Every column maps to a Go struct field. No invented columns except `ingested_at`. |
| DP-02 | Small and canonical | 6 tables only. No supplementary evidence tables, no telemetry, no fills. |
| DP-03 | Flat over nested | Nested Go structs (arrays, maps) stored as JSON strings. No premature flattening. |
| DP-04 | Queryable primitives | Scalar fields that are likely filter/group targets use typed columns, not JSON. |
| DP-05 | LowCardinality for enums | String fields with bounded value sets use `LowCardinality(String)`. |
| DP-06 | Float64 for decimals | Go decimal strings → ClickHouse Float64. Acceptable precision loss for paper trading analytics. |
| DP-07 | Uniform metadata | All tables carry `event_id`, `occurred_at`, `correlation_id`, `causation_id`, `ingested_at`. |

---

## 3. The 6 Core Tables

The tables follow the canonical pipeline stages:

```
evidence_candles → signals → decisions → strategies → risk_assessments → executions
```

Each table captures one event family from the pipeline. Together they form a complete audit trail of the analytical loop.

### 3.1 Table Summary

| # | Table | Go Source | NATS Stream | Pipeline Stage |
|---|-------|-----------|-------------|----------------|
| 1 | `evidence_candles` | `evidence.CandleSampledEvent` | EVIDENCE_EVENTS | ingest |
| 2 | `signals` | `signal.SignalGeneratedEvent` | signal.* | derive |
| 3 | `decisions` | `decision.DecisionEvaluatedEvent` | decision.* | derive |
| 4 | `strategies` | `strategy.StrategyResolvedEvent` | strategy.* | derive |
| 5 | `risk_assessments` | `risk.RiskAssessedEvent` | risk.* | derive |
| 6 | `executions` | `execution.PaperOrderSubmittedEvent` / `VenueOrderFilledEvent` | execution.* | execute |

---

## 4. Table Definitions

### 4.1 evidence_candles

**Purpose:** Historical archive of closed candle events. Primary evidence for backtesting and trend analysis.

**Go source:** `internal/domain/evidence/candle.go` → `EvidenceCandle` + `events.Metadata`

```sql
CREATE TABLE IF NOT EXISTS evidence_candles (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from EvidenceCandle)
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    open           Float64,
    high           Float64,
    low            Float64,
    close          Float64,
    volume         Float64,
    trade_count    Int64,
    open_time      DateTime64(3),
    close_time     DateTime64(3),
    final          Bool,

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)
TTL open_time + INTERVAL 90 DAY;
```

**Column mapping:**

| Go Field | Go Type | CH Column | CH Type | Notes |
|----------|---------|-----------|---------|-------|
| `Metadata.ID` | `string` | `event_id` | `String` | Hex-encoded 16-byte random |
| `Metadata.OccurredAt` | `time.Time` | `occurred_at` | `DateTime64(3)` | Event emission time |
| `Metadata.CorrelationID` | `string` | `correlation_id` | `String` | Empty if unset |
| `Metadata.CausationID` | `string` | `causation_id` | `String` | Empty if unset |
| `Source` | `string` | `source` | `LowCardinality(String)` | e.g. "binancef" |
| `Symbol` | `string` | `symbol` | `LowCardinality(String)` | e.g. "btcusdt" |
| `Timeframe` | `int` | `timeframe` | `UInt32` | Seconds (60, 300, 900, 3600) |
| `Open` | `string` (decimal) | `open` | `Float64` | Price — precision trade-off accepted |
| `High` | `string` (decimal) | `high` | `Float64` | |
| `Low` | `string` (decimal) | `low` | `Float64` | |
| `Close` | `string` (decimal) | `close` | `Float64` | |
| `Volume` | `string` (decimal) | `volume` | `Float64` | |
| `TradeCount` | `int64` | `trade_count` | `Int64` | |
| `OpenTime` | `time.Time` | `open_time` | `DateTime64(3)` | Window start |
| `CloseTime` | `time.Time` | `close_time` | `DateTime64(3)` | Window end |
| `Final` | `bool` | `final` | `Bool` | True = immutable candle |

**Partitioning rationale:** `(timeframe, toYYYYMM(open_time))` — candle queries are almost always scoped to a specific timeframe and time range. Partitioning by both ensures efficient pruning.

**Ordering rationale:** `(source, symbol, timeframe, open_time)` — the natural query pattern is "give me candles for symbol X at timeframe Y from source Z ordered by time."

---

### 4.2 signals

**Purpose:** Historical archive of computed signals (RSI, EMA crossover, etc.). Enables signal quality evaluation over time.

**Go source:** `internal/domain/signal/signal.go` → `Signal` + `events.Metadata`

```sql
CREATE TABLE IF NOT EXISTS signals (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from Signal)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    value          Float64,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL timestamp + INTERVAL 90 DAY;
```

**Column mapping:**

| Go Field | Go Type | CH Column | CH Type | Notes |
|----------|---------|-----------|---------|-------|
| `Type` | `string` | `type` | `LowCardinality(String)` | e.g. "rsi", "ema_crossover" |
| `Source` | `string` | `source` | `LowCardinality(String)` | |
| `Symbol` | `string` | `symbol` | `LowCardinality(String)` | |
| `Timeframe` | `int` | `timeframe` | `UInt32` | |
| `Value` | `string` (decimal) | `value` | `Float64` | Primary signal output |
| `Metadata` | `map[string]string` | `metadata` | `String` | JSON-encoded map (e.g. `{"period":"14","avg_gain":"0.5"}`) |
| `Final` | `bool` | `final` | `Bool` | |
| `Timestamp` | `time.Time` | `timestamp` | `DateTime64(3)` | Signal computation time |

**Partitioning rationale:** `toYYYYMM(timestamp)` — signals don't have a natural timeframe partition dimension for queries (unlike candles). Monthly partitioning is sufficient.

**Ordering rationale:** `(source, symbol, timeframe, type, timestamp)` — signal queries typically filter by symbol + timeframe + type, then scan by time.

**`metadata` as String:** The `map[string]string` contains type-specific fields (RSI has `period`, `avg_gain`, `avg_loss`; EMA crossover has different keys). Storing as JSON string avoids schema explosion per signal type. ClickHouse JSON functions (`JSONExtractString`) allow querying if needed.

---

### 4.3 decisions

**Purpose:** Historical archive of decision evaluations. Enables decision accuracy analysis.

**Go source:** `internal/domain/decision/decision.go` → `Decision` + `events.Metadata`

```sql
CREATE TABLE IF NOT EXISTS decisions (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from Decision)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    outcome        LowCardinality(String),
    confidence     Float64,
    signals        String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL timestamp + INTERVAL 90 DAY;
```

**Column mapping:**

| Go Field | Go Type | CH Column | CH Type | Notes |
|----------|---------|-----------|---------|-------|
| `Type` | `string` | `type` | `LowCardinality(String)` | e.g. "rsi_oversold" |
| `Outcome` | `Outcome` (string enum) | `outcome` | `LowCardinality(String)` | "triggered" / "not_triggered" / "insufficient" |
| `Confidence` | `string` (decimal) | `confidence` | `Float64` | |
| `Signals` | `[]SignalInput` | `signals` | `String` | JSON array: `[{"type":"rsi","value":"28.5","timeframe":60}]` |
| `Metadata` | `map[string]string` | `metadata` | `String` | JSON-encoded map |

**`outcome` as LowCardinality:** Bounded enum (3 values). Highly filterable — "show me all triggered decisions" is a natural query.

---

### 4.4 strategies

**Purpose:** Historical archive of strategy resolutions. Enables strategy effectiveness analysis.

**Go source:** `internal/domain/strategy/strategy.go` → `Strategy` + `events.Metadata`

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
    decisions      String,
    parameters     String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL timestamp + INTERVAL 90 DAY;
```

**Column mapping:**

| Go Field | Go Type | CH Column | CH Type | Notes |
|----------|---------|-----------|---------|-------|
| `Type` | `string` | `type` | `LowCardinality(String)` | e.g. "mean_reversion" |
| `Direction` | `Direction` (string enum) | `direction` | `LowCardinality(String)` | "long" / "short" / "flat" |
| `Confidence` | `string` (decimal) | `confidence` | `Float64` | |
| `Decisions` | `[]DecisionInput` | `decisions` | `String` | JSON array |
| `Parameters` | `map[string]string` | `parameters` | `String` | JSON-encoded map |
| `Metadata` | `map[string]string` | `metadata` | `String` | JSON-encoded map |

**`direction` as LowCardinality:** Bounded enum (3 values). "Show me all long entries" is a natural query.

---

### 4.5 risk_assessments

**Purpose:** Historical archive of risk assessments. Enables risk policy evaluation.

**Go source:** `internal/domain/risk/risk.go` → `RiskAssessment` + `events.Metadata`

```sql
CREATE TABLE IF NOT EXISTS risk_assessments (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from RiskAssessment)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    disposition    LowCardinality(String),
    confidence     Float64,
    strategies     String,
    constraints    String,
    rationale      String,
    parameters     String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL timestamp + INTERVAL 90 DAY;
```

**Column mapping:**

| Go Field | Go Type | CH Column | CH Type | Notes |
|----------|---------|-----------|---------|-------|
| `Type` | `string` | `type` | `LowCardinality(String)` | e.g. "position_exposure" |
| `Disposition` | `Disposition` (string enum) | `disposition` | `LowCardinality(String)` | "approved" / "modified" / "rejected" |
| `Confidence` | `string` (decimal) | `confidence` | `Float64` | |
| `Strategies` | `[]StrategyInput` | `strategies` | `String` | JSON array |
| `Constraints` | `Constraints` struct | `constraints` | `String` | JSON object: `{"max_position_size":"0.1","max_exposure":"1000"}` |
| `Rationale` | `string` | `rationale` | `String` | Free text |
| `Parameters` | `map[string]string` | `parameters` | `String` | JSON-encoded map |
| `Metadata` | `map[string]string` | `metadata` | `String` | JSON-encoded map |

**`constraints` as String:** The `Constraints` struct has 3 optional fields. Storing as JSON avoids 3 Nullable columns for a struct that may evolve. If constraint-level queries become important, a future migration can add materialized columns.

---

### 4.6 executions

**Purpose:** Historical archive of execution intents (paper orders, venue orders). Enables execution quality analysis and trade journal.

**Go source:** `internal/domain/execution/execution.go` → `ExecutionIntent` + `events.Metadata`

```sql
CREATE TABLE IF NOT EXISTS executions (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from ExecutionIntent)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    side           LowCardinality(String),
    quantity       Float64,
    filled_quantity Float64,
    status         LowCardinality(String),
    risk           String,
    fills          String,
    parameters     String,
    metadata       String,
    exec_correlation_id String DEFAULT '',
    exec_causation_id   String DEFAULT '',
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL timestamp + INTERVAL 90 DAY;
```

**Column mapping:**

| Go Field | Go Type | CH Column | CH Type | Notes |
|----------|---------|-----------|---------|-------|
| `Type` | `string` | `type` | `LowCardinality(String)` | e.g. "paper_order" |
| `Side` | `Side` (string enum) | `side` | `LowCardinality(String)` | "buy" / "sell" / "none" |
| `Quantity` | `string` (decimal) | `quantity` | `Float64` | Intended quantity |
| `FilledQuantity` | `string` (decimal) | `filled_quantity` | `Float64` | Filled so far |
| `Status` | `Status` (string enum) | `status` | `LowCardinality(String)` | 7 values: submitted through cancelled |
| `Risk` | `RiskInput` struct | `risk` | `String` | JSON object |
| `Fills` | `[]FillRecord` | `fills` | `String` | JSON array |
| `CorrelationID` | `string` | `exec_correlation_id` | `String` | Domain-level (distinct from metadata) |
| `CausationID` | `string` | `exec_causation_id` | `String` | Domain-level (distinct from metadata) |

**Two correlation ID pairs:** The metadata-level IDs (`correlation_id`, `causation_id`) track event causality in the NATS envelope. The execution-level IDs (`exec_correlation_id`, `exec_causation_id`) track order lifecycle causality in the domain. Both are preserved.

**`fills` as String:** Fill records are nested objects with timestamps. Storing as JSON array keeps the schema simple. A dedicated `fills` table is explicitly deferred.

---

## 5. Common Patterns

### 5.1 Metadata Columns

Every table includes these 5 columns from the event envelope:

```sql
event_id       String,                    -- Metadata.ID (unique per event)
occurred_at    DateTime64(3),             -- Metadata.OccurredAt (event emission time)
correlation_id String DEFAULT '',         -- Metadata.CorrelationID (empty if unset)
causation_id   String DEFAULT '',         -- Metadata.CausationID (empty if unset)
ingested_at    DateTime64(3) DEFAULT now64(3)  -- ClickHouse insertion time
```

`ingested_at` is the only column not derived from the Go event. It records when the writer inserted the row, useful for diagnosing ingestion lag.

### 5.2 Engine

All 6 tables use `MergeTree()`. No `ReplacingMergeTree`, no `AggregatingMergeTree`, no replication engines.

**Rationale:** The writer inserts events as they arrive. There is no deduplication requirement (events have unique IDs), no aggregation at write time, and no replication (single node). MergeTree is the simplest and most predictable engine.

**Note on duplicates:** If the writer restarts and replays events from NATS, duplicate rows may be inserted. This is acceptable for the first wave — deduplication can be handled at query time (`WHERE final = true`) or via `ReplacingMergeTree` in a future migration if it becomes a problem.

### 5.3 Ordering Key Pattern

All tables follow one of two ordering patterns:

| Pattern | Tables | ORDER BY |
|---------|--------|----------|
| Evidence | `evidence_candles` | `(source, symbol, timeframe, open_time)` |
| Pipeline | `signals`, `decisions`, `strategies`, `risk_assessments`, `executions` | `(source, symbol, timeframe, type, timestamp)` |

The pipeline pattern adds `type` because multiple event types share a single table (e.g., RSI and EMA signals both in `signals`).

### 5.4 Partitioning Pattern

| Pattern | Tables | PARTITION BY |
|---------|--------|-------------|
| Evidence | `evidence_candles` | `(timeframe, toYYYYMM(open_time))` |
| Pipeline | all others | `toYYYYMM(timestamp)` |

Evidence tables partition by timeframe because candle queries are almost always scoped to a specific timeframe. Pipeline tables don't benefit from timeframe partitioning because cross-timeframe analysis is a common pattern.

### 5.5 Retention

All 6 core tables: `TTL <timestamp_column> + INTERVAL 90 DAY`.

**Rationale:** 90 days is sufficient for iterating on strategy development at paper-trading scale. Longer retention can be set via a future migration (`ALTER TABLE ... MODIFY TTL`). The 90-day window provides approximately 3 months of backtesting data without unbounded disk growth.

### 5.6 Decimal String to Float64 Conversion

The Go domain uses `string` for all price/quantity/confidence values to preserve decimal precision. The ClickHouse schema uses `Float64`.

**Trade-off:** Float64 introduces floating-point representation error (e.g., `0.1` becomes `0.1000000000000000055511151231257827021181583404541015625`). This is acceptable because:
- The Foundry is a paper-trading and strategy-development system, not a settlement engine
- Analytical queries (averages, trends, comparisons) tolerate Float64 precision
- `Decimal128(18)` would preserve precision but adds complexity and performance cost
- If precision becomes critical, a migration can change column types to `Decimal128(18)`

The writer is responsible for parsing the Go decimal string to Float64 during insertion.

---

## 6. What Is NOT in This Schema

| Item | Why Excluded | When It Enters |
|------|-------------|----------------|
| `evidence_tradebursts` | Supplementary evidence, not on the core pipeline path | When trade burst analysis becomes an active analytical need |
| `evidence_volumes` | Supplementary evidence, not on the core pipeline path | When volume profile analysis becomes an active analytical need |
| `fills` (dedicated table) | Currently nested inside `executions` as JSON; insufficient volume to justify a table | When fill-level queries (slippage analysis, execution quality) demand first-class columns |
| `runtime_telemetry` | Operational, not domain; P2 priority; different ingestion pattern (scraper) | After core schema is proven; uses migration range 100–199 |
| `configctl_events` | Audit trail; P3 priority; very low volume | After P2 is delivered |
| Materialized views | Query optimization; needs query patterns first | After query surface extension (Phase 4) identifies hot paths |
| Pre-aggregated tables | Optimization; premature without data volume | After data volume justifies aggregation cost |

---

## 7. Migration File Mapping

The 6 tables map to migration files 001–006 in `deploy/migrations/`:

| Migration | File | Table |
|-----------|------|-------|
| 001 | `001_create_evidence_candles.sql` | `evidence_candles` |
| 002 | `002_create_signals.sql` | `signals` |
| 003 | `003_create_decisions.sql` | `decisions` |
| 004 | `004_create_strategies.sql` | `strategies` |
| 005 | `005_create_risk_assessments.sql` | `risk_assessments` |
| 006 | `006_create_executions.sql` | `executions` |

This follows the pipeline order (evidence → signals → decisions → strategies → risk → executions) and the reserved range 001–099 for core domain tables.

**Note:** S143 originally projected 9 DDL files (001–009) assuming 3 evidence tables + signals + decisions + strategies + risk + executions + fills. This design reduces to 6 because trade bursts, volumes, and fills are explicitly deferred.

---

## 8. Writer Responsibility Summary

The writer service (future `cmd/writer`) is responsible for:

1. **Deserializing** NATS events from JSON
2. **Parsing** decimal strings to Float64 values
3. **Serializing** nested structs (maps, arrays) to JSON strings for `String` columns
4. **Extracting** metadata fields (`event_id`, `occurred_at`, `correlation_id`, `causation_id`) from the event envelope
5. **Not setting** `ingested_at` (ClickHouse DEFAULT handles it)

The writer does NOT:
- Transform event structure
- Filter events (all events of each type are persisted)
- Deduplicate events (MergeTree accepts duplicates)
- Aggregate or pre-process data
