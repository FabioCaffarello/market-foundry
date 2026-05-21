# Operational Lifecycle Queryability and Read Consolidation

Stage: S413 | Wave: Production Readiness Hardening | Date: 2026-03-23

## Purpose

This document describes the consolidated operational read surfaces for the execution lifecycle after S411-S412 established durable persistence and endurance-validated stability across all lifecycle states.

## Problem Statement

After S411-S412, all lifecycle states (submitted, filled, partially_filled, rejected) persist correctly through both KV (latest-only) and ClickHouse (historical). However, operational queryability has three practical gaps:

1. **No key enumeration** -- KV buckets store latest state per partition key (source/symbol/timeframe), but there is no way to enumerate all tracked keys without foreknowledge.
2. **No cross-key lifecycle overview** -- The composite status query (`execution.query.status.latest`) requires exact source/symbol/timeframe. There is no list view.
3. **No unified lifecycle summary** -- Paper order, fill, and rejection surfaces are queryable independently but there is no single surface that shows all tracked entries with their effective propagation.

## Design

### New Surface: Lifecycle List Query

A new NATS request/reply route `execution.query.lifecycle.list` returns a list of all tracked partition keys across the three execution KV buckets, with per-key lifecycle state and effective propagation.

```
LifecycleListQuery {}  -->  LifecycleListReply {
    Entries: []LifecycleEntry
    Total:   int
}

LifecycleEntry {
    Key:                string    // "source.symbol.timeframe"
    Source:             string
    Symbol:             string
    Timeframe:          int
    IntentStatus:       string    // paper_order status (or empty)
    IntentTimestamp:     *time.Time
    FillStatus:         string    // venue fill status (or empty)
    FillTimestamp:       *time.Time
    RejectionStatus:    string    // venue rejection status (or empty)
    RejectionTimestamp:  *time.Time
    Propagation:        string    // effective lifecycle via DeriveEffectivePropagation
}
```

### Implementation

1. **KVStore.Keys()** -- New method on `natsexecution.KVStore` that calls NATS KV `ListKeys()` to enumerate all partition keys in a bucket. Returns empty slice when bucket is empty.

2. **QueryResponderActor.handleExecutionLifecycleList** -- Reads keys from all three execution buckets (paper_order, venue_fill, venue_rejection), merges into a unique set, then reads each key from each bucket to build the lifecycle entry.

3. **Registry.LifecycleList** -- New `ControlSpec` wired in `DefaultRegistry()` for the lifecycle list route.

### Operational Semantics

- The list query is **eventually consistent** -- it reads from KV buckets that are themselves projections of JetStream events. A recently published event may not yet be materialized.
- The query reads across three separate KV buckets per key. Each bucket read is independent; a failure in one bucket does not block the others.
- Empty fields (e.g., `FillStatus: ""`) indicate no data exists in that bucket for the partition key.
- The `Propagation` field is computed using the same `DeriveEffectivePropagation()` logic used by the composite status query, ensuring consistency.

## Read Surface Inventory (Post-S413)

| Surface | Route | Storage | Scope | Notes |
|---------|-------|---------|-------|-------|
| Paper order latest | `execution.query.paper_order.latest` | KV | Per-key | Derive output |
| Venue fill latest | `execution.query.venue_market_order.latest` | KV | Per-key | Execute output |
| Venue rejection latest | `execution.query.venue_rejection.latest` | KV | Per-key | S407 audit |
| Composite status | `execution.query.status.latest` | KV (3 buckets + control) | Per-key | S387 propagation |
| **Lifecycle list** | `execution.query.lifecycle.list` | KV (3 buckets) | **All keys** | **S413 consolidation** |
| Execution history | ClickHouse `executions` table | ClickHouse | Filtered | Historical with time range |
| Control gate | `execution.control.get` | KV | Global | Kill switch |
| Activation surface | `execution.activation.surface` | KV | Global | S339 effective mode |

## What This Does NOT Do

- Does not add dashboards or analytics surfaces.
- Does not add ClickHouse-backed list queries (ClickHouse already supports `status` filter).
- Does not add pagination to the KV list (KV bucket cardinality is bounded by the number of active source/symbol/timeframe combinations, typically < 100).
- Does not add real-time streaming/watch on lifecycle changes.
