# Embedded NATS Integration Proof

## Purpose

This document records the closure of hard blocker **HB-S89-3**: the embedded NATS integration test harness that was required before real venue activation could be formally discussed.

## Approach

An embedded NATS server (`github.com/nats-io/nats-server/v2`) runs in-process for each test. Each test gets its own server instance on a random port with a temporary JetStream store directory, providing full isolation. Tests use the `//go:build integration` build tag and run via `make test-integration`.

## Test Scenarios (11 total)

### Publish/Consume Pipeline

| # | Scenario | What It Proves |
|---|----------|---------------|
| 1 | `PublishExecution_ConsumerReceives` | Paper order event published to EXECUTION_EVENTS stream is correctly consumed by the durable intake consumer. Correlation ID preserved through CBOR encode/decode. |
| 2 | `PublishFill_FillConsumerReceives` | Venue fill event published to EXECUTION_FILL_EVENTS stream is correctly consumed by the fill consumer. VenueOrderID and Simulated=false preserved. |
| 3 | `PublishConsumeProject_Pipeline` | Full pipeline: publish → consumer → handler writes to KV → read back verifies. Proves the projection path works end-to-end with embedded NATS. |
| 4 | `FillPipeline_PublishConsumeProject` | Same as #3 but for the fill family: fill event → fill consumer → KV write → read back with real price and Simulated=false. |

### KV Store Operations

| # | Scenario | What It Proves |
|---|----------|---------------|
| 5 | `ExecutionKV_PutGet` | Basic put/get roundtrip for EXECUTION_PAPER_ORDER_LATEST bucket. Validates JSON marshal/unmarshal and post-read validation. |
| 6 | `ExecutionKV_MonotonicityGuard` | Monotonicity invariant: older timestamps → PutSkippedStale, same timestamps → PutSkippedDuplicate, newer timestamps → PutWritten. Proves the guard works against a real JetStream KV. |

### Control Gate (Kill Switch)

| # | Scenario | What It Proves |
|---|----------|---------------|
| 7 | `ControlGate_Lifecycle` | Full lifecycle: default=active → put halted → verify halted → re-activate → verify active. Tests both Get() and IsHalted() against real KV. |
| 8 | `ControlGate_BlockAndResume` | Kill switch operational flow: active → halt with reason → multiple reads confirm halted → resume → verify active. Simulates real operational halt/resume cycle. |

### Cross-Cutting Invariants

| # | Scenario | What It Proves |
|---|----------|---------------|
| 9 | `JetStream_Deduplication` | Same event published twice (same dedup key) results in exactly 1 consumer delivery. Proves JetStream MsgID-based deduplication works. |
| 10 | `MultiSymbol_Isolation` | 3 symbols published → each gets its own KV entry. No symbol bleed across partition keys. |
| 11 | `ConsumerStats_Tracking` | Consumer stats (delivered, redelivered, terminated, nakked) are accurately tracked across 3 deliveries. Proves observability counters work. |

## HB-S89-3 Coverage Matrix

| Aspect | Covered By | Status |
|--------|-----------|--------|
| Publish execution event | Scenarios 1, 3, 9, 10, 11 | PROVEN |
| Publish fill event | Scenarios 2, 4 | PROVEN |
| Consume execution event | Scenarios 1, 3, 9, 10, 11 | PROVEN |
| Consume fill event | Scenarios 2, 4 | PROVEN |
| KV projection (paper) | Scenarios 3, 5, 6, 10 | PROVEN |
| KV projection (venue) | Scenario 4 | PROVEN |
| Monotonicity guard | Scenario 6 | PROVEN |
| Deduplication | Scenario 9 | PROVEN |
| Control gate lifecycle | Scenarios 7, 8 | PROVEN |
| Multi-symbol isolation | Scenario 10 | PROVEN |
| Consumer stats | Scenario 11 | PROVEN |
| Trace preservation | Scenarios 1, 2, 3 | PROVEN |

## Verdict

**HB-S89-3 is CLOSED.** The embedded NATS integration harness provides concrete evidence that:

1. The publish/consume pipeline works correctly for both execution families (paper + venue).
2. KV projections with monotonicity guards function against real JetStream.
3. The kill switch (control gate) lifecycle is operational.
4. JetStream deduplication prevents duplicate processing.
5. Multi-symbol isolation is preserved through the full pipeline.

## Files

- `internal/adapters/nats/execution_integration_test.go` — 11 integration test scenarios
- `internal/adapters/nats/go.mod` — added `github.com/nats-io/nats-server/v2` dependency
