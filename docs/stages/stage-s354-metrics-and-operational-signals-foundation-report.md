# S354 — Metrics and Operational Signals Foundation

> Stage type: Implementation (OF-1 + OF-3)
> Wave: Operational Foundation (S353–S358)
> Status: **Closed**
> Date: 2026-03-22

## Governing Questions

| ID | Question | Answer |
|----|----------|--------|
| OFQ-1 | Does the system expose a `/metrics` endpoint? | **YES** — gateway (route table) and all HealthServer binaries |
| OFQ-2 | Does `/metrics` serve valid Prometheus exposition format? | **YES** — verified via unit test |
| OFQ-5 | Is consumer lag visible as a metric? | **YES** — `marketfoundry_consumer_lag_messages` gauge |
| OFQ-6 | Are latency histograms available for the main processing path? | **YES** — HTTP request duration + consumer processing duration |

## Summary

S354 delivers the minimal Prometheus metrics foundation for the market-foundry platform.
It closes two of the highest-priority gaps from the S352 Production Readiness Assessment:

- **RG-1** (HIGH): No Prometheus `/metrics` endpoint → **closed**
- **RG-3** (MEDIUM): No consumer lag visibility → **closed**
- **RG-5** (MEDIUM): No latency histograms → **closed**

## Deliverables

### Code

| File | Change | Purpose |
|------|--------|---------|
| `internal/shared/metrics/metrics.go` | **New** | Prometheus metric definitions, helpers, handler |
| `internal/shared/metrics/metrics_test.go` | **New** | 5 unit tests covering all metric surfaces |
| `internal/shared/webserver/server.go` | Modified | HTTP handler instrumentation in RegisterRoutes |
| `internal/shared/healthz/healthz.go` | Modified | `/metrics` endpoint on HealthServer |
| `internal/interfaces/http/routes/core.go` | Modified | `/metrics` route in gateway Core routes |
| `internal/adapters/nats/natsexecution/consumer.go` | Modified | Consumer message counters, processing histogram, lag gauge |
| `internal/adapters/nats/natsexecution/fill_consumer.go` | Modified | Same instrumentation as consumer |
| `internal/shared/go.mod` | Modified | Added `prometheus/client_golang` dependency |

### Documentation

| File | Purpose |
|------|---------|
| `docs/architecture/metrics-and-operational-signals-foundation.md` | Architecture and design decisions |
| `docs/architecture/prometheus-metrics-consumer-lag-latency-semantics-and-limits.md` | Metric semantics, operational use, known limits |
| `docs/stages/stage-s354-metrics-and-operational-signals-foundation-report.md` | This report |

## Metrics Delivered

### HTTP Metrics (OF-1)

| Metric | Type | Labels |
|--------|------|--------|
| `marketfoundry_http_request_duration_seconds` | Histogram | method, path, status_code |
| `marketfoundry_http_requests_total` | Counter | method, path, status_code |

### Consumer Metrics (OF-3)

| Metric | Type | Labels |
|--------|------|--------|
| `marketfoundry_consumer_messages_total` | Counter | consumer, outcome |
| `marketfoundry_consumer_processing_duration_seconds` | Histogram | consumer |
| `marketfoundry_consumer_lag_messages` | Gauge | consumer |

### Runtime Metrics (automatic)

Go runtime and process metrics (`go_*`, `process_*`) are included by the default
Prometheus registry.

## Test Evidence

```
=== RUN   TestHandlerFunc_ServesPrometheusMetrics        — PASS
=== RUN   TestObserveHTTPRequest_RecordsMetrics          — PASS
=== RUN   TestInstrumentHTTPHandler_CapturesStatusCode   — PASS
=== RUN   TestConsumerMetrics_DoNotPanic                 — PASS
=== RUN   TestStatusWriter_DefaultsTo200                 — PASS
ok      internal/shared/metrics    0.238s
```

### Regression Check

All 38 packages across core modules pass with zero regressions:
- `internal/shared/...` — 11 packages OK
- `internal/interfaces/http/...` — 2 packages OK
- `internal/domain/...` — 8 packages OK
- `internal/application/...` — 17 packages OK

All 15 workspace modules compile successfully with `go vet` clean.

## Design Decisions

1. **Single metrics package in `internal/shared/metrics`** — Centralized definitions,
   function wrappers to avoid leaking Prometheus types to adapter modules.

2. **Default Prometheus registry** — No custom registries. All binaries share the same
   metric set; unused metrics show zero values.

3. **Consumer lag from message metadata** — Uses `NumPending` from `jetstream.MsgMetadata`
   instead of periodic admin queries. Trade-off: stale during idle periods.

4. **`/metrics` on both gateway and HealthServer** — Gateway serves it via the route table
   (alongside domain routes). Other binaries serve it via the shared HealthServer mux.

5. **Route-pattern labels** — HTTP labels use route patterns, not resolved URLs, preventing
   cardinality explosion.

## Guard Rails

| Rule | Held? |
|------|-------|
| No dashboards or Grafana | YES |
| No OTEL/Jaeger | YES |
| No indiscriminate instrumentation | YES — only HTTP handlers + consumers |
| No observability platform expansion | YES |
| No new external dependencies beyond prometheus/client_golang | YES |

## Residual Gaps (Not in S354 Scope)

| Gap | Priority | Notes |
|-----|----------|-------|
| NATS request/reply latency | MEDIUM | Gateway-side; would separate NATS from handler time |
| Consumer lag during idle | LOW | Requires periodic ConsumerInfo queries |
| ClickHouse query latency | LOW | Analytical adapter not instrumented |
| Push alerting (Alertmanager) | MEDIUM | Requires infrastructure deployment |
| Per-symbol consumer metrics | LOW | Would increase cardinality per active symbol |

## S352 Gap Closure Summary

| Gap | Status |
|-----|--------|
| RG-1 (HIGH): No `/metrics` endpoint | **CLOSED** |
| RG-3 (MEDIUM): No consumer lag visibility | **CLOSED** |
| RG-5 (MEDIUM): No latency histograms | **CLOSED** |

## Preparation for S355+

- **S355 (OF-2: CI smoke integration)**: The `/metrics` endpoint is now available for
  CI smoke scripts to verify metric exposure (`curl -s /metrics | grep marketfoundry`).
- **S356 (OF-4: Startup credential validation)**: Independent of metrics.
- **S358 (OF-5: Evidence gate)**: Metrics foundation is now assessable.
- **Future**: NATS request/reply latency instrumentation can be added to the metrics
  package without architectural changes.
