# Stage S59 — Risk Readiness Review Report

> Formal readiness assessment for the `risk` domain layer.
> Date: 2026-03-18

| Field | Value |
|-------|-------|
| Stage | S59 |
| Title | Risk Readiness Review |
| Type | Review |
| Objective | Evaluate whether Market Foundry is ready to open a `risk` layer |
| Verdict | **CONDITIONALLY READY** — design may proceed; implementation requires hardening |

---

## Executive Summary

Market Foundry has reached a level of architectural maturity sufficient to begin designing a `risk` domain. All five existing layers (observation, evidence, signal, decision, strategy) are implemented, governed, and follow consistent patterns. The config dependency chain is validated at startup. The projection authority model is sound. The mesh is clean.

However, two systemic test gaps — adapter tests (0 publisher/consumer tests across 5 domains) and derive actor tests (0 test files across 12 actors) — represent compounding debt. Adding a 6th domain without addressing these gaps is possible but not recommended.

**Recommendation**: Begin risk domain design (S62) immediately. Execute adapter and actor test hardening (S60, S61) in parallel. Gate risk implementation (S64) on completion of hardening stages.

---

## Readiness Assessment Summary

| Dimension | Rating | Verdict |
|-----------|--------|---------|
| Observation maturity | 7.5/10 | Sufficient |
| Evidence maturity | 8.0/10 | Strong |
| Signal maturity | 8.0/10 | Strong |
| Decision maturity | 8.5/10 | Strong |
| Strategy maturity | 8.5/10 | Strong |
| Projection authority | 9.0/10 | Excellent |
| Query surfaces | 9.0/10 | Excellent |
| Governance/CLI | 8.5/10 | Strong |
| Activation model | 9.0/10 | Excellent |
| Config dependencies | 9.0/10 | Excellent |
| **Overall** | **8.38/10** | **Conditionally Ready** |

---

## Key Findings

### What is strong

1. **Strategy is production-ready**: 14 domain tests, 21 projection tests, 10 handler tests, full config integration, comprehensive documentation (2400+ line domain design).
2. **Governance covers strategy**: STD-1..STD-5 drift rules defined, quality gate profiles operational, config symmetry enforced.
3. **Config dependency chain is bulletproof**: Transitive validation (observation → evidence → signal → decision → strategy) blocks invalid configurations at startup. Extending to risk is mechanical.
4. **Projection authority is consistent**: Single-writer invariant verified across all 5 domains. Three-gate projection (final → validate → monotonicity) applied uniformly.
5. **Mesh is clean**: 5 JetStream streams, all single-writer, all consumed by expected downstream services.
6. **Activation model is two-layer**: Structural (config, requires restart) + runtime (binding watcher, dynamic per symbol). Proven pattern.
7. **Gateway remains stateless**: No pipeline config, no write access, conditional route registration.

### What needs attention

1. **Adapter test debt is systemic**: 10 publisher/consumer files across 5 domains have zero unit tests. The pattern has never been tested at the adapter level.
2. **Derive actor test debt is systemic**: 12 derive actor files have zero unit tests. The most complex binary in the system has no actor-level test coverage.
3. **Ingest actor tests absent**: 5 ingest actor files untested. WebSocket reconnection and message parsing are critical paths with no unit verification.

### What is explicitly not a problem

- Strategy maturity: MET — the most well-documented and tested of all downstream domains.
- Strategy governance: MET — drift rules, guardrails, and CLI integration complete.
- Config safety: MET — dependency chain validated, unknown families rejected.
- Mesh clarity: MET — raccoon-cli verifies stream registry, naming, and config symmetry.
- Gateway cleanliness: MET — stateless, conditional, no pipeline config.
- Store authority: MET — sole writer, monotonicity guards, three-gate projection.

---

## Prerequisites Status

| # | Prerequisite | Status | Blocking? |
|---|-------------|--------|-----------|
| P-1 | Strategy domain maturity | MET | — |
| P-2 | Strategy governance verified | MET | — |
| P-3 | Config dependency chain complete | MET | — |
| P-4 | Projection authority consistent | MET | — |
| P-5 | Query surfaces clean | MET | — |
| P-6 | Mesh integrity verified | MET | — |
| P-7 | Adapter test coverage baseline | NOT MET | For implementation |
| P-8 | Derive actor test coverage | NOT MET | For implementation |
| P-9 | Risk domain design document | NOT MET | Sequential |
| P-10 | Risk governance rules in CLI | NOT MET | For implementation |

---

## Blocking Gaps

| ID | Gap | Severity | Files Affected | Remediation |
|----|-----|----------|---------------|-------------|
| BG-1 | No adapter publisher/consumer tests | HIGH | 10 files (all domains) | S60 |
| BG-2 | No derive actor tests | HIGH | 12 files | S61 |
| BG-3 | No ingest actor tests | MEDIUM | 5 files | S60/S61 |

---

## Non-Blocking Risks

| ID | Risk | Severity | Mitigation |
|----|------|----------|------------|
| NR-1 | Risk scope creep | MEDIUM | Tight "what risk is NOT" section in design |
| NR-2 | Binary placement | MEDIUM | Start in derive, extract if needed |
| NR-3 | No strategy history | LOW | Design for latest-only first |
| NR-4 | Single exchange | LOW | Per-source evaluation is valid |
| NR-5 | Binding deactivation incomplete | LOW | Platform-level, not risk-specific |
| NR-6 | QueryResponder growth | LOW | 8 routes is fine, split at 12+ |
| NR-7 | Cannot verify evaluator purity | LOW | Code review + possible import check |

---

## Answers to Review Questions

**Is strategy mature enough?**
Yes. Production-ready implementation with exceptional documentation, full config integration, and 21 projection tests.

**Is strategy governance sufficient?**
Yes. STD-1..STD-5 drift rules, guardrails documented, quality gate integration.

**Are config dependencies secure?**
Yes. Transitive validation enforced at startup. Mechanical extension for risk.

**Does the mesh remain clear and protected by raccoon-cli?**
Yes. 5 clean streams, single-writer, verified by CLI drift-detect and runtime-bindings checks.

**Does the gateway remain clean?**
Yes. Stateless, no pipeline config, conditional route registration.

**Does the store remain clear as authority?**
Yes. Sole writer to all KV buckets, three-gate projection, monotonicity guards.

**What gaps still prevent risk from entering without generating debt?**
Two systemic test gaps: adapter tests (BG-1) and derive actor tests (BG-2). These exist across all domains and are not risk-specific, but adding a 6th domain deepens them.

**What is the smallest acceptable risk design?**
Single family (e.g., position exposure), consuming strategy output, RISK_EVENTS stream, latest-only KV projection, one HTTP endpoint, full config chain, drift rules. No portfolio, no execution, no external feeds.

---

## Recommended Stage Sequence

```
S60: Adapter Test Coverage Sweep (Round 2)
  ├── Publisher adapter test pattern
  ├── Consumer adapter test pattern
  └── Replicate across 5 domains

S61: Derive Actor Test Coverage
  ├── Sampler actor test pattern
  ├── Evaluator/resolver actor test pattern
  └── Publisher actor test pattern

S62: Risk Domain Design          ← can run in parallel with S60/S61
  ├── risk-domain-design.md
  ├── risk-stream-families.md
  ├── risk-activation-and-ownership.md
  └── risk-query-surface-guidelines.md

S63: Risk Governance Activation  ← requires S62
  ├── RD-1..RD-5 drift rules
  ├── Risk guardrails in CLI
  └── Known risk families in constants

S64: Risk First Slice            ← requires S60, S61, S62, S63
  ├── Domain model + tests
  ├── Application evaluator + tests
  ├── Adapters + tests
  ├── Actors + tests
  ├── HTTP surface + tests
  └── Config integration

S65: Risk Projection Hardening   ← requires S64
  ├── Three-gate verified
  ├── Multi-symbol verification
  └── Smoke test updated
```

---

## Deliverables Produced

| Document | Path |
|----------|------|
| Risk Readiness Review | `docs/architecture/risk-readiness-review.md` |
| Risk Entry Prerequisites | `docs/architecture/risk-entry-prerequisites.md` |
| Risk Risks and Blockers | `docs/architecture/risk-risks-and-blockers.md` |
| Stage Report | `docs/stages/stage-s59-risk-readiness-review-report.md` |

---

## Stage Acceptance Criteria

- [x] Readiness review is specific, honest, and actionable
- [x] Foundry gains a real gate before opening risk
- [x] Gaps are clear and prioritizable (BG-1, BG-2, BG-3 with severity and remediation)
- [x] Next wave of stages can be planned based on evidence, not urgency
- [x] No risk implementation performed
- [x] No gaps masked to "move forward"
- [x] No vague abstractions — every finding tied to specific files and test counts
- [x] Clear separation between readiness and implementation
