# Stage S310 — Production Guard Rails and Failure Envelope

**Status:** DELIVERED
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Predecessor:** S309 — OMS and Order Lifecycle Charter
**Successor:** S311 — Multi-Symbol Venue Isolation Proof

---

## 1. Executive Summary

S310 defines the **minimum mandatory guard rails** and the **explicit failure envelope** that must hold before the market-foundry pipeline transitions from paper execution to real venue order flow on Binance testnet.

**Key outcomes:**

1. **14 production guard rails** (PGR-01 through PGR-14) defined with enforcement points, violation responses, and dependency status — all but one (PGR-14 / response body cap) are already implemented in existing code.

2. **Explicit no-retry policy** for S310 scope — deliberate decision grounded in the venue-side idempotency gap (EC-1). Retry architecture constraints (RT-1 through RT-7) documented for post-S312 implementation.

3. **Three-tier failure classification**: pre-submission (6 modes, all acceptable), venue submission (11 modes, mixed), post-submission (6 modes, mostly acceptable). Total: 23 classified failure modes with concrete system responses.

4. **Acceptable vs. unacceptable failure boundary** explicitly drawn: transient venue errors are absorbed; invariant violations, duplicate orders, credential exposure, and terminal state mutations are unacceptable regardless of environment.

5. **Kill switch semantics** formalized: fail-open, no replay on resume, global scope sufficient for testnet.

6. **Reconciliation boundary** explicit: 6 self-verifiable properties; 5 properties require manual audit. Phantom order investigation procedure documented.

**Verdict:** The failure envelope is bounded, auditable, and sufficient for testnet venue readiness. Ready for S311 multi-symbol venue isolation proof.

---

## 2. Deliverables

| # | Artefact | Path | Content |
|---|---------|------|---------|
| 1 | Guard Rails & Failure Envelope | `docs/architecture/production-guard-rails-and-failure-envelope.md` | 14 guard rails, kill switch semantics, idempotency boundaries, retry policy, timeout semantics, reconciliation boundaries, stop mechanisms, dependency map |
| 2 | Failure Modes & Reconciliation | `docs/architecture/failure-modes-idempotency-retries-and-reconciliation-boundaries.md` | 23 failure modes matrix, acceptable/unacceptable classification, idempotency detail, timeout hierarchy, manual reconciliation procedures, verification matrix |
| 3 | Stage Report | `docs/stages/stage-s310-production-guard-rails-and-failure-envelope-report.md` | This document |

---

## 3. Guard Rails Defined

### 3.1 Production Guard Rail Summary

| ID | Guard Rail | Existing | Critical |
|----|-----------|---------|----------|
| PGR-01 | Kill switch pre-check | Yes | Yes |
| PGR-02 | State monotonicity | Yes | Yes |
| PGR-03 | Context deadline on venue calls | Yes | Yes |
| PGR-04 | Fill record from venue only | Yes | Yes |
| PGR-05 | Simulated=false for venue fills | Yes | Yes |
| PGR-06 | VenueOrderID required | Yes | Yes |
| PGR-07 | Credential isolation | Yes | Yes |
| PGR-08 | No intermediate state on failure | Yes | Yes |
| PGR-09 | Staleness guard | Yes | Yes |
| PGR-10 | Side filter enforcement | Yes | Medium |
| PGR-11 | Single submission per intent | Yes | Yes |
| PGR-12 | Terminal state absorption | Yes | Yes |
| PGR-13 | Fill consistency enforcement | Yes | Yes |
| PGR-14 | Response body size cap | New (trivial) | Medium |

**Finding:** 13 of 14 guard rails are already implemented in existing code. Only PGR-14 (response body size cap, S307 EC-2) requires new code — a single line of `io.LimitReader`.

### 3.2 Guard Rail Categories

| Category | Guard Rails | Purpose |
|----------|-----------|---------|
| Pre-submission safety | PGR-01, PGR-09, PGR-10 | Prevent unsafe venue calls |
| State machine integrity | PGR-02, PGR-12, PGR-13 | Prevent state corruption |
| Venue interaction safety | PGR-03, PGR-04, PGR-05, PGR-06, PGR-14 | Ensure correct venue data handling |
| Security | PGR-07 | Credential protection |
| Idempotency | PGR-11 | Duplicate prevention |
| Error containment | PGR-08 | No partial state on failure |

---

## 4. Failure Envelope

### 4.1 Failure Mode Count

| Category | Count | Acceptable | Unacceptable | Conditional |
|----------|-------|-----------|-------------|-------------|
| Pre-submission (F-PRE) | 6 | 6 | 0 | 0 |
| Venue submission (F-VEN) | 11 | 4 | 4 | 3 |
| Post-submission (F-POST) | 6 | 4 | 2 | 0 |
| **Total** | **23** | **14** | **6** | **3** |

### 4.2 Unacceptable Failures (Must Never Be Silently Absorbed)

| Failure | Why Unacceptable | Required Response |
|---------|-----------------|-------------------|
| Invariant violation (F-POST-01, F-POST-02) | State machine or fill consistency corruption | Kill switch + investigate |
| Duplicate venue order | Financial risk | Kill switch + manual reconciliation |
| Credential exposure | Security violation | Immediate incident response |
| Terminal state mutation | Fundamental invariant break | Kill switch + investigate |
| Persistent auth failure (F-VEN-01) | Configuration bug | Credential reconfiguration |
| Persistent client error (F-VEN-02) | Code bug | Fix request construction |

### 4.3 Dangerous But Tolerated (Testnet Only)

| Failure | Why Dangerous | Why Tolerated on Testnet |
|---------|-------------|-------------------------|
| Phantom order (F-VEN-07) | Venue-side state divergence | No financial risk; manual audit procedure defined |
| Missing client order ID (EC-1 gap) | Retry could produce duplicates | No retry in S310; EC-1 deferred to S307 |
| Fail-open kill switch | Could submit during intended halt | KV unavailability is broader failure; testnet tolerance |

---

## 5. Idempotency Assessment

| Layer | Proven | Gap |
|-------|--------|-----|
| JetStream dedup (message level) | Yes | None |
| KV monotonicity (intent level) | Yes | None |
| Venue client order ID (venue level) | **No** | EC-1 — blocks production retry |

**S310 mitigation for EC-1 gap:** No automatic retry means no retry-induced duplicates. The gap only materializes if retry is introduced before EC-1 is implemented. RT-1 constraint explicitly blocks this.

---

## 6. Retry Decision

**S310 policy: No automatic retry.**

| Factor | Assessment |
|--------|-----------|
| Safety | Cannot safely retry without venue-side idempotency (EC-1) |
| Necessity | Testnet failures are acceptable; no financial risk |
| Complexity | Retry with backoff requires circuit breaker, jitter, max-attempts — scope inflation |
| Alternative | New pipeline evaluation cycle generates new intents naturally |

Future retry (post-S312) must satisfy 7 constraints (RT-1 through RT-7), with RT-1 (client order ID prerequisite) being the hard blocker.

---

## 7. Reconciliation Assessment

| What | Automated | Manual |
|------|-----------|--------|
| State transition consistency | Yes | — |
| Fill-intent quantity match | Yes | — |
| Fill causality (timestamps) | Yes | — |
| Simulated flag consistency | Yes | — |
| KV-ClickHouse consistency | Yes (eventual) | — |
| Dedup key uniqueness | Yes | — |
| Phantom order detection | — | Yes (procedure documented) |
| Venue-side order state | — | Yes (Binance API query) |
| Cross-session reconciliation | — | Yes (restart clears state) |
| Balance verification | — | Not in scope |
| Position reconciliation | — | Not in scope |

---

## 8. Residual Gaps

| Gap | Severity | Owner | Blocks |
|-----|----------|-------|--------|
| Client order ID (EC-1) | High | S307 | Production retry; venue-side idempotency |
| Response body size cap (EC-2) | Low | S307 | PGR-14 (trivial implementation) |
| Per-symbol kill switch | Low | Post-S312 | Nothing (global sufficient for testnet) |
| Automated reconciliation | Medium | Post-S312 | Production operations |
| Async fill handling (WebSocket) | Low | Post-S312 | Nothing (sync fills sufficient for market orders) |
| Retry infrastructure | Medium | Post-S312 | Production resilience |
| Operational dashboards | Medium | Post-S312 | Production monitoring |

---

## 9. S311 Preparation

S311 (Multi-Symbol Venue Isolation Proof) can now proceed with the following inputs from S310:

### 9.1 What S311 Inherits

| Input | From S310 |
|-------|----------|
| Guard rail registry | PGR-01 through PGR-14 — verify under multi-symbol load |
| No-retry policy | Simplifies isolation proof (no retry cross-contamination) |
| Failure classification | 23 failure modes — verify no cross-symbol propagation |
| Kill switch semantics | Global scope — verify halt stops all symbols |
| Idempotency layers | Per-symbol dedup keys — verify no cross-symbol collision |
| Error containment rules | FP-1 through FP-6 — verify FP-5 (symbol isolation) |

### 9.2 Recommended S311 Scope

1. **Symbol isolation proof under guard rails:** Verify that PGR-01 through PGR-14 hold independently for each symbol under concurrent multi-symbol execution
2. **Failure containment across symbols:** Prove that F-VEN failures for symbol A do not affect symbol B's processing
3. **Kill switch multi-symbol behavior:** Verify global halt stops all symbols; no partial halt state
4. **Dedup key isolation:** Verify dedup keys are symbol-scoped; no cross-symbol collision
5. **State machine independence:** Verify each symbol's intent transitions independently

### 9.3 What S311 Must NOT Do

- Implement per-symbol kill switch (global sufficient)
- Add retry infrastructure
- Open automated reconciliation
- Add new guard rails beyond S310 registry

---

## 10. Governing Question Progress

| Question | S310 Contribution |
|----------|------------------|
| VQ1: Adapter submits and receives? | Guard rails PGR-03 through PGR-08 define safe submission |
| VQ2: Lifecycle reflects venue states? | PGR-02, PGR-12 enforce correct transitions |
| VQ3: Real fills persist without schema changes? | PGR-04, PGR-05, PGR-13 ensure fill integrity |
| VQ4: Composite read works with real data? | No impact — read model unaffected by guard rails |
| VQ5: Failures classified and contained? | **Primary contribution:** 23 failure modes classified; envelope bounded |
| VQ6: Safety gate enforced? | PGR-01 (kill switch), PGR-09 (staleness), PGR-10 (side filter) |
| VQ7: Multi-symbol isolation maintained? | Error containment rule FP-5; S311 will prove under load |

---

## 11. Acceptance Criteria Checklist

| Criterion | Met |
|-----------|-----|
| Guard rails obrigatórios ficam claramente definidos | ✓ — 14 guard rails with IDs, enforcement points, and violation responses |
| Failure envelope fica explícito e auditável | ✓ — 23 failure modes classified with system responses |
| Dependências críticas ficam mapeadas | ✓ — EC-1 gap, dependency map for S311 |
| A etapa prepara base concreta para o gate S311 | ✓ — S311 scope and non-goals documented |
| Não implementar toda a resiliência | ✓ — No retry, no circuit breaker, no dashboards |
| Não abrir observabilidade de produção ampla | ✓ — Structured logging only |
| Não inflar para programa completo de SRE | ✓ — Testnet scope; manual reconciliation only |
| Não esconder classes de falha difíceis | ✓ — Phantom orders, EC-1 gap, fail-open kill switch explicitly documented |

---

*Delivered: 2026-03-21 — Stage S310, Phase 30*
