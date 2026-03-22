# Production Readiness Assessment Wave — Charter and Scope Freeze

> Formally opens the Production Readiness Assessment Wave.
> Defines scope, blocks, sequencing, and freeze conditions.
> Authority: S347. Predecessor: S346 (Venue Activation Evidence Gate — FULL delivery).

## 1. Wave Identity

| Field | Value |
|-------|-------|
| Wave name | Production Readiness Assessment |
| Phase number | 34 |
| Charter stage | S347 |
| Predecessor wave | Venue Activation Wave (S337–S346) |
| Predecessor verdict | FULL — 18/18 governing questions answered, 34/34 tests pass, 0 regressions |
| Wave type | Assessment and hardening (not new breadth) |
| Scope status | **FROZEN** as of S347 |

## 2. Strategic Context

The Venue Activation Wave proved that the Foundry can:

- model activation as a first-class domain surface;
- transition between paper and venue adapters via configuration;
- enforce runtime gate control with dual checkpoints;
- execute real venue adapter code paths (HMAC, HTTP, response parsing);
- maintain counter integrity over extended observation windows;
- expose activation state via queryable HTTP;
- operate under validated runbook procedures.

All of this was proven against `httptest.Server` mocks and minute-scale observation
windows. The system has never:

- connected to a live Binance testnet endpoint;
- sustained operation over hours;
- been monitored by any alerting system;
- been deployed via automated pipeline;
- handled credential rotation or expiration under load.

The Production Readiness Assessment Wave exists to answer one question:

> **Can the venue activation capability operate sustainably, safely, and
> repeatably under conditions that approximate real testnet operation?**

This is explicitly an assessment wave, not a production launch wave.

## 3. Scope Blocks

The wave is organized into five executable blocks, each with a clear deliverable
and exit criterion.

### PRA-1: Live Testnet Connectivity and Credential Handling

**Objective**: Prove that the venue adapter connects to, authenticates with, and
receives responses from the real Binance Futures testnet API.

**Closes**: DG-1 (live Binance testnet not exercised).

**Deliverables**:
- Integration test exercising real testnet endpoint with real credentials
- Credential loading and validation procedure documented
- Error classification for testnet-specific failure modes (rate limits, auth errors, network timeouts)

**Exit criterion**: At least one round-trip order submission to real testnet with
fill confirmation, gate-halt, and counter consistency verified.

### PRA-2: Endurance and Sustained Activation

**Objective**: Prove that the system remains stable and counter-consistent over
hours-scale operation, not just minutes.

**Closes**: DG-2 (hours-scale soak testing).

**Deliverables**:
- Soak test harness capable of running for configurable duration (minimum 2 hours)
- Counter drift detection at regular checkpoints
- Resource consumption baseline (memory, goroutines, file descriptors)
- Evidence of zero-drift over sustained window

**Exit criterion**: 2+ hour continuous run with zero counter drift, zero error
accumulation, and stable resource consumption.

### PRA-3: Monitoring and Alertability

**Objective**: Define what must be observable and alertable for sustained venue
activation operation.

**Closes**: DG-3 (no automated circuit breaker), DG-6 (no push notifications).

**Deliverables**:
- Monitoring surface definition (which metrics, which thresholds)
- Alert rule catalog (what fires, when, to whom)
- Health endpoint enrichment if needed
- Gate-change notification mechanism (at minimum, log-based; optionally push)

**Exit criterion**: Documented monitoring surface with at least counter-based
and gate-based alert rules defined and testable.

### PRA-4: Deployment and Smoke Automation

**Objective**: Ensure that deployment, startup verification, and smoke testing
can be performed repeatably without manual intervention.

**Deliverables**:
- Deployment checklist hardened into automation (Makefile targets or scripts)
- Smoke script extended for testnet (not just httptest)
- Pre-deployment and post-deployment verification steps automated
- Rollback procedure automated (gate-halt + optional binary restart)

**Exit criterion**: Single-command deployment + smoke + verification cycle
that produces a pass/fail verdict.

### PRA-5: Production Readiness Evidence Gate

**Objective**: Formal closure of the assessment wave with evidence matrix,
gap reconciliation, and verdict.

**Deliverables**:
- Evidence matrix mapping each PRA block to its evidence
- Residual gap catalog (what remains after PRA)
- Verdict: READY (proceed to sustained testnet), NOT READY (what must be fixed), or PARTIAL (what can proceed)
- Recommendation for next wave

**Exit criterion**: All PRA blocks evaluated, verdict issued, next wave recommended.

## 4. Sequencing

The blocks are ordered by dependency:

```
PRA-1 (testnet connectivity)
  └─→ PRA-2 (endurance — requires real testnet from PRA-1)
        └─→ PRA-3 (monitoring — informed by endurance observations)
              └─→ PRA-4 (deployment automation — integrates monitoring)
                    └─→ PRA-5 (evidence gate — evaluates all blocks)
```

Each block maps to one or two stages. Estimated stage count: S348–S355.

## 5. Freeze Conditions

The following conditions lock this charter:

1. **No new domain types** may be introduced unless required by a PRA block.
2. **No new venue adapters** — the wave operates exclusively with BinanceFuturesTestnet.
3. **No architectural redesign** — the decorator pipeline, actor model, and NATS topology are fixed.
4. **No scope expansion** after S347 — new blocks require a new charter.
5. **Block ordering is binding** — PRA-1 must complete before PRA-2 starts.

## 6. Dependencies and Preconditions

| Dependency | Status | Owner |
|------------|--------|-------|
| Binance Futures testnet API key and secret | REQUIRED before PRA-1 | Operator |
| Network access to testnet.binancefuture.com | REQUIRED before PRA-1 | Infrastructure |
| All 21 Go modules passing | VERIFIED at S346 | CI |
| Gate control operational via HTTP | VERIFIED at S345 | Foundry |
| Smoke script operational | VERIFIED at S345 | Foundry |
| Activation surface queryable | VERIFIED at S344 | Foundry |

## 7. Success Criteria for Wave Closure

The wave succeeds (verdict: READY or PARTIAL) when:

- [ ] Live testnet round-trip proven with real credentials
- [ ] 2+ hour endurance run completed with zero drift
- [ ] Monitoring surface defined with actionable alert rules
- [ ] Deployment + smoke cycle automated to single command
- [ ] Evidence gate issued with honest residual gap catalog

The wave fails (verdict: NOT READY) only if PRA-1 (testnet connectivity) cannot
be achieved, since all subsequent blocks depend on it.
