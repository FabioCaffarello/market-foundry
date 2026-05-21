# Monitoring, Alertability, and Operational Signals Assessment

> Inventory and assessment of all operational signals available for sustained venue activation operation.

## Purpose

This document maps every operational signal the venue-active path currently emits or exposes, assesses its usefulness for monitoring and alertability, and identifies concrete gaps. It does NOT propose a platform-wide observability solution — it evaluates whether existing signals are sufficient for operating venue activation safely.

## Signal Inventory

### 1. Structured Logs (slog)

| Component | Key Log Points | Fields Emitted |
|-----------|---------------|----------------|
| venue-adapter-actor | activation surface resolved | adapter, gate, credentials, effective, is_live |
| venue-adapter-actor | venue adapter started | staleness_max_age, submit_timeout, control_gate, retry_submitter, post200_reconciler |
| venue-adapter-actor | intent blocked by kill switch | reason, source, symbol, timeframe, correlation_id |
| venue-adapter-actor | intent stale — skipped | reason, source, symbol, timeframe, age, max_age, correlation_id |
| venue-adapter-actor | venue submit failed | error, source, symbol, timeframe, correlation_id, retry_attempts, retry_exhausted, retry_halted, retry_deadline_exceeded |
| venue-adapter-actor | venue order filled | venue_order_id, status, source, symbol, timeframe, side, quantity, filled_quantity, correlation_id |
| venue-adapter-actor | venue adapter stats | processed, filled, skipped_stale, skipped_halt, errors |
| retry-submitter | retry succeeded / exhausted / attempt failed / deadline exceeded / halted | attempts, max_attempts, error |
| healthz | component idle | tracker, idle_seconds, last_event, event_count, error_count, counters |

**Assessment**: Structured logging is comprehensive for the venue-active path. Every decision point (gate check, staleness guard, retry loop, fill, failure) emits a structured log with correlation metadata. Log fields are machine-parseable.

**Operational usefulness**: HIGH — sufficient for post-hoc incident investigation and log-based alerting.

### 2. Health Tracker Counters

| Counter | Component | Increment Trigger |
|---------|-----------|-------------------|
| `processed` | venue-adapter | Every intent received |
| `processed:{symbol}` | venue-adapter | Intent received for specific symbol |
| `filled` | venue-adapter | Successful fill |
| `filled:{symbol}` | venue-adapter | Fill for specific symbol |
| `skipped_halt` | venue-adapter | Intent blocked by kill switch |
| `skipped_stale` | venue-adapter | Intent rejected by staleness guard |
| `retry_attempts` | retry-submitter | Each retry attempt |
| `retry_success_after_retry` | retry-submitter | Succeeded after initial failure |
| `retry_exhausted` | retry-submitter | Max attempts reached |
| `retry_halted` | retry-submitter | Kill switch aborted retry loop |
| `retry_deadline_exceeded` | retry-submitter | Global deadline expired |
| `eventCount` | healthz (per tracker) | Any event recorded |
| `errorCount` | healthz (per tracker) | Any error recorded |

**Assessment**: Counter coverage matches the critical decision points in the venue-active path. The invariant `processed == filled + skipped_halt + skipped_stale + errors` is proven stable over endurance testing (S349).

**Operational usefulness**: HIGH — counters are queryable via `/statusz` and usable for threshold-based alerting.

### 3. HTTP Query Surfaces

| Endpoint | Method | Returns | Operational Use |
|----------|--------|---------|-----------------|
| `/healthz` | GET | `{"status":"ok"}` | Liveness probe (K8s/Docker) |
| `/readyz` | GET | Readiness state + checks | Startup readiness gate |
| `/statusz` | GET | Phase, uptime, all tracker counters | Real-time operational snapshot |
| `/diagz` | GET | Runtime diagnostics, goroutines, readiness | Debugging/triage |
| `/activation/surface` | GET | Adapter, gate, credentials, effective mode | Current activation state |
| `/execution/control` | GET | Gate status, reason, updated_at, updated_by | Current gate state |
| `/execution/control` | PUT | Update gate (halt/resume) | Operational control |

**Assessment**: HTTP surfaces cover the three critical operational questions:
1. *Is the system alive?* → `/healthz`
2. *What is the current activation state?* → `/activation/surface`
3. *What are the counters showing?* → `/statusz`

**Operational usefulness**: HIGH — sufficient for manual and scripted operational queries.

### 4. Audit Fields (Domain Types)

| Type | Key Audit Fields |
|------|-----------------|
| ActivationSurface | adapter, gate, credentials, effective, observed_at |
| ControlGate | status, reason, updated_at, updated_by |
| ExecutionIntent | status, correlation_id, causation_id, timestamp, final |
| FillRecord | price, quantity, fee, simulated, timestamp |
| ActivationDimensions | adapter, credentials, reported_at, reported_by |

**Assessment**: Audit fields are present at all critical domain boundaries. The `observed_at`, `updated_at`, `reported_at`, and `timestamp` fields enable temporal correlation. The `simulated` flag on FillRecord distinguishes paper from live fills.

**Operational usefulness**: HIGH — supports traceability and incident reconstruction.

### 5. NATS Control Plane Signals

| Signal | Channel | Purpose |
|--------|---------|---------|
| Gate get/set | `execution.control.{get,set}` | Query/update execution gate via request/reply |
| Surface query | `execution.activation.surface` | Query activation surface via request/reply |
| Paper order events | `EXECUTION_EVENTS` stream | Durable stream, 72h retention, 256MB |
| Venue fill events | `EXECUTION_FILL_EVENTS` stream | Durable stream, 72h retention, 256MB |
| Control KV | `EXECUTION_CONTROL` bucket | Durable gate state, file-backed, 1MB |

**Assessment**: NATS provides durable event streams and KV-backed control state. Consumer durability (execute-venue-market-order-intake, AckWait 30s, MaxDeliver 5) ensures delivery guarantees are explicit.

**Operational usefulness**: MEDIUM-HIGH — event streams enable replay and audit; KV state persists across restarts.

### 6. Error Classification and Enrichment

| Error Category | Classification | Retryable | Detail Fields |
|---------------|---------------|-----------|---------------|
| Auth failure (401/403) | SYS_INTERNAL | No | venue_http_status |
| Rate limit (429) | SYS_UNAVAILABLE | Yes | venue_http_status |
| Client error (4xx) | SYS_INTERNAL | No | venue_http_status, venue_error_code, venue_error_message |
| Server error (5xx) | SYS_UNAVAILABLE | Yes | venue_http_status |
| Venue code overrides (-1001, -1003, -1015) | SYS_UNAVAILABLE | Yes | venue_error_class |
| Post-200 body read failure | SYS_INTERNAL | No | body_read_failure_after_200, client_order_id |
| Post-200 reconciliation failure | — | — | reconciliation_attempted, reconciliation_failed, reconciliation_error |

**Assessment**: Error classification is detailed and operationally meaningful. The `Retryable` flag drives retry-submitter behavior automatically. Venue error code overrides prevent false non-retryable classification.

**Operational usefulness**: HIGH — error taxonomy supports both automated retry and human triage.

### 7. Phase Detection (Computed Health State)

| Phase | Condition | Operational Meaning |
|-------|-----------|-------------------|
| starting | Uptime < 30s, no tracker events | System initializing |
| warming | At least one tracker awaiting first event | Waiting for traffic |
| active | All trackers receiving events, none idle | Normal operation |
| idle | At least one tracker exceeds idle threshold (2 min) | Reduced traffic |
| stalled | All active trackers exceed idle threshold | No traffic |
| degraded | Any tracker has `pipeline_degraded > 0` | Known problem |

**Assessment**: Phase computation provides a single high-level health signal derived from tracker state. The idle monitoring heartbeat (30s interval) logs warnings when components go idle.

**Operational usefulness**: MEDIUM-HIGH — phase is queryable via `/statusz` but is not currently pushed to any external alerting system.

## Alertability Assessment

### What Can Be Alerted On Today (With Log/HTTP Scraping)

| Alert Rule | Signal Source | Detection Method |
|-----------|--------------|-----------------|
| System down | `/healthz` returns non-200 or timeout | HTTP probe |
| System not ready | `/readyz` returns non-200 | HTTP probe |
| Gate halted unexpectedly | `/activation/surface` effective != venue_live | HTTP poll |
| Counter invariant violation | `/statusz` counters | Computed: processed != filled + skipped_halt + skipped_stale + errors |
| Error rate spike | `/statusz` errorCount | Rate-of-change on error counter |
| Component idle/stalled | `/statusz` phase | Phase == idle or stalled |
| Retry exhaustion | Log: "retry exhausted" | Log pattern match |
| Fill during halt | Log: venue order filled while gate halted | Cross-signal correlation |

### What Cannot Be Alerted On Today

| Gap | Why It Matters | Current State |
|-----|---------------|---------------|
| No metric export (Prometheus/OTEL) | Counter values exist but only via HTTP pull; no time-series history, no rate computation, no percentile analysis | Counters are in-memory atomics, exposed only via `/statusz` JSON |
| No latency percentiles in production | S349 proved latency tracking in tests but production code does not emit per-intent latency histograms | Only log timestamps enable post-hoc latency computation |
| No push-based alerting | All signals require pull (HTTP poll or log scrape); no webhook, PagerDuty, or Slack integration | Purely pull-based |
| No log aggregation assumed | Structured logs exist but assessment assumes no Loki/ELK/CloudWatch; without aggregation, log-based alerting is impractical at scale | Logs go to stdout |
| No stream consumer lag monitoring | NATS JetStream tracks consumer lag internally but it is not surfaced to `/statusz` or any external metric | Consumer durable names exist; lag is opaque |
| No disk/memory/goroutine alerting | `/diagz` shows goroutine count but no threshold alerting | Informational only |
