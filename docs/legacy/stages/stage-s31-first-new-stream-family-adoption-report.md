# S31 — First New Stream Family Adoption

**Stage:** S31
**Type:** Implementation + Validation
**Status:** Complete
**Date:** 2025-03-17
**Depends on:** S28 (Derive FamilyProcessor), S29 (Store ProjectionPipeline), S30 (Gateway EvidenceFamilyDeps)

## Objective

Implement the first new evidence stream family end-to-end using the canonical mesh patterns established in S26-S30, proving the patterns are reusable and the mesh supports new families without architectural distortion.

## Family Chosen: evidence.volume

Volume profile per window providing VWAP, buy/sell notional volume distribution, and total volume. Chosen for maximum simplicity, utility, and pipeline fit. See [stream-family-01-adoption.md](../architecture/stream-family-01-adoption.md) for detailed justification.

## Files Created (11 new files)

| File | Layer | Purpose |
|------|-------|---------|
| `internal/domain/evidence/volume.go` | Domain | EvidenceVolume type with validation |
| `internal/domain/evidence/volume_test.go` | Domain | 11 validation test cases |
| `internal/application/derive/volume_sampler.go` | Application | VolumeSampler pure logic (big.Float arithmetic, VWAP) |
| `internal/application/derive/volume_sampler_test.go` | Application | 4 test cases (first window, rollover, mixed prices VWAP, alignment) |
| `internal/actors/scopes/derive/volume_sampler_actor.go` | Actor | VolumeSamplerActor (receive trades, finalize, publish) |
| `internal/adapters/nats/volume_kv_store.go` | Adapter | VolumeKVStore (VOLUME_LATEST, 64 MB, monotonicity guard) |
| `internal/adapters/nats/volume_consumer.go` | Adapter | VolumeConsumer (durable: store-volume) |
| `internal/actors/scopes/store/volume_projection_actor.go` | Actor | VolumeProjectionActor (Final gate, Validate, monotonicity) |
| `internal/actors/scopes/store/volume_consumer_actor.go` | Actor | VolumeConsumerActor (JetStream → projection) |
| `internal/application/evidenceclient/get_latest_volume.go` | Application | GetLatestVolumeUseCase (input validation) |
| `internal/application/evidenceclient/get_latest_volume_test.go` | Application | 3 test cases (validation, returns volume, nil gateway) |

## Files Modified (16 additive changes)

| File | Change |
|------|--------|
| `internal/domain/evidence/events.go` | +`EventVolumeSampled` const, +`VolumeSampledEvent` struct |
| `internal/actors/scopes/derive/messages.go` | +`publishVolumeMessage` type |
| `internal/actors/scopes/derive/derive_supervisor.go` | +volume `FamilyProcessor` entry |
| `internal/actors/scopes/derive/publisher_actor.go` | +`publishVolumeMessage` case in Receive |
| `internal/adapters/nats/evidence_publisher.go` | +`PublishVolume()` method |
| `internal/adapters/nats/evidence_registry.go` | +`VolumeSampled`/`VolumeLatest` specs, +`StoreVolumeConsumer()` |
| `internal/actors/scopes/store/messages.go` | +`volumeReceivedMessage` type |
| `internal/actors/scopes/store/store_supervisor.go` | +volume `ProjectionPipeline` entry |
| `internal/actors/scopes/store/query_responder_actor.go` | +`volumeStore` field, +KV init, +volume query route, +handler |
| `internal/application/evidenceclient/contracts.go` | +`VolumeLatestQuery`/`VolumeLatestReply` |
| `internal/application/ports/evidence.go` | +`GetLatestVolume()` on EvidenceGateway |
| `internal/adapters/nats/evidence_gateway.go` | +`GetLatestVolume()` implementation |
| `internal/interfaces/http/handlers/evidence.go` | +`getLatestVolumeUseCase` interface, +constructor param, +handler method |
| `internal/interfaces/http/handlers/evidence_test.go` | Updated 14 existing test call sites for new constructor param |
| `internal/interfaces/http/routes/core.go` | +`GetLatestVolume` in `EvidenceFamilyDeps`, +use case interface |
| `internal/interfaces/http/routes/evidence.go` | +volume route block |
| `cmd/gateway/run.go` | +`getLatestVolumeUseCase` creation and wiring |
| `cmd/store/run.go` | +`volume-projection`/`volume-consumer` trackers |

## Test Results

All tests pass across all modified packages:

| Package | Result |
|---------|--------|
| `internal/domain/evidence` | OK (includes 11 new volume validation tests) |
| `internal/application/derive` | OK (includes 4 new volume sampler tests) |
| `internal/application/evidenceclient` | OK (includes 3 new volume use case tests) |
| `internal/interfaces/http/handlers` | OK (21 existing + updated call sites) |
| `internal/interfaces/http/routes` | OK (7 existing) |
| `internal/adapters/nats` | OK |

All three binaries compile: gateway, store, derive.

## Evidence That the Mesh Supported the Entry

### Pattern Reuse Score

| Pattern | Untouched? | What was added |
|---------|-----------|----------------|
| `FamilyProcessor` (derive S28) | **SourceScopeActor untouched** | 1 entry in processor list |
| `ProjectionPipeline` (store S29) | **Spawning loop untouched** | 1 entry in pipeline list |
| `EvidenceFamilyDeps` (gateway S30) | **DefaultRoutes untouched** | 1 field + 1 route block |
| Evidence derivation pattern | **ConsumerActor untouched** | Publisher gains 1 case |
| Evidence query pattern | **Query framework untouched** | Responder gains 1 store + 1 route |

### What Was NOT Changed

- SourceScopeActor (derive) — spawns volume samplers automatically via processor loop
- StoreSupervisor spawning loop — spawns volume pipeline automatically via pipeline loop
- DefaultRoutes — includes volume automatically via `HasAny()`
- ConsumerActor (derive) — trades still flow to supervisor, volume samplers receive via fan-out
- BindingWatcherActor — activation triggers all families equally
- Health server — picks up volume trackers from the tracker map
- Readiness checker — unchanged (non-blocking probe)

### Structural Metrics

| Metric | Before S31 | After S31 | Change |
|--------|-----------|-----------|--------|
| Evidence types | 2 (candle, tradeburst) | 3 (+volume) | +1 |
| JetStream streams | 3 | 3 (volume shares EVIDENCE_EVENTS) | 0 |
| Durable consumers | 5 | 6 (+store-volume) | +1 |
| KV buckets | 3 | 4 (+VOLUME_LATEST) | +1 |
| Query subjects | 3 | 4 (+evidence.query.volume.latest) | +1 |
| HTTP endpoints | 3 | 4 (+/evidence/volume/latest) | +1 |
| New files | — | 11 | — |
| Modified files | — | 16 (all additive) | — |
| Supervisor code changed | — | 0 spawning loops changed | — |

## Limitations

1. **No volume history** — latest-only, same as tradeburst. Can be added following the candle history pattern.
2. **No VWAP bands** — standard deviation around VWAP requires rolling stats, deferred to evidence.stats.
3. **Shared EvidencePublisherActor** — all three evidence types share one publisher per source scope. At very high message rates, this could become a bottleneck. Not a concern at current scale.
4. **QueryResponderActor grows linearly** — now 4 query routes and 3 KV stores. Still manageable; splitting threshold is ~10 types.

## Recommendations for S32

### R1 — Update actor-ownership.md (5th consecutive stage, BLOCKING)

This is now critical. The canonical ownership document does not reflect any changes since S12. It must be updated to capture:
- All 3 evidence types in derive and store
- FamilyProcessor and ProjectionPipeline patterns
- Updated cross-binary and control plane matrices
- Volume-specific actors and KV buckets

### R2 — Add smoke test for volume pipeline

A smoke test similar to `scripts/smoke-first-slice.sh` that validates: ingest → derive → store → gateway for volume events.

### R3 — Consider evidence.stats as next family

With volume proving the pattern, stats (volatility, spread, tick frequency) would be the next natural evidence type. It follows the same pipeline and would provide the distributional metrics that volume and candle cannot.

### R4 — Evaluate EVIDENCE_EVENTS stream capacity

Three evidence types now share a single 2 GB / 72h stream. At moderate symbol counts (~10) with 60s timeframes, this is well within budget. At 50+ symbols with multiple timeframes, the stream size should be reviewed.

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| New family enters without architectural distortion | Met — 0 spawning loops changed, patterns reused exactly |
| Pattern by family proves reusable | Met — derive, store, gateway all extended via their respective patterns |
| Derive/store/gateway remain clear | Met — each binary gains one declarative entry + supporting files |
| Mesh is richer and closer to MM strengths | Met — 3 evidence types covering price, activity, and volume |
| System stays below signal | Met — no signal-related changes |
| No generic indicator engine | Met — VolumeSampler is concrete, type-safe, no abstractions |
| Evidence not mixed with decision | Met — pure evidence domain |
| Limitations documented | Met — 4 intentional limitations listed |
