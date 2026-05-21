# ClickHouse Core Tables and DDL Rationale

> **Stage:** S144 — Core Analytical Schema Design
> **Status:** Definitive
> **Scope:** Rationale and design decisions for each core table.

---

## 1. Purpose

This document explains **why** each table exists, **why** each column was included or excluded, and **why** the engine, partitioning, ordering, and retention were chosen as specified. It is the companion rationale to `clickhouse-core-schema-design.md`.

---

## 2. Table-by-Table Rationale

### 2.1 evidence_candles

**Why this table exists:**
Candles are the foundational evidence of Market Foundry. Every signal, decision, strategy, risk assessment, and execution traces back to candle data. Without historical candles, no backtesting, no trend analysis, and no cold-start bootstrap is possible.

**Why these columns:**

| Column | Why Included | Why This Type |
|--------|-------------|---------------|
| `open`, `high`, `low`, `close` | Core OHLC data — the primary analytical payload | `Float64`: enables arithmetic in ClickHouse queries (averages, ranges, comparisons). Go strings would require per-query casting. |
| `volume` | Trading volume is essential for volume-weighted analysis | `Float64`: same rationale as OHLC |
| `trade_count` | Needed to distinguish low-activity candles from high-activity ones | `Int64`: exact integer, no precision concern |
| `open_time`, `close_time` | Candle window boundaries — essential for time-series alignment | `DateTime64(3)`: millisecond precision matches Go's `time.Time` |
| `final` | Distinguishes closed (immutable) candles from interim updates | `Bool`: the writer will receive both final and non-final candles. Queries should typically filter `WHERE final = true` for clean analysis. |
| `timeframe` | Window duration in seconds (60, 300, 900, 3600) | `UInt32`: unsigned, small range, used in partition key |
| `source`, `symbol` | Multi-source, multi-symbol filtering | `LowCardinality(String)`: bounded cardinality (currently 1 source, 2 symbols) |

**Why NOT these columns:**

| Excluded | Why |
|----------|-----|
| `event_name` / `event_type` | All rows in this table are candle events. The table itself is the type discriminator. |
| Derived columns (e.g., `range = high - low`) | Computable at query time. Storing derived values couples schema to specific queries. |
| `schema_version` | Deferred — event schema versioning is out of scope for the first wave (see schema versioning doc). |

**Why this partitioning — `(timeframe, toYYYYMM(open_time))`:**

Candle queries have a very strong access pattern: "give me 1-minute candles for BTCUSDT in March 2026." Partitioning by `(timeframe, month)` means ClickHouse can prune to the exact timeframe × month partition, skipping all other timeframes and months. This is the highest-leverage partition key for candle data.

Alternative considered: `toYYYYMM(open_time)` only. Rejected because queries without timeframe filtering would scan all timeframes within a month, which is wasteful when the Foundry runs 4 timeframes.

**Why this ordering — `(source, symbol, timeframe, open_time)`:**

The ordering key determines how data is physically sorted within each partition. This ordering means that within a `(timeframe=60, 2026-03)` partition, rows are sorted by source → symbol → time. Since source is currently single-valued and symbol has 2 values, the effective sort is by symbol then time, which is optimal for range scans.

**Why 90-day TTL:**

At current scale (2 symbols × 4 timeframes), the candle volume is approximately:
- 1-minute: 2 × 1440/day × 90 = ~260K rows
- 5-minute: 2 × 288/day × 90 = ~52K rows
- 15-minute: 2 × 96/day × 90 = ~17K rows
- 1-hour: 2 × 24/day × 90 = ~4K rows
- **Total: ~333K rows over 90 days**

This is trivially small for ClickHouse. 90 days provides meaningful backtesting depth without unbounded growth.

---

### 2.2 signals

**Why this table exists:**
Signals are the first derivative of evidence. Tracking signal history enables: evaluating whether RSI is a useful predictor, comparing signal values before and after parameter changes, and correlating signals with eventual executions.

**Why `type` as LowCardinality(String) in ORDER BY:**
The `signals` table holds multiple signal types (RSI, EMA crossover, future types). Including `type` in the ordering key means queries filtering by type benefit from data locality. Without it, RSI and EMA rows would be interleaved, forcing ClickHouse to scan both for a type-specific query.

**Why `metadata` as String (JSON) instead of typed columns:**
The `Metadata map[string]string` in the Go struct contains type-specific fields:
- RSI: `{"period": "14", "avg_gain": "0.523", "avg_loss": "0.312"}`
- EMA crossover: `{"fast_period": "12", "slow_period": "26", "fast_ema": "...", "slow_ema": "..."}`

Flattening these into typed columns would mean either:
- A wide table with many NULLable columns (one per signal type's fields) — schema explosion
- Separate tables per signal type — premature fragmentation

JSON string is the pragmatic choice. ClickHouse's `JSONExtractFloat64('metadata', 'avg_gain')` allows ad-hoc querying when needed. If a specific signal type becomes heavily queried, a materialized column can be added via migration.

**Why `value` as Float64 instead of String:**
The primary signal value (`Value string` in Go) is the single most queried field: "RSI over time", "signal vs. threshold". Storing as Float64 enables native ClickHouse comparisons (`WHERE value < 30`) without per-query parsing. The precision trade-off is documented in the core schema design.

---

### 2.3 decisions

**Why this table exists:**
Decisions are the evaluation layer — they determine whether signals justify action. Tracking decisions enables: measuring hit rate (how often "triggered" leads to profitable executions), identifying false positives, and tuning decision thresholds.

**Why `outcome` as LowCardinality(String):**
`Outcome` is a Go string enum with exactly 3 values: `triggered`, `not_triggered`, `insufficient`. LowCardinality encoding is ideal for this cardinality. The column is highly filterable — "show all triggered decisions" is a foundational analytical query.

**Why `confidence` as Float64 (not String):**
Confidence values are compared numerically: "decisions with confidence > 0.7 that were triggered." Storing as Float64 enables this directly.

**Why `signals` as String (JSON array):**
Each decision references 1+ signal inputs. The `[]SignalInput` is a heterogeneous slice — signal types, values, and timeframes vary. Storing as JSON array preserves the full input context without forcing a fixed number of signal columns.

Alternative considered: a join table (`decision_signal_inputs`). Rejected — this is an analytical store, not an OLTP database. Denormalized JSON in a single table is faster to query (no joins) and simpler to ingest.

---

### 2.4 strategies

**Why this table exists:**
Strategies are the tactical resolution of decisions — they determine direction and sizing. Tracking strategies enables: measuring strategy effectiveness (long entries that resulted in fills vs. those that didn't), comparing strategy confidence distributions, and identifying strategy types that underperform.

**Why `direction` as LowCardinality(String):**
`Direction` is a 3-value enum: `long`, `short`, `flat`. It's the most natural analytical axis for strategy evaluation: "show all long entries this week."

**Why both `parameters` and `metadata` as separate String columns:**
The Go struct distinguishes `Parameters map[string]string` (strategy configuration: entry thresholds, position sizing) from `Metadata map[string]string` (runtime context: trigger details). Preserving this distinction in the schema allows future queries to differentiate between "what the strategy was configured to do" and "what happened at runtime."

Alternative considered: merging into a single JSON column. Rejected — losing the semantic distinction would make downstream analysis harder if we ever need to compare "same parameters, different metadata" or vice versa.

---

### 2.5 risk_assessments

**Why this table exists:**
Risk assessments gate execution — they determine whether a strategy's intent should proceed. Tracking risk history enables: measuring rejection rate (how often risk blocks strategies), identifying whether risk constraints are too conservative or too permissive, and auditing the risk→execution chain.

**Why `disposition` as LowCardinality(String):**
`Disposition` is a 3-value enum: `approved`, `modified`, `rejected`. This is the primary analytical axis: "what fraction of strategies were approved?" "did rejection rate change after parameter tuning?"

**Why `constraints` as String (JSON) instead of flattened columns:**
The `Constraints` struct has 3 optional fields (`MaxPositionSize`, `MaxExposure`, `StopDistance`). Flattening would require 3 `Nullable(Float64)` columns that are empty when the constraint wasn't applied. At current scale, risk queries don't justify this complexity.

If constraint-level analysis becomes important (e.g., "what's the distribution of max_position_size for approved assessments?"), a future migration can add materialized columns:
```sql
ALTER TABLE risk_assessments
    ADD COLUMN max_position_size Float64
    MATERIALIZED JSONExtractFloat64(constraints, 'max_position_size');
```

**Why `rationale` as String:**
Free text. Provides human-readable justification. Not indexed, not LowCardinality (unbounded content). Useful for auditing, not for filtering.

---

### 2.6 executions

**Why this table exists:**
Executions are the end of the analytical chain — they represent actions taken. Tracking executions enables: building a trade journal, measuring fill quality, calculating P&L, and tracing the full event chain from candle to fill.

**Why `side` and `status` as LowCardinality(String):**
Both are bounded enums:
- `side`: 3 values (`buy`, `sell`, `none`)
- `status`: 7 values (`submitted`, `sent`, `accepted`, `filled`, `partially_filled`, `rejected`, `cancelled`)

Both are primary analytical axes: "show all buys this week", "count rejected orders."

**Why `quantity` and `filled_quantity` as Float64 (not String):**
These values are compared and aggregated: "total filled quantity per day", "fill rate = filled_quantity / quantity." Float64 enables native arithmetic.

**Why `fills` as String (JSON array) instead of a dedicated table:**
At current scale (paper trading), each execution has 0–1 fills. A dedicated `fills` table would add schema complexity for a 1:1 relationship. If execution volume grows to real trading with partial fills (1:N), a dedicated `fills` table should be introduced.

**Why `exec_correlation_id` and `exec_causation_id` (separate from metadata):**
The `ExecutionIntent` struct carries its own correlation/causation IDs for order lifecycle tracking, distinct from the `events.Metadata` correlation/causation IDs which track event causality. Both pairs are preserved:
- `correlation_id` + `causation_id`: "which event chain produced this execution event?"
- `exec_correlation_id` + `exec_causation_id`: "which order lifecycle does this execution belong to?"

**Why `risk` as String (JSON):**
The `RiskInput` struct embedded in `ExecutionIntent` has 4 fields (type, disposition, confidence, timeframe). Storing as JSON preserves the reference without creating a join dependency on the `risk_assessments` table. The writer simply serializes what the event carries.

---

## 3. Cross-Cutting Rationale

### 3.1 Why MergeTree (Not ReplacingMergeTree)

The writer may insert duplicate rows if it replays events from NATS after a restart. `ReplacingMergeTree` would deduplicate based on a version column, which seems attractive.

**Why MergeTree is still correct:**
- Deduplication in ReplacingMergeTree is **eventual** — it happens during merges, not at insert time. Queries can still return duplicates between merges.
- ReplacingMergeTree requires choosing a "version" column, which adds semantic complexity (is it `event_id`? `occurred_at`?).
- At current scale (paper trading, single developer), duplicates are rare and tolerable.
- `SELECT DISTINCT` or `GROUP BY event_id` at query time is simpler and more explicit.
- If deduplication becomes important, a migration to ReplacingMergeTree is straightforward:
  ```sql
  -- Future migration (NOT part of this schema)
  CREATE TABLE evidence_candles_new (...) ENGINE = ReplacingMergeTree(occurred_at) ...;
  INSERT INTO evidence_candles_new SELECT * FROM evidence_candles;
  RENAME TABLE evidence_candles TO evidence_candles_old, evidence_candles_new TO evidence_candles;
  ```

### 3.2 Why No Primary Key (Separate from ORDER BY)

In ClickHouse, the PRIMARY KEY defaults to the ORDER BY key. A separate PRIMARY KEY is used when you want a shorter prefix for the sparse index while keeping a more detailed sort order. At current data volumes, the full ORDER BY as PRIMARY KEY is fine — sparse index memory is negligible.

### 3.3 Why No Codec Specifications

ClickHouse applies default compression (LZ4) to all columns. Column-specific codecs (Delta, DoubleDelta, Gorilla for Float64) are optimizations that matter at scale. At current data volumes (~333K candle rows over 90 days), the default is more than adequate. Codec tuning is a future optimization.

### 3.4 Why String DEFAULT '' (Not Nullable)

`correlation_id` and `causation_id` use `DEFAULT ''` rather than `Nullable(String)`.

**Rationale:**
- `Nullable` adds a separate null bitmap per column, increasing storage overhead
- Empty string semantics are clear: "no correlation ID was set"
- ClickHouse best practice: avoid Nullable unless truly needed (three-valued logic)
- Filtering: `WHERE correlation_id != ''` is simpler than `WHERE correlation_id IS NOT NULL`

### 3.5 Why toYYYYMM (Not toYYYYMMDD)

Monthly partitions keep partition count low. At 2 symbols × 4 timeframes, daily partitions would create 120 partitions/month (30 days × 4 timeframes for evidence_candles) vs. 4 (1 month × 4 timeframes). ClickHouse recommends fewer, larger partitions for merge efficiency.

If data volume grows significantly (10+ symbols, multiple sources), daily partitioning can be introduced via `ALTER TABLE ... MODIFY PARTITION BY`.

---

## 4. Decisions Deferred to Future Migrations

| Decision | Why Deferred | Trigger to Revisit |
|----------|-------------|-------------------|
| `Decimal128` for price columns | Unnecessary precision for paper trading | Real money execution |
| Column codecs (Delta, Gorilla) | Negligible impact at current volume | Data volume > 10M rows/table |
| ReplacingMergeTree for dedup | Duplicates are rare and tolerable | Observed duplicate rate > 1% |
| Separate `fills` table | Paper trading has 0–1 fills per execution | Real trading with partial fills |
| `evidence_tradebursts` table | Not on core pipeline path | Active trade burst analysis need |
| `evidence_volumes` table | Not on core pipeline path | Active volume profile analysis need |
| Materialized columns for JSON fields | No established query patterns yet | Repeated JSON extraction queries |
| Projections (ClickHouse projections) | Optimization for alternative sort orders | Query surface reveals hot paths |
