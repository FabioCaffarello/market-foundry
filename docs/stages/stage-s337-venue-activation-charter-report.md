# Stage S337 — Venue Activation Wave Charter Report

> **Stage:** S337
> **Wave:** Venue Activation
> **Type:** Charter and Scope Freeze
> **Predecessor:** S336 — Live Stack Evidence Gate (FULL CLOSURE)
> **Status:** COMPLETE

---

## 1. Executive Summary

Stage S337 opens the **Venue Activation Wave**, the third verification wave in
the Foundry's progression toward operational execution capability.

The Live Stack Integration Wave (S332–S336) closed with **full closure**,
proving that the composed venue execution pipeline works end-to-end against
live NATS and ClickHouse infrastructure using the paper simulator. All 16
governing questions were answered with test evidence, 202+ tests remained
green, and the canonical 7-phase smoke script validated the complete stack.

The Venue Activation Wave converts this **proven technical pipeline** into
**operational capability** by activating the real Binance Futures testnet venue
adapter — the code path that already exists but has not been exercised in the
full stack context. This is the smallest step that delivers new evidence: real
fills from a real exchange.

The wave scope is frozen into five blocks (VA-1 through VA-5), following the
same governance model that proved effective in the Live Stack Integration Wave.

---

## 2. Consolidated State at Entry

### What the Foundry Has (Proven)

| Capability | Evidence |
|------------|----------|
| Full data path: Signal → Decision → Strategy → Risk → Execute → Fill → Persist → Read | S334 CRI-7, CRI-8, CRI-9 |
| Full control path: HTTP → NATS request/reply → KV store | S335 smoke phase 7 |
| Durable consumer delivery with restart recovery | S333 LF-2 |
| Correlation/causation chain immutable across NATS | S333 LF-1 |
| Kill-switch dual-checkpoint pattern | S335 CP-FP-2, CP-FP-4 |
| Fail-open safety semantics | S335 CG-RT-1, safety_gate_test |
| Composite visibility (venue fill wins over paper) | S334 CRI-8 |
| 7-phase canonical smoke script | S335 smoke-live-stack.sh |
| 202+ regression tests GREEN | S336 gate |
| 9/9 Production Wiring Tranche invariants HELD | S336 audit |

### What the Foundry Does NOT Have (Unproven)

| Gap | Source |
|-----|--------|
| Real HTTP requests to venue API | Never exercised in full stack |
| Real fill data from exchange | Paper simulator only |
| Activation/rollback procedures | No operational documentation |
| Credential management for venue access | No injection validation |
| Smoke validation with real venue adapter | Paper-only smoke |

### Residual Gaps Inherited from S336

| Gap | Severity | This Wave |
|-----|----------|-----------|
| G-1: 24h+ continuous observation | Medium | Deferred (Production Readiness) |
| G-2: Partial fills with real venue data | Low | **Addressed by VA-3/VA-4** |
| G-3: Commission uses cumQuote proxy | Low | **Addressed by VA-3/VA-4** |
| G-4: Paper bridge subject mapping | Low | **Addressed by VA-2** |
| G-5: Halt/resume under sustained load | Medium | Deferred (Production Readiness) |
| G-6: No per-symbol gate isolation | N/A | Non-goal (NG-9) |
| G-7: No WebSocket/SSE async fills | N/A | Non-goal (NG-10) |
| G-8: Single venue only | N/A | Non-goal (NG-2) |

---

## 3. Wave Charter

### Definition

Venue activation is the controlled transition from paper simulator to real
venue adapter within the execute binary. The transition is configuration-driven
(not code-driven), operator-initiated (not automated), and reversible
(rollback to paper is always available).

### Frozen Blocks

| Block | ID | Stage | Name |
|-------|----|-------|------|
| 1 | VA-1 | S338 | Activation Policy and Rollout Model |
| 2 | VA-2 | S339 | Canonical Activation Surface and Runtime Controls |
| 3 | VA-3 | S340 | Venue-Active Smoke and Acceptance Scenarios |
| 4 | VA-4 | S341 | Controlled Live Activation Verification |
| 5 | VA-5 | S342 | Evidence Gate Final |

### Governance

- Frozen scope: no blocks added or removed after this charter.
- Sequential execution: each block closes before the next starts.
- Evidence-based closure: test artifacts, not assertions.
- Invariant preservation: 202+ tests green at every block boundary.

Full charter details:
[venue-activation-wave-charter-and-scope-freeze.md](../architecture/venue-activation-wave-charter-and-scope-freeze.md)

---

## 4. Governing Questions

The wave defines **18 governing questions** across 5 blocks. Each question
maps to a concrete evidence type (integration test, smoke output, document,
or observation).

**VA-1 (Policy):** 4 questions — checklist executability, configuration-only
activation, clean rollback, responsibility assignment.

**VA-2 (Surface):** 5 questions — adapter bootstrap, credential validation,
decorator chain, kill-switch identity, staleness guard identity.

**VA-3 (Smoke):** 5 questions — real order round-trip, ClickHouse persistence,
composite visibility, kill-switch during smoke, acceptance scenario pass.

**VA-4 (Verification):** 4 questions — sequence completion, behavioral delta,
rollback confirmation, new gap analysis.

**VA-5 (Gate):** 5 questions (meta) — all prior questions answered, regression
green, both smoke scripts pass, no blocked items.

Full question matrix:
[venue-activation-capabilities-questions-and-non-goals.md](../architecture/venue-activation-capabilities-questions-and-non-goals.md)

---

## 5. Non-Goals (Explicit)

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-1 | Mainnet activation | Requires separate risk/compliance review |
| NG-2 | Multi-venue expansion | Single-venue must be proven first |
| NG-3 | Order Management System | Large separate domain |
| NG-4 | Portfolio risk management | Depends on OMS |
| NG-5 | Dashboards and monitoring UI | Production readiness concern |
| NG-6 | New breadth or domain expansion | Wave is depth-focused |
| NG-7 | Runtime redesign | Runtime is proven |
| NG-8 | Automated activation / feature flags | Manual activation is correct for testnet |
| NG-9 | Per-symbol gate isolation | Global gate is intentional design |
| NG-10 | WebSocket / streaming fills | REST polling is sufficient |

---

## 6. Stage Ordering

| Stage | Block | Deliverable | Key Evidence |
|-------|-------|-------------|--------------|
| **S337** | — | Charter and scope freeze (this stage) | Documents accepted |
| **S338** | VA-1 | Activation policy, rollout model, operator checklist | Checklist dry-run, rollback tested |
| **S339** | VA-2 | Runtime validation, credential injection, startup gates | Integration tests with real adapter config |
| **S340** | VA-3 | Venue-active smoke script, acceptance scenarios | Smoke passes with real venue |
| **S341** | VA-4 | Full activation sequence, behavioral delta | End-to-end real venue execution |
| **S342** | VA-5 | Evidence gate, closure verdict | Complete evidence matrix |

---

## 7. Preparation for S338

The next stage (S338: Activation Policy and Rollout Model) requires:

1. **Binance Futures testnet credentials** — API key and secret must be
   available before S339, but the policy (S338) can be drafted without them.

2. **Current operator model understanding** — Review how `make live` and
   `make smoke-live-stack` are currently used to establish the baseline
   operator workflow.

3. **Configuration schema review** — Read `internal/shared/settings/schema.go`
   to understand current `VenueConfig` structure and what fields are already
   defined for `binance_futures_testnet`.

4. **Environment variable conventions** — Review `cmd/execute/run.go` to
   understand current credential injection patterns.

5. **Kill-switch operational guide** — Read
   `docs/architecture/live-control-path-smoke-usage-and-operational-limitations.md`
   as the baseline for operator procedures.

**Recommended S338 deliverables:**
- `docs/operations/venue-activation-policy-and-rollout-model.md`
- `docs/stages/stage-s338-activation-policy-report.md`

---

## 8. Artifacts Produced

| Artifact | Path |
|----------|------|
| Wave charter | `docs/architecture/venue-activation-wave-charter-and-scope-freeze.md` |
| Capabilities and non-goals | `docs/architecture/venue-activation-capabilities-questions-and-non-goals.md` |
| Stage report | `docs/stages/stage-s337-venue-activation-charter-report.md` |

---

## 9. Verdict

**S337 is COMPLETE.**

The Venue Activation Wave is formally open with frozen scope. Five blocks
(VA-1 through VA-5) are defined with governing questions, evidence criteria,
and non-goals. The next stage (S338) can begin when the charter is accepted.

The wave follows the proven governance model: frozen scope, sequential blocks,
evidence-based closure, invariant preservation. No mainnet, no multi-venue,
no OMS, no runtime redesign.

The Foundry is ready to prove that its pipeline works with real venue data.
