# Kill-Switch Live and Canonical smoke-live-stack

> S335 — Architecture document for kill-switch live validation and
> `make smoke-live-stack` canonicalization.

## Context

The Live Stack Integration Wave (S332–S336) proves the execution pipeline
end-to-end against live NATS and ClickHouse infrastructure. Stages S333 and
S334 proved the NATS consumer→actor flow and the fill round-trip with
composite visibility. S335 completes the control-path proof and promotes
`make smoke-live-stack` as the canonical smoke surface for the wave.

## Kill-Switch Architecture

### Domain Model

The kill-switch is a `ControlGate` with two states:

| State | Semantics |
|-------|-----------|
| `active` | Execution pipeline publishes and submits normally |
| `halted` | All publish and submit paths are blocked immediately |

Authority: NATS KV bucket `EXECUTION_CONTROL`, key `global`.

### Dual Checkpoint Pattern

The gate is checked at two independent points in the pipeline:

1. **Derive-side publisher** (`ExecutionPublisherActor`): checks gate before
   publishing `PaperOrderSubmittedEvent` to the `EXECUTION_EVENTS` stream.
   Halted gate → event discarded, `execution:gate_halted` counter incremented.

2. **Execute-side venue adapter** (`VenueAdapterActor` via `SafetyGate`):
   checks gate before submitting to venue. Halted gate → intent blocked,
   `skipped_halt` counter incremented.

Both checkpoints are fail-open: if the KV store is unreachable or the
checker is nil, the gate defaults to active. This is intentional — a
transient NATS failure should not halt the entire pipeline.

### Control Surface

| Verb | Path | Effect |
|------|------|--------|
| `GET` | `/execution/control` | Query current gate state |
| `PUT` | `/execution/control` | Set gate to `active` or `halted` |

Request body for PUT:
```json
{
  "status": "halted",
  "reason": "risk-limit-breach",
  "updated_by": "oncall-operator"
}
```

Audit fields (`reason`, `updated_by`, `updated_at`) survive the full
round-trip through gateway → NATS request/reply → store → KV bucket.

### Propagation Semantics

- **Atomic:** KV Put is atomic; all readers see the new state within
  milliseconds.
- **No drain:** Halt is immediate. In-flight intents that already passed
  the gate check may still reach the venue. Intents arriving after the
  halt are dropped.
- **No queue:** Halted intents are discarded, not queued for later replay.

## Canonical Smoke Surface

`make smoke-live-stack` is the canonical single-command proof for the Live
Stack Integration Wave. It validates seven phases:

| Phase | Validates |
|-------|-----------|
| 1. Stack Readiness | ClickHouse, writer, gateway, NATS health |
| 2. NATS Streams | Stream existence and consumer registration |
| 3. ClickHouse Data | Row presence in all 6 analytical tables |
| 4. Composite HTTP | `/chains`, `/funnel`, `/dispositions` → 200 |
| 5. Single-Family | All 6 analytical family endpoints → 200 |
| 6. Structural Gate | S317 Go test regression suite |
| 7. Kill-Switch | Halt → confirm → resume → confirm cycle |

### Kill-Switch Smoke (Phase 7)

Phase 7 exercises the full control path against live infrastructure:

1. Query current gate state → must return 200
2. Set gate to `halted` with reason and operator fields
3. Confirm gate reads as `halted` (read-after-write consistency)
4. Resume gate to `active`
5. Confirm gate reads as `active`
6. Verify audit fields survive the round-trip

Safety: the script installs an EXIT trap that always restores the gate to
`active`, even if the script fails mid-phase or is interrupted.

## Scope Boundaries

What S335 proves:
- Kill-switch control path works end-to-end via HTTP → NATS → KV → HTTP
- Gate state transitions are consistent and auditable
- `smoke-live-stack` is a complete, reproducible wave surface

What S335 does NOT prove:
- Real-time blocking of in-flight venue submissions (requires concurrent
  load, which is a production concern beyond wave scope)
- KV Watch-based reactive propagation (current design uses poll-on-read)
- Multi-gate per execution type (current design uses a single global gate)
- Performance under load or latency SLAs

## Invariants Preserved

All 9 Production Wiring Tranche invariants remain valid:

- EC-1, EC-3: Execution event contracts and correlation
- F-1, F-4: Fill event contracts and venue column alignment
- RF-1: Round-trip fill visibility
- PGR-08: Paper gate registration
- INV-REC-1: Reconciliation invariant
- INV-RC-1: Retry/circuit-breaker invariant
- INV-OBS-1: Observability invariant

## Related Documents

- [Live Stack Integration Wave Charter](live-stack-integration-wave-charter-and-scope-freeze.md)
- [NATS Consumer to Actor Live Flow](nats-consumer-to-actor-live-flow.md)
- [Fill Event Round-Trip and Composite Visibility](fill-event-round-trip-and-composite-visibility.md)
- [Live Control Path Smoke Usage and Operational Limitations](live-control-path-smoke-usage-and-operational-limitations.md)
