# CC-02 — Family Definition: `ema_crossover` Signal

> Stage S125 — Controlled Capability 02: Signal Family Extensibility Test

## 1. Executive Summary

CC-02 introduces `ema_crossover` (Exponential Moving Average Crossover) as a new signal family, exercising the full code path from domain model through config activation. The purpose is **not** to deliver a product feature — it is to **prove that the monorepo absorbs new families with low friction and high structural discipline**.

## 2. Family Chosen: `ema_crossover`

### 2.1 Why `ema_crossover` Over Alternatives

| Candidate              | Pros                                                | Cons                                          | Verdict    |
|------------------------|-----------------------------------------------------|-----------------------------------------------|------------|
| `moving_average_crossover` (SMA) | Simple math, clear semantics               | SMA requires full window buffer; name too long | Runner-up  |
| **`ema_crossover`**    | Stateless after warm-up (like RSI), minimal buffer, tests two-parameter signal (fast/slow), canonical crossover pattern | Slightly more complex than SMA | **Selected** |
| `macd`                 | Industry-standard, rich metadata                    | Three components (MACD, signal, histogram) — over-complex for extensibility test | Too large  |
| `bollinger_bands`      | Tests multi-value output                            | Three output values complicate signal model fit | Too large  |

### 2.2 Justification

1. **Structural representativity**: `ema_crossover` exercises every layer that RSI exercises — sampler, publisher, projection, query, config — while introducing a genuinely different computation (two EMAs + crossover detection vs. single RSI).

2. **Bounded complexity**: EMA computation is O(1) per candle after warm-up (two running averages), matching RSI's stateful-but-lightweight pattern. No windowed buffers needed.

3. **Two-parameter signal**: Unlike RSI (single `period`), `ema_crossover` requires `fast_period` and `slow_period`, testing that the `Metadata` map pattern handles multi-parameter signals without domain model changes.

4. **Crossover semantics**: The signal Value represents crossover state (`bullish`, `bearish`, `neutral`), testing that the signal domain model's string-based Value field accommodates non-numeric outputs — a property RSI never exercised.

5. **Dependency alignment**: Like RSI, `ema_crossover` depends on `candle` evidence. No new evidence families required.

## 3. Domain Model

The existing `signal.Signal` struct is **reused without modification**:

```go
signal.Signal{
    Type:      "ema_crossover",
    Source:    "binancef",
    Symbol:    "btcusdt",
    Timeframe: 300,
    Value:     "bullish",  // "bullish" | "bearish" | "neutral"
    Metadata: map[string]string{
        "fast_period": "9",
        "slow_period": "21",
        "fast_ema":    "67432.1500",
        "slow_ema":    "67210.8800",
        "spread":      "221.2700",  // fast - slow
    },
    Final:     true,
    Timestamp: ts,
}
```

**Value semantics**:
- `"bullish"` — fast EMA crossed above slow EMA (or remains above)
- `"bearish"` — fast EMA crossed below slow EMA (or remains below)
- `"neutral"` — warm-up incomplete or EMAs equal within tolerance

## 4. Component Map

### 4.1 Application Layer

| File | Purpose |
|------|---------|
| `internal/application/signal/ema_crossover_sampler.go` | Pure computation: two EMAs, crossover detection |
| `internal/application/signal/ema_crossover_sampler_test.go` | Unit tests: warm-up, crossover detection, edge cases |

### 4.2 Actor Layer — Derive

| File | Purpose |
|------|---------|
| `internal/actors/scopes/derive/ema_crossover_sampler_actor.go` | Actor wrapper: receives `candleFinalizedMessage`, delegates to sampler, emits `publishSignalMessage` + `signalGeneratedMessage` |

**Wiring in `derive_supervisor.go`**: Add `SignalFamilyProcessor` entry for `"ema_crossover"`.

### 4.3 Actor Layer — Store

| File | Purpose |
|------|---------|
| *(no new files)* | Reuses existing `SignalProjectionActor` and `SignalConsumerActor` — they are type-agnostic |

**Wiring in `store_supervisor.go`**: Add `Pipeline` entry with family `"ema_crossover"`, dedicated bucket and consumer spec.

### 4.4 Adapter Layer — NATS

| Touchpoint | Change |
|------------|--------|
| `signal_registry.go` | Add `EMACrossoverGenerated EventSpec` and `EMACrossoverLatest ControlSpec` |
| `signal_registry.go` | Add `StoreEMACrossoverSignalConsumer()` function |
| `signal_publisher.go` | Add `case "ema_crossover"` in `specForType()` |
| `signal_kv_store.go` | Add `SignalEMACrossoverLatestBucket` constant |

### 4.5 Gateway / HTTP Layer

| Touchpoint | Change |
|------------|--------|
| `signal_registry.go` `LatestSpecByType()` | Add `case "ema_crossover"` |
| *(no new route files)* | Existing `/signal/:type/latest` route is type-parameterized — works automatically |

### 4.6 Config / Settings

| Touchpoint | Change |
|------------|--------|
| `schema.go` `knownSignalFamilies` | Add `"ema_crossover": true` |
| `schema.go` `signalDependsOnEvidence` | Add `"ema_crossover": {"candle"}` |

### 4.7 Config Artifact (JSONC)

```jsonc
{
  "pipeline": {
    "signal_families": ["rsi", "ema_crossover"],
    // ... existing families unchanged
  }
}
```

## 5. NATS Topology

### 5.1 Subjects

| Subject Pattern | Purpose |
|----------------|---------|
| `signal.events.ema_crossover.generated.{source}.{symbol}.{timeframe}` | Event publication |
| `signal.query.ema_crossover.latest` | Request/reply query |

### 5.2 Stream

Reuses existing `SIGNAL_EVENTS` stream (wildcard `signal.events.>`).

### 5.3 KV Bucket

| Bucket | Key Format | Purpose |
|--------|-----------|---------|
| `SIGNAL_EMA_CROSSOVER_LATEST` | `{source}.{symbol}.{timeframe}` | Materialized latest signal |

### 5.4 Consumer

| Durable Name | Filter Subject | Purpose |
|-------------|---------------|---------|
| `store-signal-ema_crossover` | `signal.events.ema_crossover.generated.>` | Store projection consumer |

## 6. Data Flow

```
┌─────────────┐    candleFinalizedMessage    ┌──────────────────────────────┐
│ CandleSampler│ ──────────────────────────→ │ EMACrossoverSamplerActor      │
│  (existing)  │                              │  ↳ EMACrossoverSampler.AddClose│
└─────────────┘                              └──────────┬───────────────────┘
                                                        │
                                       publishSignalMessage + signalGeneratedMessage
                                                        │
                                       ┌────────────────┴──────────────┐
                                       ▼                               ▼
                              SignalPublisherActor            SourceScopeActor
                              (existing, type-agnostic)       (existing, fan-out)
                                       │
                                       ▼
                              SIGNAL_EVENTS stream
                              (subject: signal.events.ema_crossover.generated.>)
                                       │
                                       ▼
                              SignalConsumerActor (store, new consumer spec)
                                       │
                                       ▼
                              SignalProjectionActor (existing, type-agnostic)
                                       │
                                       ▼
                              SIGNAL_EMA_CROSSOVER_LATEST KV bucket
                                       │
                                       ▼
                              QueryResponderActor → GET /signal/ema_crossover/latest
```

## 7. Estimated Touchpoints

| Layer | New Files | Modified Files |
|-------|-----------|---------------|
| Domain | 0 | 0 |
| Application | 2 | 0 |
| Actors/Derive | 1 | 1 (`derive_supervisor.go`) |
| Actors/Store | 0 | 1 (`store_supervisor.go`) |
| Adapters/NATS | 0 | 3 (`signal_registry.go`, `signal_publisher.go`, `signal_kv_store.go`) |
| HTTP/Routes | 0 | 0 |
| Settings | 0 | 1 (`schema.go`) |
| Config JSONC | 0 | 1 (`configctl.jsonc`) |
| **Total** | **3 new** | **7 modified** |

This is the core metric CC-02 will measure: **3 new files + 7 modified files to add a new signal family end-to-end**.
