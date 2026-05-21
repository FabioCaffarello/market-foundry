# Fail-Fast vs. Graceful Degradation Rules

> Decision matrix for when Market Foundry runtimes should fail immediately vs. degrade gracefully.

## 1. Core Distinction

| Posture | When | Effect |
|---------|------|--------|
| **Fail-fast** | A condition exists that makes correct operation impossible | Process exits, supervisor/orchestrator restarts |
| **Graceful degradation** | A condition reduces capability but correct partial operation is still possible | Process continues with reduced functionality, signals degradation via logs and diagnostics |

The default posture is **fail-fast**. Graceful degradation is an explicit opt-in that must be justified per dependency.

## 2. Decision Matrix

### 2.1 Startup Phase

| Dependency | Classification | Failure Posture | Rationale |
|------------|---------------|-----------------|-----------|
| Config file (JSONC) | Critical | Fail-fast | No config = no runtime identity |
| Config schema validation | Critical | Fail-fast | Invalid config = undefined behavior |
| Actor engine | Critical | Fail-fast | No engine = no message processing |
| NATS connection (publisher) | Critical | Fail-fast | No publisher = data pipeline broken |
| NATS connection (consumer) | Critical | Fail-fast | No consumer = no input processing |
| NATS KV store (projection) | Critical | Fail-fast | No store = no materialization target |
| NATS KV store (control gate) | Optional | Degrade | Control gate disabled, intents proceed unchecked |
| Config query gateway | Optional | Degrade | Static config used, dynamic bindings unavailable |
| Pipeline family (none enabled) | Critical | Fail-fast | No families = no work to do |

### 2.2 Runtime Phase (Steady State)

| Scenario | Classification | Failure Posture | Rationale |
|----------|---------------|-----------------|-----------|
| Publish failure (single event) | Transient | Skip + log ERROR | Event will be re-derived on next input cycle |
| Consumer decode failure (InvalidArgument) | Permanent | Term message | Malformed data cannot be retried |
| Consumer decode failure (other) | Transient | Nak message | Redelivery may succeed after transient issue resolves |
| KV write failure (single event) | Transient | Skip + log ERROR | Stale data in KV is acceptable; next write will overwrite |
| Domain validation failure | Permanent | Skip + log WARN | Reject at boundary, do not propagate garbage |
| Stale/duplicate event | Expected | Skip silently (DEBUG) | Monotonicity guards working as designed |
| NATS connection lost (mid-operation) | Transient | Readiness fails, traffic gated | Process stays alive; reconnection is handled by NATS client |
| Kill switch activated | Intentional | Skip intents + log WARN | Operator-initiated safety mechanism |
| Staleness guard triggered | Expected | Skip intent + log WARN | Prevent acting on outdated market data |
| Ack failure (after processing) | Transient | Log ERROR, continue | Message may be redelivered (idempotency guards protect) |

### 2.3 Shutdown Phase

| Scenario | Posture | Rationale |
|----------|---------|-----------|
| SIGTERM received | Graceful shutdown | Stop actors first, then health server (INV-4) |
| SIGINT received | Graceful shutdown | Same as SIGTERM |
| Actor Poison (startup failure) | Immediate exit | Critical dependency unavailable |
| Close/cleanup errors | Log + ignore | Best-effort cleanup; process is exiting anyway |

## 3. Classification Criteria

Use this checklist to classify a new dependency or failure mode:

### Mark as Critical (fail-fast) when:
- [ ] Without this dependency, the runtime cannot process any messages
- [ ] Without this dependency, the runtime would produce incorrect results
- [ ] The dependency is required for the runtime's core identity (config, engine)
- [ ] Data loss or corruption would result from continuing without this dependency

### Mark as Optional (degrade) when:
- [ ] The runtime can still perform its primary function without this dependency
- [ ] The dependency provides a safety/convenience feature that has a safe default
- [ ] The dependency is only needed for a subset of operations
- [ ] Continuing without this dependency is strictly better than crashing

## 4. Per-Runtime Summary

### Gateway
- **Critical:** config, actor engine, HTTP server binding
- **Optional:** none (gateway is the entry point; all its dependencies are critical)
- **Degradation:** none

### Ingest
- **Critical:** config, actor engine, NATS publisher (observation stream)
- **Optional:** configctl query gateway (for dynamic binding updates)
- **Degradation:** continues with static bindings if configctl unavailable

### Derive
- **Critical:** config, actor engine, NATS publisher (evidence/signal/decision/strategy/risk/execution streams), NATS consumer (observation stream)
- **Optional:** configctl query gateway
- **Degradation:** continues with static source scope configuration if configctl unavailable

### Store
- **Critical:** config, actor engine, NATS consumers (per family), NATS KV stores (per family)
- **Optional:** none (each pipeline family requires both consumer and KV store)
- **Degradation:** per-family independence — one family's failure does not affect others

### Execute
- **Critical:** config, actor engine, venue adapter, fill publisher, NATS consumer (execution intent stream)
- **Optional:** execution control KV store (kill switch)
- **Degradation:** continues without kill switch gate if control KV unavailable

### Configctl
- **Critical:** config, actor engine, NATS request-reply server
- **Optional:** none
- **Degradation:** none

## 5. Conventions for New Dependencies

When introducing a new external dependency to any runtime:

1. **Classify** using the checklist in Section 3.
2. **Startup:** If critical, fail in `start()` with `Poison(PID)`. If optional, log WARN and set the dependency reference to `nil`.
3. **Runtime:** Guard all usage of optional dependencies with nil checks. Never panic on nil optional dependency.
4. **Diagnostics:** If the dependency has a tracker, ensure both `RecordEvent()` and `RecordError()` are called on the respective paths.
5. **Document** the classification in this file's per-runtime summary.

## 6. Exceptions and Trade-Offs

| Exception | Rationale | Risk |
|-----------|-----------|------|
| Publishers don't retry failed publishes | Re-derivation on next input cycle is the recovery mechanism | If input stops arriving, the last derived event is permanently lost |
| Ack failures are logged but not tracked as RecordError | Ack failure doesn't mean processing failed; message will be redelivered | Redelivery counter may inflate for successfully-processed events |
| Control KV unavailability doesn't block execute startup | Kill switch is a safety net, not a prerequisite for operation | Operator cannot halt execution if control KV is down at startup |
| Close/cleanup errors are logged but ignored | Process is already shutting down; no recovery action is possible | Resource leak if NATS connection cleanup fails (mitigated by process exit) |
