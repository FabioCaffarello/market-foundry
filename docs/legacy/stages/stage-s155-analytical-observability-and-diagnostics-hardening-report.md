# Stage S155 — Analytical Observability and Diagnostics Hardening Report

## Objective

Strengthen observability and diagnostics of the analytical layer with minimal useful signals for operation, debugging, and reliability validation.

## Executive Summary

The analytical layer gained targeted diagnostic signals that make pipeline health, data flow, backpressure, and failure modes visible through existing health endpoints and structured logs. No external observability tooling was introduced. The writer runtime was added to the diagnostic script. The layer is now operationally diagnosable at a level appropriate for the current hardened stage.

## Changes Applied

### Code Changes

#### Inserter Actor (`cmd/writer/inserter.go`)
- **`buffer_depth` gauge**: Updated on every row insert, overflow eviction, and flush. Answers "is backpressure building?"
- **`flush_total` counter**: Counts successful batch flushes (distinct from `events_flushed` which counts rows). Answers "how many batches completed?"
- **`flush_duration_ms` gauge**: Records duration of last flush operation in milliseconds, on both success and failure. Answers "is ClickHouse slow?"
- **`flush_ms` in DEBUG log**: Batch flush log now includes duration for log-based debugging.
- **`flush_ms` in ERROR log**: Failed flush log now includes total duration across retries.

#### Consumer Pipelines (`cmd/writer/pipeline.go`)
- **`events_received` counter**: Every consumer callback now increments `events_received` on its tracker. Enables comparing inflow (consumer) vs outflow (inserter) to detect pipeline mismatches.

#### Health Server (`internal/shared/healthz/healthz.go`)
- **`degraded_trackers` field in `/statusz`**: New top-level array listing tracker names with `pipeline_degraded > 0`. Eliminates need to scan all tracker counters to find which pipeline is degraded.
- **Counter snapshot in heartbeat idle warnings**: The 30-second heartbeat monitor now includes all custom counters when logging idle warnings, providing full operational context in a single log entry.

#### Diagnostic Script (`scripts/diag-check.sh`)
- **Writer runtime added**: `writer:8085` included in the runtime ports list. `diag-check.sh` now queries the writer's `/readyz`, `/statusz`, and `/diagz` endpoints alongside all operational runtimes.

#### Tests (`internal/shared/healthz/healthz_test.go`)
- **`TestHealthServer_Statusz_Phase_Degraded`**: Extended to verify `degraded_trackers` field contains the degraded tracker name.
- **`TestHealthServer_Statusz_NoDegradedTrackers`**: New test verifying `degraded_trackers` is absent when no tracker is degraded.

### Documentation

- **`docs/architecture/analytical-observability-and-diagnostics-hardening.md`**: Complete signal catalog, design principles, diagnostic tooling guide, operational questions answered, and observability limits.
- **`docs/architecture/analytical-runtime-runbook-and-signal-interpretation.md`**: Operational runbook with phase interpretation, scenario playbooks (degraded, overflow, flush failures, stalled, unreachable), signal relationships, key invariants, and monitoring without external tools.

## Signal Summary

### New Signals (S155)

| Signal | Location | Type | Operational Value |
|---|---|---|---|
| `events_received` | consumer tracker | counter | Inflow measurement per family |
| `buffer_depth` | inserter tracker | gauge | Backpressure visibility |
| `flush_total` | inserter tracker | counter | Batch success count |
| `flush_duration_ms` | inserter tracker | gauge | ClickHouse latency signal |
| `degraded_trackers` | `/statusz` response | array | Quick degradation identification |

### Pre-existing Signals (Documented and Cataloged)

| Signal | Location | Operational Value |
|---|---|---|
| `events_flushed` | inserter tracker | Rows successfully written |
| `events_dropped` | inserter tracker | Total rows permanently lost |
| `events_overflowed` | inserter tracker | Rows lost to buffer overflow |
| `flush_failures` | inserter tracker | Batch drops after retry exhaustion |
| `pipeline_restarts` | consumer tracker | Restart attempts per family |
| `pipeline_degraded` | consumer tracker | Family restart budget exhausted |
| `phase` | `/statusz` | Aggregate operational phase |

## Files Changed

| File | Change |
|---|---|
| `cmd/writer/inserter.go` | buffer_depth gauge, flush_total counter, flush_duration_ms gauge, flush_ms in logs |
| `cmd/writer/pipeline.go` | events_received counter on all 6 consumer callbacks |
| `internal/shared/healthz/healthz.go` | degraded_trackers in /statusz, counter snapshot in heartbeat |
| `internal/shared/healthz/healthz_test.go` | Tests for degraded_trackers field |
| `scripts/diag-check.sh` | Writer runtime added to diagnostic sweep |
| `docs/architecture/analytical-observability-and-diagnostics-hardening.md` | New |
| `docs/architecture/analytical-runtime-runbook-and-signal-interpretation.md` | New |

## Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| Analytical layer gains minimal useful signals | Done — 5 new signals with clear operational value |
| Failures, overflow, pipeline state more visible | Done — buffer_depth, flush_duration_ms, degraded_trackers |
| Operation and debugging clarity improved | Done — runbook with scenario playbooks, signal relationships |
| Solution remains lightweight | Done — no external dependencies, reuses existing Tracker/Counter infrastructure |
| Ready for Wave A readiness review | Done — all signal gaps documented, limits acknowledged |

## Guard Rail Compliance

| Guard Rail | Compliance |
|---|---|
| No heavy observability | No OpenTelemetry, Prometheus, or Grafana introduced |
| No analytical functionality expansion | No new queries, tables, or reader capabilities |
| No signals without operational value | Every signal answers a specific operational question |
| Observability limits documented | Full gap table in architecture doc |

## Remaining Limits

1. **No NATS consumer lag visibility** — requires JetStream admin API integration
2. **No historical counter trends** — counters are in-memory, reset on restart
3. **No per-symbol granularity** — counters are per-pipeline-family
4. **No reader/query instrumentation** — analytical reader has no latency signals
5. **No push-based alerting** — signals are pull-only via HTTP
6. **No distributed tracing** — no cross-service correlation

## Recommended Preparation for S156

The analytical layer is now minimally observable and diagnosable. Recommended next steps:

1. **Wave A formal readiness review** — assess whether the analytical layer meets production-readiness criteria with current hardening (S150-S155).
2. **Gateway analytical reader instrumentation** — add query latency and error signals to the reader adapter for complete end-to-end analytical visibility.
3. **NATS consumer lag probe** — if lag visibility becomes critical, add a JetStream admin check to `/diagz`.
4. **Integration smoke test for writer diagnostics** — extend smoke scripts to verify writer `/statusz` signals after a pipeline run.
