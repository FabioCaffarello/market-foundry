# Core ClickHouse Migrations and Activation Proof

## Purpose

This document records the canonical DDL for the 6 core analytical tables, the corrections applied during activation, and the proof that the schema evolution mechanism works as designed.

## Migration Catalog (Final State)

| Version | Migration | Table | Status |
|---------|-----------|-------|--------|
| 000 | `000_create_migrations_metadata` | `_migrations` | Applied |
| 001 | `001_create_evidence_candles` | `evidence_candles` | Applied |
| 002 | `002_create_signals` | `signals` | Applied |
| 003 | `003_create_decisions` | `decisions` | Applied |
| 004 | `004_create_strategies` | `strategies` | Applied |
| 005 | `005_create_risk_assessments` | `risk_assessments` | Applied |
| 006 | `006_create_executions` | `executions` | Applied |

## Schema Alignment with S144 Design

The S146 migration files were draft implementations that diverged significantly from the S144 canonical schema design. S147 rewrote all 6 migration files (001–006) to match the S144 design exactly before first application.

### Corrections Applied

**All 6 tables — missing columns restored:**

| Column | Type | Present In |
|--------|------|-----------|
| `source` | `LowCardinality(String)` | All 6 tables |
| `final` | `Bool` | All 6 tables |

**All 6 tables — ORDER BY corrected:**

| Before (S146) | After (S147, matching S144) |
|---------------|---------------------------|
| `(symbol, timeframe, ...)` | `(source, symbol, timeframe, ...)` |

**Per-table corrections:**

| Table | Missing Columns Restored |
|-------|------------------------|
| `decisions` | `confidence` (Float64), `metadata` (String) |
| `strategies` | `confidence` (Float64), `decisions` (String) |
| `risk_assessments` | `confidence` (Float64), `strategies` (String), `rationale` (String), `parameters` (String), `metadata` (String) |
| `executions` | `filled_quantity` (Float64, was incorrectly `price`), `risk` (String), `parameters` (String), `metadata` (String) |

**executions — TTL corrected:** S146 used 365 DAY; S144 design specifies 90 DAY uniformly for all core tables.

### TTL Runtime Fix

ClickHouse 24.8.8 rejects TTL expressions on `DateTime64` columns directly. The fix:

```sql
-- Fails: TTL open_time + INTERVAL 90 DAY
-- Works: TTL toDateTime(open_time) + INTERVAL 90 DAY
```

All migrations use `toDateTime(<column>) + INTERVAL 90 DAY`. The subsecond precision loss in the TTL trigger is negligible — TTL granularity is at the part level (hours/days), not at the millisecond level.

## Column Mapping to Go Structs

### evidence_candles ← EvidenceCandle + events.Metadata

```
event_id       ← Metadata.ID
occurred_at    ← Metadata.OccurredAt
correlation_id ← Metadata.CorrelationID
causation_id   ← Metadata.CausationID
source         ← Candle.Source         (LowCardinality)
symbol         ← Candle.Symbol         (LowCardinality)
timeframe      ← Candle.Timeframe      (UInt32 from int)
open           ← Candle.Open           (Float64 from decimal string)
high           ← Candle.High           (Float64 from decimal string)
low            ← Candle.Low            (Float64 from decimal string)
close          ← Candle.Close          (Float64 from decimal string)
volume         ← Candle.Volume         (Float64 from decimal string)
trade_count    ← Candle.TradeCount     (Int64)
open_time      ← Candle.OpenTime       (DateTime64(3))
close_time     ← Candle.CloseTime      (DateTime64(3))
final          ← Candle.Final          (Bool)
ingested_at    ← now64(3)              (ClickHouse DEFAULT)
```

### signals ← Signal + events.Metadata

```
type           ← Signal.Type           (LowCardinality)
source         ← Signal.Source         (LowCardinality)
symbol         ← Signal.Symbol         (LowCardinality)
timeframe      ← Signal.Timeframe      (UInt32)
value          ← Signal.Value          (Float64 from decimal string)
metadata       ← Signal.Metadata       (JSON from map[string]string)
final          ← Signal.Final          (Bool)
timestamp      ← Signal.Timestamp      (DateTime64(3))
```

### decisions ← Decision + events.Metadata

```
type           ← Decision.Type         (LowCardinality)
source         ← Decision.Source       (LowCardinality)
symbol         ← Decision.Symbol       (LowCardinality)
timeframe      ← Decision.Timeframe    (UInt32)
outcome        ← Decision.Outcome      (LowCardinality: triggered/not_triggered/insufficient)
confidence     ← Decision.Confidence   (Float64 from decimal string)
signals        ← Decision.Signals      (JSON from []SignalInput)
metadata       ← Decision.Metadata     (JSON from map[string]string)
final          ← Decision.Final        (Bool)
timestamp      ← Decision.Timestamp    (DateTime64(3))
```

### strategies ← Strategy + events.Metadata

```
type           ← Strategy.Type         (LowCardinality)
source         ← Strategy.Source       (LowCardinality)
symbol         ← Strategy.Symbol       (LowCardinality)
timeframe      ← Strategy.Timeframe    (UInt32)
direction      ← Strategy.Direction    (LowCardinality: long/short/flat)
confidence     ← Strategy.Confidence   (Float64 from decimal string)
decisions      ← Strategy.Decisions    (JSON from []DecisionInput)
parameters     ← Strategy.Parameters   (JSON from map[string]string)
metadata       ← Strategy.Metadata     (JSON from map[string]string)
final          ← Strategy.Final        (Bool)
timestamp      ← Strategy.Timestamp    (DateTime64(3))
```

### risk_assessments ← RiskAssessment + events.Metadata

```
type           ← RiskAssessment.Type        (LowCardinality)
source         ← RiskAssessment.Source      (LowCardinality)
symbol         ← RiskAssessment.Symbol      (LowCardinality)
timeframe      ← RiskAssessment.Timeframe   (UInt32)
disposition    ← RiskAssessment.Disposition (LowCardinality: approved/modified/rejected)
confidence     ← RiskAssessment.Confidence  (Float64 from decimal string)
strategies     ← RiskAssessment.Strategies  (JSON from []StrategyInput)
constraints    ← RiskAssessment.Constraints (JSON from Constraints struct)
rationale      ← RiskAssessment.Rationale   (String)
parameters     ← RiskAssessment.Parameters  (JSON from map[string]string)
metadata       ← RiskAssessment.Metadata    (JSON from map[string]string)
final          ← RiskAssessment.Final       (Bool)
timestamp      ← RiskAssessment.Timestamp   (DateTime64(3))
```

### executions ← ExecutionIntent + events.Metadata

```
type                ← ExecutionIntent.Type              (LowCardinality)
source              ← ExecutionIntent.Source            (LowCardinality)
symbol              ← ExecutionIntent.Symbol            (LowCardinality)
timeframe           ← ExecutionIntent.Timeframe         (UInt32)
side                ← ExecutionIntent.Side              (LowCardinality: buy/sell/none)
quantity            ← ExecutionIntent.Quantity          (Float64 from decimal string)
filled_quantity     ← ExecutionIntent.FilledQuantity    (Float64 from decimal string)
status              ← ExecutionIntent.Status            (LowCardinality: submitted/sent/accepted/filled/partially_filled/rejected/cancelled)
risk                ← ExecutionIntent.Risk              (JSON from RiskInput struct)
fills               ← ExecutionIntent.Fills             (JSON from []FillRecord)
parameters          ← ExecutionIntent.Parameters        (JSON from map[string]string)
metadata            ← ExecutionIntent.Metadata          (JSON from map[string]string)
exec_correlation_id ← ExecutionIntent.CorrelationID     (domain-level, distinct from event metadata)
exec_causation_id   ← ExecutionIntent.CausationID       (domain-level, distinct from event metadata)
final               ← ExecutionIntent.Final             (Bool)
timestamp           ← ExecutionIntent.Timestamp         (DateTime64(3))
```

## Activation Proof Summary

| Test | Result |
|------|--------|
| `migrate up` — apply all 7 migrations | All OK |
| `migrate status` — 7 applied, 0 pending | Confirmed |
| `migrate validate` — all checksums valid | Confirmed |
| `migrate up` (re-run) — idempotent | "no pending migrations" |
| `SHOW TABLES` — 7 tables exist | Confirmed |
| `SHOW CREATE TABLE` — DDL matches S144 | Confirmed |
| Sample INSERT into all 6 domain tables | All succeed |
| Test data TRUNCATE | Clean state restored |

## Writer Readiness

The schema is ready to receive data from `cmd/writer`. The writer must:

1. Parse `events.Metadata` fields → `event_id`, `occurred_at`, `correlation_id`, `causation_id`
2. Parse decimal strings → `Float64`
3. Serialize `map[string]string` and nested structs → JSON strings
4. Pass enum strings directly (ClickHouse accepts any `LowCardinality(String)` value)
5. Not set `ingested_at` — ClickHouse DEFAULT handles it

All INSERT column lists are explicit (no `INSERT INTO table VALUES`), so the writer controls which columns it populates.
