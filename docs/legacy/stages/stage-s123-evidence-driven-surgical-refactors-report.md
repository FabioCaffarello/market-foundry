# Stage S123 — Evidence-Driven Surgical Refactors Report

> **Status:** Complete
> **Capability:** CC-01 — Multi-Symbol Live Monitoring (post-capability refinement)
> **Scope:** Targeted refactors anchored in S122 friction evidence
> **Predecessor:** S122 (Capability-Driven Friction Capture)

---

## 1. Executive Summary

S123 executed 3 surgical refactors and produced 1 design document, strictly selected from the S122 prioritized friction matrix. Total code delta: ~25 lines across 14 files. No new abstractions, no horizontal refactoring, no behavioral changes.

**Refactors executed:**
1. Per-symbol tracker counters across all 14 actors with trackers (CF-01)
2. Automated error-level log scanning in activation script (CF-04)
3. Automated memory usage snapshot in activation script (CF-05)

**Design produced:**
1. Correlation ID injection pattern sketch for future actor additions (CF-03)

**Items deferred:** CF-02 (active symbols endpoint), CF-08 (client boilerplate), CF-03 implementation — each has a documented natural trigger for revisitation.

**Trade-offs accepted:** CF-06, CF-07, CF-09, CF-10 — unchanged from S122 classification.

---

## 2. Refactors Chosen and Justification

### R1: Per-Symbol Tracker Counters — CF-01 (P1)

| Attribute | Value |
|-----------|-------|
| S122 priority | P1 — highest operational impact |
| Category | Operational fragility → Observability instrumentation |
| Justification | Cannot diagnose per-symbol stalls from `/statusz` alone. Impact scales with each symbol added. |
| Fix cost | Very low — 1 line per tracker call site, uses existing `Counter()` API |
| Structural benefit | Operators can answer "is ethusdt flowing?" from `/statusz` without log correlation |

**Actors instrumented:** 7 projection actors (store), 5 publisher actors (derive), 1 publisher actor (ingest), 1 venue adapter (execute) = 14 actors total.

### R2: Automated Error Log Scanning — CF-04 (P2)

| Attribute | Value |
|-----------|-------|
| S122 priority | P2 — quick win |
| Category | Operational fragility → Error detection automation |
| Justification | Domain-level errors go undetected without manual log review. Script may report "all healthy" while errors accumulate. |
| Fix cost | 5 lines of shell |
| Structural benefit | Automated error detection in every pipeline validation run |

### R3: Automated Memory Snapshot — CF-05 (P2)

| Attribute | Value |
|-----------|-------|
| S122 priority | P2 — quick win |
| Category | Operational fragility → Memory regression detection |
| Justification | Memory leaks under doubled multi-symbol load go undetected without manual `docker stats` checks. |
| Fix cost | 4 lines of shell (alongside R2) |
| Structural benefit | Memory baseline captured in every pipeline validation run |

### D1: Correlation ID Injection Pattern — CF-03 (P1, design only)

| Attribute | Value |
|-----------|-------|
| S122 priority | P1 — scales with actor count |
| Category | Structural debt → Cross-runtime debugging |
| Justification | Manual correlation ID copy in every actor is fragile at growth. Design now, implement when validated by first consumer. |
| Deliverable | Pattern sketch in `evidence-driven-surgical-refactors-after-capability-01.md` |
| Why design-only | No new actors being added in this stage; implementing without a consumer risks wrong abstraction |

---

## 3. Files Changed

### Code Changes

| File | Change | CF |
|------|--------|-----|
| `internal/actors/scopes/store/candle_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/store/signal_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/store/decision_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/store/strategy_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/store/risk_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/store/trade_burst_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/store/volume_projection_actor.go` | +1 line: `materialized:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/derive/publisher_actor.go` | +3 lines: `published:SYMBOL` counters (candle, trade_burst, volume) | CF-01 |
| `internal/actors/scopes/derive/signal_publisher_actor.go` | +1 line: `published:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/derive/decision_publisher_actor.go` | +1 line: `published:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/derive/strategy_publisher_actor.go` | +1 line: `published:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/derive/risk_publisher_actor.go` | +1 line: `published:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/ingest/publisher_actor.go` | +1 line: `published:SYMBOL` counter | CF-01 |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | +2 lines: `processed:SYMBOL`, `filled:SYMBOL` counters | CF-01 |
| `scripts/live-pipeline-activate.sh` | +9 lines: error log grep + memory snapshot in Phase 8 | CF-04, CF-05 |

### Documents Produced

| File | Purpose |
|------|---------|
| `docs/architecture/evidence-driven-surgical-refactors-after-capability-01.md` | Detailed record of executed refactors and design |
| `docs/architecture/refactors-deferred-after-capability-01.md` | Deferred items with rationale and triggers |
| `docs/stages/stage-s123-evidence-driven-surgical-refactors-report.md` | This report |

---

## 4. Structural Gains Obtained

| Gain | Scope | Measurement |
|------|-------|-------------|
| Per-symbol diagnostic visibility | All 6 runtimes (ingest, derive, store, execute) | `/statusz` counters now show per-symbol breakdown |
| Automated error detection | Activation script | `level=error` grep catches domain errors automatically |
| Memory baseline capture | Activation script | `docker stats` snapshot on every validation run |
| Correlation ID design ready | Architecture | Pattern sketch available for next actor addition |

**What the operator gains:** After S123, running `make live-multi-check` or the activation script answers three questions that previously required manual log correlation:
1. "Is ethusdt flowing through store?" → Check `materialized:ethusdt` counter
2. "Are there hidden domain errors?" → Phase 8 error count
3. "Is memory growing?" → Phase 8 memory snapshot

---

## 5. Items Deferred and Why

| ID | Item | Why Deferred | Natural Trigger |
|----|------|-------------|----------------|
| CF-02 | Active symbols endpoint | Workaround exists; medium impact at N=2 | When touching configctl routes |
| CF-08 | Client usecase boilerplate | Code correct, duplication mechanical | When adding a new domain family |
| CF-03 impl | Correlation ID middleware | No consumer to validate the API shape | When adding a new actor |
| CF-06 | Automated watchdog | Manual monitoring sufficient at N=2 | N>5 symbols or 24h soak |
| CF-07 | Per-symbol kill switch | Paper-only; global halt is safe | Live venue adapter activation |
| CF-09 | RSI warm-up | Mathematical requirement | Never |
| CF-10 | 300s timeframe wait | 60s provides sufficient validation | Never |

Full rationale: `docs/architecture/refactors-deferred-after-capability-01.md`

---

## 6. Acceptance Criteria Verification

| Criterion | Met? |
|-----------|------|
| Refactors have explicit and strong justification | **Yes** — each maps to a specific CF-ID from S122 |
| Refactoring remains small and focused | **Yes** — ~25 lines across 14 files, no new packages |
| Structural gains are real and localized | **Yes** — per-symbol counters in existing tracker API, script additions |
| Excessive abstraction is avoided | **Yes** — no new types, no middleware, no framework |
| Monorepo is more robust for next wave | **Yes** — diagnostic surface ready for CC-02 |

---

## 7. Guard Rail Compliance

| Guard Rail | Compliance |
|-----------|-----------|
| No new horizontal refactoring opened | **Compliant** — changes are localized to tracker call sites and script |
| No "while we're here" improvements | **Compliant** — only S122-justified changes executed |
| No aesthetic-motivated refactors | **Compliant** — every change addresses a diagnosed operational friction |
| No new framework/pattern without proven pain | **Compliant** — correlation ID design is design-only; implementation deferred to first consumer |
| Deferred items documented | **Compliant** — full rationale in dedicated document |

---

## 8. Preparation for S124

S123 leaves the platform in a clean state for the next wave. Recommended considerations for S124:

1. **CC-02 definition** — the diagnostic surface is now ready for the next controlled capability. Per-symbol counters provide the observability foundation needed to validate a new capability under multi-symbol load.

2. **Correlation ID middleware** — if CC-02 introduces a new actor, implement the middleware pattern from D1 as part of that actor's development. This validates the design with a real consumer.

3. **Active symbols endpoint (CF-02)** — if CC-02 requires configctl route changes, add the endpoint opportunistically.

4. **Client boilerplate (CF-08)** — if CC-02 introduces a new domain family, migrate existing clients to shared usecase types as part of the family addition.

The next stage should be a capability definition (CC-02), not another refactoring stage. The architecture is clean enough to add value; further structural work without a capability driver risks over-engineering.
