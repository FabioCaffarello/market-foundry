# Live Stack Integration Wave — Charter and Scope Freeze

> Opened by S332 · Predecessor: Production Wiring Tranche (S327–S331, FULL CLOSURE)

## 1. Wave Identity

| Field | Value |
|-------|-------|
| Wave name | Live Stack Integration |
| Opened by | S332 |
| Predecessor | Production Wiring Tranche (S327–S331) |
| Predecessor verdict | FULL CLOSURE — all 4 charter items delivered, 9 invariants held, 202 tests, zero regressions |
| Wave purpose | Prove the composed venue execution pipeline against live infrastructure (NATS, KV, stack) |
| Scope status | **FROZEN** as of S332 |

## 2. Strategic Context

The Production Wiring Tranche proved that all execution components — RetrySubmitter,
Post200Reconciler, observability hooks, venue error classification, and kill-switch —
are correctly composed at the Go test level. S331 closed with an explicit recommendation:

> "The evidence profile has shifted from 'components not composed' (closed) to
> 'composition not exercised against live infrastructure' (next frontier)."

This wave captures that frontier. It does **not** add new domain capabilities. It proves
that what is already built works end-to-end against running infrastructure.

## 3. Wave Capability Target

**Systemic capability to prove:** The composed venue execution pipeline receives orders
from NATS, submits them through the decorated chain, publishes fill events back to NATS,
and respects the kill-switch — all against a running local stack.

## 4. Charter Blocks

### Block LSI-1: NATS Consumer Flow Live

**Goal:** Prove that the NATS durable consumer delivers `PaperOrderSubmittedEvent`
messages to `VenueAdapterActor.onIntent()` in a running stack.

**Scope:**
- Execute supervisor starts consumer with durable spec against live NATS
- Consumer receives published event and delivers to actor
- Health tracker records delivery metrics
- Consumer restart preserves durable state (no message loss, no duplicate delivery)

**Evidence required:**
- Stack log showing consumer connection, event delivery, actor invocation
- Health tracker counter incremented after delivery

**Key files:**
- `internal/actors/scopes/execute/execute_supervisor.go` — consumer bootstrap
- `internal/adapters/nats/natsexecution/consumer.go` — durable consumer
- `internal/adapters/nats/natsexecution/registry.go` — stream/consumer registry

---

### Block LSI-2: Fill Event Round-Trip and Composite Visibility

**Goal:** Prove the full fill event path: venue submit → fill publication → NATS stream →
downstream consumer acknowledgement.

**Scope:**
- `VenueAdapterActor` publishes `VenueOrderFilledEvent` after successful submit
- Published event appears on `EXECUTION_FILL_EVENTS` stream with correct subject routing
- Subject pattern matches `execution.fill.venue_market_order.{source}.{symbol}.{timeframe}`
- JSON serialization integrity verified (round-trip decode)
- Composite visibility: gateway can query fill state after persistence

**Evidence required:**
- NATS stream inspection showing published fill event
- Downstream consumer receives and acknowledges event
- Gateway query returns fill data after persistence completes

**Key files:**
- `internal/actors/scopes/execute/venue_adapter_actor.go` — fill publisher call site
- `internal/adapters/nats/natsexecution/publisher.go` — fill publisher
- `internal/adapters/nats/natsexecution/registry.go` — fill stream definition

---

### Block LSI-3: Kill-Switch Live and Smoke-Live-Stack

**Goal:** Prove the two-layer kill-switch architecture works against a real NATS KV
bucket, and canonicalize the live-stack smoke as a reproducible ceremony.

**Scope:**
- Control KV store connects to live NATS KV bucket `EXECUTION_CONTROL`
- Gate state transitions: active → halted → active observable via KV
- Pre-submit safety gate blocks execution when gate is halted
- Retry halt checker stops retry loop when gate transitions to halted mid-retry
- Fail-open behavior confirmed: KV unavailable → gate defaults to active
- `smoke-live-stack` script extended or hardened for kill-switch exercise

**Evidence required:**
- KV bucket inspection showing gate state before and after transition
- Actor log showing blocked submit or halted retry due to gate state
- Smoke script exits cleanly with kill-switch exercise step

**Key files:**
- `internal/adapters/nats/natsexecution/control_kv_store.go` — KV reader
- `internal/adapters/nats/natsexecution/control_gateway.go` — KV writer
- `internal/domain/execution/control.go` — gate domain model
- `scripts/smoke-live-stack.sh` — smoke script

---

### Block LSI-4: Wave Gate

**Goal:** Evaluate all evidence from LSI-1 through LSI-3 and render a formal
closure verdict for the Live Stack Integration Wave.

**Scope:**
- Evidence matrix covering all three blocks
- Regression gate: existing 202+ tests still pass
- Residual risk register with accepted gaps
- Formal verdict: FULL CLOSURE, PARTIAL CLOSURE, or BLOCKED
- Next-wave recommendation

**Evidence required:**
- Completed evidence matrix
- `make test` green
- `make smoke-composed` green
- `make smoke-live-stack` green

---

## 5. Block Ordering and Stage Mapping

| Order | Block | Proposed Stage | Dependency |
|-------|-------|----------------|------------|
| 1 | LSI-1: Consumer Flow Live | S333 | None — entry point |
| 2 | LSI-2: Fill Round-Trip | S334 | LSI-1 (consumer must be proven before testing full path) |
| 3 | LSI-3: Kill-Switch Live + Smoke | S335 | LSI-1 (KV requires live NATS, reuses consumer proof) |
| 4 | LSI-4: Wave Gate | S336 | LSI-1 + LSI-2 + LSI-3 |

> **Note:** LSI-2 and LSI-3 may be parallelized if LSI-1 is proven early, but the
> default ordering is sequential to minimize risk.

## 6. Invariants Carried Forward

All 9 invariants from the Production Wiring Tranche remain active:

| ID | Invariant | Status |
|----|-----------|--------|
| EC-1 | Deterministic client order ID | Active |
| EC-3 | Per-request deadline enforcement | Active |
| F-1 | No bare errors / Problem type | Active |
| F-4 | Credential redaction | Active |
| RF-1 | Retryable flag accuracy | Active |
| PGR-08 | Intent immutability | Active |
| INV-REC-1 | No duplicate execution | Active |
| INV-RC-1 | Deadline independence | Active |
| INV-OBS-1 | Zero noise on success | Active |

## 7. Accepted Risks Carried Forward

These risks from S322–S328 remain accepted and are **not in scope** for this wave:

| Risk ID | Description | Accepted Since |
|---------|-------------|----------------|
| R-S322-1 | Single recovery attempt (no retry on query) | S322 |
| R-S322-2 | No persistence of ambiguous state | S322 |
| R-S323-1 | Deadline does not cancel in-flight submit | S323 |
| R-S325-2 | No Retry-After header extraction | S325 |
| R-S328-1 | Retry policy not config-driven | S328 |
| R-S328-2 | Reconciliation timeout not config-driven | S328 |

## 8. Success Criteria for Wave Closure

The wave achieves FULL CLOSURE when:

1. NATS consumer delivers events to the actor in a running stack (LSI-1)
2. Fill events complete the round-trip through NATS with correct serialization (LSI-2)
3. Kill-switch operates against live KV with both gate states exercised (LSI-3)
4. `make smoke-live-stack` passes as a reproducible ceremony (LSI-3)
5. All 202+ existing tests remain green (LSI-4)
6. Evidence matrix is complete with no BLOCKED items (LSI-4)

## 9. Scope Freeze Declaration

**This charter is FROZEN.** No additional blocks, capabilities, or infrastructure
may be added to this wave without a formal scope amendment documented in a
subsequent stage report. The wave must close with exactly the four blocks above
or record a PARTIAL CLOSURE explaining which blocks were deferred and why.
