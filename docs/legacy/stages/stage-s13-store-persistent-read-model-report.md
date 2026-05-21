# Stage S13 — Store Binary & Persistent Read Model

**Status:** Complete
**Date:** 2026-03-16
**Scope:** Introduce store binary with persistent candle read model via NATS KV

---

## Objective

Separate the read path from the write path by introducing a `store` binary
that consumes evidence events, materializes a persistent read model, and
serves queries — removing this responsibility from `derive`.

## Problem Statement

In S09–S12, the `derive` binary both produces evidence events AND serves queries:
- SamplerActor handles `snapshotCandleRequest` from QueryResponderActor
- QueryResponderActor subscribes to `evidence.query.candle.latest`
- The query path is tightly coupled to the write path via direct actor messaging

This causes:
- **Write/read coupling** — derive must keep sampler state accessible for queries
- **No persistence** — candle state is lost on derive restart
- **Scaling limitation** — can't scale read and write paths independently
- **Domain leakage** — derive owns both derivation logic and read model serving

## Solution: Store Binary with NATS KV

### Architecture

```
ingest → OBSERVATION_EVENTS → derive → EVIDENCE_EVENTS → store → NATS KV
                                                                    ↓
                                                     gateway ← evidence.query.candle.latest
```

### Store Actor Hierarchy

```
StoreSupervisor
├── CandleProjectionActor — materializes finalized candles to NATS KV
├── EvidenceConsumerActor — durable consumer for EVIDENCE_EVENTS
└── QueryResponderActor — serves evidence.query.candle.latest from KV
```

### Persistence: NATS KV (KeyValue Store)

**Why NATS KV:**
- Zero new infrastructure — NATS is already deployed
- Built into JetStream with FileStorage — survives restarts
- Key-value semantics are perfect for "latest candle per key"
- Consistent with existing NATS-centric architecture
- Debuggable via `nats kv` CLI tools
- JSON values for human readability

**Configuration:**
- Bucket: `CANDLE_LATEST`
- Storage: FileStorage (persisted to disk)
- MaxBytes: 64 MB (sufficient for thousands of symbol/timeframe combinations)
- Key format: `{source}.{symbol}.{timeframe}` (e.g., `binancef.btcusdt.60`)
- Value: JSON-encoded `EvidenceCandle`

### Derive Cleanup

The following were removed from derive:
- `query_responder_actor.go` — deleted (query serving moved to store)
- `SamplerLookup` type and flat sampler index — removed from supervisor
- `samplerActivatedMessage` — removed (was used only for query index)
- `snapshotCandleRequest/Reply` — removed from messages
- `SupervisorPID` field in SourceScopeConfig — removed

Derive is now **write-only**: it consumes observations, samples candles, and
publishes evidence events. It no longer serves any queries.

## Files Changed

### New Files — Store

| File | Purpose |
|------|---------|
| `cmd/store/main.go` | Entry point |
| `cmd/store/run.go` | Runtime setup |
| `cmd/store/go.mod` | Module definition |
| `internal/actors/scopes/store/store_supervisor.go` | Root actor |
| `internal/actors/scopes/store/evidence_consumer_actor.go` | Durable evidence consumer |
| `internal/actors/scopes/store/candle_projection_actor.go` | NATS KV materializer |
| `internal/actors/scopes/store/query_responder_actor.go` | Evidence query server |
| `internal/actors/scopes/store/messages.go` | Internal messages |
| `deploy/configs/store.jsonc` | Service configuration |

### New Files — Adapter Layer

| File | Purpose |
|------|---------|
| `internal/adapters/nats/evidence_consumer.go` | Durable JetStream consumer for evidence events |
| `internal/adapters/nats/candle_kv_store.go` | NATS KV adapter for candle read model |

### Modified Files

| File | Change |
|------|--------|
| `internal/adapters/nats/evidence_registry.go` | Added `StoreEvidenceConsumer()` spec |
| `internal/actors/scopes/derive/derive_supervisor.go` | Removed query responder, sampler index |
| `internal/actors/scopes/derive/source_scope_actor.go` | Removed SupervisorPID, samplerActivatedMessage |
| `internal/actors/scopes/derive/sampler_actor.go` | Removed snapshot handling |
| `internal/actors/scopes/derive/messages.go` | Removed snapshot and samplerActivated types |
| `deploy/compose/docker-compose.yaml` | Added store service |
| `go.work` | Added `./cmd/store` module |
| `docs/architecture/actor-ownership.md` | Updated store ownership section |

### Deleted Files

| File | Reason |
|------|--------|
| `internal/actors/scopes/derive/query_responder_actor.go` | Moved to store |

## Write/Read Separation

### Before S13

```
derive: consume observations → sample candles → publish evidence → serve queries
        (write path)                              (write path)     (read path)
```

### After S13

```
derive: consume observations → sample candles → publish evidence
        (write path only)

store:  consume evidence → materialize in KV → serve queries
        (read path only)
```

**Key difference:** The query response now comes from a persistent KV store,
not from ephemeral actor state. This means:
- Candle data survives derive restarts
- Read and write paths can be scaled independently
- Gateway doesn't know or care which binary serves queries (same NATS subject)

## Data Flow

```
1. derive publishes CandleSampledEvent to EVIDENCE_EVENTS stream
   Subject: evidence.events.candle.sampled.{source}.{symbol}.{timeframe}

2. store's EvidenceConsumerActor (durable: store-evidence) receives the event

3. CandleProjectionActor filters for Final=true candles only

4. CandleProjectionActor writes to NATS KV:
   Bucket: CANDLE_LATEST
   Key: {source}.{symbol}.{timeframe}
   Value: JSON(EvidenceCandle)

5. Gateway sends evidence.query.candle.latest request via NATS

6. store's QueryResponderActor reads from NATS KV and responds
```

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Store consome corretamente o evento derivado | Durable consumer on EVIDENCE_EVENTS |
| Read model materializado de forma persistente | NATS KV with FileStorage |
| Separação derive/leitura mais clara | Derive write-only, store read-only |
| Solução simples, robusta e evolutiva | 3 actors, 1 KV bucket, zero new infra |
| Evita overengineering | Minimal actors, no generic projection framework |

## Intentional Limitations

1. **Only finalized candles materialized** — interim/realtime candle snapshots not stored; store returns the last complete window only
2. **Single read model** — only `CANDLE_LATEST` KV bucket; no time-series history
3. **No PROJECTION_EVENTS stream** — store doesn't emit projection change notifications yet
4. **No config-driven projection activation** — store always projects candles; no configctl integration
5. **No store health endpoint** — uses process-alive healthcheck (same as ingest/derive)
6. **JSON for KV values** — chose readability over CBOR compactness; acceptable at current scale

## Technical Debt

1. **Evidence consumer spec lacks replay capability** — if store starts after derive has been running, it replays from the durable consumer's last ACK position, not from the beginning. First deployment should ensure store starts before or alongside derive.
2. **No candle eviction from KV** — old symbol/timeframe entries remain even if bindings are deactivated. NATS KV has no TTL per key; would need periodic cleanup.
3. **No graceful consumer draining** — on shutdown, the consumer stops but doesn't drain pending messages.

## Attention Points for S14

1. **Store health endpoint** — add HTTP health and readiness checks
2. **Candle history** — consider storing a time series (not just latest) for chart data
3. **Multiple read models** — projection framework if more models are needed
4. **Gateway server dependency update** — server's `newEvidenceGateway()` should depend on store, not derive
5. **Config-driven store** — activate projections based on configctl bindings
6. **PROJECTION_EVENTS** — emit notifications when projections update (for cache invalidation)
