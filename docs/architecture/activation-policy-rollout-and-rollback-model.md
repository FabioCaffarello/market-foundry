# Activation Policy — Rollout and Rollback Model

> **Authority:** S338 · **Wave:** Venue Activation (S337–S342)
> **Predecessor:** S337 — Venue Activation Charter (COMPLETE)
> **Scope:** Canonical policy for controlled venue activation, rollout, and rollback

---

## 1. Purpose

This document defines the **canonical activation policy** for transitioning the
execute binary from `paper_simulator` to `binance_futures_testnet`. It covers
rollout sequencing, rollback procedures, halt semantics, and operator
responsibilities.

The policy exists to prevent **opportunistic, ambiguous, or irreversible**
activation. Every activation must be deliberate, observable, and reversible.

---

## 2. Activation Model

### 2.1 What Activation Means

Activation is a **configuration-driven binary restart** that changes the venue
adapter from paper simulator to a real venue adapter. It is NOT:

- A runtime toggle (no hot-reload)
- An automated feature flag (no gradual percentage rollout)
- A permanent state transition (always reversible via config change)

### 2.2 Three Layers of Activation

| Layer | What it activates | Control mechanism | Reversibility |
|-------|-------------------|-------------------|---------------|
| **Capability activation** | Venue adapter code path exists and compiles | Code deployment | Irrelevant — code is always present |
| **Flow activation** | Execute binary starts with real venue adapter | Configuration (`venue.type`) + binary restart | Full — restart with `paper_simulator` |
| **Environment activation** | Credentials injected, testnet API reachable | Environment variables + network access | Full — remove credentials, restart |

All three layers must be satisfied for real venue execution. Removing any one
layer reverts to paper-only operation.

### 2.3 Activation is NOT Enablement

| Concept | Meaning |
|---------|---------|
| **Activation** | Binary starts with real venue adapter |
| **Enablement** | Kill-switch is in `active` state, allowing intents to reach venue |
| **Execution** | An intent actually reaches the venue and produces a fill |

A binary can be **activated** (real adapter loaded) but **not enabled**
(kill-switch halted). This is the recommended initial state for any activation.

---

## 3. Pre-Activation Checklist

Every activation attempt MUST satisfy all items before the binary is started
with `venue.type = binance_futures_testnet`.

### 3.1 Infrastructure Readiness

| # | Check | How to verify | Failure action |
|---|-------|---------------|----------------|
| PA-1 | NATS JetStream running and healthy | `nats server check jetstream` | Do not activate |
| PA-2 | ClickHouse running and accepting writes | Writer binary health endpoint | Do not activate |
| PA-3 | Gateway running and serving HTTP | `curl /health` on gateway port | Do not activate |
| PA-4 | EXECUTION_EVENTS stream exists with correct config | `nats stream info EXECUTION_EVENTS` | Do not activate |
| PA-5 | EXECUTION_FILL_EVENTS stream exists | `nats stream info EXECUTION_FILL_EVENTS` | Do not activate |
| PA-6 | EXECUTION_CONTROL KV bucket exists | `nats kv get EXECUTION_CONTROL global` | Do not activate |

### 3.2 Control Plane Readiness

| # | Check | How to verify | Failure action |
|---|-------|---------------|----------------|
| CP-1 | Kill-switch responds to GET | `curl /execution/control` returns JSON | Do not activate |
| CP-2 | Kill-switch responds to PUT | Set halted, read back, confirm | Do not activate |
| CP-3 | Kill-switch is set to **halted** | Confirm status=halted before activation | Do not activate |

### 3.3 Credential Readiness

| # | Check | How to verify | Failure action |
|---|-------|---------------|----------------|
| CR-1 | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` set | `env \| grep MF_VENUE` (existence only, never log value) | Do not activate |
| CR-2 | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET` set | Same | Do not activate |
| CR-3 | Testnet API reachable | `curl https://testnet.binancefuture.com/fapi/v1/ping` | Do not activate |

### 3.4 Operator Readiness

| # | Check | How to verify |
|---|-------|---------------|
| OR-1 | Operator has terminal access to execute binary | Can send SIGTERM |
| OR-2 | Operator has HTTP access to kill-switch | Can PUT /execution/control |
| OR-3 | Operator has access to logs | Can tail execute binary stdout |
| OR-4 | Rollback procedure reviewed | Operator confirms understanding |

---

## 4. Rollout Sequencing

### 4.1 Phase Model

Rollout proceeds through three mandatory phases. Each phase has explicit entry
criteria, duration, and exit criteria. No phase may be skipped.

```
Phase 0: Halted Activation
  ↓  binary started with real adapter, kill-switch halted
  ↓  verify: adapter initialized, credentials loaded, no orders submitted

Phase 1: Single-Order Enablement
  ↓  kill-switch set to active
  ↓  single derive cycle produces one intent
  ↓  verify: order submitted, fill received, persisted, visible in gateway
  ↓  kill-switch set to halted immediately after

Phase 2: Observation Window
  ↓  kill-switch set to active
  ↓  pipeline runs for observation period (operator-determined, minimum 5 minutes)
  ↓  verify: fills accumulate, no errors, counters healthy
  ↓  operator decides: continue or rollback
```

### 4.2 Phase 0 — Halted Activation

**Purpose:** Prove the binary starts correctly with real adapter without
executing any orders.

**Entry criteria:**
- Pre-activation checklist PA-1 through OR-4 PASS
- Kill-switch confirmed halted (CP-3)

**Procedure:**
1. Start execute binary with `venue.type = binance_futures_testnet`
2. Observe startup logs: adapter type logged, credentials loaded
3. Confirm no HTTP requests to venue API (intents blocked by kill-switch)
4. Confirm health counters: `processed=0`, `filled=0`, `skipped_halt≥0`

**Exit criteria:**
- Binary running with real adapter
- Zero venue API calls made
- Kill-switch confirmed halted via HTTP GET

**Duration:** Until operator confirms all exit criteria.

### 4.3 Phase 1 — Single-Order Enablement

**Purpose:** Prove exactly one order round-trips through the real venue.

**Entry criteria:**
- Phase 0 exit criteria met
- Derive pipeline has at least one pending intent (or will produce one)

**Procedure:**
1. Set kill-switch to active: `PUT /execution/control {"status":"active"}`
2. Wait for one fill event (observe logs or query composite surface)
3. Set kill-switch to halted immediately: `PUT /execution/control {"status":"halted"}`
4. Verify fill in ClickHouse via gateway composite endpoint
5. Verify fill status = `filled` (not `paper_filled`)

**Exit criteria:**
- Exactly one real fill observed
- Fill visible in gateway composite query
- Kill-switch confirmed halted
- No errors in logs

**Duration:** As short as possible. Operator should pre-stage the halt command.

### 4.4 Phase 2 — Observation Window

**Purpose:** Observe pipeline behavior with real venue under sustained operation.

**Entry criteria:**
- Phase 1 exit criteria met
- Operator has reviewed Phase 1 fill data and confirmed no anomalies

**Procedure:**
1. Set kill-switch to active
2. Monitor logs, health counters, and gateway surface continuously
3. Check for: venue HTTP errors, timeout rate, staleness rejections, fill latency
4. At any sign of anomaly: execute rollback (Section 5)

**Exit criteria:**
- Observation period completed without critical anomalies
- Fill count matches expected derive output rate
- Error counters remain at zero or acceptable threshold
- Operator makes explicit decision: continue or rollback

**Duration:** Minimum 5 minutes. Extended observation (24h+) is a
Production Readiness concern (out of scope for this wave).

---

## 5. Rollback Procedures

### 5.1 Rollback Triggers

Any of the following conditions MUST trigger immediate rollback:

| # | Trigger | Severity |
|---|---------|----------|
| RB-1 | Venue API returns persistent errors (>3 consecutive) | CRITICAL |
| RB-2 | Fill events not appearing in ClickHouse after submit | CRITICAL |
| RB-3 | Unexpected order behavior (wrong side, wrong quantity) | CRITICAL |
| RB-4 | Credential rejection by venue | CRITICAL |
| RB-5 | Execute binary crash or panic | CRITICAL |
| RB-6 | NATS connection loss during active execution | HIGH |
| RB-7 | Staleness rejection rate exceeds 50% | HIGH |
| RB-8 | Operator judgment: anything unexpected | OPERATOR |

### 5.2 Immediate Halt (Seconds)

**When:** Any anomaly detected, operator wants to stop execution immediately.

**Procedure:**
```bash
# Step 1: Halt the kill-switch (stops new intents from reaching venue)
curl -X PUT http://localhost:${GATEWAY_PORT}/execution/control \
  -H "Content-Type: application/json" \
  -d '{"status":"halted","reason":"rollback triggered","updated_by":"operator"}'

# Step 2: Confirm halt
curl http://localhost:${GATEWAY_PORT}/execution/control
# Expected: {"status":"halted", ...}
```

**Effect:** New intents are blocked at both checkpoints. In-flight intents
(already past the gate check) may still complete — this is expected and
documented behavior (no drain semantics).

**Limitation:** Halt is a gate, not a drain. Intents that passed the gate
before the halt command was processed will still reach the venue.

### 5.3 Full Rollback (Minutes)

**When:** Operator decides to revert to paper simulator entirely.

**Procedure:**
```bash
# Step 1: Halt (if not already halted)
# (same as Section 5.2)

# Step 2: Stop execute binary
kill -SIGTERM ${EXECUTE_PID}
# Wait for graceful shutdown (consumer closed, actor stopped)

# Step 3: Change configuration
# Set venue.type = paper_simulator in config

# Step 4: Remove credentials from environment (optional but recommended)
unset MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY
unset MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET

# Step 5: Restart execute binary
# Binary starts with paper_simulator adapter

# Step 6: Resume kill-switch
curl -X PUT http://localhost:${GATEWAY_PORT}/execution/control \
  -H "Content-Type: application/json" \
  -d '{"status":"active","reason":"rollback complete, paper mode","updated_by":"operator"}'

# Step 7: Verify paper operation
# Confirm fills arrive with paper_filled status
```

**Duration:** Under 2 minutes for a prepared operator.

### 5.4 Post-Rollback Verification

After any rollback, the operator MUST verify:

1. Execute binary running with `paper_simulator` adapter (check logs)
2. Kill-switch is `active` (paper mode should run normally)
3. Pipeline producing paper fills (check gateway composite)
4. No residual venue API calls being made
5. Health counters reset (new binary instance)

---

## 6. Halt Semantics

### 6.1 What Halt Does

- Sets `EXECUTION_CONTROL.global.status = "halted"` in NATS KV
- New intents at derive-side publisher: discarded, counter incremented
- New intents at execute-side adapter: blocked by SafetyGate, counter incremented
- Existing active binary: continues running but submits nothing new

### 6.2 What Halt Does NOT Do

- Does NOT drain in-flight intents (no drain semantics)
- Does NOT stop the execute binary
- Does NOT cancel pending HTTP requests to venue
- Does NOT affect NATS consumer acknowledgment
- Does NOT clear pending messages in JetStream

### 6.3 Halt-then-Resume Gap

Between halt and resume, intents continue to arrive via NATS but are blocked by
the safety gate. These intents are acknowledged (acked) by the consumer even
though they were not executed. They will NOT be retried.

This is correct behavior: the intent was received and a decision was made
(block). The derive pipeline may produce a new intent in the next cycle.

### 6.4 Fail-Open Implications

If NATS KV is unreachable during a gate check, the gate defaults to **active**
(fail-open). This means:

- A NATS outage during halted state could allow intents through
- This is an intentional design decision (transient failure should not
  permanently block the pipeline)
- Mitigation: if NATS is down, the consumer also cannot receive events, so
  fail-open is largely academic in practice

---

## 7. Operator Responsibility Matrix

| Responsibility | Who | When |
|----------------|-----|------|
| Pre-activation checklist execution | Operator | Before Phase 0 |
| Kill-switch halt before activation | Operator | Before Phase 0 |
| Binary restart with real adapter config | Operator | Phase 0 start |
| Phase 0 validation | Operator | Phase 0 |
| Kill-switch enable for Phase 1 | Operator | Phase 1 start |
| Kill-switch halt after single fill | Operator | Phase 1 end |
| Phase 1 fill verification | Operator | Phase 1 end |
| Kill-switch enable for Phase 2 | Operator | Phase 2 start |
| Continuous monitoring during Phase 2 | Operator | Phase 2 |
| Rollback decision | Operator | Any time |
| Post-rollback verification | Operator | After rollback |

There is no automated activation. All transitions are manual and deliberate.

---

## 8. Non-Goals

| # | Non-goal | Rationale |
|---|----------|-----------|
| NG-P-1 | Automated rollout (percentage-based, canary) | Manual activation is correct for testnet |
| NG-P-2 | Multi-venue simultaneous activation | Single venue must be proven first |
| NG-P-3 | Per-symbol activation gating | Global gate is intentional (NG-9 from charter) |
| NG-P-4 | Hot-reload of venue configuration | Restart-based activation is simpler and safer |
| NG-P-5 | Credential rotation without restart | Testnet credentials are long-lived |
| NG-P-6 | Automated rollback triggers | Operator judgment required for testnet |
| NG-P-7 | Extended observation (24h+) | Production Readiness Wave scope |
| NG-P-8 | Mainnet activation procedures | Separate risk/compliance review required |

---

## 9. Document Governance

This document is the **canonical activation policy** for the Venue Activation
Wave. It was established in S338 and governs all activation attempts in
S339–S342.

Changes to this policy require a new stage with explicit justification.

**Related documents:**
- [Activation States, Transitions, and Safety Boundaries](activation-states-transitions-and-safety-boundaries.md)
- [Venue Activation Wave Charter](venue-activation-wave-charter-and-scope-freeze.md)
- [Kill-Switch Live and Canonical Smoke](kill-switch-live-and-canonical-smoke-live-stack.md)
