# Multi-Binary Canonical Pipeline: Evidence and Limitations

> Stage S373 — Honest accounting of what was proven, what was not, and residual gaps.

## Evidence Summary

### What Was Proven

| Claim | Evidence | Confidence |
|-------|----------|------------|
| derive produces StrategyResolvedEvent and publishes to NATS | Integration test S373-MB-1: publisher publishes, fill received | High |
| execute consumes strategy events across binary boundary | Integration test S373-MB-1: separate NATS connections, fill received with correct correlation_id | High |
| Correlation chain preserved from derive to fill | All S373 integration tests verify `fill.Metadata.CorrelationID == original` | High |
| Direction→side mapping correct across boundary | S373-MB-4: long→buy, short→sell, flat→none all proven with real NATS | High |
| Control gate blocks fills across binaries | S373-MB-2: halted gate → skipped_halt counter, resumed → fill received | High |
| Control KV readable from separate binary | S373-MB-3: separate NATS connection reads gate status | High |
| Strategy type identity preserved | S373-MB-1: `fill.ExecutionIntent.Risk.StrategyType == mean_reversion_entry` | High |
| Staleness guard functional | Structural test: fresh passes, 10-min-old blocked | High |
| Store materializes to NATS KV | Smoke Phase 7: `GET /strategy/mean_reversion_entry/latest → 200` | High (compose only) |
| Writer persists to ClickHouse | Smoke Phase 9: `SELECT count() FROM strategies > 0` | High (compose only) |
| Gateway reads from store | Smoke Phase 7: HTTP endpoints return materialized data | High (compose only) |
| All 9 services boot healthy | Smoke Phase 1: all services healthy in compose | High (compose only) |
| 44+ durable consumers bound | S372 compose wiring smoke (prerequisite) | High |

### Correlation Chain Trace

The canonical correlation chain through the pipeline:

```
derive:                     correlation_id=X  →  StrategyResolvedEvent
  ↓ (NATS STRATEGY_EVENTS)
execute.StrategyConsumer:   correlation_id=X  →  intentReceivedMessage
execute.VenueAdapter:       correlation_id=X  →  PaperOrderSubmittedEvent (synthetic)
execute.VenueAdapter:       correlation_id=X  →  VenueOrderFilledEvent
  ↓ (NATS EXECUTION_FILL_EVENTS)
store:                      correlation_id=X  →  NATS KV materialization
writer:                     correlation_id=X  →  ClickHouse row
gateway:                    correlation_id=X  →  HTTP composite chain
```

Every hop preserves the original `correlation_id`. Causation IDs chain sequentially: each event's `causation_id` points to the previous event's `id`.

### Tracker Metrics as Evidence

The execute binary's `/statusz` endpoint exposes counters that serve as runtime evidence:

| Counter | Meaning | S373 Validation |
|---------|---------|-----------------|
| `strategy-consumer.received` | Events consumed from NATS | Integration + smoke |
| `strategy-consumer.evaluated` | Events that passed type check | Integration + smoke |
| `strategy-consumer.evaluated_actionable` | Events with actionable direction | Integration |
| `strategy-consumer.evaluated_flat` | Events with flat direction | Integration |
| `venue-adapter.processed` | Intents that reached venue adapter | Integration + smoke |
| `venue-adapter.filled` | Intents that produced fills | Integration + smoke |
| `venue-adapter.skipped_halt` | Intents blocked by control gate | Integration (MB-2) |

## Limitations and Gaps

### L1: Single Strategy Family Only

**What:** Only `mean_reversion_entry` was exercised. `trend_following_entry` and `squeeze_breakout_entry` were not part of the E2E proof.

**Why acceptable:** The architecture is family-generic — the NATS subject routing and consumer binding pattern is identical for all three families. The risk is low because the routing is parameterized, not coded per-family.

**Residual risk:** A future family might introduce a payload shape that breaks a downstream consumer's assumption.

### L2: Single Symbol, Single Timeframe

**What:** Only `binancef/btcusdt/60` was exercised.

**Why acceptable:** Multi-symbol support was proven in earlier stages (S220 smoke-multi-symbol). The multi-binary boundary is symbol-agnostic — NATS subject routing uses `{source}.{symbol}.{timeframe}` as partition keys.

**Residual risk:** High-cardinality symbol spaces may surface consumer lag or KV bucket contention not visible with a single symbol.

### L3: Paper Venue Only

**What:** The proof uses `PaperVenueAdapter` (simulated fills). No real exchange API was exercised.

**Why acceptable:** The venue adapter is a leaf node — it receives an intent and produces a fill. The multi-binary pipeline proof focuses on the boundary between derive and execute, not the venue API. Real venue integration is a separate concern.

**Residual risk:** Real venue adapters may introduce latency, errors, or retries that affect the fill publication timing.

### L4: No Writer Flush Timing Guarantee

**What:** The smoke test checks ClickHouse row counts but does not guarantee the writer flushed during the test window.

**Why acceptable:** Writer flushes are batch-based with configurable intervals. The smoke warns (not fails) when counts are zero.

**Residual risk:** In fast smoke runs, the writer may not have flushed yet, producing a false warning.

### L5: Composite Chain Coverage Depends on Pipeline Activity

**What:** Phase 10 (correlation chain audit) requires that the pipeline has produced at least one chain with both strategy and execution stages. This depends on market conditions producing actionable signals during the test window.

**Why acceptable:** The smoke warns (not fails) when no full chain exists. The integration tests deterministically produce actionable events and verify the chain.

### L6: No Multi-Binary Restart/Recovery Proof

**What:** S373 does not prove that the pipeline recovers correctly after a binary restart (e.g., derive crashes and restarts, execute replays from JetStream).

**Why acceptable:** Restart/recovery is covered by `smoke-restart-recovery.sh` and JetStream's durable consumer guarantees. S373 focuses on steady-state correctness.

**Residual risk:** Consumer replay after restart may re-deliver events that trigger deduplication edge cases.

### L7: No Back-Pressure or Load Testing

**What:** All proofs use low-volume, steady-state event rates. No burst or sustained load testing was performed.

**Why acceptable:** Load testing is an operational concern orthogonal to correctness. The multi-binary architecture's value is in isolation and controllability, not throughput.

## What This Means for the Wave

The S370–S373 wave set out to prove that the multi-binary architecture is not just structurally sound but functionally correct under real conditions. S373 closes this goal:

- **S371** documented the boundaries and contracts.
- **S372** proved the wiring is connected.
- **S373** proved that data flows correctly through those connections.

The pipeline is ready for production-oriented hardening (monitoring, alerting, load testing) but the core correctness is established.
