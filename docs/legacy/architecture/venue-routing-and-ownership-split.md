# Venue Routing and Ownership Split

> Stage S85 — Defines the routing model, subject ownership, and binary responsibility split for the venue execution family, distinct from the paper execution family.

## Overview

The execution domain has two families with distinct ownership, routing, and lifecycle semantics. This document specifies the routing model for each family and the ownership boundaries that prevent cross-family coupling.

## Routing Model

### Subject Hierarchy

```
execution.
├── events.                                    # EXECUTION_EVENTS stream
│   ├── paper_order.submitted.{s}.{sym}.{tf}   # Paper family (derive → store)
│   └── venue_market_order.submitted.{s}.{sym}.{tf}  # [FUTURE] Venue family intake
│
├── fill.                                      # EXECUTION_FILL_EVENTS stream
│   └── venue_market_order.{s}.{sym}.{tf}       # Venue family fill (execute → store)
│
├── query.                                     # Request/reply (stateless)
│   ├── paper_order.latest                      # Paper family query
│   ├── venue_market_order.latest               # Venue family query
│   └── status.latest                           # Cross-family composite query
│
└── control.                                   # Request/reply (stateful)
    ├── get                                     # Read global gate
    └── set                                     # Write global gate
```

### Stream Ownership

| Stream | Subject Filter | Owner (Publisher) | Consumers |
|--------|---------------|-------------------|-----------|
| `EXECUTION_EVENTS` | `execution.events.>` | derive | store (paper projection), execute (venue intake — bridge) |
| `EXECUTION_FILL_EVENTS` | `execution.fill.>` | execute | store (fill projection) |

### KV Bucket Ownership

| Bucket | Writer Authority | Readers |
|--------|-----------------|---------|
| `EXECUTION_PAPER_ORDER_LATEST` | store (`ExecutionProjectionActor`) | store (query responder), gateway |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | store (`FillProjectionActor`) | store (query responder), gateway |
| `EXECUTION_CONTROL` | store (query responder — set), derive (read), execute (read) | gateway |

## Binary Responsibility Matrix

| Binary | Paper Family Role | Venue Family Role |
|--------|------------------|-------------------|
| **derive** | Produces `PaperOrderSubmittedEvent` | None (future: produces `VenueOrderIntentEvent`) |
| **execute** | Consumes paper_order events (transitional bridge) | Produces `VenueOrderFilledEvent` |
| **store** | Projects paper intents to KV | Projects venue fills to KV |
| **gateway** | Routes paper queries | Routes venue queries |

## Consumer Routing Table

| Consumer Durable | Stream | Subject Filter | Binary | Family |
|-----------------|--------|---------------|--------|--------|
| `store-execution-paper-order` | EXECUTION_EVENTS | `execution.events.paper_order.submitted.>` | store | paper |
| `execute-venue-market-order-intake` | EXECUTION_EVENTS | `execution.events.paper_order.submitted.>` | execute | venue (bridge) |
| `store-execution-venue-market-order-fill` | EXECUTION_FILL_EVENTS | `execution.fill.venue_market_order.>` | store | venue |

## Ownership Rules

### Rule 1: No Cross-Family Publishing

A binary must not publish events to a family it does not own:
- derive must NOT publish to `execution.fill.>` (venue family stream).
- execute must NOT publish to `execution.events.paper_order.>` (paper family stream).

### Rule 2: Independent Projections

Each family's projection is independent:
- `ExecutionProjectionActor` writes only to `EXECUTION_PAPER_ORDER_LATEST`.
- `FillProjectionActor` writes only to `EXECUTION_VENUE_MARKET_ORDER_LATEST`.
- No projection actor reads from the other family's KV bucket for write decisions.

### Rule 3: Shared Control Gate

The execution control gate (`EXECUTION_CONTROL`) is family-agnostic:
- Both families respect the same gate.
- Gate is global (key: `"global"`) — no per-family or per-symbol gate yet.
- Derive publisher checks gate before publishing paper intents.
- Execute venue adapter checks gate before submitting to venue.

### Rule 4: Composite Queries Cross Families Explicitly

The `StatusLatest` query reads from both families' KV buckets:
- Intent: from `EXECUTION_PAPER_ORDER_LATEST` (paper family).
- Result: from `EXECUTION_VENUE_MARKET_ORDER_LATEST` (venue family).
- Gate: from `EXECUTION_CONTROL` (shared).
- Propagation: derived from both (result.Status > intent.Status > "none").

This is the only legitimate cross-family read surface.

## Transitional Bridge Scope

The execute binary's intake consumer currently subscribes to paper_order subjects. This bridge is bounded by:

1. **Code annotations**: All bridge points are marked with `TRANSITIONAL BRIDGE` comments in:
   - `execution_registry.go` — consumer spec definition
   - `execute_supervisor.go` — consumer spawn
   - `execute/messages.go` — message type
2. **Architecture doc**: `execution-family-separation-after-paper-step.md` documents the migration path.
3. **Drift rule**: `ED-1` tracks the existence of both separation documents.

## Guard Rails

- No real venue API calls until activation gate ceremony.
- No multi-venue routing (single adapter per execute instance).
- No per-family control gate (global gate only).
- No venue-specific intent event type until explicitly introduced.
- No new consumers subscribing to paper subjects for venue-specific processing.

## Future Evolution

| Capability | Prerequisite | Expected Stage |
|-----------|-------------|----------------|
| Venue intent event type | This document | S86+ |
| Venue-specific intake subject | Venue intent event | S86+ |
| Bridge removal | Venue-specific intake subject | S87+ |
| Per-family control gate | Global gate proven | S88+ |
| Multi-venue routing | Single venue proven | S89+ |
