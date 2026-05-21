# Evidence Read-Model Guidelines

> Checklist and rules for adding a new evidence projection to the store service.

## Prerequisites

Before adding a new evidence type, ensure:
- The domain type exists in `internal/domain/evidence/` with `Validate()`
- The event type exists with `EventName()` and `EventMetadata()`
- The derive sampler produces finalized events with `Final=true`
- The evidence publisher publishes to `evidence.events.{type}.sampled.{src}.{sym}.{tf}`

## Checklist

### NATS Layer

- [ ] Add `EventSpec` to `EvidenceRegistry` for the new event type
- [ ] Add `ControlSpec` to `EvidenceRegistry` for the latest query
- [ ] Add `ConsumerSpec` function (e.g., `StoreFooConsumer()`) with type-specific filter subject
- [ ] Add tests for subject conventions and versioning
- [ ] Verify `EVIDENCE_EVENTS` stream subjects are wide enough (`evidence.events.>`)

### Store Adapter Layer

- [ ] Create `{type}_kv_store.go` with `Put` (monotonicity guard) and `Get`
- [ ] Bucket constant: `{TYPE}_LATEST` with appropriate MaxBytes
- [ ] Create `{type}_consumer.go` (durable consumer adapter) following `evidence_consumer.go` pattern

### Store Actor Layer

- [ ] Create `{type}_projection_actor.go` with:
  - Final=true gate
  - Domain Validate() gate
  - Monotonicity guard via store.Put
  - projectionStats counters
  - Stats logging on Stopped
- [ ] Create `{type}_consumer_actor.go` following `evidence_consumer_actor.go` pattern
- [ ] Add `{type}ReceivedMessage` to `messages.go`
- [ ] Register new control route in `QueryResponderActor.start()`:
  - Open KV store connection
  - Add typed control route
  - Add handler method
  - Close KV store on Stopped
- [ ] Add actor spawns to `StoreSupervisor.start()` with dedicated trackers

### Store Binary

- [ ] Add tracker pair to `ProjectionTrackers` struct
- [ ] Create trackers in `cmd/store/run.go`
- [ ] Register trackers in health server

### Application Layer

- [ ] Add query/reply types to `evidenceclient/contracts.go`
- [ ] Create use case (`get_latest_{type}.go`) with validation
- [ ] Create use case tests
- [ ] Add to `EvidenceGateway` interface in `ports/evidence.go`
- [ ] Add gateway method to `evidence_gateway.go`

### HTTP Layer

- [ ] Add handler method to `EvidenceWebHandler` using `parseEvidenceKeyParams`
- [ ] Add use case interface and struct field to handler
- [ ] Update `NewEvidenceWebHandler` constructor
- [ ] Add route in `routes/evidence.go`
- [ ] Add dependency interface and field in `routes/core.go`
- [ ] Update conditional in `DefaultRoutes`
- [ ] Update all test constructor calls

### Gateway Binary

- [ ] Create use case from gateway in `cmd/gateway/run.go`
- [ ] Add to `routes.Dependencies`

### Smoke Tests

- [ ] Add example queries to `tests/http/evidence.http`

## Invariants (Must Hold for All Evidence Types)

| Invariant | Enforcement |
|-----------|-------------|
| Only `Final=true` events are materialized | Gate 1 in every projection actor |
| Domain validation precedes all KV writes | Gate 2 in every projection actor |
| Latest projection never regresses | Monotonicity guard (OpenTime comparison) in every KV store `Put` |
| Each type has its own durable consumer | Subject-based filtering per ConsumerSpec |
| Each type has its own health tracker pair | Declared in `ProjectionTrackers`, registered in health server |
| Gateway never touches KV directly | All queries through NATS request/reply via store |

## Naming Conventions

| Artifact | Pattern | Example |
|----------|---------|---------|
| Domain type | `Evidence{Type}` | `EvidenceCandle`, `EvidenceTradeBurst` |
| Event | `{Type}SampledEvent` | `CandleSampledEvent`, `TradeBurstSampledEvent` |
| Event name | `{type}.sampled` | `candle.sampled`, `tradeburst.sampled` |
| Publish subject | `evidence.events.{type}.sampled` | `evidence.events.candle.sampled` |
| Consumer name | `store-{type}` | `store-evidence`, `store-trade-burst` |
| KV bucket | `{TYPE}_LATEST` | `CANDLE_LATEST`, `TRADE_BURST_LATEST` |
| Query subject | `evidence.query.{type}.latest` | `evidence.query.candle.latest` |
| HTTP path | `/evidence/{type}/latest` | `/evidence/candles/latest`, `/evidence/tradeburst/latest` |
| Projection actor | `{Type}ProjectionActor` | `CandleProjectionActor` |
| Consumer actor | `{Type}ConsumerActor` | `EvidenceConsumerActor`, `TradeBurstConsumerActor` |
| Health trackers | `{type}-projection`, `{type}-consumer` | `candle-projection`, `trade-burst-consumer` |
