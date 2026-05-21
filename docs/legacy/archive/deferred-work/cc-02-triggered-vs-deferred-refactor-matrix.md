# CC-02 Triggered vs Deferred Refactor Matrix

> Explicit assessment of whether deferred debts from S124 were triggered by CC-02, with evidence and recommended next action.

## 1. Purpose

S124 (Post-Capability Readiness Review) deferred 6 debts (D1–D6) and 10 friction items (CF-01 through CF-10) with documented triggers. CC-02 was expected to activate some of these triggers. This matrix records what actually happened.

## 2. Trigger Assessment Matrix

### CF-03 — Correlation ID Middleware

| Field | Value |
|-------|-------|
| **Original trigger** | "When the first new actor is added (CC-02 or equivalent)" |
| **Was trigger met?** | **YES** — CC-02 added `EMACrossoverSignalSamplerActor` |
| **Was action taken?** | **NO** — middleware was not implemented |
| **Evidence** | `ema_crossover_signal_sampler_actor.go` copies `events.NewMetadata().WithCorrelationID(msg.CorrelationID)` manually, identical to all other actors |
| **Incidents from deferral** | None. No correlation chain breaks observed. |
| **Revised assessment** | Trigger fired but impact was zero. The manual pattern is consistent across all actors. Middleware remains a growth-protection measure, not a fix for current breakage. |
| **Recommendation** | **Implement opportunistically at CC-03.** Do not block next capability on this. Design sketch from S123 is ready. Estimated effort: 2–3 hours. |
| **Priority** | P2 (was P1 in S124; downgraded because trigger fired without incident) |

---

### CF-08 — Actor Boilerplate / Client UseCase Boilerplate

| Field | Value |
|-------|-------|
| **Original trigger** | Actor boilerplate: "Trigger at 3+ families." Client boilerplate: "When adding a new domain family." |
| **Was trigger met?** | **PARTIALLY** — Actor boilerplate is at N=2 (below threshold). Client boilerplate trigger was not met (CC-02 adds a family within signal domain, not a new domain). |
| **Was action taken?** | No (correctly — threshold not reached) |
| **Evidence** | `ema_crossover_signal_sampler_actor.go` is 97 lines, ~95% identical to RSI actor. 6 domain client packages still use hand-written patterns. |
| **Incidents from deferral** | None. Copy-paste was mechanical and error-free. |
| **Revised assessment** | At N=2, the boilerplate is tolerable and correct. The third signal family (CC-03 or equivalent) will definitively trigger actor boilerplate reduction. Client boilerplate remains correctly deferred. |
| **Recommendation** | **Actor boilerplate: implement generic `SignalSamplerActor` at N=3.** Client boilerplate: continue deferring until new domain family. |
| **Priority** | P2 (unchanged) |

---

### CF-02 — Active Symbols Endpoint

| Field | Value |
|-------|-------|
| **Original trigger** | "When touching configctl routes for another reason, or when N>5 symbols" |
| **Was trigger met?** | **NO** — CC-02 did not touch configctl routes. Symbol count remains at N=2. |
| **Was action taken?** | No (correctly) |
| **Evidence** | No configctl route changes in CC-02 diff. Workaround (parsing active config response) remains adequate. |
| **Revised assessment** | Trigger conditions remain unmet. Correctly deferred. |
| **Recommendation** | **Continue deferring.** Revisit when configctl routes are modified for another reason or symbol count exceeds 5. |
| **Priority** | P3 (unchanged) |

---

### D4 — Composition Root Tests

| Field | Value |
|-------|-------|
| **Original trigger** | "When adding a new runtime or refactoring composition roots" |
| **Was trigger met?** | **NO** — CC-02 modified existing composition roots but did not add new runtimes. |
| **Was action taken?** | No |
| **Evidence** | `cmd/derive/run.go` and `cmd/store/run.go` were modified (new processor/pipeline entries) but no wiring errors occurred. Smoke tests and live validation caught no issues. |
| **Incidents from deferral** | None. Wiring correctness confirmed by live pipeline validation. |
| **Revised assessment** | Existing integration and smoke tests provide adequate coverage. Composition root tests remain a nice-to-have. |
| **Recommendation** | **Continue deferring.** Add only if a wiring error reaches live validation that should have been caught earlier. |
| **Priority** | P3 (unchanged) |

---

### D5 — Failure Recovery Validation

| Field | Value |
|-------|-------|
| **Original trigger** | "Before any production-grade deployment" |
| **Was trigger met?** | **NO** — No production deployment planned or executed. |
| **Was action taken?** | No (correctly) |
| **Revised assessment** | Trigger condition remains unmet. Correctly deferred. |
| **Recommendation** | **Continue deferring.** Required before production, not before next capability. |
| **Priority** | P2 (unchanged) |

---

### D6 — Soak Testing Infrastructure

| Field | Value |
|-------|-------|
| **Original trigger** | "When operating at N>5 symbols or 24-hour duration" |
| **Was trigger met?** | **NO** — CC-02 operates at N=2 symbols with manual validation. |
| **Was action taken?** | No (correctly) |
| **Revised assessment** | Trigger condition remains unmet. Correctly deferred. |
| **Recommendation** | **Continue deferring.** Not near-term. |
| **Priority** | P3 (unchanged) |

---

### CF-01 — Per-Symbol Tracker Breakdown

| Field | Value |
|-------|-------|
| **Status** | **RESOLVED** in S123 (R1 refactor). Not revisited by CC-02. |
| **CC-02 impact** | EMA Crossover actors automatically benefit from per-symbol counters via shared tracker infrastructure. Confirmed working. |

---

### CF-04 — Automated Error-Level Log Scanning

| Field | Value |
|-------|-------|
| **Status** | **RESOLVED** in S123 (R2 refactor). Not revisited by CC-02. |
| **CC-02 impact** | Log scanning script automatically covers ema_crossover logs. Confirmed working. |

---

### CF-05 — Automated Memory Regression Tracking

| Field | Value |
|-------|-------|
| **Status** | **RESOLVED** in S123 (R3 refactor). Not revisited by CC-02. |
| **CC-02 impact** | Memory snapshots automatically cover ema_crossover containers. Confirmed working. |

---

### CF-06, CF-07, CF-09, CF-10

| Item | Status | CC-02 Impact |
|------|--------|-------------|
| CF-06 (No sustained watchdog) | ACCEPTED trade-off | Not impacted by CC-02 |
| CF-07 (Global kill switch) | ACCEPTED trade-off | Not impacted by CC-02 |
| CF-09 (RSI warm-up delay) | ACCEPTED (mathematical invariant) | EMA has its own warm-up (21 candles), same category |
| CF-10 (300s timeframe wait) | ACCEPTED (design choice) | Not impacted by CC-02 |

## 3. Summary Table

| Debt ID | Trigger Met? | Action Taken? | Incident? | Revised Priority | Next Action |
|---------|-------------|--------------|-----------|-----------------|-------------|
| **CF-03** | **YES** | No | None | P2 (↓ from P1) | Implement at CC-03 |
| **CF-08** (actor) | Partial (N=2 < 3) | No | None | P2 | Implement at N=3 |
| **CF-08** (client) | No | No | None | P2 | Defer to new domain |
| CF-02 | No | No | None | P3 | Continue deferring |
| D4 | No | No | None | P3 | Continue deferring |
| D5 | No | No | None | P2 | Before production |
| D6 | No | No | None | P3 | Continue deferring |
| CF-01 | Resolved | — | — | — | Done |
| CF-04 | Resolved | — | — | — | Done |
| CF-05 | Resolved | — | — | — | Done |
| CF-06/07/09/10 | Accepted | — | — | — | No action |

## 4. Key Insight

**Only CF-03 had its trigger definitively met by CC-02, and the non-action produced zero incidents.** This validates the deferral strategy: the triggers were set at the right thresholds, and the debts are genuinely non-blocking at current scale.

The most important upcoming trigger is **CF-08 actor boilerplate at N=3 signal families**. If CC-03 adds a third signal family, both CF-03 (correlation ID middleware) and CF-08 (generic sampler actor) should be addressed as part of that implementation — not as a separate refactoring stage.

## 5. New Frictions Discovered by CC-02

CC-02 also revealed frictions not previously cataloged:

| ID | Friction | Classification | Threshold |
|----|----------|---------------|-----------|
| CF-11 | NATS registry switch proliferation (4 touch points per family) | Structural debt | Implement map-based registry at N=3 |
| CF-12 | Store pipeline boilerplate (~25 lines per family in `declarePipelines()`) | Acceptable boilerplate | Evaluate at N=5 |
| CF-13 | No per-family algorithm configuration path (hardcoded periods) | Intentional limitation | Implement when A/B testing or per-binding tuning is needed |

These are documented for future tracking but do not require immediate action.
