# Stage S306 — Venue Readiness Charter and Scope Freeze Report

> Charter stage for the Venue Readiness Wave (Phase 30).
> Scope: formal wave opening, scope freeze, governing questions, stage ordering.
> Date: 2026-03-21

---

## 1. Executive Summary

Stage S306 opens the Venue Readiness Wave (Phase 30) following the successful closure of the Multi-Symbol Operational Scaling Wave (Phase 29, S300–S305, PASS). The wave objective is to replace paper execution with real exchange connectivity through the Binance Futures testnet, proving that the existing pipeline handles real venue order submission, fill reception, and lifecycle management.

The scope is frozen at 7 stages (S306–S312), 6 minimum capabilities, 7 governing questions, and 12 explicit non-goals. No concurrent wave is opened. Single-front discipline is maintained.

**Key decisions**:
- Venue readiness is defined as the minimum capability to submit real market orders and receive real fills — not a production trading system.
- Single exchange (Binance Futures testnet), market orders only, synchronous fills only.
- Existing pipeline (signal → decision → strategy → risk) remains unchanged.
- The `VenuePort` interface and `BinanceFuturesTestnetAdapter` already exist — the wave hardens and validates them, not builds from scratch.
- No OMS, no portfolio risk, no multi-venue, no dashboards, no schema changes.

---

## 2. Consolidated Post-S305 State

### 2.1 What Is Proven

| Capability | Evidence | Phase |
|-----------|----------|-------|
| Paper execution pipeline (intent → submit → fill → persist) | S264–S274 | Phase 14 |
| Multi-symbol isolation at all pipeline stages | S300–S304 | Phase 29 |
| Composite read model with symbol-scoped queries | S294–S299 | Phase 28 |
| VenuePort interface with adapter abstraction | S90–S93 | Phase 5 |
| Safety gate (kill switch + staleness guard) | S267–S268 | Phase 14 |
| 3 vertical slices (EMA, Trend, Squeeze) end-to-end | Multiple phases | Phase 10–27 |
| Credential loading and environment binding | S90 | Phase 5 |
| Execution domain model with lifecycle states and transitions | S264–S265 | Phase 14 |

### 2.2 What Is Not Proven

| Gap | Severity | Addressed By |
|-----|----------|--------------|
| Real venue order submission and fill reception | **Critical** — blocks production | S307–S309 |
| Fill model with real prices, quantities, fees | **Critical** — paper fills are zero-value | S308 |
| Venue failure classification and containment | **High** — paper mode has no failures | S310 |
| Safety gate enforcement under real venue | **High** — only tested with paper adapter | S309–S310 |
| Multi-symbol venue isolation | **Medium** — paper fills are instant/isolated by nature | S311 |
| Adapter error handling robustness | **Medium** — basic implementation exists, not hardened | S307 |

### 2.3 Existing Code Assets

| Asset | Path | Status |
|-------|------|--------|
| VenuePort interface | `internal/application/ports/venue.go` | Stable — no changes expected |
| PaperVenueAdapter | `internal/application/execution/paper_venue_adapter.go` | Stable — remains as fallback |
| BinanceFuturesTestnetAdapter | `internal/application/execution/binance_futures_testnet_adapter.go` | Exists — needs hardening |
| SafetyGate | `internal/application/execution/safety_gate.go` | Stable — no changes expected |
| StalenessGuard | `internal/application/execution/staleness_guard.go` | Stable — no changes expected |
| CredentialSet | `internal/application/execution/credentials.go` | Stable — may need minor extensions |
| ExecutionIntent domain | `internal/domain/execution/execution.go` | Stable — no changes expected |
| ControlGate domain | `internal/domain/execution/control.go` | Stable — no changes expected |
| Execution events | `internal/domain/execution/events.go` | Stable — no changes expected |

---

## 3. Charter Summary

### 3.1 Wave Objective

Prove that the market-foundry execution pipeline correctly handles real venue order submission, fill reception, and lifecycle management through Binance Futures testnet, with full observability, failure containment, and zero regression against the paper baseline.

### 3.2 Minimum Capabilities (6)

| ID | Capability | Target Stage |
|----|-----------|--------------|
| C1 | Venue adapter contract hardening | S307 |
| C2 | Fill model validation and lifecycle proof | S308 |
| C3 | End-to-end venue integration | S309 |
| C4 | Failure envelope and containment | S310 |
| C5 | Production guard rails under real venue | S309–S310 |
| C6 | Multi-symbol venue isolation | S311 |

### 3.3 Governing Questions (7)

| ID | Question | Target Stage |
|----|----------|--------------|
| VQ1 | Does the adapter submit orders and receive fills? | S307, S309 |
| VQ2 | Does ExecutionIntent lifecycle reflect venue states? | S308 |
| VQ3 | Do real fills persist without schema changes? | S309 |
| VQ4 | Does composite read model work with real data? | S309 |
| VQ5 | Are venue failures classified and contained? | S310 |
| VQ6 | Does the safety gate remain enforced? | S309, S310 |
| VQ7 | Does multi-symbol venue operation maintain isolation? | S311 |

### 3.4 Non-Goals (12)

NG-1: OMS | NG-2: Portfolio risk | NG-3: Multi-venue | NG-4: Advanced order types | NG-5: Async fills | NG-6: Dashboards | NG-7: New families | NG-8: Compliance | NG-9: Mainnet | NG-10: Schema changes | NG-11: Performance optimization | NG-12: Retry infrastructure

### 3.5 Stage Sequence (7 stages)

S306 (Charter) → S307 (Adapter hardening) → S308 (Fill model) → S309 (E2E integration) → S310 (Failure envelope) → S311 (Multi-symbol venue) → S312 (Gate)

---

## 4. Preparation for S307

S307 (Venue Adapter Contract Hardening) is the first implementation stage. The following preparation is recommended:

### 4.1 Prerequisites to Verify Before S307

1. **Binance Futures testnet credentials**: Ensure valid API key and secret are available via environment variables. Test connectivity with a simple authenticated request (e.g., account info endpoint).
2. **Testnet balance**: Ensure sufficient testnet balance for market order submissions across 3 symbols.
3. **Existing adapter test coverage**: Review `binance_futures_testnet_adapter_test.go` to understand current coverage and identify gaps.
4. **Rate limit documentation**: Review Binance Futures testnet rate limits to design test suites that respect them.

### 4.2 S307 Expected Deliverables

1. Hardened error classification for all Binance REST API error responses.
2. Timeout handling with configurable deadline.
3. Response validation (schema check before field extraction).
4. Credential safety audit (no leaks in errors, logs, or problem details).
5. Comprehensive unit test suite covering all failure modes.
6. Defined failure envelope document.

### 4.3 What S307 Should NOT Do

- Do not implement retry logic — mark retryable errors, but do not build retry infrastructure.
- Do not add WebSocket support — synchronous REST only.
- Do not submit real orders in unit tests — use httptest.Server for response simulation.
- Do not modify the `VenuePort` interface — the existing contract is sufficient.
- Do not touch the safety gate — it is validated in S309–S310.

---

## 5. Deliverables

| Deliverable | Path | Status |
|-------------|------|--------|
| Venue Readiness Charter and Scope Freeze | `docs/architecture/venue-readiness-charter-and-scope-freeze.md` | Delivered |
| Venue Readiness Capabilities, Questions, and Non-Goals | `docs/architecture/venue-readiness-capabilities-questions-and-non-goals.md` | Delivered |
| Stage report | `docs/stages/stage-s306-venue-readiness-charter-and-scope-freeze-report.md` | This document |

---

## 6. Acceptance Criteria Verification

| Criterion | Met? |
|-----------|------|
| Wave formally opened with scope frozen | Yes — charter document with 12 freeze rules |
| Venue readiness has clear, non-ambiguous definition | Yes — section 2 of charter: 6 specific capabilities |
| Non-goals are explicit | Yes — 12 non-goals with rationale for each |
| Next stages ordered with rigor | Yes — S307–S312 with dependencies and objectives |
| Governing questions defined | Yes — VQ1–VQ7 with evidence requirements |
| No venue implementation in this stage | Yes — charter and governance only; zero code changes |
| No scope inflation beyond venue readiness | Yes — single exchange, market orders, sync fills, no OMS |
| Preparation for S307 documented | Yes — section 4 with prerequisites and boundaries |
| Single-front discipline maintained | Yes — no secondary wave opened |
