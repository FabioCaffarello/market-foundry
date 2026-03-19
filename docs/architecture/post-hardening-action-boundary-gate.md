# Post-Hardening Action Boundary Gate

> **Stage:** S89 — Formal Readiness Review
> **Gate Type:** Action Boundary (GO / NO-GO for real venue frontier)
> **Date:** 2026-03-19
> **Predecessor Gates:** S74, S86
> **Prerequisite Stages:** S87 (operational hardening), S88 (design hardening)

---

## 1. Executive Summary

This document is the formal action boundary gate that determines whether Market Foundry may exit paper-integrated execution and advance toward a first guarded real-venue adapter.

**Verdict: CONDITIONAL GO — with mandatory pre-activation implementation work.**

The platform has achieved operational and design maturity sufficient to justify opening the real-venue frontier, but **three implementation prerequisites remain between design-complete and activation-ready**. These are scoped, bounded, and non-speculative — they convert S88 designs into enforceable runtime behavior.

---

## 2. Gate Evaluation Framework

The gate evaluates eight dimensions. Each receives a status:

| Status | Meaning |
|--------|---------|
| **PASS** | Meets or exceeds threshold for real-venue frontier |
| **PASS-WITH-CAVEAT** | Meets threshold but has a documented limitation |
| **BLOCKED** | Cannot proceed until resolved |

---

## 3. Dimension Assessments

### D1: Execute Runtime Maturity

**Status: PASS**

Evidence:
- Execute binary is a first-class platform citizen (Docker Compose, Makefile, healthz, configs).
- Venue adapter actor implements three-gate pipeline: kill switch → staleness guard → venue submit.
- Paper venue adapter has 100% test coverage (5 unit tests per component, 7 integration tests).
- Domain model (`ExecutionIntent`) has 20+ unit tests, comprehensive validation, and state machine discipline.
- Graceful shutdown with final stats summary confirmed.
- Fail-open semantics on control gate unavailability (documented, tested).

Caveats:
- Staleness `maxAge` (120s) is hardcoded in registry, not config-driven.
- No actor-level unit tests for supervisor/venue_adapter (covered by integration tests).

### D2: Pipeline Mesh Robustness (derive → execute → store → gateway)

**Status: PASS**

Evidence:
- Two-family architecture cleanly separates paper (derive → store) from venue (execute → store).
- Stream ownership documented: `EXECUTION_EVENTS` (derive publishes), `EXECUTION_FILL_EVENTS` (execute publishes).
- KV bucket ownership documented: `EXECUTION_PAPER_ORDER_LATEST`, `EXECUTION_VENUE_MARKET_ORDER_LATEST`, `EXECUTION_CONTROL`.
- Consumer reliability: explicit ACK, durable consumers, MaxDeliver=5, problem-based error classification (TERM vs NAK).
- Projection pipeline: 3-gate validation (final-only, domain validation, monotonicity guard).
- 29 projection tests (18 execution + 11 fill) with stats invariant validation.
- Composite status query merges intent + result + control gate with propagation rule.
- Smoke test validates full chain across 2 symbols × 2 timeframes.

Caveats:
- Consumer acks before projection completes — KV failure post-ack causes data loss (documented in failure recovery model; mitigated by error counters).
- Transitional bridge (execute consumes paper_order subjects) persists until venue migration.

### D3: Observability and Integrated Validation

**Status: PASS-WITH-CAVEAT**

Evidence:
- Three HTTP health endpoints: `/healthz`, `/readyz`, `/statusz`.
- Custom counters per tracker: `processed`, `filled`, `skipped_stale`, `skipped_halt`.
- Idle heartbeat monitoring (2-minute threshold, 30s check interval).
- Structured logging with slog: source, symbol, timeframe, correlation_id, venue_order_id, side, quantity.
- Raccoon-CLI guardian validates: binary registration, stream/consumer/bucket naming, config drift, file existence.
- Docker Compose health check on `/readyz` with 60s total timeout.

Caveat:
- No Prometheus `/metrics` endpoint (structured logging + `/statusz` sufficient for paper; real venue needs time-series metrics for latency/error-rate monitoring).
- No distributed tracing integration.
- No external alerting rules.

### D4: Fill Reconciliation and Async Fill Model

**Status: PASS-WITH-CAVEAT**

Evidence:
- S88 designed 7 reconciliation invariants (RC-1 through RC-7): fill-to-intent correlation, quantity boundaries, status transitions, orphan handling, stuck detection, dedup, timestamp ordering.
- S88 designed async fill model: two-phase execution, 4 new event types, FillTrackerActor polling design.
- Fill projection implements monotonicity guard and dedup key (`fill:{venue_order_id}:{timestamp}`).
- Paper mode trivially satisfies reconciliation (always full fills, always succeeds).

Caveat:
- **RC-1 through RC-7 are design-only — no runtime enforcement exists.** Paper mode masks this because reconciliation always trivially succeeds. Real venue requires explicit enforcement before activation.
- FillTrackerActor is design, not implemented (polling-based fill status monitoring).
- No background reconciliation actor for stuck intent detection.

### D5: Paper vs Real Venue Separation

**Status: PASS**

Evidence:
- Subject hierarchy separates families: `paper_order` (derive-owned) vs `venue_market_order` (execute-owned).
- Config-driven venue adapter selection with `knownVenueTypes` registry.
- Settings schema rejects unknown venue types at startup.
- Drift detection validates execution_families across configs.
- `VenuePort` interface enables adapter substitution without pipeline changes.
- Transitional bridge explicitly marked with comments and documented in S85.

Caveats:
- Bridge persists (conflates families operationally at execute consumer level).
- No venue-specific intake subject yet (`venue_market_order.submitted.>` designed but not implemented).

### D6: Governance / Config Symmetry / CLI / Tooling

**Status: PASS**

Evidence:
- Five governance layers documented: config-driven venue selection, family gating, kill switch, staleness guard, domain validation.
- Config symmetry verified: execute.jsonc, derive.jsonc, store.jsonc have appropriate (a)symmetry.
- Settings schema validates execution families, venue types, and cross-layer dependencies.
- Raccoon-CLI covers: binary registration, 60+ execution docs, subject/durable/bucket registries, config drift detection.
- Makefile: all build/test/run/quality targets include execute.
- 17-gate activation ceremony designed (AG-1 through AG-17).

### D7: Secrets / Credentials / Activation Prerequisites

**Status: BLOCKED (design-only)**

Evidence:
- S88 designed credential delivery mechanism: environment variables with `MF_VENUE_{TYPE}_{NAME}` prefix.
- `CredentialSet` struct and `LoadCredentials()` function designed (fail-fast on missing credentials).
- Docker Compose `env_file` template designed.
- Three-phase post-activation monitoring designed: shadow (24h), guarded (72h), operational.

Blocker:
- **No credential infrastructure exists.** LoadCredentials(), env_file template, and credential validation are design-only. Implementation required before any real venue adapter can be constructed.

### D8: Residual Risk Before Real Adapter

**Status: PASS-WITH-CAVEAT**

See `post-hardening-risks-and-blockers.md` for full enumeration. Key residual risks:
- Consumer-projection decoupling (KV failure post-ack).
- No embedded NATS integration tests (unit tests mock interactions).
- No CI pipeline automation (GitHub Actions workflow not implemented).
- Single-adapter-per-binary limitation.

---

## 4. Gate Verdict

### Scoring Summary

| Dimension | Status |
|-----------|--------|
| D1: Execute Runtime Maturity | PASS |
| D2: Pipeline Mesh Robustness | PASS |
| D3: Observability | PASS-WITH-CAVEAT |
| D4: Fill Reconciliation | PASS-WITH-CAVEAT |
| D5: Paper/Real Separation | PASS |
| D6: Governance/CLI/Tooling | PASS |
| D7: Credentials/Activation | BLOCKED |
| D8: Residual Risk | PASS-WITH-CAVEAT |

### Verdict: CONDITIONAL GO

The platform is architecturally and operationally ready to begin the transition from paper-only to guarded real-venue execution. However, **three implementation prerequisites must be completed before activation gate ceremony (AG-1..AG-17) can be run:**

1. **Credential infrastructure implementation** (converts S88 design to runtime).
2. **Reconciliation invariant enforcement** (converts S88 RC-1..RC-7 from design to runtime guards).
3. **Embedded NATS integration tests** (validates full consumer → gate → venue → publisher flow with real NATS).

These are bounded, scoped tasks — not speculative design work. They convert existing, reviewed designs into enforceable code.

### Decision Matrix for S90

| If... | Then S90 should be... |
|-------|-----------------------|
| All 3 prerequisites completed | First guarded real-venue step (activation gate ceremony) |
| 2 of 3 completed | Implementation completion + partial ceremony dry-run |
| 1 or fewer completed | Focused implementation sprint, no venue activation |

---

## 5. Gate Authority

This gate supersedes the S86 action boundary verdict. The next gate will be the activation gate ceremony itself (AG-1..AG-17), which can only be initiated after the three implementation prerequisites above are satisfied.
