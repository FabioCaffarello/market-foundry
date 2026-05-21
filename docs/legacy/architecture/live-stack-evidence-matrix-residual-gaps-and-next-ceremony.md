# Live Stack Evidence Matrix, Residual Gaps, and Next Ceremony

> **Stage:** S336 · **Wave:** Live Stack Integration (S332–S336)
> **Date:** 2025-03-21

---

## 1. Evidence Matrix

### 1.1 Capability Classification

| Block | Capability | Classification | Tests | Docs | Smoke |
|-------|-----------|---------------|-------|------|-------|
| LSI-1 | NATS Consumer → Actor Live Flow | **FULL** | LF-1, LF-2, LF-3, LF-4 | 2 arch docs | — |
| LSI-2 | Fill Round-Trip + Composite Visibility | **FULL** | BRT-18, BRT-19, CRI-7, CRI-8, CRI-9 | 2 arch docs | Phase 6–7 |
| LSI-3 | Kill-Switch Live + Canonical Smoke | **FULL** | CP-FP-1–5, CG-RT-1–6 | 2 arch docs | Phase 7 |

### 1.2 Governing Question Resolution

| GQ | Question | Block | Answered | Evidence Source |
|----|----------|-------|----------|----------------|
| GQ-1.1 | Consumer receives events? | LSI-1 | YES | LF-1 |
| GQ-1.2 | Actor executes onIntent()? | LSI-1 | YES | LF-1 |
| GQ-1.3 | Health tracker reflects delivery? | LSI-1 | YES | LF-1, LF-4 |
| GQ-1.4 | Durable consumer survives restart? | LSI-1 | YES | LF-2 |
| GQ-2.1 | Fill published to EXECUTION_FILL_EVENTS? | LSI-2 | YES | LF-1, BRT-18 |
| GQ-2.2 | Subject routing canonical? | LSI-2 | YES | Publisher code, BRT-18 |
| GQ-2.3 | Serialization integrity? | LSI-2 | YES | BRT-18, BRT-19 |
| GQ-2.4 | Composite visibility / read-after-write? | LSI-2 | YES | CRI-7, CRI-8, CRI-9 |
| GQ-3.1 | KV connection live? | LSI-3 | YES | Smoke Phase 7 |
| GQ-3.2 | Gate blocks execution? | LSI-3 | YES | LF-3, CP-FP-2 |
| GQ-3.3 | Halt checker works? | LSI-3 | YES | CP-FP-2, CP-FP-4 |
| GQ-3.4 | Recovery / resume? | LSI-3 | YES | Smoke Phase 7 |
| GQ-3.5 | Fail-open defaults? | LSI-3 | YES | CG-RT-1, safety_gate_test |
| GQ-4.1 | Evidence completeness? | LSI-4 | YES | This document |
| GQ-4.2 | Regression (202+ tests)? | LSI-4 | YES | All green, 9/9 invariants |
| GQ-4.3 | Smoke reproducibility? | LSI-4 | YES | smoke-live-stack 7 phases |
| GQ-4.4 | Risk documentation? | LSI-4 | YES | 6 arch docs + 4 stage reports |

**Result: 16/16 governing questions answered with evidence. Zero deferred.**

### 1.3 Test Coverage Matrix

| Test Suite | Stage | Build Tag | Infra Required | Tests | Status |
|-----------|-------|-----------|----------------|-------|--------|
| live_consumer_flow_test.go | S333 | integration | NATS | 4 | PASS |
| control_plane_full_path_test.go | S275 | integration | NATS | 5 | PASS |
| control_gate_runtime_test.go | S273 | integration | NATS | 6 | PASS |
| restart_recovery_test.go | — | integration | NATS | 5 | PASS |
| kv_store_roundtrip_test.go | — | integration | NATS | — | PASS |
| multi_binary_integration_test.go | — | integration | NATS | — | PASS |
| live_execution_analytical_test.go | S277 | requireclickhouse | ClickHouse | 9 | PASS |
| composite_reader_integration_test.go | S296 | requireclickhouse | ClickHouse | 6 | PASS |
| behavioral_roundtrip_test.go | S255/S334 | (unit) | None | 7+ | PASS |
| safety_gate_test.go | — | (unit) | None | — | PASS |
| safety_gate_integration_test.go | S270 | (unit) | None | — | PASS |
| control_test.go | — | (unit) | None | — | PASS |

**Total: 202+ prior tests green + 9+ new tests from wave = no regressions.**

### 1.4 Architecture Document Matrix

| Document | Stage | Covers |
|----------|-------|--------|
| live-stack-integration-wave-charter-and-scope-freeze.md | S332 | Wave charter, 4 blocks, non-goals, invariants |
| live-stack-capabilities-questions-and-non-goals.md | S332 | Governing questions GQ-1 through GQ-4, non-goals |
| nats-consumer-to-actor-live-flow.md | S333 | Consumer → actor canonical flow |
| live-consumer-flow-findings-bridges-and-limitations.md | S333 | Findings, paper bridge, lifecycle fix |
| fill-event-round-trip-and-composite-visibility.md | S334 | Fill path, composite reader, ordering |
| live-fill-round-trip-ordering-correlation-and-limitations.md | S334 | Correlation invariants, consistency model |
| kill-switch-live-and-canonical-smoke-live-stack.md | S335 | Kill-switch design, smoke phases |
| live-control-path-smoke-usage-and-operational-limitations.md | S335 | Operations guide, error remedies |

**Total: 8 architecture documents for the wave.**

### 1.5 Smoke Ceremony Matrix

| Phase | Description | Proves | Status |
|-------|-------------|--------|--------|
| 1 | Stack Readiness | ClickHouse, Writer, Gateway, NATS healthy | PASS |
| 2 | NATS Stream & Consumer Health | Streams exist, consumers durable | PASS |
| 3 | ClickHouse Analytical Data | 5 domain tables populated | PASS |
| 4 | Gateway Composite HTTP Surface | /chains, /funnel, /dispositions | PASS |
| 5 | Single-Family Endpoints | /history for 6 families | PASS |
| 6 | Structural Go Test Gate | S317 mapper, chain, dry run | PASS |
| 7 | Kill-Switch Control Path | halt→confirm→resume→confirm | PASS |

---

## 2. Residual Gaps

### 2.1 Gap Classification

| # | Gap | Severity | Category | Blocks Wave? |
|---|-----|----------|----------|-------------|
| G-1 | Extended 24h+ continuous observation | Medium | Operational | NO |
| G-2 | Partial fills with real venue data | Low | Domain | NO |
| G-3 | Commission uses cumQuote proxy | Low | Domain | NO |
| G-4 | Paper bridge subject mapping (transitional) | Low | Migration | NO |
| G-5 | Halt/resume under sustained production load | Medium | Operational | NO |
| G-6 | No per-type/per-symbol gate isolation | N/A | Design decision | NO |
| G-7 | No WebSocket/SSE async fills | N/A | Non-goal | NO |
| G-8 | Single venue only | N/A | Non-goal | NO |

### 2.2 Gap Analysis

**G-1: Extended observation (Medium)**
- What: No 24h+ continuous round-trip observation performed.
- Why not blocking: Wave is a verification wave, not a reliability wave. The pipeline is proven correct; endurance is a production readiness concern.
- Recommendation: Include in a future Production Readiness wave.

**G-2: Partial fills (Low)**
- What: Domain model supports partial fills but testnet only produces atomic fills.
- Why not blocking: Fill array and reconciliation logic handle partials. No testnet data to exercise.
- Recommendation: Test when real venue data available.

**G-3: Commission proxy (Low)**
- What: Commission uses `cumQuote` from fill response, not real commission endpoint.
- Why not blocking: Acceptable approximation for testnet. Real endpoint requires Binance account activation.
- Recommendation: Address during venue activation stage.

**G-4: Paper bridge (Low)**
- What: Execute consumer filters on `paper_order.submitted.>` instead of venue-specific intent subjects.
- Why not blocking: Transitional by design. Well-documented. Flow works correctly.
- Recommendation: Migrate when venue-specific intent subjects are introduced.

**G-5: Load testing (Medium)**
- What: Kill-switch not tested under sustained production load.
- Why not blocking: Control-plane proof is sufficient for testnet. Real-time blocking proven in integration tests (LF-3). Load testing is a production readiness concern.
- Recommendation: Include in Production Readiness wave.

**G-6–G-8: Design decisions / Non-goals**
- These are explicitly excluded from the wave charter and do not represent gaps.

### 2.3 Gap Verdict

**Zero gaps block wave closure.** Two medium-severity items (G-1, G-5) are recommended for a future Production Readiness wave. All others are low-severity or explicit non-goals.

---

## 3. Regression Verification

### 3.1 Invariant Status

| Invariant | Source | Verified By | Status |
|-----------|--------|-------------|--------|
| EC-1 | S327 | Deterministic client order ID in LF-1 | HELD |
| EC-3 | S327 | Correlation ID immutable in LF-1, CRI-7 | HELD |
| F-1 | S327 | Fill event contract in BRT-18 | HELD |
| F-4 | S327 | Column alignment in BRT-18, BRT-19 | HELD |
| RF-1 | S327 | Round-trip in CRI-7, CRI-8, CRI-9 | HELD |
| PGR-08 | S327 | Safety gate in LF-3, safety_gate_test | HELD |
| INV-REC-1 | S327 | Dedup in LF-4, durable consumer | HELD |
| INV-RC-1 | S327 | Deadline independence in venue_round_trip_test | HELD |
| INV-OBS-1 | S327 | Zero noise verified in health tracker assertions | HELD |

### 3.2 Code Changes During Wave

| Change | Stage | Risk | Regression? |
|--------|-------|------|-------------|
| Consumer ref + Close() in ExecuteSupervisor | S333 | Low — lifecycle fix | NO |
| Phase 7 added to smoke-live-stack.sh | S335 | None — additive | NO |
| Makefile target description update | S335 | None — cosmetic | NO |

**Result: Zero regressions. All changes additive or lifecycle fixes.**

---

## 4. Next Ceremony Recommendation

### 4.1 Strategic Assessment

The Live Stack Integration Wave proved the composed pipeline works. The Foundry now has:

- **Proven data path:** Signal → Decision → Strategy → Risk → Execute → Fill → Persist → Read
- **Proven control path:** Kill-switch halt/resume via HTTP → NATS KV
- **Proven durability:** Durable consumers, restart recovery, deduplication
- **Proven observability:** Health trackers, composite queries, analytical surface
- **Proven safety:** Dual-checkpoint gates, staleness guard, fail-open semantics

### 4.2 Candidate Next Waves (Fact-Based)

| Wave | Rationale | Priority |
|------|-----------|----------|
| **Venue Activation** | Real Binance testnet credentials, HTTP adapter activation, paper bridge migration | HIGH — next logical step |
| **Production Readiness** | 24h+ endurance, load testing, monitoring dashboards, alerting | MEDIUM — needed before mainnet |
| **Multi-Venue Expansion** | Second venue adapter, consumer multiplexing, venue abstraction | LOW — requires venue activation first |
| **OMS / Portfolio Risk** | Order management, position tracking, risk limits | LOW — separate domain |

### 4.3 Recommendation

**Next ceremony: Venue Activation Wave.**

Justification:
1. The pipeline is proven with simulated data. The next value increment is proving it with real venue data.
2. Venue activation resolves G-2 (partial fills), G-3 (commission), and G-4 (paper bridge) naturally.
3. It is the smallest step that delivers new evidence (real fills from Binance testnet).
4. Production Readiness can follow venue activation — endurance testing is more valuable against real venue data.

The Venue Activation Wave should be chartered as another **verification wave** with a frozen scope, following the same governance model that made S332–S336 successful.

### 4.4 What NOT To Do Next

- Do NOT open a broad "improvement" wave — keep scope frozen.
- Do NOT jump to multi-venue — prove single venue with real data first.
- Do NOT start mainnet activation — testnet verification must precede.
- Do NOT expand the kill-switch to per-type gates — global gate is sufficient until proven otherwise.
