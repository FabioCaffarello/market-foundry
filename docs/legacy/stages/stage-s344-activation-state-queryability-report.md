# Stage S344 — Activation State Queryability via Gateway HTTP

> Canonical HTTP surface for querying the three-dimensional activation state, closing the explainability gap from S341.

## Executive Summary

S344 makes the activation surface queryable via `GET /activation/surface` on the gateway HTTP API. The full three-dimensional activation state (adapter, gate, credentials) and the derived effective mode are now available to operators and runbooks through a single HTTP call. The execute binary publishes its process-local dimensions to NATS KV at startup, the store query responder composes the full surface, and the gateway exposes it via HTTP.

## Motivation

S341 explicitly recommended verifying queryable activation state via gateway HTTP. After S342–S343 proved the activation lifecycle works end-to-end, the remaining gap was operational explainability: operators had no single endpoint to answer "what is this deployment doing right now?" The existing `GET /execution/control` returned only the gate dimension; adapter and credential states were invisible outside the execute binary's process.

## Surface Delivered

| Method | Path | Description |
|--------|------|-------------|
| GET | `/activation/surface` | Full three-dimensional activation surface with audit fields |

### Response Payload

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

### Audit Fields

- `gate.reason` — why the gate is in its current state
- `gate.updated_at` — when the last gate transition occurred
- `gate.updated_by` — who or what changed the gate
- `observed_at` — when the surface was composed (staleness detection)

## Architecture

### Problem

The `ActivationSurface` type (S339) composes three dimensions. Two of these (adapter, credentials) are process-local to the execute binary and immutable per process lifetime. The gateway has no direct access to the execute binary's state.

### Solution

1. **Execute binary** publishes its immutable dimensions (adapter + credentials) to the `EXECUTION_CONTROL` KV bucket under the `dimensions` key at startup.
2. **Store query responder** reads both the gate state (`EXECUTION_CONTROL/global`) and the published dimensions (`EXECUTION_CONTROL/dimensions`), composes the full `ActivationSurface`, and serves it via NATS request/reply.
3. **Gateway** queries the store via the `execution.activation.surface` NATS subject and exposes the result at `GET /activation/surface`.

### Graceful Degradation

If the execute binary has not started, dimensions are absent from KV. The store returns `adapter=unknown` and `credentials=unknown`. The effective mode computation treats `unknown` adapter as non-live (failsafe). Operators see this clearly in the response.

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/execution/activation.go` | Added `ActivationDimensions` type |
| `internal/adapters/nats/natsexecution/control_kv_store.go` | Added `DimensionsKey`, `GetDimensions`, `PutDimensions` |
| `internal/adapters/nats/natsexecution/registry.go` | Added `ActivationSurfaceGet` spec |
| `internal/adapters/nats/natsexecution/control_gateway.go` | Added `GetActivationSurface` gateway method |
| `internal/application/ports/execution.go` | Extended `ExecutionControlGateway` interface |
| `internal/application/executionclient/get_activation_surface.go` | New `GetActivationSurfaceUseCase` |
| `internal/application/executionclient/control_contracts.go` | Pre-existing contracts (no change) |
| `internal/interfaces/http/handlers/activation.go` | New `ActivationWebHandler` |
| `internal/interfaces/http/routes/activation.go` | New `Activation()` route function |
| `internal/interfaces/http/routes/core.go` | Added `ActivationFamilyDeps`, wired into `DefaultRoutes` |
| `cmd/gateway/compose.go` | Wired activation use case from `executionControl` gateway |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Publishes dimensions at startup |
| `internal/actors/scopes/store/query_responder_actor.go` | Added `handleActivationSurfaceGet` handler |
| `scripts/smoke-activation.sh` | Added Phase 9: activation surface queryability |

### New Files

| File | Purpose |
|------|---------|
| `internal/application/executionclient/get_activation_surface.go` | Use case |
| `internal/interfaces/http/handlers/activation.go` | HTTP handler |
| `internal/interfaces/http/routes/activation.go` | Route registration |
| `internal/interfaces/http/routes/activation_test.go` | Route tests |
| `docs/architecture/activation-state-queryability-via-gateway-http.md` | Architecture doc |
| `docs/architecture/activation-http-contracts-audit-fields-and-usage-examples.md` | Contracts and usage |

## Tests and Evidence

### Unit Tests (6 tests)

| Test | Assertion |
|------|-----------|
| `TestActivationRoutesRegisterHandler` | GET /activation/surface returns 200 with full surface payload; all fields (effective, gate.status, adapter, credentials, gate.reason, gate.updated_by) validated |
| `TestActivationSurfaceReturnsAllEffectiveModes` | All four effective modes (paper, venue_halted, venue_live, venue_degraded) produce correct JSON |
| `TestDefaultRoutesIncludesActivationWhenProvided` | Route registered when deps provided |
| `TestDefaultRoutesOmitsActivationWhenNil` | Route absent (404) when deps nil |
| `TestActivationSurfaceUnavailableWhenUseCaseNil` | Returns 503 when gateway unavailable |

### Smoke Test (Phase 9)

Added Phase 9 to `scripts/smoke-activation.sh`:
- Queries `GET /activation/surface` against live stack
- Validates required fields (`observed_at`, `gate.updated_at`)
- Validates effective mode consistency with gate status
- Handles 503 gracefully (execute binary not running)

### Build Verification

- `make build` — all 8 binaries compile
- `make test` — all unit tests pass (including new activation tests)

## Limits

| Limit | Severity | Note |
|-------|----------|------|
| Dimensions written once at startup | Low | Immutable per process; stale only if execute binary is down |
| No history/audit log of surface changes | Low | Only current snapshot; gate changes tracked by KV revision |
| No push/subscription for surface changes | Low | Operators must poll |
| `unknown` adapter/credentials when execute not started | Low | Failsafe — clearly visible in response |

## Preparation for S345

S344 closes the activation queryability gap. The activation wave now has:
- Activation policy and rollout model (S338)
- Canonical activation surface domain type (S339)
- Venue-active smoke (S340)
- Controlled activation verification with live actor path (S341)
- Real venue adapter verification (S342)
- Extended observation window (S343)
- Activation state queryability via HTTP (S344)

Recommended next steps:
1. **Operational runbook validation** — use the activation surface endpoint in a documented incident response or deployment ceremony
2. **Wave evidence gate** — consolidate all S337–S344 evidence into a formal gate assessment
3. **Gate history** — if audit trail depth is needed, consider persisting gate transitions to ClickHouse (additive, not blocking)
