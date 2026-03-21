# Post-Operational-Proof Feature Gate

**Stage:** S281
**Date:** 2026-03-21
**Gate Type:** Strategic Direction Decision
**Scope:** Determine next major wave after S278–S280 operational proof tranche

---

## 1. Gate Context

### What Was Proven (S278–S280)

The operational proof tranche delivered three consecutive stages:

| Stage | Proof | Outcome |
|-------|-------|---------|
| S278 | Operational reconciliation gate | CONDITIONAL PASS — 7 debts closed, 2 CI enforcement gaps surfaced (OD-OH1, OD-OH2) |
| S279 | OS-process compose-level smoke | COMPLETE — 9 containers, 7 scenarios, gateway HTTP API proven |
| S280 | Durable restart and consumer recovery | COMPLETE — 20 scenarios across adapter/writer/compose layers |

### Consolidated Operational State

**CAN do (proven):**
- Paper order generation end-to-end across 6 independent OS processes
- Halt/resume control gate with KV-backed durability
- Durable consumer resume after any single-component restart
- KV projection persistence across all component restarts
- ClickHouse analytical round-trip with Server-Timing headers
- Stateless binary restart (no in-process state beyond NATS/ClickHouse)

**CANNOT do (documented limits):**
- Real venue execution (paper_simulator only)
- Multi-replica deployment (sole-writer by convention)
- Sub-second control plane propagation (poll-based)
- Exactly-once ClickHouse delivery (buffer loss window ≤1000 rows / ≤5s)
- CI-enforced regression detection for 25 NATS KV tests (auto-skip)
- CI-enforced regression detection for 9 ClickHouse integration tests (auto-skip)
- Metrics or distributed tracing (structured logging only)

### Open Debts Carried Forward

| Debt ID | Severity | Description | Status |
|---------|----------|-------------|--------|
| OD-OH1 | **Medium** | 25 NATS KV tests auto-skip in CI | OPEN — blocks feature velocity |
| OD-OH2 | **Medium** | 9 ClickHouse integration tests auto-skip in CI | OPEN — blocks analytical confidence |
| OD-BW2 | Medium | Configuration infrastructure deficit (hardcoded scaling factors) | DEFERRED — tracked since S254 |
| OD-CG1 | Medium | Column-opaque codegen spec (no typed mapper generation) | DEFERRED — known limitation |
| OD-PE2 | Low | S267 formal stage report missing | OPEN — governance debt only |
| OD-OH5 | Low | KV watcher for sub-second gate propagation | DEFERRED by design |
| OD-OH6 | Low | Consumer durability edge cases (long outages >2min) | DEFERRED by design |

---

## 2. Architectural Readiness Assessment

### Bounded Context Maturity

| Context | Domain | Application | Adapter | Actor | Codegen | Assessment |
|---------|--------|-------------|---------|-------|---------|------------|
| Evidence | Complete | Complete | NATS + CH | Ingest | 1 family | **Mature** |
| Signal | Complete | 3 samplers | NATS + CH | Derive | 3 families | **Mature** |
| Decision | Complete | 2 evaluators | NATS + CH | Derive | 2 families | **Mature** |
| Strategy | Complete | 2 resolvers | NATS + CH | Derive | 2 families | **Mature** |
| Risk | Complete | 2 evaluators + scaling | NATS + CH | Derive | 2 families | **Mature** |
| Execution | Complete | Paper evaluator + safety | NATS + CH | Execute | 1 family | **Mature** |
| ConfigCtl | Complete | Compile/activate/project | NATS | ConfigCtl | — | **Mature** |
| Observation | Complete | Ingest bindings | NATS | Ingest | — | **Mature** |

All 8 bounded contexts are domain-complete and adapter-wired. The system is architecturally ready for feature expansion.

### Binary Topology

8 binaries (ingest, derive, execute, store, gateway, writer, configctl, migrate) proven to operate as independent OS processes with durable state coordination through NATS JetStream and ClickHouse.

### Codegen Surface

11 families integrated (S260–S262). Bollinger is the most recent addition (S262). Codegen produces 2 artifacts per family (consumer_spec + pipeline_entry). Golden snapshot equivalence tests protect against drift.

### Test Surface

- 129 test files across all layers
- 52+ behavioral/integration/E2E tests
- 6 CI pipeline stages (unit, codegen, behavioral, integration, smoke, error scan)
- 10 operational scripts (smoke, seed, diagnostic, equivalence)
- **Gap:** 25 NATS KV + 9 ClickHouse tests auto-skip in CI

### Observability Surface

- Structured JSON logging via `slog` in all binaries
- `/readyz` and `/healthz` endpoints in gateway
- Server-Timing headers in analytical responses
- **Gap:** No Prometheus metrics, no OTEL tracing, no dashboards

---

## 3. Strategic Direction Decision

### Prior Gate Guidance

| Gate | Recommendation | Current Relevance |
|------|---------------|-------------------|
| S254 | Hardening → Codegen | ✅ Delivered (S258–S263) |
| S257 | Return to codegen | ✅ Delivered (S258–S263) |
| S263 | **Feature evolution, not more infrastructure** | ⚠️ Three more infrastructure waves followed (paper execution, operational proof, reconciliation) |
| S269 | Bounded operational hardening | ✅ Delivered (S270–S280) |
| S274 | Orchestration proof (S275–S276) | ✅ Delivered (S275–S277) |

**Critical observation:** S263 recommended pivoting to feature delivery. Instead, three additional infrastructure waves were necessary (S264–S280). This was justified — paper execution, safety gates, and operational proof were essential. But the S263 directive remains valid: **the Foundry must now deliver domain value to justify infrastructure investment.**

### Evaluation Summary

Full comparative analysis in [next-wave-strategic-options-matrix.md](next-wave-strategic-options-matrix.md).

| Candidate | Total Score | Verdict |
|-----------|-------------|---------|
| A: Composite Execution Observability | 23/30 | Strong operationally but fourth infrastructure wave |
| B: Multi-Symbol Disciplined | 23/30 | Validates scale but no new capability |
| **C: Signal Evolution** | **25/30** | **Highest value; validates entire infrastructure stack** |
| D: Venue-Readiness Charter | 16/30 | Premature; prerequisites not met |
| E: Selective Codegen Expansion | 21/30 | Side-effect, not primary objective (per S263) |
| F: CI + Observability Foundation | 23/30 | Prerequisite tranche, not a wave |

---

## 4. Gate Verdict

### PASS — Conditional on CI Enforcement Closure

The Foundry is architecturally, operationally, and strategically ready to transition from infrastructure hardening to feature delivery. The transition requires a mandatory prerequisite micro-tranche (2 stages maximum) to close CI enforcement gaps before the feature wave begins.

---

## 5. Strategic Direction: Primary

### Signal Evolution Wave

**Charter:** Deliver new signal and decision families using codegen-first approach, with behavioral tests per family, validating the full infrastructure stack end-to-end.

**Rationale:**
1. **Highest domain value** — first real feature delivery since project inception
2. **Validates infrastructure ROI** — codegen, behavioral model, NATS, ClickHouse, actors all exercised by new families
3. **Codegen-first velocity proof** — each new family proves codegen reduces delivery cost
4. **Bounded and charterable** — each family is an independent increment with clear acceptance criteria
5. **Behavioral model stress test** — new signal/decision interactions reveal behavioral composition quality

**Candidate families (ordered by readiness):**
1. **MACD** — well-defined signal with natural decision evaluator (crossover)
2. **VWAP** — volume-weighted anchor; requires evidence-layer volume data (already exists)
3. **ATR** — volatility signal; natural risk evaluator input for dynamic stop-distance
4. **Bollinger Squeeze** — decision evaluator using existing Bollinger signal (already integrated)

**Minimum viable wave scope:** 2 new signal families + 1 new decision family, each with:
- Codegen family YAML
- Generated consumer_spec and pipeline_entry
- Application-layer sampler/evaluator
- Behavioral tests (severity scaling, cross-domain composition)
- ClickHouse reader integration
- Golden snapshot equivalence

**Entry condition:** OD-OH1 and OD-OH2 closed (CI enforcement for NATS KV and ClickHouse tests).

---

## 6. Strategic Direction: Secondary

### Composite Execution Observability (Interleaved)

**Charter:** Establish minimal metrics foundation during the signal evolution wave, scoped to pipeline-level counters and health indicators.

**Rationale:**
1. New signal families benefit from observable pipeline behavior
2. Metrics infrastructure is additive and low-risk
3. Avoids dedicating a full wave to observability alone
4. Enables operational monitoring before venue readiness

**Scope (strictly bounded):**
- Prometheus metrics endpoint in each binary (`/metrics`)
- Pipeline-level counters: `events_processed_total`, `events_published_total`, `pipeline_errors_total`
- Writer-level gauges: `buffer_depth`, `flush_duration_seconds`, `rows_inserted_total`
- Control gate gauge: `execution_gate_state` (halt=0, active=1)

**NOT in scope:** Distributed tracing, dashboards, alerting, Grafana configuration.

**Delivery:** Interleaved with signal evolution stages; no dedicated observability stages.

---

## 7. What Explicitly NOT to Open Now

### 7.1 Venue-Readiness Charter
- **Why not:** Paper execution proven only at S280 level. No observability to diagnose venue adapter failures. Fill reconciliation alone is a full wave. External Binance dependency introduces non-deterministic test failures.
- **When:** After signal evolution wave proves feature delivery velocity and observability foundation exists.

### 7.2 Full Observability Platform
- **Why not:** Distributed tracing, dashboards, and alerting are premature without feature load to observe. Metrics foundation (secondary direction) is sufficient.
- **When:** After signal evolution wave reveals which observability dimensions matter most.

### 7.3 Codegen Expansion as Primary Objective
- **Why not:** S263 explicitly directed codegen expansion to be side-effect of feature work. Store consumers, starters, and mappers should emerge naturally during signal evolution.
- **When:** As side-effects during signal evolution family delivery.

### 7.4 Multi-Symbol Scale Wave
- **Why not:** Partial proof exists (smoke-multi-symbol.sh). JetStream durable consumer semantics already guarantee isolation. Full multi-symbol validation can be a gate criterion within the signal evolution wave.
- **When:** As acceptance criterion within signal evolution (new families tested with 2+ symbols).

### 7.5 Configuration Infrastructure (OD-BW2)
- **Why not:** Hardcoded scaling factors have not blocked any wave. Configuration infrastructure requires its own design charter.
- **When:** When signal evolution or venue readiness creates operational pressure for runtime configuration changes.

### 7.6 Parallel Feature Fronts
- **Why not:** Every gate since S254 has reinforced single-front discipline. Opening signal evolution AND venue readiness simultaneously would fragment focus and risk regression on both fronts.
- **When:** Never simultaneously. Sequential waves with gates between them.

---

## 8. Prerequisite Micro-Tranche (Before Wave Start)

### Stage S282: CI Infrastructure Enforcement

**Scope:**
- Add NATS JetStream service to `integration-tests` CI job
- Remove auto-skip conditions from 25 NATS KV tests (S271, S273, S275, S276)
- Add ClickHouse service to CI or promote 9 S277 tests to smoke-analytical job
- Validate all 34 previously-skipping tests pass in CI

**Exit criteria:** Zero medium-severity CI enforcement debts (OD-OH1 = CLOSED, OD-OH2 = CLOSED).

**Budget:** 1 stage. Must complete before signal evolution wave opens.

### Stage S283: Signal Evolution Charter and Scope Freeze

**Scope:**
- Define family selection (2-3 signal + 1 decision minimum)
- Freeze behavioral test requirements per family
- Define codegen-first delivery pattern per family
- Set explicit stop conditions and scope boundaries

**Exit criteria:** Charter document with frozen scope, entry/exit criteria, and permitted/prohibited changes.

---

## 9. Success Criteria for Next Gate (Post-Signal-Evolution)

The signal evolution wave will be considered successful if:

1. ≥2 new signal families delivered end-to-end (codegen → NATS → ClickHouse → gateway)
2. ≥1 new decision family consuming new signals
3. All new families have behavioral tests in CI (not auto-skipping)
4. Codegen golden snapshots cover all new families
5. Multi-symbol smoke passes with new families included
6. Pipeline metrics observable via `/metrics` endpoint
7. Zero regressions in existing 52+ behavioral/integration tests
8. Wave completed without opening secondary fronts (venue, scale, configuration)
