# Segment-Safe Routing and Leakage Hardening

> Architecture document | S401 | 2026-03-22

## Context

After S400 introduced multi-segment runtime (Spot and Futures coexisting in a
single execute binary), the system has multiple defense layers preventing
cross-segment leakage. S397 identified "cross-segment intent leakage" as a
theoretical low-severity risk. S401 hardens the remaining vectors where an
intent from one segment could theoretically reach an adapter for another.

## Leakage Vectors Assessed

### Vector 1: NATS Consumer Wildcard Subscription

**Pre-S401:** The execute binary's intake consumer subscribed to
`execution.events.paper_order.submitted.>`, receiving ALL sources regardless of
which segments were enabled.

**Risk:** If the execute binary is configured for spot-only but receives a
futures intent from the NATS stream, the intent would reach the VenueAdapterActor.
The SegmentRouter would reject it (fail-closed), but the intent still transits
the process boundary unnecessarily.

**Mitigation (S401):** New `ExecuteVenueIntakeConsumerForSegments(sources)` factory
generates segment-scoped filter subjects. A spot-only config subscribes to
`execution.events.paper_order.submitted.binances.>` — futures intents never
enter the consumer. For unified configs, `FilterSubjects` (plural) carries both
`binances.>` and `binancef.>`.

### Vector 2: Missing Application-Layer Source Guard

**Pre-S401:** The VenueAdapterActor's `onIntent` path relied entirely on the
SegmentRouter for source validation. No intermediate guard existed.

**Risk:** A future refactor could bypass the router or introduce an alternate
code path without source validation.

**Mitigation (S401):** Added `AllowedSources` map to `VenueAdapterConfig`. The
`onIntent` method now checks `AllowedSources` before any further processing
(Gate 0). Intents from unlisted sources are rejected with counter tracking and
structured logging.

### Vector 3: Producer-Side Source Stamping (Already Protected)

Every event in the pipeline carries its source, stamped at the producer:
- Observation: exchange adapter `Normalize()` stamps source
- Execution: `ExecutionIntent.Source` flows from derive through execute
- Fill/Rejection: echoes the intent's source in the published event

No hardening needed — the producer-side stamping is already enforced.

### Vector 4: KV Bucket Composite Keys (Already Protected)

KV keys include source + symbol + timeframe (e.g., `binances:ethusdt:60`).
Cross-segment collision is impossible without source manipulation upstream.

No hardening needed.

### Vector 5: Config Validation (Already Protected)

`VenueConfig.Validate()` rejects:
- Futures adapter on spot segment slot (and vice versa)
- Enabled segment without adapter
- paper_simulator as segment adapter

No hardening needed.

## Defense-in-Depth Model After S401

| Layer | Component | Scope | Added By |
|-------|-----------|-------|----------|
| L0 | Config validation | Startup-time | S393 |
| L1 | NATS consumer filter subjects | Subscription-time | **S401** |
| L2 | VenueAdapterActor source guard | Message-time | **S401** |
| L3 | SegmentRouter source->segment dispatch | Call-time | S400 |
| L4 | Producer-side source stamping | Event-time | Foundational |
| L5 | NATS subject partitioning (source in subject) | Observable | Foundational |
| L6 | Composite KV keys | Query-time | Foundational |

## Trade-offs

- **FilterSubjects vs FilterSubject:** Using `FilterSubjects` (plural) for
  multi-segment requires NATS server >= 2.10. The codebase already targets
  this version range. Single-segment configs use a single `FilterSubjects`
  entry, which is functionally equivalent to `FilterSubject`.

- **AllowedSources guard is redundant:** The source guard in VenueAdapterActor
  duplicates the SegmentRouter's own rejection logic. This is intentional —
  defense-in-depth means the same invariant is enforced at multiple layers.
  The cost is one map lookup per intent (negligible).

- **Legacy config compatibility:** When no segments are configured (legacy
  `venue.type` mode), `EnabledSegmentSources()` returns nil, and both the
  consumer filter and source guard fall back to accept-all behavior.
  This preserves backwards compatibility.

## Residual Risk

| Risk | Severity | Mitigation |
|------|----------|------------|
| Source string spoofing in NATS | Very Low | Requires NATS publish access; production NATS uses auth |
| New source prefix added without updating mapping | Low | `SegmentForSource` returns empty -> SegmentRouter rejects |
| Consumer durable name conflict on filter change | Low | NATS `CreateOrUpdateConsumer` handles config migration |
