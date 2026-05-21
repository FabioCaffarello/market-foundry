# Sustained Activation Observations, Signals, and Limitations

> S343: Behavioral observations, operational signals, and residual gaps from extended venue path observation.

## Observations

### 1. Counter Invariant Holds Under All Conditions

The fundamental invariant `processed == filled + skipped_halt` was validated:

- After every individual event injection (not just at test boundaries)
- Across gate transitions mid-window
- After burst sequences
- After idle pauses
- At the final snapshot

No violation was observed across any of the three test scenarios. This is evidence that the safety gate check and counter increment paths are correctly synchronized in the actor model.

### 2. Gate Remains Responsive Over Minutes

Gate transitions (NATS KV PUT → actor gate check) remain consistently fast:

- 200ms KV propagation delay is sufficient even after minutes of operation
- No evidence of KV read latency degradation over time
- Multiple round-trip transitions (halted→active→halted→active) in a single supervisor session all behave identically to the first transition

### 3. Venue HTTP Requests Track Fills Exactly

`venue_requests == filled` holds at every snapshot. This proves:

- No duplicate venue submissions (each fill corresponds to exactly one HTTP request)
- No phantom HTTP requests (no request without a corresponding fill)
- No requests during halted periods

### 4. No Idle Drift

During 20-second pause periods between bursts (EOW-3):

- `processed`, `filled`, `skipped_halt`, and `errors` all remain frozen
- No background goroutine is spuriously incrementing counters
- The healthz heartbeat loop runs without side effects on event counters

### 5. No Error Accumulation

Zero errors across all scenarios. The httptest.Server provides deterministic responses, so this confirms:

- No timeout accumulation from the actor engine
- No connection pool exhaustion from repeated HTTP requests
- No serialization/deserialization failures over time

### 6. Burst-and-Pause Pattern Is Safe

Five events in rapid succession followed by a 20-second idle period — repeated three times — produces no anomalies. This proves:

- The actor mailbox handles rapid event sequences without message loss
- JetStream acknowledgment remains correct under burst conditions
- No race between fill publication and the next event's gate check

## Operational Signals

### Positive Signals

| Signal | Evidence |
|--------|----------|
| Counter monotonicity | Counters only increment, never decrement or reset |
| Zero error rate | No errors over minutes of operation |
| Deterministic gate behavior | Gate transitions produce identical behavior in phase 1 and phase 4 |
| Fill/request parity | HTTP requests and fills are 1:1 |
| Idle stability | No phantom activity during pauses |

### Signals Not Observed (Expected)

| Signal | Why |
|--------|-----|
| Idle warning from healthz | Window slightly exceeds 2-minute threshold; expected but not asserted |
| Connection reconnection | NATS remains stable; reconnection path not exercised |
| Rate limiting from venue | httptest.Server does not rate-limit |

## Limitations

| Limitation | Severity | Notes |
|-----------|----------|-------|
| httptest.Server, not live testnet | Medium | Same as S342; proves code path and pipeline wiring, not network behavior |
| 2-minute window, not hours | Low | Proportional to wave scope; hours-scale soak deferred to operational validation |
| Controlled event injection, not production load | Low | Events injected at 5-10s intervals; production may have higher concurrency |
| Single supervisor session | Low | No restart during observation; restart resilience proven at domain level |
| No concurrent multi-symbol observation | Low | By wave scope design; actor model is symbol-isolated |
| No NATS partition or reconnection during window | Low | NATS remains healthy; reconnection behavior tested in unit scope |
| No memory/goroutine profiling | Low | No explicit resource leak detection; absence of errors is indirect evidence |

## Comparison: S342 vs S343 Gap Closure

| S342 Limitation | S343 Status |
|----------------|-------------|
| No sustained load test (tests run in seconds) — Low | **Closed**: EOW-1 through EOW-3 sustain operation for minutes |
| httptest.Server, not live testnet — Medium | Unchanged (same infrastructure) |
| Single symbol only — Low | Unchanged (by design) |
| No partial fill scenario — Low | Unchanged (out of scope) |
| No binary restart with real adapter — Low | Unchanged (out of scope) |
| Retry submitter not triggered — Low | Unchanged (no retryable failure injected) |

## Conclusion

S343 closes the "extended observation window" gap from S341/S342. The venue activation lifecycle is now proven to be stable over minutes of sustained operation, with multiple gate transitions, burst-and-pause patterns, and continuous counter consistency validation. Remaining limitations are low-severity and align with the wave's scope boundaries.
