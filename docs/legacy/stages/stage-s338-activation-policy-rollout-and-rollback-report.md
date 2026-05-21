# Stage S338 — Activation Policy, Rollout, and Rollback Report

> **Stage:** S338
> **Wave:** Venue Activation (S337–S342)
> **Block:** VA-1 — Activation Policy and Rollout Model
> **Type:** Policy Definition and State Modeling
> **Predecessor:** S337 — Venue Activation Charter (COMPLETE)
> **Status:** COMPLETE

---

## 1. Executive Summary

Stage S338 delivers the **canonical activation policy** for the Venue Activation
Wave. It defines how the Foundry transitions from paper simulator to real venue
execution in a controlled, observable, and reversible manner.

The stage produces two architecture documents:

1. **Activation Policy — Rollout and Rollback Model**: defines the pre-activation
   checklist, three-phase rollout sequence, rollback triggers and procedures,
   halt semantics, and operator responsibility matrix.

2. **Activation States, Transitions, and Safety Boundaries**: defines the
   three-dimensional state model (adapter × gate × credentials), composite state
   matrix, safety gate evaluation, invariants, and boundary conditions.

Together, these documents eliminate ambiguity about what activation means, how it
proceeds, and how it reverts. They establish the policy framework that S339–S342
will execute against.

---

## 2. What Was Mapped

### 2.1 Existing Activation Controls

The following controls were identified as already implemented and proven:

| Control | Location | Evidence |
|---------|----------|----------|
| Kill-switch (dual-checkpoint) | `safety_gate.go`, `execute_supervisor.go` | S335 CP-FP-2, CP-FP-4 |
| Staleness guard | `staleness_guard.go` | Safety gate tests |
| Venue type configuration | `schema.go` VenueConfig | Compilation + unit tests |
| Credential injection | `LoadCredentials()` | Exit-on-missing behavior |
| Fail-open semantics | `control_kv_store.go` | CG-RT-1, safety gate tests |
| HTTP control surface | Gateway `/execution/control` | S335 smoke Phase 7 |
| Decorator chain composition | `execute_supervisor.go` | Post200Reconciler → RetrySubmitter → adapter |

### 2.2 What Was Missing

| Gap | Resolution in S338 |
|-----|-------------------|
| No pre-activation checklist | Defined: 13 checks across 4 categories (PA, CP, CR, OR) |
| No rollout sequencing | Defined: 3-phase model (Halted Activation → Single-Order → Observation) |
| No rollback triggers | Defined: 8 explicit triggers with severity classification |
| No rollback procedures | Defined: Immediate Halt (seconds) and Full Rollback (minutes) |
| No halt semantics documentation | Defined: what halt does, what it does NOT do, gap behavior |
| No state model | Defined: 3 dimensions, 6 composite states, recommended transition sequence |
| No safety invariants | Defined: 4 activation, 5 runtime, 3 observability invariants |
| No boundary condition analysis | Defined: 4 scenarios (in-flight, crash, NATS outage, ClickHouse outage) |
| No operator responsibility matrix | Defined: 11 responsibilities with timing |

---

## 3. Activation Model Summary

### 3.1 Three Layers

Activation operates across three independent layers:

```
Capability activation  ──  code path exists (always true)
Flow activation        ──  binary config (venue.type)
Environment activation ──  credentials present + API reachable
```

All three must be satisfied for real execution. Removing any one reverts to
paper-only operation.

### 3.2 Three-Phase Rollout

```
Phase 0: Halted Activation     ──  binary starts with real adapter, gate halted
Phase 1: Single-Order          ──  one fill proves the full path
Phase 2: Observation Window    ──  sustained operation under monitoring
```

Each phase has explicit entry criteria, procedures, and exit criteria. No phase
may be skipped.

### 3.3 Composite State Matrix

| Posture | Adapter | Gate | Creds | Orders reach venue? |
|---------|---------|------|-------|---------------------|
| Paper-Active | PAPER | ACTIVE | * | No |
| Paper-Halted | PAPER | HALTED | * | No |
| Venue-Halted | VENUE | HALTED | PRESENT | No |
| **Venue-Active** | **VENUE** | **ACTIVE** | **PRESENT** | **YES** |
| Startup Failure | VENUE | * | ABSENT | Binary exits |

Only **Venue-Active** produces real execution.

### 3.4 Rollback

Two levels of rollback:

| Level | Scope | Time | What it does |
|-------|-------|------|-------------|
| Immediate Halt | Gate only | Seconds | PUT /execution/control halted → new intents blocked |
| Full Rollback | Binary + config | Minutes | Stop binary, change config to paper, restart |

---

## 4. Governing Questions Answered

S337 charter assigned four governing questions to VA-1 (S338):

| GQ | Question | Answer | Evidence |
|----|----------|--------|----------|
| GQ-VA-1.1 | Is there a documented pre-activation checklist? | YES | 13 checks in 4 categories (activation-policy doc §3) |
| GQ-VA-1.2 | Are rollout rules sequenced and phased? | YES | 3-phase model with entry/exit criteria (activation-policy doc §4) |
| GQ-VA-1.3 | Are rollback procedures explicit and executable? | YES | 8 triggers, 2 rollback levels, verification checklist (activation-policy doc §5) |
| GQ-VA-1.4 | Is there a responsibility matrix? | YES | 11 operator responsibilities with timing (activation-policy doc §7) |

---

## 5. Distinction: Capability vs Flow vs Environment Activation

A key contribution of S338 is the explicit separation of three activation layers:

| Layer | What changes | Who controls it | Reversibility |
|-------|-------------|----------------|---------------|
| **Capability** | Venue adapter code compiled into binary | Developer (code change) | N/A — code is always present |
| **Flow** | Execute binary uses real adapter | Operator (config + restart) | Full — restart with paper_simulator |
| **Environment** | Credentials injected, API reachable | Operator (env vars + network) | Full — remove credentials |

This distinction prevents confusion between "the code can do it" (capability),
"the binary is configured to do it" (flow), and "the infrastructure supports it"
(environment).

---

## 6. Safety Analysis

### 6.1 Invariants Established

- **12 invariants** across activation (4), runtime (5), and observability (3)
- All invariants are enforceable through existing mechanisms
- No new code required to enforce these invariants

### 6.2 Boundary Conditions Analyzed

| Condition | Risk | Mitigation |
|-----------|------|------------|
| In-flight intent during halt | ≤1 order per symbol may execute after halt | Expected behavior, documented |
| Binary crash during Venue-Active | KV state preserved, restart may resume | Operator should halt before investigating |
| NATS outage during Venue-Active | Self-limiting (no events arrive) | Monitor and rollback if extended |
| ClickHouse outage during Venue-Active | Fills execute but not persisted | NATS retains events, writer catches up |

### 6.3 Fail-Open Risk Assessment

The fail-open design (gate defaults to ACTIVE on KV failure) is the most
significant safety consideration. S338 documents:

- Why fail-open was chosen (transient failure should not permanently block)
- Why the risk is low in practice (NATS down → no events arrive anyway)
- What the residual risk is (partial NATS failure: JetStream works, KV does not)
- That this risk is accepted and documented, not deferred

---

## 7. Limits and Non-Goals

### 7.1 What S338 Does NOT Deliver

| Item | Why not |
|------|---------|
| Runtime activation surface validation | S339 scope (VA-2) |
| Smoke test extension for real venue | S340 scope (VA-3) |
| Actual activation execution | S341 scope (VA-4) |
| Evidence gate evaluation | S342 scope (VA-5) |
| Automated rollout or canary deployment | Non-goal (NG-P-1) |
| Multi-venue activation | Non-goal (NG-P-2) |
| Per-symbol gating | Non-goal (NG-P-3, NG-9 from charter) |
| Drain semantics for halt | Non-goal (NG-S-6) |
| Mainnet activation procedures | Non-goal (NG-P-8) — separate risk review required |
| Extended observation (24h+) | Production Readiness Wave scope |

### 7.2 Guard Rail Compliance

| Guard rail | Status |
|------------|--------|
| No venue activation in this stage | HELD — policy only, no execution |
| No observability platform opened | HELD — existing counters and logs only |
| No SRE program inflation | HELD — operator checklist, not runbook system |
| No testnet/production confusion | HELD — explicit distinction in state model |

---

## 8. Risks and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Operator skips pre-activation checklist | Medium | High — uncontrolled activation | Checklist is mandatory in policy; S340 smoke will validate |
| Fail-open allows unintended execution during partial NATS failure | Low | Medium — limited number of orders | Documented as accepted risk; self-limiting in practice |
| In-flight intents execute after halt | Certain (by design) | Low — at most 1 per symbol per cycle | Documented behavior; operators trained to expect it |
| Rollback takes longer than expected | Low | Medium — extended real execution | Full rollback is <2 min for prepared operator; immediate halt is seconds |
| Policy docs become stale as S339–S342 discover new constraints | Medium | Low — policy can be amended | New stage required for policy changes |

---

## 9. Preparation for S339

S339 (Canonical Activation Surface and Runtime Controls) can proceed with the
following inputs from S338:

### 9.1 What S339 Should Validate

1. **Credential injection works end-to-end**: binary starts with real adapter when
   credentials are present, exits with error when absent
2. **Decorator chain initializes correctly**: Post200Reconciler → RetrySubmitter →
   BinanceFuturesTestnetAdapter composes and starts
3. **Kill-switch operates identically**: halt/resume cycle works the same with
   real adapter as with paper
4. **Staleness guard operates identically**: stale intents rejected regardless of
   adapter type
5. **Startup validation**: binary logs adapter type, credential presence (not values)

### 9.2 What S339 Should Use from S338

- Pre-activation checklist as the **entry gate** for runtime validation
- Composite state matrix as the **test scenario generator** (test each valid state)
- Safety invariants as the **acceptance criteria** for runtime tests
- Boundary conditions as the **edge case test catalog**

### 9.3 What S339 Should NOT Do

- Actually activate against real venue (that is S341)
- Modify the activation policy (requires new stage)
- Add new state dimensions or transition paths
- Open observability platform or dashboards

---

## 10. Deliverables

| # | Deliverable | Path | Status |
|---|-------------|------|--------|
| 1 | Activation Policy — Rollout and Rollback Model | `docs/architecture/activation-policy-rollout-and-rollback-model.md` | DELIVERED |
| 2 | Activation States, Transitions, and Safety Boundaries | `docs/architecture/activation-states-transitions-and-safety-boundaries.md` | DELIVERED |
| 3 | Stage S338 Report (this document) | `docs/stages/stage-s338-activation-policy-rollout-and-rollback-report.md` | DELIVERED |

---

## 11. Classification

**COMPLETE** — all governing questions answered, all deliverables produced, all
guard rails held. S339 is unblocked.
