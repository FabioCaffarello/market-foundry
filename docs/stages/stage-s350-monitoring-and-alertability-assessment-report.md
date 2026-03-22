# Stage S350 — Monitoring and Alertability Assessment Report

> Assessment of whether existing operational signals are sufficient for sustained venue activation operation.

## Executive Summary

S350 inventoried and assessed all operational signals currently emitted by the venue-active path: structured logs, health tracker counters, HTTP query surfaces, domain audit fields, NATS control plane signals, error classification, and computed health phases. The assessment finds that **existing signals are sufficient for attended, operator-driven operation** but **insufficient for unattended, automated production operation**. Six concrete gaps are documented with severity and minimum-fix estimates. The minimum viable path to unattended operation is ~100 lines of Prometheus metric export code plus deployment infrastructure — no observability platform required.

## Signals Assessed

| Category | Signal Count | Coverage |
|----------|-------------|----------|
| Structured log points (slog) | 15+ distinct log messages | Every decision point in venue-active path |
| Health tracker counters | 13 named counters | processed, filled, skipped, retries, errors — by component and symbol |
| HTTP endpoints | 7 endpoints | Liveness, readiness, status, diagnostics, activation surface, execution control (read + write) |
| Domain audit fields | 5 types with timestamps | ActivationSurface, ControlGate, ExecutionIntent, FillRecord, ActivationDimensions |
| NATS control signals | 3 request/reply subjects + 2 streams + 1 KV bucket | Gate get/set, surface query, paper events, venue fills, control state |
| Error classification | 7+ categories with retryable flags | HTTP status mapping, venue error code overrides, post-200 reconciliation |
| Computed health phases | 6 phases | starting, warming, active, idle, stalled, degraded |

## Principal Findings

### 1. Incident Investigation: SUFFICIENT

Structured logs with correlation IDs cover the full intent lifecycle (receipt → gate → staleness → submission → retry → fill/failure). Error details include venue HTTP status, error codes, retry metadata, and reconciliation state. An operator with log access can reconstruct any event.

### 2. Manual Operational Queries: SUFFICIENT

HTTP endpoints answer all critical operational questions: liveness (`/healthz`), readiness (`/readyz`), activation state (`/activation/surface`), counter state (`/statusz`), and control (`PUT /execution/control`). An operator with curl can assess and control the system.

### 3. Domain Audit Trail: SUFFICIENT

Timestamps, correlation IDs, gate metadata, and fill records with simulated flags are present at all domain boundaries. NATS streams retain events for 72 hours with durable consumers.

### 4. Error Classification: SUFFICIENT

Problem type system with retryable flags, venue error code overrides, and post-200 reconciliation provides production-grade error taxonomy.

### 5. Safety Gate Correctness: SUFFICIENT

Kill switch, staleness guard, and gate transitions are proven stable under S349 endurance conditions. Counter invariant `processed == filled + skipped_halt + skipped_stale + errors` holds across ~96 events over ~15 minutes.

## Gaps and Priorities

| # | Gap | Severity | Minimum Fix | Priority |
|---|-----|----------|-------------|----------|
| 1 | No time-series metric export (Prometheus/OTEL) | HIGH (unattended) | ~100 LOC `/metrics` endpoint | P1 |
| 2 | No push-based alerting (webhook/PagerDuty/Slack) | HIGH (unattended) | Infrastructure (Alertmanager) | P1 |
| 3 | No NATS consumer lag visibility | MEDIUM | ~30 LOC in `/statusz` | P2 |
| 4 | No production latency histograms | MEDIUM | ~20 LOC in health tracker | P2 |
| 5 | No log aggregation assumed | MEDIUM | Deploy decision (not code) | P2 |
| 6 | No resource profiling alerts | LOW | ~10 LOC (Go runtime collector) | P3 |

**Key finding**: The code already has the right internal abstractions (typed counters, structured logs, health phases). The gaps are in *export and integration* — connecting existing signals to external alerting infrastructure — not in signal generation itself.

## Remaining Limitations

| Limitation | Severity | Note |
|-----------|----------|------|
| Assessment is static analysis, not runtime validation | Medium | Signal inventory verified by code reading and S349 evidence, not by running production traffic |
| No multi-instance signal correlation assessed | Low | Current architecture is single-instance per execution family |
| No retention/rotation policy for logs assessed | Low | Log rotation is a deployment concern, not code |
| Alert threshold values not specified | Medium | Thresholds require production baseline data that does not yet exist |
| Assessment covers venue-active path only | Low | Other subsystems (writer, codegen) are out of scope for S350 |

## Files Changed

### New Files

| File | Purpose |
|------|---------|
| `docs/architecture/monitoring-alertability-and-operational-signals-assessment.md` | Complete signal inventory with operational usefulness assessment and alertability analysis |
| `docs/architecture/current-signals-sufficiency-gaps-and-priority-findings.md` | Sufficiency verdict, concrete gaps with severity and minimum fix estimates, inflation guard |
| `docs/stages/stage-s350-monitoring-and-alertability-assessment-report.md` | This report |

### Modified Files

| File | Change |
|------|--------|
| `docs/stages/INDEX.md` | Added S350 entry |

## Inflation Guard

This stage explicitly avoided:
- Proposing a full observability platform (OTEL collector, Jaeger, distributed tracing)
- Opening a dashboarding or APM initiative
- Recommending multi-environment monitoring federation
- Hiding signal insufficiencies behind "future work" hand-waving

The gaps are real, specific, and scoped. The minimum viable path is documented.

## Preparation for S351

The monitoring assessment establishes the foundation for deployment and smoke automation assessment:

1. **Signal sufficiency for smoke automation**: `/statusz` and `/activation/surface` are sufficient for scripted smoke checks without requiring metric export (Gap 1).
2. **Deployment readiness signals**: `/healthz` and `/readyz` are ready for container orchestration probes.
3. **Smoke script foundation**: `scripts/smoke-activation.sh` already exists and can be extended with HTTP-based health assertions.
4. **Suggested S351 scope**: Assess whether the current smoke script, health endpoints, and deployment artifacts are sufficient for automated deployment validation — without opening a CI/CD platform initiative.

## Acceptance Criteria Evaluation

| Criterion | Met? | Evidence |
|-----------|------|---------|
| Honest assessment of signal sufficiency | Yes | Verdict: sufficient for attended, insufficient for unattended — with specific gaps |
| Monitoring/alertability gaps explicit | Yes | Six gaps documented with severity, minimum fix, and priority |
| Avoids inflation to observability platform | Yes | Inflation guard section; no OTEL/Jaeger/APM recommended |
| Ready for deployment/smoke automation assessment | Yes | Existing HTTP surfaces and smoke script identified as S351 foundation |
