# S24: Evidence Multi-Projection Support — Stage Report

## Summary

Hardened the store and gateway to cleanly support multiple evidence projections (candle + trade burst) with independent health tracking, shared infrastructure where appropriate, and documented guidelines for adding future types. No new evidence types introduced — this stage consolidates the multi-projection pattern.

## Changes Made

### 1. Independent Health Trackers per Projection

**Before:** One `projectionTracker` ("candle-projection") and one `consumerTracker` ("evidence-consumer") shared by all projection pipelines. `/statusz` showed a single tracker that mixed candle and trade-burst activity.

**After:** `ProjectionTrackers` struct holds four named trackers:
- `candle-projection` — candle materialization events
- `candle-consumer` — candle evidence consumer messages
- `trade-burst-projection` — trade burst materialization events
- `trade-burst-consumer` — trade burst consumer messages

Each surfaces independently on `/statusz`. Operators can now see which projection type is healthy, idle, or failing.

### 2. Shared Query Param Parser

**Before:** `GetLatestCandle`, `GetCandleHistory`, and `GetLatestTradeBurst` each duplicated identical source/symbol/timeframe parsing (18 lines × 3 handlers).

**After:** Extracted `parseEvidenceKeyParams(r) (evidenceKeyParams, *problem.Problem)` helper. All evidence handlers call it, reducing duplication and ensuring consistent validation.

### 3. Supervisor Pipeline Grouping

**Before:** Actor spawns in `StoreSupervisor.start()` were interleaved and hard to scan.

**After:** Grouped by pipeline with clear visual separation:
```
--- Candle projection pipeline ---
--- Trade burst projection pipeline ---
--- Query responder (serves all evidence queries) ---
```

Startup log now includes a `projections` field listing all active types and a `buckets` field listing all KV bucket names.

### 4. Architecture Documentation

Three new documents formalize the multi-projection model:

- **`multi-projection-pattern.md`** — defines what is shared (stream, responder, registry, queue group) vs isolated (consumer, projection, bucket, trackers) across projection pipelines. Documents anti-patterns avoided.

- **`evidence-read-model-guidelines.md`** — complete checklist for adding a new evidence type. 30+ items across 8 layers (NATS, store adapter, store actor, store binary, application, HTTP, gateway, smoke). Documents naming conventions and invariants.

- **`read-model-authority.md`** — updated with projection inventory table, trade burst entries, and per-tracker health documentation.

## Files Modified

| File | Change |
|------|--------|
| `cmd/store/run.go` | 4 independent health trackers via `ProjectionTrackers` struct |
| `internal/actors/scopes/store/store_supervisor.go` | `ProjectionTrackers` struct, pipeline-grouped spawning, inventory logging |
| `internal/interfaces/http/handlers/evidence.go` | Extracted `parseEvidenceKeyParams` helper |
| `docs/architecture/read-model-authority.md` | Projection inventory table, per-tracker health |
| `docs/architecture/multi-projection-pattern.md` | **New** |
| `docs/architecture/evidence-read-model-guidelines.md` | **New** |
| `docs/stages/stage-s24-evidence-multi-projection-support-report.md` | **New** |

## Ownership and Query Decisions

### Store owns all projections

Each evidence type's read model lives entirely in the store:
- Own durable consumer (independent stream position)
- Own projection actor (independent validation, stats, write logic)
- Own KV bucket(s) (independent retention, capacity, key format)
- Own health tracker pair (independent operational visibility)

### Query responder is the single read-side gateway

One `QueryResponderActor` serves all evidence queries. This is deliberate:
- Avoids N NATS subscription connections (one per type would be wasteful)
- All queries share the `evidence.query` queue group for horizontal scaling
- Each query route is typed (compile-time type safety via `NewTypedControlRoute`)

### Gateway maps 1:1 to query subjects

Each query subject maps to exactly one HTTP endpoint. No aggregation, no joins, no cross-type queries. The gateway is a thin translation layer.

## Limits Observed

1. **Query responder grows linearly** — each new evidence type adds ~10 lines to `QueryResponderActor` (open store, register route, add handler, close on stop). At 10+ types this could warrant splitting into per-type responders.

2. **Handler/Dependencies grow linearly** — each type adds a use case interface and constructor field. At 10+ types, grouping into an `EvidenceDeps` sub-struct would help. Not needed at 2 types.

3. **No runtime projection discovery** — no API to list active projections. The startup log is the only inventory. Sufficient while the type set is small.

4. **Shared queue group** — all evidence queries share `evidence.query`. High-volume types could starve low-volume types. Not a concern at current scale.

5. **No cross-type invariant** — no mechanism enforces that "every symbol with candles also has trade bursts". The derive layer spawns both, but there's no store-level check.

## S25 Preparation

1. **Trade burst history** — add `TRADE_BURST_HISTORY` bucket following the candle history pattern (S19). The guidelines doc now makes this straightforward.
2. **Configurable burst threshold** — parameterize the 2.0× burst ratio via configctl bindings.
3. **Projection lag metric** — track delta between stream head and last projected event per pipeline.
4. **Third evidence type evaluation** — with 2 types proven and guidelines documented, evaluate whether a third type (e.g., trade flow imbalance) warrants any abstraction.
5. **Multi-symbol activation** — wire BindingWatcherActor in store for dynamic projection lifecycle.
