# Derive Actor Confidence Rules

## Purpose

Codify the invariants and testing boundaries for derive actors — the most critical processing pipeline in market-foundry. These rules ensure that adding new families or domains does not silently break existing behavior.

## Testing Pattern

Derive actors are tested using Hollywood engine-based tests with message collectors. This pattern:
- Spawns a real Hollywood engine (no mocks of the actor framework)
- Uses `msgCollector` actors as stand-ins for publisher and scope PIDs
- Sends messages to the actor under test and verifies what it forwards
- Tests the full actor lifecycle: `Started` → message processing → output routing

### Why not mock-based like store actors?

Store actors call mockable interfaces (`store.Put()`, `store.PutHistory()`). Derive actors call `c.Send()` on `actor.Context`, which requires a live engine. The engine-based approach tests the actual dispatch path.

## Invariants

### Evidence Samplers (candle, tradeburst, volume)

1. **No publish before window transition** — Trades within the same time window accumulate but do not emit events.
2. **Finalization on window boundary** — The first trade in a new window finalizes the previous window and publishes an event.
3. **Domain validation** — Every finalized event passes its domain `Validate()` before publish.
4. **Nil ScopePID safety** — Actors with `ScopePID=nil` publish to the evidence publisher without panic.
5. **Correlation ID propagation** — The triggering trade's correlation ID flows through to both the publish message and the fan-out message.

### Signal Samplers (RSI)

6. **Warm-up period silence** — No signals are emitted until `period+1` candle closes are received (14+1=15 for RSI).
7. **Post-warmup production** — Every candle close after warm-up produces a signal.
8. **Fan-out to scope** — `signalGeneratedMessage` is sent to `ScopePID` with primitive data (no `Signal` struct — DBI-9 compliance).

### Decision Evaluators (RSI Oversold)

9. **Threshold boundary** — RSI < 30 → `triggered`; RSI >= 30 → `not_triggered`.
10. **Invalid input resilience** — Non-numeric signal values are silently dropped (no panic, no publish).
11. **Independent evaluation** — Each signal is evaluated independently; no state bleed between evaluations.

### Strategy Resolvers (Mean Reversion Entry)

12. **Outcome → Direction mapping** — `triggered` → `long`; `not_triggered` → `flat`; `insufficient` → `flat` with reason metadata.
13. **Unknown outcome rejection** — Unrecognized outcomes produce no strategy (silent drop).
14. **Parameter attachment** — Only `triggered` strategies carry `entry`, `target_offset`, `stop_offset` parameters.

## Isolation Boundaries

- **Per-actor sampler ownership** — Each actor owns its own sampler/evaluator/resolver instance. No shared state between actors.
- **Symbol isolation** — Separate actor instances per symbol; trades for symbol A never affect symbol B's actor.
- **Family isolation** — Evidence actors do not influence signal/decision/strategy actors except through explicit fan-out messages via the scope PID.

## Remaining Gaps

- **Publisher actors** (evidence, signal, decision, strategy) — Require NATS connection; not unit-testable without infrastructure mocks.
- **SourceScopeActor** — Spawns publisher actors on startup; full isolation test requires NATS or publisher interface extraction.
- **DeriveSupervisor** — Spawns entire tree; integration-level test only.
- **ConsumerActor / BindingWatcherActor** — NATS-dependent infrastructure actors.
