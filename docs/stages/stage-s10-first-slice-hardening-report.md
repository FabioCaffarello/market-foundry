# Stage S10 — First Slice Hardening

**Status:** Complete
**Date:** 2026-03-16

## Executive Summary

S10 hardens the first vertical slice without expanding scope. Changes focus on five areas: input validation, reconnect resilience, structured logging, response type safety, and documentation alignment. No domain logic was added or changed.

## Hardening Applied

### 1. WebSocket Exponential Backoff

**File:** `internal/adapters/exchanges/binancef/websocket.go`

**Before:** Fixed 3-second reconnect delay on every failure.

**After:** Exponential backoff: 1s → 2s → 4s → 8s → 16s → 32s → 60s (cap). Resets to 1s after a connection that lasts longer than 30 seconds (stable connection detected).

This prevents ingest from hammering a failing exchange with rapid reconnect attempts while still recovering quickly from transient failures.

### 2. HTTP Evidence Handler Input Validation

**File:** `internal/interfaces/http/handlers/evidence.go`

**Before:** `timeframe` parsed with `strconv.Atoi` ignoring errors — invalid input silently treated as 0, then rejected deep in the use case layer.

**After:**
- Missing `timeframe` → 400 with `"timeframe query parameter is required"`
- Non-integer `timeframe` → 400 with `"timeframe must be a valid integer"`
- Validation happens at the boundary, before dispatching to the use case

### 3. Response Type Safety

**File:** `internal/interfaces/http/handlers/evidence.go`

**Before:** `latestCandleResponse` used `any` for the candle field and `omitempty` — losing schema strictness and omitting the key when null.

**After:** Uses `*evidence.EvidenceCandle` (typed) without `omitempty` — the `candle` key is always present in the response, either with a candle object or `null`. Consumers can reliably check the key.

### 4. Structured Actor Logging

All actor files in `internal/actors/scopes/ingest/` and `internal/actors/scopes/derive/` were updated with contextual slog attributes.

**Pattern applied:**
```go
a.logger = slog.Default().With("actor", "candle-sampler", "source", a.cfg.Source, "symbol", a.cfg.Symbol)
```

This means every log line from an actor carries its identity without repetition:

| Actor | Attributes |
|-------|-----------|
| `ws-adapter` | actor, symbol |
| `observation-publisher` | actor |
| `observation-consumer` | actor |
| `candle-sampler` | actor, source, symbol, timeframe_s |
| `evidence-publisher` | actor |
| `query-responder` | actor |

Log lines from actors now carry richer context. For example, a publish failure from the evidence publisher now logs: actor, error, code, source, symbol, timeframe, open_time.

### 5. Documentation Alignment

**File:** `DEVELOPMENT.md`

- Removed stale "stub" labels from ingest and derive service descriptions
- Added `make smoke` to the quick reference table
- Updated project structure descriptions to reflect implemented services

## Residues Removed or Neutralized

| Residue | Status | Action |
|---------|--------|--------|
| ingest/derive described as "stub" in DEVELOPMENT.md | Removed | Updated to real descriptions |
| Fixed reconnect delay in WS adapter | Replaced | Exponential backoff with stable-connection reset |
| `any` type in evidence response struct | Replaced | Typed `*evidence.EvidenceCandle` |
| Silent `strconv.Atoi` error on timeframe | Fixed | Explicit 400 response at boundary |
| Flat log messages without actor context | Improved | `slog.With()` on all actors |

## Files Changed

| File | Change |
|------|--------|
| `internal/adapters/exchanges/binancef/websocket.go` | Exponential backoff (1s→60s cap, reset on stable) |
| `internal/interfaces/http/handlers/evidence.go` | Input validation + typed response |
| `internal/interfaces/http/handlers/evidence_test.go` | Tests for missing/invalid timeframe + null candle |
| `internal/actors/scopes/ingest/websocket_actor.go` | Structured logger with actor/symbol context |
| `internal/actors/scopes/ingest/publisher_actor.go` | Structured logger + richer error context |
| `internal/actors/scopes/derive/consumer_actor.go` | Structured logger + stream name in startup log |
| `internal/actors/scopes/derive/sampler_actor.go` | Structured logger with source/symbol/timeframe context |
| `internal/actors/scopes/derive/publisher_actor.go` | Structured logger + richer publish failure context (code, timeframe, open_time) |
| `internal/actors/scopes/derive/query_responder_actor.go` | Structured logger |
| `DEVELOPMENT.md` | Removed stale "stub" labels, added `make smoke` |

## Gaps That Must Remain for Next Phase

These are explicitly documented, not hidden.

### Operational

1. **No real health checks for ingest/derive** — Docker Compose uses process-alive checks. Adding HTTP health endpoints to NATS-only services would require either a health port or NATS-based health probing. Deferred until operational maturity demands it.

2. **No metrics** — The `reportError` hook in the NATS adapter is stubbed. Metrics collection (Prometheus or similar) is not in scope for the first slice.

3. **No alerting** — No mechanism to detect a stale pipeline (ingest connected but no trades flowing). Requires either heartbeat events or metrics.

### Data

4. **No candle state replay on restart** — When derive restarts, the sampler starts fresh. Historical candle data is in `EVIDENCE_EVENTS` (72h retention) but not replayed into the sampler.

5. **No persistent read model** — The sampler actor is the only read model. A proper store binary consuming `EVIDENCE_EVENTS` is the next structural addition.

6. **Volume semantics undecided** — Candle volume is `sum(price * qty)` (notional). Whether this should be base quantity needs a domain decision before external exposure.

### Scale

7. **Single source/symbol/timeframe hardcoded** — Both ingest and derive hardcode `binancef/btcusdt/60s`. Config-driven activation via `configctl.events.config.ingestion_runtime_changed` is the next step.

8. **Unbounded actor mailboxes** — Hollywood's default mailbox has no backpressure. Under extreme trade volume, memory could grow. Monitoring (gap #2) would surface this.

9. **Single NATS connection per adapter** — Each publisher/consumer/responder opens its own NATS connection. Connection pooling may be needed at scale.

## Strategic Recommendation for Next Cycle

The first slice is complete and hardened. The system proves:
- Layer sovereignty works end-to-end
- NATS as sole inter-service bus works
- Actor-per-concern works
- Contract-first works

**Recommended next priorities (in order):**

1. **Config-driven activation** — Replace hardcoded source/symbol with BindingWatcherActor consuming configctl events. This is the structural prerequisite for multi-symbol support without code changes.

2. **Multi-symbol routing in derive** — Introduce `ExchangeScopeActor → SymbolScopeActor → SamplerActor` hierarchy. The current flat-sampler design doesn't scale to multiple symbols.

3. **Store binary (persistent read model)** — Consume `EVIDENCE_EVENTS`, build a queryable projection, and move the `evidence.query.candle.latest` responder from derive into the store. This separates write-path from read-path cleanly.

4. **Second timeframe (300s)** — Adding 5-minute candles proves the sampler and routing design work for multiple timeframes per symbol.

5. **Operational health** — Add real health endpoints to ingest/derive, wire metrics, add pipeline heartbeat detection.

Items 1-2 should be done together (one stage). Item 3 is a separate stage. Items 4-5 can be interleaved.
