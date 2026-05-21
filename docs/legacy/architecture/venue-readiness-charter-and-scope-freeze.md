# Venue Readiness Wave — Charter and Scope Freeze

> Wave: Phase 30 — Venue Readiness
> Status: **OPEN**
> Charter stage: S306
> Date: 2026-03-21
> Predecessor wave: Phase 29 — Multi-Symbol Operational Scaling (S300–S305, CLOSED)

---

## 1. Strategic Context

The Multi-Symbol Operational Scaling Wave (S300–S305) closed with PASS verdict: symbol isolation proven at every pipeline stage, 71 new tests, zero regressions, zero scope inflation. The architecture is confirmed inherently multi-symbol safe — stateless evaluators, per-instance partition keys, and symbol-scoped queries generalize to N symbols without code changes.

The single largest remaining capability gap between the Foundry and production is **real venue connectivity**. Every other high-value wave — portfolio risk aggregation, operational dashboards, live concurrency hardening — depends on understanding real venue behavior first. The S305 options matrix scored Venue Readiness at 21/25, the highest of five candidates, and explicitly recommended single-front discipline with no concurrent wave.

This charter opens Phase 30 and freezes its scope.

---

## 2. What "Venue Readiness" Means in the Foundry

Venue readiness is **not** building a production trading system. It is the minimum set of capabilities required to replace the paper fill stub (`PaperVenueAdapter`) with a real exchange adapter that:

1. Submits market orders to an exchange testnet via authenticated REST API.
2. Receives fill confirmations (price, quantity, fee, timestamp) from the exchange.
3. Maps venue responses to the existing `ExecutionIntent` lifecycle (submitted → sent → accepted → filled/rejected/cancelled).
4. Persists real fill records through the existing ClickHouse write path.
5. Exposes real execution data through the existing composite read model without schema changes.
6. Operates under the existing safety gate (kill switch + staleness guard) without relaxation.

The Foundry already has:
- A `VenuePort` interface (`internal/application/ports/venue.go`) with `SubmitOrder` contract.
- A `PaperVenueAdapter` that satisfies the interface with instant, zero-price fills.
- A `BinanceFuturesTestnetAdapter` that implements authenticated order submission with fill parsing (S90–S93).
- A `SafetyGate` with kill switch and staleness guard.
- A `ControlGate` domain type with halt/active semantics.
- A complete `ExecutionIntent` domain with lifecycle states, valid transitions, fill records, and partition keys.
- Credential loading (`CredentialSet`) with environment variable binding.

The gap is the **operational envelope around the adapter**: reconciliation, failure handling, fill model validation, and end-to-end proof that real venue data flows correctly through the existing pipeline.

---

## 3. Wave Objective

**Prove that the market-foundry execution pipeline correctly handles real venue order submission, fill reception, and lifecycle management through a single exchange testnet, with full observability, failure containment, and zero regression against the paper baseline.**

"Correctly" means:
- Orders reach the venue and return with real prices, quantities, and fees.
- Fill records are non-simulated and carry venue-sourced timestamps.
- The `ExecutionIntent` lifecycle transitions match venue reality.
- Composite read model returns real execution data without schema changes.
- Failures (network, auth, rate limit, venue rejection) are classified, contained, and observable.
- The existing pipeline (signal → decision → strategy → risk) is unchanged.
- Safety gates remain enforced — no bypasses for venue convenience.

---

## 4. Venue and Adapter Scope

### Target venue: Binance Futures Testnet

| Property | Value |
|----------|-------|
| Exchange | Binance Futures |
| Environment | Testnet (`testnet.binancefuture.com`) |
| Order type | Market orders only |
| Auth | HMAC-SHA256 signed REST API |
| Symbols | btcusdt, ethusdt, solusdt (same as S300–S304) |
| Adapter | `BinanceFuturesTestnetAdapter` (exists, needs hardening) |

### Why testnet, not mainnet?

- Testnet eliminates financial risk while exercising the identical API contract.
- Testnet credentials are free and revocable.
- Testnet validates the full adapter lifecycle (auth, submit, parse, map) without capital exposure.
- Mainnet promotion is a deployment decision, not an architectural one — the adapter code is identical.

### Why single exchange?

- Multi-exchange introduces adapter registry, routing, and normalization concerns that are orthogonal to proving venue readiness.
- Single exchange keeps scope containable at 5–7 stages.
- Multi-venue is a successor wave that builds on proven single-venue infrastructure.

---

## 5. Scope Freeze Rules

1. **No new pipeline stages**. Signal, decision, strategy, and risk remain unchanged. Only the execution layer's venue adapter and its operational envelope are in scope.
2. **No write-side schema changes**. ClickHouse tables remain as-is. Real fill data must conform to the existing `execution_intents` schema.
3. **No new HTTP endpoints**. The existing composite surface is validated with real data, not extended.
4. **No OMS**. No order tracking, order book, position state machine, or order amendment/cancellation flows. The adapter is fire-and-forget with synchronous fill response.
5. **No portfolio-level risk**. Per-symbol risk assessment remains as-is. Cross-symbol exposure management is a separate wave.
6. **No multi-venue**. Single exchange adapter only. No adapter registry, exchange routing, or venue failover.
7. **No new families or symbols**. Existing 3 families (EMA, Trend, Squeeze) and 3 symbols (btcusdt, ethusdt, solusdt).
8. **No operational dashboards**. Observability is validated through existing composite HTTP surface, not Grafana/Prometheus.
9. **No compliance or regulatory work**. No KYC integration, audit trails, or regulatory reporting.
10. **No mainnet deployment**. All venue work targets testnet exclusively. Mainnet promotion is a deployment decision outside this wave.
11. **Market orders only**. No limit orders, stop orders, or conditional order types.
12. **Synchronous fills only**. The adapter waits for the venue response. Asynchronous fill reconciliation (WebSocket user data streams) is a successor capability.

---

## 6. Wave Entry Conditions (All Met)

| Condition | Evidence |
|-----------|----------|
| Multi-symbol isolation proven | S300–S304 — Phase 29 PASS |
| Paper execution pipeline complete | S264–S274 — full lifecycle: intent → submit → fill → persist |
| VenuePort interface defined | `internal/application/ports/venue.go` — `SubmitOrder(ctx, req) → (receipt, *problem)` |
| Binance testnet adapter exists | `internal/application/execution/binance_futures_testnet_adapter.go` — S90–S93 |
| Safety gate operational | `SafetyGate` with kill switch + staleness guard |
| Execution domain model complete | `ExecutionIntent` with lifecycle states, transitions, fill records, partition keys |
| Credential infrastructure exists | `CredentialSet` with env var loading |
| Composite read model operational | 4 endpoints, 5-table composition, symbol-scoped queries |
| Zero regressions in predecessor wave | S305 — confirmed across all dimensions |

---

## 7. Governing Questions

This wave is governed by seven questions (VQ1–VQ7). The wave is complete when all are answerable with evidence.

| ID | Question | Validation Method |
|----|----------|-------------------|
| **VQ1** | Does the adapter successfully submit market orders to Binance Futures testnet and receive fill confirmations? | Integration test against testnet; receipt contains real price, quantity, fee |
| **VQ2** | Does the `ExecutionIntent` lifecycle correctly reflect venue order states (accepted, filled, rejected, cancelled)? | Unit tests mapping all Binance status values; lifecycle transition validation |
| **VQ3** | Do real fill records persist through the existing ClickHouse write path without schema changes? | End-to-end test: submit → fill → persist → query; `Simulated=false` in stored records |
| **VQ4** | Does the composite read model correctly display real execution data through existing endpoints? | Composite chain/funnel/disposition queries return non-simulated fill data |
| **VQ5** | Are venue failures (network, auth, rate limit, rejection) correctly classified and contained without pipeline disruption? | Failure injection tests: timeout, 401, 429, 400; verify pipeline continues for healthy symbols |
| **VQ6** | Does the safety gate (kill switch + staleness) remain enforced for real venue submissions? | Safety gate tests with venue adapter; verify halt blocks real submissions |
| **VQ7** | Does multi-symbol operation with real venue maintain isolation proven in Phase 29? | Multi-symbol venue test: 3 symbols, concurrent submissions, no cross-contamination |

---

## 8. Recommended Stage Sequence

| Stage | Title | Objective | Dependencies |
|-------|-------|-----------|--------------|
| **S306** | Venue Readiness Charter and Scope Freeze (this document) | Open wave, freeze scope, define governing questions | S305 |
| **S307** | Venue Adapter Contract Hardening | Harden `BinanceFuturesTestnetAdapter`: error classification, retry semantics, timeout handling, response validation. Define failure envelope. | S306 |
| **S308** | Fill Model Validation and Lifecycle Proof | Prove real fills map correctly to `ExecutionIntent` lifecycle. Validate all Binance status → domain status mappings. Prove fill records carry real price/qty/fee/timestamp. | S307 |
| **S309** | Venue Execution End-to-End Integration | End-to-end proof: venue submit → fill → persist → composite read. Validate ClickHouse write path with non-simulated fills. No schema changes. | S308 |
| **S310** | Venue Failure Envelope and Containment | Prove failure modes are classified, contained, and observable. Network timeout, auth failure, rate limit, venue rejection — all must be non-fatal to pipeline. | S309 |
| **S311** | Multi-Symbol Venue Isolation Proof | Prove multi-symbol operation with real venue maintains isolation proven in Phase 29. 3 symbols, concurrent venue submissions, no cross-contamination. | S310 |
| **S312** | Venue Readiness Gate and Wave Closure | Evidence gate: all governing questions answered; regression check; wave closure verdict; next-wave recommendation. | S311 |

**Estimated wave size**: 7 stages (S306–S312), consistent with predecessor waves.

---

## 9. Success Criteria

The wave is complete when:

1. The Binance Futures testnet adapter submits real market orders and receives real fills.
2. Fill records contain venue-sourced price, quantity, fee, and timestamp (`Simulated=false`).
3. The `ExecutionIntent` lifecycle transitions match real venue order states.
4. Real execution data flows through the existing ClickHouse write path without schema changes.
5. Composite read model returns correct results for real execution data.
6. Venue failures are classified, contained, and do not disrupt the pipeline for healthy symbols.
7. Safety gates remain enforced — kill switch and staleness guard block real submissions when appropriate.
8. Multi-symbol venue operation maintains isolation proven in Phase 29.
9. All governing questions (VQ1–VQ7) are answerable with evidence.
10. Zero regressions against the S305 baseline.

---

## 10. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Binance testnet instability or downtime | Medium | Medium | Tests must tolerate testnet unavailability; adapter marks unavailable as retryable |
| Testnet rate limiting during test suites | Medium | Low | Respect rate limits; tests use minimal order count; adapter classifies 429 as retryable |
| Fill parsing diverges from documented API | Low | High | Response validation with strict schema; integration tests against real testnet responses |
| Scope creep into OMS or position tracking | Medium | High | Charter freeze rules (section 5) explicitly prevent; gate reviews enforce |
| Scope creep into async fills / WebSocket | Medium | Medium | Synchronous fills only (section 5, rule 12); WebSocket is a successor capability |
| Safety gate bypass for "just testing" | Low | Very High | Tests must prove safety gate is enforced, not bypassed; no test-only relaxation |
| Schema pressure from real fill data | Low | Medium | Existing schema accommodates real data (price, qty, fee fields exist); no new columns |
| Credential exposure in logs or errors | Low | Very High | Adapter already sanitizes (S90); S307 adds explicit credential leak tests |

---

## 11. Dependency Graph

```
S306: Charter (this document)
  └── S307: Adapter contract hardening
       └── S308: Fill model validation
            └── S309: E2E integration
                 └── S310: Failure envelope
                      └── S311: Multi-symbol venue isolation
                           └── S312: Gate and wave closure
```

No stage can be parallelized — each depends on the deliverables of its predecessor. This is intentional: venue integration is sequential by nature (contract → model → integration → failure → scale → gate).

---

## 12. Relationship to Existing Architecture

This wave **replaces one component** of the execution pipeline: the `PaperVenueAdapter` is swapped for `BinanceFuturesTestnetAdapter` behind the existing `VenuePort` interface. Everything upstream (signal, decision, strategy, risk) and downstream (ClickHouse persistence, composite read model) remains unchanged.

- **VenuePort interface** — already defined, already consumed by the execute actor; no changes needed.
- **ExecutionIntent domain** — already has lifecycle states, fill records, partition keys; no changes needed.
- **SafetyGate** — already enforced before `SubmitOrder`; no changes needed.
- **ControlGate** — already operational with halt/active semantics; no changes needed.
- **Composite read model** — already symbol-scoped and fill-aware; real data should flow through without changes.

The wave produces **evidence that the existing architecture handles real venue data**, not new architecture.
