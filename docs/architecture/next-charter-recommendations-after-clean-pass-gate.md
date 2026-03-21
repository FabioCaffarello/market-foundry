# Next-Charter Recommendations After Clean-Pass Gate

**Gate:** S232
**Date:** 2026-03-20
**Status:** Clean-pass confirmed; next charter authorized

---

## 1. Context

The S229–S231 tranche closed the final mechanical blockers. The repository is in clean-pass state with:

- All CI gates green (local and remote)
- Quality-gate tooling reconciled with architecture
- Active documentation aligned
- Release tag `v0.1.0-s231` on verified-green commit

The question is: what should the next charter prioritize?

---

## 2. Candidate Charter Directions

Based on the current codebase state (399 Go files, 8 services, 7 domain pipelines, analytical layer operational), the following directions are viable:

### Direction A: Feature Evolution — Deepen Domain Logic

**What:** Expand the strategy, risk, and execution domains with real trading logic beyond the current single-evaluator implementations.

**Current state:**
- Signal: EMA crossover + RSI samplers (functional)
- Decision: RSI oversold evaluator (minimal)
- Strategy: Mean reversion entry resolver (single strategy)
- Risk: Position exposure evaluator (basic)
- Execution: Paper venue adapter only

**Value:** Moves the system from "demonstrates the pipeline" to "demonstrates useful trading logic."

**Risk:** Feature work without strengthening the CI pipeline could introduce regressions.

### Direction B: Platform Hardening — CI and Test Coverage

**What:** Expand the CI pipeline to include integration tests, strengthen test coverage in under-tested areas, add security scanning.

**Current state:**
- CI runs unit tests + codegen golden + smoke analytical
- Integration tests (`make test-integration`) run locally only
- 108 test files (27% of codebase) — reasonable but uneven distribution

**Value:** Higher confidence in future changes. Catches NATS-level regressions in CI.

**Risk:** Pure hardening doesn't advance feature capabilities. Could feel like "more mechanical work."

### Direction C: Operational Readiness — Observability and Deployment

**What:** Add structured logging, metrics collection, distributed tracing, and production deployment configurations.

**Current state:**
- Health endpoints exist (`/healthz`, `/readyz`)
- No structured metrics export
- No distributed tracing
- Docker compose for local development only

**Value:** Prepares the system for real deployment scenarios.

**Risk:** Premature if the feature set isn't yet worth deploying.

### Direction D: marketmonkey Absorption

**What:** Begin absorbing marketmonkey functionality into market-foundry, as referenced in the project sanitization memory.

**Value:** Consolidates the ecosystem. Reduces maintenance surface.

**Risk:** Large scope. Requires careful scoping to avoid regression.

---

## 3. Recommended Approach

**Primary recommendation: Direction A (Feature Evolution) with a lightweight Direction B component.**

Rationale:

1. The mechanical foundation is solid. The system needs to demonstrate value, not more infrastructure work.
2. Adding `make test-integration` to CI (from Direction B) is a small, high-value addition that protects feature work.
3. Deepening domain logic is the natural next step after proving the pipeline architecture works end-to-end.
4. marketmonkey absorption (Direction D) is better deferred until the feature foundation is stronger.

### Suggested Charter Scope

**Charter title:** "Domain Logic Depth — Strategy and Risk Expansion"

**Objectives:**
1. Add at least one additional strategy resolver (e.g., momentum-based).
2. Expand risk evaluation beyond single-position exposure.
3. Add decision logic that considers multiple signals.
4. Integrate `make test-integration` into the CI pipeline.
5. Prove the expanded logic through the existing analytical pipeline.

**Acceptance criteria:**
- New strategy/risk logic passes unit tests.
- Codegen golden snapshots updated if new families are added.
- CI pipeline includes integration tests.
- Analytical smoke test validates the expanded pipeline.
- `make quality-gate-ci` remains at 0 errors.

---

## 4. What to Avoid in the Next Charter

1. **Do not reopen mechanical corrections.** The S229–S231 tranche is closed. If new drift surfaces, address it as part of the charter's own cleanup, not as a reopening of prior work.

2. **Do not expand documentation without pruning.** If the charter adds new architecture docs, consider archiving superseded ones to prevent further entropy.

3. **Do not break the quality-gate convergence.** Any new raccoon-cli rules must pass both fast and CI profiles from the start.

4. **Do not skip the gate.** The next charter should end with its own gate evaluation, maintaining the discipline established in S228–S232.

---

## 5. Pre-Charter Checklist

Before the next charter begins implementation:

- [ ] Charter document written with scope, objectives, and acceptance criteria
- [ ] Starting baseline verified (`make verify` passes)
- [ ] `make quality-gate-ci` confirms 0 errors
- [ ] Branch strategy decided (feature branch vs. main)
- [ ] Stage numbering established (S233+)
