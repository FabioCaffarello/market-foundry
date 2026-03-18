# Signal First Slice

> Tracks which preconditions (P-1 through P-10) from S35 are resolved and what remains.
> Updated at end of S36.

## Precondition Status

| # | Precondition | Status | Resolved In |
|---|-------------|--------|-------------|
| P-1 | `pipeline.signal_families` key in settings schema | **Done** | S36 — `settings.go` schema updated |
| P-2 | `SIGNAL_EVENTS` JetStream stream created on startup | **Done** | S36 — `SignalPublisher.Start()` ensures stream |
| P-3 | `IsFamilyEnabled` extended for signal families | **Done** | S36 — derive supervisor checks `signal_families` config |
| P-4 | SignalFamilyProcessor registration in DeriveSupervisor | **Done** | S36 — `SignalSamplerActor` spawned per binding |
| P-5 | Signal projection pipelines in StoreSupervisor | **Done** | S36 — `SignalConsumerActor` + `SignalProjectionActor` |
| P-6 | Signal KV buckets created on store startup | **Done** | S36 — `SignalKVStore.Start()` ensures `SIGNAL_RSI_LATEST` |
| P-7 | Signal query subjects in QueryResponderActor | **Done** | S36 — `QueryResponderActor` handles `signal.query.rsi.latest` |
| P-8 | Signal HTTP routes in gateway | **Done** | S36 — `GET /signal/:type/latest` wired end-to-end |
| P-9 | BindingWatcherActor fully wired in derive | **Active** | S34 |
| P-10 | Evidence families operational (candle at minimum) | **Active** | S06 |

## Implemented Signal Types

| Type | Domain | Sampler | Publisher | Consumer | Projection | KV Bucket | Gateway Route |
|------|--------|---------|-----------|----------|------------|-----------|---------------|
| RSI | `internal/domain/signal/` | `RSISampler` | `SignalPublisherActor` | `SignalConsumerActor` | `SignalProjectionActor` | `SIGNAL_RSI_LATEST` | `GET /signal/rsi/latest` |

## Query Chain (RSI end-to-end)

```
HTTP GET /signal/rsi/latest?source=X&symbol=Y&timeframe=Z
  → gateway: SignalWebHandler.GetLatestSignal
    → NATS request: signal.query.rsi.latest
      → store: QueryResponderActor
        → SignalKVStore.Get(source, symbol, timeframe)
          → SIGNAL_RSI_LATEST KV bucket
      ← SignalLatestReply { signal: Signal | null }
    ← HTTP 200 { "signal": {...} | null }
```

## What Is NOT in This Slice

- MACD sampler (sampler not implemented — S37)
- Signal history projections
- Multi-evidence signals
- Signal-to-signal composition
- WebSocket/streaming
- Raccoon-CLI drift rules for signal contracts

## Deferred to S37

- MACD sampler implementation and full pipeline
- Signal history projections (if needed)
- Per-type domain structs (if Metadata proves insufficient)
- Raccoon-CLI signal contract drift rules
