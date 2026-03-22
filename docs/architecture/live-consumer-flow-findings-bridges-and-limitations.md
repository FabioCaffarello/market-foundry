# Live Consumer Flow — Findings, Bridges, and Limitations

> S333 findings from proving the NATS consumer to actor live flow.

## 1. Gap Fixed: Consumer Lifecycle in ExecuteSupervisor

**Finding:** The `ExecuteSupervisor` created a NATS `Consumer` in `start()` but did
not store a reference to it. On actor stop, the consumer's NATS connection and
goroutines persisted indefinitely, causing:

- Durable consumer lock held after binary shutdown (prevents new consumer binding)
- Orphaned goroutines in the consumer's `Consume` callback
- Connection leak to NATS server

**Fix:** Added `consumer` field to `ExecuteSupervisor` and `Close()` call in the
`actor.Stopped` handler. The consumer's NATS connection and goroutine are now
properly released when the actor is poisoned.

**File:** `internal/actors/scopes/execute/execute_supervisor.go`

## 2. Transitional Bridge: Paper Order Subject Mapping

The execute binary's intake consumer subscribes to `execution.events.paper_order.submitted.>`
because derive currently only produces `PaperOrderSubmittedEvent`. This is a **transitional
bridge** documented in:

- `execute_supervisor.go` (comment at consumer creation)
- `registry.go` (`ExecuteVenueMarketOrderIntakeConsumer` docstring)
- `messages.go` (`intentReceivedMessage` type comment)

**When to resolve:** When venue-specific intent subjects are introduced, the intake
consumer's spec will migrate to venue-specific subjects. The `intentReceivedMessage`
type will carry the venue intent event instead of `PaperOrderSubmittedEvent`.

**Risk:** Low. The bridge is well-documented and the migration path is clear.

## 3. JetStream Push Consumer Activation Race

**Finding:** JetStream push consumers have an inherent activation race — there is a
window between `CreateOrUpdateConsumer` and the first `Consume` callback during which
published events may not be delivered to the consumer. Core NATS subscriptions do not
have this race (they become active after `Flush()`).

**Impact on S333:** Integration tests require a 1-second startup delay after spawning
the `ExecuteSupervisor` to ensure the durable consumer is fully active before
publishing events. This is acceptable for testing but worth noting.

**Production mitigation:** Not required. In production, the durable consumer starts
before any events are published (separate binary lifecycle). Events published during
the activation window are retained by JetStream and delivered once the consumer is
active.

## 4. Deduplication Key Granularity

**Finding:** The `ExecutionIntent.DeduplicationKey()` uses `Timestamp.Unix()` (second
precision). Events with timestamps differing only by sub-second amounts share the
same dedup key and are treated as duplicates by JetStream.

**Impact:** In production, this is correct behavior — the derive pipeline produces
at most one execution intent per symbol per timeframe per second. In testing, event
builders must space timestamps by at least 1 second to avoid accidental dedup.

**Risk:** None in production. Test-only concern, addressed in the S333 test helper.

## 5. What Was Proven

| Property | Evidence |
|----------|----------|
| NATS durable consumer delivers to real Hollywood actor | LF-1: fill event on stream |
| Correlation ID preserved across NATS boundary | LF-1: source → fill chain |
| Causation ID links fill to source event | LF-1: fill.CausationID = source.Metadata.ID |
| Consumer restart preserves durable state | LF-2: second supervisor processes new events |
| Kill switch blocks actor despite consumer delivery | LF-3: processed=1, skipped_halt=1, filled=0 |
| Multiple events processed sequentially | LF-4: 3 events → 3 fills |
| Health tracker metrics reflect real delivery | All tests: counter assertions |
| Consumer cleanup on actor stop | LF-2: durable consumer released between phases |

## 6. What Was NOT Proven (Out of Scope for S333)

| Item | Why | Target |
|------|-----|--------|
| Fill event round-trip to downstream consumer | LSI-2 scope (S334) | S334 |
| Kill-switch live state transitions via KV | LSI-3 scope (S335) | S335 |
| `make smoke-live-stack` reproducibility | LSI-3 scope (S335) | S335 |
| Real venue HTTP adapter (Binance testnet) | Requires credentials + activation gate | Post-wave |
| Multi-venue consumer multiplexing | NG-2: single venue first | Post-wave |
| Throughput/latency benchmarks | Not a benchmark stage | Future |

## 7. Residual Risks

| Risk | Level | Mitigation |
|------|-------|------------|
| JetStream activation race in rapid restart | Low | 1s startup delay; durable state handles replay |
| Paper bridge subject mismatch after migration | Low | Well-documented; single migration point |
| Consumer leak if supervisor panics (not poisoned) | Low | Hollywood restart strategy; fixed by S333 cleanup |
