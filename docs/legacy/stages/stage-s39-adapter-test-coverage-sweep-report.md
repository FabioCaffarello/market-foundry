# Stage S39 — Adapter Test Coverage Sweep

**Status:** Complete
**Date:** 2025-07-24
**Objective:** Increase structural confidence in critical adapters and read-side components before opening the `decision` domain.

## Executive Summary

S39 executed a disciplined test coverage sweep on the four adapter components identified by the S38 readiness review as coverage gaps: `signal_registry.go`, `signal_kv_store.go`, `trade_burst_kv_store.go`, and `volume_kv_store.go`. The sweep added **39 new test cases** covering contract invariants, nil-guard safety, key generation semantics, and constructor/lifecycle correctness. Two bugs were discovered and fixed in `volume_kv_store.go` during the process.

## Targets Covered

### 1. `signal_registry.go` → `signal_registry_test.go`

**Why:** The signal domain's NATS contracts had zero test coverage. Registry correctness is load-bearing for the entire signal query path.

**Tests added (17 cases):**
- Subject taxonomy validation (events + query subjects follow `signal.{events|query}.{type}.{action}`)
- Type versioning invariant (all types contain `.v1.`)
- Stream name, subjects wildcard, max age, max bytes
- Query queue group and reply type conventions
- `LatestSpecByType` — known type returns correct spec, unknown/empty return false
- `StoreRSISignalConsumer` — durable naming, hyphen convention, wildcard routing, stream binding, max deliver bounds, ack wait positivity

### 2. `signal_kv_store.go` → `signal_kv_store_test.go`

**Why:** Signal projection store had no tests. Nil-guard correctness is critical for safe process startup/shutdown sequencing.

**Tests added (7 cases):**
- Nil pointer guard on `Put` and `Get` (returns `problem.Unavailable`)
- Uninitialized store guard on `Put` and `Get` (constructed but not started)
- Constructor correctness (url, bucket assignment)
- Bucket constant value
- `Close` nil-safety (nil pointer and unstarted store)

### 3. `trade_burst_kv_store.go` → `trade_burst_kv_store_test.go`

**Why:** Trade burst KV store had no test coverage. Key generation and guard invariants needed validation.

**Tests added (10 cases):**
- Key generation: determinism, uniqueness across source/symbol/timeframe
- Nil pointer guard on `Put` and `Get`
- Uninitialized store guard on `Put` and `Get`
- Constructor correctness
- Bucket constant value
- `Close` nil-safety

### 4. `volume_kv_store.go` → `volume_kv_store_test.go`

**Why:** Volume KV store had no tests AND had two latent bugs discovered during analysis.

**Tests added (10 cases):**
- Key generation: determinism, uniqueness across source/symbol/timeframe
- Nil pointer guard on `Put` and `Get`
- Uninitialized store guard on `Put` and `Get`
- Constructor correctness
- Bucket constant value
- `Close` nil-safety

## Bugs Found and Fixed

### Bug 1: `VolumeKVStore.Put` — Missing nil guard (panic on uninitialized store)

Unlike `CandleKVStore`, `TradeBurstKVStore`, and `SignalKVStore`, `VolumeKVStore.Put` had no `s == nil || s.latest == nil` guard. Calling `Put` before `Start()` would panic with a nil pointer dereference.

**Fix:** Added nil guard returning `problem.Unavailable`, consistent with all other KV stores.

### Bug 2: `VolumeKVStore.Get` — Missing nil guard + silent error swallowing

`Get` had no nil guard (would panic) and returned `nil, nil` for ALL errors, not just `ErrKeyNotFound`. This silently swallowed real infrastructure errors (connection failures, timeouts), making them indistinguishable from "no data yet."

**Fix:** Added nil guard and `ErrKeyNotFound` distinction, consistent with `CandleKVStore.Get`, `TradeBurstKVStore.Get`, and `SignalKVStore.Get`.

## Files Changed

| File | Action |
|------|--------|
| `internal/adapters/nats/signal_registry_test.go` | **Created** — 17 test cases |
| `internal/adapters/nats/signal_kv_store_test.go` | **Created** — 7 test cases |
| `internal/adapters/nats/trade_burst_kv_store_test.go` | **Created** — 10 test cases |
| `internal/adapters/nats/volume_kv_store_test.go` | **Created** — 10 test cases |
| `internal/adapters/nats/volume_kv_store.go` | **Fixed** — added nil guards to Put/Get, ErrKeyNotFound handling in Get |
| `docs/stages/stage-s39-adapter-test-coverage-sweep-report.md` | **Created** — this report |

## Remaining Gaps

1. **Monotonicity guard integration tests** — The `Put` methods on all KV stores have a monotonicity guard (skip stale/duplicate writes). These guards are tested only through the nil-guard path; full round-trip monotonicity testing requires a running NATS server (integration test scope, not unit test scope).

2. **Signal publisher/consumer actors** — `signal_publisher.go` and `signal_consumer.go` have no test coverage. These are runtime actors that require NATS infrastructure to test meaningfully.

3. **Signal gateway** — `signal_gateway.go` (request/reply handler) has no unit tests. The HTTP handler layer (`signal_test.go` in handlers) covers the query path, but the NATS adapter layer is untested.

4. **Volume consumer actor** — `volume_consumer_actor.go` and `volume_projection_actor.go` are untested actors that depend on running infrastructure.

5. **`VolumeKVStore.Put` return values on error** — Returns `PutSkippedStale` (not `PutWritten`) on marshal and put errors. This is inconsistent with other KV stores but functionally harmless since the error is always checked first. Flagged but not changed to avoid scope creep.

## Impact on Readiness for S40/S41/S42

- **Contract confidence:** Signal registry contracts are now tested. Any future signal type addition (e.g., MACD) must follow the tested taxonomy invariants.
- **Projection authority:** All four KV stores now have verified nil-guard safety, meaning the store service cannot silently corrupt state during startup races.
- **Bug reduction:** The `VolumeKVStore` bugs were latent production risks. The silent error swallowing in `Get` could have caused the decision domain to treat infrastructure failures as "no data," leading to incorrect decision inputs.
- **Decision readiness:** The adapter layer that `decision` will consume (signal queries, evidence queries) is now structurally more trustworthy. The remaining gaps are integration-level concerns, not contract-level risks.
