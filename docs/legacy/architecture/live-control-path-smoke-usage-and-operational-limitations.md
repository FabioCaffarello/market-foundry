# Live Control Path: Smoke Usage and Operational Limitations

> S335 — Operational reference for using the kill-switch and smoke-live-stack
> in practice, including known limitations and prerequisites.

## Quick Reference

### Run the Canonical Smoke

```bash
# Prerequisites
make up && make seed

# Run
make smoke-live-stack

# With longer flush wait
SMOKE_WAIT=180 make smoke-live-stack
```

### Operate the Kill-Switch Manually

```bash
# Query current state
curl http://localhost:8080/execution/control

# Halt execution (e.g., risk incident)
curl -X PUT http://localhost:8080/execution/control \
  -H "Content-Type: application/json" \
  -d '{"status":"halted","reason":"risk-limit-breach","updated_by":"operator"}'

# Resume execution
curl -X PUT http://localhost:8080/execution/control \
  -H "Content-Type: application/json" \
  -d '{"status":"active","reason":"limits-reset","updated_by":"operator"}'
```

## Prerequisites

| Requirement | Why |
|-------------|-----|
| `make up` | All 9 services must be running with health checks passing |
| `make seed` | configctl must be seeded for pipeline to produce data |
| Docker Compose | Stack runs in containers; host network access via published ports |
| `python3` | JSON parsing in smoke script |
| `curl` | HTTP calls to gateway |
| `docker` | Container exec commands for NATS and ClickHouse checks |

## What the Smoke Proves

### Data Path (Phases 1–6)

- Infrastructure health (NATS, ClickHouse, gateway, writer)
- NATS stream creation and consumer registration
- ClickHouse write path (all 6 analytical tables)
- Gateway read path (composite and single-family endpoints)
- Structural correctness (S317 Go test gate)

### Control Path (Phase 7)

- HTTP → NATS request/reply → store → KV bucket round-trip
- Gate state transitions: active → halted → active
- Read-after-write consistency of gate state
- Audit field preservation (reason, updated_by, updated_at)
- Safe cleanup via EXIT trap

## Operational Limitations

### 1. Smoke Does Not Inject Synthetic Data

The smoke validates whatever the running pipeline has produced. If no
market data has been ingested (e.g., exchange is down or configctl has no
active symbols), Phases 3–5 will show warnings but not failures.

**Implication:** An empty stack passes the structural checks but shows
`[WARN]` for data presence. This is intentional — the smoke proves the
path works, not that external data sources are available.

### 2. Kill-Switch Proof Is Control-Plane Only

Phase 7 proves the control surface (HTTP set/get) and KV persistence. It
does NOT prove real-time blocking of in-flight intents under concurrent
load. That proof exists in integration tests:

- `control_gate_runtime_test.go` — CG-RT-1 through CG-RT-6
- `control_plane_full_path_test.go` — CP-FP-1 through CP-FP-5
- `live_consumer_flow_test.go` — LF-3 (kill switch blocks actor path)

### 3. No Concurrent Load During Halt

The smoke sets the gate to halted and immediately reads it back. It does
not publish execution intents during the halt window to verify they are
blocked. That would require:

- A running market data feed producing execution intents
- Timing coordination between halt and intent arrival
- Stream message counting before/after halt

This is proven in integration tests (LF-3, CP-FP-2, CP-FP-3) but not
in the smoke, by design.

### 4. Fail-Open Is Not Tested in Smoke

The system defaults to `active` when KV is unavailable. Testing this in
the smoke would require stopping NATS mid-test, which conflicts with the
stack-readiness requirement of other phases.

### 5. Single Global Gate

The current implementation uses a single global gate key (`global`). There
is no per-type or per-symbol gate. All execution types are affected by a
single halt command.

### 6. No In-Flight Drain

When the gate transitions to `halted`, intents that have already passed
the gate check but have not yet reached the venue may still be submitted.
The halt is a gate, not a drain.

### 7. Script Temp Files

Phase 7 writes temporary JSON responses to `/tmp/ks_*.json`. These are
cleaned up at the end of the phase but may persist if the script is
killed with SIGKILL (which bypasses traps).

## Error Diagnosis

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Phase 7 returns HTTP 400 | Path parameter wrong | Verify gateway routes include `/execution/:type` |
| Phase 7 returns HTTP 503 | Store binary not running or NATS down | `make logs SERVICE=store` |
| Gate stuck on halted | Previous smoke run interrupted | `curl -X PUT .../execution/control -d '{"status":"active"}'` |
| Phase 1 dies | Infrastructure not started | `make up && make seed` |
| Phase 4/5 returns 503 | ClickHouse not connected to gateway | `make logs SERVICE=gateway` |

## Relationship to Integration Tests

| Test Suite | Scope | Infrastructure |
|------------|-------|---------------|
| `safety_gate_test.go` | Unit: SafetyGate logic | None (mocks) |
| `control_test.go` | Unit: domain model | None |
| `control_gate_runtime_test.go` | Integration: KV round-trip | Embedded NATS |
| `control_plane_full_path_test.go` | Integration: dual checkpoint | Embedded NATS |
| `live_consumer_flow_test.go` | Integration: actor flow with gate | Embedded NATS |
| **`smoke-live-stack.sh` Phase 7** | **Smoke: HTTP control surface** | **Full Docker stack** |

The smoke is the outermost ring — it proves the HTTP surface works against
real infrastructure. The integration tests prove the internal mechanics.
Together they form a complete evidence chain.

## Related Documents

- [Kill-Switch Live and Canonical smoke-live-stack](kill-switch-live-and-canonical-smoke-live-stack.md)
- [Live Stack Integration Wave Charter](live-stack-integration-wave-charter-and-scope-freeze.md)
