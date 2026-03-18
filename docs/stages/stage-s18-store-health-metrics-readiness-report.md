# Stage S18 — Store Health, Metrics & Readiness Hardening

**Status:** Complete
**Scope:** Operational hardening of health/readiness/telemetry for store and first-slice pipeline components.

## Objective

Harden the runtime operational visibility of the `store` service and critical first-slice components (`ingest`, `derive`) with health, readiness, activity tracking, and idle detection — without introducing framework overhead or decorative metrics.

## What Changed

### 1. Shared Health Infrastructure (`internal/shared/healthz/`)

New package providing three reusable primitives:

- **`Tracker`** — Thread-safe activity tracker using atomic counters. Records event count and last-event timestamp. Used by actors to report processing activity without coupling to any metrics framework.
- **`HealthServer`** — Lightweight HTTP server exposing three endpoints:
  - `GET /healthz` — Liveness probe (always 200).
  - `GET /readyz` — Readiness probe with pluggable checks (returns 503 with failing check name on failure).
  - `GET /statusz` — Activity status with per-tracker event counts, last-event time, idle duration, and idle warnings.
- **Heartbeat monitor** — Background goroutine that logs warnings when tracked components go idle beyond a configurable threshold (default: 2 minutes). Detects stalled pipeline without polling.

### 2. Store Actors — Activity Tracking

- **`StoreSupervisor`** now accepts `projectionTracker` and `consumerTracker` parameters and threads them to child actors.
- **`CandleProjectionActor`** records an event via tracker each time a candle is materialized to NATS KV.
- **`EvidenceConsumerActor`** records an event each time an evidence event is consumed from JetStream.

### 3. Ingest Actors — Activity Tracking

- **`IngestSupervisor`** accepts a `publisherTracker` and passes it through `ExchangeScopeActor` to `PublisherActor`.
- **`PublisherActor`** records an event on each successful trade publication.

### 4. Derive Actors — Activity Tracking

- **`DeriveSupervisor`** accepts a `publisherTracker` and passes it through `SourceScopeActor` to `EvidencePublisherActor`.
- **`EvidencePublisherActor`** records an event on each successful candle publication.

### 5. Service Entrypoints — Health Servers

- **`cmd/store/run.go`** — Starts health server on configured HTTP addr (`:8081`). Readiness checks NATS connectivity via TCP dial. Exposes evidence-consumer and candle-projection trackers.
- **`cmd/ingest/run.go`** — Starts health server on `:8082`. Readiness checks NATS. Exposes observation-publisher tracker.
- **`cmd/derive/run.go`** — Starts health server on `:8083`. Readiness checks NATS. Exposes evidence-publisher tracker.

### 6. Docker Compose — Real Healthchecks

Replaced process-existence healthchecks (`grep cmdline`) with actual HTTP readiness probes for all three headless services:

| Service  | Before                                  | After                                              |
|----------|-----------------------------------------|----------------------------------------------------|
| store    | `grep -qa '/usr/local/bin/service' ...` | `wget -q -O - http://127.0.0.1:8081/readyz \| grep -q 'ready'` |
| ingest   | `grep -qa '/usr/local/bin/service' ...` | `wget -q -O - http://127.0.0.1:8082/readyz \| grep -q 'ready'` |
| derive   | `grep -qa '/usr/local/bin/service' ...` | `wget -q -O - http://127.0.0.1:8083/readyz \| grep -q 'ready'` |

### 7. Configuration Updates

Added `http.addr` to deployment configs for headless services:
- `store.jsonc` → `:8081`
- `ingest.jsonc` → `:8082`
- `derive.jsonc` → `:8083`

## Files Changed

| File | Change |
|------|--------|
| `internal/shared/healthz/healthz.go` | New — Tracker, HealthServer, heartbeat monitor |
| `internal/shared/healthz/healthz_test.go` | New — Unit tests for tracker and health endpoints |
| `internal/actors/scopes/store/store_supervisor.go` | Accept and thread activity trackers |
| `internal/actors/scopes/store/candle_projection_actor.go` | Record projection activity |
| `internal/actors/scopes/store/evidence_consumer_actor.go` | Record consumer activity |
| `internal/actors/scopes/ingest/ingest_supervisor.go` | Accept and thread publisher tracker |
| `internal/actors/scopes/ingest/exchange_scope_actor.go` | Pass tracker to publisher |
| `internal/actors/scopes/ingest/publisher_actor.go` | Record publish activity |
| `internal/actors/scopes/derive/derive_supervisor.go` | Accept and thread publisher tracker |
| `internal/actors/scopes/derive/source_scope_actor.go` | Pass tracker to publisher |
| `internal/actors/scopes/derive/publisher_actor.go` | Record publish activity |
| `cmd/store/run.go` | Start health server, wire trackers and readiness checks |
| `cmd/ingest/run.go` | Start health server, wire tracker and readiness check |
| `cmd/derive/run.go` | Start health server, wire tracker and readiness check |
| `deploy/configs/store.jsonc` | Add `http.addr: ":8081"` |
| `deploy/configs/ingest.jsonc` | Add `http.addr: ":8082"` |
| `deploy/configs/derive.jsonc` | Add `http.addr: ":8083"` |
| `deploy/compose/docker-compose.yaml` | Real HTTP healthchecks for store/ingest/derive |

## Signals & Checks Added

| Signal | Source | What it detects |
|--------|--------|-----------------|
| `/readyz` (store, ingest, derive) | Health server | Service up + NATS reachable |
| `/healthz` (store, ingest, derive) | Health server | Process alive |
| `/statusz` (store, ingest, derive) | Health server | Last event time, event count, idle warnings |
| Heartbeat log warning | Background monitor | Pipeline stalled (no events for >2 min) |
| `candle-projection` tracker | CandleProjectionActor | Candle materialization activity |
| `evidence-consumer` tracker | EvidenceConsumerActor | Evidence event consumption activity |
| `observation-publisher` tracker | PublisherActor (ingest) | Trade publication activity |
| `evidence-publisher` tracker | EvidencePublisherActor (derive) | Candle publication activity |

## Remaining Gaps

1. **Per-symbol/per-timeframe granularity** — Current trackers are per-component aggregate. Per-symbol breakdown would help isolate which symbol stalled but adds complexity beyond current need.
2. **Prometheus/OpenTelemetry metrics** — Structured logs + `/statusz` are sufficient for the current scale. Prometheus export can be added when the deployment needs external monitoring integration.
3. **WebSocket connection health** — Ingest WebSocket adapters log disconnect/reconnect but don't expose connection status in `/statusz`. Adding this would require tracking across dynamically spawned actors.
4. **NATS JetStream consumer lag** — Consumer lag (pending messages) is available from NATS but not yet exposed. Would help detect slow consumers.
5. **Configctl readiness integration** — Ingest and derive depend on configctl for binding discovery but don't include it in readiness checks (only NATS is checked). This is acceptable because binding watcher handles gateway absence gracefully.

## S19 Preparation

With operational visibility in place, the system is ready for:

- **Historical candle backfill** — Store can now report its projection activity, making it possible to verify backfill progress.
- **Multi-exchange support** — Per-exchange activity tracking via existing tracker pattern.
- **Alerting integration** — `/statusz` JSON output is suitable for external monitoring scrapers.
- **Consumer lag monitoring** — Expose JetStream pending message counts for derive and store consumers.
