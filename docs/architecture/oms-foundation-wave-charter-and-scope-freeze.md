# OMS Foundation Wave — Charter and Scope Freeze

> Companion document: [OMS Foundation — Capabilities, Questions, and Non-Goals](oms-foundation-capabilities-questions-and-non-goals.md)

## Wave Identity

| Property | Value |
|---|---|
| Wave name | OMS Foundation |
| Charter stage | S382 |
| Predecessor wave | Exchange Listening & Dry-Run Foundation (S376–S381), closed UNCONDITIONAL |
| Scope status | **Frozen** — changes require formal ceremony |

## Strategic Context

Every wave since S337 has built the pipeline vertically toward one goal: a
signal-to-execution system that can safely place orders against live market
data.  The Exchange Listening & Dry-Run Foundation wave (S376–S381) closed
that chapter with an UNCONDITIONAL verdict: live WebSocket ingestion, full
derive pipeline on live data, and four independent fail-closed layers
preventing accidental real trading are all proven.

The remaining gap is the OMS itself.  The domain model exists (S309:
`ExecutionIntent`, seven-state lifecycle, `FillRecord`), the write-path
decorator exists (`DryRunSubmitter`), and the read-path projections exist
(KV + ClickHouse + HTTP).  What does **not** yet exist is end-to-end proof
that these pieces compose into a coherent order lifecycle — from intent
creation through venue submission, fill recording, persistence, and query —
under both dry-run and paper modes, with live market data as the trigger.

This wave closes that gap.  It is **not** an OMS rewrite.  It is the
foundation proof that the existing order primitives compose correctly.

## What the Predecessor Wave Proved

| Capability | Classification | Source |
|---|---|---|
| Live WebSocket ingestion (Binance Futures mainnet) | FULL | S378 |
| Normalization fidelity (string decimal preservation) | FULL | S377 |
| Full derive pipeline on live data (evidence → strategy) | FULL | S378, S380 |
| Dry-run execution by config (fail-closed, four layers) | FULL | S379, S380 |
| Activation surface integrity (12 combos, 1 live gate) | FULL | S377 |
| Read/write path independence | FULL | S378 |
| Sustained stability (5+ min live operation) | FULL | S380 |
| ClickHouse persistence of live-sourced data | FULL | S380 |
| Runtime observability (activation surface queryable) | FULL | S377 |

### Inherited Gaps Relevant to This Wave

| ID | Gap | Severity | Disposition |
|---|---|---|---|
| G1 | Dry-run fills use Price: "0" | LOW | Addressed in this wave (price realism) |
| G10 | Single fill shape (no partials) | LOW | Addressed in this wave (fill model) |
| G6 | No backpressure on ingestion | MEDIUM | Out of scope — operational hardening |

## What This Wave Must Prove

**Primary objective:** The existing order domain model, write-path
decorators, persistence layer, and query surface compose into a coherent
OMS foundation that can track an order from intent to terminal state, with
full auditability, under live market data.

**Secondary objectives:**
- Price realism in dry-run fills (close G1)
- Fill model completeness for paper and venue modes (close G10)
- Order lifecycle state machine exercised end-to-end with live triggers
- Read-path consistency: KV materialization and ClickHouse agree on terminal state
- Correlation chain integrity from strategy event to fill event

## What Already Exists

| Component | Location | Status |
|---|---|---|
| ExecutionIntent domain model | `internal/domain/execution/execution.go` | Proven (S309) |
| Seven-state lifecycle + ValidTransition | `internal/domain/execution/execution.go` | Proven (S309) |
| FillRecord model | `internal/domain/execution/execution.go` | Proven (S309) |
| ControlGate (kill switch) | `internal/domain/execution/control.go` | Proven (S337–S346) |
| ActivationSurface | `internal/domain/execution/activation.go` | Proven (S377) |
| PaperOrderEvaluator | `internal/application/execution/paper_order_evaluator.go` | Proven (S364–S369) |
| DryRunSubmitter | `internal/application/execution/dry_run_submitter.go` | Proven (S379) |
| SafetyGate + StalenessGuard | `internal/application/execution/safety_gate.go` | Proven (S337–S346) |
| RetrySubmitter | `internal/application/execution/retry_submitter.go` | Proven (S321–S326) |
| BinanceFuturesTestnetAdapter | `internal/adapters/exchanges/binancef/` | Proven (S321–S326) |
| PaperVenueAdapter | `internal/adapters/exchanges/paper/` | Proven (S321–S326) |
| NATS execution registry (2 streams, 2 families) | `internal/adapters/nats/natsexecution/registry.go` | Proven (S370–S375) |
| KV materialization + HTTP query | store + gateway binaries | Proven (S370–S375) |
| ClickHouse analytical writer | writer consumer | Proven (S380) |
| Codegen integration (paper_order family) | `codegen/families/paper_order.yaml` | Proven |

## Wave Structure

| Block | Stage(s) | Description |
|---|---|---|
| 0. Charter and scope freeze | S382 | This document. Define scope, non-goals, governing questions |
| 1. Canonical order model and lifecycle state machine proof | S383 | Exercise ValidTransition exhaustively; prove state monotonicity, fill invariants, terminal finality under test; close G1 (price realism in dry-run) |
| 2. Write-path integration across dry_run / paper / venue_live | S384 | Prove the composed write-path (SafetyGate → DryRunSubmitter → RetrySubmitter → VenueAdapter) produces correct lifecycle transitions for each mode |
| 3. Order tracking, persistence, and read-path foundation | S385 | Prove KV materialization, ClickHouse writer, and HTTP query surface agree on terminal state and fill details; close G10 (fill model completeness) |
| 4. End-to-end OMS foundation proof | S386 | Compose smoke: live exchange data → derive → strategy → execute → fill → persist → query, validating full lifecycle under dry-run with live triggers |
| 5. OMS foundation evidence gate | S387 | Formal gate ceremony: evaluate all governing questions, classify capabilities, verify regressions, issue verdict |

## Block Descriptions

### Block 1 — Canonical Order Model and Lifecycle State Machine Proof (S383)

**Objective:** Prove that the seven-state lifecycle model, as implemented in
code, enforces all invariants from S309: transition monotonicity, terminal
finality, fill-status consistency, quantity monotonicity.

**Scope:**
- Exhaustive transition-matrix tests (every valid + invalid pair)
- Fill invariant tests (FR-1 through FR-9 from S309)
- Terminal state invariant tests (TERM-1 through TERM-5)
- Price realism: DryRunSubmitter fills use last observed price (not "0")
- Paper vs. venue fill discrimination (`Simulated` flag)

**Acceptance:** All S309 invariants covered by automated tests; G1 closed.

### Block 2 — Write-Path Integration Across Modes (S384)

**Objective:** Prove the composed write-path produces correct lifecycle
transitions under each execution mode: dry_run=true, paper_simulator, and
venue_live (testnet).

**Scope:**
- DryRunSubmitter: submitted → accepted → filled (simulated, realistic price)
- PaperVenueAdapter: submitted → accepted → filled (simulated, instant fill)
- BinanceFuturesTestnetAdapter: submitted → accepted → filled (venue, real fill)
- Safety gate enforcement: halted gate → no transition past submitted
- Staleness guard: stale intent → no transition past submitted
- Correlation chain preservation across all modes

**Acceptance:** Each mode traverses the expected state path; safety gates
block correctly; no mode leaks into another.

### Block 3 — Order Tracking, Persistence, and Read-Path Foundation (S385)

**Objective:** Prove that the persistence and query layers faithfully
represent order terminal state.

**Scope:**
- KV materialization: latest state reflects terminal status + fills
- ClickHouse writer: execution row contains all fill details
- HTTP query surface: returns consistent view of terminal order
- Fill model completeness: partial fills, full fills, rejections all represented
- Close G10: fill shape covers paper and venue modes

**Acceptance:** KV, ClickHouse, and HTTP agree on terminal state for every
execution mode; G10 closed.

### Block 4 — End-to-End OMS Foundation Proof (S386)

**Objective:** Compose-level smoke test proving the full OMS lifecycle under
live market data.

**Scope:**
- Live Binance aggTrade → derive pipeline → strategy event
- Strategy event → execute binary → DryRunSubmitter → fill event
- Fill event → store (KV + ClickHouse) → gateway query
- Correlation chain: strategy correlation ID traceable through fill to query
- Multi-binary: derive, execute, store, gateway all operational
- Sustained operation: 5+ minutes without state inconsistency

**Acceptance:** End-to-end lifecycle proven under live data; correlation
chain intact; all binaries healthy.

### Block 5 — OMS Foundation Evidence Gate (S387)

**Objective:** Formal ceremony evaluating whether the wave met its charter.

**Scope:**
- Evaluate all governing questions
- Classify all capabilities (FULL / SUBSTANTIAL / PARTIAL / ABSENT)
- Verify zero regressions across all packages
- Catalog residual gaps
- Issue verdict: UNCONDITIONAL / CONDITIONAL / FAIL
- Recommend next ceremony direction

## Acceptance Criteria

| ID | Criterion | Verification |
|---|---|---|
| AC-1 | All S309 lifecycle invariants covered by automated tests | Test suite green, invariant coverage matrix |
| AC-2 | Dry-run fills use realistic prices (G1 closed) | Test + smoke evidence |
| AC-3 | Fill model covers paper + venue + partial fills (G10 closed) | Test evidence |
| AC-4 | Write-path produces correct transitions for all three modes | Integration tests per mode |
| AC-5 | Safety gates block correctly across all modes | Integration tests |
| AC-6 | KV, ClickHouse, and HTTP agree on terminal state | Persistence consistency tests |
| AC-7 | End-to-end lifecycle proven under live data | Compose smoke script |
| AC-8 | Correlation chain intact from strategy to query | Smoke + test evidence |
| AC-9 | Zero regressions across all packages | `go test ./...` green |

## Risk Register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| OMS-R1 | Scope inflation toward full OMS (amendments, cancellations, limit orders) | HIGH | Non-goals frozen; S309 guard rails enforced |
| OMS-R2 | Price realism requires market data access in DryRunSubmitter | MEDIUM | Use last-observed price from NATS KV, not external API call |
| OMS-R3 | Partial fill representation may require state machine extension | LOW | S309 already defines `partially_filled`; exercise existing path |
| OMS-R4 | ClickHouse schema may need migration for fill details | LOW | Codegen-governed; schema change follows family convention |
| OMS-R5 | Compose smoke complexity with live data | MEDIUM | Build on existing S380 smoke scripts; extend, don't rewrite |

## Invariants

| ID | Invariant |
|---|---|
| OMS-I1 | `ExecutionIntent` is the only order abstraction. No new order types introduced. |
| OMS-I2 | The seven-state lifecycle is the only state machine. No new states added. |
| OMS-I3 | Terminal states are absorbing. No transitions out of `filled`, `rejected`, or `cancelled`. |
| OMS-I4 | `DryRunSubmitter` never delegates to the inner adapter. |
| OMS-I5 | Safety gate is checked before every `VenuePort.SubmitOrder()` call. |
| OMS-I6 | `Simulated` flag discriminates paper from venue fills. |
| OMS-I7 | Correlation chain (CorrelationID → CausationID) is preserved end-to-end. |
| OMS-I8 | No new NATS streams or families are introduced. Existing `EXECUTION_EVENTS` and `EXECUTION_FILL_EVENTS` are sufficient. |
| OMS-I9 | No new binaries are introduced. |

## Scope Freeze Notice

This charter is **frozen** as of S382.  The wave structure, block
definitions, acceptance criteria, invariants, and non-goals defined here
are authoritative.  Any change to scope requires a formal amendment
ceremony with explicit justification.

Adding capabilities, states, order types, venues, or binaries is out of
scope.  The wave proves that what exists composes correctly.  It does not
build new things.
