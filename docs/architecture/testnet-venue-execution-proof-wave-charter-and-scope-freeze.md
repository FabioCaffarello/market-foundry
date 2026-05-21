# Testnet Venue Execution Proof Wave — Charter and Scope Freeze

**Wave:** Testnet Venue Execution Proof
**Charter stage:** S389
**Date frozen:** 2026-03-22
**Predecessor:** OMS Foundation Wave (S382–S388, PASSED — CONDITIONAL)

---

## 1. Wave Identity

This wave proves that the canonical order lifecycle, proven at unit and
integration test level in the OMS Foundation Wave, behaves correctly when
exercised against **real Binance Futures testnet responses**.

The wave is narrow by design: one venue, one execution mode (`venue_live`),
market orders only, single symbol scope. It validates the lifecycle state
machine against real-world venue behavior — acceptance, fill, rejection, and
partial fill — and confirms that the persistence and read-path surfaces
remain consistent under real responses.

---

## 2. Wave Motivation

The OMS Foundation Wave proved the lifecycle state machine exhaustively at
the domain and integration level. However:

- The `venue_live` write-path was tested only with `httptest` stubs, not
  against a real venue HTTP endpoint.
- Rejection scenarios used simulated HTTP errors, not real venue rejection
  responses (e.g., insufficient margin, invalid quantity).
- Partial fills were proven at the model level but never observed from a
  real venue response.
- The persistence read-path (KV + HTTP + ClickHouse) was validated with
  synthetic events, not events produced by real venue interaction.
- The `sent` status was never exercised end-to-end (reserved for async
  protocols, but worth confirming the transition path).

This wave closes those gaps by running the proven pipeline against the
Binance Futures testnet, collecting real evidence, and confirming lifecycle
fidelity.

---

## 3. Capability Target

At wave closure, the system must demonstrate:

| ID | Capability | Acceptance |
|---|---|---|
| **TV-C1** | Real venue acceptance lifecycle | `submitted → accepted → filled` observed with real testnet response |
| **TV-C2** | Real venue fill lifecycle | Fill records carry real `avgPrice`, `executedQty`, `cumQuote` from testnet |
| **TV-C3** | Real venue rejection lifecycle | `submitted → rejected` observed with real testnet rejection (e.g., insufficient margin, invalid params) |
| **TV-C4** | Real venue partial fill lifecycle | `submitted → accepted → partially_filled → filled` observed or structurally proven feasible with testnet |
| **TV-C5** | Lifecycle invariant fidelity under real responses | All 8 invariant categories (ST, TERM, FR, QM, SM, SAFE, CORR, FINAL) hold with real venue data |
| **TV-C6** | Persistence consistency under real responses | KV projection, HTTP query, and ClickHouse row agree on terminal state after real venue interaction |
| **TV-C7** | Rejection event auditability under real responses | `VenueOrderRejectedEvent` published with real venue rejection code, reason, and correlation chain intact |
| **TV-C8** | Post-200 reconciliation under real conditions | Body-read-failure-after-200 recovery path exercised or confirmed structurally sound |
| **TV-C9** | Compose-level E2E with real testnet | Full pipeline (derive → execute → store) runs in compose with `venue_live` mode against testnet |
| **TV-C10** | OMS read-path auditability under real responses | `ExecutionStatusReply` returns correct composite status (intent + result + rejection + gate) after real venue interaction |

---

## 4. Wave Blocks and Stage Order

| Stage | Block | Description |
|---|---|---|
| **S389** | Charter | Wave charter and scope freeze (this document) |
| **S390** | Real venue acceptance/fill lifecycle proof | Prove `submitted → accepted → filled` against real testnet; validate fill records with real prices/quantities |
| **S391** | Real venue rejection lifecycle proof | Prove `submitted → rejected` against real testnet with real rejection causes; validate rejection event contract |
| **S392** | Partial fill and lifecycle projection proof | Prove partial fill path against testnet or structurally demonstrate feasibility; validate quantity monotonicity |
| **S393** | OMS lifecycle read-path and auditability under real responses | Prove KV, HTTP, and ClickHouse surfaces agree after real venue interactions; close RG-1 (ClickHouse rejection writer) |
| **S394** | Compose-level E2E testnet pipeline proof | Full compose stack with real testnet; smoke scripts; sustained operation proof |
| **S395** | Evidence gate | Wave closure ceremony with evidence matrix and verdict |

---

## 5. Entry Preconditions

All preconditions are met:

| Precondition | Status | Evidence |
|---|---|---|
| Seven-state lifecycle proven | MET | S383: 49/49 transition pairs, 8 invariant categories |
| Write-path per mode proven | MET | S385: 19 integration tests across dry_run/paper/venue_live |
| Rejection event path proven | MET | S386: 19 tests at domain, actor, registry layers |
| Persistence read-path proven | MET | S387: KV + HTTP consistent, ClickHouse fills wired |
| BinanceFuturesTestnetAdapter exists | MET | Production code in `internal/application/execution/` |
| Post-200 reconciliation exists | MET | S322: `Post200Reconciler` decorator |
| Retry policy exists | MET | S325: `RetrySubmitter` with backoff and exhaustion |
| Configuration validation exists | MET | FC-1 through FC-9 in `settings/schema.go` |
| Safety gates exist | MET | Kill switch + staleness guard, S344 |
| Activation surface exists | MET | 3D mode computation in `execution/activation.go` |

---

## 6. Scope Freeze Rules

### What enters the wave

1. Real testnet HTTP interactions via `BinanceFuturesTestnetAdapter`.
2. Lifecycle state transitions validated against real venue responses.
3. Fill record fidelity: real prices, quantities, fees from testnet.
4. Rejection event fidelity: real venue rejection codes and reasons.
5. Partial fill observation or structural feasibility proof.
6. Persistence surface consistency under real event data.
7. Compose-level E2E proof with `venue_live` mode.
8. ClickHouse rejection writer wiring (RG-1 closure).
9. OMS-specific compose smoke script (RG-7 closure).

### What does NOT enter the wave

See companion document:
[`testnet-venue-execution-capabilities-questions-and-non-goals.md`](testnet-venue-execution-capabilities-questions-and-non-goals.md)

### Scope amendment protocol

Any scope addition requires:

1. Written justification linking to a governing question.
2. Proof that the addition does not violate any non-goal.
3. Explicit acknowledgment in the stage report that introduced it.

---

## 7. Risk Register

| Risk | Likelihood | Mitigation |
|---|---|---|
| Testnet API unavailability | Medium | Tests should be idempotent and retriable; smoke scripts should tolerate transient 5xx |
| Partial fills hard to trigger on testnet | High | Accept structural proof if real partial fill cannot be reliably triggered; document limitation |
| Testnet rate limits | Medium | Single-symbol, low-frequency execution; retry policy already handles 429/418 |
| Credential management | Low | Environment variables only; never committed; documented in setup instructions |
| Testnet behavior diverges from mainnet | Accepted | This wave explicitly targets testnet only; mainnet is a non-goal |

---

## 8. Success Criteria for Wave Closure

The evidence gate (S395) will evaluate:

1. **10/10 capabilities at FULL or SUBSTANTIAL** — no PARTIAL or PENDING.
2. **All governing questions ANSWERED or SUBSTANTIAL** — no UNANSWERED.
3. **Zero non-goal violations.**
4. **Zero regressions** in existing test suites.
5. **All residual gaps from OMS Foundation (RG-1, RG-7) closed** or
   explicitly deferred with justification.

Verdict options: `PASSED`, `PASSED — CONDITIONAL`, `FAILED`.

---

## 9. Links

- Predecessor charter: [`oms-foundation-wave-charter-and-scope-freeze.md`](oms-foundation-wave-charter-and-scope-freeze.md)
- Predecessor gate: [`oms-foundation-evidence-gate.md`](oms-foundation-evidence-gate.md)
- Capabilities and non-goals: [`testnet-venue-execution-capabilities-questions-and-non-goals.md`](testnet-venue-execution-capabilities-questions-and-non-goals.md)
- Canonical order model: [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md)
- Lifecycle invariants: [`order-lifecycle-invariants-transitions-and-boundaries.md`](order-lifecycle-invariants-transitions-and-boundaries.md)
- Stage report: [`../stages/stage-s389-testnet-venue-execution-proof-charter-report.md`](../stages/stage-s389-testnet-venue-execution-proof-charter-report.md)
