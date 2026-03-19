# Refactors Deferred After Vertical Slice 01

> Stage S111 — Items consciously not refactored, with rationale.
> Date: 2026-03-19

---

## Deferred Items

### D1: Execute Actor Unit Tests (F07, P0)

**S110 Finding:** The venue adapter actor contains kill switch gate check, staleness guard, and submit timeout — all safety-critical logic with zero unit tests.

**Why deferred:** This is a testing gap, not a refactoring opportunity. Writing tests for the execute actor requires either:
- Extracting the gate/staleness/timeout logic into testable functions (a refactor), or
- Building actor-level test harnesses with mock KV stores.

Both approaches need design thought beyond what a targeted refactor stage should contain. The risk is real, but the mitigation is test authoring, not structural change.

**Recommendation:** Prioritize in S112 as a standalone testing initiative. Consider extracting gate check logic into a pure function first, then writing table-driven tests.

---

### D2: Publisher Actor Generic Extraction (F06 full, P2)

**S110 Finding:** 5 publisher actors share near-identical `Receive()` implementations. The only differences are domain type names and message struct types.

**Why deferred:** The signal publisher `correlation_id` fix (the most impactful part of F06) was applied in R1. Extracting a generic publisher would:
- Require parameterizing the actor over domain event types and publisher types
- Introduce Go generics into the actor layer, which currently uses a straightforward switch-based dispatch
- Provide marginal maintenance savings (5 files, each <90 LOC) versus the complexity of a generic actor

The duplication is manageable at current scale. The risk of a generic actor obscuring the simple message-type switch is higher than the cost of the current copy-paste approach.

**Revisit when:** A 6th publisher actor is needed, or a cross-cutting change (e.g., changing the timeout from 5s to configurable) requires touching all 5 files.

---

### D3: Query Client UseCase Generics (F04 extension)

**S110 Finding:** The other client packages (evidenceclient, signalclient, decisionclient, strategyclient, riskclient, executionclient) also follow a boilerplate pattern, but each has unique inline validation logic.

**Why deferred:** Unlike configctlclient where `Normalize()`/`Validate()` is on the command type itself, query clients embed validation inline — field-level checks that differ per query type. A generic would need a validator function parameter, which trades one form of boilerplate for another (function literals instead of files). The net gain is small.

The `usecase.GatewayUseCase` generic created in R4 is available for future query clients that need only nil-check + delegate. Clients with custom validation logic are better served by explicit code.

**Revisit when:** A query client package exceeds 5 use case files, or a pattern emerges where query validation can be expressed declaratively.

---

### D4: Ingest Actor Tests (F08, P2)

**S110 Finding:** The ingest supervisor (611 LOC) has no unit tests. Dynamic exchange scope creation involves 3 actor hops.

**Why deferred:** Similar to D1 — this is a test coverage gap, not a structural debt. The ingest actor's dynamic scope creation is complex but correct (validated by the vertical slice). Testing it requires mock WebSocket connections and multi-hop actor harnesses.

**Recommendation:** Address alongside D1 in a dedicated testing stage.

---

### D5: Configctl Actor Tests (F09, P2)

**S110 Finding:** The configctl actor scope (612 LOC) has no unit tests for control routing dispatch or request/reply error mapping.

**Why deferred:** The configctl actor scope is indirectly tested via `configctl_gateway_test.go` at the adapter layer. The gap is in actor-specific routing logic, which is boilerplate dispatch. Adding direct tests would be valuable but is low risk relative to D1 (execute actor).

**Recommendation:** Address after D1.

---

### D6: Route Registration Abstraction (F10, P3)

**S110 Verdict:** Acceptable trade-off at current scale (7 families).

**Why deferred:** The explicit route registration pattern is readable and self-documenting. Each family file is <50 lines. An abstraction (e.g., a generic route builder) would save ~10 lines per family but obscure the explicit routing.

**Revisit when:** Family count exceeds 12.

---

### D7: Gateway Wiring DRY (F11, P3)

**S110 Verdict:** Acceptable trade-off. Each connection is 2-3 lines of explicit setup.

**Why deferred:** The explicit wiring in `buildGatewayConns()` serves as documentation. A loop-based approach would save ~10 lines but require a registry of gateway factory functions, adding indirection.

**Revisit when:** Gateway count exceeds 12 or a new pattern (e.g., gateway health checks) needs to be applied uniformly.

---

### D8: Derive-Configctl Dependency Model (F12, P3)

**S110 Verdict:** Correct behavior — not a problem.

**Why not addressed:** The eventual consistency model between derive and configctl is intentional. Derive starts idle and catches up when configctl publishes `IngestionRuntimeChangedEvent`. Hard-depending on configctl for startup would be wrong.

---

## Summary

| ID | Finding | Priority | Status | Rationale |
|----|---------|----------|--------|-----------|
| D1 | Execute actor tests | P0 | **DEFERRED** | Testing gap, not refactoring — needs design for actor test harness |
| D2 | Publisher actor generic | P2 | **DEFERRED** | Marginal gain vs. generic actor complexity; signal correlation_id fixed |
| D3 | Query client generics | P1 ext | **DEFERRED** | Unique validation per query; GatewayUseCase available for simple cases |
| D4 | Ingest actor tests | P2 | **DEFERRED** | Testing gap — needs mock WebSocket harness |
| D5 | Configctl actor tests | P2 | **DEFERRED** | Indirectly tested; lower risk than D1 |
| D6 | Route registration | P3 | **ACCEPTED** | Manageable at 7 families |
| D7 | Gateway wiring DRY | P3 | **ACCEPTED** | Explicit wiring serves as documentation |
| D8 | Derive-configctl dep | P3 | **ACCEPTED** | Correct behavior by design |
