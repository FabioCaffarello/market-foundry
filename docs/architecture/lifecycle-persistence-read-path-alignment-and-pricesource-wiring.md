# Lifecycle Persistence, Read-Path Alignment, and PriceSource Wiring

> Stage: S387 — OMS Foundation Wave
> Status: Complete
> Scope: Lifecycle persistence, read-path queryability, PriceSource runtime wiring

## Purpose

S387 closes three foundational gaps identified across S383–S386:

1. **Lifecycle persistence gap**: Venue rejection events (S386) had stream and consumer specs but no KV projection — rejections were not queryable via the read model.
2. **Read-path gap**: The composite `execution.query.status.latest` endpoint returned intent + fill + gate but excluded rejection state — lifecycle visibility was incomplete.
3. **PriceSource wiring gap (G1)**: S384 introduced `ports.PriceSource` and proved it works with mocks, but production code defaulted to `"0"` because no NATS KV adapter existed.

## Deliverables

### 1. PriceSource NATS KV Adapter

**File**: `internal/adapters/nats/natsevidence/price_source.go`

`CandleKVPriceSource` implements `ports.PriceSource` by reading the `Close` field from the `CANDLE_LATEST` KV bucket. It delegates to the existing `CandleKVStore.Get()` method.

Semantics (matching the `ports.PriceSource` contract):

| Condition | Returns |
|-----------|---------|
| Candle found with Close | `(close_price, nil)` |
| No candle for key | `("0", nil)` |
| Empty Close field | `("0", nil)` |
| KV store error | `("0", problem)` |
| Nil store | `("0", problem)` |

### 2. Production Wiring in Execute Binary

**File**: `cmd/execute/run.go`

At binary startup, before the DryRunSubmitter/PaperVenueAdapter wiring:

1. Open a `CandleKVStore` connection to the NATS server.
2. Create a `CandleKVPriceSource` backed by the store.
3. Inject it into `PaperVenueAdapter` via `WithPriceSource()`.
4. Inject it into `DryRunSubmitter` via `WithPriceSource()`.

Degradation: If the candle KV store is unavailable (e.g., no NATS server, bucket not yet created), the binary starts normally with price source disabled and logs a warning. Fill prices default to `"0"`.

### 3. Rejection KV Projection

**New files**:
- `internal/adapters/nats/natsexecution/rejection_consumer.go` — durable JetStream consumer for `VenueOrderRejectedEvent`
- `internal/actors/scopes/store/rejection_projection_actor.go` — sole writer for `EXECUTION_VENUE_REJECTION_LATEST` KV bucket

**Modified files**:
- `internal/adapters/nats/natsexecution/registry.go` — added `VenueRejectionLatestBucket` constant
- `internal/actors/scopes/store/messages.go` — added `rejectionReceivedMessage`
- `internal/actors/scopes/store/store_supervisor.go` — added `venue_rejection` pipeline entry

The rejection pipeline is activated by the same `venue_market_order` execution family flag, since rejections are part of the venue outcome family.

### 4. Read-Path Enhancement

**Modified files**:
- `internal/application/executionclient/contracts.go` — `ExecutionStatusReply` gains `Rejection *ExecutionIntent` field; `DeriveEffectivePropagation` signature extended to accept rejection
- `internal/actors/scopes/store/query_responder_actor.go` — `handleExecutionStatusLatest` reads from rejection KV store; best-effort (degraded mode if bucket unavailable)

Propagation priority: `most-recent(result, rejection) > intent > "none"`. When both a fill and a rejection exist for the same partition key, the one with the newer timestamp determines propagation.

## Architecture Invariants Preserved

| Invariant | Evidence |
|-----------|----------|
| Sole writer per KV bucket | `RejectionProjectionActor` is exclusive writer for `EXECUTION_VENUE_REJECTION_LATEST` |
| Monotonicity guard | Reuses `KVStore.Put()` with timestamp comparison |
| Fail-closed execution | DryRunSubmitter behavior unchanged — PriceSource is additive |
| Best-effort price | PriceSource callers never fail on error (contract enforced) |
| Backward compatible | Existing endpoints unchanged; Rejection field is additive (null when absent) |

## What Remains Outside Scope

- **OMS complete**: No order management system, position tracking, or portfolio state.
- **Dashboards**: No observability dashboards or Grafana panels.
- **Reporting**: No aggregate reporting or analytics surfaces.
- **Writer persistence for rejections**: ClickHouse writer consumer specs exist (S386) but writer actor wiring is not in scope — deferred to a future stage.
- **Quantity enforcement in domain**: Quantity invariants are tested but not enforced at domain level (documented in S384).
