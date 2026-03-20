# Stage S191 — Family 06 Trigger Assessment and Candidate Selection Report

## Stage Identity

| Field | Value |
|-------|-------|
| Stage | S191 |
| Title | Family 06 Trigger Assessment and Candidate Selection |
| Objective | Evaluate candidates for Family 06 against S190 gate conditions; select or abort |
| Predecessor | S190 (Post-Family-05 / Pre-Family-06 Gate) |
| Status | **COMPLETE** |

## Executive Summary

Stage S191 evaluated all potential Family 06 candidates against the four S190 gate conditions. **No candidate satisfies condition C1 (no write-path changes).** The manual analytical expansion pattern is formally aborted at 6 families / 6 vertical layers.

The root cause is structural: the analytical read path is already generic (type-parameterized readers serve any event type within their layer), so within-layer expansion is a write-path problem, not a read-path problem. The gate condition correctly prevented an expansion that would have broken Wave B's zero-write-path-change guarantee.

The recommended next step is codegen tranche scoping (S192), following the S190 alternate path.

## Candidates Evaluated

| # | Candidate | Layer | Closest to Qualifying | Disqualification |
|---|-----------|-------|:---------------------:|------------------|
| A | EMA Crossover | L2 Signal | ✅ (only needs 1 writer pipeline entry) | C1: requires write-path change |
| B | Venue Market Order | L6 Execution | Moderate (needs mapper + pipeline) | C1: requires write-path changes |
| C | Trade Burst | L1 Evidence | Low (needs full 9-artifact expansion) | C1: requires write-path changes + new migration |
| D | Volume Metrics | L1 Evidence | Low (needs full 9-artifact expansion) | C1: requires write-path changes + new migration |
| E | Observation Trades | L0 | None (not an analytical family) | C1: not in analytical pipeline scope |

### Why EMA Crossover Was the Closest

EMA Crossover is the only candidate where:
- The ClickHouse table already exists (`signals`, migration 002).
- The mapper already works (`mapSignalRow()` is generic for all `SignalGeneratedEvent` types).
- The reader already supports it (`QuerySignalHistory()` accepts any `signalType`).
- The handler already exposes it (`GET /analytical/signal/history?type=ema_crossover` returns 200 today).
- The NATS registry already defines it (`EMACrossoverGenerated` in `SignalRegistry`).

The **only** missing artifact is a writer pipeline entry: `WriterEMACrossoverSignalConsumer()` + a `writerPipeline{}` entry in `pipeline.go`. This is a write-path change that violates C1.

## Formal Decision

| Question | Answer |
|----------|--------|
| Is there a Family 06 candidate that satisfies all S190 conditions? | **No** |
| Is the gate condition C1 unreasonable? | **No** — it correctly identifies the structural boundary between analytical read-path expansion and write-path extension |
| Should C1 be relaxed to accommodate EMA Crossover? | **No** — relaxing gate conditions undermines the discipline that kept Wave B healthy |
| What is the correct next step? | **Codegen tranche scoping (S192)** — following the S190 alternate path |

## Key Finding: Generic Read Path

The most significant architectural insight from this assessment:

> **The analytical read path is already more generic than the write path.**

All readers (L2–L6) accept a `type` parameter. The ClickHouse tables use `LowCardinality(String)` for type columns. The handlers pass through any type value. This means:
- Within-layer analytical expansion requires zero read-path changes.
- The only bottleneck for new event types is write-path enablement.
- Future family expansion (post-codegen) can focus on generating write-path + test artifacts; the read path will "just work."

This insight directly informs codegen design: codegen should primarily target write-path artifacts (consumer specs, pipeline entries) and test generation, since the read path is already generic.

## Wave B Final Scorecard

| Metric | Value |
|--------|-------|
| Families delivered | 6 (baseline + 5 expansions) |
| Vertical layers covered | 6/6 (L1–L6) — complete |
| Creative decisions | 0 across 5 expansions |
| Write-path modifications | 0 across 6 expansions |
| Total analytical LOC | ~3,950 |
| Total unit tests | 289 |
| Handler file (post-H-5) | 501 lines |
| Hardenings delivered | 2 (H-3 tranche, H-5 tranche) |
| Gate reviews passed | 6 (one per family + post-family gates) |
| Manual expansion pattern | **Retired** — complete specification for codegen |

## Deliverables Produced

| # | Document | Path |
|---|----------|------|
| 1 | Trigger assessment (principal) | `docs/architecture/family-06-trigger-assessment-and-candidate-selection.md` |
| 2 | Candidate comparison matrix | `docs/architecture/family-06-candidate-comparison-matrix.md` |
| 3 | Selection/abort rationale | `docs/architecture/family-06-selection-rationale-or-abort-rationale.md` |
| 4 | Stage report | `docs/stages/stage-s191-family-06-trigger-assessment-and-candidate-selection-report.md` |

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Family 06 is formally selected or expansion is formally aborted | ✅ Aborted with explicit rationale |
| Decision is based on explicit S190 criteria | ✅ Each candidate tested against all 4 conditions |
| Deferred candidates are well-justified | ✅ Per-candidate disqualification documented with specific missing artifacts |
| Risks and ceilings are clear | ✅ Risk matrix in abort rationale; ceiling evidence carried from S188 |
| Base is ready for next step (codegen scoping) | ✅ 6 families provide complete specification for codegen templates |

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| Family 06 not implemented | ✅ No code changes, no new artifacts |
| Multiple families not opened | ✅ Single assessment, single decision |
| S190 conditions not relaxed | ✅ C1 applied strictly; no "close enough" exceptions |
| Decision not driven by product interest | ✅ Decision is structural: no read-path work exists for any candidate |
| Out-of-scope items documented | ✅ Write-path enablement, codegen, and future candidates all scoped |

## Recommended Sequence After S191

```
S191: Family 06 Trigger Assessment ← THIS STAGE (COMPLETE)
  │
  └── Alternate path (no viable candidate):
      ├── S192: Codegen Tranche Scoping
      │   └─ Define templates, generation approach, artifact coverage
      │   └─ Use 6 existing families as specification evidence
      │   └─ Include write-path generation (consumer specs, pipeline entries)
      │
      ├── S193: Codegen Implementation
      │   └─ Build and validate templates against existing families
      │   └─ Prove generated output matches hand-crafted code
      │
      └── S194: First Generated Family (Family 07)
          └─ EMA Crossover via codegen (write + read artifacts generated)
          └─ Validates codegen end-to-end
          └─ Gate review with ceiling metrics
```

## Conclusion

Stage S191 fulfills its mission: a rigorous, evidence-based evaluation that led to a clear decision. The manual analytical expansion pattern has reached its natural completion — not because it failed, but because it succeeded so thoroughly that the read path became generic. The next investment is codegen, which will make family expansion trivially cheap and will unify write-path and read-path generation under a single automated process.

Wave B's legacy: 6 families, zero creative decisions, zero write-path modifications, and a complete pattern specification ready for automation.
