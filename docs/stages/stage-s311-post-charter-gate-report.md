# Stage S311 — Post-Charter Gate and Strategic Direction

**Status:** DELIVERED
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Predecessor:** S310 — Production Guard Rails and Failure Envelope
**Successor:** S312 — Venue Readiness Wave Closure

---

## 1. Executive Summary

S311 executes the **formal gate assessment** of the Venue Readiness Charter Wave (S306–S310). The charter wave invested 5 stages in design, producing 12 architecture documents that define contracts, invariants, lifecycle semantics, guard rails, and failure envelope for venue readiness — without changing a single line of venue-facing production code.

### Gate Verdict

**PARTIALLY READY — short foundational tranche recommended before implementation wave.**

The design foundation is **sound and complete**:
- Architecture does not need redesign (18 of 36 capabilities already exist)
- Domain model is production-sufficient (no changes to ExecutionIntent or FillRecord)
- State machine is correct (`validTransitions` matches venue lifecycle exactly)
- 13 of 14 guard rails are already in production code
- 23 failure modes are classified with explicit acceptable/unacceptable boundaries

One critical gap prevents direct implementation: **EC-1 (venue-side client order ID)** — the single item that separates "testnet-safe under no-retry" from "safe for any retry scenario." Five additional adapter items (EC-2, EC-3, VA-1, RF-1, OB-1) require hardening before E2E integration.

### Recommended Direction

**Option B: Short Foundational Tranche (2–3 stages) + Implementation Wave (3–4 stages).**

The tranche resolves EC-1 and adapter prerequisites in isolation. The implementation wave then proceeds to E2E venue integration with full confidence. This adds 2–3 stages to the timeline but provides a critical checkpoint between "adapter is hardened" and "adapter talks to real venue."

---

## 2. Deliverables

| # | Artefact | Path | Content |
|---|---------|------|---------|
| 1 | Post-Charter Gate Assessment | `docs/architecture/post-charter-gate-for-venue-readiness.md` | Formal assessment, blocker ledger, risk analysis, hard truths |
| 2 | Next-Wave Options Matrix | `docs/architecture/venue-readiness-next-wave-options-matrix.md` | 3 options compared, tranche scope freeze, implementation wave outline |
| 3 | Stage Report | `docs/stages/stage-s311-post-charter-gate-report.md` | This document |

---

## 3. Charter Wave Assessment

### 3.1 What the Wave Proved

| Finding | Evidence |
|---------|---------|
| Architecture does not need redesign | S307: 18 of 36 capabilities already exist and are reusable |
| Domain model is production-sufficient | S308: ExecutionIntent, FillRecord, Status lifecycle unchanged |
| State machine is correct | S308: `validTransitions` map matches venue lifecycle exactly |
| Guard rails are already in code | S310: 13 of 14 exist; only PGR-14 (body cap) is new |
| Failure modes are bounded | S310: 23 modes classified; envelope is auditable |
| OMS scope is locked | S309: 15 non-goals prevent creep; no new module needed |
| The gap is narrow | S307: concentrated in adapter hardening (EC-1/2/3, VA-1) |

### 3.2 What the Wave Did NOT Prove

| Gap | Required For |
|-----|-------------|
| Real venue call succeeds | VQ1 code-level evidence |
| Real fills persist correctly | VQ3 code-level evidence |
| Composite read model with real data | VQ4 code-level evidence |
| Failure injection resilience | VQ5 code-level evidence |
| Multi-symbol venue concurrency | VQ7 code-level evidence |
| Client order ID works | EC-1 resolution |

### 3.3 Readiness Classification

| Level | Definition | Applies? |
|-------|-----------|----------|
| Ready | Direct implementation wave | No — EC-1 gap |
| **Partially Ready** | **Design complete; short tranche needed** | **Yes** |
| Not Ready | Fundamental design gaps | No |

---

## 4. Blocker Ledger

### 4.1 Resolved Blockers (9)

| ID | Blocker | Resolved By |
|----|---------|------------|
| VA-2 | EXPIRED status mapping | S308 contract C-ACK |
| VA-3 | CANCELED status mapping | S308 contract C-ACK |
| VA-4 | REJECTED status mapping | S308 contract C-ACK |
| FM-1 | Real price mapping | S308 contract C-FILL |
| FM-2 | Real quantity mapping | S308 contract C-FILL |
| FM-3 | Real fee mapping | S308 contract C-FILL |
| FM-4 | Real timestamp mapping | S308 contract C-FILL |
| FM-5 | Simulated flag = false | S308 contract C-FILL |
| RF-4 | Network failure classification | S310 failure modes |

### 4.2 Open Blockers (6)

| ID | Blocker | Design Spec | Code Status | Priority |
|----|---------|------------|-------------|----------|
| **EC-1** | Client order ID | S308 IDEM-3 | **Not implemented** | **Critical** |
| EC-2 | Response body size cap | S310 PGR-14 | Not implemented | Low |
| EC-3 | Per-request context deadline | S310 PGR-03 | Not implemented | Medium |
| VA-1 | Error classification completeness | S308 C-FAIL | Partial | High |
| RF-1 | Retryable flag on all errors | S310 failure modes | Partial | High |
| PC-1/PC-2 | Kill switch + staleness under venue | S310 PGR-01/PGR-09 | Exists but unproven under venue | Medium |

### 4.3 Critical Path

```
EC-1 ──▶ VA-1 + RF-1 ──▶ Tranche Gate ──▶ E2E Integration ──▶ Failure Injection ──▶ Multi-Symbol ──▶ Evidence Gate
   │         │                 │
   └── EC-2  └── EC-3         └── PC-1/PC-2 proven here
```

**EC-1 is the root of the critical path.** Everything downstream depends on adapter hardening being complete.

---

## 5. Options Evaluated

### 5.1 Comparative Matrix

| Factor | A: Direct | B: Tranche | C: Pivot |
|--------|----------|-----------|---------|
| Speed to venue readiness | ++ | + | -- |
| Risk control | - | ++ | -- |
| EC-1 confidence | + | ++ | -- |
| Scope discipline | - | ++ | -- |
| Recovery from surprises | -- | ++ | 0 |
| Strategic coherence | + | ++ | -- |
| Design investment preservation | ++ | ++ | -- |
| **Verdict** | **Viable** | **Recommended** | **Not recommended** |

### 5.2 Why Not Option A (Direct)

Option A is viable but places EC-1 implementation and E2E integration in a single continuous wave without checkpoint. If EC-1 reveals unexpected complexity (Binance `newClientOrderId` format constraints, derivation collisions), all downstream stages are affected with no natural pause point.

### 5.3 Why Not Option C (Pivot)

Option C wastes 5 stages of charter wave investment and leaves the primary capability gap (signal → execution → real venue) unaddressed. No other direction produces comparable strategic value.

---

## 6. Recommended Direction

### 6.1 Primary: Option B — Short Foundational Tranche

**Scope:** 5 concrete items from S307–S310 specifications.

| Item | Source | Implementation |
|------|--------|---------------|
| EC-1 | S308 IDEM-3 | Deterministic client order ID derivation; `newClientOrderId` in Binance requests |
| EC-2 | S310 PGR-14 | `io.LimitReader(body, 64*1024)` |
| EC-3 | S310 PGR-03 | `context.WithTimeout` wrapping `SubmitOrder` |
| VA-1 | S308 C-FAIL | Complete error code → `*problem.Problem` mapping with correct categories |
| RF-1 | S310 failure modes | Retryable flag on all problem returns |

**Tranche structure:**
- T1: Adapter contract hardening (EC-1, EC-2, EC-3)
- T2: Error classification hardening (VA-1, RF-1)
- T3: Tranche gate (verify in isolation; zero regressions)

**Exit criteria:**
- EC-1 unit-tested: same intent → same ID; different intent → different ID
- EC-2 unit-tested: oversized response → parse error
- EC-3 unit-tested: slow server → context deadline exceeded
- VA-1 unit-tested: all 8 C-FAIL classes → correct `*problem.Problem`
- RF-1 unit-tested: retryable flag correct for all classes
- Zero regressions against paper pipeline

### 6.2 After Tranche: Implementation Wave

| Stage | Scope | Questions Answered |
|-------|-------|-------------------|
| I1 | E2E venue integration proof | VQ1, VQ3, VQ4, VQ6 |
| I2 | Failure injection + guard rail verification | VQ5 |
| I3 | Multi-symbol venue isolation proof | VQ7 |
| I4 | Evidence gate and wave closure | VQ1–VQ7 final |

### 6.3 Secondary Direction: None

Single-front discipline. No parallel work streams.

---

## 7. What NOT to Open

| Direction | Why Not |
|-----------|--------|
| Observability wave | Testnet doesn't need dashboards; structured logging sufficient |
| New domain features | Venue readiness is strategic priority |
| Multi-venue abstraction | Single venue not yet proven |
| OMS implementation | S309 proved no OMS module needed |
| Retry infrastructure | Blocked by EC-1 (RT-1 constraint) |
| Production hardening | System is testnet-only |

---

## 8. Governing Question Status at Gate

| Question | Design | Code | Required For Closure |
|----------|--------|------|---------------------|
| VQ1: Adapter submits and receives? | ✓ | Pending | E2E venue call (I1) |
| VQ2: Lifecycle reflects venue states? | ✓ | ✓ | Already in code |
| VQ3: Real fills persist correctly? | ✓ | Pending | E2E fill persistence (I1) |
| VQ4: Composite read with real data? | ✓ | Pending | E2E composite query (I1) |
| VQ5: Failures classified and contained? | ✓ | Partial | Failure injection (I2) |
| VQ6: Safety gate enforced? | ✓ | ✓ (paper) | Venue-path proof (I1) |
| VQ7: Multi-symbol isolation? | Partial | Pending | Concurrent venue proof (I3) |

---

## 9. S312 Preparation

S312 (Venue Readiness Wave Closure) serves as the final evidence gate. Its entry conditions:

| Condition | Source |
|-----------|--------|
| Tranche delivered and gated | T3 verdict |
| E2E venue integration proven | I1 deliverables |
| Failure injection passed | I2 deliverables |
| Multi-symbol isolation proven | I3 deliverables |
| VQ1–VQ7 answerable with evidence | I1–I3 evidence |
| Zero regressions vs. Phase 29 | I4 regression check |

S312 will produce:
- Final VQ1–VQ7 evidence matrix
- Wave closure verdict
- Successor wave recommendation

---

## 10. Acceptance Criteria Checklist

| Criterion | Met |
|-----------|-----|
| Avaliação formal e específica da charter wave | ✓ — 12 criteria assessed; 10 PASS, 1 PARTIAL, 1 PASS |
| Blockers e gaps residuais explícitos | ✓ — 9 resolved, 6 open, critical path mapped |
| Próxima direção recomendada com base em evidência | ✓ — Option B with comparative matrix of 3 options |
| Disciplina estratégica mantida | ✓ — single-front; 6 directions explicitly blocked |
| Não implementar venue readiness nesta etapa | ✓ — gate and decision only |
| Não usar critérios vagos | ✓ — every criterion has specific evidence |
| Não esconder gaps críticos | ✓ — EC-1, phantom orders, fail-open kill switch documented |
| Não abrir múltiplas frentes paralelas | ✓ — secondary direction: none |

---

*Delivered: 2026-03-21 — Stage S311, Phase 30*
