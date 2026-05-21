# Stage S14 — Store Query & Read Path Consolidation

**Status:** Complete
**Date:** 2026-03-16
**Scope:** Consolidate the read path through store; harden server dependency on store

---

## Objective

Move the server's evidence read path to depend on `store` (not `derive`),
completing the write/read separation started in S13. Ensure the compose
dependency graph, readiness checks, and code comments reflect this reality.

## Problem Statement

After S13, the store binary serves `evidence.query.candle.latest` from NATS KV,
but several artifacts still referenced derive as the read path owner:

1. **Compose**: server depended only on configctl — no dependency on store
2. **Readiness**: server only checked configctl health, not evidence store
3. **Comments**: `EvidenceGateway`, `GetLatestCandleUseCase`, and port docs said "query derive"
4. **Startup order**: server could start before store, causing query failures

## Changes

### 1. Compose Dependency Graph

**Before:**
```
nats → configctl → server
nats → configctl → ingest
nats → derive → store
```

**After:**
```
nats → configctl ─────→ server
nats → configctl → ingest
nats → derive → store → server
```

Server now waits for store to be healthy before starting. This ensures the
evidence query path is available when server begins accepting HTTP traffic.

### 2. Readiness Check — Evidence Store Probe

The server's `/readyz` endpoint now probes the evidence store during health checks:

- **Mandatory**: NATS reachable + configctl responsive
- **Non-blocking**: evidence store probe logs a warning if unreachable, but does
  NOT fail readiness. This preserves graceful degradation — the server can still
  serve configctl routes even if store is temporarily unavailable.

The probe sends a lightweight `evidence.query.candle.latest` request with dummy
parameters. If store responds (even with "no candle found"), the read path is healthy.

New tests:
- `TestServerReadinessCheckerPassesWithEvidenceStore` — happy path
- `TestServerReadinessCheckerPassesWhenEvidenceStoreIsUnavailable` — graceful degradation

### 3. Comment & Documentation Cleanup

All references to "derive" in the read path have been updated to "store":

| File | Change |
|------|--------|
| `internal/adapters/nats/evidence_gateway.go` | "query derive" → "query the store" |
| `internal/application/evidenceclient/get_latest_candle.go` | "queries derive" → "queries the store" |
| `internal/application/ports/evidence.go` | Added "store binary serves these queries" |
| `cmd/server/gateway.go` | Added doc comment explaining store dependency |
| `cmd/server/run.go` | "if derive is not running" → "if the store is not running" |

## Files Changed

| File | Change |
|------|--------|
| `cmd/server/readiness.go` | Extended with evidence gateway probe + evidence parameter |
| `cmd/server/readiness_test.go` | Added 2 new tests, updated signatures |
| `cmd/server/run.go` | Pass evidence gateway to readiness checker |
| `cmd/server/gateway.go` | Updated doc comment |
| `internal/adapters/nats/evidence_gateway.go` | Updated comment |
| `internal/application/evidenceclient/get_latest_candle.go` | Updated comment |
| `internal/application/ports/evidence.go` | Updated comment |
| `deploy/compose/docker-compose.yaml` | server depends_on store |

## Architectural Rationale

### Why non-blocking evidence probe?

The server serves two orthogonal concerns:
1. **Config control plane** — create/validate/compile/activate configs
2. **Evidence read plane** — query candle projections

Making the evidence probe blocking would mean the server can't serve config
routes when store is down. This would couple the config control plane to the
evidence read plane — exactly the kind of coupling we're eliminating.

The evidence endpoint already returns HTTP 503 when store is unreachable.
The readiness probe adds observability (logged warning) without coupling.

### Why server depends on store in compose?

Even though the evidence probe is non-blocking for readiness, the compose
dependency ensures store is started and healthy before server. This means:
- In normal operation, the evidence path works immediately
- If store fails after startup, server continues serving config routes
- The dependency is a startup ordering concern, not a runtime coupling

## Read Path — Final State

```
HTTP GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
    ↓
server (HTTP handler)
    ↓ EvidenceGateway.GetLatestCandle()
NATS request/reply → evidence.query.candle.latest
    ↓
store (QueryResponderActor)
    ↓ CandleKVStore.Get()
NATS KV bucket: CANDLE_LATEST
    ↓
key: binancef.btcusdt.60 → JSON(EvidenceCandle)
    ↓
HTTP 200 { "candle": { ... } }
```

No component in this path touches derive. The read path is fully decoupled
from the write path.

## Remaining Gaps

1. **Store has no HTTP health endpoint** — uses process-alive check in compose.
   If store's NATS consumer hangs silently, compose still thinks it's healthy.
2. **No circuit breaker on evidence gateway** — if store is slow, server's
   evidence handler waits until NATS request timeout (2s). No backpressure.
3. **Readiness probe sends a real query** — the probe query (source=readiness-probe)
   reaches store and gets processed. Trivial overhead but not a dedicated ping.

## Preparation for S15

1. **Store HTTP health endpoint** — expose `/healthz` and `/readyz` with
   KV connectivity check, similar to server
2. **Smoke test update** — `make smoke` should verify the full read path
   through store, not just that the endpoint responds
3. **Second timeframe** — add 300s candles to prove multi-timeframe readiness
4. **Derive binding watcher** — derive should react to configctl events
   dynamically (currently only queries at startup)
