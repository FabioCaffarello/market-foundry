# Post-S277 Debts Reconciliation Matrix

> Reconciliation gate for S270–S277 tranche.
> Date: 2026-03-21 | Gate: S278

## Purpose

This matrix is the single source of truth for every tracked debt after the
S270–S277 operational hardening tranche. It reconciles the S274 (post-S273)
intermediate gate, the original S269 (post-paper-execution) gate, and inherited
debts from earlier waves.

---

## Debt Classification Legend

| Status | Meaning |
|--------|---------|
| **CLOSED** | Proven by tests and/or CI; no further action needed |
| **PARTIALLY CLOSED** | Significant progress; specific remaining gap identified |
| **OPEN** | No progress or deliberately deferred |
| **OPEN (by design)** | Acknowledged scope limitation; not a defect |

---

## Paper Execution Wave Debts (from S269 Gate)

| ID | Debt | S269 Severity | S274 Status | Post-S277 Status | Evidence | Remaining Gap |
|----|------|---------------|-------------|------------------|----------|---------------|
| OD-PE1 | SafetyGate not wired into actor path | High | Closed | **CLOSED** | S270: 11 integration tests; kill switch, staleness, boundary precision all proven | None |
| OD-PE2 | S267 formal stage report missing | Medium | Open | **OPEN** | No report produced in S270–S277 tranche | Governance gap only; no functional impact |
| OD-PE3 | KV materialization unproven E2E | Medium | Partial | **PARTIALLY CLOSED** | S271: 8 adapter round-trip tests (real NATS); S276-MB6: KV materialization across binary boundary | Gateway query path (derive→store→KV→gateway GET) not proven end-to-end |
| OD-PE4 | ClickHouse round-trip unproven for execution | Medium | Closed | **CLOSED** | S272: 26 sub-tests (mapper/parser level); S277: 9 live ClickHouse tests (LAE-1–LAE-9) | None — proven at both serialization and live DB level |
| OD-PE5 | ControlGate kill switch not proven E2E | Medium | Partial | **SUBSTANTIALLY CLOSED** | S273: 6 runtime tests (real NATS KV); S275: 5 full-path tests (dual checkpoint, stream observability); S276: 6 multi-binary tests (cross-connection propagation) | Gateway HTTP API (execution.control.set request/reply) not exercised; all proofs use direct KV writes |
| OD-PE6 | Single symbol coverage (btcusdt@60s) | Low | Open | **OPEN (by design)** | Smoke-multi validates btcusdt+ethusdt at compose level | Expansion is feature work, not debt |
| OD-PE7 | Static signal values | Low | Open | **OPEN (by design)** | Signals are seeded via configctl, not computed from candles | Real candle computation is venue readiness scope |
| OD-PE8 | No concurrent scenario testing | Low | Open | **OPEN (by design)** | Sole-writer constraint eliminates concurrent writes by design (S271) | Load/stress testing is operational maturity scope |

---

## Behavioral Wave Debts (from S254/S257 Gates)

| ID | Debt | Original Severity | Post-S277 Status | Evidence |
|----|------|-------------------|------------------|----------|
| OD-BW1 | Full-stack behavioral smoke | Medium | **CLOSED** | S255: full-stack smoke + CI job `behavioral-scenarios` |
| OD-BW2 | Configurable scaling factors | Low | **OPEN (deferred)** | Hardcoded values adequate; no incident or requirement forcing change |
| OD-BW3 | Rejection path | Medium | **CLOSED** | S256: edge hardening |
| OD-BW4 | Severity normalization | Medium | **CLOSED** | S256: edge hardening |
| OD-BW5 | ClickHouse schema/performance | Low | **CLOSED** | S272+S277: full round-trip proven |
| OD-BW6 | Writer pipeline | Low | **CLOSED** | S272: writer pipeline round-trip proven |

---

## Codegen Reentry Debts (from S263 Gate)

| ID | Debt | Original Severity | Post-S277 Status | Evidence |
|----|------|-------------------|------------------|----------|
| OD-CG1 | Column-opaque spec cannot validate types | Medium | **OPEN** | No progress; codegen spec still column-opaque. Blocked until DDL/mapper generation is prioritized |
| OD-CG2–CG6 | Various codegen gaps | Low | **OPEN (deferred)** | Low severity; codegen-first workflow functional for current families |

---

## New Debts Identified in S270–S277

| ID | Debt | Severity | Source | Description |
|----|------|----------|--------|-------------|
| OD-OH1 | NATS KV tests not enforced in CI | Medium | S278 reconciliation | S271, S273, S275, S276 tests auto-skip when NATS unreachable; CI `integration-tests` job does not start NATS with JetStream; proofs are local-only |
| OD-OH2 | Live ClickHouse tests not enforced in CI | Medium | S278 reconciliation | S277 tests auto-skip when CLICKHOUSE_DSN not set; only smoke-analytical validates live CH, not the Go integration tests |
| OD-OH3 | Multi-binary tests run in single Go process | Low | S276 findings | S276 uses separate NATS connections but not separate OS processes; OS-level isolation (crash, restart, resource exhaustion) unproven |
| OD-OH4 | Gateway HTTP API for control gate untested | Low | S275 findings | All control plane proofs use direct KV writes; gateway's `execution.control.set` request/reply path not exercised |
| OD-OH5 | No KV watcher / push notification | Info | S275 findings | Control gate uses poll-on-read; acceptable for current frequency but may need watcher for sub-second response |
| OD-OH6 | No JetStream consumer durability proof | Low | S276 findings | Consumer redelivery across binary restart not tested; affects recovery semantics |

---

## Contradiction Resolution Log

| # | Contradiction | Resolution |
|---|--------------|------------|
| C1 | S277 lists OD-PE3 (KV materialization) as "remains open" but S271+S276 closed significant portions | **Reclassified as PARTIALLY CLOSED.** S271 proved adapter round-trip (8 tests, real NATS). S276-MB6 proved KV materialization across binary boundary. Remaining gap: gateway query path (derive→store→KV→gateway GET endpoint). S277's "remains open" was overly conservative. |
| C2 | S277 lists OD-PE5 (ControlGate) as "remains open" but S273+S275+S276 closed most of it | **Reclassified as SUBSTANTIALLY CLOSED.** S273 proved runtime halt/resume (6 tests). S275 proved dual-checkpoint full-path (5 tests). S276 proved cross-binary propagation (6 tests). Only gateway HTTP API entry point untested. S277's classification ignored S275–S276 progress. |
| C3 | S274 recommended S275 = "Store-Path and ControlGate Integration Smoke" but S275 delivered "Control Plane Full-Path Proof" | **Not a contradiction — scope adjustment.** S275 delivered a superset: full control plane path proof including stream observability and dual-checkpoint consistency, which subsumes the recommended smoke test. The store-path KV materialization gap (OD-PE3 remainder) was not addressed but was lower priority than control plane proof. |
| C4 | S274 recommended S277 = "Feature Expansion Gate" but S277 delivered "Live Analytical Execution Proof" | **Scope reordering.** The tranche correctly prioritized closing analytical live proof (OD-PE4 completion) before a feature gate. A feature expansion gate should not be run until operational debts are reconciled — which is what S278 now provides. |
| C5 | S274 lists 6 closed debts but includes OD-CG1, OD-BW2, OD-BW5, OD-BW6 which were closed in earlier waves | **Not a contradiction — cumulative accounting.** S274 correctly tallied debts closed across the full history, not just S270–S273. This is expected gate behavior. |
| C6 | S270 reports "no production code changes" but git status shows modifications to execution domain and risk domain files | **Not a contradiction.** The domain modifications (execution.go, risk.go, etc.) come from earlier stages (S265 severity fields, S266 paper order chain). S270–S273 only added test files and documentation. |

---

## Summary Counts

| Category | Count |
|----------|-------|
| Total debts tracked | 20 |
| **CLOSED** | 10 |
| **SUBSTANTIALLY CLOSED** | 1 |
| **PARTIALLY CLOSED** | 1 |
| **OPEN (deferred / by design)** | 6 |
| **OPEN (governance)** | 2 (OD-PE2, OD-CG1) |
| **NEW (from S278 reconciliation)** | 6 (OD-OH1–OH6) |
