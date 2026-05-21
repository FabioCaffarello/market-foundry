# Stage S388 — OMS Foundation Evidence Gate Report

| Field | Value |
|-------|-------|
| **Stage** | S388 |
| **Type** | Evidence Gate (Wave Closure) |
| **Wave** | OMS Foundation (Phase 40) |
| **Predecessor** | S387 — Lifecycle Persistence, Read-Path, and PriceSource Wiring |
| **Scope** | Formal evaluation of S382–S387 deliverables to determine wave closure |
| **Verdict** | **WAVE PASSED — CONDITIONAL** |

---

## Executive Summary

The OMS Foundation Wave (S382–S387) is **closed with conditional pass**. Six stages delivered 7 architecture documents, ~100 Go tests across 8 test files, 9 production code files, 1 new domain event (`VenueOrderRejectedEvent`), 1 new NATS stream, 1 new KV bucket, and 1 new interface (`PriceSource`) — all proving that the Foundry's order lifecycle foundation is canonical, exhaustively tested, and auditable across all execution modes. Invariant coverage moved from 16% to 100%. Zero regressions across ~50 packages.

The single condition is low-severity: ClickHouse rejection writer not wired (consumer spec and stream exist; JetStream 72h retention provides interim persistence). This does not block the next wave.

---

## 1. Wave Stages Reviewed

| Stage | Purpose | Verdict |
|-------|---------|---------|
| S382 | Charter and scope freeze | **COMPLETE** |
| S383 | Canonical order model and lifecycle state machine | **COMPLETE** |
| S384 | Lifecycle invariant coverage and price realism | **COMPLETE** |
| S385 | Write-path integration by execution mode | **COMPLETE** |
| S386 | Rejection event path and write-path observability | **COMPLETE** |
| S387 | Lifecycle persistence, read-path, and PriceSource wiring | **COMPLETE** |

All six stages delivered their chartered scope. No stage required rework or scope expansion.

## 2. Evidence Matrix Summary

### Governing Questions

| ID | Question | Verdict |
|----|----------|---------|
| OMS-Q1 | Lifecycle enforces all S309 invariants? | **ANSWERED** (S383+S384) |
| OMS-Q2 | Realistic prices without external API? | **ANSWERED** (S384+S387) |
| OMS-Q3 | Write-path correct per mode? | **ANSWERED** (S385) |
| OMS-Q4 | Safety gates block cross-mode? | **ANSWERED** (S385) |
| OMS-Q5 | Three surfaces agree on terminal state? | **SUBSTANTIAL** (KV+HTTP agree; CH rejection writer deferred) |
| OMS-Q6 | Fill model sufficient? | **ANSWERED** (S385) |
| OMS-Q7 | E2E OMS lifecycle with live data? | **SUBSTANTIAL** (composed proof, no dedicated smoke) |
| OMS-Q8 | Correlation chain intact? | **ANSWERED** (S385+S387) |
| OMS-Q9 | Sustained stability? | **SUBSTANTIAL** (inferred from prior wave + zero regressions) |

**6/9 ANSWERED, 3/9 SUBSTANTIAL.**

### Capability Classification

| ID | Capability | Classification |
|----|-----------|---------------|
| OMS-C1 | Lifecycle state machine | **FULL** |
| OMS-C2 | Terminal state finality | **FULL** |
| OMS-C3 | Fill-status consistency | **FULL** |
| OMS-C4 | Quantity monotonicity | **FULL** |
| OMS-C5 | Price realism in dry-run | **FULL** |
| OMS-C6 | Write-path dry_run | **FULL** |
| OMS-C7 | Write-path paper | **FULL** |
| OMS-C8 | Write-path venue_live | **FULL** |
| OMS-C9 | Safety gate enforcement | **FULL** |
| OMS-C10 | Correlation chain preservation | **FULL** |
| OMS-C11 | KV materialization | **FULL** |
| OMS-C12 | ClickHouse persistence | **SUBSTANTIAL** |
| OMS-C13 | HTTP query consistency | **FULL** |
| OMS-C14 | Fill model completeness | **FULL** |
| OMS-C15 | E2E OMS under live data | **SUBSTANTIAL** |
| OMS-C16 | Correlation traceable to query | **FULL** |
| OMS-C17 | Sustained stability | **SUBSTANTIAL** |

**13 FULL, 4 SUBSTANTIAL.**

### Evidence Layers

| Layer | What | Count |
|-------|------|-------|
| Domain tests (no infra) | State machine, invariants, rejection events | ~50 tests |
| Application tests (mock adapters) | Price realism, write-path per mode, lifecycle persistence | ~30 tests |
| Adapter tests (real NATS patterns) | Rejection publishing/consuming, price source | ~15 tests |
| Actor integration tests | Rejection event path through actor system | 5 tests |
| Architecture documents | Model, invariants, write-path, rejection, persistence | 7 documents |
| Stage reports | Delivery evidence trail | 7 reports (incl. this) |

## 3. Key Achievements

1. **Invariant coverage: 16% → 100%** — The wave's defining contribution. Every transition pair (49/49), terminal property, fill invariant, quantity bound, correlation chain, and validation rule now has explicit automated test evidence.

2. **Price realism closed (G1)** — `PriceSource` interface + `CandleKVPriceSource` implementation reads last observed market price from CANDLE_LATEST KV bucket. Wired in production with graceful fallback to "0".

3. **Rejection event observability** — `VenueOrderRejectedEvent` published to NATS on every submission failure; materialized to KV for queryability; included in composite status response.

4. **Write-path proven across all modes** — dry_run, paper, and venue_live paths validated with mode-specific assertions for Simulated flag, VenueOrderID prefix, state transitions, and correlation chain.

5. **Read-path completeness** — Single composite query returns Intent + Result + Rejection + Gate + Propagation. `DeriveEffectivePropagation()` resolves most-recent terminal state.

## 4. Gaps Closed by This Wave

| Gap | Origin | Closed by |
|-----|--------|-----------|
| G1: Price="0" in dry-run fills | S381 (prior wave) | S384+S387: PriceSource wired |
| G10: Single fill shape | S381 (prior wave) | S385: partial fills demonstrated |
| 41 invariant gaps (8/49 covered) | S383 (within wave) | S384: 100% coverage |
| Rejection observability | S385 (within wave) | S386: rejection events |
| Rejection persistence | S386 (within wave) | S387: KV projection + read-path |

## 5. Residual Gaps

| ID | Gap | Severity |
|----|-----|----------|
| RG-1 | ClickHouse rejection writer not wired | LOW |
| RG-2 | Domain-level quantity enforcement deferred | LOW |
| RG-3 | Fee realism in dry-run/paper (Fee="0") | LOW |
| RG-4 | Status `sent` never exercised E2E | LOW |
| RG-5 | Status `cancelled` via adapter not tested E2E | MEDIUM |
| RG-6 | No RejectionProjectionActor direct unit tests | LOW |
| RG-7 | No OMS-specific compose smoke script | LOW |
| RG-8 | Sustained stability not re-proven with OMS path | LOW |
| RG-9 | `knownExecutionFamilies` lacks rejection family | LOW |
| RG-10 | Rejection projection best-effort availability | LOW |

**1 MEDIUM, 9 LOW. None blocking.**

RG-5 (MEDIUM) is the only gap with non-trivial impact: `cancelled` status is in the state machine but has no E2E exercise. This is expected — order cancellation is a future capability (NG-6). The state machine correctly handles it; only the E2E path is untested.

## 6. Regression Verification

| Scope | Result |
|-------|--------|
| All Go workspace modules (~50 packages) | **ALL PASS** |
| New tests (~100 across 8 files) | **ALL PASS** |
| Pre-existing tests | **ZERO REGRESSIONS** |
| Binary compilation (5 binaries) | **ALL COMPILE** |

## 7. Formal Verdict

### **WAVE PASSED — CONDITIONAL**

The OMS Foundation Wave established a canonical, robust, and auditable order lifecycle foundation. The seven-state ExecutionIntent lifecycle is exhaustively tested (100% invariant coverage), the write-path is proven across all execution modes, rejections are fully observable, and the read-path provides complete lifecycle visibility.

**Condition:** ClickHouse rejection writer wiring (LOW severity, does not block next wave).

## 8. Next Direction

**Recommended:** Testnet Venue Execution Proof — exercise `venue_live` path with real testnet exchange responses, proving the OMS lifecycle end-to-end with actual venue fills and rejections.

**Alternative:** Operational Hardening — package OMS compose smoke, wire ClickHouse rejection writer, add observability dashboards.

The repository owner determines direction and timing. This gate only closes the current wave.

---

## Companion Documents

- [`docs/architecture/oms-foundation-evidence-gate.md`](../architecture/oms-foundation-evidence-gate.md) — Formal gate evaluation
- [`docs/architecture/oms-foundation-evidence-matrix-residual-gaps-and-next-ceremony.md`](../architecture/oms-foundation-evidence-matrix-residual-gaps-and-next-ceremony.md) — Detailed evidence matrix, gap analysis, and next ceremony recommendation

---

**Gate evaluated:** 2026-03-22
**Evaluator:** S388 evidence gate
**Wave status:** CLOSED — PASSED (conditional)
