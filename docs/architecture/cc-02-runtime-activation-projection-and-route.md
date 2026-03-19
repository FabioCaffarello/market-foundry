# CC-02 Runtime Activation, Projection, and Route

## Activation (Config-Driven)

The `ema_crossover` family follows the same opt-in activation model as all signal families. It is activated by including `"ema_crossover"` in the `pipeline.signal_families` list of the runtime config.

### Derive Binary Config

```jsonc
{
  "pipeline": {
    "families": ["candle"],
    "signal_families": ["rsi", "ema_crossover"],
    "timeframes": [300]
  }
}
```

When `ema_crossover` is present in `signal_families`:
1. `DeriveSupervisor.start()` includes the `ema_crossover` processor in the filtered signal processors list.
2. For each activated source/symbol, `SourceScopeActor` spawns one `EMACrossoverSignalSamplerActor` per timeframe.
3. The shared `SignalPublisherActor` routes `ema_crossover` events to the correct NATS subject via `specForType()`.

When `ema_crossover` is absent, the processor is filtered out at startup — zero overhead.

### Store Binary Config

```jsonc
{
  "pipeline": {
    "families": ["candle"],
    "signal_families": ["rsi", "ema_crossover"],
    "timeframes": [300]
  }
}
```

When `ema_crossover` is present:
1. `StoreSupervisor.start()` spawns the `ema_crossover` pipeline (projection + consumer actors).
2. The `SignalProjectionActor` materializes events into the `SIGNAL_EMA_CROSSOVER_LATEST` KV bucket.
3. The `QueryResponderActor` receives the `SignalRegistry` which includes `EMACrossoverLatest` — queries are automatically served.

### Dependency Validation

The settings validation layer enforces:
- `ema_crossover` requires evidence family `candle` to be enabled.
- Config with `signal_families: ["ema_crossover"]` but without `families: ["candle"]` (when families is explicitly configured) is rejected at validation time.

## NATS Topology

### Subjects

| Purpose | Subject Pattern |
|---------|----------------|
| Event publication | `signal.events.ema_crossover.generated.{source}.{symbol}.{timeframe}` |
| Query (latest) | `signal.query.ema_crossover.latest` |

### Stream

| Name | Reused? | Notes |
|------|---------|-------|
| `SIGNAL_EVENTS` | Yes | Wildcard `signal.events.>` covers all signal families |

### KV Bucket

| Name | Key Format | New? |
|------|------------|------|
| `SIGNAL_EMA_CROSSOVER_LATEST` | `{source}.{symbol}.{timeframe}` | Yes |

### Consumer

| Durable Name | Filter Subject | Stream |
|-------------|----------------|--------|
| `store-signal-ema-crossover` | `signal.events.ema_crossover.generated.>` | `SIGNAL_EVENTS` |

## Projection

The `ema_crossover` pipeline reuses `SignalProjectionActor` — the same actor type used by RSI. The only difference is the injected bucket name (`SIGNAL_EMA_CROSSOVER_LATEST` vs `SIGNAL_RSI_LATEST`).

Projection behavior:
1. Receives `signalReceivedMessage` from the consumer actor.
2. Validates the signal (non-final check, domain validation).
3. Applies monotonicity guard (rejects stale/duplicate timestamps).
4. Writes to KV bucket with key `{source}.{symbol}.{timeframe}`.

## Query Route

The existing `/signal/:type/latest` route is fully type-parameterized:
1. Gateway extracts `:type` from URL path.
2. `SignalGateway` calls `LatestSpecByType("ema_crossover")` to get the NATS query subject.
3. Request is forwarded to store's query responder.
4. Response is the latest `signal.Signal` JSON from the KV bucket.

**No new route files or handlers were needed.** This confirms EX-06 (HTTP route reuse).

### Example Query

```
GET /signal/ema_crossover/latest?source=binancef&symbol=btcusdt&timeframe=300
```

Response:
```json
{
  "type": "ema_crossover",
  "source": "binancef",
  "symbol": "btcusdt",
  "timeframe": 300,
  "value": "bullish",
  "metadata": {
    "fast_period": "9",
    "slow_period": "21",
    "fast_ema": "67432.1500",
    "slow_ema": "67210.8800",
    "spread": "221.2700"
  },
  "final": true,
  "timestamp": "2025-03-19T12:00:00Z"
}
```

## Data Flow (End-to-End)

```
Trade → CandleSampler → candleFinalizedMessage
  → EMACrossoverSignalSamplerActor
    → EMACrossoverSampler.AddClose()
    → publishSignalMessage → SignalPublisherActor → NATS (signal.events.ema_crossover.generated.>)
    → signalGeneratedMessage → SourceScopeActor (decision fan-out, no consumers yet)
  → SIGNAL_EVENTS stream
    → store-signal-ema-crossover consumer
    → SignalConsumerActor → signalReceivedMessage
    → SignalProjectionActor → SIGNAL_EMA_CROSSOVER_LATEST KV
    → QueryResponderActor → GET /signal/ema_crossover/latest
```

## Coexistence

RSI and EMA crossover coexist independently:
- Separate NATS consumers with distinct durable names.
- Separate KV buckets.
- Shared `SIGNAL_EVENTS` stream (wildcard subjects).
- Shared `SignalPublisherActor` (routes by type).
- Shared HTTP route (dispatches by `:type` parameter).
- Each can be independently enabled/disabled via config.
