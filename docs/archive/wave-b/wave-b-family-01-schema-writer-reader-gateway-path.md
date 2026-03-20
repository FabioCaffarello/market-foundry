# Wave B Family-01 — Schema/Writer/Reader/Gateway Path (Signals)

## Schema Coherence Verification

The DDL, writer mapper, and reader adapter must be column-aligned. This section proves alignment.

### DDL Columns (`deploy/migrations/002_create_signals.sql`)

```sql
event_id       String,
occurred_at    DateTime64(3),
correlation_id String DEFAULT '',
causation_id   String DEFAULT '',
type           LowCardinality(String),
source         LowCardinality(String),
symbol         LowCardinality(String),
timeframe      UInt32,
value          Float64,
metadata       String,
final          Bool,
timestamp      DateTime64(3),
ingested_at    DateTime64(3) DEFAULT now64(3)
```

### Writer Mapper (`cmd/writer/mappers.go:mapSignalRow`)

```
event_id       ← m.ID           (String)
occurred_at    ← m.OccurredAt    (DateTime64)
correlation_id ← m.CorrelationID (String)
causation_id   ← m.CausationID   (String)
type           ← s.Type          (String → LowCardinality)
source         ← s.Source        (String → LowCardinality)
symbol         ← s.Symbol        (String → LowCardinality)
timeframe      ← uint32(s.Timeframe) (UInt32)
value          ← parseFloat(s.Value) (Float64)
metadata       ← marshalJSON(s.Metadata) (String)
final          ← s.Final         (Bool)
timestamp      ← s.Timestamp     (DateTime64)
```

### Reader Adapter (`internal/adapters/clickhouse/signal_reader.go`)

```
SELECT type, source, symbol, timeframe, value, metadata, final, timestamp
```

Scan targets:
```
type      → string   → Signal.Type
source    → string   → Signal.Source
symbol    → string   → Signal.Symbol
timeframe → uint32   → Signal.Timeframe (cast to int)
value     → float64  → FormatFloat → Signal.Value
metadata  → string   → ParseMetadataJSON → Signal.Metadata
final     → bool     → Signal.Final
timestamp → time.Time → Signal.Timestamp
```

### Coherence Result

| Column | DDL Type | Writer Go Type | Reader Go Type | Aligned |
|---|---|---|---|---|
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 | YES |
| value | Float64 | float64 (parseFloat) | float64 (FormatFloat) | YES |
| metadata | String | string (marshalJSON) | string (ParseMetadataJSON) | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |

**Schema coherence: VERIFIED** — all 8 domain columns are type-aligned across DDL, writer, and reader.

## Data Flow

```
NATS JetStream                 ClickHouse                    Gateway HTTP
─────────────                  ──────────                    ────────────
signal.events.rsi.generated
        │
        ▼
  writerConsumer
   (consumer.go)
        │
        ▼
   mapSignalRow()          INSERT INTO signals
   (mappers.go) ────────► (inserter.go batch)
                                    │
                                    ▼
                           ┌──────────────┐
                           │   signals    │
                           │  (MergeTree) │
                           └──────┬───────┘
                                  │
                                  ▼
                          QuerySignalHistory()          GET /analytical/signal/history
                          (signal_reader.go) ◄──────── (analytical.go handler)
                                  │                            │
                                  ▼                            ▼
                           []signal.Signal           JSON + Server-Timing
```

## Endpoint Specification

```
GET /analytical/signal/history
```

### Query Parameters

| Parameter | Required | Type | Default | Constraints |
|---|---|---|---|---|
| type | yes | string | — | Signal family (e.g., "rsi") |
| source | yes | string | — | Exchange identifier |
| symbol | yes | string | — | Trading pair |
| timeframe | yes | int | — | Must be positive |
| limit | no | int | 50 | 1–500 |
| since | no | int64 | 0 (unset) | Unix seconds, inclusive |
| until | no | int64 | 0 (unset) | Unix seconds, inclusive |

### Response (200 OK)

```json
{
  "signals": [
    {
      "type": "rsi",
      "source": "binancef",
      "symbol": "btcusdt",
      "timeframe": 60,
      "value": "32.5",
      "metadata": {"period": "14", "avg_gain": "1.5", "avg_loss": "2.3"},
      "final": true,
      "timestamp": "2026-03-19T12:00:00Z"
    }
  ],
  "source": "clickhouse",
  "meta": {
    "query_ms": 5,
    "row_count": 1
  }
}
```

### Headers

- `Server-Timing: total;dur=<ms>, query;dur=<ms>` — same observability as candle endpoint.

### Error Responses

| Status | Condition |
|---|---|
| 400 | Missing/invalid required parameter |
| 503 | ClickHouse unavailable or reader nil |

## Writer Pipeline (Pre-Existing)

The signal writer pipeline was established in S148. For reference:

- **Pipeline name:** `rsi`
- **NATS consumer:** `WriterRSISignalConsumer`
- **NATS subject:** `signal.events.rsi.generated`
- **Insert SQL:** standard batch insert into `signals`
- **Mapper:** `mapSignalRow()` in `cmd/writer/mappers.go`

No changes were made to the write path in this stage.
