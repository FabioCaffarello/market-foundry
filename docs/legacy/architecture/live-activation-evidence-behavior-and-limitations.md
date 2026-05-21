# Live Activation Evidence, Behavior, and Limitations

> S341 — Documents what the controlled activation verification proves, how the system behaves under gate transitions, and what remains unproven.

## Evidence Summary

### What Is Proven

| Claim | Evidence Source | Mechanism |
|-------|----------------|-----------|
| Halted gate blocks execution on real path | CAV-1, CAV-4/phase-1 | skipped_halt counter increments, filled=0 |
| Active gate enables execution on real path | CAV-2, CAV-4/phase-2 | VenueOrderFilledEvent received on NATS |
| Runtime halt transition blocks subsequent events | CAV-3, CAV-4/phase-3 | filled count unchanged after halt |
| Full lifecycle (halted→live→halted) works on single supervisor | CAV-4 | Three phases, counters consistent |
| Audit fields round-trip through NATS KV | CAV-5 | reason, updated_by, updated_at preserved |
| Activation surface composes correctly from NATS KV gate | CAV-5 | effective mode matches truth table |
| Kill switch is checked per-intent, not per-session | CAV-3 | Same supervisor, different outcome after gate change |
| Health counters accurately reflect gate decisions | CAV-1 through CAV-4 | processed, filled, skipped_halt consistent |

### Observed Runtime Behavior

**Gate transition latency**: gate changes propagate through NATS KV within ~200ms. The SafetyGate reads the current gate value on every intent via `GateChecker.Check()`. There is no caching or polling interval — each intent queries the KV store.

**Per-intent evaluation**: the kill switch is evaluated for every incoming intent. An intent arriving 1ms after a gate halt will be blocked. There is no batch window or deferred evaluation.

**No drain semantics**: when the gate transitions to halted, any intent currently executing `SubmitOrder` will complete. Only subsequent intents are blocked. This is consistent with the fail-open design documented in S339.

**Counter consistency**: the `processed` counter always increments (event reached actor). The `filled` and `skipped_halt` counters partition the processed events: `processed = filled + skipped_halt + skipped_stale + errors`.

**Fill event correlation**: fill events preserve the source event's CorrelationID and set CausationID to the source event's Metadata.ID, creating a complete audit chain.

## Behavioral Boundaries

### In-Flight Intent During Halt

If an intent is between the SafetyGate check and SubmitOrder completion when the gate transitions to halted:

- The in-flight intent **will complete** (order submitted, fill published)
- The next intent **will be blocked**
- This is by design: the gate does not cancel in-progress work

### KV Propagation Window

Between a KV write and the next intent's gate check, there is a small window (~1-10ms) where the old gate value may be read. In practice, this is negligible because:

- Intent arrival rate is low (market signals, not high-frequency)
- The gate is a safety mechanism, not a precision tool

### Paper Adapter Scope

All S341 tests use the paper adapter (`PaperVenueAdapter`). This proves the activation surface controls the flow correctly but does not exercise real venue HTTP calls. The paper adapter simulates fills synchronously, so there is no venue-side latency or failure.

## Limitations

### Not Proven by S341

| Limitation | Why | Impact |
|-----------|-----|--------|
| Real venue HTTP path not exercised | Paper adapter used for safety | Fill simulation differs from real venue response parsing |
| Binary restart with adapter config change not tested | Requires process lifecycle orchestration | Rollback from venue→paper proven only at domain level (AC-3) |
| Multi-venue activation not tested | Single global gate design | Gate controls all venues identically |
| Credential rotation not tested | Credentials immutable per process | Credential change requires binary restart |
| HTTP control surface not tested in integration | Smoke script tests HTTP; integration tests use NATS KV directly | Two separate paths to the same KV store |
| Automatic rollback not implemented | Design decision: manual operator control only | Rollback requires explicit operator action |
| Extended observation window not tested | Tests run in seconds, not hours | Long-running behavior (memory, counter overflow) unverified |
| ClickHouse persistence of activation events not tested | Activation is control-plane, not data-plane | No fill events land in ClickHouse during halt |

### Accepted Risks

**Fail-open on KV unavailability**: if NATS KV becomes unreachable, the SafetyGate defaults to allowing execution. This is documented in S339 and accepted because NATS unavailability also means no events arrive (the consumer uses the same NATS connection).

**Single global gate**: all venues share one EXECUTION_CONTROL KV key. Per-venue or per-symbol gating requires future work.

## Operational Readiness Assessment

| Capability | Status | Evidence |
|-----------|--------|----------|
| Deploy with gate halted | Ready | CAV-1, CAV-4/phase-1 |
| Enable via gate open | Ready | CAV-2, CAV-4/phase-2 |
| Emergency halt | Ready | CAV-3, CAV-4/phase-3 |
| Audit trail | Ready | CAV-5 |
| Rollback to paper | Domain-proven only | AC-3 (requires binary restart) |
| Multi-venue gating | Not available | Single global gate |
| Automatic rollback | Not available | Design decision |

## Relationship to Rollout Phases

Per the S338 activation policy:

| Phase | Gate State | S341 Coverage |
|-------|-----------|---------------|
| Phase 0: Halted Activation | halted | CAV-1, CAV-4/phase-1 — **fully verified** |
| Phase 1: Single-Order | active | CAV-2, CAV-4/phase-2 — **flow verified** (single intent) |
| Phase 2: Observation Window | active | Not tested — requires sustained operation |

S341 proves Phase 0 and Phase 1 readiness. Phase 2 requires real venue deployment with extended monitoring.
