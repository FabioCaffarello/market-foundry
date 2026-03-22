# Activation State Queryability via Gateway HTTP

> S344 — Venue Activation Wave

## Purpose

This document defines the HTTP surface for querying the canonical activation state of the market-foundry venue path. Prior to S344, the activation surface existed as a domain type computed locally in the execute binary but was not queryable by operators via the gateway HTTP API. S344 closes this gap by wiring the full three-dimensional activation surface through the store query responder to the gateway HTTP layer.

## Architecture

The activation surface requires three dimensions:

| Dimension | Source | Mutability |
|-----------|--------|------------|
| Adapter | Execute binary config (`venue.type`) | Immutable per process |
| Gate | NATS KV `EXECUTION_CONTROL/global` | Runtime-mutable |
| Credentials | Execute binary env vars | Immutable per process |

**Challenge**: The gateway binary does not have access to the execute binary's process-local state (adapter, credentials).

**Solution**: The execute binary publishes its immutable dimensions to the `EXECUTION_CONTROL` KV bucket under the `dimensions` key at startup. The store query responder reads both the gate state and the published dimensions, composes the full `ActivationSurface`, and serves it via NATS request/reply. The gateway queries the store and exposes the result via HTTP.

### Data Flow

```
execute binary (startup)
    │
    ├── writes dimensions {adapter, credentials} → NATS KV EXECUTION_CONTROL/dimensions
    │
operator (HTTP)
    │
    └── GET /activation/surface
            │
            gateway binary
                │
                └── NATS request → execution.activation.surface
                        │
                        store binary (query responder)
                            │
                            ├── reads EXECUTION_CONTROL/global → gate state
                            ├── reads EXECUTION_CONTROL/dimensions → adapter + credentials
                            └── composes ActivationSurface → reply
```

### Graceful Degradation

If the execute binary has not started (dimensions not yet published), the store returns a surface with `adapter=unknown` and `credentials=unknown`. The effective mode computation still works — `unknown` adapter produces a non-live mode. This prevents false positives and makes the system state transparent.

## HTTP Endpoint

| Method | Path | Description |
|--------|------|-------------|
| GET | `/activation/surface` | Query the canonical activation surface |

### Response Contract

```json
{
  "surface": {
    "adapter": "venue",
    "gate": {
      "status": "halted",
      "reason": "operator halt",
      "updated_at": "2026-03-22T10:30:00Z",
      "updated_by": "admin"
    },
    "credentials": "present",
    "effective": "venue_halted",
    "observed_at": "2026-03-22T10:30:05Z"
  }
}
```

### Error Responses

| Code | Condition |
|------|-----------|
| 200 | Surface composed successfully |
| 503 | Execution control gateway unavailable (NATS down) |

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/execution/activation.go` | Added `ActivationDimensions` type |
| `internal/adapters/nats/natsexecution/control_kv_store.go` | Added `GetDimensions`/`PutDimensions` methods |
| `internal/adapters/nats/natsexecution/registry.go` | Added `ActivationSurfaceGet` spec |
| `internal/adapters/nats/natsexecution/control_gateway.go` | Added `GetActivationSurface` method |
| `internal/application/ports/execution.go` | Extended `ExecutionControlGateway` interface |
| `internal/application/executionclient/get_activation_surface.go` | New use case |
| `internal/interfaces/http/handlers/activation.go` | New HTTP handler |
| `internal/interfaces/http/routes/activation.go` | New route registration |
| `internal/interfaces/http/routes/core.go` | Added `ActivationFamilyDeps`, wired into `DefaultRoutes` |
| `cmd/gateway/compose.go` | Wired activation use case |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Publishes dimensions at startup |
| `internal/actors/scopes/store/query_responder_actor.go` | Added activation surface handler |

## Relationship to Prior Stages

- **S339**: Defined the `ActivationSurface` domain type and three-dimensional model.
- **S341**: Suggested verifying queryable state via gateway HTTP.
- **S342–S343**: Proved activation lifecycle works end-to-end.
- **S344**: Makes the activation surface queryable via HTTP, closing the explainability gap.

## Limits

- The dimensions KV entry is written once at execute binary startup. If the execute binary restarts with different config, the dimensions update. There is no push notification — the gateway reads on demand.
- The endpoint reflects the last-known dimensions. If the execute binary is down, stale dimensions may persist in KV.
- No history or audit log of dimension changes — only the current snapshot.
