# Execution KV Materialization — End-to-End Proof

**Stage**: S271
**Status**: Proven
**Date**: 2026-03-21

## Purpose

Prove the minimum viable materialization path for execution paper intents into NATS KV, closing the debt recorded in S269 regarding unproven KV materialization for execution.

## Validated Path

```
PaperOrderEvaluatorActor (derive)
  ↓ publishExecutionMessage
ExecutionPublisherActor (derive)
  ↓ NATS JetStream: EXECUTION_EVENTS stream
  ↓ subject: execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}
Consumer: store-execution-paper-order (durable)
  ↓ executionReceivedMessage
ExecutionProjectionActor (store)
  ↓ Gate 1: Final guard (intent.Final must be true)
  ↓ Gate 2: Domain validation (ExecutionIntent.Validate())
  ↓ Gate 3: Monotonicity guard (KV adapter compares Timestamp)
KVStore.Put() → EXECUTION_PAPER_ORDER_LATEST bucket
  ↓
QueryResponderActor reads via KVStore.Get()
  ↓ subject: execution.query.paper_order.latest
Gateway.GetLatestExecution() → ExecutionLatestReply
```

## What Was Proven

### KV-RT-1: Full field round-trip
All ExecutionIntent fields survive the JSON serialization round-trip through NATS KV. Verified fields:
- Identity: Type, Source, Symbol, Timeframe
- Execution: Side, Quantity, FilledQuantity, Status
- Risk context: Type, Disposition, Confidence, Timeframe, StrategyType, DecisionSeverity
- Fill records: Price, Quantity, Fee, Simulated, Timestamp
- Maps: Parameters, Metadata
- Trace: CorrelationID, CausationID
- Lifecycle: Final, Timestamp

### KV-RT-2: Monotonicity guard — stale rejection
When a newer intent already exists in KV, an older intent (earlier timestamp) is rejected with `PutSkippedStale`. The stored value remains unchanged.

### KV-RT-3: Monotonicity guard — deduplication
When an intent with the same timestamp as the existing entry is submitted, it is rejected with `PutSkippedDuplicate`. No double-writes occur.

### KV-RT-4: Multi-symbol partition isolation
Intents for different symbols (btcusdt, ethusdt, solusdt) are stored under independent partition keys (`{source}.{symbol}.{timeframe}`). No cross-symbol data bleed occurs.

### KV-RT-5: Missing key semantics
`Get()` returns `(nil, nil)` for keys that have never been written. This is not an error — it's the expected initial state.

### KV-RT-6: Timestamp-advancing overwrites
A newer intent correctly overwrites an older one for the same partition key. The stored value advances monotonically.

### KV-RT-7: No-action intent round-trip
Intents with `Side=none`, `Quantity="0"`, and empty fills survive the round-trip and pass post-read validation.

### KV-RT-8: JSON serialization fidelity
Direct `json.Marshal`/`json.Unmarshal` produces identical results to what the KV store uses. PartitionKey and DeduplicationKey are stable across the cycle.

## Projection Actor Gate Pipeline (Unit-Proven)

The `ExecutionProjectionActor` enforces a three-gate pipeline before any KV write:

| Gate | Check | On Failure |
|------|-------|-----------|
| 1 — Final | `intent.Final == true` | `skippedNonFinal++`, no KV write |
| 2 — Validation | `intent.Validate() == nil` | `rejected++`, no KV write |
| 3 — Monotonicity | `KVStore.Put()` result | `skippedStale++` or `skippedDedup++` |

Stats invariant: `received == materialized + skippedStale + skippedDedup + skippedNonFinal + rejected + errors`

This invariant is checked at actor shutdown and any violation is logged as an error.

## Read Surface

The query responder at `internal/actors/scopes/store/query_responder_actor.go` opens a read-only `KVStore` connection to the same `EXECUTION_PAPER_ORDER_LATEST` bucket and serves queries via NATS request/reply on subject `execution.query.paper_order.latest`.

The `Gateway` at `internal/adapters/nats/natsexecution/gateway.go` wraps this with the `ports.ExecutionGateway` interface, providing `GetLatestExecution()` and the composite `GetExecutionStatus()` which reads both paper order and venue market order KV stores.

## Ownership

| Component | Owner | Sole Writer? |
|-----------|-------|-------------|
| `EXECUTION_PAPER_ORDER_LATEST` bucket | ExecutionProjectionActor (store) | Yes |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` bucket | FillProjectionActor (store) | Yes |
| `EXECUTION_CONTROL` bucket | QueryResponderActor (store, via control set) | Yes |

## Bucket Configuration

| Property | Value |
|----------|-------|
| Bucket name | `EXECUTION_PAPER_ORDER_LATEST` |
| Storage | FileStorage |
| Max bytes | 64 MB |
| History | 1 (latest only) |
| Key format | `{source}.{symbol}.{timeframe}` |

## Test Evidence

| Test file | Coverage |
|-----------|----------|
| `natsexecution/kv_store_roundtrip_test.go` | KV-RT-1 through KV-RT-8 (requires live NATS) |
| `store/execution_projection_actor_test.go` | Gate pipeline, stats invariant, multi-symbol, trace persistence |
| `derive/paper_order_end_to_end_test.go` | Full actor chain from signal through execution intent |
| `execution/safety_gate_integration_test.go` | Safety gate path (S270) |
