# Stage S427 -- Production Hardening and Mainnet Readiness Audit Wave Charter Report

## Stage Identity

| Field | Value |
|---|---|
| Stage | S427 |
| Type | Charter and scope freeze |
| Wave | Production Hardening and Mainnet Readiness Audit |
| Predecessor | S426 (Futures Venue Execution Evidence Gate -- PASS, FULL DELIVERY) |
| Date | 2026-03-23 |

## Executive Summary

S427 opens the Production Hardening and Mainnet Readiness Audit Wave. This is a short, focused wave (S428--S431) that closes the highest-priority operational gaps remaining after eleven consecutive passing evidence gates and produces a formal mainnet readiness audit.

The wave does NOT enable mainnet execution. It produces the information, infrastructure, and decision artifacts needed for a future mainnet authorization ceremony.

### What Was Analyzed

The consolidated state across all waves from S370 to S426 was evaluated:

- **11 consecutive passing gates** with zero regressions.
- **16 residual gaps**: 1 medium-severity (RG-13: fee semantic divergence), 15 low-severity.
- **Zero high-severity gaps**.
- **84+ tests** across the Futures proof wave alone; **2,000+ endurance cycles** from S412.
- **50% entropy reduction** on config/compose surfaces from S416--S418.
- Complete Spot and Futures execution chains proven on Binance testnet with real venue responses.

### What This Wave Resolves

| Block | Stage | Target |
|---|---|---|
| Fee normalization | S428 | Close RG-13: canonical fee model across Spot and Futures |
| Health signals | S429 | Per-segment operational readiness signals |
| Mainnet audit | S430 | Formal readiness checklist evaluation + KV history decision (RG-3) |
| Evidence gate | S431 | Wave verdict and next-ceremony recommendation |

## Consolidated State Assessment

### Proven Capabilities

| Dimension | Status | Evidence |
|---|---|---|
| Spot venue execution (real testnet) | PROVEN | S404--S409, S410--S414 |
| Futures venue execution (real testnet) | PROVEN | S415--S420, S421--S426 |
| Seven-state lifecycle model | FROZEN | S383, invariant tests in S384 |
| Unified multi-segment runtime | PROVEN | S398--S403 |
| Segment isolation (fail-closed routing) | PROVEN | AllowedSources gate, S401 |
| Rejection persistence (KV + ClickHouse) | PROVEN | S411, S414 |
| Endurance (2,000+ cycles, zero races) | PROVEN | S412 |
| Operational lifecycle queries | PROVEN | S413 |
| Dry-run safety wrapper | PROVEN | S379, S380 |
| Kill switch and staleness guard | PROVEN | S378, S412 |
| Config/compose simplification (3+3) | PROVEN | S416--S418 |
| Multi-binary orchestration (8 binaries) | PROVEN | S370--S375 |

### Open Residual Gaps

#### Medium Severity

| ID | Gap | Wave Resolution |
|---|---|---|
| RG-13 | Fee semantic divergence (Spot commission vs Futures cumQuote) | S428: canonical fee model |

#### Low Severity (Carried, Non-Blocking)

| ID | Gap | Wave Resolution |
|---|---|---|
| RG-2 | Partial fill live observation (testnet limitation) | Carried; structural proof sufficient |
| RG-3 | Latest-only KV semantics | S430: formal decision |
| RG-4 | Segment-scoped list queries | Carried; operational listing sufficient |
| RG-6 | Rejection code in JSON, not column | Carried; queryable via JSONExtractString |
| RG-7 | No dedicated rejection endpoint | Carried; filtered general endpoint sufficient |
| RG-8 | Synthetic endurance (cycle-based) | Carried; compose smoke phases mitigate |
| RG-9 | No time-based drift detection | Carried; actor health tracker mitigates |
| RG-10 | No pagination on lifecycle list | Carried; bounded cardinality |
| RG-11 | Lifecycle list eventually consistent | Carried; <1s lag acceptable |
| RG-12 | cumQuote as Futures fee proxy | S428: normalized in canonical model |
| RG-14 | No parallel Spot+Futures live execution | Carried; NG-10 |
| RG-15 | Single symbol at compose level | Carried; multi-symbol structurally supported |
| RG-16 | 97 untracked docs | Carried; NG-7 |
| RG-17 | Smoke script naming inconsistency | Carried; cosmetic |
| RG-18 | Doc suitability not assessed | Carried; no runtime impact |

## Wave Design Rationale

### Why Fee Normalization First (S428)

RG-13 is the only medium-severity gap in the system. It directly affects data consumer correctness: any analytics or monitoring layer that reads fee data must currently branch on source to interpret the value. A canonical fee model removes this ambiguity and is a prerequisite for the mainnet readiness audit.

### Why Health Signals Second (S429)

Per-segment health monitoring is a missing operational capability. Without it, operators must inspect logs to determine whether a segment is healthy. This is acceptable for development but not for any production-adjacent deployment. Health signals are a prerequisite for the mainnet audit's monitoring/alerting evaluation.

### Why Mainnet Audit Third (S430)

The audit depends on both fee normalization (C-5 parity) and health signals (C-6 through C-8) being in place. It is a read-only assessment that evaluates the system against a production checklist and renders a readiness verdict. It does not implement production changes; it documents what is ready and what is not.

### Why Evidence Gate Last (S431)

Standard wave closure. Evaluates all blocks against the charter, scores capabilities, classifies any new residual gaps, and recommends next ceremony direction.

## Governing Questions

1. Can a single canonical fee field represent both Spot and Futures without information loss?
2. Does the normalized model require `/fapi/v1/userTrades`, or is cumQuote sufficient?
3. Can the ClickHouse schema accept the normalized field without breaking migration?
4. What is the minimal signal set for per-segment health assessment?
5. Should health be push-based or pull-based?
6. Does the actor health tracker support per-segment scoping?
7. What is the complete mainnet readiness checklist?
8. Is latest-only KV sufficient for production operations?
9. Does the adapter architecture allow mainnet credentials without code changes?
10. What capital controls (if any) are prerequisites for mainnet?

## Non-Goals (Frozen)

| ID | Non-Goal |
|---|---|
| NG-1 | Mainnet enablement (audit only) |
| NG-2 | Multi-exchange support |
| NG-3 | OMS expansion (limit orders, amendments, cancel) |
| NG-4 | Dashboard or UI development |
| NG-5 | Config/compose surface re-expansion |
| NG-6 | `/fapi/v1/userTrades` integration |
| NG-7 | Documentation governance ceremony |
| NG-8 | Large structural refactoring |
| NG-9 | Pagination or advanced query expansion |
| NG-10 | Parallel Spot+Futures live execution proof |

Full rationale for each non-goal is documented in:
[`production-hardening-capabilities-questions-and-non-goals.md`](../architecture/production-hardening-capabilities-questions-and-non-goals.md)

## Stage Order

| Stage | Block | Description |
|---|---|---|
| S427 | Charter | Production Hardening and Mainnet Readiness Audit Wave -- charter and scope freeze (this stage) |
| S428 | Block 1 | Fee normalization and cross-segment consistency |
| S429 | Block 2 | Per-segment health and operational readiness signals |
| S430 | Block 3 | Mainnet readiness audit and KV history strategy decision |
| S431 | Block 4 | Evidence gate: Production Hardening and Mainnet Readiness Audit |

## Preparation for S428

S428 should begin with:

1. **Read** `internal/domain/execution/events.go` to inventory current fee-related fields on `VenueOrderFilledEvent` and `FillRecord`.
2. **Read** `internal/adapters/clickhouse/writerpipeline/support.go` to understand how fee fields map to ClickHouse columns.
3. **Read** the Spot and Futures test files (`s405_*`, `s416_*`) to see how fee values are asserted today.
4. **Design** the canonical fee field: additive, no removal of existing fields, no breaking migration.
5. **Implement** the normalized field in the domain, wire through write-path, prove via read-path queries.
6. **Test** cross-segment fee query parity.

## Deliverables

| Artifact | Path | Status |
|---|---|---|
| Wave charter and scope freeze | [`production-hardening-and-mainnet-readiness-wave-charter-and-scope-freeze.md`](../architecture/production-hardening-and-mainnet-readiness-wave-charter-and-scope-freeze.md) | Delivered |
| Capabilities, questions, and non-goals | [`production-hardening-capabilities-questions-and-non-goals.md`](../architecture/production-hardening-capabilities-questions-and-non-goals.md) | Delivered |
| Stage report | This document | Delivered |

## Verdict

**S427: COMPLETE**. The Production Hardening and Mainnet Readiness Audit Wave is formally open with frozen scope. Four execution stages (S428--S431) are authorized. Non-goals are explicit. Governing questions are formulated. The wave is ready to proceed to S428.
