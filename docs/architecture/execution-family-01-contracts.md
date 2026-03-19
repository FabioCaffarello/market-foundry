# Execution Family 01 — Paper Order Contracts

> S71 — NATS contracts for the paper_order execution family.

## Stream

| Property | Value |
|----------|-------|
| Name | `EXECUTION_EVENTS` |
| Subjects | `execution.events.>` |
| Storage | FileStorage |
| MaxAge | 72 hours |
| MaxBytes | 2 GB |
| Dedup window | JetStream MsgID-based |

## Event Subject

```
execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}
```

Example: `execution.events.paper_order.submitted.binancef.btcusdt.60`

## Durable Consumer

| Property | Value |
|----------|-------|
| Durable | `store-execution-paper-order` |
| FilterSubject | `execution.events.paper_order.submitted.>` |
| AckPolicy | AckExplicit |
| AckWait | 30s |
| MaxDeliver | 5 |
| Stream | `EXECUTION_EVENTS` |

## KV Bucket

| Property | Value |
|----------|-------|
| Bucket | `EXECUTION_PAPER_ORDER_LATEST` |
| Storage | FileStorage |
| MaxBytes | 64 MB |
| Key format | `{source}.{symbol}.{timeframe}` |

## Query Subject

```
execution.query.paper_order.latest
```

Queue group: `execution.query`

## Envelope Types

| Type | Value |
|------|-------|
| Event | `execution.events.v1.paper_order_submitted` |
| Request | `execution.query.v1.paper_order_latest_request` |
| Reply | `execution.query.v1.paper_order_latest_reply` |

## Deduplication Key

```
exec:paper_order:{source}:{symbol}:{timeframe}:{unix_timestamp}
```

## Partition Key

```
{source}.{symbol}.{timeframe}
```
