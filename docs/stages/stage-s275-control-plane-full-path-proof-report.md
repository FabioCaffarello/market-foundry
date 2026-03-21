# Stage S275: Control Plane Full-Path Proof — Report

Stage: S275
Status: complete
Date: 2026-03-21

## Objective

Validate the complete control plane path from control surface write to observable execution behavior, proving that the ControlGate propagates through KV persistence and impacts both the derive publisher and venue adapter execution flows.

## What Was Done

### Test Artifact

Created `internal/adapters/nats/natsexecution/control_plane_full_path_test.go` with 5 integration tests:

| Test | What It Proves |
|------|---------------|
| `TestControlPlane_FullPath_Active_PublishesToStream` | KV active → publisher gate allows → message on EXECUTION_EVENTS stream |
| `TestControlPlane_FullPath_Halted_BlocksStreamPublish` | KV halted → publisher gate blocks → no message on stream |
| `TestControlPlane_FullPath_ActiveHaltedResume_Cycle` | 3-phase cycle: active→publish, halted→block, resume→publish |
| `TestControlPlane_FullPath_DualCheckpoint_PublisherAndVenue` | Both derive and execute paths observe same KV state |
| `TestControlPlane_FullPath_ImmediatePropagation` | 10 rapid state changes, zero stale reads, publish behavior matches |

### Architecture Documentation

- `docs/architecture/control-plane-full-path-proof.md` — what was proven, full topology diagram, relationship to prior proofs
- `docs/architecture/control-plane-runtime-propagation-findings.md` — runtime findings, limits, and operational implications

## Evidence

All 5 tests pass against NATS v2.12.4 with JetStream:

```
--- PASS: TestControlPlane_FullPath_Active_PublishesToStream (0.00s)
--- PASS: TestControlPlane_FullPath_Halted_BlocksStreamPublish (0.51s)
--- PASS: TestControlPlane_FullPath_ActiveHaltedResume_Cycle (0.51s)
--- PASS: TestControlPlane_FullPath_DualCheckpoint_PublisherAndVenue (0.01s)
--- PASS: TestControlPlane_FullPath_ImmediatePropagation (0.01s)
PASS
ok  	internal/adapters/nats/natsexecution	1.230s
```

Zero regressions on existing test suites:
- S273 ControlGateRuntime: 6/6 PASS
- S271 KV Roundtrip: 8/8 PASS
- Derive actor chain: PASS
- Execution application: PASS

## Full Path Proven

```
ControlKVStore.Put(gate)
  → EXECUTION_CONTROL KV bucket (key: "global")
    → ControlKVStore.IsHalted(ctx)  [real KV read]
      → if active: Publisher.PublishExecution() → EXECUTION_EVENTS stream ✓
      → if halted: drop + halted counter ✓
      → resume: publish resumes ✓
    → SafetyGate.Check()  [same KV source]
      → if active: VenueAdapter.SubmitOrder() ✓
      → if halted: skip + kill_switch ✓
```

## Files Changed

| File | Action | Purpose |
|------|--------|---------|
| `internal/adapters/nats/natsexecution/control_plane_full_path_test.go` | added | 5 full-path integration tests |
| `docs/architecture/control-plane-full-path-proof.md` | added | Proof topology and proven properties |
| `docs/architecture/control-plane-runtime-propagation-findings.md` | added | Runtime findings and limits |
| `docs/stages/stage-s275-control-plane-full-path-proof-report.md` | added | This report |

## Gap Closure

| Prior Gap | Closed By |
|-----------|-----------|
| Derive publisher gate path not proven at full-path level | CP-FP-1, CP-FP-2, CP-FP-3 |
| No stream-level observation of gate decisions | CP-FP-1, CP-FP-2, CP-FP-3 |
| No proof that both gate check points see same state | CP-FP-4 |
| No rapid propagation test | CP-FP-5 |

## Remaining Limits

1. **Gateway request/reply surface** (`execution.control.set`) not exercised in full-path test — KV writes are direct
2. **Multi-node JetStream** not validated (single-node only)
3. **Concurrent writer contention** not tested (last-write-wins assumed)
4. **No KV watcher** — poll-on-read only, no push notification of gate changes
5. **No append-only audit log** — gate transitions overwrite previous state

## Recommendations for S276

1. **Gateway surface integration**: Wire `ControlGateway.SetExecutionControl()` through the store's NATS request/reply responder and verify the full path from gateway HTTP → NATS request → KV Put → gate read → stream effect. This closes the last wiring gap between the operator-facing surface and the proven KV path.

2. **CI integration**: Add the full-path tests to the CI matrix alongside the existing S273 and S271 tests, with NATS JetStream service.

3. **Operational observability**: Consider exposing gate status via the gateway's `/status` endpoint so operators can verify gate state without reading KV directly.
