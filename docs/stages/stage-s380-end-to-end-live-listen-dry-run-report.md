# Stage S380 — End-to-End Live-Listen + Dry-Run Proof

**Wave:** Exchange Listening and Dry-Run Foundation (S376–S381)
**Predecessor:** S379 (Dry-Run Execution Path by Config)
**Status:** COMPLETE

---

## Executive Summary

S380 delivers the capstone proof of the exchange listening and dry-run
foundation wave: **an end-to-end pipeline from live exchange data through
the canonical signal/strategy/execution path to dry-run fills**, with full
auditability, correlation preservation, and safety guarantees. No real
trading occurs. The system is proven to operate safely with live market
data while the write path is governed by config-driven dry-run semantics.

## Objective

Execute, validate, and document a proof connecting:

```
exchange real listening → source/signal/strategy path → activation/control →
dry-run execution sink → persistence/read/explain
```

in a controlled, auditable manner without any real trading request.

## Deliverables

### Code Changes

| File | Change |
|---|---|
| `internal/actors/scopes/execute/s380_live_listen_dry_run_test.go` | **New.** 5 integration tests proving DryRunSubmitter in multi-binary pipeline |
| `scripts/smoke-e2e-live-listen-dry-run.sh` | **New.** 12-phase smoke combining live listening + dry-run + multi-binary pipeline |
| `Makefile` | Added `smoke-live-dry-run` target and smoke-help entry |

### Tests

| Test | What it proves |
|---|---|
| `TestS380_LiveListenDryRun_FullPipeline` | Complete pipeline: derive → NATS → execute (DryRunSubmitter) → dry-run fill. dryrun- prefix, Simulated=true, correlation preserved, health counters. |
| `TestS380_LiveListenDryRun_FlatDirectionNoAction` | Flat → SideNone → DryRunSubmitter returns StatusAccepted, no fills, dryrun- prefix |
| `TestS380_LiveListenDryRun_ControlGateStillBlocks` | Safety gates block BEFORE DryRunSubmitter. Gate halted → skipped_halt, dryrun_intercepted=0 |
| `TestS380_LiveListenDryRun_UniqueOrderIDsAcrossPipeline` | 3 events → 3 unique dryrun-{hex} IDs across multi-binary pipeline |
| `TestS380_DryRunSubmitter_NeverDelegatesInPipelineContext` | Bomb-adapter test: all 3 sides (buy/sell/none) survive without panic |

### Documentation

| Document | Content |
|---|---|
| `docs/architecture/end-to-end-live-listen-plus-dry-run-proof.md` | Proof structure, pipeline flow diagram, safety guarantees, how to run |
| `docs/architecture/live-listen-dry-run-canonical-pipeline-evidence-and-limitations.md` | Evidence matrix, safety evidence, auditability, limitations (L1–L8), wave objective assessment |
| `docs/stages/stage-s380-end-to-end-live-listen-dry-run-report.md` | This report |

## Pipeline Validated

```
Exchange (Binance Futures mainnet, live)
    ↓ WebSocket aggTrade
ingest → OBSERVATION_EVENTS
    ↓ derive-observation consumer
derive → evidence → signal → decision → strategy → risk → execution
    ↓ STRATEGY_EVENTS (StrategyResolvedEvent)
execute → StrategyConsumerActor → PaperOrderEvaluator → ExecutionIntent
    ↓ VenueAdapterActor
    ├── Safety gate: kill switch (NATS KV)
    ├── Safety gate: staleness guard (120s)
    ↓
    DryRunSubmitter (outermost decorator)
    ├── Intercepts SubmitOrder
    ├── Produces dryrun-{hex} receipt
    ├── Simulated=true on all fills
    ↓
    VenueOrderFilledEvent → EXECUTION_FILL_EVENTS
    ├── store → KV materialization → gateway HTTP
    ├── writer → ClickHouse → analytical queries
    └── correlation chain preserved end-to-end
```

## Evidence Summary

### Integration Test Evidence

```
=== RUN   TestS380_LiveListenDryRun_FullPipeline
  [S380-DR-1] derive published: correlation_id=... direction=long
  [S380-DR-1] venue_order_id=dryrun-... (dryrun- prefix confirmed)
  [S380-DR-1] fill: venue_order_id=dryrun-... side=buy status=filled simulated=true dryrun_intercepted=1
  [S380-DR-1] PASS — full pipeline: derive → NATS → execute → DryRunSubmitter → dry-run fill

=== RUN   TestS380_LiveListenDryRun_FlatDirectionNoAction
  [S380-DR-2] venue_order_id=dryrun-... side=none (flat/no-action confirmed)
  [S380-DR-2] PASS — flat direction → DryRunSubmitter no-action receipt

=== RUN   TestS380_LiveListenDryRun_ControlGateStillBlocks
  [S380-DR-3] PASS — control gate blocks before DryRunSubmitter (safety gates > dry-run)

=== RUN   TestS380_LiveListenDryRun_UniqueOrderIDsAcrossPipeline
  [S380-DR-4] 3 unique dryrun- order IDs confirmed across pipeline
  [S380-DR-4] PASS — DryRunSubmitter generates unique order IDs in multi-binary context

=== RUN   TestS380_DryRunSubmitter_NeverDelegatesInPipelineContext
  [S380-DR-5] PASS — DryRunSubmitter never delegates for any side (bomb adapter survived)
```

### Smoke Evidence (12 phases)

| Phase | Evidence |
|---|---|
| 1. Stack Readiness | All 9 services healthy |
| 2. Dry-Run Mode | effective=paper, dry_run=true logged, no venue_live |
| 3. Live Exchange Data | OBSERVATION_EVENTS delta > 0 |
| 4. Strategy Production | STRATEGY_EVENTS delta > 0 from live data |
| 5. Execute Consumption | strategy-consumer received > 0 |
| 6. Dry-Run Fill Evidence | dryrun- prefix in logs, interception counters |
| 7. Fill Stream | EXECUTION_FILL_EVENTS populated |
| 8. Store Materialization | Strategy/candle latest → 200, control gate accessible |
| 9. Analytical Persistence | ClickHouse candles/strategies row count > 0 |
| 10. Correlation Chains | Composite chains endpoint, correlation_id present |
| 11. Integration Tests | S380 test suite passes |
| 12. Stream Deltas | All key streams show growth during proof |

## Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| End-to-end proof with live exchange listening + dry-run | PASS |
| Operational value of the wave concretely demonstrated | PASS |
| Primary objective of the foundation wave closed | PASS |
| Residual gaps explicitly documented | PASS (L1–L8) |

## Guard Rails Compliance

| Guard rail | Status |
|---|---|
| No multiple parallel slices | PASS — single pipeline (mean_reversion_entry) |
| No new families opened in batch | PASS — uses existing families only |
| No real trading as principal goal | PASS — DryRunSubmitter intercepts all |
| No inflation beyond canonical pipeline | PASS — single symbol, single timeframe |

## Remaining Limitations

1. **L1: Price realism.** Dry-run fills use `Price: "0"`.
2. **L2: No runtime toggle.** Changing dry_run requires binary restart.
3. **L3: Single exchange.** Only Binance Futures wired.
4. **L4: No latency measurement.** Pipeline latency not quantified.
5. **L5: No throughput assertion.** Minimum-one-event check, not sustained volume.
6. **L6: No backpressure.** WebSocket reads unbounded when NATS slow.
7. **L7: Simulated fill shape.** Single fill record, no partial fills.
8. **L8: Pipeline timing.** Depends on market activity during smoke window.

See [`../architecture/live-listen-dry-run-canonical-pipeline-evidence-and-limitations.md`](../architecture/live-listen-dry-run-canonical-pipeline-evidence-and-limitations.md) for detailed limitation analysis.

## Preparation for S381 (Wave Evidence Gate)

S381 should serve as the evidence gate for the exchange listening and dry-run
foundation wave. Recommended scope:

1. **Evidence reconciliation:** Verify all wave stages (S376–S380) produced
   their documented deliverables and all tests pass.
2. **Cross-stage consistency:** Verify that S380 references and extends
   (not contradicts) S377 contracts, S378 live proof, S379 dry-run proof.
3. **Limitation triage:** Classify L1–L8 as "acceptable for wave closure"
   or "blocking" and document the decision.
4. **Next wave recommendation:** Based on remaining gaps, recommend whether
   the next wave should focus on:
   - Price realism and P&L simulation
   - Multi-exchange expansion
   - Testnet venue integration (dry_run=false with real testnet)
   - Performance and latency characterization
5. **Documentation promotion:** Promote key S380 architecture docs to the
   long-term reference surface if they pass the gate.

## Verdict

**S380: PASS.** The system is proven to operate end-to-end with live
exchange data flowing through the canonical pipeline while the execution
path is safely governed by a config-driven dry-run submitter. This closes
the primary objective of the exchange listening and dry-run foundation wave.
