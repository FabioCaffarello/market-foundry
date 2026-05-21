# Runtime Recentralization

## Summary

After stages S6-S17 (first vertical slice through multi-symbol proof), architectural drift had begun to emerge. This document records the corrections made and decisions crystallized.

## Drifts Corrected

### 1. Binary Identity: server to gateway

- **Problem**: Binary `cmd/server` contradicted the canonical "gateway" identity defined in runtime-target.md and actor-ownership.md
- **Correction**: Renamed cmd/server to cmd/gateway, internal/actors/scopes/server to internal/actors/scopes/gateway, deploy/configs/server.jsonc to deploy/configs/gateway.jsonc
- **Scope**: go.mod, go.work, Makefile, docker-compose.yaml, all docs, scripts, test files
- **Source identifier**: changed from "server.http" to "gateway.http" in NATS request clients

### 2. Actor Identity: Server to Gateway

- **Problem**: Actor struct named "Server", package "actorserver", spawned as "server"
- **Correction**: Renamed to "Gateway", "actorgateway", spawned as "gateway"

### 3. Readiness Function Naming

- **Problem**: newServerReadinessChecker contradicted gateway identity
- **Correction**: Renamed to newGatewayReadinessChecker with consistent error messages

### 4. Documentation Drift

- **Problem**: runtime-target.md still had "rename note" and "Not started" for operational services
- **Correction**: Updated phase map to reflect current operational state (all 5 binaries exist)

## Decisions Crystallized

### Gateway Pattern

- Gateway is a stateless HTTP-to-NATS translator (documented in gateway-pattern.md)
- It owns HTTP routes and NATS request clients, nothing else
- Composition root follows: config, NATS clients, use cases, routes, actor engine

### Derive Pipeline Pattern

- Canonical pattern: Consume, Transform, Publish (documented in derive-pipeline-pattern.md)
- Transform logic is pure and I/O-free (CandleSampler)
- Future pipelines follow the same structure

### Read Model Authority

- Store is the sole read-side authority (documented in read-model-authority.md)
- Gateway queries store via NATS request/reply -- never accesses KV directly
- Store never produces canonical events -- only materializes projections

## Intentional Deviations

- configctl subjects still use pre-taxonomy naming (migration documented in stream-taxonomy.md, deferred as separate task)
- Health binaries expose HTTP for healthz/readyz checks (documented exception to "gateway is the only HTTP surface" -- health endpoints are operational, not domain)

## Remaining Limits

- No deactivation support in binding watchers (known from S16)
- Silent trade drop when source scope not yet created (race window on startup)
- No health metrics endpoint yet (S18 scope)
