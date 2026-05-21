# Risk Family 01 — Position Exposure Contracts

## Stream

| Property | Value |
|----------|-------|
| Name | `RISK_EVENTS` |
| Max bytes | 2 GB |
| Max age | 72 h |
| Storage | File |
| Dedup window | 2 min (via JetStream MsgID) |

## Event

| Property | Value |
|----------|-------|
| Subject | `risk.events.position_exposure.assessed.{source}.{symbol}.{timeframe}` |
| Type | `risk.events.v1.position_exposure_assessed` |

### Payload Fields

- `source` — exchange or data source identifier
- `symbol` — trading pair
- `timeframe` — aggregation interval
- `disposition` — one of: `approved`, `modified`, `rejected`
- `original_size` — size proposed by strategy
- `adjusted_size` — size after risk adjustment (equals original if approved, zero if rejected)
- `max_exposure` — configured exposure limit applied
- `confidence` — upstream strategy confidence passed through
- `strategy_type` — originating strategy family
- `timestamp` — event production time (RFC 3339)

## Consumer

| Property | Value |
|----------|-------|
| Name | `store-risk-position-exposure` |
| Durable | Yes |
| Ack wait | 30 s |
| Max deliver | 5 |
| Filter subject | `risk.events.position_exposure.assessed.>` |

## KV Bucket

| Property | Value |
|----------|-------|
| Name | `RISK_POSITION_EXPOSURE_LATEST` |
| Max bytes | 64 MB |
| Storage | File |
| Key format | `{source}.{symbol}.{timeframe}` |

## Query

| Property | Value |
|----------|-------|
| Subject | `risk.query.position_exposure.latest` |
| Request type | `risk.query.v1.position_exposure_latest_request` |
| Reply type | `risk.query.v1.position_exposure_latest_reply` |
| Queue group | `risk.query` |

### Request Fields

- `source` — exchange or data source identifier
- `symbol` — trading pair
- `timeframe` — aggregation interval

### Reply Fields

- `disposition` — assessed disposition
- `original_size` — proposed size
- `adjusted_size` — size after risk adjustment
- `max_exposure` — limit applied
- `confidence` — pass-through confidence
- `strategy_type` — originating strategy family
- `timestamp` — assessment time
- `error` — empty on success, error description on failure
