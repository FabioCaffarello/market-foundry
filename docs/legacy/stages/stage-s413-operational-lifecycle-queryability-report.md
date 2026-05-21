# Stage S413: Operational Lifecycle Queryability and Read Consolidation

Wave: Production Readiness Hardening | Date: 2026-03-23

## Objective

Consolidate the operational read surfaces for the execution lifecycle, making the query of persisted states (submitted, filled, partially_filled, rejected) more coherent, useful, and auditable without inflating into dashboards or analytics.

## Strategic Context

S411 closed the ClickHouse persistence gap for rejections (RG-1). S412 validated temporal stability and consistency under sustained endurance testing. With all lifecycle states durably persisting across KV and ClickHouse, S413 shifts from "does it persist?" to "can I practically query and audit the lifecycle?"

## What Was Done

### 1. KV Key Enumeration

Added `Keys()` method to `natsexecution.KVStore` that calls NATS KV `ListKeys()` to enumerate all partition keys in a bucket. This closes the fundamental gap where operators had to know exact source/symbol/timeframe to query execution state.

**File:** `internal/adapters/nats/natsexecution/kv_store.go`

### 2. Lifecycle List Query Surface

Added a new NATS request/reply route `execution.query.lifecycle.list` that enumerates all tracked partition keys across the three execution KV buckets (paper_order, venue_fill, venue_rejection) and returns a per-key lifecycle summary with effective propagation.

**New contracts:**
- `LifecycleListQuery` -- empty request (list all)
- `LifecycleEntry` -- per-key summary (source, symbol, timeframe, per-surface status + timestamp, effective propagation)
- `LifecycleListReply` -- entries + total count

**File:** `internal/application/executionclient/contracts.go`

### 3. Registry and Query Responder Wiring

- Added `LifecycleList` ControlSpec to `natsexecution.Registry`
- Wired `handleExecutionLifecycleList` handler in `QueryResponderActor`
- Handler merges keys from all three buckets, reads each key from each bucket, and builds lifecycle entries with `DeriveEffectivePropagation()`

**Files:**
- `internal/adapters/nats/natsexecution/registry.go`
- `internal/actors/scopes/store/query_responder_actor.go`

### 4. Tests

Created `internal/application/execution/s413_lifecycle_queryability_test.go` with 12 test cases across 4 test functions:

| Test | Cases | What It Validates |
|------|-------|-------------------|
| TestS413_LifecycleEntry_PropagationAlignment | 9 | All propagation derivation scenarios (intent-only, fill-only, rejection-only, combinations, timestamp priority) |
| TestS413_LifecycleEntry_FieldPopulation | 1 | Correct field population, absent fields remain empty/nil |
| TestS413_LifecycleListReply_Aggregation | 1 | Multi-entry reply with diverse propagation states |
| TestS413_LifecycleEntry_PartiallyFilledPropagation | 1 | Partially filled status propagates correctly |

### 5. Architecture Documentation

| Document | Purpose |
|----------|---------|
| `docs/architecture/operational-lifecycle-queryability-and-read-consolidation.md` | Design, inventory of all read surfaces, operational semantics |
| `docs/architecture/lifecycle-read-surfaces-list-queries-kv-clickhouse-alignment-and-limitations.md` | KV vs ClickHouse alignment, gap treatment, invariants, limitations |

## Files Changed

| File | Change |
|------|--------|
| `internal/adapters/nats/natsexecution/kv_store.go` | Added `Keys()` method |
| `internal/application/executionclient/contracts.go` | Added `LifecycleListQuery`, `LifecycleEntry`, `LifecycleListReply` |
| `internal/adapters/nats/natsexecution/registry.go` | Added `LifecycleList` ControlSpec |
| `internal/actors/scopes/store/query_responder_actor.go` | Added `handleExecutionLifecycleList`, `parsePartitionKey`, `toSet` helpers; wired route |
| `internal/application/execution/s413_lifecycle_queryability_test.go` | New test file (12 cases) |
| `docs/architecture/operational-lifecycle-queryability-and-read-consolidation.md` | New |
| `docs/architecture/lifecycle-read-surfaces-list-queries-kv-clickhouse-alignment-and-limitations.md` | New |

## Evidence

### Build Verification
- All affected workspace modules compile cleanly: `cmd/store`, `cmd/gateway`, `cmd/execute`, `internal/actors`, `internal/adapters/nats`, `internal/application`

### Test Results
- 12/12 S413 tests pass
- All existing S384, S386, S387 tests continue to pass (no regression)

### Read Surface Consolidation

| Before S413 | After S413 |
|-------------|------------|
| Per-key queries only (must know source/symbol/timeframe) | Per-key queries + lifecycle list across all keys |
| No KV key enumeration | `Keys()` on execution KV stores |
| 7 execution query routes | 8 execution query routes |
| Composite status per-key only | Composite status per-key + lifecycle overview |

## Limitations

| Limitation | Impact | Notes |
|-----------|--------|-------|
| KV latest-only | Low | History available in ClickHouse |
| No KV-to-ClickHouse join | Low | Different concerns: current state vs history |
| Stale KV entries persist | Low | Timestamps in lifecycle list enable identification |
| No pagination on lifecycle list | Low | Bounded cardinality (< 100 in practice) |
| Rejection fields in ClickHouse JSON column | Low | Queryable via JSONExtractString |

## Preparation for S414

S413 completes the read-path consolidation for the Production Readiness Hardening wave. The gate review in S414 should verify:

1. **Write-path completeness** -- All lifecycle states persist to both KV and ClickHouse (established in S405-S411).
2. **Endurance stability** -- Sustained operation does not degrade persistence or state consistency (established in S412).
3. **Read-path consolidation** -- Operational queryability covers all lifecycle states with practical list/detail surfaces (established in S413).
4. **Rejection audit trail** -- Rejection code, reason, and venue details are queryable from both KV and ClickHouse (established in S407/S411/S413).
5. **Propagation consistency** -- `DeriveEffectivePropagation()` is the single source of truth across all read surfaces (validated in S413 tests).

The wave is ready for the evidence gate.
