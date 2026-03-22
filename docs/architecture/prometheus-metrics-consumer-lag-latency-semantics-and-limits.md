# Prometheus Metrics: Consumer Lag, Latency Semantics and Limits

> S354 — Operational Foundation Wave (OF-1 + OF-3)

## Purpose

This document specifies the exact semantics of the Prometheus metrics introduced in S354,
their operational meaning, known limitations, and guidance for consumers (dashboards,
alerting rules, or human operators).

---

## HTTP Request Metrics

### `marketfoundry_http_request_duration_seconds`

- **Type**: Histogram
- **Unit**: Seconds
- **Buckets**: 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
- **Labels**: `method` (HTTP verb), `path` (route pattern), `status_code` (HTTP status)
- **Semantics**: Measures wall-clock time from handler entry to response completion.
  Includes all handler processing, NATS request/reply wait time, and JSON serialization.
- **Scope**: Gateway HTTP routes only. The HealthServer endpoints (`/healthz`, `/readyz`,
  `/statusz`, `/diagz`) on non-gateway binaries are NOT instrumented.

**Limitations**:
- Does not include network transfer time (time-to-first-byte to client).
- Does not break down time spent in NATS vs. handler logic. If NATS request/reply
  latency instrumentation is needed, it should be added as a separate metric in a future stage.
- Bucket boundaries may need tuning after production observation. Current buckets are
  biased toward sub-second latencies typical of NATS-backed queries.

### `marketfoundry_http_requests_total`

- **Type**: Counter
- **Labels**: `method`, `path`, `status_code`
- **Semantics**: Monotonically increasing count of HTTP requests. Incremented once per
  completed request, after the handler has written its response.

**Operational use**:
- `rate(marketfoundry_http_requests_total[5m])` gives request rate.
- Filter by `status_code=~"5.."` for error rate.
- Filter by `path` for per-endpoint breakdown.

---

## Consumer Metrics

### `marketfoundry_consumer_messages_total`

- **Type**: Counter
- **Labels**: `consumer` (durable consumer name), `outcome`
- **Outcome values**:
  - `delivered` — message received from JetStream and processing began
  - `redelivered` — message received with `NumDelivered > 1` (retry)
  - `terminated` — message terminated (non-recoverable error, `msg.Term()`)
  - `nakked` — message negatively acknowledged for redelivery (`msg.Nak()`)

**Semantics**:
- Every message increments `delivered` exactly once.
- A redelivered message increments BOTH `delivered` and `redelivered`.
- `terminated` and `nakked` are mutually exclusive error outcomes.
- `delivered - terminated - nakked` gives successfully processed messages
  (though some delivered messages may be in-flight).

**Operational use**:
- `rate(marketfoundry_consumer_messages_total{outcome="delivered"}[5m])` — throughput
- `rate(marketfoundry_consumer_messages_total{outcome="redelivered"}[5m])` — retry pressure
- `marketfoundry_consumer_messages_total{outcome="terminated"}` — permanent failures (should be near zero)

### `marketfoundry_consumer_processing_duration_seconds`

- **Type**: Histogram
- **Unit**: Seconds
- **Buckets**: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
- **Labels**: `consumer` (durable consumer name)
- **Semantics**: Measures wall-clock time from message callback entry to ACK/NAK/TERM.
  Includes: payload decoding, handler execution (safety gate + venue submission + fill
  publish for execution consumers), and acknowledgment.

**What it captures**:
- For the execution consumer: full intent-to-ack cycle including venue adapter processing,
  retry loops, and fill event publishing.
- For the fill consumer: full fill-event-to-ack cycle including store materialization.

**What it does NOT capture**:
- Queue wait time (time between publish and delivery). This would require comparing
  the message timestamp with delivery time, which is a different metric.
- Actor mailbox queue time (time between `ctx.Send()` and actor `Receive()`).

**Limitations**:
- Processing duration includes all handler work. In the execution consumer, a single
  observation may include multiple venue submission retries (up to 10s deadline).
  High p99 values may indicate retry pressure rather than slow venue responses.
- Bucket boundaries assume sub-second typical processing. If venue submission with
  retries regularly exceeds 5s, the 10s bucket may need expansion.

### `marketfoundry_consumer_lag_messages`

- **Type**: Gauge
- **Labels**: `consumer` (durable consumer name)
- **Semantics**: Set to `msg.Metadata().NumPending` on each message delivery. Represents
  the number of messages in the stream that have not yet been delivered to this consumer,
  as reported by JetStream at the time of the last delivery.

**Important caveats**:
1. **Stale during idle periods** — When no messages arrive, the gauge retains its last
   value. A lag of 0 from the last message does NOT mean the stream is still empty now.
   Use this metric in conjunction with throughput rate.
2. **Not real-time** — The value is a snapshot from the last delivery, not a live query.
   True real-time lag requires periodic `ConsumerInfo` queries, which this foundation
   stage does not implement.
3. **Per-consumer, not per-partition** — JetStream consumers are not partitioned like
   Kafka consumer groups. The lag value is global for the durable consumer.

**Operational use**:
- `marketfoundry_consumer_lag_messages > 100` suggests the consumer is falling behind.
- Combined with `rate(messages_total{outcome="delivered"})`, determines if the consumer
  is catching up or falling further behind.
- A sustained lag of 0 with positive delivery rate confirms the consumer is keeping up.

---

## Go Runtime Metrics

The default Prometheus registry provides these automatically:

| Metric | Use |
|--------|-----|
| `go_goroutines` | Active goroutine count — detects leaks |
| `go_memstats_alloc_bytes` | Current heap allocation |
| `go_gc_duration_seconds` | GC pause latency |
| `process_open_fds` | Open file descriptors — detects connection leaks |
| `process_resident_memory_bytes` | RSS — overall memory footprint |

---

## Label Cardinality

| Metric | Max Labels | Cardinality Bound |
|--------|-----------|-------------------|
| HTTP request duration/total | ~20 routes x 5 methods x 10 status codes | ~1000 (theoretical), ~50 (practical) |
| Consumer messages total | 2 consumers x 4 outcomes | 8 |
| Consumer processing duration | 2 consumers | 2 |
| Consumer lag | 2 consumers | 2 |

Cardinality is well within safe bounds for a single-instance Prometheus scrape target.

---

## Scrape Configuration

```yaml
# Example Prometheus scrape config
scrape_configs:
  - job_name: 'marketfoundry-gateway'
    static_configs:
      - targets: ['gateway:8080']
    scrape_interval: 15s

  - job_name: 'marketfoundry-execute'
    static_configs:
      - targets: ['execute:8080']
    scrape_interval: 15s
```

---

## Known Gaps (Not in Scope for S354)

1. **NATS request/reply latency** — The gateway's NATS request client is not instrumented.
   Adding a `nats_request_duration_seconds` histogram to `NATSRequestClient.Request()` would
   separate NATS latency from handler processing time.

2. **Consumer lag during idle** — The gauge is stale when no messages arrive. Periodic
   `ConsumerInfo` queries would provide accurate idle-period lag.

3. **Push alerting** — Metrics are pull-only. Alertmanager integration requires a separate
   deployment.

4. **ClickHouse query latency** — The analytical adapter is not instrumented.

5. **Per-symbol metrics** — Consumer metrics do not carry symbol/timeframe labels. Adding
   these would increase cardinality proportionally to the number of active symbols.
