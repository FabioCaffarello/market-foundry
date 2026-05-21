# Stage S382 — OMS Foundation Charter Report

## Executive Summary

Stage S382 opens the **OMS Foundation Wave** (S382–S387), the natural
vertical continuation of the pipeline after the Exchange Listening & Dry-Run
Foundation wave closed with an UNCONDITIONAL verdict at S381.

This stage defines scope, freezes it, and orders the next five stages.
No code was written.  No architecture was changed.  The deliverable is
alignment: what the wave must prove, what it must not touch, and in what
order.

## Context

The Exchange Listening & Dry-Run Foundation wave (S376–S381) proved:

- Live WebSocket ingestion from Binance Futures mainnet
- Full derive pipeline (evidence → signal → decision → strategy → risk) on live data
- Dry-run execution with four independent fail-closed layers
- Multi-binary orchestration across derive, execute, store, gateway
- ClickHouse persistence of live-sourced data
- Sustained 5+ minute stability under live market data

Two gaps remain relevant to OMS:

| Gap | Description | Disposition |
|---|---|---|
| G1 | Dry-run fills use Price: "0" | Closed by this wave (S383) |
| G10 | Single fill shape (no partials) | Closed by this wave (S385) |

The domain model for orders exists since S309 (ExecutionIntent, seven-state
lifecycle, FillRecord, ControlGate).  The write-path decorators exist since
S321–S346 (SafetyGate, DryRunSubmitter, RetrySubmitter).  The read-path
exists since S370–S375 (KV + ClickHouse + HTTP).

What is missing is the **composition proof**: evidence that these pieces
work together as a coherent OMS lifecycle under live market data.

## Wave Definition

| Property | Value |
|---|---|
| Wave name | OMS Foundation |
| Stages | S382 (charter) → S383 → S384 → S385 → S386 → S387 (gate) |
| Predecessor | Exchange Listening & Dry-Run Foundation (S376–S381) |
| Primary objective | Prove the existing order primitives compose into a coherent OMS lifecycle |
| Scope status | Frozen |

## Wave Blocks

| # | Stage | Block | What it proves |
|---|---|---|---|
| 0 | S382 | Charter and scope freeze | This document |
| 1 | S383 | Canonical order model and lifecycle state machine proof | Transition invariants, terminal finality, fill consistency, price realism (closes G1) |
| 2 | S384 | Write-path integration across dry_run / paper / venue_live | Composed write-path correctness per mode; safety gate enforcement |
| 3 | S385 | Order tracking, persistence, and read-path foundation | KV + ClickHouse + HTTP agree on terminal state; fill model completeness (closes G10) |
| 4 | S386 | End-to-end OMS foundation proof | Compose smoke: live data → full lifecycle → queryable terminal state |
| 5 | S387 | OMS foundation evidence gate | Formal gate ceremony with verdict |

## Governing Questions

| ID | Question | Target |
|---|---|---|
| OMS-Q1 | Does the seven-state lifecycle enforce all S309 invariants? | S383 |
| OMS-Q2 | Can dry-run fills carry realistic prices without external API? | S383 |
| OMS-Q3 | Does the composed write-path produce correct transitions per mode? | S384 |
| OMS-Q4 | Do safety gates block correctly regardless of mode? | S384 |
| OMS-Q5 | Do KV, ClickHouse, and HTTP agree on terminal state? | S385 |
| OMS-Q6 | Is the fill model sufficient without schema extension? | S385 |
| OMS-Q7 | Can the full lifecycle execute end-to-end with live triggers? | S386 |
| OMS-Q8 | Is the correlation chain intact from strategy to query? | S386 |
| OMS-Q9 | Does the system maintain consistency under sustained live operation? | S386 |

## Non-Goals (Summary)

| ID | Non-goal |
|---|---|
| NG-1 | Full OMS (order book, amendments, routing, allocation) |
| NG-2 | Portfolio risk and position tracking |
| NG-3 | Multi-venue |
| NG-4 | Mainnet trading |
| NG-5 | Advanced order types (limit, stop, OCO) |
| NG-6 | Order amendments and cancellations |
| NG-7 | Async order lifecycle (WebSocket fills) |
| NG-8 | Dashboards and UI |
| NG-9 | Operational hardening (latency, throughput, backpressure) |
| NG-10 | New binaries, streams, or families |
| NG-11 | Multi-account |
| NG-12 | Historical order search |
| NG-13 | Retry strategy redesign |
| NG-14 | State machine extension |

Full rationale in [OMS Foundation — Capabilities, Questions, and Non-Goals](../architecture/oms-foundation-capabilities-questions-and-non-goals.md).

## Capabilities Under Proof

17 capabilities (OMS-C1 through OMS-C17) organized across four blocks:

- **S383 (5):** Lifecycle invariants, terminal finality, fill consistency, quantity monotonicity, price realism
- **S384 (5):** Write-path per mode (dry_run, paper, venue_live), safety gates, correlation chain
- **S385 (4):** KV consistency, ClickHouse consistency, HTTP consistency, fill model completeness
- **S386 (3):** End-to-end lifecycle, correlation traceability, sustained stability

Full capability table in [companion document](../architecture/oms-foundation-capabilities-questions-and-non-goals.md).

## Risk Register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| OMS-R1 | Scope inflation toward full OMS | HIGH | Non-goals frozen; S309 guard rails enforced |
| OMS-R2 | Price realism requires market data in DryRunSubmitter | MEDIUM | Use last-observed price from NATS KV |
| OMS-R3 | Partial fill representation may need state machine extension | LOW | S309 already defines `partially_filled` |
| OMS-R4 | ClickHouse schema migration for fill details | LOW | Codegen-governed |
| OMS-R5 | Compose smoke complexity with live data | MEDIUM | Extend existing S380 smoke scripts |

## Preparation for S383

S383 (Canonical Order Model and Lifecycle State Machine Proof) should:

1. **Read** the S309 architecture docs:
   - `docs/architecture/oms-and-order-lifecycle-charter.md`
   - `docs/architecture/order-lifecycle-semantics-states-and-non-goals.md`
   - `docs/architecture/venue-order-state-model-transitions-and-boundaries.md`

2. **Read** the domain implementation:
   - `internal/domain/execution/execution.go` (lifecycle, transitions, FillRecord)
   - `internal/domain/execution/control.go` (ControlGate)
   - `internal/domain/execution/activation.go` (ActivationSurface)

3. **Read** the DryRunSubmitter:
   - `internal/application/execution/dry_run_submitter.go`
   - `internal/application/execution/dry_run_submitter_test.go`

4. **Implement** exhaustive transition-matrix tests covering every `ValidTransition(from, to)` pair.

5. **Implement** fill invariant tests (FR-1 through FR-9) and terminal invariant tests (TERM-1 through TERM-5).

6. **Close G1:** Modify DryRunSubmitter to use last observed market price instead of zero.

7. **Deliver:**
   - Tests: `internal/domain/execution/s383_lifecycle_invariants_test.go`
   - Architecture doc: `docs/architecture/oms-canonical-order-model-and-lifecycle-proof.md`
   - Stage report: `docs/stages/stage-s383-canonical-order-model-lifecycle-proof-report.md`

## Promoted Documents

| Document | Location |
|---|---|
| OMS Foundation Wave Charter and Scope Freeze | `docs/architecture/oms-foundation-wave-charter-and-scope-freeze.md` |
| OMS Foundation Capabilities, Questions, and Non-Goals | `docs/architecture/oms-foundation-capabilities-questions-and-non-goals.md` |

## Acceptance Criteria Evaluation

| Criterion | Status |
|---|---|
| Wave formally opened with frozen scope | **MET** — charter document delivered and frozen |
| Target capability clear | **MET** — 17 capabilities defined with acceptance criteria |
| Non-goals explicit | **MET** — 14 non-goals with rationale |
| Next stages ordered | **MET** — S383 through S387 defined with block descriptions |

## Verdict

**S382 COMPLETE.** The OMS Foundation Wave is formally open. Scope is frozen.
Proceed to S383.
