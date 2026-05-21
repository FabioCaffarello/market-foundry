# Stage S43 — Decision First Slice Report

> **Status**: Complete
> **Date**: 2026-03-17
> **Objective**: Implement the first vertical slice of the `decision` domain.

---

## Executive Summary

S43 introduces the `decision` domain as the fourth layer in Market Foundry's progression:
`observation → evidence → signal → decision`. The first decision family — **RSI Oversold (DF-01)** —
evaluates whether the RSI signal indicates an oversold condition (RSI < 30) and produces a categorical
`triggered`/`not_triggered` judgment with graduated confidence.

The slice proves that `decision` is a distinct domain with its own stream, KV buckets, projections,
and query surface — not merely "another rule on top of signal."

All artifacts compile, all new tests pass. The implementation follows the exact same patterns
established by the signal domain (S35-S36), maintaining architectural consistency.

---

## S42 Validation

S42's design document (`decision-domain-design.md`) was used as the canonical source of truth:

- **Domain model**: Implemented exactly as specified (Decision struct, Outcome enum, SignalInput)
- **Binary placement**: Decision lives in derive, as prescribed
- **DBI-9 compliance**: Decision evaluators receive signal data as primitive messages (`signalGeneratedMessage`), not `signal.Signal` structs
- **Separate stream**: `DECISION_EVENTS` is distinct from `SIGNAL_EVENTS`
- **Activation model**: `pipeline.decision_families` (opt-in, independent of `signal_families`)
- **No deviations from S42** — all invariants (DBI-1 through DBI-9, OI-1 through OI-7) are honored

---

## Family Chosen

**RSI Oversold (DF-01)** — simplest possible evaluation:
- Single signal input (RSI)
- Deterministic threshold crossing (< 30.0)
- No warm-up state (evaluates immediately per signal)
- Graduated confidence formula

**Justification**: Validates the entire pipeline with minimal logic. The evaluation itself is trivial;
the value is in proving the domain boundary, stream separation, and projection authority.

---

## Implementation Artifacts

### New Files (25 files)

| Layer | File | Purpose |
|---|---|---|
| **Domain** | `internal/domain/decision/decision.go` | Decision aggregate, Outcome enum, SignalInput, validation |
| **Domain** | `internal/domain/decision/events.go` | DecisionEvaluatedEvent |
| **Domain** | `internal/domain/decision/decision_test.go` | 12 tests: validation, partition keys, isolation |
| **Application** | `internal/application/decision/rsi_oversold_evaluator.go` | Pure evaluator logic |
| **Application** | `internal/application/decision/rsi_oversold_evaluator_test.go` | 7 tests: triggered, not triggered, threshold, extreme, invalid |
| **Application** | `internal/application/decisionclient/contracts.go` | Query/reply contracts |
| **Application** | `internal/application/decisionclient/get_latest_decision.go` | Use case with validation |
| **Application** | `internal/application/decisionclient/get_latest_decision_test.go` | 5 tests: input validation, gateway delegation |
| **Ports** | `internal/application/ports/decision.go` | DecisionGateway interface |
| **Adapters** | `internal/adapters/nats/decision_registry.go` | Stream, subject, consumer contracts |
| **Adapters** | `internal/adapters/nats/decision_publisher.go` | JetStream publisher |
| **Adapters** | `internal/adapters/nats/decision_consumer.go` | Durable JetStream consumer |
| **Adapters** | `internal/adapters/nats/decision_kv_store.go` | KV latest bucket with monotonicity guard |
| **Adapters** | `internal/adapters/nats/decision_gateway.go` | Request/reply gateway for queries |
| **Actors/Derive** | `internal/actors/scopes/derive/decision_evaluator_actor.go` | RSIOversoldEvaluatorActor |
| **Actors/Derive** | `internal/actors/scopes/derive/decision_publisher_actor.go` | DecisionPublisherActor |
| **Actors/Store** | `internal/actors/scopes/store/decision_consumer_actor.go` | DecisionConsumerActor |
| **Actors/Store** | `internal/actors/scopes/store/decision_projection_actor.go` | DecisionProjectionActor (3-gate) |
| **HTTP** | `internal/interfaces/http/handlers/decision.go` | DecisionWebHandler |
| **HTTP** | `internal/interfaces/http/handlers/decision_test.go` | 4 tests: OK, unavailable, missing params, null |
| **HTTP** | `internal/interfaces/http/routes/decision.go` | Decision route registration |
| **HTTP** | `internal/interfaces/http/routes/decision_test.go` | 3 tests: register, include, omit |
| **Docs** | `docs/architecture/decision-first-slice.md` | Implementation reference |
| **Docs** | `docs/architecture/decision-family-01-contracts.md` | DF-01 contract reference |
| **Tests** | `tests/http/decision.http` | Manual HTTP test file |

### Modified Files (13 files)

| File | Change |
|---|---|
| `internal/actors/scopes/derive/messages.go` | Added `signalGeneratedMessage`, `publishDecisionMessage` |
| `internal/actors/scopes/derive/signal_sampler_actor.go` | Added `ScopePID` config, fan-out to scope on signal generation |
| `internal/actors/scopes/derive/source_scope_actor.go` | Added `DecisionFamilyProcessor`, decision publisher, evaluator spawning, signal→decision routing |
| `internal/actors/scopes/derive/derive_supervisor.go` | Added decision registry, decision processor registration |
| `internal/actors/scopes/store/messages.go` | Added `decisionReceivedMessage` |
| `internal/actors/scopes/store/store_supervisor.go` | Added `DecisionPipeline`, decision pipeline registration |
| `internal/actors/scopes/store/query_responder_actor.go` | Added decision KV store, query route |
| `internal/interfaces/http/routes/core.go` | Added `DecisionFamilyDeps`, wired into `DefaultRoutes` |
| `internal/shared/settings/schema.go` | Added `DecisionFamilies`, `IsDecisionFamilyEnabled`, `EnabledDecisionFamilies` |
| `cmd/gateway/gateway.go` | Added `newDecisionGateway` |
| `cmd/gateway/run.go` | Wired decision gateway and use case |
| `cmd/store/run.go` | Added decision tracker registration |
| `deploy/configs/derive.jsonc` | Added `decision_families: ["rsi_oversold"]` |
| `deploy/configs/store.jsonc` | Added `decision_families: ["rsi_oversold"]` |

---

## Design Decisions

### Signal→Decision Fan-out

The signal sampler already sends `publishSignalMessage` directly to the signal publisher.
To feed decision evaluators, we added a `ScopePID` to `SignalSamplerConfig`. When a signal
is generated, the sampler notifies the scope via `signalGeneratedMessage` (primitive data,
not signal.Signal — per DBI-9). The scope routes to matching decision evaluators.

This mirrors the existing candle→signal pattern (`candleFinalizedMessage`).

### Decision Evaluator is Stateless

Unlike signal samplers (which carry warm-up state), the RSI Oversold evaluator is stateless —
it evaluates each signal independently. This simplifies the first slice and matches the
design decision in S42 Section 16 (Q1: start with single-signal families).

---

## Boundary Invariants Verified

| Invariant | Status |
|---|---|
| DBI-1: No imports from signal/evidence/observation in decision domain | Verified |
| DBI-2: Signal data via actor messages only | Verified (signalGeneratedMessage) |
| DBI-3: Publishes only to DECISION_EVENTS | Verified |
| DBI-4: Evaluator is pure (no I/O) | Verified |
| DBI-5: Projections owned by store only | Verified |
| DBI-6: Gateway reads only, no writes/cache/transform | Verified |
| DBI-7: No feedback from decision to signal | Verified |
| DBI-8: Independent config (`decision_families`) | Verified |
| DBI-9: Primitive data in signalGeneratedMessage | Verified |

---

## Test Results

```
internal/domain/decision          — 12 tests PASS
internal/application/decision     — 7 tests PASS
internal/application/decisionclient — 5 tests PASS
internal/interfaces/http/handlers — 4 decision tests PASS (+ existing)
internal/interfaces/http/routes   — 3 decision tests PASS (+ existing)
```

All binaries compile: `cmd/gateway`, `cmd/derive`, `cmd/store`.

---

## Limits Encountered

1. **No actor-level tests**: Decision evaluator and publisher actors have no isolated tests.
   This is a systemic gap shared with signal and evidence actors (noted in S42 risks).

2. **No raccoon-cli decision governance**: Drift detection and guardrails for decision
   domain not yet added to raccoon-cli.

3. **No config validation across families**: The system does not validate that
   `signal_families` includes `rsi` when `decision_families` includes `rsi_oversold`.
   This is an operational dependency, not a code coupling.

---

## Items Explicitly Deferred to S44+

| Item | Rationale |
|---|---|
| Decision history bucket (`DECISION_RSI_OVERSOLD_HISTORY`) | Latest-only is sufficient for first slice |
| Decision history query endpoint | Requires history bucket |
| Multi-signal confluence families (DF-03) | Requires temporal alignment logic |
| MACD crossover family (DF-02) | Requires MACD signal family |
| Raccoon-CLI decision governance rules | Separate tooling stage |
| Decision actor tests | Systemic gap, not decision-specific |
| Config cross-validation (signal→decision dependency) | Operational concern, not blocking |
| Strategy/risk/execution/portfolio domains | Phase 3+ |

---

## References

- [decision-domain-design.md](../architecture/decision-domain-design.md) — S42 canonical design
- [decision-first-slice.md](../architecture/decision-first-slice.md) — Implementation reference
- [decision-family-01-contracts.md](../architecture/decision-family-01-contracts.md) — DF-01 contracts
- [decision-stream-families.md](../architecture/decision-stream-families.md) — Family catalog
- [decision-activation-and-ownership.md](../architecture/decision-activation-and-ownership.md) — Activation model
