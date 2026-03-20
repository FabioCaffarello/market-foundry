# Current Baseline — Recovery and Restart Semantics

> S140 deliverable. Documents the validated recovery, restart, and shutdown semantics of the market-foundry baseline as of 2026-03-19.

---

## 1. Shutdown Semantics

### Signal Handling

All services use `actorcommon.WaitTillShutdown()` which blocks on `os.Interrupt`, `syscall.SIGTERM`, and `syscall.SIGINT`.

**Shutdown sequence (per service):**

| Phase | Action | Timeout | Notes |
|-------|--------|---------|-------|
| 1 | Signal received | — | Unblocks `WaitTillShutdown` |
| 2 | Actor poison pill | 10s | Stops message processing, drains in-flight, recurses to children |
| 3 | Health server shutdown | 5s | Stops heartbeat monitor, drains HTTP requests |
| 4 | Deferred cleanups | — | NATS client close, gateway connections close (LIFO) |

**Total max shutdown window: 15s.** Orchestration layer (docker-compose `stop_grace_period`) should be configured accordingly.

### What Happens to In-Flight State on Shutdown

| Runtime | In-flight state | Behavior on graceful shutdown |
|---------|-----------------|-------------------------------|
| ingest | WebSocket connection | Closed; trades during shutdown gap are lost |
| derive | Candle sampler windows | Lost; partial windows are not flushed to stream |
| store | Consumer processing | Durable consumer position reflects last ack; unacked messages redelivered on restart |
| execute | Paper order intent | Lost if not yet submitted to venue adapter |
| gateway | HTTP requests | Drained within 5s; late requests get connection refused |
| configctl | Config operations | Stream-backed; no in-flight loss |

### Unhandled Signals

Only SIGTERM, SIGINT, and os.Interrupt are handled. All other signals (SIGKILL, SIGQUIT, etc.) cause immediate process termination with no cleanup. This is expected and acceptable — NATS durable consumers handle the recovery.

---

## 2. Restart Semantics

### Service Restart Behavior

On restart, every service follows the same bootstrap sequence:

1. Config load from JSONC → validation → fail-fast on invalid config
2. Logger initialization
3. Actor engine creation
4. NATS connection (single attempt, no retry) → `os.Exit(1)` on failure
5. Service-specific wiring (gateways, trackers, consumers)
6. Actor spawn → children spawned recursively
7. Health server start (background)
8. Block on signal

**Critical invariant:** Services do not retry NATS connection. If NATS is unavailable at startup, the service exits immediately. External orchestration (docker-compose restart policy, Kubernetes restart) must handle rescheduling.

### Per-Service Restart Recovery

#### configctl
- **Recovery mechanism:** Replays `CONFIGCTL_EVENTS` stream from NATS
- **Data loss:** None — fully replay-able
- **Recovery time:** Seconds (fast stream replay)
- **Post-restart state:** Identical to pre-shutdown

#### ingest
- **Recovery mechanism:** Reconnects Binance WebSocket, queries configctl for active bindings
- **Data loss:** All trades during downtime (exchange does not buffer for consumers)
- **Recovery time:** ~2s (WebSocket reconnect + binding query)
- **Post-restart state:** Functional but with observation gap

#### derive
- **Recovery mechanism:** Durable consumer `derive-observation` resumes from last ack; samplers reset on first trade
- **Data loss:** Up to one full window per timeframe (see cold-start doc)
- **Recovery time:** First candle emitted after one full window elapses post-restart
- **Post-restart state:** Producing valid candles, but first post-restart candle may have fewer trades than normal

#### store
- **Recovery mechanism:** Durable consumers resume from last ack; NATS KV projections are idempotent
- **Data loss:** None — stream + KV survive restart
- **Recovery time:** Seconds (consumer resume)
- **Post-restart state:** Identical to pre-shutdown (KV already contains latest values)

#### execute
- **Recovery mechanism:** Durable consumer resumes; paper fills in KV survive
- **Data loss:** In-flight paper order intents not yet submitted (no WAL)
- **Recovery time:** Seconds
- **Post-restart state:** Functional; may miss one execution cycle if intent was in-flight

#### gateway
- **Recovery mechanism:** Immediate — stateless HTTP proxy
- **Data loss:** None
- **Recovery time:** Instant (TCP listener bind)
- **Post-restart state:** Identical

### Restart Order Dependencies

| Runtime | Hard dependency | Soft dependency |
|---------|----------------|-----------------|
| NATS | None | — |
| configctl | NATS | — |
| gateway | NATS | configctl (for readiness), store (for query probes) |
| ingest | NATS, configctl | — |
| derive | NATS | — |
| store | NATS | — |
| execute | NATS | — |

**Safe restart order:** NATS → configctl → {ingest, derive, store, execute} → gateway

**Individual service restart:** Any service except NATS can be restarted independently without cascading failures. Gateway degrades gracefully if upstream services are temporarily unavailable.

---

## 3. NATS Reconnection Semantics

### Connection Layer

The NATS connection in `internal/adapters/nats/connection.go` performs a single synchronous `nats.Connect()` call. There is no application-level retry loop.

**Reconnection behavior relies on the nats.go client library defaults:**
- The Go NATS client has built-in reconnection with exponential backoff
- Default: up to 60 reconnect attempts, 2s reconnect wait
- During disconnection: publishes are buffered (up to flush limit); subscriptions auto-resume on reconnect

**Application-level behavior during NATS disconnect:**
- Health checks (`/readyz`) fail immediately (TCP dial check)
- Consumers stall (no new messages delivered)
- Publishes may buffer briefly, then fail
- Services do NOT exit on transient disconnect — only on initial connect failure

### WebSocket Reconnection (ingest)

Binance WebSocket adapter has its own reconnection with exponential backoff: 1s → 2s → 4s → ... → 60s cap. This is independent of NATS reconnection.

---

## 4. Recovery Guarantees Summary

| Guarantee | Level | Notes |
|-----------|-------|-------|
| NATS stream data survives restart | **Guaranteed** | JetStream persistence with configured retention |
| NATS KV latest values survive restart | **Guaranteed** | KV backed by JetStream |
| Durable consumer position survives restart | **Guaranteed** | NATS durable consumer semantics |
| Config state survives restart | **Guaranteed** | Event-sourced via NATS stream |
| In-memory candle samplers survive restart | **Not guaranteed** | Ephemeral; reset on restart |
| In-flight paper order intents survive restart | **Not guaranteed** | No WAL; lost on crash |
| Observation continuity during restart | **Not guaranteed** | Exchange does not buffer |
| Gateway query availability during restart | **Best effort** | Depends on upstream service availability |

---

## 5. Crash vs Graceful Shutdown

| Aspect | Graceful (SIGTERM/SIGINT) | Crash (SIGKILL/OOM) |
|--------|--------------------------|---------------------|
| Actor drain | Yes (10s) | No |
| Health server drain | Yes (5s) | No |
| NATS client close | Yes (deferred) | No (server detects via heartbeat) |
| Consumer position | Reflects last ack | Reflects last ack (same) |
| Partial candle window | Lost (not flushed) | Lost (not flushed) |
| Recovery difference | None meaningful | NATS server detects client gone after ~30s heartbeat timeout |

**Key insight:** The practical difference between graceful and crash shutdown is minimal for this system. The primary recovery mechanism (durable consumers + KV) works identically in both cases. The only difference is cleanup timing — graceful shutdown releases resources immediately, while crash relies on NATS server-side timeout detection.

---

## 6. Operational Expectations

### Time to Healthy After Restart

| Scenario | Expected time | Measured by |
|----------|---------------|-------------|
| Single service restart | < 30s | `/readyz` returns 200 |
| Full stack restart | < 2 min | All `/readyz` return 200 |
| First data after restart | 60–75s | First candle appears in KV |
| RSI convergence (60s TF) | ~15 min | RSI signal emitted (requires 15 candles) |
| RSI convergence (3600s TF) | ~15 hours | RSI signal emitted (requires 15 × 1h candles) |

### Phase Progression After Restart

```
starting (< 30s, no events)
  → warming (trackers registered, awaiting first event)
    → active (all trackers receiving events)
```

Operators should expect `starting → warming` within 30s, and `warming → active` within one full candle window (60s for shortest timeframe).
