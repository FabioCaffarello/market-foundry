# Gateway Pattern — Market Foundry

> Canonical document. Defines the gateway binary's role as a stateless HTTP-to-NATS translator.
> Approved: 2026-03-16. The gateway must remain a thin composition root with zero domain logic.

---

## Definition

The gateway is a **stateless HTTP-to-NATS translator**. It accepts HTTP requests, encodes them into NATS request/reply messages, waits for a domain binary to respond, decodes the reply, and returns an HTTP response. It holds no domain state, owns no repositories, and publishes no events.

The binary lives at `cmd/gateway`. The actor is `Gateway` in `internal/actors/scopes/gateway/`. The NATS source identifier is `gateway.http`.

---

## What Belongs in the Gateway

| Responsibility | Location |
|---|---|
| HTTP route registration | `internal/interfaces/http/routes/` |
| Readiness and liveness probes | `cmd/gateway/readiness.go` |
| Request correlation (NATS request/reply) | `internal/adapters/nats/` gateway clients |
| Use-case wiring (client-side use cases only) | `cmd/gateway/run.go` |
| NATS request client construction | `cmd/gateway/gateway.go` |

The gateway binary is a **composition root**. It builds NATS request clients, instantiates client-side use cases, assembles route groups, spawns the `Gateway` actor, and blocks on shutdown signals.

---

## What Does NOT Belong in the Gateway

- **Domain logic.** No aggregates, no state transitions, no validation rules.
- **Repositories or persistent storage.** The gateway never reads from or writes to KV stores, databases, or files.
- **Event publishing.** The gateway never publishes to JetStream streams. It only uses NATS request/reply.
- **JetStream subscriptions.** Durable consumers belong in domain binaries (`ingest`, `derive`, `store`), never in the gateway.
- **Long-lived background processing.** The gateway serves requests and returns. It does not run samplers, projections, or watchers.

---

## Request/Reply Flow

```
HTTP client
  |
  v
Gateway (cmd/gateway)
  | 1. Decode HTTP request
  | 2. Encode into NATS request payload
  | 3. Publish to NATS subject, await reply
  |       |
  |       v
  |   Domain binary (e.g., configctl, store)
  |       | 4. Decode request from NATS
  |       | 5. Execute domain logic
  |       | 6. Encode reply, publish to reply subject
  |       |
  | 7. Decode NATS reply
  | 8. Return HTTP response
  v
HTTP client
```

The gateway adapter (`internal/adapters/nats/`) handles encoding and subject routing. The source identifier `gateway.http` is passed on every request so domain binaries can trace the origin.

---

## Current Route Groups

| Group | Purpose | Domain binary |
|---|---|---|
| `configctl` | Configuration CRUD: create draft, validate, compile, activate, list, get active | configctl (server) |
| `evidence` | Candle queries: get latest candle by source/symbol/timeframe | store |

Route groups grow as domains are added. Each group is a flat set of handlers under a URL prefix. Depth stays constant: one handler per endpoint, one use case per handler, one NATS subject per use case.

---

## Readiness

The gateway readiness probe gates on **configctl availability**. It issues a lightweight `ListConfigs` request through the NATS gateway client. If NATS is disabled or the configctl binary is unreachable, the probe fails and the gateway reports not-ready.

The **evidence store probe is non-blocking**. If the store binary is unavailable, the readiness check logs a warning but still passes. The `/evidence/candles/latest` endpoint returns 503 independently; the gateway itself remains ready to serve configctl routes.

---

## Composition Root Structure

`cmd/gateway/run.go` follows a strict top-down wiring sequence:

1. **Logger** — build from config, set as default.
2. **Actor engine** — create the Hollywood engine.
3. **NATS clients** — one request client per domain gateway (configctl, evidence). Each is optional and degrades gracefully.
4. **Use cases** — client-side use cases wrapping the gateway ports.
5. **Routes** — assemble all route groups with their dependencies, including the readiness checker.
6. **Actor spawn** — spawn `Gateway` actor with HTTP config and routes.
7. **Signal handling** — block until SIGINT/SIGTERM, then poison the actor tree.

No step depends on a later step. No circular wiring. No lazy initialization.
