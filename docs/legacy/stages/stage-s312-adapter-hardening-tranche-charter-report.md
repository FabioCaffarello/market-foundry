# Stage S312 — Adapter Hardening Tranche Charter

**Status:** DELIVERED
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Predecessor:** S311 — Post-Charter Gate and Strategic Direction
**Successor:** S313 — Adapter Contract Hardening (EC-1, EC-2, EC-3)

---

## 1. Executive Summary

S312 transforms the S311 recommendation (Option B: Short Foundational Tranche) into a **formal, scope-frozen charter** for the Adapter Hardening Tranche. The tranche contains exactly 5 items — EC-1, EC-2, EC-3, VA-1, RF-1 — all derived from S307 gap map with complete specifications in S308 contracts and S310 guard rails. No design work is needed; every item is pure implementation against existing specs.

The tranche is structured as 3 stages (S313–S315): adapter contract hardening, error classification hardening, and tranche gate. The gate produces a PASS/FAIL verdict that controls entry to the E2E implementation wave.

**Key decisions:**
- Tranche scope frozen at 5 items — no additions permitted
- No E2E venue calls within the tranche — isolation-only verification
- No VenuePort interface redesign — implementation gaps only
- 25 non-goals explicitly enumerated to prevent inflation
- 40 testable exit criteria across 5 items + 10 gate criteria

---

## 2. Deliverables

| # | Artefact | Path | Content |
|---|---------|------|---------|
| 1 | Tranche Charter and Scope Freeze | `docs/architecture/adapter-hardening-tranche-charter-and-scope-freeze.md` | Formal charter, tranche structure, dependency chain, inflation prevention rules, constraints |
| 2 | Items Exit Criteria and Non-Goals | `docs/architecture/adapter-hardening-items-exit-criteria-and-non-goals.md` | Per-item exit criteria (40 criteria), tranche gate criteria (10), non-goals (25) |
| 3 | Stage Report | `docs/stages/stage-s312-adapter-hardening-tranche-charter-report.md` | This document |

---

## 3. What S312 Established

### 3.1 Tranche Identity

| Property | Value |
|----------|-------|
| Name | Adapter Hardening Tranche |
| Items | 5 (frozen) |
| Stages | S313, S314, S315 |
| Design work | None — all specs from S308–S310 |
| Gate | S315 (PASS/FAIL verdict) |
| Successor | Implementation Wave (I1–I4) |

### 3.2 Item Summary

| Item | Name | Stage | Priority | Exit Criteria Count |
|------|------|-------|----------|-------------------|
| EC-1 | Client order ID derivation | S313 | Critical | 6 |
| EC-2 | Response body size cap | S313 | Low | 5 |
| EC-3 | Per-request context deadline | S313 | Medium | 6 |
| VA-1 | Error classification completeness | S314 | High | 13 |
| RF-1 | Retryable flag completeness | S314 | High | 10 |
| — | **Total** | — | — | **40** |

### 3.3 Critical Path

```
EC-1 ──▶ VA-1 ──▶ RF-1 ──▶ Tranche Gate (S315)
  │                           │
  ├── EC-2 (parallel)         │
  └── EC-3 (parallel)         │
                              └──▶ Implementation Wave (I1–I4)
```

EC-1 is the root of the critical path. VA-1 depends on adapter error paths being stable (which EC-1/EC-2/EC-3 affect). RF-1 depends on VA-1 classes being complete.

### 3.4 Dependency Chain Verified

| Dependency | From | To | Status |
|-----------|------|----|--------|
| S308 IDEM-3 spec | S308 | EC-1 | Available |
| S310 PGR-14 spec | S310 | EC-2 | Available |
| S310 PGR-03 spec | S310 | EC-3 | Available |
| S308 C-FAIL spec | S308 | VA-1 | Available |
| S310 failure modes | S310 | RF-1 | Available |
| EC-1/EC-2/EC-3 stable | S313 | VA-1 | Sequential (S313 before S314) |
| VA-1 complete | S314 | RF-1 | Sequential within S314 |

---

## 4. Scope Discipline

### 4.1 Inflation Prevention

6 inflation rules (IR-1 through IR-6) established:
- No new items after charter
- Discovered gaps become residuals, not tranche work
- No unrelated code changes
- No design documents
- Gate verifies exactly 5 items
- Oversized items logged as residuals for implementation wave

### 4.2 Non-Goals Count

25 non-goals across 5 categories:
- Venue integration: 5 non-goals (NG-1 through NG-5)
- Infrastructure: 5 non-goals (NG-6 through NG-10)
- Domain: 5 non-goals (NG-11 through NG-15)
- Architecture: 6 non-goals (NG-16 through NG-21)
- Process: 4 non-goals (NG-22 through NG-25)

### 4.3 Constraints Inherited

7 constraints (CN-1 through CN-7) inherited from S310 and this charter:
- No ClickHouse schema changes
- No new NATS subjects or KV buckets
- No new binaries or services
- No changes to derive or store binaries
- Paper pipeline zero-regression
- No design documents produced
- VenuePort interface unchanged

---

## 5. Tranche Gate Design

### 5.1 Gate Criteria

10 gate criteria (G-1 through G-10):
- G-1 through G-5: Per-item exit criteria pass (40 criteria total)
- G-6: All existing tests pass (zero regressions)
- G-7: Paper pipeline unaffected
- G-8: No scope inflation (exactly 5 items)
- G-9: Residual log published
- G-10: TQ1 answered (adapter hardened per specs)

### 5.2 Gate Verdicts

| Verdict | Action |
|---------|--------|
| PASS | Proceed to implementation wave |
| PASS WITH RESIDUALS | Proceed; residuals enter implementation wave backlog |
| FAIL | Remediate within tranche; do not proceed |

---

## 6. S313 Preparation

S313 (Adapter Contract Hardening) is the first implementation stage of the tranche. Its scope and entry conditions:

### 6.1 S313 Scope

| Item | Implementation |
|------|---------------|
| EC-1 | Deterministic `ClientOrderID()` function; integration into `VenueOrderRequest`; `newClientOrderId` in Binance HTTP request |
| EC-2 | `io.LimitReader(body, 64*1024)` on all venue response reads |
| EC-3 | `context.WithTimeout(ctx, timeout)` wrapping `SubmitOrder`; configurable timeout |

### 6.2 S313 Entry Conditions

| Condition | Status |
|-----------|--------|
| S312 charter delivered | SATISFIED |
| Tranche scope frozen | SATISFIED |
| S308 IDEM-3 spec available | SATISFIED |
| S310 PGR-03 and PGR-14 specs available | SATISFIED |
| Paper pipeline green | VERIFIED |
| Adapter code readable and understood | S313 will read before modifying |

### 6.3 S313 Expected Deliverables

1. `ClientOrderID()` implementation with unit tests
2. `io.LimitReader` integration with unit tests
3. `context.WithTimeout` integration with unit tests
4. httptest-based verification of all three items
5. Stage report documenting implementation decisions

### 6.4 S314 Preview

S314 (Error Classification Hardening) implements VA-1 and RF-1:
- Complete Binance error code → `*problem.Problem` mapping
- Retryable flag on all problem returns
- 23 combined exit criteria
- Depends on S313 completing (adapter error paths must be stable)

### 6.5 S315 Preview

S315 (Tranche Gate) is verification-only:
- Runs all 40 exit criteria
- Checks zero regressions
- Audits scope (5 items, no inflation)
- Produces gate verdict
- Logs residuals if any
- Produces tranche closure report

---

## 7. Acceptance Criteria Checklist

| Criterion | Met |
|-----------|-----|
| Tranche formalmente aberta com escopo congelado | ✓ — 5 items frozen; IR-1 through IR-6 prevent inflation |
| Os 5 itens ficam claramente delimitados | ✓ — each item has source spec, acceptance criteria, and stage assignment |
| Exit criteria ficam objetivos | ✓ — 40 per-item criteria + 10 gate criteria, all testable |
| Impede inflation e prepara S313–S315 | ✓ — 25 non-goals, 7 constraints, 6 inflation rules |
| Não implementar venue E2E | ✓ — NG-1 explicitly blocks real venue calls |
| Não redesenhar o VenuePort | ✓ — NG-16, CN-7 preserve interface |
| Não abrir OMS, dashboards ou retry infrastructure | ✓ — NG-6, NG-9, NG-11 block all three |
| Não transformar a tranche em nova wave conceitual ampla | ✓ — tranche definition (§2.1) distinguishes tranche from wave; 3-stage budget enforced |

---

*Delivered: 2026-03-21 — Stage S312, Phase 30*
