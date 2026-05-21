# Operational Foundation — Evidence Matrix, Residual Gaps, and Next Ceremony

> Companion to the [Operational Foundation Evidence Gate](operational-foundation-evidence-gate.md).
> Maps every S352 residual gap to its wave outcome, classifies residual gaps by
> actionability, and recommends the next strategic ceremony.
>
> Date: 2026-03-22.
> Wave: Operational Foundation (S353–S357).

---

## 1. S352 Residual Gap Disposition

| S352 Gap | Severity | OF Block | Wave Outcome | Status |
|----------|----------|----------|--------------|--------|
| RG-1: No Prometheus `/metrics` endpoint | HIGH | OF-1 | `/metrics` on all binaries via gateway routes + HealthServer; HTTP + consumer metrics exported | **CLOSED** |
| RG-2: No push alerting | HIGH | — | Out of scope (NG-10); metrics export enables external alerting | **OPEN — deferred by design** |
| RG-3: No CI smoke integration | MEDIUM | OF-2 | 7 CI jobs; `smoke-composed` + `smoke-analytical` + `repository-checks` in pipeline | **CLOSED** |
| RG-4: Endurance 5 min not hours | MEDIUM | — | Out of scope (NG-11); requires its own wave | **OPEN — deferred by design** |
| RG-5: No startup credential validation | MEDIUM | OF-4 | `RunPreflight()` in all 7 binaries; 13 unit tests; fail-fast with structured error | **CLOSED** |
| RG-6: No NATS consumer lag visibility | MEDIUM | OF-3 | `marketfoundry_consumer_lag_messages` gauge from `meta.NumPending` on every message | **CLOSED** |
| RG-7: No production latency histograms | MEDIUM | OF-3 | `marketfoundry_consumer_processing_duration_seconds` (12 buckets) + `marketfoundry_http_request_duration_seconds` (11 buckets) | **CLOSED** |
| RG-8: No log aggregation | MEDIUM | — | Out of scope (NG-9); logs remain stdout | **OPEN — infrastructure decision** |
| RG-9: Timeout variables fragmented | MEDIUM | OF-2 | `ci-wait-ready.sh` centralizes polling; configurable timeout | **CLOSED** |
| RG-10: No port pre-check | LOW | — | Not addressed; low priority | **OPEN — deferred** |
| RG-11: CLICKHOUSE_DSN undocumented | LOW | — | Writer preflight validates ClickHouse config presence; DSN docs not added | **PARTIAL — config validated, not documented** |
| RG-12: Venue credentials not in automation | LOW | — | Credentials validated at adapter build; not in CI automation | **OPEN — deferred** |
| RG-13: No resource profiling alerts | LOW | — | Not addressed; requires Alertmanager (NG-10) | **OPEN — deferred** |
| RG-14: No credential rotation | LOW | — | Out of scope (NG-12); credentials process-immutable by design | **OPEN — by design** |
| RG-15: No credential expiration awareness | LOW | — | Out of scope (NG-12) | **OPEN — by design** |

### Disposition Summary

| Status | Count | Gaps |
|--------|-------|------|
| **CLOSED** | 6 | RG-1, RG-3, RG-5, RG-6, RG-7, RG-9 |
| **PARTIAL** | 1 | RG-11 |
| **OPEN — deferred by design** | 4 | RG-2, RG-4, RG-14, RG-15 |
| **OPEN — infrastructure decision** | 1 | RG-8 |
| **OPEN — deferred** | 3 | RG-10, RG-12, RG-13 |

---

## 2. Capability Classification

| Capability | Classification | Justification |
|------------|---------------|---------------|
| Prometheus `/metrics` endpoint | **FULL** | All binaries expose `/metrics`; HTTP + consumer metrics instrumented; tested |
| Consumer lag visibility | **FULL** | Gauge updated from JetStream metadata on every message; exported via Prometheus |
| Latency histograms | **FULL** | Consumer processing + HTTP request histograms with proper bucket distributions |
| CI smoke integration | **FULL** | 7 CI jobs; stackless + stack-dependent coverage; readiness polling centralized |
| Startup credential validation | **FULL** | All 7 binaries run `RunPreflight()`; fail-fast with structured errors; 13 tests |
| Health tracker Prometheus export | **SUBSTANTIAL** | HTTP/consumer metrics exported; health tracker phase/idle state remains `/statusz`-only |
| Aggregate smoke target | **SUBSTANTIAL** | `make ci-smoke` for stackless; stack-dependent runs as separate CI jobs (architectural split, not gap) |
| Push alerting | **PENDING** | Deferred (NG-10); metric export enables external integration |
| Hours-scale endurance | **PENDING** | Deferred (NG-11); requires separate wave |
| Log aggregation | **PENDING** | Deferred (NG-9); infrastructure decision |
| Credential rotation | **PENDING** | Deferred (NG-12); by design |

---

## 3. Residual Gaps — Honest Assessment

### Gaps That Matter for Next Macro-Front

1. **RG-2 (HIGH): No push alerting.** Metrics export is necessary but not sufficient
   for unattended operation. Without Alertmanager or equivalent, metrics only help
   during active observation. This is the single highest-priority residual gap, but
   it is correctly an infrastructure decision, not an application concern.

2. **RG-4 (MEDIUM): Endurance limited to minutes.** The Foundry has been tested for
   5-minute sustained operation. Hours-scale soak would prove stability under
   realistic conditions. This matters for production confidence but requires its
   own assessment scope.

### Gaps That Are Ergonomic, Not Structural

3. **RG-8 (MEDIUM): No log aggregation.** Logs are structured and go to stdout.
   Aggregation is a deployment concern (ELK, Loki, etc.) not an application gap.

4. **RG-11 (LOW): ClickHouse DSN documentation.** Config is validated at startup;
   documentation is ergonomic, not structural.

5. **RG-10, RG-12, RG-13 (LOW):** Port pre-check, credential automation, resource
   profiling — all ergonomic improvements that don't block operation.

### Gaps That Are Architectural Decisions

6. **RG-14, RG-15 (LOW): Credential rotation/expiration.** Process-immutable
   credentials are a deliberate design choice for this stage. Rotation would
   require runtime credential refresh, which is a significant architectural change
   best deferred until the deployment model evolves.

---

## 4. What the Wave Proved

1. **The Foundry can be observed.** Prometheus metrics cover HTTP requests, consumer
   throughput, consumer lag, and processing latency. Any Prometheus-compatible
   monitoring stack can scrape and alert on these signals.

2. **The Foundry fails fast on misconfiguration.** All 7 binaries validate NATS
   connectivity, URL format, and service-specific configuration before any I/O.
   Bad configuration produces immediate, actionable errors.

3. **CI catches regressions automatically.** Seven CI jobs cover unit tests,
   integration tests, codegen validation, behavioral scenarios, stackless smoke,
   repository consistency, and full E2E analytical smoke.

4. **The operational foundation holds under the existing test envelope.** Zero
   regressions across 38 packages. All guard rails held. All non-goals respected.

---

## 5. What the Wave Did Not Prove

1. **Hours-scale stability.** The longest test runs ~5 minutes. Production requires
   hours-to-days stability evidence.

2. **Alerting works end-to-end.** Metrics are exported but no alert rule has been
   tested. The Foundry can be scraped, but nobody is watching.

3. **Recovery from infrastructure failure.** NATS/ClickHouse outage scenarios are
   not tested. The Foundry's behavior under infrastructure loss is unknown.

4. **Deployment automation.** Docker Compose is the only deployment model. No
   orchestration, no rolling updates, no health-based restarts.

---

## 6. Next Ceremony Recommendation

### Recommended: Strategy and Signal Integration Wave

**Rationale:**

The Operational Foundation Wave closed the operational infrastructure gaps that
were blocking unattended operation. The Foundry now exports metrics, validates
configuration, and integrates smoke into CI. The remaining residual gaps (push
alerting, hours-scale endurance, log aggregation) are infrastructure and
deployment concerns, not application-level blockers.

The highest-leverage next step is to advance the Foundry's **functional
capability** — specifically, connecting the execution domain to strategy and
signal sources. This is the domain work that the operational foundation was built
to support.

**However**, the choice of next wave depends on the project owner's priorities:

| Option | Focus | Prerequisite | Risk |
|--------|-------|--------------|------|
| **A. Strategy/Signal Integration** | Connect execution to strategy signals | Operational foundation (done) | Domain complexity; requires signal model design |
| **B. Deployment Hardening** | K8s/Helm, health-based restarts, rolling updates | Operational foundation (done) | Infrastructure scope; may not advance domain |
| **C. Extended Endurance** | Hours-scale soak, recovery testing | Operational foundation (done) | Valuable but delays domain progress |
| **D. Multi-Venue Expansion** | Add venue adapters beyond Binance testnet | Single-venue proven | Significant breadth; risky without strategy integration |

**Recommendation**: Option A (Strategy/Signal Integration) is the highest-leverage
choice. The operational foundation exists to support domain work. Deferring domain
advancement in favor of more infrastructure polish risks gold-plating the
foundation without delivering business value.

**Next action**: Open a new charter ceremony (S358) to define the scope, governing
questions, and blocks for the chosen wave. The charter must not inherit
Operational Foundation scope — it starts fresh.

---

## References

- [Operational Foundation Evidence Gate](operational-foundation-evidence-gate.md)
- [Operational Foundation Wave Charter](operational-foundation-wave-charter-and-scope-freeze.md)
- [Exit Criteria and Non-Goals](operational-foundation-items-exit-criteria-and-non-goals.md)
- [S352 — Production Readiness Assessment Gate](../stages/stage-s352-production-readiness-assessment-gate-report.md)
- [S354 — Metrics Foundation Report](../stages/stage-s354-metrics-and-operational-signals-foundation-report.md)
- [S355 — CI Smoke Integration Report](../stages/stage-s355-ci-smoke-integration-report.md)
- [S356 — Startup Credential Validation Report](../stages/stage-s356-startup-credential-validation-report.md)
