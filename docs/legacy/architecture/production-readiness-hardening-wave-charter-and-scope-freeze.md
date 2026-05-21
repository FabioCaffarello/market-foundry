# Production Readiness Hardening Wave -- Charter and Scope Freeze

## Wave Identity

| Field | Value |
|---|---|
| Wave | Production Readiness Hardening |
| Charter Stage | S410 |
| Planned Stages | S411--S415 |
| Predecessor Wave | Testnet Venue Execution Proof (Unified Runtime, Spot-First), S404--S409 |
| Predecessor Verdict | PASS -- SUBSTANTIAL DELIVERY |
| Date Opened | 2026-03-23 |

## Strategic Context

The Foundry has accumulated six consecutive wave passes since S370:

1. Multi-binary orchestration (S370--S375): PASS
2. Exchange listening + dry-run (S376--S381): PASS
3. OMS Foundation (S382--S388): PASS
4. Binance segmentation (S390--S395): PASS
5. Unified segment runtime (S398--S403): PASS (FULL DELIVERY)
6. Testnet venue execution, Spot-first (S404--S408): PASS (SUBSTANTIAL DELIVERY)

The system has a complete, proven Spot execution chain: exchange ingress through lifecycle outcome to persistence and read-path on a unified multi-segment runtime with real venue connectivity.

The S409 evidence gate closed with SUBSTANTIAL DELIVERY due to one medium-severity gap (RG-1: ClickHouse rejection writer) and four low-severity residual gaps. The SUBSTANTIAL verdict means the proof is sound but operational hardening is required before the system can be considered production-grade for the Spot segment.

This wave exists to close those gaps with surgical precision, not to expand scope.

## Wave Objective

Close the prioritized operational gaps from the S409 evidence gate, add endurance confidence through soak testing, and consolidate lifecycle read-path into a production-grade posture for Spot on the unified runtime.

## Scope Freeze

### In Scope (Frozen)

The wave is organized into four execution blocks followed by a final evidence gate:

#### Block 1: Rejection Persistence and Read-Path Closure (S411)

Close RG-1 (the only medium-severity gap from S409).

- Wire the ClickHouse rejection writer consumer to the `EXECUTION_REJECTION_EVENTS` stream.
- Prove rejection events flow from venue adapter through NATS to ClickHouse persistence.
- Verify rejection queryability via the analytical path (ClickHouse) in addition to the operational path (KV).
- Prove fill and rejection events share consistent schema conventions in ClickHouse.

**Exit Criteria**: Rejection events persisted to ClickHouse with query proof. RG-1 closed.

#### Block 2: Endurance and Soak Hardening (S412)

Add temporal confidence that the proven pipeline sustains operation beyond single-cycle proof.

- Multi-symbol concurrent execution on Spot testnet (minimum 3 symbols).
- Sustained multi-cycle operation (minimum 50 order cycles or 30 minutes continuous, whichever is reached first).
- Memory and goroutine stability observation (no leaks over soak duration).
- Graceful shutdown and restart without state corruption.
- Error recovery: transient venue errors do not cascade into permanent state corruption.

**Exit Criteria**: Soak test passes with documented stability evidence.

#### Block 3: Operational Queryability and Lifecycle Read Consolidation (S413)

Close RG-5 and improve operational read-path coverage.

- Capture commission asset type from Binance Spot fill responses (RG-5 closure).
- Add segment-scoped list query capability for operational diagnostics (partial RG-4 closure).
- Consolidate fill and rejection read-path into a single operational query surface.
- Document operational query patterns and their expected outputs.

**Exit Criteria**: Commission asset captured. List query operational. Read surface consolidated.

#### Block 4: Evidence Gate (S414)

Formal evidence gate for the Production Readiness Hardening Wave.

- Audit all stage deliverables (S411--S413).
- Verify RG-1, RG-5 closure.
- Verify soak evidence.
- Classify residual gaps and recommend next ceremony.
- Produce evidence matrix with FULL/SUBSTANTIAL/PARTIAL/NONE classification.

**Exit Criteria**: Wave verdict rendered. Residual gaps classified.

### Deferred (Acknowledged, Not In Scope)

| ID | Item | Rationale |
|---|---|---|
| D-1 | RG-2: Live partial fill observation | Venue constraint (Spot market orders fill instantly). Structural proof sufficient. |
| D-2 | RG-3: KV history beyond latest | Requires JetStream stream history design. Analytical path (ClickHouse) covers historical queries. |
| D-3 | Full RG-4: Cross-intent list queries at scale | Partial closure in S413. Full analytical listing deferred to analytics wave. |

## Stage Execution Order

| Stage | Block | Title | Depends On |
|---|---|---|---|
| S410 | Charter | Production Readiness Hardening Wave Charter and Scope Freeze | S409 |
| S411 | Block 1 | Rejection Persistence and Read-Path Closure | S410 |
| S412 | Block 2 | Endurance and Soak Hardening | S411 |
| S413 | Block 3 | Operational Queryability and Lifecycle Read Consolidation | S411 |
| S414 | Block 4 | Production Readiness Hardening Evidence Gate | S412, S413 |

Note: S412 and S413 depend on S411 (rejection persistence must be wired before soak testing can verify it). S412 and S413 are independent of each other and may execute in parallel after S411.

## Risk Register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| R-1 | ClickHouse schema drift during rejection writer wiring | Medium | Use existing fill writer schema patterns. No schema redesign. |
| R-2 | Soak test flakiness from testnet instability | Medium | Define minimum acceptable uptime. Allow retry with documented interruptions. |
| R-3 | Scope creep into Futures or analytics | High | Non-goals frozen. No Futures execution in this wave. |
| R-4 | Commission asset type requires adapter changes | Low | Field already in Binance response. Extraction is additive. |

## Exit Criteria (Wave Level)

The wave passes if:

1. RG-1 (ClickHouse rejection writer) is closed with automated evidence.
2. RG-5 (commission asset type) is closed with automated evidence.
3. Soak test evidence demonstrates sustained multi-cycle, multi-symbol stability.
4. No regressions against prior wave capabilities (82+ tests from S404--S408, all prior waves).
5. Evidence gate produces formal verdict with FULL/SUBSTANTIAL/PARTIAL/NONE classification.
6. All non-goals remain respected.

## Scope Freeze Declaration

This scope is frozen as of 2026-03-23. No additional capabilities, blocks, or stages may be added without a formal scope amendment ceremony. The wave is intentionally small (3 execution stages + 1 gate) to prevent inflation.

## References

| Document | Path |
|---|---|
| S409 Evidence Gate | `docs/stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md` |
| S409 Evidence Matrix | `docs/architecture/testnet-venue-execution-unified-runtime-spot-first-evidence-matrix-residual-gaps-and-next-ceremony.md` |
| Capabilities and Non-Goals | `docs/architecture/production-readiness-hardening-capabilities-questions-and-non-goals.md` |
| OMS Foundation Evidence Gate | `docs/architecture/oms-foundation-evidence-gate.md` |
