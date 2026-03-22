# Production Readiness Evidence Matrix, Residual Gaps, and Next Ceremony

> Companion to the [Production Readiness Assessment Gate](production-readiness-assessment-gate.md).
> This document provides the objective evidence matrix, catalogs residual gaps,
> and recommends the next strategic ceremony.

**Date**: 2026-03-22
**Wave**: Production Readiness Assessment (S347–S352)

---

## 1. Evidence Matrix

### 1.1 Live Testnet Connectivity (PRA-1, S348)

| Evidence Item | Result | Source |
|---------------|--------|--------|
| DNS resolution to Binance CDN | PASS | LTC-1 |
| TCP connection on port 443 | PASS | LTC-1 |
| TLS 1.2+ handshake with valid cert chain | PASS | LTC-2 |
| Public endpoint /fapi/v1/time → 200 | PASS | LTC-3 |
| Unauthenticated private endpoint → 4xx | PASS | LTC-3 |
| Invalid credentials → structured error, no leakage | PASS | LTC-4 |
| Valid credentials → venue response (conditional) | CONDITIONAL | LTC-5 (requires real API key) |
| Credential loading fail-fast on missing | PASS | LTC-6 |
| Activation surface under credential variations | PASS | LTC-7 |
| Adapter timeout → Unavailable+Retryable | PASS | LTC-8 |

**Connectivity verdict**: 9/10 unconditional PASS, 1 conditional (real credentials).

### 1.2 Credential Handling (PRA-1, S348)

| Evidence Item | Result | Source |
|---------------|--------|--------|
| Env-var convention (MF_VENUE_{TYPE}_{KEY}) | VERIFIED | S348 credential model |
| Fail-fast on missing credentials | VERIFIED | LTC-6 |
| HMAC-SHA256 per-request signing | VERIFIED | Adapter code review |
| No credential in logs or errors | VERIFIED | LTC-4, security audit |
| Process-immutable lifetime | VERIFIED | Architecture review |
| Activation surface reflects credential state | VERIFIED | LTC-7, RVA-6 |

**Credential verdict**: Model is minimal, secure, and documented. Five known risks cataloged.

### 1.3 Endurance and Sustained Activation (PRA-2, S349)

| Evidence Item | Result | Source |
|---------------|--------|--------|
| END-1: 5-min sustained active, 20 events | PASS | S349 |
| END-2: 5-min mixed workload, 9 phases, 36 events | PASS | S349 |
| END-3: 5-min burst cycles, 10×4 events | PASS | S349 |
| Counter invariant (processed == filled + skipped_halt) | HOLDS | All 3 tests |
| Zero idle drift (12 pauses) | HOLDS | END-1, END-2, END-3 |
| Venue/fill parity (venueReqs == filled) | HOLDS | All 3 tests |
| No latency regression (last-third < 3× first-third) | HOLDS | END-1 |
| Gate transitions clean (4 transitions) | HOLDS | END-2 |
| Burst resilience (10 cycles) | HOLDS | END-3 |
| Error counter stable at zero | HOLDS | All 3 tests |
| Total events validated | ~96 | Combined |

**Endurance verdict**: All invariants hold. Zero drift, zero errors. Window is 5 minutes, not hours.

### 1.4 Monitoring and Alertability (PRA-3, S350)

| Signal Category | Coverage | Operational Usefulness |
|----------------|----------|----------------------|
| Structured logs (slog) | 15+ distinct messages | HIGH — post-hoc investigation |
| Health tracker counters | 13 named counters | HIGH — threshold alerting |
| HTTP query surfaces | 7 endpoints | HIGH — manual operation |
| Domain audit fields | 5 types with timestamps | HIGH — traceability |
| NATS control plane | 3 req/reply + 2 streams + 1 KV | MEDIUM-HIGH — replay/audit |
| Error classification | 7+ categories with retryable flags | HIGH — operational |
| Computed health phases | 6 phases | MEDIUM-HIGH — status |

**Alertable signals identified (pull-based today)**:

| Signal | Source | Alert Type |
|--------|--------|-----------|
| System down | /healthz non-200 | Liveness |
| System not ready | /readyz non-200 | Readiness |
| Gate halted unexpectedly | /activation/surface | State |
| Counter invariant violation | /statusz computed | Safety |
| Error rate spike | /statusz rate-of-change | Operational |
| Component stalled | /statusz phase | Health |
| Retry exhaustion | Log pattern | Error |

**Monitoring verdict**: Sufficient for attended operation. NOT sufficient for unattended.

### 1.5 Deployment and Smoke Automation (PRA-4, S351)

| Evidence Item | Result | Source |
|---------------|--------|--------|
| make bootstrap validates 8 tools | READY | S351 |
| make docker-build builds 8 services | READY | S351 |
| make up brings stack with migrations | READY | S351 |
| make seed/seed-multi configures lifecycle | READY | S351 |
| make live orchestrates build+up+seed+validate | READY | S351 |
| make down idempotent teardown | READY | S351 |
| 9 smoke targets with shared conventions | READY | S351 |
| Security: 127.0.0.1, cap_drop:ALL, no-new-privileges | VERIFIED | S351 |
| 3-command zero-to-validated path | VERIFIED | S351 |

**Deployment verdict**: LOCAL fully automated. CI integration has 4 P2 gaps (~90 LOC).

### 1.6 Safety Gates (Cross-Wave Validation)

| Safety Mechanism | Proven In | Status |
|-----------------|-----------|--------|
| Kill switch (gate halt) | RVA-1, RVA-3, RVA-4, END-2 | PROVEN — zero venue HTTP requests when halted |
| Staleness guard | S343, S349 | PROVEN — stale intents skipped |
| Counter invariant | S343, S349 (all 3 tests) | PROVEN — holds across ~96 events, gate transitions |
| Error classification | S348, RVA-5 | PROVEN — 7+ categories with correct retryable flags |
| Credential security | S348, LTC-4 | PROVEN — no leakage in any error path |

---

## 2. Residual Gaps

### 2.1 HIGH Severity (Block Unattended Operation)

| ID | Gap | Current State | Minimum Fix | Effort |
|----|-----|---------------|-------------|--------|
| RG-1 | No metric export (Prometheus/OTEL) | In-memory atomics, HTTP pull only | `/metrics` endpoint | ~100 LOC |
| RG-2 | No push-based alerting | All signals pull-based | Alertmanager deployment | Infrastructure |

### 2.2 MEDIUM Severity (Operational Maturity)

| ID | Gap | Current State | Minimum Fix | Effort |
|----|-----|---------------|-------------|--------|
| RG-3 | No CI integration pipeline | Local-only smoke execution | Machine-readable output + aggregate target + artifact collection | ~90 LOC |
| RG-4 | Endurance at 5 minutes, not hours | httptest.Server, minute-scale | Hours-scale soak test | Test design |
| RG-5 | No startup credential validation | First failure on first SubmitOrder | Lightweight venue ping at startup | ~20 LOC |
| RG-6 | No NATS consumer lag visibility | JetStream tracks internally | Query consumer info in /statusz | ~30 LOC |
| RG-7 | No production latency histograms | Log timestamps only | Duration recording in tracker | ~20 LOC |
| RG-8 | No log aggregation | Logs to stdout | Deployment decision | Infrastructure |
| RG-9 | Timeout variables fragmented | 3 variables with different defaults | Centralize in lib.sh | ~30 LOC |

### 2.3 LOW Severity (Ergonomic / Deferred)

| ID | Gap | Current State | Minimum Fix | Effort |
|----|-----|---------------|-------------|--------|
| RG-10 | No port availability pre-check | Fails at bind time | Check in lib.sh | ~15 LOC |
| RG-11 | CLICKHOUSE_DSN undocumented | Hardcoded default in script | Add to local.env | 1 line |
| RG-12 | Venue credentials not in automation | Manual env setup | Script helper | ~10 LOC |
| RG-13 | No resource profiling alerts | /diagz goroutines only | Go runtime metrics | ~10 LOC |
| RG-14 | No credential rotation | Restart required | Acceptable for testnet | Deferred |
| RG-15 | No credential expiration awareness | Discovered on first failure | Periodic health probe | Deferred |

### 2.4 Gap Priority Summary

| Priority | Count | Total Effort | Impact |
|----------|-------|-------------|--------|
| P1 (unattended blocker) | 2 | ~100 LOC + infra | Unlocks monitoring |
| P2 (operational maturity) | 7 | ~190 LOC + infra | Unlocks CI, hours-scale |
| P3 (ergonomic) | 6 | ~35 LOC + decisions | Quality of life |

**Total implementation gap**: ~325 LOC + infrastructure decisions.

---

## 3. Regression Verification

### 3.1 Module Health

| Check | Result |
|-------|--------|
| 17/17 Go modules pass `go vet` | PASS |
| Domain tests (8 packages) | ALL PASS |
| Application tests (16 packages) | ALL PASS |
| HTTP interface tests (2 packages) | ALL PASS |

### 3.2 Pre-Existing Test Preservation

No pre-existing test was modified or broken during this wave.
All wave-produced tests are additive (new files with appropriate build tags).

### 3.3 API Contract Stability

No existing API signatures, interfaces, or contracts were modified.
New additions:
- `GET /activation/surface` — additive HTTP endpoint
- `ActivationSurface` type — new domain type
- `GetActivationSurfaceUseCase` — new use case

### 3.4 Regression Verdict

**ZERO regressions.** The wave was assessment-only with additive code changes.

---

## 4. Wave Scorecard

| Dimension | Score | Evidence |
|-----------|-------|----------|
| Governing questions answered | 15/15 | 12 YES, 1 NO (honest), 1 PARTIAL, 1 YES-deferred |
| Capabilities classified | 4/4 | 0 FULL, 3 SUBSTANTIAL, 1 PARTIAL |
| Non-goals respected | 10/10 | All held |
| Guard rails held | 8/8 | All held |
| Regressions | 0 | Zero |
| Scope inflation | 0 | Wave boundary respected |

---

## 5. Honest Assessment: What the Foundry Can and Cannot Do Today

### CAN DO (Proven by Evidence)

1. **Connect to real Binance Futures testnet** with proper DNS, TLS, authentication
2. **Sustain venue activation** for 5+ minutes with zero counter drift and zero errors
3. **Classify venue errors** correctly (auth, rate limit, timeout, venue codes)
4. **Gate venue access** instantly via kill switch (zero HTTP requests when halted)
5. **Deploy locally** from zero to validated stack in 3 commands
6. **Run 9 smoke targets** with consistent conventions and reliable exit codes
7. **Query activation state** via HTTP with full audit fields
8. **Maintain safety invariants** under mixed workloads, bursts, idle pauses, and gate transitions
9. **Operate attended** — an operator with curl can assess, control, and investigate

### CANNOT DO (Honest Gaps)

1. **Operate unattended** — no metric export, no push alerting
2. **Prove hours-scale stability** — only 5 minutes demonstrated
3. **Run in CI** — no machine-readable output, no aggregate target
4. **Validate credentials at startup** — first failure delayed to first order
5. **Export latency baselines** — tracked in tests but not in production runtime
6. **Monitor consumer lag** — JetStream tracks internally but doesn't expose
7. **Aggregate logs** — stdout only, no aggregation infrastructure

---

## 6. Recommendation for Next Ceremony

### 6.1 Strategic Assessment

The Production Readiness Assessment Wave achieved its objective: honest evaluation
of operational readiness. The findings reveal that:

- **Domain correctness** is not the bottleneck (activation model, gates, safety invariants all proven)
- **Operational infrastructure** is the bottleneck (~325 LOC + infrastructure decisions)
- The gap between "attended operation" and "unattended operation" is concrete and bounded

### 6.2 Recommended Next Wave: **Operational Foundation Wave**

**Objective**: Close the minimum gaps that block unattended operation and CI integration.

**Suggested scope (bounded)**:

| Block | Description | Effort |
|-------|-------------|--------|
| OF-1 | Prometheus /metrics endpoint | ~100 LOC |
| OF-2 | CI smoke integration (machine-readable output, aggregate target, artifacts) | ~90 LOC |
| OF-3 | Consumer lag + latency histograms in /statusz | ~50 LOC |
| OF-4 | Startup credential validation | ~20 LOC |
| OF-5 | Evidence gate for operational foundation | Assessment |

**Total estimated implementation**: ~260 LOC

### 6.3 Explicit Non-Goals for Next Wave

- Mainnet activation
- Multi-venue expansion
- Full observability platform (OTEL, Jaeger, distributed tracing)
- Custom dashboards
- Hours-scale soak testing (should be a separate assessment after OF-1–OF-4)
- Strategy/signal integration
- OMS or portfolio management

### 6.4 Decision Point

Before opening the next wave, the project owner should decide:

1. **Is unattended operation the priority?** If yes → OF-1 (metrics) + Alertmanager
2. **Is CI integration the priority?** If yes → OF-2 (machine-readable smoke)
3. **Are both needed before hours-scale testing?** If yes → both OF-1 and OF-2 first
4. **Is hours-scale endurance testing the priority?** If yes → soak test design first

The evidence supports any of these priorities. The gate does not prescribe the order —
it provides the facts for an informed decision.

---

## References

- [Production Readiness Assessment Gate](production-readiness-assessment-gate.md)
- [S347 Charter](../stages/stage-s347-production-readiness-assessment-charter-report.md)
- [S348 Live Testnet Assessment](live-testnet-connectivity-and-credential-handling-assessment.md)
- [S349 Endurance Assessment](endurance-and-sustained-activation-assessment.md)
- [S350 Monitoring Assessment](monitoring-alertability-and-operational-signals-assessment.md)
- [S351 Deployment Assessment](deployment-automation-and-smoke-automation-assessment.md)
- [S346 Venue Activation Evidence Gate](venue-activation-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Credential Handling Risks](credential-gated-operation-risks-ergonomics-and-limitations.md)
- [Current Signals Sufficiency](current-signals-sufficiency-gaps-and-priority-findings.md)
- [Reproducibility Gaps](reproducibility-automation-gaps-and-operational-frictions.md)
