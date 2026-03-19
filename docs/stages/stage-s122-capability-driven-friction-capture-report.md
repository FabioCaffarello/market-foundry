# Stage S122 — Capability-Driven Friction Capture Report

> **Status:** Complete
> **Capability:** CC-01 — Multi-Symbol Live Monitoring
> **Scope:** Friction capture and architectural analysis of CC-01 live validation evidence
> **Predecessor:** S121 (Live Validation of Controlled Capability)

---

## 1. Executive Summary

S122 analyzed the operational evidence produced by CC-01 validation (S119–S121) to distinguish real frictions from hypothetical concerns. The analysis covered config activation, diagnostic surfaces, query shape, lifecycle, correlation, materialization, and runtime behavior across the full multi-symbol pipeline.

**Key finding:** The architecture's core patterns are sound. Of 12 predicted pressure points, 7 produced zero friction. The frictions that emerged are operational tooling gaps (observability instrumentation, log automation, operator discovery) — not architectural failures.

**10 findings total:**
- 0 bugs
- 5 operational fragilities (2 at P1, 3 at P2)
- 2 structural debts (1 at P1, 1 at P2)
- 3 trade-offs (all P3, accepted)

**Recommended next actions:** 2 quick wins (~30 min), 1 targeted instrumentation fix (~3 hours), and 1 design task before next actor expansion. No large-scale refactors warranted.

---

## 2. Frictions and Key Findings

### 2.1 Confirmed Frictions

| ID | Finding | Category | Priority |
|----|---------|----------|----------|
| CF-01 | `/statusz` trackers are aggregate-only — no per-symbol breakdown | Operational fragility | P1 |
| CF-02 | No endpoint to discover active symbols | Operational fragility | P2 |
| CF-03 | Correlation ID propagation is manual, not framework-enforced | Structural debt | P1 |
| CF-04 | No automated error-level log scanning in validation scripts | Operational fragility | P2 |
| CF-05 | No automated memory regression tracking | Operational fragility | P2 |
| CF-06 | No sustained automated validation (30-min watchdog) | Trade-off | P3 |
| CF-07 | Kill switch is global, not per-symbol | Trade-off | P3 |
| CF-08 | Client usecase boilerplate partially deduplicated | Structural debt | P2 |
| CF-09 | RSI warm-up delays full-chain validation (~15 min) | Trade-off | P3 |
| CF-10 | 300s timeframe requires extended wait for materialization | Trade-off | P3 |

### 2.2 Nature of the Frictions

The frictions cluster into two categories:

1. **Observability instrumentation** (CF-01, CF-04, CF-05): The diagnostic foundation works but lacks granularity for multi-symbol debugging. The infrastructure is already in place (`healthz.Tracker` supports custom counters, scripts already have a Phase 8 summary) — the gap is in how actors and scripts use it.

2. **Operator convenience** (CF-02, CF-06): The system is operationally functional but requires the operator to carry context that the system could expose. These are not architectural gaps — they are UX improvements for the operational surface.

The structural debts (CF-03, CF-08) are growth risks, not present-day failures. Current code is correct and consistent; the fragility lies in what happens when new actors or families are added.

---

## 3. Prioritized Friction Matrix

### Tier 1 — Fix Before Next Capability (CC-02)

| ID | Friction | Why Now | Effort |
|----|----------|---------|--------|
| CF-01 | Per-symbol tracker counters | Diagnostic value compounds with each symbol added. Fix is minimal (add symbol to counter key). | ~3 hours |
| CF-03 | Correlation ID design | Design the pattern now; implement when next actor is added. Prevents silent propagation breakage. | ~2 hours (design only) |

### Tier 2 — Quick Wins (Do Immediately)

| ID | Friction | Why Now | Effort |
|----|----------|---------|--------|
| CF-04 | Error log grep in script | Single line of shell. Detects silent domain errors. | ~15 min |
| CF-05 | Docker stats snapshot in script | Single line of shell. Detects memory regressions. | ~15 min |

### Tier 3 — Opportunistic (Next Touch)

| ID | Friction | When | Effort |
|----|----------|------|--------|
| CF-02 | List active symbols endpoint | When touching configctl routes | ~1 hour |
| CF-08 | Migrate domain clients to shared usecase | When adding a new domain family | ~1 hour |

### Tier 4 — Accepted Trade-offs (No Action)

| ID | Friction | Rationale |
|----|----------|-----------|
| CF-06 | No watchdog script | Manual monitoring sufficient at N=2 |
| CF-07 | Global kill switch | Per-symbol control is a separate capability |
| CF-09 | RSI warm-up delay | Inherent to indicator math |
| CF-10 | 300s timeframe wait | 60s timeframe provides sufficient validation |

---

## 4. Trade-offs Accepted

| Trade-off | Why Accepted | Revisit Condition |
|-----------|-------------|-------------------|
| Global kill switch (CF-07) | Paper-only execution; halting both symbols simultaneously is safe. | If live venue adapter is activated or if operator explicitly needs per-symbol control. |
| RSI warm-up period (CF-09) | Mathematical requirement of the indicator (14 candles). Cannot be shortened without changing the signal definition. | Never — inherent to RSI. |
| No automated watchdog (CF-06) | Manual `make live-multi-check` at intervals is sufficient for 2 symbols over 30 minutes. | If soak testing at N>5 symbols or 24-hour duration becomes a goal. |
| 300s timeframe wait (CF-10) | The 60s timeframe validates the pipeline completely. 300s is supplementary. | Never — inherent to timeframe design. |
| Client boilerplate (CF-08) | 6 packages × ~30 LOC each. Code is correct, just duplicated. Migration is mechanical but has no urgency. | When adding a new domain family (natural trigger). |

---

## 5. Items That Did Not Confirm as Problems

These were actively monitored during CC-01 validation and produced no evidence of friction:

| Predicted Issue | S119 Reference | Why It Didn't Materialize |
|----------------|---------------|--------------------------|
| Cross-symbol state contamination | DT-1 | Composite key design (`source.symbol.timeframe`) provides reliable isolation. Smoke test cross-symbol checks pass consistently. |
| WebSocket goroutine leak | WS-1 | Per-binding goroutine lifecycle is correctly scoped. No goroutine growth over 30 minutes. |
| Actor mailbox backpressure | DT-2 | Proto.Actor handles 2× throughput without observable delay. Tracker idle times remain low. |
| KV write contention | KV-2 | NATS JetStream KV handles concurrent writes from both symbols without errors. Zero error_count on all store trackers. |
| Staleness guard false positives | EX-1 | 120s staleness window absorbs normal network jitter. No false rejections observed. |
| Projection actor inconsistency | F05 (S110) | Fixed since S110. All 7 projection actors now have consistent `received` counters, `checkStatsInvariant()`, and logger initialization. |
| Signal publisher correlation_id gap | F06 (S110) | Fixed since S110. All 4 publisher actors now log `correlation_id` on publish errors. |

**Takeaway:** The architecture's core patterns — key-based isolation, subject-based partitioning, per-key actor state, composite KV keys — handle multi-symbol load without structural strain. The design decisions made in S96–S118 are validated by real operational evidence.

---

## 6. Preparation for S123

### 6.1 Recommended S123 Scope: Targeted Observability Refinements

S122 establishes that the next stage should address the small, well-bounded frictions that have the highest diagnostic payoff — not open a new capability or refactoring wave.

**Recommended scope:**

1. **Implement CF-04 + CF-05** (quick wins): Add error log grep and docker stats snapshot to activation script Phase 8.

2. **Implement CF-01** (per-symbol tracker counters): Add symbol dimension to tracker counter keys in projection and publisher actors. This is the single highest-value instrumentation improvement.

3. **Design CF-03** (correlation ID injection pattern): Produce a design document for context-based or middleware-based correlation ID propagation. Do not implement yet — the design informs how new actors should be written.

4. **Optionally implement CF-02** (list active symbols): If touching configctl routes for any other reason, add the endpoint.

### 6.2 What S123 Should NOT Do

- Do not open CC-02 (next capability) until observability refinements are in place.
- Do not refactor client usecase boilerplate as a standalone task.
- Do not redesign the correlation ID system — design it, then implement when the next actor is written.
- Do not build soak test infrastructure — manual monitoring at N=2 is sufficient.

### 6.3 Expected Outcome of S123

After S123, the platform should have:
- Per-symbol diagnostic visibility via `/statusz`
- Automated error and memory regression detection in validation scripts
- A documented correlation ID injection pattern for future actor development
- A clean base for CC-02 (next controlled capability)

---

## 7. Files Produced

| File | Purpose |
|------|---------|
| `docs/architecture/controlled-capability-01-frictions-and-structural-findings.md` | Detailed friction capture with evidence, classification, and recommendations |
| `docs/architecture/controlled-capability-01-prioritized-friction-matrix.md` | Decision-support matrix with fix/defer/accept framework |
| `docs/stages/stage-s122-capability-driven-friction-capture-report.md` | This report |

---

## 8. Acceptance Criteria Verification

| Criterion | Met? |
|-----------|------|
| Frictions captured with evidence from real operation | **Yes** — all findings reference specific code paths, S121 validation results, or script behavior |
| Bugs, debts, and trade-offs clearly distinguished | **Yes** — 0 bugs, 5 operational fragilities, 2 structural debts, 3 trade-offs |
| Prioritization useful for decision-making | **Yes** — 4-tier matrix with effort estimates and trigger conditions |
| Base ready for small, well-justified refactors | **Yes** — 2 quick wins + 1 targeted fix scoped for S123 |
| Next stage guided by concrete pain, not hypothesis | **Yes** — CF-01 and CF-03 are the only items worth investing in before CC-02 |

---

## 9. Guard Rail Compliance

| Guard Rail | Compliance |
|-----------|-----------|
| No friction treated as automatic refactor justification | **Compliant** — each finding has explicit fix/defer/accept recommendation with rationale |
| No new product wave opened | **Compliant** — S123 recommendation is observability refinement, not capability |
| Intentional limitations not confused with failures | **Compliant** — CF-07 (global kill switch), CF-09 (RSI warm-up), CF-10 (300s wait) classified as trade-offs |
| No vague or unsubstantiated frictions | **Compliant** — every finding references code paths, S121 evidence, or script inspection |
| Focus on real operational impact | **Compliant** — 7 non-confirmed items documented as validation of sound architecture |
