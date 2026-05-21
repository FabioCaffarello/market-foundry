# Vertical Slice 01 — Definition

## Purpose

Define the minimal, architecturally representative vertical slice that validates the market-foundry platform end-to-end. This slice is an **architecture proof instrument**, not a product feature.

## Slice Identity

**Name:** `candle-to-paper-order`
**Pipeline:** `observation → candle → rsi → rsi_oversold → mean_reversion_entry → position_exposure → paper_order → venue_market_order`

## Why This Slice

This is the only complete pipeline the codebase already implements. Every domain, every family in the dependency chain, every actor, publisher, consumer, projection, and query surface already exists in code. The vertical slice work is therefore **integration validation**, not new feature development.

Choosing this specific chain maximizes architectural coverage with zero speculative code:

| Concern | How the Slice Exercises It |
|---------|---------------------------|
| Config activation | A config must be created, validated, compiled, and activated to start the pipeline |
| Dynamic binding | `BindingWatcherActor` in both ingest and derive must react to `IngestionRuntimeChangedEvent` |
| Observation capture | Ingest WebSocket adapter must connect, normalize trades, publish to `OBSERVATION_EVENTS` |
| Evidence sampling | Derive must buffer trades, emit finalized candles to `EVIDENCE_EVENTS` |
| Signal computation | RSI sampler must consume candles, emit to `SIGNAL_EVENTS` |
| Decision evaluation | RSI oversold evaluator must consume signals, emit to `DECISION_EVENTS` |
| Strategy resolution | Mean reversion entry resolver must consume decisions, emit to `STRATEGY_EVENTS` |
| Risk assessment | Position exposure evaluator must consume strategies, emit to `RISK_EVENTS` |
| Execution intent | Paper order evaluator must consume risk assessments, emit to `EXECUTION_EVENTS` |
| Venue execution | Execute runtime must consume intents, submit to paper venue, emit fills to `EXECUTION_FILL_EVENTS` |
| Read model materialization | Store must project all 8 event types into NATS KV buckets |
| Query surface | Gateway must serve queries for all 8 domains via HTTP → NATS request/reply |
| Health & diagnostics | All 6 runtimes must expose `/healthz`, `/readyz`, `/statusz`, `/diagz` |
| Cross-runtime routing | Envelope correlation/causation chains must flow end-to-end |

## Participating Runtimes

All 6 runtimes participate:

| Runtime | Role in Slice |
|---------|---------------|
| **configctl** | Stores and activates the slice config; publishes binding events |
| **ingest** | Captures trades from a single source/symbol; publishes observations |
| **derive** | Runs the full sampling/signal/decision/strategy/risk/execution pipeline |
| **store** | Materializes all 8 projection types into KV; serves queries |
| **gateway** | Exposes HTTP query surface for all domains |
| **execute** | Consumes paper order intents; submits to paper venue; publishes fills |

## Participating Families

| Domain | Family | Upstream Dependency |
|--------|--------|---------------------|
| Evidence | `candle` | (observation trades) |
| Signal | `rsi` | `candle` |
| Decision | `rsi_oversold` | `rsi` |
| Strategy | `mean_reversion_entry` | `rsi_oversold` |
| Risk | `position_exposure` | `mean_reversion_entry` |
| Execution | `paper_order` | `position_exposure` |
| Execution | `venue_market_order` | `position_exposure` |

**Excluded families** (explicitly out of scope): `tradeburst`, `volume`. These are evidence families that do not participate in the signal→execution chain. Including them would widen the slice without proving additional architectural concerns.

## Binding Configuration

The slice uses a single binding to minimize variables:

```
source:    binancef
symbol:    btcusdt
timeframe: 60 (1-minute candles)
```

This matches the existing `make seed` smoke configuration and requires only the Binance Futures testnet WebSocket — no API keys needed for observation-only mode.

## Pipeline Configuration

```jsonc
{
  "pipeline": {
    "timeframes": [60],
    "evidence_families": ["candle"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"],
    "risk_families": ["position_exposure"],
    "execution_families": ["paper_order"]
  },
  "venue": {
    "type": "paper_simulator"
  }
}
```

## What This Slice Does NOT Prove

- Multi-symbol behavior (single binding only)
- Multi-source behavior (single exchange only)
- Multi-timeframe behavior (single timeframe only)
- Production venue connectivity (paper simulator only)
- Horizontal scaling (single instance per runtime)
- Schema evolution or versioning
- Persistence beyond NATS KV (no ClickHouse projections)
- Authentication, authorization, or multi-tenant isolation

These exclusions are deliberate. The slice proves the **architectural wiring**, not the operational breadth.

## Architecture Proof Points

The slice is successful when these architecture proof points are demonstrated:

1. **Config-driven activation**: Pipeline starts only after config activation; no hardcoded behavior
2. **Dynamic binding propagation**: ingest and derive react to `IngestionRuntimeChangedEvent` without restart
3. **Event chain integrity**: Each domain event flows through its JetStream stream to the correct downstream consumer
4. **Projection correctness**: All 8 KV buckets contain valid, finalized data for the active binding
5. **Query surface completeness**: All 8 query endpoints return data consistent with projected state
6. **Diagnostic visibility**: `/statusz` on all runtimes shows non-zero event counts and no stale trackers
7. **Error tracking coherence**: `/diagz` on all runtimes shows zero error counts under normal operation
8. **Envelope traceability**: Correlation IDs survive the full observation→fill chain
9. **Graceful lifecycle**: All runtimes start clean and shut down without resource leaks
10. **Health convergence**: `/readyz` on all runtimes returns 200 within 30 seconds of startup
