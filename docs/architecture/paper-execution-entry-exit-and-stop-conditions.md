# Paper Execution — Entry, Exit, and Stop Conditions

Stage: S264
Charter: PAPER-EXECUTION-WAVE-1
Date: 2026-03-21
Status: Active

---

## 1. Entry Conditions

| ID | Condition | Evidence | Status |
|----|-----------|----------|--------|
| EC-1 | S263 post-codegen reentry gate passed | Stage report verdict: PASS | Met |
| EC-2 | All five paper execution components implemented | `paper_order_evaluator.go`, `paper_fill_simulator.go`, `paper_venue_adapter.go`, `safety_gate.go`, `staleness_guard.go` present | Met |
| EC-3 | Decision → strategy → risk behavioral activation proven | S250, S251 stage reports; scenario tests green | Met |
| EC-4 | Execution actors wired in derive and store scopes | `execution_evaluator_actor.go`, `execution_publisher_actor.go`, `execution_projection_actor.go` present | Met |
| EC-5 | NATS execution streams configured | `EXECUTION_EVENTS`, `EXECUTION_FILL_EVENTS` in natsexecution registry | Met |
| EC-6 | Codegen equivalence checks passing (zero drift) | `codegen-equivalence-check.sh` green | Met |
| EC-7 | CI green on main branch | GitHub Actions workflow passing | Met |
| EC-8 | Charter formally opened and scope frozen | This document + charter document | Met (S264) |

## 2. Exit Conditions

| ID | Condition | Verification method |
|----|-----------|-------------------|
| EX-1 | All 7 minimum viable scenarios pass in CI | Automated test suite green in CI |
| EX-2 | Paper mode exclusively — no real venue code paths exercised | Code review + `grep` for real venue imports in test files |
| EX-3 | Guard rails proven active: staleness rejection, kill switch halt, no-action passthrough | Scenarios 2, 3, 7 with explicit assertions |
| EX-4 | Behavioral context preservation proven across full chain | Scenario 5 asserts correlation_id, causation_id, severity at execution boundary |
| EX-5 | Execution projection round-trip proven | Scenario 6 verifies KV materialization after event publish |
| EX-6 | Zero regression in existing test suite | CI green on all pre-existing tests |
| EX-7 | Zero codegen drift | `codegen-equivalence-check.sh` passes post-wave |
| EX-8 | Stage reports for S265–S268 delivered | All stage report files present in `docs/stages/` |

## 3. Stop Conditions

### Hard stops (immediate halt, revert if necessary)

| ID | Condition | Action |
|----|-----------|--------|
| HS-1 | Real venue API call detected in any code path or test | Immediate halt; revert offending change; charter review |
| HS-2 | Real money, real orders, or real positions introduced | Immediate halt; revert; incident review |
| HS-3 | New NATS stream, KV bucket, or JetStream infrastructure created | Immediate halt; revert; redesign within existing infrastructure |
| HS-4 | New domain model type or execution event type added | Immediate halt; revert; prove scenario with existing types |
| HS-5 | Behavioral test regression (S249–S257 tests fail) | Immediate halt; fix regression before any further wave work |
| HS-6 | Codegen equivalence drift detected and not immediately fixable | Immediate halt; restore equivalence before continuing |

### Soft stops (pause and assess)

| ID | Condition | Action |
|----|-----------|--------|
| SS-1 | CI regression exceeding 2 tests in non-execution packages | Pause wave; investigate root cause; fix before continuing |
| SS-2 | Scenario requires touching code outside execution boundary | Pause; assess if change is a bug fix (permitted) or scope expansion (prohibited) |
| SS-3 | Test infrastructure exceeds 15% hardening budget | Pause; assess if remaining scenarios can be proven with current infrastructure |
| SS-4 | Wiring correction requires actor lifecycle changes | Pause; assess if correction is minimal fix or architectural change |
| SS-5 | Tier 1 scenario cannot be proven as designed | Pause; redesign scenario within permitted changes; do NOT expand scope |

## 4. Amendment Conditions

| Condition | Allowed amendment | NOT allowed |
|-----------|------------------|-------------|
| Tier 1 scenario proves harder than expected | Redesign scenario approach; add test helpers | Remove scenario from Tier 1; expand scope |
| Bug discovered in existing execution code | Fix within execution boundary; add regression test | Rewrite component; change domain model |
| Tier 2 scenario reveals valuable insight | Promote insight to documentation; keep scenario in Tier 2 | Promote scenario to Tier 1 without charter review |
| External dependency update required | Update if security-critical only | Update for convenience or new features |
| New scenario idea emerges | Add to Tier 2 if it serves a Tier 1 item | Add to Tier 1 without charter amendment |

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Existing wiring has subtle bugs blocking loop closure | Medium | Medium | S265 boundary alignment validates wiring before scenario implementation |
| SafetyGate fail-open behavior masks real problems | Low | High | Scenario 3 explicitly tests kill switch halt; fail-open is documented and intentional |
| Execution projection has monotonicity edge cases | Low | Medium | Scenario 6 includes explicit timestamp ordering assertions |
| Behavioral context lost at execution boundary | Medium | High | Scenario 5 asserts full causal chain preservation |
| Scope creep toward real venue integration | Low | Critical | Hard stop HS-1; prohibited changes document; charter governance |
| Test infrastructure grows beyond budget | Medium | Low | 15% hardening budget cap; soft stop SS-3 |

## 6. Monitoring Checkpoints

| Checkpoint | When | What to verify |
|------------|------|---------------|
| Post-S265 | After boundary alignment | All entry conditions still met; wiring validated; no scope expansion |
| Mid-S266 | After first 3 scenarios pass | Progress on track; no prohibited changes introduced; CI green |
| Post-S266 | After all loop scenarios | All happy-path scenarios pass; guard rail scenarios ready |
| Post-S267 | After guard rail proof | All 7 scenarios pass; behavioral context preserved; guard rails active |
| Pre-S268 gate | Before gate assessment | All exit conditions met; zero regression; zero drift; all reports delivered |

## 7. Relationship to Deferred Debts

| Debt | Status | Interaction |
|------|--------|------------|
| Real venue integration | Deferred (future wave) | Paper execution wave explicitly does NOT address this; paper loop must pass first |
| OMS implementation | Deferred (future wave) | Requires proven paper loop as prerequisite |
| Portfolio / PnL tracking | Deferred (future wave) | Downstream of execution; no interaction with this wave |
| Multi-venue routing | Deferred (future wave) | Venue abstraction exists but is not exercised in this wave |
| Execution retry / recovery | Deferred (future wave) | Paper fills are instant; retry logic not needed |
| Performance optimization | Deferred (future wave) | Observation permitted in T2-2; optimization prohibited |
| Remaining 86% manual artifact coverage | Accepted debt (S263) | Domain logic stays manual by design; no codegen expansion in this wave |
