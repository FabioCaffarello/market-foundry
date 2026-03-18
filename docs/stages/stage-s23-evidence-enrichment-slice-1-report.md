# S23: Evidence Enrichment Slice 1 — Trade Burst

## Summary

Introduced **Trade Burst** as the first evidence type beyond candles. It proves the Foundry can derive, publish, project, and serve multiple evidence types through the same structural backbone without architectural distortion.

Trade Burst captures per-window trade activity that candles don't: trade count, buy/sell volume split, and a simple burst detection flag. It runs end-to-end through the identical pipeline: observation → derive → evidence stream → store → gateway → HTTP.

## Evidence Chosen: Trade Burst

### Why Trade Burst

| Criterion | Assessment |
|-----------|-----------|
| Simplicity | Counts trades, sums buy/sell volume. Simpler than candle OHLCV. |
| Utility | Directional volume pressure (buy vs sell) and activity anomaly detection. Candles lack both. |
| Compatibility | Same windowing, same finalization, same key structure. Zero new mechanics. |
| Distinctness | Clearly separate from candles: activity metrics vs price metrics. No overlap. |
| Non-signal | Pure evidence — factual observation summary. No prediction, no decision. |

### Why Not Volume Burst

Volume burst (total volume per window) overlaps with candle `Volume` field. Trade burst adds genuinely new data: directional split and burst detection.

## Architecture

### Full Pipeline

```
Binance WS trade
  → ingest → OBSERVATION_EVENTS
  → derive: TradeBurstSamplerActor (per source/symbol/timeframe)
    → TradeBurstSampler (pure logic: count, buy/sell vol, burst flag)
    → EvidencePublisher.PublishTradeBurst
  → EVIDENCE_EVENTS (evidence.events.tradeburst.sampled.{src}.{sym}.{tf})
  → store: TradeBurstConsumerActor (durable: store-trade-burst)
    → TradeBurstProjectionActor → TRADE_BURST_LATEST KV
  → QueryResponderActor (evidence.query.tradeburst.latest)
  → gateway → GET /evidence/tradeburst/latest
```

### Key Design Decision: Structural Reuse, Not Abstraction

Every infrastructure component (consumer, publisher, KV store, projection actor, query responder, HTTP handler) was duplicated for the new type — not generalized. This is intentional:

- Each evidence type has its own domain validation, projection invariants, and query contracts
- A generic framework would hide type-specific behavior and complicate debugging
- The duplication is small (each file follows an established pattern) and explicit
- When a third evidence type arrives, the pattern is proven — abstraction can be considered then

## Files Created

| File | Purpose |
|------|---------|
| `internal/domain/evidence/trade_burst.go` | Domain type + Validate() |
| `internal/domain/evidence/events.go` | TradeBurstSampledEvent (extended) |
| `internal/application/derive/trade_burst_sampler.go` | Pure sampler logic |
| `internal/application/derive/trade_burst_sampler_test.go` | 5 test cases |
| `internal/application/evidenceclient/get_latest_trade_burst.go` | Use case |
| `internal/application/evidenceclient/get_latest_trade_burst_test.go` | Use case tests |
| `internal/adapters/nats/trade_burst_kv_store.go` | KV store with monotonicity guard |
| `internal/adapters/nats/trade_burst_consumer.go` | Durable JetStream consumer |
| `internal/actors/scopes/derive/trade_burst_sampler_actor.go` | Actor wrapping sampler |
| `internal/actors/scopes/store/trade_burst_projection_actor.go` | Projection with validation gates |
| `internal/actors/scopes/store/trade_burst_consumer_actor.go` | Store-side event consumer |
| `docs/architecture/evidence-derivation-pattern.md` | Canonical derivation pattern |
| `docs/architecture/evidence-type-01-contracts.md` | Trade burst contracts |

## Files Modified

| File | Change |
|------|--------|
| `internal/adapters/nats/evidence_registry.go` | Widened stream subjects to `evidence.events.>`, added TradeBurst specs |
| `internal/adapters/nats/evidence_registry_test.go` | Trade burst subject convention tests |
| `internal/adapters/nats/evidence_publisher.go` | Added PublishTradeBurst method |
| `internal/adapters/nats/evidence_gateway.go` | Added GetLatestTradeBurst method |
| `internal/application/evidenceclient/contracts.go` | TradeBurstLatestQuery/Reply types |
| `internal/application/ports/evidence.go` | Added to EvidenceGateway interface |
| `internal/actors/scopes/derive/messages.go` | publishTradeBurstMessage |
| `internal/actors/scopes/derive/source_scope_actor.go` | Spawn trade burst samplers alongside candle samplers |
| `internal/actors/scopes/derive/publisher_actor.go` | Handle trade burst publish messages |
| `internal/actors/scopes/store/messages.go` | tradeBurstReceivedMessage |
| `internal/actors/scopes/store/query_responder_actor.go` | Trade burst query route |
| `internal/actors/scopes/store/store_supervisor.go` | Spawn trade burst actors |
| `internal/interfaces/http/handlers/evidence.go` | GetLatestTradeBurst handler |
| `internal/interfaces/http/handlers/evidence_test.go` | Updated constructor calls |
| `internal/interfaces/http/routes/evidence.go` | Trade burst route |
| `internal/interfaces/http/routes/core.go` | GetLatestTradeBurst dependency |
| `internal/interfaces/http/routes/evidence_test.go` | Trade burst route tests |
| `cmd/gateway/run.go` | Wire GetLatestTradeBurstUseCase |
| `tests/http/evidence.http` | Trade burst smoke examples |

## Intentional Limitations

1. **Latest only** — no `TRADE_BURST_HISTORY` bucket. History can follow the candle history pattern (S19) when needed.
2. **Hardcoded burst threshold** — 2.0× previous window. Sufficient for proving the pattern. Configurable threshold deferred.
3. **Single-window baseline** — burst detection uses only the previous window's count, not a rolling average.
4. **No burst-specific query** — no "show me only burst windows" endpoint. Clients filter on the `burst` field.
5. **Same timeframes as candles** — trade burst samplers use the same configured timeframes. Independent timeframe selection deferred.

## Verification

1. `go build ./...` — all packages compile
2. `go test ./...` — all tests pass
3. `GET /evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60` — returns trade burst or null
4. `GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60` — still works unchanged

## S24 Preparation

1. **Trade burst history** — add `TRADE_BURST_HISTORY` bucket following the candle history pattern. Enables "last N bursts" queries.
2. **Configurable burst threshold** — make the 2.0× ratio configurable per binding via configctl.
3. **Rolling baseline** — use an exponential moving average of recent window counts instead of just the previous window.
4. **Cross-evidence correlation** — future signal layer could combine candle price movement with burst detection. S23 deliberately does NOT do this (evidence ≠ signal).
5. **Third evidence type** — once two types prove the pattern, evaluate whether an abstraction (e.g., generic projection framework) reduces boilerplate without hiding behavior.
