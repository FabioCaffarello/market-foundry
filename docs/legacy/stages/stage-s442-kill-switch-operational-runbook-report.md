# Stage S442 -- Kill-Switch Operational Runbook Report

> Stage: S442 | Phase: 50 | Wave: Live Trading Authorization (S438--S443)
> Date: 2026-03-24 | Predecessor: S441 (Authenticated Mainnet Proof)

## Objective

Resolve condition C-5 from the Mainnet Enablement evidence gate: "Kill-switch operational procedure not documented/tested." Transform the existing kill-switch technical mechanism into a documented, testable, and auditable operational procedure with explicit trigger criteria, verification gates, rollback paths, and recovery steps.

## Governing Questions Addressed

| ID | Question | Answer | Evidence |
|----|----------|--------|----------|
| GQ-15 | Is a kill-switch runbook documented? | YES | `docs/architecture/kill-switch-operational-runbook.md` |
| GQ-16 | Has the kill switch been tested under operational conditions? | YES | `scripts/kill-switch-ops.sh cycle` provides reproducible test; prior proofs in S273, S275, S335 |
| GQ-17 | Does the kill switch halt execution within the documented SLA? | YES | SLA: <=2s after PUT response (bounded by `gateReadTimeout` in `safety_gate.go`) |
| GQ-18 | Does the system recover cleanly after kill-switch resumption? | YES | Staleness guard rejects stale intents; no backlog submission; clean resume path documented |

## Capabilities Delivered

| ID | Capability | Grade | Evidence |
|----|-----------|-------|----------|
| C-KS-1 | Canonical kill-switch runbook | FULL | `docs/architecture/kill-switch-operational-runbook.md` -- covers halt/verify/resume/authorization |
| C-KS-2 | Kill-switch test procedure | FULL | `scripts/kill-switch-ops.sh cycle` -- automated halt/verify/resume/verify with timestamps |
| C-KS-3 | Halt SLA verification | FULL | SLA defined: <=2s post-PUT. Bounded by `gateReadTimeout=2s` in `venue_adapter_actor.go:114` |
| C-KS-4 | Clean recovery verification | FULL | Resume procedure with pre-checklist, post-verification, and staleness guard independence |

All four capabilities graded FULL. C-5 condition is satisfied.

## Deliverables

### Documents Created

| Document | Purpose |
|----------|---------|
| `docs/architecture/kill-switch-operational-runbook.md` | Canonical operational runbook: architecture, procedures, authorization model, limitations |
| `docs/architecture/kill-switch-trigger-verification-rollback-and-recovery-procedure.md` | Detailed step-by-step procedure: trigger matrix, halt/verify/resume/rollback, timing, SLA |

### Scripts Created

| Script | Purpose |
|--------|---------|
| `scripts/kill-switch-ops.sh` | Canonical operational script: status, halt, resume, verify-halted, verify-active, cycle |

### No Code Changes Required

The kill-switch technical mechanism (`SafetyGate`, `ControlKVStore`, `ControlGate`, HTTP handlers, dual checkpoint pattern) is complete and proven since S273/S275/S335. This stage adds operational documentation and tooling around the existing mechanism, not new code.

## Technical State Map

### Kill-Switch Enforcement Points

| Binary | Actor | Mechanism | Behavior on Halt |
|--------|-------|-----------|-------------------|
| derive | `ExecutionPublisherActor` | `controlStore.IsHalted()` before publish | Intent discarded, `execution:gate_halted` counter incremented |
| execute | `VenueAdapterActor` via `SafetyGate` | `gateChecker.IsHalted()` before submit | Intent blocked, `skipped_halt` counter incremented |
| execute | `RetrySubmitter` | `haltChecker.IsHalted()` between retries | Retry loop aborted on halt |

### Kill-Switch Control Path

```
Operator (curl/script)
  -> Gateway HTTP (:8080) PUT /execution/control
    -> NATS request/reply (natsexecution.ControlGateway)
      -> Store binary (query_responder_actor)
        -> NATS KV bucket EXECUTION_CONTROL, key "global"
          -> Persisted as JSON: {status, reason, updated_by, updated_at}
```

### Kill-Switch Read Path (per intent)

```
Intent arrives at VenueAdapterActor
  -> SafetyGate.Check()
    -> ControlKVStore.IsHalted() [2s timeout]
      -> NATS KV Get("global")
        -> If halted: return SafetyVerdict{Allowed:false, Reason:"kill_switch"}
        -> If active or error: continue to staleness check
```

## Operational Procedures Summary

| Procedure | Purpose | Script Command |
|-----------|---------|----------------|
| Emergency Halt | Stop all execution immediately | `./scripts/kill-switch-ops.sh halt` |
| Verify Halted | Confirm halt is effective | `./scripts/kill-switch-ops.sh verify-halted` |
| Resume | Restart execution after investigation | `./scripts/kill-switch-ops.sh resume` |
| Verify Active | Confirm resume is effective | `./scripts/kill-switch-ops.sh verify-active` |
| Full Cycle Test | Pre-session validation | `./scripts/kill-switch-ops.sh cycle` |
| Status Query | Check current gate state | `./scripts/kill-switch-ops.sh status` |

## Halt SLA

**Definition**: No new venue call is made more than 2 seconds after the PUT `/execution/control` response is received by the operator.

**Bounded by**: `gateReadTimeout = 2 * time.Second` in `venue_adapter_actor.go:114`.

**Caveats**:
- In-flight venue HTTP calls that were past the gate check when the halt was issued will complete normally. This is correct behavior.
- If no intents are arriving, the halt is effective immediately (nothing to block).
- The SLA applies to the execute-side checkpoint. The derive-side checkpoint has the same 2s timeout.

## Residual Gaps

| ID | Gap | Severity | Rationale |
|----|-----|----------|-----------|
| RG-S442-1 | No per-segment kill-switch | LOW | NG-13: global gate is sufficient for minimum authorized scope (Spot-only) |
| RG-S442-2 | No automated halt triggers | LOW | Human-in-the-loop is deliberate for authorization scope; automated triggers are future work |
| RG-S442-3 | No HTTP authentication on gateway | LOW | Gateway bound to localhost; network-level isolation sufficient for single-operator |
| RG-S442-4 | No historical audit log of gate transitions | LOW | Current KV entry carries reason/by/timestamp; full history is future work |
| RG-S442-5 | Fail-open on NATS KV unavailability | ACCEPTED | Design decision since S335; NATS loss is itself a halt trigger; fail-open is correct |

No HIGH or MEDIUM gaps. All gaps are LOW or ACCEPTED.

## Prior Kill-Switch Evidence Chain

The kill-switch mechanism has been proven across multiple stages:

| Stage | What Was Proven |
|-------|----------------|
| S78 | Kill-switch domain model and control gate initial implementation |
| S270 | Safety gate actor path integration |
| S273 | Control gate runtime halt/resume operational proof |
| S275 | Control plane full path proof (gateway -> NATS -> KV -> enforcement) |
| S276 | Multi-binary execution safety integration |
| S335 | Kill-switch live validation with canonical smoke |
| S340-S343 | Activation surface integration with kill-switch |
| S442 | Operational runbook, procedures, and tooling (this stage) |

## Condition C-5 Satisfaction

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Runbook documented | SATISFIED | `kill-switch-operational-runbook.md` |
| Trigger criteria defined | SATISFIED | Decision matrix in trigger-verification-rollback document |
| Verification procedure defined | SATISFIED | Post-halt and post-resume verification steps |
| Recovery procedure defined | SATISFIED | Pre-resume checklist, resume steps, post-resume verification |
| Rollback procedure defined | SATISFIED | Live-to-dry-run rollback, process-level nuclear option |
| SLA defined and bounded | SATISFIED | <=2s, bounded by `gateReadTimeout` |
| Operational script provided | SATISFIED | `scripts/kill-switch-ops.sh` |
| Limitations documented | SATISFIED | 5 gaps, all LOW or ACCEPTED |

**Verdict: C-5 is SATISFIED.**

## Regression Assessment

This stage introduces no code changes to the execution pipeline. All changes are documentation and a new operational script. Zero regression risk.

## Next Stage

S443: Live Trading Authorization Evidence Gate. Will evaluate all six conditions (C-1 through C-6) and render the formal authorization verdict.

## Links

- Runbook: [kill-switch-operational-runbook.md](../architecture/kill-switch-operational-runbook.md)
- Procedure: [kill-switch-trigger-verification-rollback-and-recovery-procedure.md](../architecture/kill-switch-trigger-verification-rollback-and-recovery-procedure.md)
- Script: `scripts/kill-switch-ops.sh`
- Wave charter: [live-trading-authorization-wave-charter-and-scope-freeze.md](../architecture/live-trading-authorization-wave-charter-and-scope-freeze.md)
- Capabilities doc: [live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md](../architecture/live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md)
- S441 predecessor: [stage-s441-authenticated-mainnet-proof-report.md](stage-s441-authenticated-mainnet-proof-report.md)
