# Live-Listen + Dry-Run Canonical Pipeline: Evidence and Limitations

**Stage:** S380
**Wave:** Exchange Listening and Dry-Run Foundation (S376–S381)

---

## Purpose

This document consolidates the evidence collected during S380 and the
remaining limitations that future waves must address. It serves as the
evidence gate reference for the wave closure stage (S381).

## Evidence Matrix

### Pipeline Segment Evidence

| Segment | Proven by | Evidence type |
|---|---|---|
| Exchange → WebSocket | S378 smoke (Phase 5) | Log inspection: connection activity |
| WebSocket → OBSERVATION_EVENTS | S378 smoke (Phase 6), S380 smoke (Phase 3) | NATS monitoring: message count growth |
| OBSERVATION_EVENTS → derive | S378 smoke (Phase 7), S380 smoke (Phase 4) | Consumer delivered count > 0 |
| Derive → evidence sampling | S380 smoke (Phase 8) | Gateway: `/evidence/candles/latest` → 200 |
| Evidence → signal → decision | Inferred from strategy production | STRATEGY_EVENTS delta > 0 |
| Decision → strategy resolution | S380 smoke (Phase 4) | STRATEGY_EVENTS count growth |
| Strategy → risk assessment | Inferred from execution intent | Execute strategy-consumer received > 0 |
| Risk → execution intent | S380 smoke (Phase 5) | Execute statusz: evaluated_actionable |
| Intent → DryRunSubmitter | S380 test S380-DR-1 | dryrun- prefix on VenueOrderID |
| DryRunSubmitter → fill event | S380 test S380-DR-1 | Simulated=true, StatusFilled |
| Fill → EXECUTION_FILL_EVENTS | S380 test S380-DR-1, smoke (Phase 7) | NATS stream message count |
| Fill → store KV | S380 smoke (Phase 8) | Gateway HTTP read path → 200 |
| Fill → writer → ClickHouse | S380 smoke (Phase 9) | ClickHouse row count > 0 |
| Correlation chain | S380 test S380-DR-1, smoke (Phase 10) | correlation_id preserved derive→execute→fill |

### Safety Evidence

| Safety property | Proven by | Evidence type |
|---|---|---|
| DryRunSubmitter never delegates | S380-DR-5 (bomb-adapter) | Panic-free execution with bomb inner |
| Config fail-closed | S379 TestS379_DryRunConfig_FailClosed | nil DryRun → IsDryRun() = true |
| Paper + dry_run=false rejected | S379 TestS379_DryRunConfig_Validation | Validation returns problem |
| Control gate blocks before dry-run | S380-DR-3 | skipped_halt > 0, dryrun_intercepted = 0 |
| Staleness guard works with live data | S380 smoke (implicit) | 120s max age with live timestamps |
| Activation surface non-live | S380 smoke (Phase 2) | effective = paper |
| No venue_live in logs | S380 smoke (Phase 2) | grep count = 0 |
| Unique dry-run order IDs | S380-DR-4 | 3 events, 3 unique IDs |
| Flat direction produces no-action | S380-DR-2 | SideNone, StatusAccepted |

### Auditability Evidence

| Audit dimension | Mechanism | Evidence |
|---|---|---|
| Order identification | `dryrun-{128-bit-hex}` prefix | All fills carry prefix (S380-DR-1, DR-4) |
| Simulation marking | `Simulated: true` on fill records | S380-DR-1 asserts Simulated=true |
| Structured logging | `"dry-run intercepted venue submit"` | S380 smoke Phase 6 |
| Health counters | `dryrun_intercepted`, `dryrun_filled`, `dryrun_noop` | S380-DR-1 counter assertions |
| Correlation chain | `CorrelationID` preserved through all layers | S380-DR-1 assertion |
| Causation chain | `CausationID` preserved through all layers | Implicit via StrategyResolvedEvent metadata |

## Canonical Pipeline Under Test

```
Exchange (Binance Futures mainnet)
  │
  │ wss://fstream.binance.com/ws/{symbol}@aggTrade
  │
  ▼
ingest: ParseAggTrade → Normalize → Validate → NATS Publish
  │
  │ OBSERVATION_EVENTS stream (Msg-Id dedup)
  │
  ▼
derive: [per binding/source scope]
  │
  ├── Sampler (candle, tradeburst, volume)  →  EVIDENCE_EVENTS
  ├── Signal  (rsi)                         →  SIGNAL_EVENTS
  ├── Decision (rsi_oversold)               →  DECISION_EVENTS
  ├── Strategy (mean_reversion_entry)       →  STRATEGY_EVENTS
  ├── Risk (position_exposure)              →  RISK_EVENTS
  └── Execution (paper_order)               →  EXECUTION_EVENTS
  │
  │ STRATEGY_EVENTS: strategy.events.mean_reversion_entry.resolved.>
  │
  ▼
execute: [StrategyConsumerActor]
  │
  ├── PaperOrderEvaluator: strategy → execution intent
  ├── Correlation/causation preservation
  ├── Direction→Side mapping (long→buy, short→sell, flat→none)
  │
  ├── VenueAdapterActor:
  │   ├── Safety gate: kill switch (NATS KV EXECUTION_CONTROL)
  │   ├── Safety gate: staleness (120s max age)
  │   │
  │   └── DryRunSubmitter (outermost decorator, S379):
  │       ├── Intercepts SubmitOrder
  │       ├── Produces VenueOrderReceipt with dryrun-{hex} ID
  │       ├── Fill: Price="0", Quantity=intent, Fee="0", Simulated=true
  │       └── Never calls inner pipeline
  │
  │ EXECUTION_FILL_EVENTS: execution.fill.venue_market_order
  │
  ├── store: ProjectionActor → KV bucket materialization
  ├── writer: ClickHouse persistence
  └── gateway: HTTP query surface
```

## Limitations

### L1: Price Realism

**Status:** Known, accepted for foundation wave.

Dry-run fills use `Price: "0"`. This is intentionally unrealistic for the
proof-of-concept phase. Realistic P&L simulation requires injecting the
last-known market price from the evidence pipeline.

**Impact:** Downstream analytics that compute returns or P&L from dry-run
fills will show zero returns. The structural pipeline is correct; only the
price value is synthetic.

**Mitigation path:** A future stage can enhance DryRunSubmitter to accept a
price provider function, sourcing the last candle close or best bid/ask.

### L2: No Runtime Dry-Run Toggle

**Status:** Deferred by design.

Changing `venue.dry_run` requires a binary restart. A NATS KV-based runtime
toggle was considered but deferred — it adds complexity and risk surface to
a safety-critical flag. The kill switch (control gate) provides runtime
halt/resume without changing the dry-run/live boundary.

**Impact:** Transitioning from dry-run to live requires a deploy cycle, not
a runtime API call. This is conservative by design.

### L3: Single Exchange

**Status:** Architectural boundary, not a bug.

Only Binance Futures is wired (mainnet for market data, testnet for venue
adapter). Adding exchanges requires new adapter implementations with
exchange-specific WebSocket parsing and REST API integration.

### L4: No Latency Measurement

**Status:** Out of scope.

WebSocket-to-NATS publish latency and end-to-end pipeline latency (trade
arrival → fill publication) are not quantified. The proof validates
correctness, not performance.

### L5: No Throughput Assertion

**Status:** Out of scope.

Smoke validation checks for "at least one event." Sustained throughput under
high-frequency trading data is not validated. The proof is correctness-focused.

### L6: No Backpressure

**Status:** Known limitation from S378.

WebSocket reads are not paused when NATS publish is slow. Under sustained
high throughput, this could cause memory growth in the ingest binary. A
future stage should implement flow control.

### L7: Simulated Fill Shape

**Status:** Acceptable for proof.

Dry-run fills have a single fill record with `Quantity = intent.Quantity`.
Real venue fills may return partial fills (multiple fill records). The
dry-run path does not simulate partial fill scenarios.

### L8: Pipeline Timing Dependency

**Status:** Inherent to live data proofs.

The smoke script depends on the market being active enough to produce trades
during the polling window. During very low-activity periods (weekends,
holidays), the pipeline may not produce strategy events within the default
wait time. This is mitigated by the `SMOKE_WAIT` override.

## Wave Objective Assessment

| Wave objective | Status |
|---|---|
| Exchange real listening proven | PASS (S378, confirmed by S380 Phase 3) |
| Dry-run execution path proven | PASS (S379, confirmed by S380 tests) |
| End-to-end pipeline with live + dry-run | PASS (S380) |
| Safety gates respected in dry-run mode | PASS (S380-DR-3) |
| Auditability and correlation preserved | PASS (S380-DR-1, smoke Phase 10) |
| No real trading possible in default config | PASS (FC-8 through FC-11, activation surface) |

## References

- [End-to-end live-listen + dry-run proof](end-to-end-live-listen-plus-dry-run-proof.md) — proof structure and flow diagram
- [Dry-run execution path by config](dry-run-execution-path-by-config.md) — S379 config model
- [Dry-run submitter fail-closed semantics](dry-run-submitter-fail-closed-semantics-and-auditability.md) — FC-8 through FC-11
- [Compose live exchange listening proof](compose-live-exchange-listening-proof.md) — S378 live data proof
- [Exchange ingress contracts and runtime mode model](exchange-ingress-contracts-and-runtime-mode-model.md) — S377 contract invariants
