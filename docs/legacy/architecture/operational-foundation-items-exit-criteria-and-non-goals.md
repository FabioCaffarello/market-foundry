# Operational Foundation Items — Exit Criteria and Non-Goals

> Companion to the [Operational Foundation Wave Charter](operational-foundation-wave-charter-and-scope-freeze.md).
> Defines objective exit criteria per block, explicit non-goals, and guard rails.
>
> Date: 2026-03-22.
> Wave: Operational Foundation (S353–S358).

---

## 1. Exit Criteria Per Block

### OF-1 — Prometheus /metrics Endpoint

| # | Criterion | Evidence Required |
|---|-----------|-------------------|
| EC-1.1 | `/metrics` endpoint responds with HTTP 200 | `curl -s localhost:<port>/metrics` returns Prometheus text format |
| EC-1.2 | All 13 health-tracker counters appear in output | Each named counter visible in `/metrics` response |
| EC-1.3 | Metrics are type-annotated (counter, gauge) | `# TYPE` lines present for each metric family |
| EC-1.4 | Existing endpoints unaffected | `/healthz`, `/readyz`, `/statusz`, `/activation/surface` still work |
| EC-1.5 | Zero regressions in existing test suite | `go vet` + unit tests pass across all 17 modules |

**Done when**: OFQ-1 and OFQ-2 answered YES with evidence.

### OF-2 — CI Smoke Integration

| # | Criterion | Evidence Required |
|---|-----------|-------------------|
| EC-2.1 | Each smoke script exits with 0 (pass) or non-zero (fail) | Exit code verified for each of 9 targets |
| EC-2.2 | Machine-readable output produced | JSON or TAP output file per smoke target |
| EC-2.3 | `make smoke-ci` runs all targets and collects results | Single command, aggregate exit code, artifacts in output directory |
| EC-2.4 | Timeout variables centralized | Single source of truth in `lib.sh`, no fragmented defaults |
| EC-2.5 | Existing `make smoke-*` targets unbroken | Manual smoke targets still work as before |

**Done when**: OFQ-3 and OFQ-4 answered YES with evidence.

### OF-3 — Consumer Lag + Latency Histograms

| # | Criterion | Evidence Required |
|---|-----------|-------------------|
| EC-3.1 | `/statusz` shows NATS consumer pending count | JSON response includes `consumer_lag` or equivalent field |
| EC-3.2 | Consumer lag exported as Prometheus gauge | Metric visible in `/metrics` output |
| EC-3.3 | Order-submission latency recorded as histogram | `_duration_seconds` histogram with buckets in `/metrics` |
| EC-3.4 | Latency recording does not degrade submission path | No measurable overhead (same order of magnitude as before) |
| EC-3.5 | Zero regressions in existing test suite | All tests pass |

**Done when**: OFQ-5 and OFQ-6 answered YES with evidence.

### OF-4 — Startup Credential Validation

| # | Criterion | Evidence Required |
|---|-----------|-------------------|
| EC-4.1 | Adapter validates credentials at init | Code path executes venue ping before accepting orders |
| EC-4.2 | Missing credentials → fail-fast with structured error | Test: omit credentials, observe structured error and process exit |
| EC-4.3 | Invalid credentials → fail-fast with structured error | Test: provide bad credentials, observe structured error |
| EC-4.4 | Valid credentials → startup proceeds | Test: provide valid credentials (or mock), observe normal startup |
| EC-4.5 | No credential leakage in validation error path | Error message does not contain API key or secret |

**Done when**: OFQ-7 and OFQ-8 answered YES with evidence.

### OF-5 — Evidence Gate

| # | Criterion | Evidence Required |
|---|-----------|-------------------|
| EC-5.1 | All 8 governing questions answered | Evidence matrix complete |
| EC-5.2 | Zero regressions | `go vet` + full test suite across 17 modules |
| EC-5.3 | Non-goal compliance verified | Each non-goal audited |
| EC-5.4 | Formal verdict issued | COMPLETE / PARTIAL / INCOMPLETE with justification |
| EC-5.5 | Next wave recommended | Based on residual gaps and evidence |

---

## 2. Explicit Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-1 | **Mainnet activation** | Testnet-only scope; mainnet requires its own wave |
| NG-2 | **Multi-venue expansion** | Single venue (Binance Futures Testnet) not yet operationally proven |
| NG-3 | **OMS integration** | Execution is submission-only; OMS sits above |
| NG-4 | **Portfolio risk management** | Sits above execution domain |
| NG-5 | **Broad dashboards (Grafana, custom UI)** | Wave adds metric export, not visualization |
| NG-6 | **New functional breadth** | No new domain types beyond what OF blocks require |
| NG-7 | **Strategy/signal integration** | Depends on operational readiness verdict |
| NG-8 | **Full observability platform (OTEL, Jaeger, distributed tracing)** | Prometheus `/metrics` only; no tracing, no OTEL collector |
| NG-9 | **Log aggregation infrastructure** | Logs remain stdout; aggregation is a deployment decision |
| NG-10 | **Alertmanager / push alerting deployment** | Wave exports metrics; alert routing is infrastructure, not application |
| NG-11 | **Hours-scale soak testing** | Separate concern; requires its own assessment after operational foundation |
| NG-12 | **Credential rotation or expiration handling** | Credentials remain process-immutable by design |
| NG-13 | **Infrastructure platform changes (K8s, Helm, Terraform)** | Application-level changes only |
| NG-14 | **CI/CD pipeline construction** | Wave makes smoke CI-ready; actual pipeline is DevOps scope |
| NG-15 | **Chaos engineering / fault injection** | Requires the stable operational baseline this wave establishes |

---

## 3. Guard Rails

| # | Guard Rail | Enforcement |
|---|-----------|-------------|
| GR-1 | No mainnet endpoints in any code path | Code review; testnet-only configuration |
| GR-2 | No new venue adapters | Single BinanceFuturesTestnet adapter only |
| GR-3 | No new domain types unless directly required by an OF block | Each new type must trace to an exit criterion |
| GR-4 | No architectural redesign | Decorator pipeline, actor model, NATS topology fixed |
| GR-5 | No scope expansion after S353 | New blocks require a new charter |
| GR-6 | Dependency ordering is binding | OF-3 depends on OF-1; OF-5 depends on all |
| GR-7 | `/metrics` endpoint only — no OTEL collector, no Jaeger | Prometheus client library only |
| GR-8 | CI smoke output only — no pipeline construction | Machine-readable output and aggregate target, not CI YAML |
| GR-9 | Credential validation is lightweight — no retry loops, no background probes | Single synchronous ping at startup |
| GR-10 | No dashboard construction | Export metrics; do not build visualization |

---

## 4. Residual Gap Coverage Map

This table maps which S352 residual gaps each OF block addresses:

| S352 Gap | Severity | OF Block | Status After Wave |
|----------|----------|----------|-------------------|
| RG-1: No Prometheus /metrics | HIGH | OF-1 | CLOSED |
| RG-2: No push alerting | HIGH | — | OPEN (NG-10: infrastructure decision) |
| RG-3: No CI smoke integration | MEDIUM | OF-2 | CLOSED |
| RG-4: Endurance 5 min not hours | MEDIUM | — | OPEN (NG-11: separate assessment) |
| RG-5: No startup credential validation | MEDIUM | OF-4 | CLOSED |
| RG-6: No NATS consumer lag visibility | MEDIUM | OF-3 | CLOSED |
| RG-7: No production latency histograms | MEDIUM | OF-3 | CLOSED |
| RG-8: No log aggregation | MEDIUM | — | OPEN (NG-9: deployment decision) |
| RG-9: Timeout variables fragmented | MEDIUM | OF-2 | CLOSED |
| RG-10: No port pre-check | LOW | — | OPEN (deferred) |
| RG-11: CLICKHOUSE_DSN undocumented | LOW | — | OPEN (deferred) |
| RG-12: Venue credentials not in automation | LOW | — | OPEN (deferred) |
| RG-13: No resource profiling alerts | LOW | — | OPEN (deferred) |
| RG-14: No credential rotation | LOW | — | OPEN (NG-12: by design) |
| RG-15: No credential expiration awareness | LOW | — | OPEN (NG-12: by design) |

**Wave closes**: 6 of 15 gaps (2 HIGH → 1 closed + 1 deferred, 7 MEDIUM → 5 closed).

**After wave**: 9 gaps remain open (1 HIGH, 2 MEDIUM, 6 LOW). All are infrastructure
decisions, ergonomic improvements, or explicitly deferred.

---

## 5. What This Wave Does and Does Not Change

### Changes

- Gateway gains `/metrics` endpoint (new HTTP route, Prometheus client)
- Health tracker counters become Prometheus-exportable
- NATS consumer lag becomes observable via `/statusz` and `/metrics`
- Order-submission latency becomes a histogram in `/metrics`
- Smoke scripts gain machine-readable output mode
- `make smoke-ci` aggregate target created
- Venue adapter validates credentials at startup

### Does Not Change

- Domain model (activation, execution, gates, safety invariants)
- Actor topology (decorator pipeline, venue adapter, query responder)
- NATS topology (streams, subjects, KV stores)
- HTTP API contracts (existing endpoints unchanged)
- Deployment model (Docker Compose, local-only)
- Security model (127.0.0.1 binding, cap_drop:ALL, credential handling)

---

## References

- [Operational Foundation Wave Charter](operational-foundation-wave-charter-and-scope-freeze.md)
- [S352 — Production Readiness Assessment Gate Report](../stages/stage-s352-production-readiness-assessment-gate-report.md)
- [Evidence Matrix and Residual Gaps](production-readiness-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Monitoring Assessment](monitoring-alertability-and-operational-signals-assessment.md)
- [Deployment Assessment](deployment-automation-and-smoke-automation-assessment.md)
