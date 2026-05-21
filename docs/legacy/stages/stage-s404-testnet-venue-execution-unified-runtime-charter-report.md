# S404: Testnet Venue Execution Proof Wave Charter on Unified Runtime (Spot-First)

**Stage**: S404
**Wave**: Phase 43 -- Testnet Venue Execution Proof on Unified Runtime (S404--S409)
**Type**: Charter and scope freeze
**Status**: Complete

## Objective

Open formally the Testnet Venue Execution Proof Wave on the unified segment
runtime, defining scope, stage order, governing questions, capability targets,
acceptance criteria, and non-goals with scope freeze.

This stage is NOT implementation. It is a charter ceremony that freezes the
wave's scope and unblocks execution stages S405--S409.

## What Changed

### New Files

| File | Purpose |
|---|---|
| `docs/architecture/testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md` | Wave charter: blocks, stages, questions, capabilities, risks, success criteria |
| `docs/architecture/testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md` | Companion: 12 governing questions, 10 capabilities, 35 non-goals, boundary conditions |

### No Code Changes

S404 is a pure charter ceremony. No production code, tests, configs, compose
files, or scripts were modified.

## Executive Summary

The Unified Segment Runtime Foundation Wave (S398--S403) closed with PASS --
FULL DELIVERY on 2026-03-22. All 10 capabilities at FULL, all 12 governing
questions at FULL, all 4 structural debts resolved, zero regressions. The
unified runtime is proven under dry-run and structural test conditions.

The next strategic step is proving that the unified runtime can execute real
orders against a real venue. This wave resumes the Testnet Venue Execution
Proof, which was originally chartered in S389, refreshed in S396, and is now
refreshed again in S404 to target the unified runtime architecture.

**Key decisions:**

1. **Spot-first preserved.** All 12 governing questions are answered against
   Binance Spot testnet. Futures proof remains deferred.
2. **Unified runtime consumed, not modified.** The single-binary, multi-adapter,
   unified-config architecture is treated as a stable foundation.
3. **35 non-goals frozen.** Seven new exclusions (NG-29 through NG-35) prevent
   unified runtime redesign, per-segment dry_run, concurrent live trading
   across segments, and config schema changes.
4. **Five execution stages.** S405 (acceptance/fill), S406 (rejection/partial),
   S407 (read-path/auditability), S408 (compose E2E), S409 (evidence gate).

## Wave Architecture

### Prior Charter Chain

```
S389 (original, Futures-only) -> S396 (segmented, Spot-first) -> S404 (unified runtime, Spot-first)
                                                                   ^-- ACTIVE
```

S389 and S396 are superseded. Only S404 governs execution.

### Governing Questions (12, unchanged since S389)

| ID | Question | Target |
|---|---|---|
| TV-Q1 | Real acceptance + fill lifecycle | S405 |
| TV-Q2 | Fill record fidelity (price, qty, fees) | S405 |
| TV-Q3 | Real rejection lifecycle | S406 |
| TV-Q4 | Rejection event fidelity (code, reason, HTTP status) | S406 |
| TV-Q5 | Partial fill observation or structural proof | S406 |
| TV-Q6 | Quantity monotonicity under partial fills | S406 |
| TV-Q7 | KV/HTTP/ClickHouse terminal state agreement | S407 |
| TV-Q8 | ClickHouse rejection writer wiring (RG-1) | S407 |
| TV-Q9 | Full compose pipeline in `venue_live` | S408 |
| TV-Q10 | Sustained multi-cycle correct behavior | S407 |
| TV-Q11 | Correlation chain integrity | S405 |
| TV-Q12 | Post-200 reconciliation under real conditions | S405 |

### Stage Order

| Stage | Block | Title |
|---|---|---|
| S404 | B0 | Charter and scope freeze (this stage) |
| S405 | B1 | Spot real venue connectivity, acceptance, and fill proof |
| S406 | B2 | Spot real rejection and partial-fill evidence |
| S407 | B3 | Unified runtime read-path and auditability under real responses |
| S408 | B4 | Unified compose E2E proof against real Spot testnet |
| S409 | B5 | Evidence gate: Testnet Venue Execution Proof (final) |

### Non-Goals Summary (35 frozen)

| Range | Category | Count |
|---|---|---|
| NG-1--NG-5 | Venue and market scope | 5 |
| NG-6--NG-10 | OMS and lifecycle | 5 |
| NG-11--NG-13 | Risk, portfolio, strategy | 3 |
| NG-14--NG-18 | Infrastructure and operations | 5 |
| NG-19--NG-22 | Architecture | 4 |
| NG-23--NG-28 | Segmentation | 6 |
| NG-29--NG-35 | Unified runtime | 7 |

### Key Non-Goals Highlighted

- **NG-23:** No parallel Futures testnet proof.
- **NG-29:** No unified runtime redesign.
- **NG-30:** No per-segment dry_run toggle.
- **NG-34:** No concurrent Spot + Futures `venue_live`.
- **NG-35:** No config schema changes to unified model.

## Entry Preconditions

All preconditions are met at charter time:

| Precondition | Status |
|---|---|
| OMS Foundation (S382--S388) | PASSED |
| Segmentation Foundation (S390--S395) | PASSED |
| Unified Runtime Foundation (S398--S403) | PASSED -- FULL DELIVERY |
| Spot ingest bindings (S397) | Complete |
| Spot testnet credentials | Required before S405 |

## Risk Highlights

| Risk | Mitigation |
|---|---|
| Spot testnet insufficient balance | Verify funding + document top-up before S405 |
| `dry_run=false` activates all enabled adapters | Config must enable Spot only |
| RG-1 (ClickHouse rejection writer) still open | S407 targets explicit closure |
| Partial fills hard to trigger on Spot | Structural proof acceptable |

## Preparation for S405

Before S405 can begin:

1. **Provision Spot testnet credentials** in the environment (env vars, not committed).
2. **Verify Spot testnet account balance** is sufficient for market order execution.
3. **Document the top-up procedure** for the Spot testnet account.
4. **Establish the `venue_live` config pattern** (Spot-only unified config with `dry_run=false`).
5. **Confirm `binances` ingest bindings are active** (S397 prerequisite).
6. **Run existing test suites** to confirm zero regressions at wave entry.

## Guard Rails Compliance

| Rule | Status |
|---|---|
| Do not open Futures proof in parallel | Respected (NG-23, NG-34) |
| Do not reopen segmentation wave | Respected (NG-24) |
| Do not open mainnet, multi-exchange, or full OMS | Respected (NG-1, NG-2, NG-6) |
| Do not redesign unified runtime | Respected (NG-29, NG-35) |
| Do not transform charter into implementation | Respected -- zero code changes |

## Promoted Documents

| Document | Location |
|---|---|
| Wave charter | [`docs/architecture/testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md`](../architecture/testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md) |
| Capabilities, questions, non-goals | [`docs/architecture/testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md`](../architecture/testnet-venue-execution-unified-runtime-capabilities-questions-and-non-goals.md) |
