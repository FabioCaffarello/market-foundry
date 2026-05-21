# S389 — Testnet Venue Execution Proof Wave Charter Report

**Stage:** S389
**Type:** Charter and scope freeze
**Date:** 2026-03-22
**Wave:** Testnet Venue Execution Proof
**Predecessor:** S388 (OMS Foundation Evidence Gate — PASSED — CONDITIONAL)

---

## 1. Executive Summary

Stage S389 formally opens the **Testnet Venue Execution Proof Wave** with
frozen scope, governing questions, capability targets, and explicit non-goals.

The OMS Foundation Wave (S382–S388) proved the seven-state order lifecycle
exhaustively at domain and integration test level, closing 41 invariant gaps
and achieving 100% transition coverage. The wave closed with `PASSED —
CONDITIONAL`, with 6/9 governing questions fully answered and 3/9 at
SUBSTANTIAL.

The natural next gap is: **none of this lifecycle has been validated against
real venue responses**. The `venue_live` write-path was tested only with
`httptest` stubs. Rejection scenarios used simulated HTTP errors. Partial
fills were modeled but never observed from a real venue. Persistence surfaces
were validated with synthetic events.

This wave closes those gaps by executing the proven pipeline against the
Binance Futures testnet, collecting real evidence of lifecycle fidelity, and
confirming that the persistence and read-path surfaces remain consistent
under real-world venue behavior.

---

## 2. Foundation Assessment

### 2.1 What the OMS Foundation Wave Proved

| Area | Evidence | Coverage |
|---|---|---|
| Lifecycle state machine | 49/49 transition pairs (10 valid, 39 invalid) | 100% |
| Invariant categories | 8/8 (ST, TERM, FR, QM, SM, SAFE, CORR, FINAL) | 100% |
| Price realism | PriceSource interface + CANDLE_LATEST KV wiring | Production-wired |
| Write-path per mode | 19 integration tests (dry_run, paper, venue_live) | All modes |
| Rejection event path | VenueOrderRejectedEvent + stream + KV projection | Domain → actor → registry |
| Persistence read-path | KV + HTTP consistent; ClickHouse fills wired | 2/3 surfaces |
| Correlation chain | CorrelationID + CausationID through all modes | End-to-end |

### 2.2 Residual Gaps Inherited

| Gap | Severity | Wave disposition |
|---|---|---|
| RG-1: ClickHouse rejection writer not wired | Low | Close in S393 |
| RG-3: Fee realism in dry-run/paper | Low | Remains deferred (NG-12) |
| RG-4: `sent` status never exercised E2E | Low | Remains deferred (NG-5) |
| RG-5: `cancelled` via adapter not tested E2E | Low | Remains deferred (NG-7) |
| RG-6: RejectionProjectionActor lacks unit tests | Low | Close opportunistically |
| RG-7: No OMS-specific compose smoke script | Low | Close in S394 |

### 2.3 Infrastructure Readiness

| Component | Ready | Location |
|---|---|---|
| BinanceFuturesTestnetAdapter | Yes | `internal/application/execution/binance_futures_testnet_adapter.go` |
| Post200Reconciler | Yes | `internal/application/execution/` |
| RetrySubmitter | Yes | `internal/application/execution/` |
| DryRunSubmitter | Yes | `internal/application/execution/dry_run_submitter.go` |
| PaperVenueAdapter | Yes | `internal/application/execution/paper_venue_adapter.go` |
| Kill switch + staleness guard | Yes | `internal/domain/execution/control.go` |
| Activation surface (3D mode) | Yes | `internal/domain/execution/activation.go` |
| Configuration validation (FC-1–FC-9) | Yes | `internal/shared/settings/schema.go` |
| EXECUTION_REJECTION_EVENTS stream | Yes | Consumer spec exists |
| EXECUTION_VENUE_REJECTION_LATEST KV | Yes | Projection actor exists |

---

## 3. Wave Charter

### 3.1 Governing Questions (12)

| ID | Question | Target |
|---|---|---|
| TV-Q1 | Does `venue_live` produce correct lifecycle transitions on real acceptance + fill? | S390 |
| TV-Q2 | Do fill records carry accurate real prices, quantities, and fees? | S390 |
| TV-Q3 | Does the lifecycle transition to `rejected` on real venue rejection? | S391 |
| TV-Q4 | Does `VenueOrderRejectedEvent` carry real venue rejection code and reason? | S391 |
| TV-Q5 | Can partial fill be observed or structurally proven from testnet? | S392 |
| TV-Q6 | Does quantity monotonicity hold under real partial fills? | S392 |
| TV-Q7 | Do KV, HTTP, and ClickHouse agree on terminal state after real interactions? | S393 |
| TV-Q8 | Is the ClickHouse rejection writer wired and producing correct rows? | S393 |
| TV-Q9 | Does the full compose pipeline work in `venue_live` against testnet? | S394 |
| TV-Q10 | Does the system sustain correct behavior over multiple order cycles? | S394 |
| TV-Q11 | Is the correlation chain intact through real venue interactions? | S390, S391 |
| TV-Q12 | Does post-200 reconciliation work under real conditions? | S390 |

### 3.2 Stage Order

| Stage | Description | Depends on |
|---|---|---|
| S389 | Charter and scope freeze | S388 |
| S390 | Real venue acceptance/fill lifecycle proof | S389 |
| S391 | Real venue rejection lifecycle proof | S390 |
| S392 | Partial fill and lifecycle projection proof | S391 |
| S393 | OMS read-path and auditability under real responses | S392 |
| S394 | Compose E2E testnet pipeline proof | S393 |
| S395 | Evidence gate | S394 |

### 3.3 Non-Goals (22)

Fully enumerated in
[`testnet-venue-execution-capabilities-questions-and-non-goals.md`](../architecture/testnet-venue-execution-capabilities-questions-and-non-goals.md).

Key exclusions:
- **NG-1:** No mainnet execution
- **NG-2:** No multi-venue support
- **NG-6:** No full OMS
- **NG-7:** No cancel-order API
- **NG-9:** No lifecycle state machine extension
- **NG-11:** No portfolio risk management
- **NG-19:** No lifecycle redesign

---

## 4. Preparation Recommended for S390

Before S390 begins:

1. **Credential provisioning.** Create or verify Binance Futures testnet API
   key and secret. Ensure the testnet account has sufficient test margin.

2. **Configuration template.** Prepare a `deploy/configs/execute-testnet.jsonc`
   (or equivalent) with:
   - `venue.type = "binance_futures_testnet"`
   - `venue.dry_run = false`
   - `venue.symbol` set to a liquid testnet pair (e.g., `BTCUSDT`)

3. **Gate activation.** Document the HTTP PUT ceremony to activate the
   execution gate for `venue_live` mode.

4. **Environment documentation.** Document the environment variable setup
   for testnet credentials (never committed).

5. **Smoke script skeleton.** Prepare a `scripts/smoke-testnet-venue.sh`
   skeleton that will be fleshed out across stages.

---

## 5. Artifacts Produced

| Artifact | Path |
|---|---|
| Wave charter and scope freeze | `docs/architecture/testnet-venue-execution-proof-wave-charter-and-scope-freeze.md` |
| Capabilities, questions, and non-goals | `docs/architecture/testnet-venue-execution-capabilities-questions-and-non-goals.md` |
| Stage report (this document) | `docs/stages/stage-s389-testnet-venue-execution-proof-charter-report.md` |

---

## 6. Verdict

**S389 COMPLETE.** The Testnet Venue Execution Proof Wave is formally open
with frozen scope. The wave covers stages S390–S395 with 12 governing
questions, 10 capability targets, and 22 explicit non-goals.

---

## 7. Links

- Wave charter: [`../architecture/testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](../architecture/testnet-venue-execution-proof-wave-charter-and-scope-freeze.md)
- Capabilities and non-goals: [`../architecture/testnet-venue-execution-capabilities-questions-and-non-goals.md`](../architecture/testnet-venue-execution-capabilities-questions-and-non-goals.md)
- Predecessor gate: [`../architecture/oms-foundation-evidence-gate.md`](../architecture/oms-foundation-evidence-gate.md)
- Predecessor charter: [`../architecture/oms-foundation-wave-charter-and-scope-freeze.md`](../architecture/oms-foundation-wave-charter-and-scope-freeze.md)
- Canonical order model: [`../architecture/canonical-order-model-and-lifecycle-state-machine.md`](../architecture/canonical-order-model-and-lifecycle-state-machine.md)
