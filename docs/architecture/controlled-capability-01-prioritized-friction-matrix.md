# Controlled Capability 01 — Prioritized Friction Matrix

> Stage S122 — Decision-support matrix for capability-driven friction capture.
> Date: 2026-03-19

---

## Purpose

This matrix consolidates all frictions identified during CC-01 (multi-symbol live monitoring) validation, prioritized by operational impact, recurrence probability, fix cost, and risk of deferral. It serves as the primary input for deciding which refactors — if any — are worth executing before the next capability wave.

---

## Priority Criteria

| Factor | Weight | How Assessed |
|--------|--------|-------------|
| **Operational impact** | High | Does this friction block or degrade daily operation under multi-symbol load? |
| **Recurrence** | High | Will this friction recur or amplify with each new symbol, family, or runtime? |
| **Fix cost** | Medium | How much effort is required to resolve? Does the fix touch safety-critical paths? |
| **Risk of deferral** | Medium | What happens if we leave this for 2+ more stages? |

---

## Friction Matrix — Actionable Items

### Tier 1: High Impact, Worth Addressing Before Next Capability

| ID | Friction | Category | Impact | Recurrence | Fix Cost | Deferral Risk | Recommendation |
|----|----------|----------|--------|------------|----------|---------------|----------------|
| CF-01 | No per-symbol tracker breakdown in `/statusz` | Oper. fragility | **High** — cannot diagnose per-symbol stalls from diagnostics alone | **Scales with N** — each new symbol dilutes aggregate counts further | **Low** — actors already call `tracker.Counter(name)`; add symbol dimension to counter key | **Medium** — deferred means every debugging session at N>2 symbols requires log correlation | **Fix before CC-02.** Minimal change, high diagnostic payoff. |
| CF-03 | Correlation ID propagation is manual | Structural debt | **High** — cross-runtime debugging requires timestamp matching across 6 services × N symbols | **Scales with actors** — every new actor must remember to manually copy correlation ID | **Medium** — requires either context injection pattern or publish middleware | **Low-Medium** — current actors are consistent; risk is in new actor additions | **Design solution, implement in next wave.** Current state works; fragility is in growth, not in present behavior. |

### Tier 2: Moderate Impact, Address Opportunistically

| ID | Friction | Category | Impact | Recurrence | Fix Cost | Deferral Risk | Recommendation |
|----|----------|----------|--------|------------|----------|---------------|----------------|
| CF-02 | No list-active-symbols endpoint | Oper. fragility | **Medium** — operator must know symbols in advance | **Grows with symbols** — more symbols = harder to track what's active | **Low** — thin query over active config, one route handler | **Low** — workaround exists (query active config) | **Add when convenient.** Not blocking, but improves operator UX. |
| CF-04 | No automated error log scanning | Oper. fragility | **Medium** — domain errors go undetected without manual log review | **Constant** — same gap regardless of symbol count | **Very Low** — single grep command in activation script | **Low** — manual review covers today | **Quick win.** Add to activation script Phase 8 in next touch. |
| CF-05 | No automated memory regression tracking | Oper. fragility | **Medium** — memory leaks under doubled load go undetected | **Constant** — same gap regardless of symbol count | **Very Low** — docker stats snapshot in script | **Low** — manual check covers today | **Quick win.** Add alongside CF-04. |
| CF-08 | Client usecase boilerplate partially deduplicated | Structural debt | **Low-Medium** — copy-paste still required for new domain clients | **Grows with families** — each new family adds ~30 LOC of boilerplate | **Very Low** — mechanical type alias migration | **Low** — current code works correctly | **Do when adding a new domain family.** Not worth a dedicated refactor. |

### Tier 3: Accepted Trade-offs (No Action Required)

| ID | Friction | Category | Impact | Recurrence | Recommendation |
|----|----------|----------|--------|------------|----------------|
| CF-06 | No sustained automated validation (watchdog) | Trade-off | **Low** — manual monitoring is sufficient at N=2 | Grows at N>5 | Accept. Revisit when sustained operation at scale becomes a goal. |
| CF-07 | Kill switch is global, not per-symbol | Trade-off | **Low** — both symbols stop together | Constant | Accept by design. Per-symbol control is a separate capability if needed. |
| CF-09 | RSI warm-up delays full-chain validation | Trade-off | **Low** — inherent to RSI indicator math | Constant | Accept. Documented in runbook. |
| CF-10 | 300s timeframe requires extended wait | Trade-off | **Minimal** — 60s timeframe provides sufficient validation | Constant | Accept. No action needed. |

---

## Non-Friction Items (Predicted But Not Confirmed)

These pressure points from S119 were systematically tracked but did not materialize:

| Pressure Point | Why It Didn't Manifest |
|---------------|----------------------|
| Cross-symbol state contamination (DT-1) | Composite key design (`source.symbol.timeframe`) isolates correctly |
| WebSocket goroutine leak (WS-1) | Per-binding goroutine lifecycle is properly scoped |
| Actor mailbox backpressure (DT-2) | Proto.Actor handles 2× throughput without observable delay |
| KV write contention (KV-2) | NATS handles concurrent writes from both symbols without errors |
| Staleness guard false positives (EX-1) | 120s staleness window is sufficiently generous |
| Projection actor inconsistency (F05 from S110) | Resolved — all 7 actors are now consistent |
| Publisher correlation_id gap (F06 from S110) | Resolved — all 4 publishers now log correlation_id |

**Significance:** 7 of 12 predicted pressure points produced zero friction. This confirms that the architecture's core patterns (key-based isolation, subject-based partitioning, per-key actor state) are sound under doubled load. The frictions that did emerge are operational tooling gaps, not architectural failures.

---

## Decision Framework: Fix vs Defer vs Accept

```
                        ┌─────────────────────────────────┐
                        │  Does it block daily operation?  │
                        └──────────┬──────────────────────┘
                             YES   │   NO
                              ▼    │    ▼
                          ┌───┴────┴────────────────────┐
                          │  Does it scale with symbols? │
                          └──────┬──────────────────────┘
                           YES   │   NO
                            ▼    │    ▼
                     Fix before   │  Quick win?
                     next wave    │    ▼
                                  │  YES → Do it now
                                  │  NO  → Defer
```

Applying this framework:
- **CF-01** (per-symbol trackers): blocks diagnosis → scales → **Fix before CC-02**
- **CF-03** (correlation ID): blocks debugging → scales → **Design now, implement in wave**
- **CF-04, CF-05** (log/memory scanning): doesn't block → doesn't scale → quick win → **Do it now**
- **CF-02** (list symbols): doesn't block → scales → not quick → **Defer**
- **CF-08** (client boilerplate): doesn't block → scales → **Do when adding a family**
- **CF-06, CF-07, CF-09, CF-10**: trade-offs → **Accept**

---

## Recommended Action Sequence

| Order | ID | Action | Effort | Gate |
|-------|----|--------|--------|------|
| 1 | CF-04 | Add `level=error` grep to activation script Phase 8 | 15 min | None |
| 2 | CF-05 | Add `docker stats` snapshot to activation script Phase 8 | 15 min | None |
| 3 | CF-01 | Add per-symbol counter keys to projection and publisher actors | 2-3 hours | Before CC-02 |
| 4 | CF-03 | Design correlation ID injection pattern (context or middleware) | 1-2 hours design | Before actor expansion |
| 5 | CF-02 | Add `GET /configctl/active-symbols` endpoint | 1 hour | Opportunistic |
| 6 | CF-08 | Migrate remaining domain clients to shared usecase types | 1 hour | When adding a family |

**Total estimated effort for items 1-3:** ~3 hours — bounded, well-justified, high-payoff refactors.
