# Stage S08 — Minimal Query/Projection: Read Path for Evidence

**Status:** Complete
**Date:** 2026-03-16

## Executive Summary

S08 closes the read path of the first vertical slice. The gateway (server) binary can now query derive for the latest sampled candle via NATS request/reply. The derive binary exposes a query responder that reads the candle sampler's current state and returns it synchronously.

No store, no persistent projection, no materialized view. The candle sampler actor **is** the read model — it holds the current window state in memory and responds to snapshot queries via Hollywood's actor request/reply mechanism.

## Solution: In-Memory Actor-Based Read Model

### Architecture Decision

The simplest correct read model for the first slice is the sampler actor itself. It already holds the current candle state. Rather than introducing a separate store, projection consumer, or database, the query path asks the sampler directly:

```
HTTP GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
  → EvidenceWebHandler
    → GetLatestCandleUseCase (validates input)
      → EvidenceGateway (NATS request/reply)
        → evidence.query.candle.latest
          → QueryResponderActor (in derive)
            → engine.Request(samplerPID, snapshotCandleRequest{})
              → SamplerActor.Snapshot()
            ← snapshotCandleReply{Candle}
          ← CandleLatestReply
        ← CBOR envelope
      ← decoded reply
    ← CandleLatestReply
  ← JSON response
```

### Why This Is Correct (Not a Hack)

1. **State ownership is clear** — The sampler actor owns the candle state. Queries go through the actor's mailbox, serialized with trade processing. No concurrent access issues.

2. **The pattern matches the existing architecture** — ConfigctlGateway uses the exact same NATS request/reply pattern (gateway → NATS → responder → actor → reply). The evidence query follows the identical flow.

3. **No premature persistence** — The first slice doesn't need historical candles, multiple timeframes, or durable projections. It needs "what is the current candle?" — which the sampler knows.

4. **The path to a proper store is clean** — When S09+ needs historical queries, a store binary consuming `EVIDENCE_EVENTS` can build a persistent projection. The query subject (`evidence.query.candle.latest`) and contracts are already stable.

## Files Changed/Created

### New Files

| File | Layer | Purpose |
|------|-------|---------|
| `internal/adapters/nats/evidence_gateway.go` | Adapter | NATS request/reply client for evidence queries (gateway side) |
| `internal/actors/scopes/derive/query_responder_actor.go` | Actor | NATS responder for `evidence.query.candle.latest` |
| `internal/application/evidenceclient/get_latest_candle.go` | Application | Use case with input validation |
| `internal/application/evidenceclient/get_latest_candle_test.go` | Application | Unit tests for validation and gateway delegation |
| `internal/interfaces/http/handlers/evidence.go` | Interface | HTTP handler for `/evidence/candles/latest` |
| `internal/interfaces/http/handlers/evidence_test.go` | Interface | Unit tests for handler |
| `internal/interfaces/http/routes/evidence.go` | Interface | Route registration |
| `tests/http/evidence.http` | Test | HTTP smoke test file |

### Modified Files

| File | Change |
|------|--------|
| `internal/interfaces/http/routes/core.go` | Added `GetLatestCandle` to Dependencies, wired evidence routes |
| `internal/actors/scopes/derive/messages.go` | Added `snapshotCandleRequest` / `snapshotCandleReply` messages |
| `internal/actors/scopes/derive/sampler_actor.go` | Added `onSnapshot` handler for query responses |
| `internal/actors/scopes/derive/derive_supervisor.go` | Spawns `QueryResponderActor` |
| `cmd/server/gateway.go` | Added `newEvidenceGateway()` factory |
| `cmd/server/run.go` | Wired evidence gateway + use case + routes |

## Subject Naming and Ownership

| Subject | Owner | Direction | Pattern |
|---------|-------|-----------|---------|
| `evidence.query.candle.latest` | derive | Gateway → Derive | Request/Reply |

The query subject was already defined in `EvidenceRegistry.CandleLatest` from S05. S08 wires the responder and client.

## Request/Reply Contract

**Request:** `CandleLatestQuery`
```json
{"source": "binancef", "symbol": "btcusdt", "timeframe": 60}
```

**Reply:** `CandleLatestReply`
```json
{
  "candle": {
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "open": "84521.30000000",
    "high": "84589.90000000",
    "low": "84510.00000000",
    "close": "84575.40000000",
    "volume": "12345678.00000000",
    "trade_count": 87,
    "open_time": "2026-03-16T12:00:00Z",
    "close_time": "2026-03-16T12:01:00Z",
    "final": false
  }
}
```

If no candle is active yet (no trades received), `candle` is `null`.

## How to Test

```bash
# Boot the full stack
make up

# Wait ~60s for trades to accumulate, then:
curl -s http://127.0.0.1:8080/evidence/candles/latest?source=binancef\&symbol=btcusdt\&timeframe=60 | jq .

# Or use the .http file:
# tests/http/evidence.http
```

## Intentional Gaps

1. **No historical candles** — Only the current (or most recently finalized interim) candle is returned. No time-series queries.

2. **No persistent projection** — The read model is in-memory. If derive restarts, the current candle state is lost (the sampler starts fresh from the next trade).

3. **No store binary** — The `store` binary from the runtime target is not yet implemented. The sampler actor serves as a temporary read model.

4. **Single source/symbol/timeframe** — The query always hits the single hardcoded sampler. Multi-key routing is deferred.

5. **No caching** — Every query goes through NATS to derive. There's no HTTP caching, ETags, or gateway-side projection.

6. **Graceful degradation** — If derive is not running, the gateway returns 503 on the evidence endpoint. This is correct behavior, not a gap.

## Risks Before S09

1. **Sampler restart state loss** — When derive restarts, the current candle window is lost. The OBSERVATION_EVENTS stream retains data for 6h, so replaying from a JetStream offset could restore state. This is not implemented.

2. **Query timeout under load** — The sampler actor processes both trades and queries in a single mailbox. Under high trade volume, snapshot queries may be delayed. Hollywood's `engine.Request` has a configurable timeout (defaults to `NATS.RequestTimeout`).

3. **Multi-sampler routing** — When adding more symbols/timeframes, the query responder needs to route to the correct sampler. The current design sends all queries to a single sampler PID. A sampler registry (map[key]*actor.PID) in the supervisor would solve this.

4. **Import cycle avoidance** — The `evidenceclient` package defines its own `evidenceGateway` interface instead of importing from `ports` to avoid a cycle (`ports` → `evidenceclient` → `ports`). This is intentional and follows Go's interface-at-the-consumer pattern. The `ports.EvidenceGateway` interface remains as the canonical contract for adapter implementors.

5. **Volume semantics** — The candle volume field contains notional value (sum of price*qty). This should be documented clearly or reconsidered before exposing to external consumers.
