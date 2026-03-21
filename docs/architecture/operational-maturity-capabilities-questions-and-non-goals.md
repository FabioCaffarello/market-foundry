# Operational Maturity Wave — Capabilities, Governing Questions, and Non-Goals

> **Wave:** Operational Maturity and Engineering Governance (Phase 31)
> **Charter:** [operational-maturity-and-engineering-governance-wave-charter.md](operational-maturity-and-engineering-governance-wave-charter.md)
> **Date:** 2026-03-21

---

## 1. Target Capabilities

Each capability maps to one or more wave blocks and has a governing question that must be answered with evidence before the wave can close.

### C1 — Full-Surface CI/CD Coverage

**Block:** S314
**Description:** The CI pipeline covers every category of test and operational proof that exists in the repository. No proof runs only locally.

**Current state:**
- CI runs: unit tests, codegen golden, behavioral scenarios, integration tests (NATS), smoke-analytical E2E
- CI does NOT run: smoke-first-slice, smoke-multi-symbol, smoke-operational, smoke-restart-recovery, ClickHouse integration tests

**Target state:**
- All 5 smoke scripts run in CI on push/PR
- ClickHouse integration tests run in CI with service container
- CI failure on any proof blocks merge

### C2 — Explicit Testing Strategy

**Block:** S315
**Description:** The repository has a documented testing strategy that classifies every test category, defines coverage targets, and identifies gaps.

**Current state:**
- 152 test files exist across the codebase
- Tests are build-tagged (integration, unit implicit) but no unified strategy document exists
- No explicit coverage targets or gap analysis

**Target state:**
- Testing strategy document with test pyramid definition
- Classification of every test category (unit, integration, behavioral, codegen, smoke/E2E)
- Gap map identifying untested boundaries
- Coverage targets per category

### C3 — CI-Enforced Quality Gates

**Block:** S316
**Description:** Quality gate profiles (fast/ci/deep) are enforced in CI, not just locally. The CI pipeline runs the appropriate profile and blocks merge on failure.

**Current state:**
- raccoon-cli quality-gate runs locally via `make check`, `make check-deep`
- `make quality-gate-ci` target exists with JSON output profile
- CI does not run quality-gate checks

**Target state:**
- `quality-gate-ci` profile runs in CI pipeline
- Gate failure blocks merge
- Profile coverage includes all current invariants (arch-guard, drift-detect, contract checks)

### C4 — marketmonkey Absorption Readiness

**Block:** S317
**Description:** The repository's structural readiness to absorb the marketmonkey codebase is assessed and documented.

**Current state:**
- market-foundry was originally sanitized from quality-service
- marketmonkey absorption identified as next phase in project memory
- No formal readiness assessment exists

**Target state:**
- Structural compatibility assessment (module boundaries, dependency graph, naming conventions)
- Blocker identification (conflicting patterns, missing abstractions, namespace collisions)
- Prerequisites list for safe absorption
- Recommended absorption strategy (big-bang vs. incremental)

### C5 — Engineering Governance Policies

**Block:** S318
**Description:** The repository has documented policies for contribution, code review, release management, and branch strategy.

**Current state:**
- DEVELOPMENT.md documents daily workflow
- Stage governance documents evolution protocol
- No formal contribution guidelines, review policy, or release strategy

**Target state:**
- Contribution guidelines linked from DEVELOPMENT.md
- Code review policy (what requires review, approval thresholds)
- Release tagging and versioning strategy
- Branch strategy (main-only vs. feature branches, merge vs. rebase)

### C6 — Operational Runbook Maturity

**Block:** S319
**Description:** The operational runbook covers failure diagnosis, alert thresholds, and incident response for pre-production operation.

**Current state:**
- `docs/operations/developer-onboarding-and-troubleshooting-guide.md` exists
- `docs/operations/proof-execution-user-flows-and-failure-diagnosis.md` exists
- `make diag` provides runtime diagnostics
- No formalized alert thresholds or incident response flow

**Target state:**
- Unified operational runbook with failure classification
- Alert threshold definitions for key metrics (latency, error rate, queue depth)
- Incident response flow (detect → diagnose → mitigate → postmortem)
- Integration with existing `make diag` and log scanning

---

## 2. Governing Questions

Each question must be answered with evidence before the post-wave gate (S320) can pass.

| ID | Question | Evidence Source |
|----|----------|----------------|
| GQ1 | Does the CI pipeline cover every category of test and operational proof in the repository? | CI workflow configuration + green pipeline run with all jobs |
| GQ2 | Is the testing strategy documented with explicit classification, coverage targets, and gap map? | Testing strategy document + gap map |
| GQ3 | Are quality gate profiles enforced in CI with merge-blocking semantics? | CI workflow showing quality-gate-ci job + blocked PR evidence |
| GQ4 | Is the repository structurally ready to absorb marketmonkey, with blockers and prerequisites documented? | Absorption readiness assessment document |
| GQ5 | Are engineering governance policies documented and linked from the developer workflow? | Governance documents + DEVELOPMENT.md links |
| GQ6 | Is the operational runbook formalized with failure diagnosis flows and alert thresholds? | Runbook document + integration with existing diagnostics |
| GQ7 | Has every block in the wave been completed without scope expansion beyond the frozen charter? | Stage reports for S314–S319 + charter compliance check |

---

## 3. Non-Goals

These items are explicitly excluded from this wave. Each non-goal includes the rationale for exclusion.

### NG1 — Venue Readiness Implementation

**Rationale:** Venue readiness design is complete (S306–S311) but blocked by EC-1 (client order ID derivation). Implementation belongs to a separate foundational tranche after EC-1 resolution. This wave is governance, not functional implementation.

### NG2 — New Domain Families

**Rationale:** The domain model (signal → decision → strategy → risk → execution) is stable. Adding new families is breadth expansion, which contradicts the governance focus of this wave.

### NG3 — New Signal Families or Strategies

**Rationale:** Signal evolution (Phase 27) and behavioral features (Phase 23) are closed waves. New signals or strategies would be a new breadth wave, not governance.

### NG4 — marketmonkey Absorption Execution

**Rationale:** This wave produces a readiness assessment, not the absorption itself. Executing absorption without the assessment would repeat the anti-pattern of implementation before analysis.

### NG5 — Production Deployment Infrastructure

**Rationale:** Kubernetes, Terraform, cloud provider configuration, and production deployment tooling are out of scope. This wave matures the engineering discipline of the repository, not the deployment target.

### NG6 — Multi-Venue Abstraction

**Rationale:** Single-venue focus (Binance Futures) is the current design. Multi-venue abstraction is deferred until testnet is proven. This is a venue readiness concern, not governance.

### NG7 — Retry Architecture Implementation (RT-1 through RT-7)

**Rationale:** Retry constraints are documented in the venue readiness charter but implementation is deferred pending EC-1. This wave does not implement venue-related retry logic.

### NG8 — Observability Dashboard or Metrics Infrastructure

**Rationale:** Structured logging exists; Grafana/Prometheus/metrics infrastructure is production deployment tooling (see NG5). This wave may define alert thresholds conceptually but does not deploy monitoring infrastructure.

### NG9 — Architectural Redesign

**Rationale:** The monorepo architecture is stable after 30 phases of evolution. This wave hardens governance around the existing architecture; it does not redesign module boundaries, composition roots, or domain contracts.

### NG10 — Performance Optimization

**Rationale:** Performance work requires production load profiles that don't exist yet. Premature optimization contradicts the discipline focus of this wave.

---

## 4. Capability–Question–Block Traceability Matrix

| Capability | Governing Question | Block |
|------------|-------------------|-------|
| C1 — Full-Surface CI/CD Coverage | GQ1 | S314 |
| C2 — Explicit Testing Strategy | GQ2 | S315 |
| C3 — CI-Enforced Quality Gates | GQ3 | S316 |
| C4 — marketmonkey Absorption Readiness | GQ4 | S317 |
| C5 — Engineering Governance Policies | GQ5 | S318 |
| C6 — Operational Runbook Maturity | GQ6 | S319 |
| — (wave integrity) | GQ7 | S320 |

---

## 5. Block Ordering Rationale

The blocks are ordered to maximize foundation-before-extension:

1. **S314 (CI/CD)** first because all subsequent blocks benefit from pipeline coverage — testing strategy without CI is theory, quality gates without CI are local-only.
2. **S315 (Testing Strategy)** second because it produces the gap map that informs quality gate evolution and documents what CI should enforce.
3. **S316 (Quality Gates)** third because it integrates the CI pipeline (S314) with the testing strategy (S315) into enforceable invariants.
4. **S317 (marketmonkey Readiness)** fourth because it requires the governance infrastructure (CI, testing, gates) to be in place before assessing absorption readiness.
5. **S318 (Engineering Governance)** fifth because policies codify the practices established in S314–S317.
6. **S319 (Operational Runbook)** sixth because it builds on all prior blocks to formalize operational maturity.
7. **S320 (Gate)** last — evidence gate closing the wave.
