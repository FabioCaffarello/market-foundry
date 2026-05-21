# Venue Readiness Next-Wave Options Matrix

**Stage:** S311 — Post-Charter Gate and Strategic Direction
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Companion:** `post-charter-gate-for-venue-readiness.md`

---

## 1. Purpose

This document compares the available strategic options for the next phase of work after the Venue Readiness Charter Wave (S306–S310). Each option is evaluated against evidence from the charter wave, with explicit trade-offs and a recommendation.

---

## 2. Options Under Evaluation

| Option | Name | Summary |
|--------|------|---------|
| **A** | Direct Implementation Wave | Proceed immediately to implementation stages (adapter hardening → E2E → failure verification → multi-symbol → gate) |
| **B** | Short Foundational Tranche + Implementation Wave | Execute a bounded 2–3 stage tranche to resolve EC-1 and adapter prerequisites, then proceed to E2E implementation |
| **C** | Lateral Pivot | Pause venue readiness; pursue a different capability direction (observability, new domain features, operational maturity) |

---

## 3. Option A: Direct Implementation Wave

### 3.1 Description

Proceed immediately from S311 to an implementation wave that delivers EC-1, adapter hardening, E2E venue integration, failure verification, and multi-symbol venue proof as a single continuous sequence.

### 3.2 Stage Sequence

| Stage | Scope |
|-------|-------|
| W1 | Adapter hardening: EC-1 (client order ID), EC-2 (body cap), EC-3 (deadline), VA-1 (error classification), RF-1 (retryable flags) |
| W2 | Fill model validation: real venue response parsing, CR-1–CR-5 enforcement, fill-intent consistency |
| W3 | E2E venue integration: intent → venue → fill → KV → ClickHouse → HTTP composite query |
| W4 | Failure injection and guard rail verification: F-VEN-01–F-VEN-11 under real conditions |
| W5 | Multi-symbol venue isolation proof: concurrent symbols to testnet |
| W6 | Evidence gate: VQ1–VQ7 answered with code evidence; wave closure |

### 3.3 Evaluation

| Factor | Assessment | Score |
|--------|-----------|-------|
| **Speed to venue readiness** | Fastest path — no intermediate gate | ++ |
| **Risk of scope inflation** | Higher — 6-stage wave without checkpoint | - |
| **EC-1 timing** | Implemented in W1 before E2E; safe | + |
| **Architectural confidence** | Design is complete; confidence is high | + |
| **Operational complexity** | Single continuous wave; simple tracking | + |
| **Regression risk** | Each stage builds on previous; integration failures accumulate | - |
| **Recovery from surprises** | No natural pause point until W6; course correction harder | -- |

### 3.4 Verdict

**Viable but risky.** The charter wave produced sufficient design artifacts for a direct implementation push. However, 6 stages without an intermediate checkpoint increases the cost of surprises. If EC-1 implementation reveals unexpected complexity (e.g., Binance client order ID format constraints), stages W2–W5 are affected.

---

## 4. Option B: Short Foundational Tranche + Implementation Wave

### 4.1 Description

Execute a bounded 2–3 stage tranche focused exclusively on resolving the S307 adapter prerequisites (EC-1, EC-2, EC-3, VA-1, RF-1), then proceed to E2E integration with full confidence.

### 4.2 Stage Sequence

**Tranche (2–3 stages):**

| Stage | Scope |
|-------|-------|
| T1 | **Adapter contract hardening:** EC-1 (client order ID derivation + Binance `newClientOrderId` integration), EC-2 (`io.LimitReader`), EC-3 (per-request `context.WithTimeout`) |
| T2 | **Error classification hardening:** VA-1 (complete Binance error code mapping), RF-1 (retryable flag on all `*problem.Problem` returns), OB-1 (structured venue response logging) |
| T3 | **Tranche gate:** Verify EC-1–EC-3, VA-1, RF-1 in isolation (unit + httptest); confirm adapter is hardened before E2E |

**Implementation Wave (3–4 stages, post-tranche):**

| Stage | Scope |
|-------|-------|
| I1 | E2E venue integration: real venue call → fill → KV → ClickHouse → HTTP query; kill switch + staleness enforcement proof |
| I2 | Failure injection + guard rail verification under real venue conditions |
| I3 | Multi-symbol venue isolation proof with concurrent testnet calls |
| I4 | Evidence gate: VQ1–VQ7 answered; wave closure |

### 4.3 Evaluation

| Factor | Assessment | Score |
|--------|-----------|-------|
| **Speed to venue readiness** | Slightly slower — adds 2–3 stages before E2E | - |
| **Risk of scope inflation** | Lower — tranche scope is 5 items, gate enforces boundary | ++ |
| **EC-1 timing** | Resolved and verified in T1 before any venue call | ++ |
| **Architectural confidence** | Tranche validates assumptions before committing to E2E | ++ |
| **Operational complexity** | Two phases; slightly more tracking | 0 |
| **Regression risk** | Lower — adapter hardened and tested before integration | + |
| **Recovery from surprises** | T3 gate provides natural checkpoint; course correction easy | ++ |

### 4.4 Verdict

**Recommended.** The tranche adds 2–3 stages but provides a critical checkpoint between "adapter is hardened" and "adapter talks to real venue." The cost is small (adapter hardening is a bounded set of changes); the risk reduction is significant (EC-1 proven before any real venue interaction).

---

## 5. Option C: Lateral Pivot

### 5.1 Description

Pause venue readiness. Redirect effort to a different capability direction.

### 5.2 Candidate Directions

| Direction | Rationale | Risk |
|-----------|----------|------|
| Observability maturity | OpenTelemetry, Prometheus, Grafana dashboards | Scope inflation; no venue progress |
| New domain features | Additional signal families, strategy variations | Lateral expansion without venue depth |
| Operational hardening | SRE practices, runbooks, on-call procedures | Premature for testnet-only system |
| Multi-venue architecture | Abstract venue layer for future exchanges | Premature; single venue not yet proven |

### 5.3 Evaluation

| Factor | Assessment | Score |
|--------|-----------|-------|
| **Speed to venue readiness** | Zero progress — venue readiness stalled | -- |
| **Risk of scope inflation** | High — new direction opens new questions | -- |
| **EC-1 timing** | Deferred indefinitely | -- |
| **Architectural confidence** | Does not address venue gap | - |
| **Operational complexity** | New charter, new scope freeze needed | - |
| **Regression risk** | None (no venue code) but design knowledge decays | - |
| **Recovery from surprises** | N/A — no venue work | 0 |

### 5.4 Verdict

**Not recommended.** The charter wave invested 5 stages in venue readiness design. Pivoting away would waste that investment and leave the venue gap unaddressed. The system's primary value proposition (signal → decision → strategy → risk → execution) depends on venue readiness. No other direction produces comparable strategic value.

---

## 6. Comparative Matrix

| Factor | Option A (Direct) | Option B (Tranche) | Option C (Pivot) |
|--------|------------------|-------------------|-----------------|
| Speed to venue readiness | ++ | + | -- |
| Risk control | - | ++ | -- |
| EC-1 resolution confidence | + | ++ | -- |
| Scope discipline | - | ++ | -- |
| Recovery from surprises | -- | ++ | 0 |
| Strategic coherence | + | ++ | -- |
| Operational overhead | + | 0 | - |
| Design investment preservation | ++ | ++ | -- |
| **Overall** | **Viable** | **Recommended** | **Not recommended** |

---

## 7. Recommendation

### 7.1 Primary Direction: Option B — Short Foundational Tranche

**Execute a bounded 2–3 stage adapter hardening tranche before proceeding to E2E venue integration.**

**Rationale:**
1. **EC-1 is the single highest-priority gap** and must be proven in isolation before any real venue call. Option B ensures this.
2. **The tranche scope is narrow and bounded** — 5 concrete items (EC-1, EC-2, EC-3, VA-1, RF-1) with clear acceptance criteria from S308–S310.
3. **The tranche gate (T3) provides a checkpoint** — if adapter hardening reveals unexpected problems, course correction happens before E2E, not during.
4. **The implementation wave post-tranche is simpler** — adapter is already hardened; E2E stages focus on integration, not adapter bugs.
5. **No design work is needed** — all contracts, invariants, and specifications exist from S308–S310. The tranche is pure implementation against existing specs.

### 7.2 Secondary Direction: None

**Do not open a secondary direction.** Single-front discipline has been a consistent strength of the Foundry's stage governance. The charter wave's own non-goals explicitly block lateral expansion.

### 7.3 What NOT to Open

| Direction | Why Not |
|-----------|--------|
| Observability wave | Premature; testnet doesn't need dashboards; structured logging sufficient |
| New domain features | Venue readiness is the strategic priority; new features don't address the gap |
| Multi-venue abstraction | Single venue not yet proven; abstraction before evidence is premature |
| OMS implementation | S309 explicitly proved no OMS module is needed |
| Retry infrastructure | Blocked by EC-1 (RT-1 constraint); cannot start until tranche delivers EC-1 |
| Production hardening | System is testnet-only; production hardening is premature |

---

## 8. Tranche Scope Freeze (If Option B Accepted)

### 8.1 In Scope

| Item | Source | Specification | Acceptance Criterion |
|------|--------|--------------|---------------------|
| EC-1: Client order ID | S307/S308 IDEM-3 | Deterministic derivation from intent fields; sent as `newClientOrderId` to Binance | Unit test: same intent → same ID; different intent → different ID. httptest: ID present in request |
| EC-2: Response body cap | S307/S310 PGR-14 | `io.LimitReader(body, 64*1024)` | Unit test: oversized response truncated and classified as parse error |
| EC-3: Per-request deadline | S307/S310 PGR-03 | `context.WithTimeout(ctx, d)` wrapping every `VenuePort.SubmitOrder` call | Unit test: slow httptest server triggers context deadline |
| VA-1: Error classification | S307/S308 C-FAIL | All 8 failure classes return `*problem.Problem` with correct category and retryable flag | Unit test: each HTTP status code → correct problem category |
| RF-1: Retryable flags | S307/S310 | Every `*problem.Problem` carries correct `Retryable` field | Unit test: 429/503/5xx → retryable=true; 400/401/403 → retryable=false |

### 8.2 Out of Scope (Tranche Guard Rails)

| Item | Why Out |
|------|---------|
| E2E venue call | Tranche hardens adapter; E2E is implementation wave |
| Retry infrastructure | Blocked until EC-1 proven (RT-1) |
| Multi-symbol testing | Requires E2E first |
| ClickHouse schema changes | S306 non-goal |
| New HTTP endpoints | S306 non-goal |
| Kill switch changes | Existing mechanism sufficient |
| Fill model code changes | C-FILL contracts already match existing adapter code |

### 8.3 Tranche Exit Criteria

| Criterion | Verification |
|-----------|-------------|
| EC-1 implemented and unit-tested | Client order ID present in all venue requests; deterministic derivation proven |
| EC-2 implemented and unit-tested | Body size cap active; oversized responses handled gracefully |
| EC-3 implemented and unit-tested | Per-request context deadline fires within configured time |
| VA-1 complete and unit-tested | All 8 C-FAIL classes return correct `*problem.Problem` |
| RF-1 consistent and unit-tested | Retryable flag correct for all failure classes |
| Zero regressions | Existing tests pass; paper pipeline unaffected |
| No scope inflation | Only 5 items delivered; no additional changes |

---

## 9. Implementation Wave Scope (Post-Tranche)

### 9.1 Stages

| Stage | Scope | Governing Questions |
|-------|-------|-------------------|
| I1 | E2E venue integration proof | VQ1, VQ3, VQ4, VQ6 |
| I2 | Failure injection + guard rail verification | VQ5 |
| I3 | Multi-symbol venue isolation proof | VQ7 |
| I4 | Evidence gate and wave closure | VQ1–VQ7 final |

### 9.2 Entry Conditions for Implementation Wave

| Condition | Source |
|-----------|--------|
| Tranche gate passed | T3 verdict |
| EC-1 proven in isolation | T1 unit tests |
| All C-FAIL classes implemented | T2 unit tests |
| Adapter hardened per S308 contracts | T1–T2 deliverables |
| Zero regressions against Phase 29 baseline | T3 regression check |

---

*Delivered: 2026-03-21 — Stage S311, Phase 30*
