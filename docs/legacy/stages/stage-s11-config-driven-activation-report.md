# Stage S11 — Config-Driven Activation

**Status:** Complete
**Date:** 2026-03-16

## Executive Summary

S11 replaces the hardcoded source/symbol/timeframe in both `ingest` and `derive` with config-driven activation. Both services now query configctl on startup for active ingestion bindings and spawn their actors dynamically based on what's configured. The ingest binary also subscribes to `IngestionRuntimeChangedEvent` for real-time binding activation.

No hardcoded `"binancef"`, `"btcusdt"`, or `60 * time.Second` remains in any supervisor. Source and symbol come from configctl bindings; timeframe comes from `deploy/configs/derive.jsonc`.

## Activation Flow

### Startup Path

```
                    ┌─────────────┐
                    │  configctl  │
                    │ (bindings)  │
                    └──────┬──────┘
                           │ ListActiveIngestionBindings
                    ┌──────┴──────┐
                    │             │
              ┌─────▼─────┐ ┌────▼─────┐
              │  ingest   │ │  derive  │
              │           │ │          │
              │ Binding   │ │ Query    │
              │ Watcher   │ │ bindings │
              │  Actor    │ │ on start │
              └─────┬─────┘ └────┬─────┘
                    │            │
              for each binding   for each binding
                    │            │
              spawn WS adapter   spawn sampler
```

### Dynamic Update Path (ingest only)

```
configctl activates config
    → publishes IngestionRuntimeChangedEvent to CONFIGCTL_EVENTS
        → ingest BindingWatcherActor (durable consumer: ingest-binding-watcher)
            → parses binding topics
            → sends activateBindingMessage to supervisor
                → supervisor spawns new WebSocket adapter
```

### Binding Topic Convention

A configctl binding's `Topic` field encodes the market data source and symbol:

```
Topic = "{source}.{symbol}"
```

Examples:
- `"binancef.btcusdt"` → source=binancef, symbol=btcusdt
- `"binancef.ethusdt"` → source=binancef, symbol=ethusdt

This is parsed by `ParseBindingTopic()` in `internal/application/ingest/binding.go`.

### Timeframe Configuration

The candle timeframe is a derive processing parameter, not a binding property. It comes from the config file:

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "default_timeframe_seconds": 60
  }
}
```

Added `PipelineConfig` to the shared `AppConfig` schema with `DefaultTimeframeDuration()` helper.

## Operational Workflow

To activate the first slice via configctl:

```bash
# 1. Start the stack
make up

# 2. Create a config with a binding for btcusdt
curl -X POST http://127.0.0.1:8080/configctl/configs \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "first-slice",
    "format": "json",
    "content": "{\"metadata\":{\"name\":\"First Slice\",\"description\":\"btcusdt aggTrade\"},\"bindings\":[{\"name\":\"btcusdt-trades\",\"topic\":\"binancef.btcusdt\"}],\"fields\":[{\"name\":\"price\",\"type\":\"string\",\"required\":true}],\"rules\":[{\"name\":\"price_required\",\"field\":\"price\",\"operator\":\"required\",\"severity\":\"error\"}]}"'

# 3. Validate, compile, activate (using the version ID from step 2)
curl -X POST http://127.0.0.1:8080/configctl/config-versions/{id}/validate
curl -X POST http://127.0.0.1:8080/configctl/config-versions/{id}/compile -d '{}'
curl -X POST http://127.0.0.1:8080/configctl/config-versions/{id}/activate \
  -H 'Content-Type: application/json' \
  -d '{"scope_kind":"global","scope_key":"default"}'

# 4. Ingest and derive react: WebSocket adapter + sampler spawn dynamically
# 5. Query the evidence endpoint
curl 'http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60'
```

## Files Changed/Created

### New Files

| File | Layer | Purpose |
|------|-------|---------|
| `internal/application/ingest/binding.go` | Application | Binding topic parser: `ParseBindingTopic()` |
| `internal/application/ingest/binding_test.go` | Application | Unit tests for topic parsing |
| `internal/adapters/nats/binding_event_consumer.go` | Adapter | JetStream durable consumer for `IngestionRuntimeChangedEvent` |
| `internal/actors/scopes/ingest/binding_watcher_actor.go` | Actor | Queries configctl + subscribes to binding events |

### Modified Files

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added `PipelineConfig` to `AppConfig` |
| `internal/adapters/nats/configctl_registry.go` | Added `IngestBindingConsumer()` spec |
| `internal/actors/scopes/ingest/ingest_supervisor.go` | Rewritten: dynamic adapter map, accepts gateway, responds to activate/clear messages |
| `internal/actors/scopes/ingest/messages.go` | Added `activateBindingMessage` / `clearBindingMessage` |
| `internal/actors/scopes/derive/derive_supervisor.go` | Rewritten: queries configctl, dynamic sampler map, routes trades to correct sampler |
| `internal/actors/scopes/derive/query_responder_actor.go` | Changed from fixed PID to `SamplerLookup` function |
| `internal/actors/scopes/derive/messages.go` | Added `activateSamplerMessage` |
| `cmd/ingest/run.go` | Wires configctl gateway, passes to supervisor |
| `cmd/derive/run.go` | Wires configctl gateway, passes to supervisor, logs timeframe |
| `deploy/configs/derive.jsonc` | Added `pipeline.default_timeframe_seconds` |

## Architectural Rationale

### Why binding topics encode source.symbol

The configctl `Binding` type has `Name` and `Topic` — a general-purpose pair. Rather than adding market-specific fields to the domain model (which would couple configctl to trading), the topic string encodes the routing information using a convention. This is:
- Zero domain changes to configctl
- Parseable at the ingest/derive boundary
- Extensible to other source types later

### Why the BindingWatcher is in ingest only

Derive doesn't need real-time binding watching because:
- It consumes from a wildcard JetStream subject (`observation.events.market.trade.>`)
- When ingest starts publishing for a new symbol, derive's consumer automatically receives those trades
- Derive only needs to know which samplers to create, which it queries on startup
- A derive restart picks up any binding changes

For full dynamic derive activation, S12 can add an equivalent watcher.

### Why the query responder uses a lookup function

The `SamplerLookup` function replaces the hardcoded `SamplerPID` from S08. This allows the supervisor to manage multiple samplers dynamically and the responder to find the right one for any query. The function captures the supervisor's sampler map via closure.

### Why trades route through the supervisor

In the derive rewrite, the observation consumer sends trades to the supervisor (not directly to a sampler). The supervisor routes trades by matching `source.symbol` to the correct sampler PID. This is necessary because:
- Multiple samplers may exist for different source/symbol pairs
- The consumer doesn't know which sampler handles which trade
- The supervisor owns the sampler registry

## Remaining Risks

1. **Graceful degradation when configctl is slow** — If configctl is not ready when ingest/derive start, the initial binding query fails and no adapters/samplers spawn. The services start empty. Only ingest has the event subscription to recover when bindings activate later.

2. **No scope-based clearing in ingest** — When an `IngestionRuntimeChangeCleared` event arrives, the watcher logs it but doesn't know which specific bindings to clear (the event carries the scope, not the binding topics). Full reconciliation requires tracking scope → binding mappings.

3. **Single timeframe per binding** — Each binding gets one sampler at the configured timeframe. Multi-timeframe requires spawning multiple samplers per binding.

4. **Derive has no dynamic watcher** — Derive queries bindings only on startup. A new binding activated after derive starts won't create a new sampler until derive restarts.

5. **Trade routing overhead** — Every trade passes through the supervisor's mailbox for routing. Under high volume, this could become a bottleneck. Direct consumer → sampler routing (with a shared registry) would reduce this hop.

## Points to Recalibrate Before S12

1. **Operational smoke test update** — The `scripts/smoke-first-slice.sh` script assumes the pipeline works immediately on boot. With config-driven activation, the operator must first create and activate a config via configctl. The smoke script should be updated to include the configctl lifecycle steps.

2. **Derive dynamic watcher** — If derive must react to new bindings without restart, add a BindingWatcherActor analogous to ingest's.

3. **Multi-timeframe support** — The current design spawns one sampler per binding. Adding a second timeframe (e.g., 300s) requires either a list of timeframes in config or a `PipelineConfig.Timeframes` array.

4. **Compose dependency ordering** — With config-driven activation, ingest and derive need configctl to be healthy AND to have active bindings. The docker-compose `depends_on: configctl: service_healthy` only ensures the process is alive, not that bindings are activated.

5. **Config seeding** — For development, a seed script or init container that creates+activates a default config would restore the "make up and it works" experience lost by removing hardcodes.
