# S353 — Operational Foundation Wave Charter Report

> Opens the Operational Foundation Wave with frozen scope, governing questions,
> ordered block plan, exit criteria, and binding constraints.
>
> Predecessor: S352 (Production Readiness Assessment Gate — SUBSTANTIAL delivery).
> Wave type: Implementation (bounded).
> Date: 2026-03-22.

---

## 1. Executive Summary

Stage S353 formally opens the Operational Foundation Wave, the direct successor
to the Production Readiness Assessment Wave (S347–S352).

The assessment wave closed with a SUBSTANTIAL verdict: 15/15 governing questions
answered, 25 tests passing, zero regressions, 10/10 non-goals respected. Its
honest conclusion: domain correctness is proven; operational infrastructure is the
concrete, bounded bottleneck blocking unattended operation.

The Operational Foundation Wave translates that finding into exactly four
implementation blocks totaling ~260 LOC:

| Block | Scope | Effort |
|-------|-------|--------|
| OF-1 | Prometheus `/metrics` endpoint | ~100 LOC |
| OF-2 | CI smoke integration | ~90 LOC |
| OF-3 | Consumer lag + latency histograms | ~50 LOC |
| OF-4 | Startup credential validation | ~20 LOC |
| OF-5 | Evidence gate | Assessment |

The wave scope is frozen. 15 explicit non-goals prevent inflation into
observability platform, CI/CD platform, or production program. 10 guard rails
enforce discipline throughout execution.

---

## 2. What Was Delivered

### Charter Document

**File**: [`../architecture/operational-foundation-wave-charter-and-scope-freeze.md`](../architecture/operational-foundation-wave-charter-and-scope-freeze.md)

Contents:
- Wave identity and strategic context
- 8 governing questions (OFQ-1 through OFQ-8)
- 5 executable blocks with scope definitions
- Sequencing with dependency chain
- 5 freeze conditions
- Dependencies and preconditions table
- Success criteria for wave closure

### Exit Criteria and Non-Goals Document

**File**: [`../architecture/operational-foundation-items-exit-criteria-and-non-goals.md`](../architecture/operational-foundation-items-exit-criteria-and-non-goals.md)

Contents:
- Objective exit criteria per block (5 per block, 25 total)
- 15 explicit non-goals with rationale
- 10 guard rails with enforcement notes
- Residual gap coverage map (S352 gaps → OF blocks)
- Change/no-change impact summary

---

## 3. Post-S352 State Analysis

### What the Foundry Proved (Assessment Wave Evidence)

| Capability | Verdict | Key Evidence |
|-----------|---------|-------------|
| Real venue connectivity | SUBSTANTIAL | DNS, TLS, auth, error classification against live testnet |
| Sustained operation | SUBSTANTIAL | 5 min, ~96 events, zero drift, zero errors |
| Operational observability | PARTIAL | 7 signal categories, sufficient for attended operation only |
| Deployment repeatability | SUBSTANTIAL | 3-command path, 9 smoke targets, secure defaults |

### Where the Bottleneck Is

The assessment wave identified 15 residual gaps. Analysis shows:

- **2 HIGH gaps** block unattended operation (metric export, push alerting)
- **7 MEDIUM gaps** block operational maturity (CI, lag, latency, credentials, timeouts)
- **6 LOW gaps** are ergonomic or explicitly deferred

The Operational Foundation Wave targets **6 of these 15 gaps** — the ones that
are application-level, bounded, and high-ROI. The remaining 9 are infrastructure
decisions, deployment concerns, or deferred by design.

---

## 4. Wave Architecture

### Governing Questions

| # | Question | Block |
|---|----------|-------|
| OFQ-1 | Does the gateway expose `/metrics` in Prometheus format? | OF-1 |
| OFQ-2 | Are health-tracker counters exported as Prometheus metrics? | OF-1 |
| OFQ-3 | Do smoke scripts produce machine-readable output? | OF-2 |
| OFQ-4 | Does `make smoke-ci` run all targets and collect artifacts? | OF-2 |
| OFQ-5 | Does `/statusz` expose NATS consumer lag? | OF-3 |
| OFQ-6 | Are order-submission latencies exposed as histograms? | OF-3 |
| OFQ-7 | Does the adapter validate credentials at startup? | OF-4 |
| OFQ-8 | Does validation fail-fast with clear error on bad credentials? | OF-4 |

### Block Dependencies

```
OF-1 ──→ OF-3 (histogram needs /metrics)
OF-2 ──→ (independent)
OF-4 ──→ (independent)
OF-1, OF-2, OF-3, OF-4 ──→ OF-5 (gate)
```

### Ordered Stage Plan

| Stage | Block | Description | Prerequisite |
|-------|-------|-------------|-------------|
| **S353** | Charter | This document | S352 |
| **S354** | OF-1 | Prometheus /metrics endpoint | S353 |
| **S355** | OF-3 | Consumer lag + latency histograms | S354 |
| **S356** | OF-2 | CI smoke integration | S353 |
| **S357** | OF-4 | Startup credential validation | S353 |
| **S358** | OF-5 | Operational foundation evidence gate | S354–S357 |

S354, S356, and S357 can execute in parallel where capacity permits.
S355 must follow S354 (depends on `/metrics` endpoint).

---

## 5. Exit Criteria Summary

Each block has 5 objective exit criteria (25 total across the wave).
Full criteria are in the companion document.

**Key acceptance tests**:

| Block | Key Test |
|-------|----------|
| OF-1 | `curl localhost:<port>/metrics` returns Prometheus text with all 13 counters |
| OF-2 | `make smoke-ci` runs 9 targets, produces JSON/TAP output, collects artifacts |
| OF-3 | `/statusz` shows consumer lag; `/metrics` shows latency histogram |
| OF-4 | Missing/invalid credentials → structured error + fast exit; valid → normal startup |
| OF-5 | 8/8 governing questions answered; zero regressions; formal verdict |

---

## 6. Non-Goals (Frozen)

15 explicit non-goals prevent scope inflation:

| Category | Non-Goals |
|----------|-----------|
| Domain expansion | Mainnet (NG-1), multi-venue (NG-2), OMS (NG-3), portfolio risk (NG-4), strategy (NG-7), new breadth (NG-6) |
| Infrastructure expansion | Full observability (NG-8), log aggregation (NG-9), Alertmanager (NG-10), K8s/Helm/Terraform (NG-13) |
| Operational expansion | Dashboards (NG-5), hours-scale soak (NG-11), credential rotation (NG-12), CI/CD pipeline (NG-14), chaos engineering (NG-15) |

The wave adds **metric export**, not an observability platform.
The wave makes smoke **CI-ready**, not a CI/CD pipeline.
The wave validates credentials at **startup**, not at runtime rotation.

---

## 7. Guard Rails

| # | Guard Rail |
|---|-----------|
| GR-1 | No mainnet endpoints |
| GR-2 | No new venue adapters |
| GR-3 | No new domain types unless directly required by an OF block |
| GR-4 | No architectural redesign |
| GR-5 | No scope expansion after S353 |
| GR-6 | Dependency ordering is binding |
| GR-7 | `/metrics` only — no OTEL collector, no Jaeger |
| GR-8 | CI smoke output only — no pipeline YAML |
| GR-9 | Credential validation is lightweight — single sync ping |
| GR-10 | No dashboard construction |

---

## 8. Residual Gap Disposition

After this wave closes (assuming full delivery):

| Severity | Before Wave | Closed by Wave | Remaining |
|----------|-------------|----------------|-----------|
| HIGH | 2 | 1 (RG-1) | 1 (RG-2: push alerting — infrastructure) |
| MEDIUM | 7 | 5 (RG-3, RG-5, RG-6, RG-7, RG-9) | 2 (RG-4: hours soak, RG-8: log aggregation) |
| LOW | 6 | 0 | 6 (ergonomic / deferred) |
| **Total** | **15** | **6** | **9** |

---

## 9. Preparation for S354

Before S354 begins:

1. **`prometheus/client_golang`** — add to `go.mod` for the gateway module
2. **Health tracker counter names** — inventory the 13 named counters from S350 evidence
3. **Gateway HTTP router** — identify where to register the `/metrics` handler
4. **Existing test suite green** — all 17 Go modules passing (verified at S352)

---

## 10. Verdict

S353 is **COMPLETE**. The Operational Foundation Wave is formally open with
frozen scope.

The next stage is **S354: Prometheus /metrics Endpoint** (OF-1).

---

## Promoted Documents

| Document | Location |
|----------|----------|
| Wave charter and scope freeze | [`../architecture/operational-foundation-wave-charter-and-scope-freeze.md`](../architecture/operational-foundation-wave-charter-and-scope-freeze.md) |
| Exit criteria and non-goals | [`../architecture/operational-foundation-items-exit-criteria-and-non-goals.md`](../architecture/operational-foundation-items-exit-criteria-and-non-goals.md) |
