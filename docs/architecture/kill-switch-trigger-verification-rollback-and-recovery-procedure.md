# Kill-Switch Trigger, Verification, Rollback, and Recovery Procedure

> Authority: S442 | Date: 2026-03-24 | Phase: 50 (Live Trading Authorization)

## Purpose

This document provides the detailed step-by-step procedure for every kill-switch operation, with explicit verification gates at each transition. It is the technical companion to the [operational runbook](kill-switch-operational-runbook.md).

## Trigger Decision Matrix

### Immediate Halt (No Deliberation Required)

| Trigger | Severity | Rationale |
|---------|----------|-----------|
| Unexpected fill (order filled without submission) | Critical | State integrity compromised |
| Unknown venue order ID in fill event | Critical | Possible stale or corrupted state |
| Credential compromise or suspected leak | Critical | Unauthorized access risk |
| Real order placed when dry_run=true expected | Critical | Safety layer failure |

### Conditional Halt (Assess Within 5 Minutes)

| Trigger | Threshold | Rationale |
|---------|-----------|-----------|
| Exchange API errors | >10% rate for >5 minutes | Sustained venue degradation |
| ClickHouse write failures | Any write failure | Audit trail integrity broken |
| NATS connectivity loss | >30 seconds | Control plane unreliable |
| Fill price deviation | >5% from last market price | Possible venue anomaly |
| Resource exhaustion | Memory, disk, or CPU saturation | Process stability at risk |

### Operator Discretion

The operator may halt at any time for any reason. No justification is required to halt. Justification IS required to resume.

## Procedure A: Halt Execution

### Pre-Halt State Capture

Before halting, capture the current state for post-halt comparison:

```bash
# 1. Record current gate state
curl -s http://127.0.0.1:8080/execution/control | tee /tmp/killswitch-pre-halt-gate.json

# 2. Record execute statusz (counters before halt)
curl -s http://127.0.0.1:8084/statusz | tee /tmp/killswitch-pre-halt-execute.json

# 3. Record derive statusz
curl -s http://127.0.0.1:8083/statusz | tee /tmp/killswitch-pre-halt-derive.json

# 4. Record timestamp
date -u +"%Y-%m-%dT%H:%M:%S.%3NZ" | tee /tmp/killswitch-pre-halt-ts.txt
```

### Execute Halt

```bash
./scripts/kill-switch-ops.sh halt "<reason>" "<operator>"
```

The reason MUST be specific and actionable. Examples:
- `exchange-api-5xx-rate-12pct-sustained-7min`
- `unexpected-fill-without-submission-btcusdt`
- `credential-rotation-in-progress`
- `operator-precaution-before-maintenance`

### Post-Halt Verification (Mandatory)

This verification MUST be performed immediately after every halt. Do not proceed to investigation without it.

```bash
# Automated verification:
./scripts/kill-switch-ops.sh verify-halted
```

**Manual verification steps** (if script is unavailable):

1. **Gate state check**:
   ```bash
   curl -s http://127.0.0.1:8080/execution/control | jq .
   ```
   Expected: `"status": "halted"`, correct `reason` and `updated_by`.

2. **Execute checkpoint check**:
   ```bash
   curl -s http://127.0.0.1:8084/statusz | jq '.trackers[] | {name, counters: {skipped_halt: .counters.skipped_halt, filled: .counters.filled}}'
   ```
   Expected: `skipped_halt` counter >= pre-halt value. No new `filled` increments after halt timestamp.

3. **Derive checkpoint check**:
   ```bash
   curl -s http://127.0.0.1:8083/statusz | jq '.trackers[] | {name, counters: {gate_halted: .counters["execution:gate_halted"]}}'
   ```
   Expected: `execution:gate_halted` counter >= pre-halt value.

4. **Absence of new fills**:
   ```bash
   # Check latest fill timestamp (if available via gateway)
   curl -s http://127.0.0.1:8080/execution/venue_market_order/latest | jq '.metadata.timestamp'
   ```
   Expected: timestamp is before the halt time.

### Verification Verdict

- All four checks pass: **HALT CONFIRMED**. Proceed to investigation.
- Gate is halted but counters not incrementing: **HALT CONFIRMED** (no intents arriving, which is expected if upstream pipeline is also idle).
- Gate is NOT halted: **HALT FAILED**. Re-issue halt command. If second attempt fails, escalate to process restart.

## Procedure B: Investigation During Halt

While execution is halted, the operator investigates the triggering condition. The pipeline is safe -- no new orders are submitted.

### Investigation Checklist

- [ ] Identify the triggering event with timestamps
- [ ] Check exchange status (Binance API status page or manual test call)
- [ ] Check NATS health: `curl -s http://127.0.0.1:8081/readyz`
- [ ] Check ClickHouse health: `curl -s http://127.0.0.1:8085/readyz`
- [ ] Review execute logs: `docker compose logs execute --since 10m`
- [ ] Review derive logs: `docker compose logs derive --since 10m`
- [ ] Check for orphaned or unexpected KV state: `GET /execution/venue_market_order/latest`
- [ ] Document findings

### Investigation Output

Before resuming, the operator must have:
1. A documented root cause (or explicit "unknown -- accepted risk" statement)
2. A remediation action taken (or explicit "no action needed" statement)
3. An explicit decision to resume

## Procedure C: Resume Execution

### Pre-Resume Checklist (All Must Be True)

- [ ] Root cause is documented
- [ ] If credential issue: credentials rotated and re-verified via preflight
- [ ] If exchange issue: exchange API health confirmed
- [ ] If state issue: KV and ClickHouse state audited, no orphaned records
- [ ] If process issue: affected binary restarted if needed
- [ ] Operator explicitly decides to resume (not delegated, not automatic)

### Execute Resume

```bash
./scripts/kill-switch-ops.sh resume "<reason>" "<operator>"
```

The reason MUST reference the investigation:
- `exchange-api-recovered-confirmed-via-manual-call`
- `credential-rotated-preflight-passed`
- `false-alarm-fill-was-delayed-not-orphaned`

### Post-Resume Verification (Mandatory)

```bash
./scripts/kill-switch-ops.sh verify-active
```

**Extended verification** (within 5 minutes of resume):

1. Gate is `active`
2. Next intent cycle produces a fill or dry-run interception (not `skipped_halt`)
3. `filled` counter begins incrementing again
4. No backlog of stale intents is submitted (staleness guard rejects old intents)

## Procedure D: Full Cycle Validation

Run before any live trading session to prove the kill-switch path end-to-end.

```bash
./scripts/kill-switch-ops.sh cycle "pre-session-validation" "operator-name"
```

### Expected Output

```
=============================================
  Kill-Switch Full Cycle Test
  2026-03-24T10:00:00Z
=============================================

[INFO]  Step 1/4: HALT
[PASS]  Gate set to HALTED at 2026-03-24T10:00:00Z

[INFO]  Step 2/4: VERIFY HALTED
[PASS]  Gate is HALTED
[PASS]  Execute reachable -- gate is enforced at both checkpoints

[INFO]  Step 3/4: RESUME
[PASS]  Gate set to ACTIVE at 2026-03-24T10:00:02Z

[INFO]  Step 4/4: VERIFY ACTIVE
[PASS]  Gate is ACTIVE

=============================================
[PASS]  Kill-Switch Full Cycle PASSED
  Pre-halt:     2026-03-24T10:00:00.000Z
  Post-halt:    2026-03-24T10:00:02.000Z
  Post-resume:  2026-03-24T10:00:04.000Z
=============================================
```

### Cycle Failure Handling

If the cycle fails at any step:
- Do NOT start a live trading session
- Investigate the failed step
- Common causes: gateway not running, NATS not connected, network binding issue
- Re-run cycle after fix

## Rollback Procedure

### Rollback from Live to Dry-Run

If live trading has been enabled (`dry_run=false`) and must be reverted:

1. **Immediate**: Trigger kill-switch halt (stops all execution)
2. **Config change**: Edit `execute` config to set `dry_run: true`
3. **Restart**: Restart execute binary with new config
4. **Verify**: Confirm DryRunSubmitter is active via logs:
   ```
   grep "dry_run_submitter" <execute-logs> | head -5
   ```
5. **Resume**: Resume gate to `active` -- now executing in dry-run mode

### Rollback from Active to Halted (Standard Kill-Switch)

This is Procedure A above. No config change needed.

### Process-Level Recovery (Nuclear Option)

If the kill-switch mechanism itself is unreliable (NATS down, gateway unreachable):

1. **Stop the execute binary**: `docker compose stop execute`
2. **Stop the derive binary**: `docker compose stop derive` (stops intent generation)
3. Investigate infrastructure
4. Restart in order: NATS -> store -> derive -> execute

This is a last resort. The HTTP kill-switch should be preferred in all normal scenarios.

## Timing Characteristics

| Operation | Expected Latency |
|-----------|-----------------|
| PUT /execution/control (halt) | <100ms (gateway -> NATS request/reply -> KV write) |
| Gate propagation to next intent | 0--2s (depends on when next intent arrives + 2s gate read timeout) |
| Full cycle test | ~6s (halt + 2s wait + verify + resume + 2s wait + verify) |
| Resume to first new fill | Depends on upstream intent generation rate (typically <60s) |

### SLA Definition

The kill-switch halt SLA is: **no new venue call is made more than 2 seconds after the PUT response is received**. This is bounded by the `gateReadTimeout` in `SafetyGate` (hardcoded to 2 seconds in `venue_adapter_actor.go:114`).

In-flight venue calls that were already past the gate check when the halt was issued will complete. This is correct behavior -- cancelling a mid-flight venue call could leave an orphaned order.

## Limitations

| Limitation | Accepted? | Rationale |
|------------|-----------|-----------|
| Global gate only | Yes | NG-13: single-segment scope for minimum authorization |
| No automated triggers | Yes | Human-in-the-loop is deliberate for authorization scope |
| No per-operator authentication | Yes | Single-operator, localhost-only deployment |
| Fail-open on NATS unavailability | Yes | Design decision since S335; NATS loss is itself a halt trigger |
| No gate state versioning/CAS | Yes | Single-operator; NATS KV last-write-wins is sufficient |
| No audit log beyond KV state | Yes | Current KV entry carries reason/by/timestamp; historical log is future work |
| Staleness guard is independent | Yes | Stale intents are rejected regardless of gate state; this is correct |

## Links

- Operational runbook: [kill-switch-operational-runbook.md](kill-switch-operational-runbook.md)
- Operational script: `scripts/kill-switch-ops.sh`
- Domain model: `internal/domain/execution/control.go`
- Safety gate: `internal/application/execution/safety_gate.go`
- Venue adapter gate check: `internal/actors/scopes/execute/venue_adapter_actor.go`
- Derive-side gate check: `internal/actors/scopes/derive/execution_publisher_actor.go`
- KV store: `internal/adapters/nats/natsexecution/control_kv_store.go`
- Prior control-path proof: [control-gate-runtime-halt-resume-operational-proof.md](control-gate-runtime-halt-resume-operational-proof.md)
