# Stage S159 — End-to-End Analytical Integration Proof Report

## Objective

Prove the complete analytical data path (NATS → writer → ClickHouse → reader → HTTP) in the minimum disciplined scope necessary to close the last S156 blocker.

## Executive Summary

S159 delivers the integration proof that was the single remaining blocker from the S156 Wave A readiness review. The proof is automated, repeatable, and validates all segments of the analytical data path through a 7-phase script. Infrastructure gaps found during implementation (writer missing from build targets, lib.sh service registry, and absence of an analytical smoke target) were fixed as part of the work.

The analytical layer is now proven end-to-end for the candle family. Boundaries are coherent. The foundation is ready for read-path hardening and operability improvements.

## What was done

### Proof script: `scripts/smoke-analytical-e2e.sh`

Automated 7-phase integration proof:

| Phase | Validates | Method |
|-------|-----------|--------|
| 1. Infrastructure readiness | ClickHouse + writer + gateway healthy | Health/readiness probes |
| 2. Migration status | 7 tables exist, 6 migrations applied | Direct ClickHouse queries |
| 3. Writer pipeline health | NATS → writer event consumption | Writer /statusz inspection |
| 4. ClickHouse data | Rows persisted in evidence_candles | Row count + sample query |
| 5. Reader → HTTP | Analytical endpoint returns candles | curl + structure validation |
| 6. Error handling | Invalid params rejected | Three negative cases (400) |
| 7. Writer observability | No degraded pipelines | Writer /diagz inspection |

### Infrastructure fixes

| Fix | File | Change |
|-----|------|--------|
| Writer in build targets | `Makefile` | Added `writer` to `BUILDABLE_SERVICES` |
| Analytical smoke target | `Makefile` | Added `make smoke-analytical` |
| Writer in lib.sh | `scripts/utils/lib.sh` | Added to `ALL_SERVICES` and `SVC_PORTS` |

## Files changed

| File | Action | Purpose |
|------|--------|---------|
| `scripts/smoke-analytical-e2e.sh` | Created | E2E integration proof script |
| `Makefile` | Modified | Writer in BUILDABLE_SERVICES, smoke-analytical target |
| `scripts/utils/lib.sh` | Modified | Writer in ALL_SERVICES and SVC_PORTS |
| `docs/architecture/analytical-end-to-end-integration-proof.md` | Created | Proof definition and method |
| `docs/architecture/analytical-end-to-end-validation-findings.md` | Created | Findings, gaps, and observations |
| `docs/stages/stage-s159-end-to-end-analytical-integration-proof-report.md` | Created | This report |

## Proof points

The complete data path is proven:

```
NATS JetStream (EVIDENCE_EVENTS)
  → writer consumer (durable: writer-candle)
  → writer inserter (batch + flush_interval)
  → ClickHouse (evidence_candles table)
  → CandleReader (parameterized SELECT)
  → GetCandleHistoryUseCase (validation + limit bounds)
  → GET /analytical/evidence/candles (HTTP 200 + JSON structure)
```

Each segment is independently verified:
- **NATS → writer**: Writer /statusz shows events_received > 0
- **Writer → ClickHouse**: evidence_candles row count > 0
- **ClickHouse → reader → HTTP**: Response contains candles with all 12 fields and source="clickhouse"
- **Error handling**: 400 for missing timeframe, invalid limit, since > until

## Boundaries confirmed coherent

| Boundary | Status | Evidence |
|----------|--------|----------|
| Writer ↔ NATS | Sound | Dedicated durable consumers (writer-* prefix), independent of store consumers |
| Writer ↔ ClickHouse | Sound | Batch insert with retry + backoff, FIFO eviction on overflow |
| Reader ↔ ClickHouse | Sound | Adapter-layer implementation, compile-time interface assertion |
| Gateway ↔ Reader | Sound | Thin bridge delegation, ClickHouse not in readiness check |
| Analytical ↔ Operational | Sound | Independent paths, independent health, independent failure modes |

## Limits and remaining gaps

| Gap | Severity | Source | Recommendation |
|-----|----------|--------|----------------|
| Reader has zero observability (no counters, no logging) | Medium | S156-precondition | Add structured logging + query timing in S160 |
| Non-candle read path not implemented | Low | By design | Expand only when needed |
| Writer degradation invisible to gateway | Medium | Architecture | Acceptable — lateral projection by design |
| Batch flush non-determinism in tests | Low | Inherent | Polling with configurable timeout handles this |
| No CI integration yet | Medium | Infrastructure | Wire smoke-analytical into CI pipeline |

## What was NOT done (guard rails respected)

- No expansion of the analytical endpoint surface
- No new read paths for non-candle families
- No changes to writer, reader, or gateway application code
- No masking of failures — script reports all phases independently
- No auto-migration — manual `make migrate-up` remains the deliberate path

## Preparation for S160

The analytical layer now has:
- Proven end-to-end data path (S156 blocker closed)
- Hardened boundaries (S158)
- Reviewed responsibilities (S157)
- Automated repeatable proof (`make smoke-analytical`)

Recommended focus for S160:

1. **Reader observability**: Add structured logging and query timing to the read path — the single most impactful open debt.
2. **CI integration**: Wire `make smoke-analytical` into the CI pipeline so regressions are caught automatically.
3. **Operational runbook hardening**: Document the analytical layer's startup sequence, failure modes, and diagnostic procedures.
4. **Non-candle family read path**: Only if there is a concrete consumer need — do not expand speculatively.

## Verdict

**S159 complete.** The analytical layer has concrete evidence of end-to-end integration. The last S156 blocker is closed. The foundation is ready for read-path hardening and operability improvements.
