# Domain Readiness

**Date**: 2026-03-16
**Status**: Pre-absorption — domains are planned but not yet implemented

## Current State

The repository contains only the **configctl** domain as a preserved foundation. All quality-service-specific domains (validator, consumer, emulator, dataplane) have been removed.

## Planned Domain Evolution Axes

The following domains are anticipated for the marketmonkey absorption phase. None of these are implemented yet.

### 1. Observation
Capturing and structuring raw market data from external sources.

### 2. Evidence
Transforming observations into structured evidence with provenance.

### 3. Signal
Deriving actionable signals from aggregated evidence.

### 4. Strategy
Defining and managing trading/market strategies.

### 5. Risk
Assessing and managing risk across strategies and positions.

### 6. Execution
Orchestrating trade execution based on signals and strategies.

### 7. Portfolio
Managing portfolio state, positions, and performance tracking.

## Foundation Available for New Domains

Each new domain can leverage the following foundation:

| Component | What It Provides |
|-----------|-----------------|
| `internal/shared/settings/` | Config loading and validation |
| `internal/shared/bootstrap/` | Logger setup, config bootstrap |
| `internal/shared/problem/` | Structured error handling |
| `internal/shared/envelope/` | Message envelope pattern |
| `internal/shared/events/` | Domain event dispatcher |
| `internal/shared/memdb/` | In-memory database for prototyping |
| `internal/actors/common/` | Actor engine, lifecycle, entrypoint |
| `internal/adapters/nats/` | NATS connection, request/reply, codecs |
| `internal/interfaces/http/` | HTTP server, routing, handlers |
| `tools/raccoon-cli/` | Architecture enforcement, quality gates |

## Boundary Guidelines

When implementing new domains:

1. **Domain isolation**: Each domain gets its own package under `internal/domain/`
2. **Application layer**: Use cases go in `internal/application/<domain>/`
3. **Ports**: Define gateway interfaces in `internal/application/ports/`
4. **Adapters**: Infrastructure adapters in `internal/adapters/`
5. **No cross-domain imports**: Domains communicate through events or messaging, not direct imports
6. **Layer direction**: domain ← application ← adapters ← actors ← interfaces ← cmd

## What Must NOT Happen

- Do not recreate the quality-service validation pipeline under new names
- Do not bypass the actor model for service orchestration
- Do not introduce Kafka without explicit architectural justification
- Do not add domains that don't map to market-foundry's purpose
