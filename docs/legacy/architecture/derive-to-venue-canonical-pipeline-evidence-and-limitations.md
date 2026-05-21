# Derive-to-Venue Canonical Pipeline — Evidence and Limitations

> S368 — Evidence matrix and gap catalog for the end-to-end
> analytical-to-execution pipeline.
>
> Date: 2026-03-22.

---

## 1. Evidence Matrix

### Pipeline Segments

| Segment | Status | Evidence stage | Test count |
|---------|--------|---------------|------------|
| Derive resolver produces correct StrategyResolvedEvent | VERIFIED | S365, S366 | 20 |
| Derive publisher produces correct NATS messages | VERIFIED | S366 | 29 |
| Store materializes derive-produced events | VERIFIED | S367, S368 | 21 + 6 |
| Gateway returns derive-produced state via HTTP | VERIFIED | S367 | 6 |
| Execute consumer evaluates derive-produced events | VERIFIED | S360, S368 | 15 + 12 |
| Safety gates accept/reject derive-produced events | VERIFIED | S316, S368 | 10 + 2 |
| Venue adapter submits evaluated intents | VERIFIED | S316, S328 | 12 |
| Correlation chain unbroken across all scopes | VERIFIED | S368 | 3 |

### Contract Invariants

| ID | Invariant | Scope | Status |
|----|-----------|-------|--------|
| INV-1 | Type = mean_reversion_entry | derive, execute | VERIFIED E2E |
| INV-2 | Direction→side mapping deterministic | execute | VERIFIED E2E |
| INV-3 | Correlation/causation chain preserved | all scopes | VERIFIED E2E |
| INV-4 | Pass-through risk explicit | execute | VERIFIED E2E |
| INV-5 | Strategy timestamp, not time.Now() | derive, execute | VERIFIED E2E |
| INV-6 | Only mean_reversion_entry consumed | execute | VERIFIED (unit) |
| INV-7 | Flat = no execution | derive, execute | VERIFIED E2E |
| INV-8 | Event schema backward-compatible | transport | NOT TESTED (no schema evolution yet) |
| INV-9 | At-least-once with dedup | transport | VERIFIED (publisher tests) |
| INV-10 | Partition key deterministic | store | VERIFIED E2E |
| INV-11 | Dedup key unique per event | derive | VERIFIED E2E |

---

## 2. Capabilities Assessment

| Capability | Classification | Evidence |
|------------|---------------|----------|
| Derive produces strategy events | FULL | 20 producer invariant tests + 12 E2E tests |
| Strategy events flow to execution | FULL | 12 E2E tests proving connected path |
| Execution produces correct intents | FULL | Direction, quantity, risk, trace all verified |
| Strategy events materialize in store | FULL | 6 E2E store tests + 15 existing |
| Materialized state queryable via HTTP | FULL | Use case test with real derive output |
| Safety gates protect execution path | FULL | Staleness + confidence threshold verified |
| Correlation chain unbroken | FULL | 5-hop chain verified end-to-end |
| Severity scaling flows end-to-end | FULL | 3 severity levels verified |
| Multi-symbol isolation | SUBSTANTIAL | Partition key isolation verified; no E2E multi-symbol pipeline test |
| Analytical storage (ClickHouse) | NOT VERIFIED | Writer path deferred to separate scope |

---

## 3. Remaining Limitations

| ID | Limitation | Impact | Mitigation | Priority |
|----|-----------|--------|------------|----------|
| L1 | Event metadata (correlation_id, causation_id) not persisted in KV | No HTTP-visible trace for operational debugging | ClickHouse analytical path; NATS replay | LOW — operational, not correctness |
| L2 | No multi-binary orchestration test | Pipeline proven in-process, not across OS processes | Live smoke scripts (smoke-venue-integration.sh) cover this | MEDIUM — operational proof exists separately |
| L3 | ClickHouse writer path not verified for strategy events | Analytical completeness gap | Writer pipeline exists; separate verification scope | LOW — not blocking execution proof |
| L4 | squeeze_breakout_entry and trend_following_entry not E2E tested | Only mean_reversion_entry proven | Pattern is mechanical; families share same wiring | LOW — pattern proven, others follow |
| L5 | No rate limiting or backpressure from execution to derive | Under extreme load, execute could lag behind derive | NATS consumer buffering + durable consumer prevents loss | LOW — no operational evidence of issue |
| L6 | No push-based cache invalidation on store read surface | HTTP reflects KV at query time, may be slightly stale | Acceptable for analytical use case | LOW |
| L7 | Confidence threshold gate only on execute consumer, not on derive | Derive produces all events; filtering is at consumption | By design — store needs all events including low-confidence | NOT A BUG |

---

## 4. Controls and Guards

### Production Safety

| Control | Location | Behavior |
|---------|----------|----------|
| Kill switch | VenueAdapterActor | Blocks all venue submissions immediately |
| Staleness guard | VenueAdapterActor | Blocks intents older than configured max age |
| Confidence threshold | StrategyConsumerActor | Skips strategy events below minimum confidence |
| Type filter | StrategyConsumerActor | Only processes mean_reversion_entry |
| Final gate | StrategyProjectionActor | Only materializes finalized strategies |
| Validation gate | StrategyProjectionActor + Derive resolver | Rejects malformed strategies at both ends |
| Monotonicity guard | KV store | Rejects stale/duplicate writes |
| Retry with halt check | RetrySubmitter | Stops retrying if kill switch engages mid-retry |
| Post-200 reconciler | Post200Reconciler | Recovers from body-read-failure without re-submission |

### Observability

| Signal | Location | Purpose |
|--------|----------|---------|
| `strategy_evaluation` counter | StrategyConsumerActor | Track evaluation outcomes (actionable, flat, error) |
| `execution_intent` counter | StrategyConsumerActor | Track intent side distribution |
| Health tracker counters | All actors | received, evaluated, skipped, errors per actor |
| Structured logs with correlation_id | All actors | End-to-end trace in log aggregation |
| Stats invariant check | StrategyProjectionActor | received == sum(outcomes) at shutdown |

---

## 5. Auditability

The pipeline is auditable through three complementary paths:

1. **Structured logs**: Every actor logs with correlation_id, causation_id,
   source, symbol, timeframe, direction, and confidence at each boundary
   crossing. Log aggregation by correlation_id reconstructs the full path.

2. **NATS JetStream**: All events are durably stored in JetStream streams
   (72h retention for strategy events). NATS replay can reconstruct the exact
   event sequence.

3. **KV latest state**: The operational read surface shows the current
   materialized state for each source/symbol/timeframe partition. The HTTP
   endpoint returns the full strategy with decisions, parameters, and metadata.

**Gap**: The KV path loses event metadata (L1). Full audit trail requires
log aggregation or NATS replay, not HTTP alone.

---

## 6. Business Value Demonstration

The S368 proof demonstrates concrete business value for the derive integration wave:

1. **Signal → Strategy → Execution is real**: A decision evaluation
   (RSI oversold on BTCUSDT) produces a mean reversion entry strategy that
   drives a paper buy order through the full safety gate pipeline to venue
   submission. This is not synthetic — the derive resolver applies real severity
   scaling and parameter adjustment.

2. **The system is self-protecting**: Stale events, low-confidence signals,
   and kill switch states all correctly block execution. The safety pipeline
   is proven with real derive output, not just synthetic test payloads.

3. **The state is queryable**: Materialized strategy state is readable via
   HTTP with full field preservation, enabling operational dashboards and
   debugging.

4. **The trace is complete**: Any execution can be traced back to its
   originating decision through the correlation/causation chain, enabling
   post-hoc analysis and incident investigation.

---

## References

- [End-to-End Analytical-to-Execution Proof](end-to-end-analytical-to-execution-proof.md)
- [Derive Integration Wave Charter (S364)](derive-integration-wave-charter-and-scope-freeze.md)
- [Source Selection and Canonical Contract (S359)](source-selection-and-canonical-integration-contract.md)
- [Store/Gateway Read-Path Verification (S367)](store-gateway-and-read-path-verification-for-derive-produced-strategy-events.md)
