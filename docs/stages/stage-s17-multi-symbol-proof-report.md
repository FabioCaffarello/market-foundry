# Stage S17 — Multi-Symbol Proof

**Status:** Complete
**Objective:** Prove the runtime supports 2 symbols × 2 timeframes via configuration, without rework.

## Context

S15 proved multi-timeframe (60s + 300s) with zero downstream changes.
S16 added dynamic binding discovery to derive.
S17 validates that the architecture scales to multiple symbols — the first real test of the 2D matrix (symbols × timeframes) in a controlled scenario.

## Scenario

| Dimension | Values |
|-----------|--------|
| Source | `binancef` (single exchange) |
| Symbols | `btcusdt`, `ethusdt` |
| Timeframes | `60s` (1-minute), `300s` (5-minute) |
| KV entries | 4 (`source.symbol.timeframe`) |
| Event types | `trade_received`, `candle_sampled` (unchanged) |

## What Changed

### Code Changes: None

**Zero Go code was modified.** The architecture designed in S12–S16 already supports multi-symbol natively:

| Component | Multi-Symbol Mechanism | Status |
|-----------|----------------------|--------|
| `ExchangeScopeActor` | `adapters map[string]*actor.PID` keyed by symbol | Ready |
| `SourceScopeActor` | `samplers map[string][]*actor.PID` keyed by symbol | Ready |
| `ObservationPublisher` | Subject `...trade.{source}` — symbol in payload | Ready |
| `EvidencePublisher` | Subject `...sampled.{source}.{symbol}.{timeframe}` | Ready |
| `CandleKVStore` | Key `{source}.{symbol}.{timeframe}` | Ready |
| `QueryResponderActor` | Reads KV by key — any symbol works | Ready |
| `EvidenceHandler` (HTTP) | Takes `source`, `symbol`, `timeframe` as query params | Ready |
| `BindingWatcherActor` | Iterates all bindings from configctl, sends per binding | Ready |
| `DeriveObservationConsumer` | Wildcard `observation.events.market.trade.>` | Ready |
| `StoreEvidenceConsumer` | Wildcard `evidence.events.candle.sampled.>` | Ready |

### Configuration Changes

A seed script was added to activate bindings via the configctl lifecycle:

```bash
make seed-multi    # Creates + validates + compiles + activates config with 2 bindings
```

The config document contains:
```json
{
  "bindings": [
    {"name": "btcusdt-trades", "topic": "binancef.btcusdt"},
    {"name": "ethusdt-trades", "topic": "binancef.ethusdt"}
  ]
}
```

### Operational Changes

| File | Change |
|------|--------|
| `scripts/seed-configctl.sh` | New — Seeds configctl via HTTP lifecycle |
| `scripts/smoke-multi-symbol.sh` | New — Validates 2×2 matrix + cross-symbol isolation |
| `tests/http/evidence.http` | Updated — Added ethusdt queries |
| `Makefile` | Updated — Added `seed`, `seed-multi`, `smoke-multi` targets |
| `DEVELOPMENT.md` | Updated — Documented seed and multi-symbol workflow |

## Actor Topology at Runtime (2 symbols × 2 timeframes)

```
IngestSupervisor
├── BindingWatcherActor
└── ExchangeScopeActor [binancef]
    ├── PublisherActor (shared)
    ├── WebSocketAdapterActor [btcusdt]
    └── WebSocketAdapterActor [ethusdt]

DeriveSupervisor
├── ObservationConsumerActor (single durable: derive-observation)
├── BindingWatcherActor
└── SourceScopeActor [binancef]
    ├── EvidencePublisherActor (shared)
    ├── SamplerActor [btcusdt-60s]
    ├── SamplerActor [btcusdt-300s]
    ├── SamplerActor [ethusdt-60s]
    └── SamplerActor [ethusdt-300s]

StoreSupervisor
├── EvidenceConsumerActor (single durable: store-evidence)
├── CandleProjectionActor (writes 4 KV keys)
└── QueryResponderActor (reads by source.symbol.timeframe)
```

**Total actors added by second symbol:** +1 WebSocket adapter, +2 samplers = **3 actors**.
**Total KV keys:** 4 (was 2 with single symbol).

## NATS Subject Space

| Subject | Cardinality |
|---------|-------------|
| `observation.events.market.trade.binancef` | 1 (all symbols share source subject) |
| `evidence.events.candle.sampled.binancef.btcusdt.60` | 1 |
| `evidence.events.candle.sampled.binancef.btcusdt.300` | 1 |
| `evidence.events.candle.sampled.binancef.ethusdt.60` | 1 |
| `evidence.events.candle.sampled.binancef.ethusdt.300` | 1 |

**Observation subject is shared per source** — trades for all symbols are interleaved.
Derive routes trades to the correct symbol's samplers via `event.Trade.Symbol`.

## How to Run

```bash
make up              # Start the full stack
make seed-multi      # Seed configctl with btcusdt + ethusdt
# Wait ~90s for pipeline to produce candles
make smoke-multi     # Validate 2 symbols × 2 timeframes
```

## Validation Points

The `smoke-multi-symbol.sh` script validates:

1. **Gateway health** — `/healthz` and `/readyz` return 200
2. **Pipeline production** — At least one 60s candle appears within timeout
3. **2×2 matrix** — All 4 `source/symbol/timeframe` combinations return 200 with valid structure
4. **Cross-symbol isolation** — btcusdt and ethusdt produce independent candle data (different OHLCV)
5. **Error handling** — Missing/empty params return 400

## Evidence of Scalability

### Architectural Properties Confirmed

1. **Zero code changes** — Multi-symbol support is a pure configuration concern
2. **Linear actor growth** — Adding a symbol adds 1 WS adapter + T samplers (T = timeframes)
3. **Shared infrastructure** — Publisher actors are shared within a source scope (not per symbol)
4. **Idempotent activation** — Duplicate binding messages are safely ignored
5. **Independent lifecycles** — Each symbol's samplers operate independently
6. **Config-canonical** — All symbol activation flows through configctl → binding watcher → supervisor

### Resource Growth Model

| Symbols | Timeframes | WS Adapters | Samplers | KV Keys | Evidence Subjects |
|---------|-----------|-------------|----------|---------|-------------------|
| 1 | 2 | 1 | 2 | 2 | 2 |
| 2 | 2 | 2 | 4 | 4 | 4 |
| 5 | 3 | 5 | 15 | 15 | 15 |
| N | T | N | N×T | N×T | N×T |

Growth is **strictly linear** in N×T with no combinatorial explosion.

## Limits and Observations

### Known Limits

1. **Single observation subject per source** — All trades from a source share `observation.events.market.trade.{source}`. At high symbol counts, this becomes a throughput bottleneck for the derive consumer. Mitigation: add symbol to the observation subject (`...trade.{source}.{symbol}`) and use per-symbol consumers — but this changes the observation contract and is deferred.

2. **Global timeframe list** — All symbols share the same timeframe configuration. Per-symbol timeframe overrides would require config schema changes. Not needed for current scope.

3. **No per-symbol backpressure** — If one symbol produces significantly more trades (e.g., BTCUSDT vs a low-volume pair), the shared observation consumer processes them sequentially. The sampler fan-out is fast (in-memory), so this is acceptable at current scale.

4. **KV stores latest only** — The CANDLE_LATEST bucket stores one entry per key. Historical candles are not persisted. This is a known Store limitation tracked for future stages.

5. **No deactivation** — Binding watcher handles activation but not removal of symbols. Clearing a scope logs the event but doesn't stop actors. This is consistent with S16's documented limitation.

### What Didn't Break

- NATS stream configuration unchanged — `OBSERVATION_EVENTS` and `EVIDENCE_EVENTS` streams handle multi-symbol naturally via wildcard subjects
- Docker Compose unchanged — no new services or configuration needed
- HTTP API unchanged — query params already accept any symbol
- Store consumer unchanged — wildcard `evidence.events.candle.sampled.>` captures all symbols
- Derive observation consumer unchanged — wildcard `observation.events.market.trade.>` captures all sources

## Conclusion

The multi-symbol proof confirms that the first vertical slice architecture scales to N symbols × T timeframes with **zero code changes**. The 2×2 scenario (btcusdt + ethusdt, 60s + 300s) validates:

- Config-driven activation through the full configctl lifecycle
- Correct actor hierarchy with per-symbol isolation
- Independent NATS subjects and KV keys per symbol/timeframe
- Linear resource growth without architectural rework

The architecture is ready for controlled expansion. The next stages can focus on higher-level domain concerns (richer evidence, signals) or operational hardening (deactivation, per-symbol backpressure) without revisiting the core multi-symbol mechanics.

## Recommendations Before S18

1. **Deactivation support** — Implement symbol removal when a binding is cleared (tracked since S16)
2. **Observation subject partitioning** — Evaluate adding symbol to observation subject for throughput isolation (becomes relevant at ~10+ symbols)
3. **Operational dashboard** — Consider exposing actor/sampler counts via a management endpoint for visibility
4. **Integration test** — Add a unit/integration test that validates the 2×2 matrix programmatically (not just via live Binance WS)
5. **Candle history** — Store needs historical candle persistence before the slice is production-ready
