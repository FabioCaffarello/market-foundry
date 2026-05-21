# Operational Maturity and Engineering Governance Wave — Charter

> **Stage:** S313
> **Phase:** 31
> **Status:** OPEN — Scope Frozen
> **Date:** 2026-03-21
> **Predecessor:** Phase 30 Venue Readiness Wave (S306–S311), Post-Charter Gate (S311)

---

## 1. Strategic Context

The market-foundry monorepo has completed 30 phases of evolutionary development spanning foundation, domain design (signal → decision → strategy → risk → execution), analytical infrastructure, codegen, behavioral features, multi-symbol operational scaling, and venue readiness charter.

The structural base consolidated in Phases 9–10 (S95–S118) provided the technical platform. Subsequent phases added domain depth, behavioral correctness, composite observability, and production gap mapping through S311.

**The repository now has:**
- 8 service binaries across 17 Go modules
- 1 Rust architecture guardian (raccoon-cli)
- 152 test files spanning unit, integration, behavioral, and codegen tests
- 5 CI jobs (unit, codegen-golden, behavioral-scenarios, integration, smoke-analytical)
- 6 operational smoke scripts
- Multi-profile quality gates (fast/ci/deep)
- Layer boundary enforcement and drift detection

**What the repository does NOT yet have at production-grade maturity:**
- CI/CD pipeline covering the full operational proof surface
- Formalized testing strategy with documented coverage targets and gap map
- marketmonkey absorption readiness assessment
- Quality gate evolution path from current raccoon-cli profiles to full CI enforcement
- Engineering governance policies for contribution, review, and release

---

## 2. Wave Identity

| Attribute | Value |
|-----------|-------|
| **Wave Name** | Operational Maturity and Engineering Governance |
| **Wave Type** | Governance and discipline — NOT functional breadth |
| **Phase Number** | 31 |
| **Stage Range** | S313–S320 (estimated, subject to gate evidence) |
| **Entry Condition** | S311 post-charter gate passed; venue readiness design complete |
| **Exit Condition** | All governing questions answered with evidence; formal gate passed |

---

## 3. Wave Purpose

This wave exists to raise the engineering discipline of the monorepo to a level where:

1. Every code change is automatically validated by a CI pipeline that covers the full proof surface.
2. Testing strategy is explicit, documented, and gap-free for the current capability set.
3. The repository is structurally ready to absorb the marketmonkey codebase without architectural regression.
4. Quality gates evolve from local-only enforcement to CI-enforced invariants.
5. Engineering governance policies are documented and enforceable.

This wave does **not** add new functional capabilities, new domains, new families, or venue implementation code.

---

## 4. Wave Blocks (Ordered)

| Block | ID | Name | Purpose |
|-------|----|------|---------|
| 1 | S313 | Charter and Scope Freeze | This document — opens the wave, freezes scope |
| 2 | S314 | CI/CD Pipeline Maturity | Expand CI to cover all smoke proofs, ClickHouse integration, multi-symbol scenarios |
| 3 | S315 | Testing Strategy and Coverage Map | Document explicit testing strategy, classify tests, identify gaps, set coverage targets |
| 4 | S316 | Quality Gate Evolution | Evolve raccoon-cli quality gates from local-only to CI-enforced; add missing profiles |
| 5 | S317 | marketmonkey Absorption Readiness Assessment | Audit structural readiness for absorbing marketmonkey; identify blockers and prerequisites |
| 6 | S318 | Engineering Governance Policies | Contribution guidelines, review policy, release tagging, branch strategy |
| 7 | S319 | Operational Runbook and Incident Readiness | Formalize runbook, alert thresholds, failure diagnosis flow for pre-production operation |
| 8 | S320 | Post-Wave Gate and Strategic Direction | Evidence gate closing the wave; authorize next charter |

---

## 5. Scope Boundary

### In Scope

- CI/CD pipeline expansion to cover the full operational proof surface
- Testing strategy documentation and gap analysis
- Quality gate profile evolution and CI enforcement
- marketmonkey absorption structural readiness assessment (analysis only — NOT implementation)
- Engineering governance policy documentation
- Operational runbook formalization
- Wave-level evidence gate

### Out of Scope (Frozen)

- Venue readiness implementation (deferred to post-EC-1 foundational tranche)
- New domain families or signal families
- New functional capabilities or behavioral features
- marketmonkey absorption execution (assessment only in this wave)
- Production deployment infrastructure (Kubernetes, Terraform, etc.)
- Multi-venue abstraction
- Real-venue integration testing
- Retry architecture implementation (RT-1 through RT-7)
- Error classification hardening (VA-1)

---

## 6. Dependencies

| Dependency | Status | Impact |
|------------|--------|--------|
| S311 post-charter gate | Passed | Wave can open |
| raccoon-cli quality gate infrastructure | Available | Block 4 builds on existing profiles |
| GitHub Actions CI workflow | Available | Block 2 extends existing pipeline |
| marketmonkey repository access | Required for Block 5 | Assessment requires read access to marketmonkey codebase |
| Docker Compose stack stability | Available | CI smoke expansion requires stable compose-up path |

---

## 7. Success Criteria

The wave is complete when:

1. CI pipeline runs all operational proofs (smoke-first-slice, smoke-multi, smoke-analytical, smoke-operational, smoke-restart-recovery) on every push/PR.
2. Testing strategy document exists with explicit classification, gap map, and coverage targets.
3. Quality gate profiles are enforced in CI with zero bypass paths.
4. marketmonkey absorption readiness assessment is documented with clear blockers and prerequisites.
5. Engineering governance policies are documented and linked from DEVELOPMENT.md.
6. Operational runbook is formalized with failure diagnosis flows.
7. Post-wave gate passes with evidence for all governing questions.

---

## 8. Guard Rails

- **No new families.** This wave does not introduce new domain, signal, or analytical families.
- **No venue readiness implementation.** Venue readiness remains in design-complete state pending EC-1 resolution.
- **No architectural redesign.** Existing architecture is stable; this wave hardens governance, not structure.
- **No scope expansion.** Block list is frozen. New items go to the next wave's charter.
- **Assessment ≠ execution.** The marketmonkey absorption block produces an assessment document, not code changes.
