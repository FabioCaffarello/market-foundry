# Activation States, Transitions, and Safety Boundaries

> **Authority:** S338 · **Wave:** Venue Activation (S337–S342)
> **Companion:** [Activation Policy — Rollout and Rollback Model](activation-policy-rollout-and-rollback-model.md)
> **Scope:** State machine, transition rules, and safety invariants for venue activation

---

## 1. Purpose

This document defines the **canonical state model** for venue activation. It
enumerates all valid states, legal transitions, and the safety boundaries that
prevent ambiguous or dangerous state combinations.

---

## 2. State Dimensions

Venue activation is composed of three independent state dimensions. The
combination of these dimensions defines the **effective activation posture** of
the system.

### 2.1 Adapter State (Binary-Level)

The adapter state is determined at binary startup and is immutable for the
lifetime of the process.

| State | `venue.type` value | Meaning |
|-------|--------------------|---------|
| **PAPER** | `paper_simulator` | Paper adapter loaded; no real venue calls |
| **VENUE** | `binance_futures_testnet` | Real venue adapter loaded; HTTP calls possible |

**Transitions:** Only via binary restart with different configuration.

```
PAPER ──[restart with venue.type=binance_futures_testnet]──→ VENUE
VENUE ──[restart with venue.type=paper_simulator]──────────→ PAPER
```

### 2.2 Gate State (Runtime)

The gate state is the kill-switch value in NATS KV, mutable at runtime via HTTP.

| State | `status` value | Meaning |
|-------|----------------|---------|
| **ACTIVE** | `active` | Intents allowed to reach venue adapter |
| **HALTED** | `halted` | Intents blocked at both checkpoints |

**Transitions:** Via HTTP PUT to `/execution/control`.

```
ACTIVE ──[PUT {"status":"halted"}]──→ HALTED
HALTED ──[PUT {"status":"active"}]──→ ACTIVE
```

### 2.3 Credential State (Environment)

The credential state is determined by environment variables at binary startup.

| State | Meaning |
|-------|---------|
| **PRESENT** | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `_API_SECRET` both set |
| **ABSENT** | One or both credentials missing |

**Transitions:** Only via environment change and binary restart.

**Binary behavior:**
- `venue.type=binance_futures_testnet` + ABSENT → binary exits with error at startup
- `venue.type=paper_simulator` + ABSENT → binary starts normally (credentials not required)
- `venue.type=paper_simulator` + PRESENT → binary starts normally (credentials ignored)

---

## 3. Composite State Matrix

The effective activation posture is the combination of all three dimensions:

| Adapter | Gate | Credentials | Effective Posture | Orders Reach Venue? |
|---------|------|-------------|-------------------|---------------------|
| PAPER | ACTIVE | ABSENT | **Paper-Active** (normal dev) | No |
| PAPER | ACTIVE | PRESENT | **Paper-Active** (credentials ignored) | No |
| PAPER | HALTED | * | **Paper-Halted** (pipeline paused) | No |
| VENUE | HALTED | PRESENT | **Venue-Halted** (safe activation) | No |
| VENUE | ACTIVE | PRESENT | **Venue-Active** (real execution) | **YES** |
| VENUE | * | ABSENT | **Startup Failure** (binary exits) | No |

Only one combination produces real venue execution: **VENUE + ACTIVE + PRESENT**.

### 3.1 Recommended Activation Sequence

```
Paper-Active          (normal operation)
    │
    ▼  [set gate to HALTED]
Paper-Halted          (pipeline paused)
    │
    ▼  [set credentials, restart with venue.type=binance_futures_testnet]
Venue-Halted          (Phase 0: safe activation, no orders)
    │
    ▼  [set gate to ACTIVE]
Venue-Active          (Phase 1/2: real execution)
    │
    ▼  [set gate to HALTED]  ← immediate halt
Venue-Halted          (execution stopped, binary still running)
    │
    ▼  [restart with venue.type=paper_simulator]  ← full rollback
Paper-Halted          (back to paper)
    │
    ▼  [set gate to ACTIVE]
Paper-Active          (normal operation restored)
```

### 3.2 Forbidden Transitions

| Transition | Why forbidden |
|------------|---------------|
| Paper-Active → Venue-Active (single step) | Must pass through Venue-Halted to verify adapter initialization |
| Venue-Active → Paper-Active (single step) | Must halt before restarting to prevent in-flight race |
| Any state → Startup Failure | Not a transition — binary refuses to start, no state change |

---

## 4. Safety Gate Evaluation

The `SafetyGate` evaluates two sequential checks before any intent reaches the
venue adapter. Both checks must pass for execution to proceed.

### 4.1 Check Sequence

```
Intent arrives at VenueAdapterActor.onIntent()
    │
    ▼
SafetyGate.Check(ctx, intent)
    │
    ├─ Check 1: Kill-Switch
    │  ├─ Read EXECUTION_CONTROL.global from NATS KV (2s timeout)
    │  ├─ status=halted → Verdict{Allowed:false, Reason:"kill_switch"}
    │  ├─ status=active → proceed to Check 2
    │  └─ KV unreachable → proceed to Check 2 (fail-open)
    │
    ├─ Check 2: Staleness Guard
    │  ├─ intent.Timestamp vs time.Now()
    │  ├─ age > staleness_max_age → Verdict{Allowed:false, Reason:"stale"}
    │  └─ age ≤ staleness_max_age → Verdict{Allowed:true}
    │
    ▼
Verdict returned to actor
    ├─ Allowed=true → call VenuePort.SubmitOrder()
    └─ Allowed=false → increment counter, skip
```

### 4.2 Dual Checkpoint Architecture

The kill-switch gate is checked at **two independent points**:

| Checkpoint | Location | Actor | Effect when halted |
|------------|----------|-------|--------------------|
| **Derive-side** | Before publishing to EXECUTION_EVENTS | ExecutionPublisherActor | Event discarded, never enters NATS stream |
| **Execute-side** | Before calling VenuePort.SubmitOrder() | VenueAdapterActor (via SafetyGate) | Intent blocked, acked but not executed |

Both checkpoints operate independently. Even if one is bypassed (e.g., derive
checkpoint missed due to race), the execute checkpoint blocks execution.

### 4.3 Fail-Open Semantics

| Scenario | Gate behavior | Rationale |
|----------|---------------|-----------|
| KV store unreachable (NATS down) | Gate defaults to ACTIVE | Transient failure should not permanently block |
| KV read timeout (>2s) | Gate defaults to ACTIVE | Same |
| KV bucket does not exist | Gate defaults to ACTIVE | Same |
| KV returns unknown status | Gate defaults to ACTIVE | Conservative — unknown ≠ halted |

**Practical mitigation:** If NATS is down, the consumer also cannot receive
events from JetStream, so fail-open is largely academic. The real risk scenario
is a partial NATS failure where JetStream works but KV does not — unlikely but
possible.

---

## 5. Safety Invariants

These invariants MUST hold in all states and transitions:

### 5.1 Activation Invariants

| ID | Invariant | Enforcement |
|----|-----------|-------------|
| AI-1 | Binary with `venue.type=binance_futures_testnet` MUST NOT start without credentials | `LoadCredentials()` exits with error |
| AI-2 | Kill-switch MUST be halted before adapter state changes | Operator checklist (CP-3) |
| AI-3 | Venue-Halted state MUST be verified before enabling | Phase 0 exit criteria |
| AI-4 | Rollback MUST restore Paper-Active state completely | Post-rollback verification checklist |

### 5.2 Runtime Invariants

| ID | Invariant | Enforcement |
|----|-----------|-------------|
| RI-1 | Kill-switch halt blocks ALL execution families | Single global gate design |
| RI-2 | Staleness guard rejects intents older than configured max age | SafetyGate sequential check |
| RI-3 | Venue adapter decorator chain is fully composed before first intent | ExecuteSupervisor initialization |
| RI-4 | Fill events from real venue carry correct correlation/causation chain | VenueAdapterActor preserves metadata |
| RI-5 | Composite reader correctly prioritizes venue fills over paper fills | ORDER BY timestamp DESC LIMIT 1 |

### 5.3 Observability Invariants

| ID | Invariant | Enforcement |
|----|-----------|-------------|
| OI-1 | Every gate check result is reflected in health counters | `processed`, `filled`, `skipped_halt`, `skipped_stale` |
| OI-2 | Kill-switch state changes are auditable | `reason`, `updated_by`, `updated_at` in KV |
| OI-3 | Adapter type is logged at startup | ExecuteSupervisor initialization log |

---

## 6. Boundary Conditions

### 6.1 In-Flight Intent During Halt

**Scenario:** An intent passes the gate check (ACTIVE), then the gate is set to
HALTED, then the intent reaches the venue.

**Behavior:** The intent executes. The gate check is a point-in-time read, not a
lock. There is no drain mechanism.

**Impact:** At most one order cycle worth of intents (typically 1 per symbol)
may execute after halt is issued.

**Mitigation:** This is expected behavior, documented in kill-switch architecture.
Operators should account for it during Phase 1 (single-order enablement).

### 6.2 Binary Crash During Venue-Active

**Scenario:** Execute binary crashes while in Venue-Active state.

**Behavior:**
- In-flight HTTP request to venue may complete or timeout (server-side)
- NATS durable consumer retains position; messages will be redelivered on restart
- Kill-switch state is preserved in KV (independent of binary)
- Restarted binary reads same config → returns to Venue-Active (if gate still active)

**Mitigation:** Operator should set gate to HALTED before investigating crash.
If binary restarts automatically (e.g., systemd), it will resume in same adapter
state but gate check will block if halted.

### 6.3 NATS Outage During Venue-Active

**Scenario:** NATS becomes unavailable while binary is in Venue-Active state.

**Behavior:**
- Consumer cannot receive new events (no new intents arrive)
- Gate check fails open (intents that somehow arrive would proceed)
- Fill publisher cannot publish fill events to NATS
- Health counters show no new activity

**Mitigation:** NATS outage is self-limiting — no events arrive, so no
execution occurs. Operator should investigate and consider rollback if outage
is extended.

### 6.4 ClickHouse Outage During Venue-Active

**Scenario:** ClickHouse becomes unavailable while real fills are being produced.

**Behavior:**
- Fills still execute at venue (execution is independent of persistence)
- Writer consumer cannot INSERT (events queue in NATS with MaxDeliver retry)
- Gateway composite queries return stale or empty results
- When ClickHouse recovers, writer catches up from NATS position

**Mitigation:** Fill data is not lost — NATS retains events until acked.
Operator should monitor writer catch-up after ClickHouse recovery.

---

## 7. State Verification Commands

Operators can verify the current activation posture at any time:

```bash
# Check adapter state (from execute binary logs)
# Look for: "venue adapter initialized" with type

# Check gate state
curl -s http://localhost:${GATEWAY_PORT}/execution/control | jq .

# Check credential state (existence only, never log values)
env | grep -c MF_VENUE_BINANCE_FUTURES_TESTNET

# Check composite state (are real fills appearing?)
curl -s "http://localhost:${GATEWAY_PORT}/analytical/composite/chains?source=binancef&symbol=btcusdt&timeframe=60" | jq '.data | length'

# Check health counters
# (via structured logs or health endpoint, depending on binary configuration)
```

---

## 8. Non-Goals

| # | Non-goal | Rationale |
|---|----------|-----------|
| NG-S-1 | State persistence across restarts (beyond KV) | Binary reads config at startup; gate is in KV |
| NG-S-2 | State machine enforcement in code | States are emergent from three independent dimensions |
| NG-S-3 | Automated state transition triggers | Manual control is correct for testnet |
| NG-S-4 | State dashboard or UI | Log-based observation sufficient for testnet |
| NG-S-5 | Per-symbol state dimensions | Global gate is intentional design (NG-9) |
| NG-S-6 | Drain semantics for halt | Gate-check design is proven and documented |

---

## 9. Document Governance

This document is the **canonical state model** for venue activation. It was
established in S338 and is the authoritative reference for activation state
questions in S339–S342.

**Related documents:**
- [Activation Policy — Rollout and Rollback Model](activation-policy-rollout-and-rollback-model.md)
- [Venue Activation Wave Charter](venue-activation-wave-charter-and-scope-freeze.md)
- [Kill-Switch Live and Canonical Smoke](kill-switch-live-and-canonical-smoke-live-stack.md)
