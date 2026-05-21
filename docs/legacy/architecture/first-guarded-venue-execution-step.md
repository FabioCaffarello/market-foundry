# First Guarded Venue Execution Step

**Stage**: S80
**Status**: Implemented
**Scope**: Minimum viable execute binary with PaperVenueAdapter

## Decision

S80 implements the **smallest acceptable step** toward venue-integrated execution:
a standalone `execute` binary that consumes execution intents, passes them through a
`PaperVenueAdapter`, and publishes fill events — all without contacting any real exchange.

This step was authorized because:
- S74-S79 resolved all 5 hard blockers (lifecycle, fills, failure recovery, trace, kill switch)
- Activation gates 1-10 are fully passed
- The two remaining BLOCKING gaps (execute binary, VenuePort) are exactly what S80 delivers

## Architecture

```
derive → EXECUTION_EVENTS → execute → EXECUTION_FILL_EVENTS → store (future S81+)
                               │
                          PaperVenueAdapter
                          (simulated fills)
```

### Binary: `execute`

| Layer | Component | Responsibility |
|-------|-----------|---------------|
| cmd/execute | main.go, run.go | Bootstrap, config, actor engine, health server |
| actors/scopes/execute | ExecuteSupervisor | Root actor: spawns venue adapter, manages consumer |
| actors/scopes/execute | VenueAdapterActor | Kill switch → staleness → VenuePort → publish fill |
| application/ports | VenuePort | Interface for venue order submission |
| application/execution | PaperVenueAdapter | Simulated venue: instant fills, paper-prefixed IDs |
| application/execution | StalenessGuard | Rejects intents older than configurable max age |

### Guards Active at Every Intent

1. **Kill switch**: VenueAdapterActor reads `EXECUTION_CONTROL` KV before each intent
2. **Staleness**: Intents older than 120s (default) are silently dropped with stats
3. **Domain validation**: Intent must pass `Validate()` before venue submission
4. **Trace propagation**: `correlation_id` and `causation_id` flow from intent to fill event

### Contracts

| Stream | Subject Pattern | Producer | Consumer |
|--------|----------------|----------|----------|
| EXECUTION_EVENTS | execution.events.paper_order.submitted.{source}.{symbol}.{timeframe} | derive | execute |
| EXECUTION_FILL_EVENTS | execution.fill.venue_market_order.{source}.{symbol}.{timeframe} | execute | store (S81+) |

| KV Bucket | Key Pattern | Authority |
|-----------|-------------|-----------|
| EXECUTION_CONTROL | global | store |
| EXECUTION_VENUE_MARKET_ORDER_LATEST | {source}.{symbol}.{timeframe} | store (S81+) |

### Status Lifecycle Extension

S80 adds `sent` to the lifecycle to support venue round-trips:

```
submitted → sent → accepted → filled
                 → rejected
         → accepted → filled / partially_filled / cancelled
```

Paper mode skips `sent` (submitted → filled directly). Venue mode will use `sent`
when the adapter confirms the order was dispatched to the exchange.

## What This Step Does NOT Do

- No real exchange integration (paper only)
- No multi-venue support
- No OMS, portfolio, or position tracking
- No fill projection in store (consumer exists, projection deferred to S81)
- No venue_market_order in derive (reuses paper_order stream for now)
- No HTTP query surface for venue orders (deferred to S81)
- No retry framework beyond single-attempt VenuePort call

## Configuration

```jsonc
// deploy/configs/execute.jsonc
{
  "log": { "level": "info", "format": "text" },
  "http": { "addr": ":8084" },
  "nats": { "enabled": true, "url": "nats://nats:4222" },
  "pipeline": {
    "execution_families": ["paper_order"]
  }
}
```

## Observability

- Health server on `:8084/healthz`
- Trackers: `venue-adapter`, `venue-consumer`
- Stats on actor stop: processed, filled, skipped_stale, skipped_halt, errors
- Structured logging for every gate skip, fill, and error

## Invariants

- **INV-1**: Kill switch halts venue execution within 1 event cycle
- **INV-2**: Stale intents are never submitted to VenuePort
- **INV-3**: Every fill event carries correlation_id/causation_id from the originating intent
- **INV-4**: VenuePort never sees an intent without domain validation
- **INV-5**: Store remains the sole read-side authority (execute does not write to KV)
