# S332 — Live Stack Integration Wave Charter and Scope Freeze

> **Type:** Charter / Scope freeze
> **Predecessor:** S331 — Production Wiring Tranche Evidence Gate (FULL CLOSURE)
> **Wave:** Live Stack Integration (S332–S336)

---

## 1. Executive Summary

S331 closed the Production Wiring Tranche with FULL CLOSURE: all 4 charter items
delivered, 9 invariants held, 202 tests passing, zero regressions. The closure
shifted the evidence frontier from "components not composed" to "composition not
exercised against live infrastructure."

S332 opens the Live Stack Integration Wave — a verification wave, not a feature
wave. It freezes a small, disciplined scope of four blocks aimed at proving the
composed venue execution pipeline against running NATS infrastructure: consumer
flow, fill round-trip, kill-switch, and a formal gate.

No new domain capabilities are introduced. No mainnet, no multi-venue, no OMS.

## 2. State Assessment at Entry

### 2.1 What S331 Proved

| Aspect | Status | Evidence |
|--------|--------|----------|
| Venue composition | READY | 26+ composition tests, zero regressions |
| Retry orchestration | READY | 27 unit tests + SC + VP suites |
| Post-200 recovery | READY | 9 reconciliation tests, INV-REC-1 held |
| Kill-switch integration | READY | 2-level architecture, KV store connected, halt tests pass |
| Error classification | READY | 10 venue-specific tests, 22 subtests, Binance mapping complete |
| Observability hooks | READY | Structured logs + tracker counters wired |
| NATS infrastructure | COMPLETE | Consumer, publisher, KV store all in code; proven in unit tests |

### 2.2 What S331 Did Not Prove

| Gap | Risk Level | Description |
|-----|------------|-------------|
| R-S330-1 | Medium | Smoke does not exercise live NATS |
| R-S330-2 | Medium | Smoke does not exercise real venue HTTP in composed pipeline |
| Consumer → actor flow | High Priority | Message delivery path not proven against live NATS |
| Control KV live | Medium | Fail-open pattern tested; live KV connection not exercised |
| Fill publisher E2E | Low | Publisher wired; round-trip not proven against live stream |

### 2.3 Production-Ready Decorator Chain (Verified)

```
VenueAdapterActor.onIntent()
  └── SafetyGate (staleness + kill switch)
        └── Post200Reconciler(retrySubmitter, queryPort)
              └── RetrySubmitter(adapter)
                    .WithHaltChecker(controlStore)
                    .WithLogger(logger)
                    .WithTracker(tracker)
                    └── BinanceFuturesTestnetAdapter
```

## 3. Wave Charter

### 3.1 Wave Identity

- **Name:** Live Stack Integration
- **Scope:** FROZEN as of S332
- **Purpose:** Prove the composed pipeline against live infrastructure
- **Duration:** 4 stages (S333–S336)

### 3.2 Charter Blocks

| Block | Name | Stage | Governing Questions |
|-------|------|-------|---------------------|
| LSI-1 | NATS Consumer Flow Live | S333 | GQ-1.1–GQ-1.4 |
| LSI-2 | Fill Event Round-Trip + Composite Visibility | S334 | GQ-2.1–GQ-2.4 |
| LSI-3 | Kill-Switch Live + Smoke-Live-Stack | S335 | GQ-3.1–GQ-3.5 |
| LSI-4 | Wave Gate | S336 | GQ-4.1–GQ-4.4 |

### 3.3 Block Dependencies

```
LSI-1 (Consumer Flow)
  ├──→ LSI-2 (Fill Round-Trip)     [depends on consumer proof]
  └──→ LSI-3 (Kill-Switch Live)    [depends on NATS connectivity]
            └──→ LSI-4 (Gate)      [depends on all blocks]
```

LSI-2 and LSI-3 may be parallelized if LSI-1 completes early.

## 4. Governing Questions

### Consumer Flow (GQ-1)

- **GQ-1.1:** Does the durable consumer receive events from `EXECUTION_EVENTS` within acceptable latency?
- **GQ-1.2:** Does `onIntent()` execute when the consumer delivers a message?
- **GQ-1.3:** Does the health tracker reflect delivery metrics accurately?
- **GQ-1.4:** Does consumer restart preserve durable state?

### Fill Round-Trip (GQ-2)

- **GQ-2.1:** Does `VenueOrderFilledEvent` appear on the NATS stream after submit?
- **GQ-2.2:** Does the subject follow the canonical pattern with correct routing?
- **GQ-2.3:** Can a downstream consumer deserialize the event without loss?
- **GQ-2.4:** Does the gateway return fill data after persistence?

### Kill-Switch Live (GQ-3)

- **GQ-3.1:** Does the control KV store connect to the live `EXECUTION_CONTROL` bucket?
- **GQ-3.2:** Does `halted` gate state block the next submit?
- **GQ-3.3:** Does `halted` gate state stop an in-progress retry loop?
- **GQ-3.4:** Does returning to `active` restore normal execution?
- **GQ-3.5:** Does KV unavailability trigger fail-open behavior?

### Wave Gate (GQ-4)

- **GQ-4.1:** Is the evidence matrix complete for all blocks?
- **GQ-4.2:** Do all 202+ tests pass?
- **GQ-4.3:** Does `make smoke-live-stack` pass reproducibly?
- **GQ-4.4:** Are residual risks documented?

## 5. Non-Goals (Explicit Exclusions)

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-1 | Mainnet activation | Not validated for real-money execution |
| NG-2 | Multi-venue expansion | Depth-first; single venue must be proven live first |
| NG-3 | OMS (order management) | Large domain requiring its own design phase |
| NG-4 | Portfolio risk management | Depends on OMS and portfolio state |
| NG-5 | Broad dashboards/monitoring UI | Structured logs sufficient for verification |
| NG-6 | Config-driven retry/reconciliation | Hardcoded values sufficient for testnet |
| NG-7 | Runtime architecture redesign | Current architecture is proven and stable |
| NG-8 | New breadth (new domains/scopes) | Wave is depth-first on execution path |

## 6. Invariants Carried Forward

All 9 invariants from the Production Wiring Tranche remain active:

EC-1, EC-3, F-1, F-4, RF-1, PGR-08, INV-REC-1, INV-RC-1, INV-OBS-1.

See [charter document](../architecture/live-stack-integration-wave-charter-and-scope-freeze.md)
for the full invariant table.

## 7. Accepted Risks Carried Forward

R-S322-1, R-S322-2, R-S323-1, R-S325-2, R-S328-1, R-S328-2.

These are explicitly **not in scope** for this wave.

## 8. Infrastructure Dependencies

| Component | Source | Status |
|-----------|--------|--------|
| NATS JetStream | Docker Compose | Available |
| NATS KV (EXECUTION_CONTROL) | Seed script | Available |
| ClickHouse | Docker Compose | Available |
| Gateway HTTP | Execute binary | Available |
| Binance Futures Testnet | External | Requires credentials |

No new infrastructure is introduced.

## 9. Success Criteria

The wave achieves FULL CLOSURE when:

1. Consumer delivers events to actor in running stack (LSI-1)
2. Fill events complete NATS round-trip with correct serialization (LSI-2)
3. Kill-switch operates against live KV with both states exercised (LSI-3)
4. `make smoke-live-stack` passes reproducibly (LSI-3)
5. All 202+ tests remain green (LSI-4)
6. Evidence matrix complete with no BLOCKED items (LSI-4)

## 10. Preparation for S333

S333 (LSI-1: Consumer Flow Live) should:

1. **Verify stack prerequisites:** `make up && make seed` brings NATS with the
   required streams and consumers.
2. **Identify the test harness:** Determine whether the consumer flow proof
   is a Go integration test tagged `//go:build integration`, a smoke script,
   or both.
3. **Focus on the message path:** Publish a `PaperOrderSubmittedEvent` to
   `EXECUTION_EVENTS` and verify that `VenueAdapterActor.onIntent()` is invoked.
4. **Instrument evidence capture:** Ensure that health tracker counters and
   structured logs provide the evidence needed for the gate evaluation.
5. **Keep the venue adapter in stub mode:** The consumer flow proof does not
   require a real venue submit — a test double or the existing testnet adapter
   in dry-run mode is sufficient.

## 11. Promoted Documents

| Document | Location |
|----------|----------|
| Wave charter and scope freeze | [`docs/architecture/live-stack-integration-wave-charter-and-scope-freeze.md`](../architecture/live-stack-integration-wave-charter-and-scope-freeze.md) |
| Capabilities, questions, non-goals | [`docs/architecture/live-stack-capabilities-questions-and-non-goals.md`](../architecture/live-stack-capabilities-questions-and-non-goals.md) |

## 12. Verdict

**S332 COMPLETE.** The Live Stack Integration Wave is formally open with scope
frozen. Four blocks (LSI-1 through LSI-4) are chartered across stages S333–S336.
Non-goals are explicit. Governing questions define exit criteria for each block.
The wave targets verification of existing capabilities against live infrastructure,
not new feature development.

---

| Field | Value |
|-------|-------|
| Stage | S332 |
| Type | Charter / Scope freeze |
| Verdict | COMPLETE |
| Wave opened | Live Stack Integration (S333–S336) |
| Scope | FROZEN |
| Next stage | S333 — NATS Consumer Flow Live |
