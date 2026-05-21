# Kill-Switch Operational Runbook

> Authority: S442 | Date: 2026-03-24 | Phase: 50 (Live Trading Authorization)

## Purpose

This is the canonical operational runbook for the market-foundry kill-switch. It transforms the existing technical mechanism into a documented, testable, and auditable operational procedure.

The kill-switch exists to halt all execution immediately and reversibly. This document defines who can trigger it, how, when, and what happens after.

## Technical Foundation

### Architecture Summary

The kill-switch is a `ControlGate` stored in NATS JetStream KV bucket `EXECUTION_CONTROL`, key `global`.

| Component | Role |
|-----------|------|
| `domain/execution/control.go` | Domain model: `GateStatus` (active/halted), `ControlGate` struct |
| `natsexecution/control_kv_store.go` | KV persistence: Get/Put/IsHalted against NATS JetStream |
| `execution/safety_gate.go` | Pre-submit enforcement: checks gate before every venue call |
| `derive/execution_publisher_actor.go` | Derive-side enforcement: checks gate before publishing intents |
| `execute/venue_adapter_actor.go` | Execute-side enforcement: SafetyGate check on every intent |
| `http/handlers/execution_control.go` | HTTP surface: GET/PUT `/execution/control` |
| `natsexecution/control_gateway.go` | Gateway-to-store NATS request/reply bridge |

### Dual Checkpoint Pattern

The gate is checked at two independent points:

1. **Derive binary** (`ExecutionPublisherActor`): blocks intent publishing before it reaches the NATS stream.
2. **Execute binary** (`VenueAdapterActor` via `SafetyGate`): blocks venue submission even if an intent was already in-flight.

Both checkpoints are fail-open: if NATS KV is unreachable, execution continues. This is by design -- a transient NATS hiccup should not halt the pipeline. The kill-switch is an affirmative halt, not a consent mechanism.

### Control Surface

| Verb | Path | Effect |
|------|------|--------|
| `GET` | `/execution/control` | Read current gate state |
| `PUT` | `/execution/control` | Set gate to `active` or `halted` |

Gateway port: `127.0.0.1:8080` (default compose mapping).

### Audit Fields

Every gate mutation carries:

| Field | Purpose |
|-------|---------|
| `status` | `active` or `halted` |
| `reason` | Free-text: why the gate was changed |
| `updated_by` | Identity of the operator who made the change |
| `updated_at` | UTC timestamp (set by domain layer) |

These fields persist in the NATS KV entry and are returned on every GET.

## Operational Procedures

### Procedure 1: Emergency Halt (Kill-Switch Activation)

**When to use**: Any situation requiring immediate cessation of all venue interaction.

**Trigger criteria** (any one is sufficient):

- Unexpected order state (fill without submission, unknown venue order ID)
- Exchange API error rate >10% sustained over 5 minutes
- Credential compromise or suspected leak
- ClickHouse write failures (audit trail broken)
- NATS connectivity loss >30 seconds
- Risk limit breach
- Operator judgment (no justification required to halt)

**Steps**:

```bash
# Using the canonical script:
./scripts/kill-switch-ops.sh halt "reason-for-halt" "operator-name"

# Or manual curl:
curl -X PUT http://127.0.0.1:8080/execution/control \
  -H "Content-Type: application/json" \
  -d '{"status":"halted","reason":"reason-for-halt","updated_by":"operator-name"}'
```

**Expected outcome**:
- Response contains `"status":"halted"`
- No new intents are published by derive binary
- No new venue calls are made by execute binary
- In-flight venue calls complete normally (the gate does not cancel active HTTP requests)
- `skipped_halt` counter increments on subsequent intent arrivals

**Latency SLA**: Gate state propagates on next intent cycle. No intent is held longer than the gate-read timeout (2 seconds). Effective halt latency = time until next intent arrives + 2s gate read.

### Procedure 2: Verification After Halt

**When to use**: Immediately after every halt, before any investigation.

**Steps**:

```bash
# Using the canonical script:
./scripts/kill-switch-ops.sh verify-halted

# Manual verification:
# 1. Read gate state:
curl -s http://127.0.0.1:8080/execution/control | jq .

# 2. Check execute statusz for halt evidence:
curl -s http://127.0.0.1:8084/statusz | jq '.trackers[].counters.skipped_halt'

# 3. Check derive statusz for halt evidence:
curl -s http://127.0.0.1:8083/statusz | jq '.trackers[].counters["execution:gate_halted"]'
```

**Acceptance criteria**:
- Gate status is `halted`
- `reason` and `updated_by` match the halt command
- No new fills appear after halt timestamp
- `skipped_halt` or `execution:gate_halted` counters are incrementing (if intents are still arriving from upstream)

### Procedure 3: Resume After Investigation

**When to use**: Only after the root cause of the halt has been identified and either resolved or documented as accepted.

**Pre-resume checklist**:

- [ ] Root cause identified and documented
- [ ] If credential issue: credentials rotated and verified
- [ ] If exchange issue: exchange health verified (API status page, manual test call)
- [ ] If state corruption: KV and ClickHouse state audited
- [ ] Operator makes explicit decision to resume (no automatic resume)

**Steps**:

```bash
# Using the canonical script:
./scripts/kill-switch-ops.sh resume "investigation-complete-reason" "operator-name"

# Manual curl:
curl -X PUT http://127.0.0.1:8080/execution/control \
  -H "Content-Type: application/json" \
  -d '{"status":"active","reason":"investigation-complete","updated_by":"operator-name"}'
```

**Post-resume verification**:

```bash
./scripts/kill-switch-ops.sh verify-active
```

**Expected outcome**:
- Gate status is `active`
- Next arriving intent is processed normally
- `filled` counter begins incrementing again
- No backlog of stale intents is submitted (staleness guard rejects old intents independently)

### Procedure 4: Full Cycle Test

**When to use**: Before any live trading session. Validates the complete halt/verify/resume/verify path.

```bash
./scripts/kill-switch-ops.sh cycle "pre-session-test" "operator-name"
```

This executes: halt -> 2s wait -> verify-halted -> resume -> 2s wait -> verify-active, with timestamped evidence at each step.

## Authorization Model

### Who May Trigger the Kill Switch

Any operator with HTTP access to the gateway may halt execution. There is no authentication layer on the gateway HTTP API beyond network access.

**Current limitation**: The gateway HTTP API has no authentication. Access control is network-level only (compose binds to `127.0.0.1:8080`). This is sufficient for single-operator, single-machine deployments but is a known gap for multi-operator environments.

### Who May Resume

Same access model as halt. The `updated_by` field is operator-asserted (not verified). Resume requires conscious operator action -- there is no automatic resume, timer-based resume, or watchdog resume.

## Fail-Open Semantics and Limitations

### Known Limitations

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| Fail-open on NATS KV unavailability | If NATS is down, gate check returns "active" (execution continues) | NATS health is monitored via `/readyz`; NATS loss >30s is itself a halt trigger |
| Global gate only (no per-segment) | Cannot halt Spot without halting Futures | Sufficient for minimum authorized scope (Spot-only); NG-13 in charter |
| No HTTP authentication on gateway | Anyone with network access can halt or resume | Gateway bound to `127.0.0.1`; network-level isolation |
| No automatic halt on anomaly detection | Kill-switch is manual-only; no automated triggers | Operator must monitor and decide; this is deliberate (human-in-the-loop) |
| In-flight requests complete | An active venue HTTP call is not cancelled by halt | Design choice: cancelling mid-flight could leave orphaned orders |
| `updated_by` is operator-asserted | No identity verification on the audit field | Acceptable for single-operator scope |
| Gate state is not versioned | No conflict detection on concurrent writes | Single-operator scope; NATS KV provides last-write-wins |

### Fail-Open Rationale

The gate defaults to `active` when unreadable because:

1. A transient NATS reconnection should not halt production execution
2. The kill-switch is a manual safety mechanism, not a consent gate
3. If NATS is genuinely unavailable, the entire pipeline is already degraded (no intent delivery, no fill publishing)

The fail-open behavior is a design decision documented since S335. It is NOT a gap -- it is the correct behavior for this architecture.

## Observability During Halt

| Signal | Endpoint | What to Check |
|--------|----------|---------------|
| Gate state | `GET /execution/control` | `status`, `reason`, `updated_at` |
| Execute halt evidence | `GET :8084/statusz` | `skipped_halt` counter |
| Derive halt evidence | `GET :8083/statusz` | `execution:gate_halted` counter |
| Execute phase | `GET :8084/statusz` | `phase` field (may show `idle` during halt) |
| Prometheus gate metric | `GET :8084/metrics` | `execution_gate_active` (0 = halted) |

## Links

- Technical detail: [kill-switch-trigger-verification-rollback-and-recovery-procedure.md](kill-switch-trigger-verification-rollback-and-recovery-procedure.md)
- Operational script: `scripts/kill-switch-ops.sh`
- Prior art: [kill-switch-live-and-canonical-smoke-live-stack.md](kill-switch-live-and-canonical-smoke-live-stack.md)
- Wave charter: [live-trading-authorization-wave-charter-and-scope-freeze.md](live-trading-authorization-wave-charter-and-scope-freeze.md)
- Safety gate source: `internal/application/execution/safety_gate.go`
- Control domain model: `internal/domain/execution/control.go`
