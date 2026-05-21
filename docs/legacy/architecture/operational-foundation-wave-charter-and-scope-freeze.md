# Operational Foundation Wave — Charter and Scope Freeze

> Opens the Operational Foundation Wave with frozen scope, governing questions,
> ordered block plan, and binding constraints.
>
> Predecessor: S352 (Production Readiness Assessment Gate — SUBSTANTIAL delivery).
> Wave type: Implementation (bounded).
> Date: 2026-03-22.

---

## 1. Strategic Context

The Production Readiness Assessment Wave (S347–S352) proved that domain
correctness is not the bottleneck. The activation model, safety gates, counter
invariants, and error classification all hold under sustained operation.

The bottleneck is operational infrastructure:

| Blocker | Impact |
|---------|--------|
| No metric export | Cannot operate unattended |
| No CI smoke integration | Cannot validate automatically |
| No consumer lag / latency visibility | Cannot baseline production behavior |
| No startup credential validation | First failure delayed to first order |

Total estimated implementation: **~260 LOC**.

This wave closes exactly these four gaps. Nothing more.

---

## 2. Wave Identity

| Field | Value |
|-------|-------|
| **Wave name** | Operational Foundation Wave |
| **Wave type** | Implementation (bounded) |
| **Predecessor wave** | Production Readiness Assessment (S347–S352) |
| **Predecessor verdict** | SUBSTANTIAL (3 SUBSTANTIAL, 1 PARTIAL, 0 regressions) |
| **Opening stage** | S353 (this document) |
| **Estimated stages** | S353 (charter) → S354 → S355 → S356 → S357 → S358 (gate) |
| **Total estimated LOC** | ~260 |
| **Strategic goal** | Close the minimum gaps blocking unattended operation and CI integration |

---

## 3. Governing Questions

The wave is governed by 8 questions (OFQ-1 through OFQ-8). Each must receive
a YES/NO/PARTIAL answer backed by evidence before the wave can close.

### OF-1: Prometheus /metrics

| # | Question |
|---|----------|
| OFQ-1 | Does the gateway expose a `/metrics` endpoint in Prometheus exposition format? |
| OFQ-2 | Are the proven health-tracker counters (13 named) exported as Prometheus metrics? |

### OF-2: CI Smoke Integration

| # | Question |
|---|----------|
| OFQ-3 | Do smoke scripts produce machine-readable output (exit codes + structured results)? |
| OFQ-4 | Does an aggregate `make smoke-ci` target run all smoke targets and collect artifacts? |

### OF-3: Consumer Lag + Latency Histograms

| # | Question |
|---|----------|
| OFQ-5 | Does `/statusz` expose NATS JetStream consumer lag? |
| OFQ-6 | Are order-submission latencies recorded as histograms and exposed via `/metrics`? |

### OF-4: Startup Credential Validation

| # | Question |
|---|----------|
| OFQ-7 | Does the venue adapter validate credentials at startup before accepting orders? |
| OFQ-8 | Does startup validation fail-fast with a clear error if credentials are invalid? |

---

## 4. Executable Blocks

### OF-1 — Prometheus /metrics Endpoint (~100 LOC)

**Objective**: Expose all proven health-tracker counters and key runtime metrics
via a standard Prometheus `/metrics` endpoint on the gateway.

**Scope**:
- Add `prometheus/client_golang` dependency
- Register health-tracker counters as Prometheus gauges/counters
- Expose `/metrics` on gateway HTTP server
- Validate with `curl localhost:<port>/metrics | promhttp_metric_handler`

**Answers**: OFQ-1, OFQ-2.

**Dependencies**: None. Can start immediately.

### OF-2 — CI Smoke Integration (~90 LOC)

**Objective**: Make smoke scripts CI-friendly with machine-readable output,
an aggregate target, and artifact collection.

**Scope**:
- Add machine-readable output mode to `scripts/lib.sh` (JSON or TAP)
- Create `make smoke-ci` aggregate target
- Ensure all 9 existing smoke targets produce machine-readable results
- Collect artifacts (logs, exit codes) in a known output directory
- Centralize timeout variables (closes RG-9)

**Answers**: OFQ-3, OFQ-4.

**Dependencies**: None. Can start immediately. Parallel with OF-1.

### OF-3 — Consumer Lag + Latency Histograms (~50 LOC)

**Objective**: Expose NATS consumer lag and order-submission latency as
observable metrics.

**Scope**:
- Query JetStream consumer info and expose pending count in `/statusz`
- Record order-submission duration in health tracker
- Expose latency as Prometheus histogram via `/metrics`
- Expose consumer lag as Prometheus gauge via `/metrics`

**Answers**: OFQ-5, OFQ-6.

**Dependencies**: OF-1 (needs `/metrics` endpoint to exist).

### OF-4 — Startup Credential Validation (~20 LOC)

**Objective**: Validate venue credentials at adapter startup before the system
accepts order submissions.

**Scope**:
- Add lightweight venue ping (e.g., `/fapi/v1/time` with auth headers) at adapter init
- Fail-fast with structured error if credentials are missing, malformed, or rejected
- Log validation result at startup

**Answers**: OFQ-7, OFQ-8.

**Dependencies**: None. Can start immediately. Parallel with OF-1, OF-2.

### OF-5 — Operational Foundation Evidence Gate (Assessment)

**Objective**: Compile evidence across OF-1 through OF-4, answer all 8
governing questions, issue wave verdict.

**Scope**:
- Evidence matrix for OFQ-1 through OFQ-8
- Regression audit (all 17 Go modules)
- Non-goal compliance verification
- Wave verdict (COMPLETE / PARTIAL / INCOMPLETE)
- Residual gaps catalog
- Next wave recommendation

**Dependencies**: OF-1, OF-2, OF-3, OF-4 all complete.

---

## 5. Sequencing and Stage Plan

```
S353 — Charter and scope freeze (this document)
  │
  ├── S354 — OF-1: Prometheus /metrics endpoint
  │     └── S355 — OF-3: Consumer lag + latency histograms (depends on OF-1)
  │
  ├── S354 ∥ S356 — OF-2: CI smoke integration (parallel with OF-1)
  │
  ├── S354 ∥ S357 — OF-4: Startup credential validation (parallel with OF-1)
  │
  └── S358 — OF-5: Operational foundation evidence gate
```

**Recommended execution order**:

| Stage | Block | Parallel? | Prerequisite |
|-------|-------|-----------|-------------|
| S353 | Charter | — | S352 |
| S354 | OF-1: /metrics | — | S353 |
| S355 | OF-3: lag + histograms | — | S354 |
| S356 | OF-2: CI smoke | Can parallel S354 | S353 |
| S357 | OF-4: credential validation | Can parallel S354 | S353 |
| S358 | OF-5: evidence gate | — | S354, S355, S356, S357 |

---

## 6. Freeze Conditions

The following constraints are binding for the duration of this wave:

| # | Condition |
|---|-----------|
| FC-1 | No new blocks may be added after S353 |
| FC-2 | Block scope is frozen as defined in section 4 |
| FC-3 | Non-goals list is frozen as defined in the companion document |
| FC-4 | Block ordering may be parallelized but not reordered in dependency chain |
| FC-5 | Any scope change requires a new charter stage |

---

## 7. Dependencies and Preconditions

| Precondition | Status | Notes |
|-------------|--------|-------|
| S352 closed with SUBSTANTIAL verdict | DONE | All 15 governing questions answered |
| 17/17 Go modules passing `go vet` | DONE | Verified at S352 |
| All existing tests green | DONE | Zero regressions at S352 |
| Residual gaps cataloged with effort estimates | DONE | 15 gaps, ~325 LOC total |
| `prometheus/client_golang` available | AVAILABLE | Standard Go library |
| NATS JetStream consumer info API available | AVAILABLE | Standard NATS Go client |

---

## 8. Success Criteria for Wave Closure

The wave may close when:

1. All 8 governing questions (OFQ-1 through OFQ-8) receive YES/NO/PARTIAL answers with evidence
2. Zero regressions across all 17 Go modules
3. All 4 implementation blocks produce working, tested code
4. Non-goal compliance is verified
5. Evidence gate (OF-5) issues a formal verdict

---

## References

- [S352 — Production Readiness Assessment Gate Report](../stages/stage-s352-production-readiness-assessment-gate-report.md)
- [Production Readiness Assessment Gate](production-readiness-assessment-gate.md)
- [Evidence Matrix and Residual Gaps](production-readiness-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [S347 — Production Readiness Assessment Charter](../stages/stage-s347-production-readiness-assessment-charter-report.md)
