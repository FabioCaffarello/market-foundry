# Stage S119 — Controlled Capability Definition Report

> Status: Complete | Date: 2025-03-19

## 1. Executive Summary

Stage S119 formally defines the first controlled capability delivery for market-foundry: **Multi-Symbol Live Monitoring (CC-01)**.

After 22 stages of architectural investment (S96–S118), the platform is operationally proven under single-symbol controlled conditions. S118 concluded that the next wave must deliver **capability, not architecture**. S119 selects and formalizes the smallest, lowest-risk capability that exercises the full proven mesh and reveals the next real friction points.

**Chosen capability:** Activate and sustain the complete pipeline chain for two symbols simultaneously (btcusdt + ethusdt), using zero new application code — config activation only.

**Why this one:** It is the only capability that stresses every runtime, every domain, and every architectural boundary simultaneously while requiring zero code changes. It converts the architecture into operational value.

## 2. Capability Selection

### 2.1 Chosen: Multi-Symbol Live Monitoring

| Dimension | Assessment |
|-----------|-----------|
| **Risk** | Low — already smoke-tested via `make smoke-multi` |
| **New code** | Zero application code. Config + operational tooling only. |
| **Architectural coverage** | 6/6 runtimes, 8/8 domains, 9/9 streams |
| **Operational value** | Immediate — monitoring multiple markets is fundamental |
| **Architectural signal** | High — proves horizontal scaling via config-driven activation |

### 2.2 Alternatives Evaluated

| Alternative | Risk | Verdict |
|-------------|------|---------|
| New signal family (MACD, Bollinger) | Medium | Premature. Prove horizontal scaling before vertical expansion. |
| Candle history enrichment | Low-Med | Exercises only store+gateway. Insufficient mesh pressure. |
| Strategy performance tracking | Medium | New domain concept. Better as CC-02 after CC-01 proves the base. |
| Live venue adapter (testnet) | High | External dependency, credentials, new safety gates. Explicitly gated. |
| MarketMonkey absorption | High | Do not import external code before proving Foundry delivers independently. |

### 2.3 Decision Rationale

The strategic directive is clear: the next evolution must come through **controlled capability, not horizontal refactoring**. Multi-symbol monitoring is the only candidate that:

1. Requires zero new code (pure config activation)
2. Exercises the complete event pipeline chain
3. Creates natural soak testing pressure without dedicated infrastructure
4. Proves the fundamental scaling mechanism (config-driven binding activation)
5. Has been partially validated (smoke-multi)

Any other capability either introduces new code risk or exercises only a subset of the mesh.

## 3. Scope Definition

### 3.1 What CC-01 Delivers

- Full pipeline operation for btcusdt + ethusdt concurrently
- Config activation proof with multiple simultaneous bindings
- Validation across all 12 gateway query endpoints (6 domains × 2 symbols)
- Diagnostic surface validation under doubled load
- 30+ minute sustained operation stability baseline
- Friction capture for S120 planning

### 3.2 What CC-01 Does Not Deliver

- No new application code
- No new domains, families, or actors
- No new NATS streams or consumers
- No new gateway endpoints
- No third symbol (2 is sufficient to prove the pattern)
- No per-symbol configuration (both symbols use identical family chains)
- No correlation ID injection (known friction, deferred to S120)
- No live venue activation (paper-only)

## 4. Deliverables Produced

| # | Document | Purpose |
|---|----------|---------|
| 1 | `docs/architecture/controlled-capability-01-definition.md` | Full capability definition: objective, rationale, flow, activation, impact |
| 2 | `docs/architecture/controlled-capability-01-success-criteria-and-out-of-scope.md` | Binary pass/fail criteria, explicit exclusions, friction capture protocol |
| 3 | `docs/architecture/controlled-capability-01-architectural-pressure-points.md` | Detailed pressure map: what is stressed, where to look, what might break |
| 4 | `docs/stages/stage-s119-controlled-capability-definition-report.md` | This report |

## 5. Success Criteria Summary

### 5.1 Mandatory (Must Pass)

- **Activation:** Config with 2 bindings activates. Both appear in active config. Ingest discovers both.
- **Pipeline flow:** All 7 pipeline stages (observation → execution) produce data for both symbols.
- **Diagnostics:** All health/readiness endpoints pass during multi-symbol operation.
- **Stability:** No crashes, no domain errors for 15+ minutes.
- **Tests:** `make smoke-multi` passes. `make test` passes.

### 5.2 Desired (Captured if Failed)

- Memory usage scales linearly (no unbounded growth at 30 minutes)
- Zero data loss in event chain (monotonically increasing tracker counts)
- raccoon-cli quality gate passes

## 6. Architectural Pressure Points

CC-01 applies pressure across 7 distinct areas:

| Area | Expected Outcome |
|------|-----------------|
| Config activation under concurrent state | Works as designed |
| Concurrent WebSocket management | Works as designed |
| Derive throughput doubling | Works, with known warm-up period (RSI needs 14 candles) |
| Store projection write amplification | Works as designed |
| Execute concurrent order processing | Works as designed |
| Gateway multi-symbol queries | Works, with known gaps (no symbol listing endpoint) |
| **Cross-runtime debugging** | **Will confirm correlation ID friction (F1 from S118)** |

The most likely new insight from CC-01 is confirmation that cross-runtime debugging without correlation IDs becomes painful at multi-symbol scale. This is expected and acceptable — it produces an evidence-based P1 item for S120.

## 7. Relationship to Prior Stages

| Stage | Relationship |
|-------|-------------|
| S113 | Execute safety model (SafetyGate) applies per-event for both symbols |
| S114 | Live activation procedure extended to multi-symbol |
| S115 | Operational validation pattern reused for CC-01 |
| S116 | Deferred items monitored for trigger conditions |
| S117 | Operational baseline extended to multi-symbol |
| S118 | Strategic recommendation fulfilled: deliver capability, not architecture |

## 8. Deferred Items Watch List

These S116 deferred items may trigger during CC-01 execution:

| Item | Trigger | Action if Triggered |
|------|---------|-------------------|
| Soak test infrastructure | Multi-symbol sustained operation | Record evidence, plan in S120 |
| Cross-runtime correlation tracing | Debugging multi-symbol flow | Record evidence, plan in S120 |
| Config parameterization | Need different families per symbol | Record evidence, defer |

## 9. Preparation for S120

S120 will be the **implementation and validation** stage for CC-01. The following preparation is recommended:

### 9.1 Pre-Implementation (Can Be Done in S120 Day 1)

1. **Update seed scripts** — ensure `seed-configctl.sh` and `seed-multi-configctl.sh` support incremental symbol addition
2. **Update activation script** — extend `live-pipeline-activate.sh` to validate both symbols
3. **Prepare monitoring checklist** — print the monitoring checklist from the pressure points document

### 9.2 During Implementation

1. **Follow activation flow** from CC-01 definition document
2. **Execute monitoring checklist** from pressure points document
3. **Record any friction** using the friction capture protocol from success criteria document
4. **Capture resource baseline** at 10-min and 30-min marks

### 9.3 Post-Implementation

1. **Evaluate success criteria** — all mandatory criteria must pass
2. **Compile friction records** — carry forward to S120 report
3. **Update deferred items** — mark any items whose triggers fired
4. **Recommend CC-02** — based on what CC-01 reveals

## 10. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No new horizontal refactoring wave opened | **Compliant** — CC-01 is capability delivery |
| Capability is not excessively large | **Compliant** — zero new code, config-only |
| No reopened architectural discussions | **Compliant** — all prior decisions preserved |
| No scope inflation with multiple capabilities | **Compliant** — single capability, clear boundaries |
| Out of scope is documented | **Compliant** — 13 items explicitly excluded |

## 11. Verdict

S119 is **complete**. The next controlled capability is formally defined with:
- Clear objective and rationale
- Objective, binary success criteria
- Explicit scope boundaries and exclusions
- Detailed pressure point map for informed monitoring
- Preparation steps for immediate S120 execution

The Foundry is ready to deliver its first capability.
