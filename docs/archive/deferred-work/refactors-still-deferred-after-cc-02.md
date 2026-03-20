# Refactors Still Deferred After CC-02

## Context

This document records frictions and debts that remain deferred after S129. Each entry explains why the trigger was not met or why deferral is the correct evidence-based decision.

---

## CF-03 (Actor Layer): Correlation ID in Actor Message Passing

**Pattern:** Every actor manually copies `events.NewMetadata().WithCorrelationID(msg.CorrelationID)`.

**Trigger condition:** N=3 signal families (bundled with CF-08 generic actor).

**Why deferred:**
- The HTTP layer was addressed (S129-R1), but actor-layer centralization requires a generic actor framework change.
- At N=2 families, N=10 actors, manual propagation has produced zero incidents.
- Centralizing at the actor level should be bundled with CF-08 (generic `SignalSamplerActor`) at CC-03, where the shared lifecycle already provides a natural injection point.

**Revisit:** CC-03 (third signal family).

---

## CF-08: Actor Boilerplate (Generic SignalSamplerActor)

**Pattern:** `EMACrossoverSignalSamplerActor` (97 lines) is ~95% identical to `RSISignalSamplerActor`. Copy-paste with only constructor, sampler type, and actor name differing.

**Trigger condition:** N=3 signal families.

**Current state:** N=2. CC-02 added the second instance. Copy-paste was mechanical and error-free.

**Why deferred:**
- At N=2, the duplication is tolerable and the pattern is clear.
- Extracting a generic `SignalSamplerActor[T]` at N=2 would be premature — only two data points to generalize from.
- At N=3, the pattern is definitively stable and the extraction eliminates ~97 lines per additional family.

**Estimated effort at N=3:** ~2 hours.

**Revisit:** CC-03 (third signal family).

---

## CF-11: NATS Registry Switch Proliferation

**Pattern:** Each new signal family requires 4 manual touch points across NATS adapter files:
- `signal_registry.go` — EventSpec + ControlSpec + consumer function (~30 lines)
- `signal_publisher.go` — `specForType()` switch case (~3 lines)
- `signal_publisher.go` — `LatestSpecByType()` switch case (~3 lines)
- `signal_kv_store.go` — bucket constant (~1 line)

**Trigger condition:** N=3 signal families.

**Current state:** N=2. Switch statements have `"rsi"` and `"ema_crossover"` cases.

**Why deferred:**
- Scatter across files is the friction, not line count. At N=2, the dispatch is manageable.
- A map-based registry replacing switch statements would centralize 4 touch points into 1 registration site.
- The refactor is straightforward but not justified at N=2 where there have been zero wiring errors.

**Estimated effort at N=3:** ~1-2 hours.

**Revisit:** CC-03 (third signal family).

---

## CF-12: Store Pipeline Boilerplate

**Pattern:** Each signal family requires ~25 lines in `store_supervisor.go` `declarePipelines()`.

**Trigger condition:** N=5 signal families.

**Current state:** N=2. The pipeline struct is declarative and self-documenting.

**Why deferred:**
- The Pipeline struct is already a reasonable abstraction.
- At N=2, entries are easy to review and the pattern is clear.
- Further reduction (code generation, catalog-driven entries) is premature.

**Revisit:** N=5 families.

---

## CF-02: Active Symbols Endpoint

**Pattern:** No dedicated endpoint for listing active symbols; workaround requires parsing active config response.

**Trigger condition:** When touching configctl routes for another reason, OR N>5 symbols.

**Current state:** CC-02 did not modify configctl routes. Symbol count = 2.

**Why deferred:** Neither trigger condition was met. The workaround is adequate at current scale.

**Revisit:** Next configctl route change or N>5 symbols.

---

## CF-13: Per-Family Algorithm Configuration

**Pattern:** Signal algorithm parameters (RSI period, EMA fast/slow periods) are hardcoded in sampler constructors.

**Trigger condition:** When A/B testing or per-binding parameter tuning is needed.

**Current state:** Hardcoded values are correct for current use cases. No operational need for runtime tuning.

**Why deferred:** Intentional limitation. Configuration surface should be added when there's a concrete use case, not speculatively.

**Revisit:** A/B testing or multi-parameter optimization requirements.

---

## D4: Composition Root Unit Tests

**Trigger condition:** When a wiring error reaches live validation that should have been caught earlier.

**Current state:** CC-02 modified composition roots in `cmd/derive/run.go` and `cmd/store/run.go` without any wiring errors. Smoke tests and live validation caught everything.

**Why deferred:** Integration and smoke tests provide adequate coverage. No wiring error has escaped to production.

**Revisit:** First wiring error that reaches live validation.

---

## D5: Failure Recovery Validation

**Trigger condition:** Before any production-grade deployment.

**Why deferred:** System operates in paper-trading mode only. Production deployment is not near-term.

**Revisit:** Before production deployment.

---

## D6: Soak Testing Infrastructure

**Trigger condition:** N>5 symbols or 24-hour continuous operation.

**Current state:** N=2 symbols with manual validation at intervals.

**Why deferred:** Current scale does not require automated soak testing.

**Revisit:** N>5 symbols or 24h operation target.

---

## Summary Matrix

| ID | Friction | Trigger Threshold | Current State | Deferred Until |
|----|----------|-------------------|---------------|----------------|
| CF-03 (actor) | Actor correlation propagation | N=3 families | N=2 | CC-03 |
| CF-08 | Actor boilerplate | N=3 families | N=2 | CC-03 |
| CF-11 | NATS registry switches | N=3 families | N=2 | CC-03 |
| CF-12 | Store pipeline boilerplate | N=5 families | N=2 | N=5 |
| CF-02 | Active symbols endpoint | Route change or N>5 symbols | N=2 | Opportunistic |
| CF-13 | Per-family config | A/B testing need | Hardcoded | Demand-driven |
| D4 | Composition root tests | Wiring error escape | Zero errors | Incident-driven |
| D5 | Failure recovery | Production deployment | Paper-trading | Pre-production |
| D6 | Soak testing | N>5 symbols or 24h | N=2, manual | Scale-driven |

**Key insight:** Three frictions (CF-08, CF-11, CF-03-actor) converge at the N=3 signal family threshold. CC-03 is the natural bundling point for these refactors, estimated at ~5-7 hours total.
