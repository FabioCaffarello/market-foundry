# Current Signals Sufficiency, Gaps, and Priority Findings

> Honest assessment of whether existing operational signals are sufficient for sustained venue activation operation.

## Sufficiency Verdict

**The existing signals are sufficient for controlled, operator-attended operation. They are NOT sufficient for unattended, automated production operation.**

This is not a failure — it reflects the appropriate maturity stage. The codebase has been built with observability-ready patterns (structured logs, typed counters, HTTP surfaces, audit fields). What is missing is the integration layer that connects these signals to automated alerting and historical analysis.

## What Is Already Sufficient

### 1. Incident Investigation (SUFFICIENT)

Every decision point in the venue-active path emits structured logs with correlation IDs. Given access to logs, an operator can reconstruct the full lifecycle of any intent: receipt → gate check → staleness guard → submission → retry loop → fill/failure. Error details include venue HTTP status, error codes, retry metadata, and reconciliation state.

**Verdict**: No gap. Existing logs are sufficient for post-hoc investigation.

### 2. Manual Operational Queries (SUFFICIENT)

The HTTP surface (`/activation/surface`, `/execution/control`, `/statusz`, `/diagz`) answers the critical operational questions:
- Is the system alive? → `/healthz`
- Is it accepting traffic? → `/readyz`
- What mode is it in? → `/activation/surface`
- How many events processed/filled/errored? → `/statusz`
- Can I halt/resume? → `PUT /execution/control`

**Verdict**: No gap for manual operation. An operator with curl can assess and control the system.

### 3. Domain Audit Trail (SUFFICIENT)

Audit fields (timestamps, correlation IDs, gate metadata, fill records with simulated flags) are present at all critical domain boundaries. NATS streams retain events for 72 hours with durable consumers.

**Verdict**: No gap. Audit trail is structurally complete.

### 4. Error Classification (SUFFICIENT)

The Problem type system with retryable flags, venue error code overrides, and post-200 reconciliation handling provides operationally useful error taxonomy. Retry-submitter logs exact attempt counts, exhaustion state, and halt reasons.

**Verdict**: No gap. Error classification is production-grade.

### 5. Safety Gate Correctness (SUFFICIENT)

Kill switch and staleness guard are tested under endurance conditions (S349). Gate transitions produce exact expected counter outcomes. The invariant `processed == filled + skipped_halt + skipped_stale + errors` is proven stable.

**Verdict**: No gap. Safety gates are auditable and correct.

## Concrete Gaps

### Gap 1: No Time-Series Metric Export

**Severity**: HIGH for unattended operation, LOW for attended operation.

**Current state**: Counters exist as in-memory atomics, queryable only via `/statusz` HTTP JSON. There is no Prometheus endpoint, no OTEL exporter, no StatsD emitter. This means:
- No rate-of-change computation (e.g., "fills per minute dropped to zero")
- No historical trend analysis (e.g., "error rate has been climbing for 10 minutes")
- No percentile computation (e.g., "p99 submission latency exceeds 2s")
- No dashboard visualization without custom scraping

**Minimum fix**: Expose a `/metrics` endpoint in Prometheus exposition format, exporting the existing health tracker counters. This is ~100 lines of code on top of existing counter infrastructure.

**Priority**: P1 for unattended operation roadmap. Not blocking for attended operation.

### Gap 2: No Push-Based Alerting

**Severity**: HIGH for unattended operation, LOW for attended operation.

**Current state**: All signals require pull (HTTP poll or log scrape). There is no webhook, PagerDuty, Slack, or email integration. An operator must actively poll to detect problems.

**Minimum fix**: This is not an in-code fix — it requires deployment infrastructure (Alertmanager, Grafana Alerting, or equivalent). The prerequisite is Gap 1 (metric export).

**Priority**: P1 for unattended operation, blocked by Gap 1.

### Gap 3: No Consumer Lag Visibility

**Severity**: MEDIUM.

**Current state**: NATS JetStream tracks consumer lag internally, but this information is not surfaced to any operational endpoint. If the execute-venue-market-order-intake consumer falls behind, the only symptom is increased intent staleness — which the staleness guard catches but does not explain.

**Minimum fix**: Query NATS consumer info in the health tracker or `/statusz` response. JetStream's `ConsumerInfo.NumPending` provides this directly.

**Priority**: P2. Consumer lag is a leading indicator that existing signals (staleness guard) handle reactively.

### Gap 4: No Production Latency Histograms

**Severity**: MEDIUM.

**Current state**: S349 endurance tests track per-event latency with regression analysis, but the production venue-adapter-actor does not record or expose submission latency. Log timestamps allow post-hoc computation, but there is no real-time latency signal.

**Minimum fix**: Record submission duration in the health tracker as a counter or simple histogram (min/max/avg/count). This is ~20 lines within the existing tracker infrastructure.

**Priority**: P2. Latency is important for detecting venue degradation but is partially covered by timeout settings and retry behavior.

### Gap 5: No Log Aggregation Assumption

**Severity**: MEDIUM (environmental, not code).

**Current state**: Structured logs go to stdout. Without Loki, ELK, CloudWatch, or equivalent, log-based alerting (e.g., "alert when retry_exhausted appears more than 3 times in 5 minutes") is impractical.

**Minimum fix**: This is a deployment decision, not a code change. The code already emits structured JSON logs suitable for any aggregation stack.

**Priority**: P2 for operational maturity. Not a code gap.

### Gap 6: No Resource Profiling Alerts

**Severity**: LOW.

**Current state**: `/diagz` reports goroutine count. No memory, CPU, or file descriptor monitoring. Go runtime metrics are not exported.

**Minimum fix**: If Gap 1 is addressed (Prometheus endpoint), Go runtime metrics can be added with `promhttp` and `collectors.NewGoCollector()`.

**Priority**: P3. Resource issues are unlikely to be the first failure mode for venue activation.

## Gap Priority Matrix

| Gap | Severity | Code Change Needed | Dependency | Priority |
|-----|----------|-------------------|------------|----------|
| 1. Metric export | HIGH (unattended) | ~100 LOC | None | P1 |
| 2. Push alerting | HIGH (unattended) | None (infra) | Gap 1 | P1 |
| 3. Consumer lag | MEDIUM | ~30 LOC | None | P2 |
| 4. Latency histograms | MEDIUM | ~20 LOC | None | P2 |
| 5. Log aggregation | MEDIUM | None (deploy) | None | P2 |
| 6. Resource profiling | LOW | ~10 LOC | Gap 1 | P3 |

## Inflation Guard

This assessment explicitly avoids recommending:
- A full observability platform (OTEL collector, Jaeger, distributed tracing)
- Custom dashboarding infrastructure
- APM tooling
- Multi-environment monitoring federation

The minimum viable path from current state to unattended operation is:
1. Add `/metrics` Prometheus endpoint (~100 LOC)
2. Deploy Prometheus + Alertmanager (infrastructure)
3. Define alert rules for the signals already proven stable

Everything else is incremental and should be scoped to concrete operational need.
