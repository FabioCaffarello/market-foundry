# S415: Futures Venue Execution Proof Wave -- Charter Report

## Stage Identity

| Field | Value |
|---|---|
| Stage | S415 |
| Type | Charter and scope freeze |
| Wave | Futures Venue Execution Proof |
| Date | 2026-03-23 |
| Predecessor | S414 (Production Readiness Hardening Evidence Gate -- PASS, FULL DELIVERY) |

---

## 1. Executive Summary

S415 opens the Futures Venue Execution Proof Wave, formally chartering the
extension of real venue execution proof from Spot (completed S404--S414) to
the Binance Futures testnet on the unified runtime.

The wave is scoped as a mirror of the Spot proof wave: same architectural
pipeline, same lifecycle invariants, same persistence surfaces, but exercised
against Futures-specific API semantics (`/fapi/v1/order`, top-level `avgPrice`,
margin-based rejections, higher partial fill likelihood).

**Key outcomes:**
- Wave formally opened with frozen scope (5 execution stages + 1 gate).
- 10 capabilities defined (FV-C1 through FV-C10).
- 12 governing questions defined (FV-Q1 through FV-Q12).
- 40 non-goals frozen (NG-1 through NG-40).
- Stage order established: S416--S420.
- NG-36 (no Futures proof) from S410 lifted as this wave's primary goal.

---

## 2. Consolidated State Analysis

### 2.1 Wave History

Seven consecutive waves have passed since S370:

| # | Wave | Range | Verdict |
|---|---|---|---|
| 1 | Multi-binary orchestration | S370--S375 | PASS |
| 2 | Exchange listening + dry-run | S376--S381 | PASS |
| 3 | OMS Foundation | S382--S388 | PASS |
| 4 | Binance segmentation | S390--S395 | PASS |
| 5 | Unified segment runtime | S398--S403 | PASS -- FULL DELIVERY |
| 6 | Testnet venue execution (Spot-first) | S404--S409 | PASS -- SUBSTANTIAL |
| 7 | Production readiness hardening | S410--S414 | PASS -- FULL DELIVERY |

### 2.2 What Is Proven

| Layer | Evidence |
|---|---|
| **Lifecycle** | Seven-state machine with 49/49 transitions, 8 invariant categories |
| **Spot venue execution** | Real acceptance, fill, rejection on Binance Spot testnet |
| **Persistence** | KV, ClickHouse (fills + rejections), operational lifecycle list |
| **Endurance** | 2,000+ submission cycles, 5 symbols, 2 sources, 10 concurrent goroutines |
| **Unified runtime** | Single binary, multi-adapter, source-based routing, fail-closed dispatch |
| **Segment isolation** | Spot/Futures routing with leakage invariant tests |

### 2.3 What Remains Unproven

| Gap | Rationale for This Wave |
|---|---|
| Futures real venue execution | `BinanceFuturesTestnetAdapter` exists but never exercised against real testnet |
| Futures-specific response parsing | `avgPrice`/`cumQuote` response model untested against real data |
| Futures rejection semantics | Margin-based rejections (`-2019`, `-4003`) not observed live |
| Real partial fill | RG-2 carried forward; Futures more likely to produce real partial fills |
| Dual-segment read-path parity | Unified read surfaces never verified with both Spot and Futures data |

### 2.4 Residual Gaps Carried Forward

| Gap | Severity | Disposition |
|---|---|---|
| RG-2: Partial fill live observation | Low | ELEVATED: Futures partial fills more likely; S417 attempts |
| RG-3: Latest-only KV semantics | Low | DEFERRED: by design |
| RG-4: Segment-scoped list queries (partial) | Low | DEFERRED: operational listing sufficient |
| RG-6--RG-11 | Low | CARRIED: not addressed in this wave |

---

## 3. Wave Charter

### 3.1 Objective

Prove that the canonical OMS lifecycle behaves correctly against real Binance
Futures testnet responses on the unified runtime, covering acceptance, fill,
rejection, partial fill, persistence, read-path parity, and compose-level E2E.

### 3.2 Scope Freeze

**In scope:** 9 items (real Futures HTTP interactions, lifecycle transitions,
fill fidelity, rejection fidelity, partial fills, persistence consistency,
read-path segment parity, compose E2E, segment isolation).

**Out of scope:** 40 non-goals (NG-1 through NG-40) covering mainnet,
multi-exchange, advanced order types, full OMS, portfolio risk, leverage
management, position mode, liquidation, funding rates, and more.

Full scope definition in:
- [`docs/architecture/futures-venue-execution-proof-wave-charter-and-scope-freeze.md`](../architecture/futures-venue-execution-proof-wave-charter-and-scope-freeze.md)
- [`docs/architecture/futures-venue-execution-capabilities-questions-and-non-goals.md`](../architecture/futures-venue-execution-capabilities-questions-and-non-goals.md)

---

## 4. Governing Questions

| ID | Question | Target |
|---|---|---|
| FV-Q1 | Real Futures acceptance + fill lifecycle | S416 |
| FV-Q2 | Futures fill record fidelity (avgPrice, cumQuote, fees) | S416 |
| FV-Q3 | Real Futures rejection lifecycle | S417 |
| FV-Q4 | Futures rejection event fidelity (error codes, reasons) | S417 |
| FV-Q5 | Futures partial fill observation or structural proof | S417 |
| FV-Q6 | Quantity monotonicity under Futures partial fills | S417 |
| FV-Q7 | KV/HTTP/ClickHouse agreement after real Futures interactions | S418 |
| FV-Q8 | ClickHouse rejection writer for Futures events (no code changes) | S418 |
| FV-Q9 | Full compose pipeline with Futures `venue_live` | S419 |
| FV-Q10 | Sustained multi-cycle Futures operation | S418 |
| FV-Q11 | Correlation chain integrity through Futures venue | S416 |
| FV-Q12 | Post-200 reconciliation under Futures conditions | S416 |

---

## 5. Non-Goals (Summary)

40 frozen non-goals organized in 8 categories:

| Category | Count | Key Items |
|---|---|---|
| Venue/market scope | NG-1--NG-5 | No mainnet, no multi-exchange, no limit orders |
| OMS/lifecycle scope | NG-6--NG-10 | No full OMS, no cancel API, no machine extension |
| Risk/portfolio/strategy | NG-11--NG-13 | No portfolio risk, no P&L, no strategy optimization |
| Infrastructure/operations | NG-14--NG-18 | No dashboards, no CI/CD, no vault |
| Architecture | NG-19--NG-22 | No lifecycle redesign, no domain extension |
| Segmentation | NG-23--NG-27 | No re-open segmentation, no shared core extraction |
| Runtime | NG-28--NG-32 | No runtime redesign, no concurrent dual venue_live, no schema changes |
| **Futures-specific** | **NG-33--NG-40** | **No leverage config, no position mode, no margin type, no funding rates, no liquidation, no mark price, no multi-asset margin, no income API** |

---

## 6. Stage Order

| Stage | Block | Title | Depends On |
|---|---|---|---|
| **S415** | B0 | Charter and scope freeze (this stage) | S414 |
| **S416** | B1 | Futures real venue connectivity, acceptance, and fill proof | S415 |
| **S417** | B2 | Futures real rejection and partial-fill evidence | S416 |
| **S418** | B3 | Unified runtime read-path, auditability, and segment parity | S416 |
| **S419** | B4 | Unified compose E2E with Futures live execution path | S417, S418 |
| **S420** | B5 | Evidence gate: Futures Venue Execution Proof | S419 |

**Dependency notes:**
- S417 and S418 both depend on S416 (connectivity must be proven first).
- S417 and S418 are independent of each other and may execute in parallel.
- S419 depends on both S417 and S418 (compose E2E integrates all prior evidence).
- S420 evaluates all prior stages.

---

## 7. Preparation Recommended for S416

1. **Provision Futures testnet credentials**: `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`.
2. **Verify Futures testnet account state**: sufficient USDT balance, default leverage (20x), cross margin, one-way position mode.
3. **Verify `BinanceFuturesTestnetAdapter` compilation**: adapter exists at `internal/application/execution/binance_futures_testnet_adapter.go` with unit tests.
4. **Prepare unified config variant**: `execute-unified.jsonc` with `futures.enabled=true`, `futures.dry_run=false`, Spot disabled or in `dry_run`.
5. **Verify segment routing**: confirm `SegmentForSource("binancef")` returns `futures` segment.
6. **Select Futures test symbol**: recommend `BTCUSDT` on Futures testnet for broadest liquidity.
7. **Review Futures API differences**: confirm adapter handles `/fapi/v1/order` response format with top-level `avgPrice`/`cumQuote` (no `fills[]` array).

---

## 8. Deliverables

| Artifact | Path |
|---|---|
| Wave charter | `docs/architecture/futures-venue-execution-proof-wave-charter-and-scope-freeze.md` |
| Capabilities, questions, non-goals | `docs/architecture/futures-venue-execution-capabilities-questions-and-non-goals.md` |
| Stage report | `docs/stages/stage-s415-futures-venue-execution-proof-charter-report.md` |

---

## 9. Verdict

**CHARTER OPENED. SCOPE FROZEN.**

The Futures Venue Execution Proof Wave is formally open with:
- 10 capability targets (FV-C1 through FV-C10)
- 12 governing questions (FV-Q1 through FV-Q12)
- 40 non-goals (NG-1 through NG-40)
- 5 execution stages + 1 evidence gate (S416--S420)
- Zero scope inflation risk (scope amendment protocol enforced)

The wave builds on seven consecutive passing gates and reuses the entire
proven architecture: unified runtime, segment routing, persistence pipeline,
lifecycle state machine, and read-path surfaces. The incremental scope is
Futures adapter-level proof, not architectural redesign.

Next action: **S416 -- Futures real venue connectivity, acceptance, and fill proof.**
