# Stage S45 — Decision Adapter & Contract Test Sweep

**Status:** Complete
**Date:** 2026-03-17

## Objective

Increase structural confidence in the `decision` domain by adding disciplined test coverage to adapters, registries, KV stores, and contracts — without scope expansion or redesign.

## Summary

The `decision` domain had existing test coverage for domain validation, use-case input validation, HTTP handlers, and route registration, but was missing critical adapter-layer tests. This sweep closes those gaps and hardens the evaluator's edge-case coverage.

## Targets Covered

### 1. `decision_registry_test.go` (NEW — 22 test assertions)
- Subject taxonomy validation (NATS subject naming conventions)
- Versioned type contracts (`.v1.` in event/request/reply types)
- Stream configuration invariants (name, subjects, max age, max bytes)
- Query spec lookup behavior (`LatestSpecByType`) — registered, unknown, empty, cross-domain boundary (signal type ≠ decision type)
- Consumer spec validation: durable naming convention (hyphens not underscores), wildcard routing, stream binding, bounded max deliver, positive ack wait, decision-type subject filtering

### 2. `decision_kv_store_test.go` (NEW — 10 test cases)
- Nil guard for Put and Get (returns `Unavailable` problem)
- Uninitialized guard for Put and Get (constructed but not started)
- Constructor field assignment validation
- Bucket constant value assertion
- Multi-symbol key isolation (3 symbols × 2 timeframes = 6 unique keys)
- Multi-source key isolation (3 sources)
- Close nil-safety (nil store + unstarted store)
- Get key format consistency with PartitionKey (Put/Get key agreement)

### 3. `rsi_oversold_evaluator_test.go` (ENHANCED — 6 new test cases added)
- Empty signal value rejection
- Confidence bounds validation across 7 RSI values (all in [0.5, 1.0])
- Confidence monotonicity: further from threshold → higher confidence (both sides)
- Negative RSI edge case: confidence capped at 1.0
- Timestamp preservation through evaluation
- Signal input preservation (type, value, timeframe)

### 4. `decision_test.go` (ENHANCED — 3 new test cases)
- Precise deduplication key format assertion (exact string match with known timestamp)
- Deduplication key uniqueness across different timestamps
- Deduplication key uniqueness across different decision types

## Files Changed

| File | Action | Tests Added |
|------|--------|-------------|
| `internal/adapters/nats/decision_registry_test.go` | Created | 22 assertions in 3 top-level tests |
| `internal/adapters/nats/decision_kv_store_test.go` | Created | 10 test cases |
| `internal/application/decision/rsi_oversold_evaluator_test.go` | Enhanced | 6 new test cases |
| `internal/domain/decision/decision_test.go` | Enhanced | 3 new test cases |

## Test Results

All decision-domain packages pass:

```
ok  internal/adapters/nats            (decision registry + KV store tests)
ok  internal/domain/decision          (12 tests — was 9)
ok  internal/application/decision     (13 tests — was 7)
ok  internal/application/decisionclient (3 tests — unchanged)
ok  internal/interfaces/http/handlers (4 decision tests — unchanged)
ok  internal/interfaces/http/routes   (3 decision tests — unchanged)
```

## Invariants Now Tested

1. **Registry contracts are stable** — subject names, versioned types, stream config, consumer specs
2. **KV store guards are sound** — nil/uninitialized access returns `Unavailable`, not panic
3. **Key isolation is guaranteed** — different symbols, timeframes, and sources produce distinct KV keys
4. **Put/Get key agreement** — both use `{source}.{symbol}.{timeframe}` format
5. **Evaluator confidence is bounded** — always in [0.5, 1.0], monotonically increasing away from threshold
6. **Evaluator rejects invalid input** — empty string, non-numeric values return `false`
7. **Deduplication keys are deterministic and unique** — different timestamps/types produce different keys
8. **Domain validation passes for evaluator output** — evaluator always produces valid decisions

## Remaining Gaps

1. **`decision_publisher.go`** — no unit test (requires live NATS; integration-test territory)
2. **`decision_consumer.go`** — no unit test (requires live JetStream consumer; integration-test territory)
3. **`decision_gateway.go`** — no unit test (requires NATS request/reply; integration-test territory)
4. **`decision_evaluator_actor.go` / `decision_publisher_actor.go`** — actor-layer tests require Proto.Actor harness (out of scope for adapter sweep)
5. **`decision_consumer_actor.go` / `decision_projection_actor.go`** — same as above
6. **Monotonicity guard in KV Put** — tested structurally (key format), but full write-read-write cycle requires live NATS

These gaps are infrastructure-integration concerns, not contract or logic gaps. They do not block `strategy` readiness.

## Impact on Readiness

- **S46/S47**: Decision domain now has the same adapter-test discipline as `signal` and `evidence`. No structural blockers remain.
- **S49 (strategy)**: `strategy` can safely depend on decision contracts — registry subjects, KV key format, evaluator behavior, and query contracts are all tested.
- **Confidence**: The decision domain moves from "implemented but undertested" to "structurally validated at adapter, domain, and application layers."
