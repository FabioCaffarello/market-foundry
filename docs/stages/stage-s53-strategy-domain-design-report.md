# Stage S53 — Strategy Domain Design Report

> Formal design of the `strategy` domain for Market Foundry.
> Design only — no implementation code produced.

**Date:** 2026-03-17
**Status:** COMPLETE
**Type:** Design
**Predecessor:** S52 (Strategy Readiness Rerun)

---

## 1. Executive Summary

S52 declared the foundation CONDITIONALLY READY for strategy domain design. S53 delivers the canonical design of `strategy` as a standalone bounded context in the Market Foundry mesh. The design establishes strategy as the fifth domain layer — consuming decisions and producing directional trade intents — while preserving strict boundaries with decision, signal, evidence, store, and gateway. One initial family (`mean_reversion_entry`) is specified. No implementation code was written.

---

## 2. Key Design Decisions

### D-1: Strategy is a separate bounded context

Strategy is NOT an extension of decision. It has its own domain package (`internal/domain/strategy/`), its own stream (`STRATEGY_EVENTS`), its own KV buckets, its own event types, and its own query surface. The separation is enforced by the same boundary invariants used for signal and decision.

**Why:** Decision says "condition met". Strategy says "therefore propose trade X". These are fundamentally different concerns. Merging them would create a domain that both evaluates conditions and proposes actions, violating single responsibility.

### D-2: Strategy lives in the derive binary

Strategy resolvers run inside the derive binary's SourceScopeActor, consuming decisions via local actor messages. No separate binary is needed.

**Why:** Strategy consumes decisions that are already computed locally. A separate binary would add operational overhead and cross-process latency for no architectural benefit. This follows the same rationale that placed decision in derive (not a separate binary).

**Reconsidered when:** Strategy logic becomes stateful enough (portfolio-aware, multi-symbol) to warrant its own process. Current design does not permit this — strategy is per-symbol, per-timeframe, stateless per evaluation.

### D-3: First family is `mean_reversion_entry`

The initial strategy family consumes a single decision (`rsi_oversold`) and produces a directional trade intent (`long` when triggered, `flat` otherwise).

**Why:** Single-decision, simplest possible resolution logic. Proves the entire pipeline end-to-end with minimal risk. Follows the same pattern: candle was the first evidence, RSI the first signal, rsi_oversold the first decision.

### D-4: Strategy introduces `Direction` as the distinguishing domain concept

Where decision has `Outcome` (triggered/not_triggered/insufficient), strategy has `Direction` (long/short/flat). This is the core semantic boundary between the two domains.

**Why:** Direction is position-aware. Outcome is position-agnostic. This distinction ensures strategy cannot be confused with decision — they operate on fundamentally different abstractions.

### D-5: `flat` is a valid direction, not an error

When no decision triggers or data is insufficient, strategy resolves to `flat` with `confidence: "0.0"`. This is a legitimate strategy output that means "no trade recommended".

**Why:** Downstream consumers (future risk domain) need to distinguish between "no data yet" (404) and "data processed, no trade recommended" (flat). Omitting flat would create ambiguity.

### D-6: Event verb is `resolved`, not `generated` or `evaluated`

Strategy events use `strategy_resolved` as the event name, distinguishing from signal (`signal_generated`) and decision (`decision_evaluated`).

**Why:** "Resolved" communicates that strategy has reached a determination — including `flat` as a valid determination. "Generated" implies creation of something new. "Evaluated" implies assessment. "Resolved" implies a conclusion.

### D-7: No implicit activation chains

Activating `mean_reversion_entry` in `strategy_families` does NOT auto-activate `rsi_oversold` in `decision_families`. The operator is responsible for the full dependency chain.

**Why:** Implicit chains create hidden coupling and make debugging harder. Explicit configuration has been the foundational principle since evidence activation. raccoon-cli will warn (not error) when dependencies appear incomplete.

### D-8: Latest-only projections in Phase 1

Strategy starts with `STRATEGY_{TYPE}_LATEST` only. No history bucket.

**Why:** Every prior domain proved that latest-first is correct. No concrete consumer requires strategy history yet. History adds complexity that should wait until the domain is stable.

---

## 3. Deliverables

| Deliverable | Path |
|---|---|
| Domain design | [strategy-domain-design.md](../architecture/strategy-domain-design.md) |
| Stream families | [strategy-stream-families.md](../architecture/strategy-stream-families.md) |
| Activation & ownership | [strategy-activation-and-ownership.md](../architecture/strategy-activation-and-ownership.md) |
| Query surface guidelines | [strategy-query-surface-guidelines.md](../architecture/strategy-query-surface-guidelines.md) |
| Stage report | This document |

---

## 4. Domain Position in the Mesh

```
observation → evidence → signal → decision → strategy → [risk → execution → portfolio]
    (ingest)   (derive)  (derive)  (derive)   (derive)    ← all in derive binary
```

Strategy is the **last purely analytical layer**. Everything after strategy crosses the "action boundary" — risk evaluates against real portfolio state, execution interacts with exchanges.

### Full Dependency Chain

```
OBSERVATION_EVENTS → EVIDENCE_EVENTS → SIGNAL_EVENTS → DECISION_EVENTS → STRATEGY_EVENTS
     (ingest)           (derive)         (derive)         (derive)          (derive)
```

Each stream is independently owned. Each domain is independently testable. No circular dependencies.

---

## 5. Boundaries Summary

| Boundary | Strategy Side | Other Side | Separation Mechanism |
|---|---|---|---|
| Strategy ↔ Decision | Consumes decision outcomes via actor messages | Decision unaware of strategy | SBI-1, SBI-2, SBI-7, SBI-10 |
| Strategy ↔ Signal | Never consumes signals directly | Signal unaware of strategy | SBI-10, OI-6 |
| Strategy ↔ Evidence | Never consumes evidence directly | Evidence unaware of strategy | SBI-1, OI-6 |
| Strategy ↔ Store | Store projects strategy events to KV | Strategy never writes to KV | SBI-5, OI-3 |
| Strategy ↔ Gateway | Gateway serves strategy HTTP routes | Gateway stateless, no domain logic | SBI-6, OI-5 |
| Strategy ↔ Risk | Not implemented — strategy is the boundary | Risk will consume strategy (future) | Documented as out-of-scope |
| Strategy ↔ Execution | Not implemented | Execution is downstream of risk | Documented as out-of-scope |

---

## 6. What Was Intentionally Deferred

| Item | Why Deferred | Target |
|---|---|---|
| Strategy implementation code | S53 is design-only | S54 |
| raccoon-cli strategy governance rules | Hard prerequisite for S54, not for design | S54 (P-7) |
| `strategy_families` in settings schema | Implementation detail | S54 (P-6) |
| Strategy history projections | No consumer requires them yet | S55+ |
| Multi-decision strategies | Single-decision must prove first | S55+ |
| Strategy cooldown/debounce | Enhancement after base pattern proven | S55+ |
| MACD Momentum Entry family (STF-02) | Depends on macd_crossover decision | S55+ |
| Confluence Entry family (STF-03) | Multi-decision pattern not yet designed | S56+ |
| Risk domain design | Requires operational strategy layer | S56+ |
| Execution domain design | Requires operational risk layer | Indefinite |
| Portfolio domain design | Requires operational execution layer | Indefinite |

---

## 7. Tensions Identified

### T-1: Derive binary scope growth

Strategy adds one more actor type to the derive binary (StrategyResolverActor + StrategyPublisherActor). The derive binary now hosts: evidence samplers, signal samplers, decision evaluators, and strategy resolvers.

**Assessment:** The actor model isolates concerns. Each actor has its own lifecycle. No shared mutable state. This is manageable today. If a sixth layer is added to derive, binary splitting should be actively evaluated.

**Registered in:** SR-3 in [strategy-risks-and-blockers-rerun.md](../architecture/strategy-risks-and-blockers-rerun.md).

### T-2: Chain depth (5 layers)

The full chain `observation → evidence → signal → decision → strategy` is 5 layers deep. A failure at any layer cascades upward.

**Assessment:** Each layer has independent health tracking. Strategy resolvers handle `insufficient` and `not_triggered` outcomes gracefully (emit `flat`). No chain explosion risk because each layer filters and reduces volume (many observations → fewer evidence → fewer signals → fewer decisions → fewer strategies).

### T-3: Store actor count growth

Store now has consumer + projection pairs for: candle, tradeburst, volume, signal, decision, and (soon) strategy. That's 12 consumer+projection actors plus QueryResponderActor.

**Assessment:** All actors are lightweight and I/O-bound (JetStream read → KV write). No CPU-intensive work. Manageable for the current single-instance deployment model.

---

## 8. Preparation for S54 and S55

### S54 — Strategy Implementation (recommended scope)

**Hard prerequisites (must complete before any strategy code):**
1. P-6: Add `strategy_families` to `internal/shared/settings/schema.go`
2. P-7: Add raccoon-cli strategy drift rules (SD-1 through SD-5) and guardrails
3. P-7: Add strategy as sensitive area in raccoon-cli coverage map

**Implementation scope:**
1. `internal/domain/strategy/` — Strategy type, Direction, Validate(), events
2. `internal/application/strategy/` — mean_reversion_entry resolver (pure logic, table-driven tests)
3. `internal/actors/scopes/derive/strategy_resolver_actor.go` — actor wiring
4. `internal/actors/scopes/derive/strategy_publisher_actor.go` — publisher actor
5. `internal/actors/scopes/store/strategy_consumer_actor.go` — consumer
6. `internal/actors/scopes/store/strategy_projection_actor.go` — projection with 3 gates
7. `internal/adapters/nats/strategy_*` — registry, publisher, consumer, KV store
8. `internal/application/strategyclient/` — query use case
9. `internal/interfaces/http/handlers/strategy.go` — HTTP handler
10. `internal/interfaces/http/routes/strategy.go` — route registration
11. Deploy config updates (`derive.jsonc`, `store.jsonc`, `gateway.jsonc`)
12. Stream and KV bucket creation in startup initialization

**Acceptance criteria for S54:**
- `mean_reversion_entry` strategy resolves from live `rsi_oversold` decisions
- Strategy event published to `STRATEGY_EVENTS`
- Strategy projected to `STRATEGY_MEAN_REVERSION_ENTRY_LATEST`
- Strategy queryable via `GET /strategy/mean_reversion_entry/latest`
- raccoon-cli strategy governance rules active and passing
- All domain, application, adapter, and handler tests passing

### S55 — Strategy Hardening (recommended scope)

1. Strategy projection actor tests (following S51 pattern)
2. Strategy history projections (if needed by a concrete consumer)
3. Multi-decision strategy design (confluence pattern)
4. Strategy cooldown/debounce design
5. Second strategy family (macd_momentum_entry, if DF-02 is ready)

---

## 9. Stage Metrics

| Metric | Value |
|---|---|
| Design documents produced | 4 |
| Implementation code written | 0 |
| Families designed | 1 (mean_reversion_entry) + 2 deferred |
| Boundary invariants defined | 10 (SBI) + 9 (OI) |
| Activation preconditions defined | 10 (AP) |
| Query invariants defined | 9 (QI) |
| Family invariants defined | 9 (FI) |
| Ownership rules defined | 7 (OR) |
| Items explicitly deferred | 11 |
| Tensions identified | 3 |
| Recommendation | Proceed to S54 after P-6 and P-7 are met |

---

## 10. Verdict

Strategy domain design is **complete**. The domain is formally defined as a separate bounded context with clear boundaries, stream families, activation model, projection/query implications, and invariants. The design follows the proven pattern established by signal (S35) and decision (S42).

**Next step:** S54 — Strategy implementation, gated on P-6 (config schema) and P-7 (governance infrastructure).
