# S410: Production Readiness Hardening Wave Charter Report

## Stage Identity

| Field | Value |
|---|---|
| Stage | S410 |
| Type | Charter and scope freeze |
| Wave | Production Readiness Hardening |
| Scope | S411--S414 |
| Date | 2026-03-23 |
| Predecessor | S409 (Testnet Venue Execution Evidence Gate, Unified Runtime, Spot-First) |

## Executive Summary

S410 opens the Production Readiness Hardening Wave following the S409 evidence gate that closed the Testnet Venue Execution Proof Wave with PASS -- SUBSTANTIAL DELIVERY (9/10 capabilities FULL, 1/10 SUBSTANTIAL).

The wave targets surgical closure of the prioritized operational gaps from S409, endurance validation through soak testing, and read-path consolidation. It is intentionally small (3 execution stages + 1 evidence gate) and scope-frozen to prevent inflation.

The strategic decision is clear: the system has proven real Spot venue execution on the unified runtime. The next step is not breadth expansion (Futures, multi-exchange, analytics) but depth hardening of what is already proven.

## Motivation

### Why This Wave, Why Now

1. **RG-1 is the only medium-severity gap** from the entire S404--S409 wave. Closing it completes the analytical persistence path for rejection events.
2. **Single-cycle proof is necessary but not sufficient** for production confidence. Soak testing adds temporal evidence.
3. **Operational read-path has small gaps** (RG-5, partial RG-4) that are cheap to close and improve operator experience.
4. **The alternative directions** (Futures proof, analytics platform, multi-exchange) all benefit from having a fully hardened Spot foundation first.

### Why Not Other Directions

| Direction | Why Deferred |
|---|---|
| Futures Testnet Venue Execution | The Spot foundation must be production-grade before extending to a new segment with different API semantics. |
| Analytics / Observability Platform | Only the ClickHouse rejection writer (RG-1) is operationally blocking. A broad analytics wave is premature. |
| Mainnet | Requires risk ceremony, credential hardening, and operational runbooks beyond current scope. |
| OMS Expansion | The OMS Foundation model is sufficient for the current execution modes. |

## Deliverables

| Deliverable | Path |
|---|---|
| Wave Charter and Scope Freeze | `docs/architecture/production-readiness-hardening-wave-charter-and-scope-freeze.md` |
| Capabilities, Questions, and Non-Goals | `docs/architecture/production-readiness-hardening-capabilities-questions-and-non-goals.md` |
| Stage Report | `docs/stages/stage-s410-production-readiness-hardening-charter-report.md` |

## Wave Structure

### Blocks and Stages

| Stage | Block | Title | Key Deliverable |
|---|---|---|---|
| S410 | Charter | Charter and Scope Freeze | This document |
| S411 | Block 1 | Rejection Persistence and Read-Path Closure | ClickHouse rejection writer, RG-1 closure |
| S412 | Block 2 | Endurance and Soak Hardening | Multi-symbol, multi-cycle soak evidence |
| S413 | Block 3 | Operational Queryability and Lifecycle Read Consolidation | Commission asset, list queries, read surface |
| S414 | Block 4 | Production Readiness Hardening Evidence Gate | Wave verdict |

### Dependency Graph

```
S410 (charter)
  |
  v
S411 (rejection persistence)
  |
  +-------+-------+
  |               |
  v               v
S412 (soak)    S413 (read consolidation)
  |               |
  +-------+-------+
          |
          v
       S414 (evidence gate)
```

S412 and S413 are independent of each other but both depend on S411 (rejection persistence must be wired before soak can verify it, and read consolidation builds on the same persistence path).

## Governing Questions

| ID | Question | Target |
|---|---|---|
| PRH-Q1 | Do rejection events reach ClickHouse with correct schema? | S411 |
| PRH-Q2 | Are fill and rejection records structurally consistent in ClickHouse? | S411 |
| PRH-Q3 | Can the pipeline sustain 50+ cycles across 3+ symbols without leaks? | S412 |
| PRH-Q4 | Does the system recover from transient venue errors? | S412 |
| PRH-Q5 | Does graceful shutdown/restart preserve state? | S412 |
| PRH-Q6 | Is commission asset type captured and queryable? | S413 |
| PRH-Q7 | Can an operator list intents/rejections by symbol without partition keys? | S413 |
| PRH-Q8 | Is there a unified operational read surface for fills and rejections? | S413 |

## Non-Goals (Summary)

15 new non-goals frozen (NG-36 through NG-50), covering:

- No Futures proof, no mainnet, no multi-exchange
- No OMS expansion, no portfolio risk
- No broad analytics/observability platform
- No runtime redesign, no config schema changes
- No limit orders, no per-segment dry_run
- No KV history redesign, no CI/CD changes

All 35 prior non-goals (NG-1 through NG-35) remain in force.

Full list: [`production-readiness-hardening-capabilities-questions-and-non-goals.md`](../architecture/production-readiness-hardening-capabilities-questions-and-non-goals.md).

## Gap Closure Plan

| Gap | Severity | Disposition | Target Stage |
|---|---|---|---|
| RG-1: ClickHouse rejection writer | Medium | CLOSE | S411 |
| RG-2: Partial fill live observation | Low | DEFER (venue constraint) | -- |
| RG-3: Latest-only KV semantics | Low | DEFER (NG-47) | -- |
| RG-4: Segment-scoped list queries | Low | PARTIAL CLOSE | S413 |
| RG-5: Commission asset type | Low | CLOSE | S413 |

## Risk Register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| R-1 | ClickHouse schema drift | Medium | Reuse fill writer patterns |
| R-2 | Soak test flakiness from testnet | Medium | Define minimum uptime, allow documented retries |
| R-3 | Scope creep into Futures/analytics | High | Non-goals frozen, no amendments without ceremony |
| R-4 | Commission asset extraction | Low | Field already in Binance response |

## Verdict

**CHARTER COMPLETE -- SCOPE FROZEN**

The Production Readiness Hardening Wave is formally open with:

- 11 capabilities defined and assigned to stages
- 8 governing questions with clear target stages
- 15 new non-goals frozen (50 total with inherited)
- 3 execution stages + 1 evidence gate planned
- 2 gaps targeted for full closure (RG-1, RG-5), 1 for partial closure (RG-4)
- 2 gaps explicitly deferred with rationale (RG-2, RG-3)

## Preparation for S411

S411 (Rejection Persistence and Read-Path Closure) should:

1. Audit the existing `WriterVenueMarketOrderRejectionConsumer` spec in `natsexecution/registry.go`.
2. Audit the existing fill writer path to understand the ClickHouse persistence pattern.
3. Wire the rejection consumer to a ClickHouse writer that follows the same pattern.
4. Add automated tests proving rejection event persistence and queryability.
5. Produce architecture document and stage report.

The consumer spec already exists in the registry (created in S386). The wiring is the missing piece.

## References

| Document | Path |
|---|---|
| Wave Charter | `docs/architecture/production-readiness-hardening-wave-charter-and-scope-freeze.md` |
| Capabilities and Non-Goals | `docs/architecture/production-readiness-hardening-capabilities-questions-and-non-goals.md` |
| S409 Evidence Gate | `docs/stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md` |
| S409 Evidence Matrix | `docs/architecture/testnet-venue-execution-unified-runtime-spot-first-evidence-matrix-residual-gaps-and-next-ceremony.md` |
| S388 OMS Evidence Gate | `docs/architecture/oms-foundation-evidence-gate.md` |
| S403 Unified Runtime Gate | `docs/architecture/unified-segment-runtime-evidence-gate.md` |
