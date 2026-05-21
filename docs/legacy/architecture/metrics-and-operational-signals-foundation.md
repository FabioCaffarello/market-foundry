# Metrics and Operational Signals Foundation

> S354 — Operational Foundation Wave (OF-1 + OF-3)

## Purpose

This document describes the minimal Prometheus metrics foundation introduced in S354.
The goal is to close the two highest-value gaps identified in S352: absence of a `/metrics`
endpoint (RG-1) and lack of consumer lag and latency visibility (RG-3, RG-5).

## Scope

| Block | Deliverable | LOC |
|-------|------------|-----|
| OF-1 | Prometheus `/metrics` endpoint on gateway and HealthServer | ~80 |
| OF-3 | Consumer lag gauge + processing duration histogram | ~40 |

## Architecture

### Metrics Package

All Prometheus collectors are defined in `internal/shared/metrics/metrics.go` under the
`marketfoundry` namespace. The package exposes:

- **Metric definitions** — registered in `init()` on the default Prometheus registry.
- **Helper functions** — `ObserveHTTPRequest`, `InstrumentHTTPHandler`, `IncConsumerMessage`,
  `ObserveConsumerProcessing`, `SetConsumerLag`.
- **Handler** — `Handler()` and `HandlerFunc()` wrap `promhttp.Handler()`.

### Exposure Surfaces

| Binary | Surface | Mechanism |
|--------|---------|-----------|
| gateway | `/metrics` on main HTTP port (8080) | Route registered in `routes.Core()` |
| execute | `/metrics` on health HTTP port | `HealthServer` mux handler |
| store | `/metrics` on health HTTP port | `HealthServer` mux handler |
| other | `/metrics` on health HTTP port | `HealthServer` mux handler |

Every binary that imports `internal/shared/metrics` (directly or transitively through
`healthz` or `webserver`) registers metrics on the default Prometheus registry. The
`/metrics` endpoint serves the standard Prometheus exposition format.

### HTTP Request Instrumentation

The `webserver.RegisterRoutes` method wraps each handler with `metrics.InstrumentHTTPHandler`,
which records:

- `marketfoundry_http_request_duration_seconds` — histogram with method, path, status_code labels
- `marketfoundry_http_requests_total` — counter with method, path, status_code labels

The `/metrics` endpoint itself is excluded from instrumentation to avoid self-observation noise.
Labels use route patterns (e.g., `/execution/:type/latest`) not resolved URLs, keeping
cardinality bounded.

### Consumer Instrumentation

JetStream consumers (`Consumer`, `FillConsumer`) record:

- `marketfoundry_consumer_messages_total` — counter with consumer name and outcome labels
  (delivered, redelivered, terminated, nakked)
- `marketfoundry_consumer_processing_duration_seconds` — histogram of end-to-end message
  processing time (from delivery to ack/nak)
- `marketfoundry_consumer_lag_messages` — gauge updated from `msg.Metadata().NumPending`
  on each delivered message

Consumer lag is derived from JetStream message metadata, not from periodic info queries.
This means lag is only updated when messages arrive. When the consumer is idle, the last
known lag value persists as a gauge.

## Metric Inventory

### HTTP Metrics

| Name | Type | Labels | Description |
|------|------|--------|-------------|
| `marketfoundry_http_request_duration_seconds` | Histogram | method, path, status_code | Request latency |
| `marketfoundry_http_requests_total` | Counter | method, path, status_code | Request count |

### Consumer Metrics

| Name | Type | Labels | Description |
|------|------|--------|-------------|
| `marketfoundry_consumer_messages_total` | Counter | consumer, outcome | Message delivery outcomes |
| `marketfoundry_consumer_processing_duration_seconds` | Histogram | consumer | Message processing latency |
| `marketfoundry_consumer_lag_messages` | Gauge | consumer | Pending messages (lag) |

### Go Runtime Metrics

The default Prometheus registry automatically includes `go_*` and `process_*` metrics
(goroutines, memory, GC, file descriptors, etc.).

## Design Decisions

1. **Single package, default registry** — All metrics register on the default Prometheus
   registry via `init()`. This keeps wiring minimal. Unused metrics in a binary show zero
   values, which is harmless.

2. **Function wrappers over exported vars** — Consumer code calls `metrics.IncConsumerMessage()`
   instead of accessing `prometheus.CounterVec` directly. This keeps the `prometheus` import
   contained to `internal/shared/metrics` and avoids adding the dependency to adapter modules.

3. **Lag from message metadata** — Using `NumPending` from `jetstream.MsgMetadata` avoids
   periodic JetStream admin queries. The trade-off is that lag is only updated on delivery,
   not during idle periods.

4. **Route-pattern labels** — HTTP labels use the registered route pattern, not the resolved
   URL. This prevents label cardinality explosion from path parameters.

## Non-Goals

- Grafana dashboards or alerting rules
- OpenTelemetry / Jaeger tracing
- NATS request/reply client latency instrumentation (deferred)
- ClickHouse query latency instrumentation (deferred)
- Push-based alerting (Alertmanager)
- Soak or endurance testing of the metrics path itself
