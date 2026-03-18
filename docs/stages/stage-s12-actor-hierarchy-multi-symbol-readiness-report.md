# Stage S12 — Actor Hierarchy & Multi-Symbol Readiness

**Status:** Complete
**Date:** 2026-03-16
**Scope:** Introduce scoped actor hierarchy for multi-symbol readiness

---

## Objective

Evolve the first-slice runtime from a flat actor structure to a scoped hierarchy
organized by exchange/source, preparing the system for multi-symbol operation
without expanding domain boundaries.

## Problem Statement

The S09–S11 first slice uses a flat supervisor pattern:

```
IngestSupervisor
├── PublisherActor (shared, single)
├── BindingWatcherActor
├── ws-binancef-btcusdt      ← flat under supervisor
└── ws-binancef-ethusdt      ← flat under supervisor

DeriveSupervisor
├── EvidencePublisherActor (shared, single)
├── ConsumerActor
├── QueryResponderActor
├── sampler-binancef-btcusdt-60s  ← flat under supervisor
└── sampler-binancef-ethusdt-60s  ← flat under supervisor
```

This causes:
- **No failure isolation by exchange** — a Binance failure affects actors mixed with future Bybit actors
- **Shared publisher bottleneck** — one NATS connection for all exchanges
- **Supervisor overload** — root supervisor handles routing, lifecycle, and coordination
- **No clear ownership** — symbol lifecycle is managed at the wrong level

## Solution: Exchange/Source Scope Actors

### New Hierarchy — Ingest

```
IngestSupervisor
├── BindingWatcherActor
└── ExchangeScopeActor (source-{source})    ← NEW
    ├── PublisherActor (publisher)           ← moved here
    ├── ws-{symbol}                         ← moved here
    └── ws-{symbol}
```

**ExchangeScopeActor** owns:
- One NATS observation publisher per exchange
- All WebSocket adapter actors for that exchange
- Symbol activation/deactivation lifecycle within the exchange

### New Hierarchy — Derive

```
DeriveSupervisor
├── ConsumerActor
├── QueryResponderActor
└── SourceScopeActor (source-{source})      ← NEW
    ├── EvidencePublisherActor (publisher)   ← moved here
    ├── sampler-{symbol}-{tf}s              ← moved here
    └── sampler-{symbol}-{tf}s
```

**SourceScopeActor** owns:
- One NATS evidence publisher per source
- All sampler actors for that source
- Trade routing by symbol within the source
- Reports sampler PIDs back to supervisor for query index

### Trade Routing

**Before:** Supervisor → flat sampler map lookup → direct send
**After:** Supervisor → source scope by source → symbol sampler by symbol

### Query Routing

**Before:** QueryResponder → supervisor.lookupSampler() → direct PID send
**After:** QueryResponder → supervisor.lookupSampler() → direct PID send (unchanged)

The supervisor maintains a flat sampler index (`map[string]*actor.PID`) for O(1) query
lookups. SourceScopeActor reports sampler PIDs via `samplerActivatedMessage`. This avoids
adding a message hop to every query path while keeping lifecycle ownership in the scope actor.

## Ownership Decisions

| Actor | Owner | Lifecycle |
|-------|-------|-----------|
| BindingWatcherActor | IngestSupervisor | singleton, watches configctl |
| ExchangeScopeActor | IngestSupervisor | one per exchange, created on first binding |
| PublisherActor (obs) | ExchangeScopeActor | one per exchange scope |
| WebSocketAdapterActor | ExchangeScopeActor | one per symbol, dynamic |
| ConsumerActor | DeriveSupervisor | singleton, shared consumer |
| QueryResponderActor | DeriveSupervisor | singleton, uses flat index |
| SourceScopeActor | DeriveSupervisor | one per source, created on first binding |
| EvidencePublisherActor | SourceScopeActor | one per source scope |
| SamplerActor | SourceScopeActor | one per symbol, dynamic |

## Files Changed

### New Files
- `internal/actors/scopes/ingest/exchange_scope_actor.go` — ExchangeScopeActor
- `internal/actors/scopes/derive/source_scope_actor.go` — SourceScopeActor

### Modified Files
- `internal/actors/scopes/ingest/ingest_supervisor.go` — routes to exchange scopes instead of flat adapters
- `internal/actors/scopes/derive/derive_supervisor.go` — routes to source scopes, maintains flat sampler index
- `internal/actors/scopes/derive/messages.go` — added `samplerActivatedMessage`

### Unchanged Files
- `internal/actors/scopes/ingest/websocket_actor.go` — no changes needed
- `internal/actors/scopes/ingest/publisher_actor.go` — no changes needed
- `internal/actors/scopes/ingest/binding_watcher_actor.go` — no changes needed
- `internal/actors/scopes/derive/consumer_actor.go` — no changes needed
- `internal/actors/scopes/derive/sampler_actor.go` — no changes needed
- `internal/actors/scopes/derive/publisher_actor.go` — no changes needed
- `internal/actors/scopes/derive/query_responder_actor.go` — no changes needed
- All domain, adapter, application, and config files — no changes needed

## Design Principles Applied

1. **Source as natural supervision boundary** — all symbols from one exchange share failure domain
2. **Lifecycle flows down, registration flows up** — scope actors own children, report PIDs to supervisor
3. **Hot path through hierarchy, cold path direct** — trades follow ownership chain, queries use flat index
4. **Lazy scope creation** — exchange/source scopes created on first binding activation, not eagerly
5. **Publisher per scope** — each scope gets its own NATS connection for isolation

## Intentional Limitations

1. **No SymbolScopeActor** — adding a symbol-level supervisor was considered and rejected as premature; the current two-level hierarchy (supervisor → source scope → actors) is sufficient
2. **No derive BindingWatcherActor** — derive still queries configctl only at startup; dynamic watcher is deferred to S13
3. **No exchange scope teardown** — if all symbols are cleared, the exchange scope actor remains alive; graceful cleanup is deferred
4. **Single timeframe per sampler** — multi-timeframe spawning (multiple samplers per symbol) is unchanged from S11
5. **No persistent sampler registry** — the flat index is in-memory only; lost on restart (acceptable: configctl re-queried on startup)

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Ownership claro por escopo | Source/exchange owns publisher + children |
| Runtime não depende de estrutura flat | Supervisor delegates to scope actors |
| Preparado para multi-symbol | New source = new scope; new symbol = new child |
| Desenho simples, observável, explicável | Two-level hierarchy, structured logging per scope |
| Robustez operacional melhorada | Failure isolation by exchange, publisher per scope |

## Attention Points for S13

1. **Derive BindingWatcherActor** — derive should react to binding changes dynamically (like ingest already does)
2. **Exchange scope cleanup** — when all symbols are cleared, the scope should be garbage collected
3. **Multi-timeframe** — spawn multiple samplers per symbol (60s, 300s) within SourceScopeActor
4. **Health reporting** — scope actors should report health status to their parent for operational visibility
5. **Sampler index cleanup** — when a sampler is stopped, remove it from the supervisor's flat index
