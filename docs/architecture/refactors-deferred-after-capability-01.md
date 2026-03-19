# Refactors Deferred After Capability 01

> Stage S123 — Items consciously left for later, with rationale.
> Date: 2026-03-19

---

## Purpose

Not every friction identified in S122 warrants immediate action. This document records what was deferred and why, so that future stages can make informed decisions about when (or whether) to address each item.

---

## Deferred Items

### CF-02: List Active Symbols Endpoint

**Friction:** No endpoint to discover which symbols are currently active. Operator must know in advance.

**Why deferred:** The workaround exists (query active config and parse bindings). Impact is medium at N=2 symbols. The fix is a thin route handler over existing data — not architecturally complex, but not blocking anything.

**Trigger to revisit:** When touching configctl routes for another reason, or when N>5 symbols makes manual tracking burdensome.

**Estimated effort when addressed:** ~1 hour.

---

### CF-08: Client UseCase Boilerplate Migration

**Friction:** 6 domain client packages still hand-write the same `struct + Execute` pattern (~30 LOC each) instead of using the `shared/usecase` type aliases that `configctlclient` already uses.

**Why deferred:** Current code is correct and consistent. The duplication is mechanical, not behavioral. Migrating now provides zero diagnostic or operational benefit — it's a maintenance-cost reduction that only pays off when a new domain family is added.

**Trigger to revisit:** When adding a new domain family (natural trigger for the migration).

**Estimated effort when addressed:** ~1 hour (mechanical type alias migration).

---

### CF-03: Correlation ID Injection Implementation

**Friction:** Correlation ID propagation is manual. Each actor must remember to copy the ID.

**Why deferred (implementation, not design):** A design sketch was produced in S123 (see `evidence-driven-surgical-refactors-after-capability-01.md`, D1). However, implementing a publish middleware now has no consumer to validate the API shape. Implementing prematurely risks designing the wrong abstraction.

**Trigger to revisit:** When the first new actor is added (CC-02 or equivalent). The new actor should be the first consumer of the middleware pattern, validating the API in practice.

**Estimated effort when addressed:** ~2-3 hours (middleware implementation + migration of first actor).

---

## Accepted Trade-offs (No Action Planned)

These were classified as trade-offs in S122 and remain accepted:

| ID | Friction | Rationale | Revisit Condition |
|----|----------|-----------|-------------------|
| CF-06 | No sustained automated validation (watchdog) | Manual monitoring is sufficient at N=2 symbols. | N>5 symbols or 24-hour soak testing. |
| CF-07 | Kill switch is global, not per-symbol | Paper-only execution; halting both symbols simultaneously is safe. | Live venue adapter activation or explicit operator need. |
| CF-09 | RSI warm-up delays full-chain validation (~15 min) | Inherent to RSI indicator math (14 candles). Cannot be shortened. | Never — mathematical requirement. |
| CF-10 | 300s timeframe requires extended wait for materialization | 60s timeframe provides sufficient validation signal. | Never — inherent to timeframe design. |

---

## Decision Principle

Every item above was evaluated against the same framework used in S122:

1. **Does it block daily operation?** → No for all deferred items.
2. **Does it scale with symbols/actors?** → CF-02 and CF-08 scale mildly; CF-03 scales with actor count.
3. **Is it a quick win?** → No — each requires at least 1 hour of focused work.
4. **Is there a natural trigger?** → Yes for CF-02 (route changes), CF-08 (new family), CF-03 (new actor).

Items with natural triggers are best addressed at those triggers, not as standalone refactoring tasks. This avoids unnecessary disruption and ensures the fix is validated by immediate use.
