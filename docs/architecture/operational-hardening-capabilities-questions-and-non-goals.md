# Operational Hardening — Capabilities, Questions, and Non-Goals

**Stage**: S498
**Date**: 2026-03-28
**Wave**: Operational Hardening (S498–S502)

---

## 1. Capability Details

### C-OH1: Futures Fee Retrieval or Bounded Estimation (MUST — S499)

**Problem**: Binance Futures RESULT response type does not include commission data. The current adapter records `Fee="0"` and `FeeAsset=""` for all Futures fills (documented as RG-22 since S428). This means:
- Net P&L for Futures round-trips overstates return by the actual fee amount.
- `fee_gap` flag is always raised for Futures pairs.
- Cross-segment fee comparison is structurally unreliable.

**Target state**: Either:
1. Post-fill async call to `/fapi/v1/userTrades` to retrieve actual commission; or
2. Bounded estimation using Binance's published fee schedule (maker/taker rates by tier); or
3. Both — async retrieval when credentials allow, estimation as fallback.

The chosen approach must be documented with its accuracy bounds.

**Evidence required**: Test proving Futures fills carry non-zero fee data (or documented estimation) after S499.

### C-OH2: Historical Fee Field Normalization (SHOULD — S499)

**Problem**: Pre-S428 Futures fills stored `cumQuote` (notional) in the `Fee` field. Query-time comparison of fee data across historical periods produces incorrect results unless the caller understands the schema change boundary.

**Target state**: Either:
1. Query-time normalization function that detects pre-S428 records and adjusts; or
2. Documentation of the schema boundary with operator guidance.

Migration of historical ClickHouse data is explicitly NOT required (GR-2).

**Evidence required**: Test or documented procedure proving historical fee queries return correct results.

### C-OH3: Fee Reconciliation Rule Tightening (MUST — S499)

**Problem**: The reconciliation framework (S482, S496) raises `fee_gap` and `fee_asset_mismatch` flags but does not:
- Distinguish "Futures structural zero" from "genuine data loss";
- Propagate fee reliability assessment to effectiveness P&L calculations;
- Flag pre-S428 historical records with overloaded fee semantics.

**Target state**: Reconciliation rules that:
1. Classify fee gap cause: `futures_structural`, `data_missing`, `historical_overloaded`;
2. Mark `fee_reliable` as false for any round-trip with a classified gap;
3. Propagate fee reliability to effectiveness attribution.

**Evidence required**: Tests proving cause classification works for all three gap types.

### C-OH4: Explicit Session Close with Terminal State Enforcement (MUST — S500)

**Problem**: Session close currently relies on the supervisor's `Stopped` callback. There is no explicit close ceremony that:
- Confirms all in-flight orders reached terminal state;
- Records close timestamp with precision;
- Prevents new order submission after close signal.

**Target state**: Explicit `CloseSession()` flow that:
1. Stops accepting new execution intents;
2. Waits for in-flight orders (with bounded timeout);
3. Records terminal session state with close timestamp;
4. Logs any non-terminal orders at close as warnings.

**Evidence required**: Test proving session close produces terminal state with no orphan in-flight orders.

### C-OH5: Duplicate Leg Prevention at Session Boundaries (MUST — S500)

**Problem**: L-S496-3 documents that "duplicate legs from improper session closure" can occur. The current system relies on "operational discipline" — there is no programmatic guard.

**Target state**: Deduplication guard that:
1. Detects duplicate `correlation_id` within a session;
2. Prevents duplicate persistence of the same execution event;
3. Logs duplicate detection as a warning.

**Evidence required**: Test proving duplicate submission produces a single persisted record with warning.

### C-OH6: Session Boundary Timestamp Guard (SHOULD — S500)

**Problem**: L-S495-2 documents "session time overlap risk" where fill timestamps near session close boundary may be ambiguous. The current ±5 minute buffer (S485) is operational convention, not enforced.

**Target state**: Programmatic enforcement of the session boundary buffer:
1. Fills arriving after close signal within buffer are accepted and attributed;
2. Fills arriving after buffer are logged as boundary violations;
3. Buffer duration is configurable.

**Evidence required**: Test proving boundary buffer enforcement for fills arriving after close.

### C-OH7: Futures Segment Endurance Coverage (MUST — S501)

**Problem**: L-S412-6 documents "no Futures segment endurance" — the 200-cycle endurance soak (S412) was Spot-only. While the architecture is segment-agnostic (S398–S403), Futures fills have different field mappings (avgPrice, cumQuote, no commission) that have never been endurance-tested.

**Target state**: Endurance soak covering Futures segment with:
1. ≥100 cycles (matching Spot rigor);
2. Writer row mapping stability for Futures-specific fields;
3. Lifecycle state machine stability under Futures fill semantics.

**Evidence required**: Endurance test for Futures segment: all cycles pass, zero drift.

### C-OH8: Wall-Clock-Aware Stability Assertions (SHOULD — S501)

**Problem**: L-S412-1 and L-S412-2 document that "endurance window is synthetic" and there is "no time-based drift detection." All 200-cycle tests run in-process without real clock progression.

**Target state**: Stability assertions that:
1. Inject configurable time progression between cycles;
2. Detect state drift that accumulates over time (monotonic counters, timestamps);
3. Document any time-dependent behavior.

**Evidence required**: Test with simulated time progression proving no time-based drift.

### C-OH9: Batch Flush SLO Definition and Enforcement (SHOULD — S501)

**Problem**: L-S412-3 documents "ClickHouse batch flush lag" as expected behavior but no SLO bounds it. The current ~5s flush interval is configurable but there is no assertion that lag stays within acceptable bounds.

**Target state**:
1. Defined SLO: maximum acceptable lag between NATS event receipt and ClickHouse row visibility;
2. Test proving lag stays within SLO under sustained load;
3. Documented degradation behavior when SLO is exceeded.

**Evidence required**: Test measuring flush lag under load with SLO assertion.

---

## 2. Governing Questions — Detail

### Q-OH1: Is Futures fee data as accurate as the Binance API structurally allows?

**What "YES" means**: After S499, Futures fills carry actual commission data (from `/fapi/v1/userTrades`) or a documented bounded estimation. The `fee_gap` flag is no longer universally raised for Futures.

**What "NO" means**: Futures fills still carry `Fee="0"` after S499. The wave cannot close at FULL PASS.

### Q-OH2: Can an operator distinguish reliable from unreliable fee data across all segments and historical periods?

**What "YES" means**: The reconciliation framework classifies fee gaps by cause (`futures_structural`, `data_missing`, `historical_overloaded`). An operator querying historical data gets explicit guidance on which records have reliable fees.

**What "NO" means**: Fee reliability is still binary (`fee_gap` yes/no) without cause classification.

### Q-OH3: Does session close produce a deterministic terminal state with no orphan or duplicate legs?

**What "YES" means**: `CloseSession()` enforces terminal state. Duplicate leg submission is detected and prevented. No orphan in-flight orders survive close.

**What "NO" means**: Session close still relies on supervisor stop callback without explicit guarantees.

### Q-OH4: Is the writer pipeline proven stable across both Spot and Futures segments under sustained load?

**What "YES" means**: Endurance soak covers both Spot (existing) and Futures (new) segments. Writer row mapping, lifecycle state machine, and correlation chain are all proven for Futures.

**What "NO" means**: Futures endurance remains undocumented.

### Q-OH5: Are batch flush lag bounds defined and enforced?

**What "YES" means**: An SLO is defined (e.g., "99th percentile lag ≤ 10s under N events/second"). A test proves the SLO holds.

**What "NO" means**: Flush lag is acknowledged but unbounded.

---

## 3. Non-Goals — Detail

### NG-1: OMS Expansion

No new order types (limit, stop-loss), no position engine, no portfolio management. The current market-order scope is sufficient for operational hardening. OMS expansion is a separate macro-direction requiring its own risk assessment and charter.

### NG-2: Multi-Exchange

No adapters for exchanges beyond Binance (Spot and Futures). Multi-exchange requires adapter-layer expansion, venue model changes, and credential management redesign — none of which are operational hardening.

### NG-3: Broad Observability Platform

No distributed tracing integration, no centralized log aggregation, no Grafana/Prometheus dashboard suite. This wave hardens specific operational edges. Observability platform is a separate infrastructure concern.

### NG-4: New Strategy Families or Signal Types

No new strategy types, no new signal sources, no new decision evaluation logic. The derive layer is stable and not an operational concern.

### NG-5: Structural Redesign

No changes to binary topology (gateway/execute/writer remain), no NATS subject model changes, no ClickHouse schema redesign (only fee field normalization). The architecture is proven across 20+ waves and stable.

### NG-6: Portfolio/Risk Engine

No exposure tracking, no margin management, no portfolio-level risk assessment. This is a separate domain requiring its own domain model and evidence chain.

### NG-7: Dashboard or UI Surfaces

No operator-facing UI, no web dashboards, no visualization tools. All surfaces remain HTTP query endpoints.

### NG-8: Real-Time Streaming Analytics

No real-time event streaming to external consumers, no live alerting rules, no streaming aggregation. Operator surfaces are retrospective/query-based.

### NG-9: Cross-Binary Health Aggregation

Health signals remain constrained to the execute/writer pipeline. Ingest and store binary health signals are a separate concern for a future wave.

### NG-10: Write-Path Schema Changes Beyond Fee Fields

ClickHouse `executions` table schema is stable at 20 columns. Only fee-related fields may be modified in S499. No new columns, no table restructuring.

---

## 4. Success Metrics

| Metric | Target |
|--------|--------|
| MUST capabilities at FULL | 5/5 |
| SHOULD capabilities at SUBSTANTIAL+ | 4/4 |
| Governing questions YES | 5/5 |
| Regressions | ZERO |
| Critical/High gaps | NONE |
| Guard rails compliant | 8/8 |
| Wave span | ≤ 5 stages |

---

## References

- [Wave Charter and Scope Freeze](operational-hardening-wave-charter-and-scope-freeze.md)
- [Fee Semantics](fees-commission-assets-cross-segment-semantics-and-limitations.md)
- [Writer Stability](sustained-execution-state-consistency-writer-stability-and-limitations.md)
- [Fills and Fee Reconciliation](fills-fees-pairing-result-reconciliation-semantics-and-limitations.md)
