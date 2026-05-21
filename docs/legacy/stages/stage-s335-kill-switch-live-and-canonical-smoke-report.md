# S335 — Kill-Switch Live and Canonical Smoke Report

> **Block:** LSI-3 (Live Stack Integration Wave)
> **Charter:** [S332](stage-s332-live-stack-integration-charter-report.md)
> **Predecessor:** [S334](stage-s334-fill-event-round-trip-and-composite-visibility-report.md)
> **Successor:** S336 (Wave Gate — planned)

## Objective

Validate the execution kill-switch control path against live infrastructure
and consolidate `make smoke-live-stack` as the canonical operational proof
surface for the Live Stack Integration Wave (S332–S336).

## Executive Summary

S335 delivers kill-switch live proof via a new Phase 7 in `smoke-live-stack.sh`
that exercises the full halt→confirm→resume→confirm cycle through the gateway
HTTP surface against live NATS KV. The smoke script is promoted from an S318
verification tool to the canonical wave surface, covering data path (Phases
1–6) and control path (Phase 7) in a single reproducible command.

## Deliverables

### Code and Script Changes

| File | Change |
|------|--------|
| `scripts/smoke-live-stack.sh` | Added Phase 7: kill-switch control path validation with safe cleanup trap |
| `scripts/smoke-live-stack.sh` | Updated header, banner, and summary to reflect S335 canonical status |
| `Makefile` | Updated `smoke-live-stack` target description to S335 canonical |
| `docs/stages/INDEX.md` | Added S334 and S335 entries to Phase 32 |

### Architecture Documents

| Document | Purpose |
|----------|---------|
| [`kill-switch-live-and-canonical-smoke-live-stack.md`](../architecture/kill-switch-live-and-canonical-smoke-live-stack.md) | Architecture: kill-switch design, dual checkpoint pattern, canonical smoke structure |
| [`live-control-path-smoke-usage-and-operational-limitations.md`](../architecture/live-control-path-smoke-usage-and-operational-limitations.md) | Operations: usage guide, prerequisites, known limitations, error diagnosis |

## Kill-Switch Live Proof

### What Is Proven

Phase 7 of `smoke-live-stack.sh` validates the full control path:

| Step | Operation | Assertion |
|------|-----------|-----------|
| 7a | `GET /execution/control` | Returns 200 with current gate state |
| 7b | `PUT {"status":"halted"}` | Returns 200, response shows `halted` |
| 7c | `GET /execution/control` | Confirms gate reads `halted` (read-after-write) |
| 7d | `PUT {"status":"active"}` | Returns 200, response shows `active` |
| 7e | `GET /execution/control` | Confirms gate reads `active` (resume complete) |
| 7f | Audit field check | `updated_by` and `updated_at` survive round-trip |

### Control Path Traversed

```
curl (smoke script)
  → Gateway HTTP handler (execution_control.go)
    → SetExecutionControlUseCase
      → ControlGateway (NATS request/reply)
        → Store QueryResponderActor
          → ControlKVStore.Put() → NATS KV bucket EXECUTION_CONTROL
```

### Safety Guarantees

- EXIT trap restores gate to `active` on any exit (success, failure, or
  interrupt via SIGTERM/SIGINT)
- Temp files (`/tmp/ks_*.json`) cleaned up at end of phase
- No side effects on data pipeline — Phase 7 only touches the control surface

### What Is NOT Proven (Honest Delimitation)

| Limitation | Reason | Where It Is Proven Instead |
|-----------|--------|---------------------------|
| Real-time blocking of in-flight intents | Requires concurrent load | `live_consumer_flow_test.go` (LF-3) |
| Dual checkpoint blocking (derive + execute) | Requires pipeline activity | `control_plane_full_path_test.go` (CP-FP-2, CP-FP-4) |
| Fail-open on KV unavailability | Would require stopping NATS | `safety_gate_test.go`, `control_gate_runtime_test.go` (CG-RT-1) |
| Halt/resume under sustained load | Production concern | Out of wave scope |
| Per-type gate isolation | Not implemented (global gate) | N/A — design decision |

## Canonical Smoke Surface

### Before S335

`smoke-live-stack.sh` was an S318 verification tool with 6 phases covering
the data path only. Kill-switch was tested exclusively in Go integration tests.

### After S335

`make smoke-live-stack` is the canonical single-command surface for the Live
Stack Integration Wave. It covers:

- **Data path:** Phases 1–6 (infrastructure → NATS → ClickHouse → gateway)
- **Control path:** Phase 7 (kill-switch halt/resume via HTTP → KV)
- **Structural regression:** Phase 6 (S317 Go test gate)

All 7 phases run sequentially. A single exit code (0 or 1) reports overall
result. The script is safe for CI or manual execution.

## Evidence Chain

### Integration Tests (Proven in S333)

| Test | Proves |
|------|--------|
| LF-1 | Real supervisor → actor flow with NATS consumer |
| LF-2 | Durable consumer restart preserves state |
| LF-3 | Kill switch blocks real actor path |
| LF-4 | Multiple events processed sequentially |

### Integration Tests (Proven in Prior Stages)

| Test | Proves |
|------|--------|
| CG-RT-1..6 | KV gate round-trip: fail-open, halt, resume, cycle, audit |
| CP-FP-1..5 | Dual checkpoint: active→halted→resume, propagation |
| SafetyGate | Unit: kill switch + staleness gates |

### Smoke (Proven in S335)

| Phase | Proves |
|-------|--------|
| 1–6 | Data path end-to-end against live Docker stack |
| 7 | Control path end-to-end against live NATS KV via HTTP |

## Invariants Preserved

All 9 Production Wiring Tranche invariants hold:

| Invariant | Status |
|-----------|--------|
| EC-1 (execution event contract) | Held |
| EC-3 (correlation preservation) | Held |
| F-1 (fill event contract) | Held |
| F-4 (venue column alignment) | Held |
| RF-1 (round-trip fill visibility) | Held |
| PGR-08 (paper gate registration) | Held |
| INV-REC-1 (reconciliation) | Held |
| INV-RC-1 (retry/circuit-breaker) | Held |
| INV-OBS-1 (observability) | Held |

## Preparation for S336 (Wave Gate)

S336 is the formal closure gate for the Live Stack Integration Wave. After
S335, the wave status is:

| Block | Stage | Status |
|-------|-------|--------|
| LSI-1: NATS Consumer Flow | S333 | Complete |
| LSI-2: Fill Round-Trip | S334 | Complete |
| LSI-3: Kill-Switch + Smoke | S335 | Complete |
| LSI-4: Wave Gate | S336 | Ready to evaluate |

### Recommended S336 Checklist

1. Confirm all 4 architecture documents are consistent and cross-linked
2. Confirm all integration tests pass (`go test ./internal/...`)
3. Confirm `make smoke-live-stack` passes end-to-end
4. Verify stage INDEX completeness (S332–S335 entries present)
5. Evaluate whether any non-goals from the charter should be promoted
6. Record formal closure decision with evidence summary

### Remaining Non-Goals (Carried Forward)

- No mainnet activation
- No production SLA or latency monitoring
- No multi-gate per execution type
- No KV Watch reactive propagation
- No dashboard or alerting infrastructure
