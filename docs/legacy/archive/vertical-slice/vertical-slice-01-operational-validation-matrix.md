# Vertical Slice 01 — Operational Validation Matrix

> Stage S110 — Operational validation of the `candle-to-paper-order` vertical slice.
> Validated: 2026-03-19

---

## Validation Method

This matrix records the outcome of each validation step applied to the vertical slice.
All checks were executed against the current codebase state (post-S109).

**Validation tiers used:**

| Tier | Method | Description |
|------|--------|-------------|
| T1 | Compilation | `go build ./...` per module, `cargo check` for raccoon-cli |
| T2 | Unit tests | `go test -race ./...` per module |
| T3 | Static analysis | `go vet ./...` per module |
| T4 | Compose validation | `docker compose config --quiet` |
| T5 | Structural audit | Code review of wiring, contracts, naming, duplication |

---

## 1. Build & Compilation (T1)

| Module | Result | Notes |
|--------|--------|-------|
| cmd/configctl | PASS | Builds cleanly |
| cmd/derive | PASS | Builds cleanly |
| cmd/execute | PASS | Builds cleanly |
| cmd/gateway | PASS | Builds cleanly |
| cmd/ingest | PASS | Builds cleanly |
| cmd/store | PASS | Builds cleanly |
| internal/actors | PASS | Builds cleanly |
| internal/adapters/exchanges | PASS | Builds cleanly |
| internal/adapters/nats | PASS | Builds cleanly |
| internal/adapters/repositories | PASS | Builds cleanly |
| internal/application | PASS | Builds cleanly |
| internal/domain | PASS | Builds cleanly |
| internal/interfaces/http | PASS | Builds cleanly |
| internal/shared | PASS | Builds cleanly |
| tools/raccoon-cli (cargo check) | PASS | 26 warnings (dead code from removed commands) |
| tools/raccoon-cli (cargo test) | **FAIL** | 12 compile errors — tests reference removed `TracePack`, `ResultsInspect`, `ScenarioSmoke` variants |

---

## 2. Unit Tests with Race Detector (T2)

| Module | Result | Test Count | Duration |
|--------|--------|------------|----------|
| cmd/gateway | PASS | >4 tests | 1.3s |
| internal/actors/common | PASS | tests present | 1.2s |
| internal/actors/scopes/derive | PASS | tests present | 4.3s |
| internal/actors/scopes/store | PASS | tests present | 1.2s |
| internal/adapters/exchanges/binancef | PASS | tests present | 1.2s |
| internal/adapters/nats | PASS | tests present | 1.2s |
| internal/adapters/repositories/memory/configctl | PASS | tests present | 1.2s |
| internal/application/ingest | PASS | tests present | 2.2s |
| internal/application/risk | PASS | tests present | 2.2s |
| internal/application/riskclient | PASS | tests present | 2.4s |
| internal/application/runtimecontracts | PASS | tests present | 1.4s |
| internal/application/signal | PASS | tests present | 1.4s |
| internal/application/signalclient | PASS | tests present | 1.4s |
| internal/application/strategy | PASS | tests present | 1.4s |
| internal/application/strategyclient | PASS | tests present | 1.4s |
| internal/domain/configctl | PASS | tests present | 1.4s |
| internal/domain/decision | PASS | tests present | 1.2s |
| internal/domain/evidence | PASS | tests present | 1.5s |
| internal/domain/execution | PASS | tests present | 1.7s |
| internal/domain/observation | PASS | tests present | 2.0s |
| internal/domain/risk | PASS | tests present | 2.1s |
| internal/domain/signal | PASS | tests present | 1.8s |
| internal/domain/strategy | PASS | tests present | 2.3s |
| internal/interfaces/http/handlers | PASS | tests present | 1.3s |
| internal/interfaces/http/routes | PASS | tests present | 1.5s |
| internal/interfaces/http/webserver | PASS | tests present | 1.6s |
| internal/shared/bootstrap | PASS | tests present | 1.3s |
| internal/shared/envelope | PASS | tests present | 1.2s |
| internal/shared/events | PASS | tests present | 1.5s |
| internal/shared/healthz | PASS | tests present | 1.6s |
| internal/shared/memdb | PASS | tests present | 1.9s |
| internal/shared/problem | PASS | tests present | 2.1s |
| internal/shared/settings | PASS | tests present | 1.8s |

**Race conditions:** None detected across any module.

---

## 3. Static Analysis (T3)

| Module | Result | Notes |
|--------|--------|-------|
| All 14 Go workspace modules | PASS | No issues detected by `go vet` |

---

## 4. Compose Validation (T4)

| Check | Result | Notes |
|-------|--------|-------|
| docker-compose.yaml syntax | PASS | YAML is valid |
| docker-compose config | **FAIL** | ClickHouse healthcheck requires `CLICKHOUSE_PASSWORD` env var; `local.env` file exists but compose interpolation still fails without env export |
| Service dependency graph | PASS | DAG is acyclic: nats → configctl → {ingest, derive} → {store, execute} → gateway |
| Healthcheck port alignment | PASS | Ports match per-service config (fixed in S109) |
| Config volume mounts | PASS | All jsonc files exist |
| Dockerfile reference | PASS | `deploy/docker/go-service.Dockerfile` exists |
| NATS config reference | PASS | `deploy/nats/nats-server.conf` exists |

---

## 5. Modules Without Test Files

| Module | LOC | Risk | Rationale |
|--------|-----|------|-----------|
| cmd/configctl | 41 | LOW | Pure composition wiring |
| cmd/derive | 58 | LOW | Pure composition wiring |
| cmd/execute | 94 | MEDIUM | `buildVenueAdapter` has credential and venue selection logic |
| cmd/ingest | 56 | LOW | Pure composition wiring |
| cmd/store | 79 | MEDIUM | `buildTrackers` has configuration validation logic |
| internal/actors/scopes/configctl | 612 | MEDIUM-HIGH | Control routing dispatch, request/reply error mapping, lazy init |
| internal/actors/scopes/execute | 350 | HIGH | Kill switch gate, staleness guard, timeout — safety-critical |
| internal/actors/scopes/gateway | 57 | LOW | Lifecycle wrapper only |
| internal/actors/scopes/ingest | 611 | MEDIUM-HIGH | Dynamic exchange scope creation, binding state machine |
| internal/application/executionclient | 198 | MEDIUM | Query validation, status derivation logic |
| internal/application/ports | 152 | LOW | Interface definitions only |
| internal/shared/requestctx | 21 | LOW | Simple context utility |

---

## 6. Success Criteria Readiness (from S108)

| SC | Description | Status | Evidence |
|----|-------------|--------|----------|
| SC-1 | Config-driven pipeline activation | ARCHITECTURALLY READY | Config lifecycle tested: draft → validate → compile → activate; `usecases_test.go` covers full flow |
| SC-2 | Dynamic binding propagation | ARCHITECTURALLY READY | BindingWatcherActor wired; `IngestionRuntimeChangedEvent` published on activation |
| SC-3 | Observation capture | ARCHITECTURALLY READY | Ingest supervisor, exchange scope, WebSocket adapter all wired |
| SC-4 | Full derive pipeline | ARCHITECTURALLY READY | 6 publisher actors + source scope routing tested; `derive_supervisor_test` passes |
| SC-5 | Execution fill processing | ARCHITECTURALLY READY | Execute supervisor + venue adapter actor wired; **no unit tests for kill switch/staleness** |
| SC-6 | Read model materialization | ARCHITECTURALLY READY | 7 projection actors + consumer actors wired; store tests pass |
| SC-7 | Query surface completeness | ARCHITECTURALLY READY | Gateway compose wires all 8+ query routes; gateway readiness test passes |
| SC-8 | Diagnostic visibility | ARCHITECTURALLY READY | `/healthz`, `/readyz`, `/statusz`, `/diagz` wired; health server tests pass |
| SC-9 | Graceful lifecycle | ARCHITECTURALLY READY | `WaitTillShutdown` handles SIGTERM; 10s poison timeout per PID |
| SC-10 | Envelope integrity | ARCHITECTURALLY READY | Envelope tests validate ID, Kind, Type, Source, timestamps, validation issues |

**Note:** All SC items are "architecturally ready" — they require `docker compose up` with live NATS to confirm runtime behavior. This is expected and consistent with the S109 findings.

---

## 7. Raccoon-CLI Guardian Status

| Check | Result | Notes |
|-------|--------|-------|
| cargo check | PASS | Binary compiles; 26 dead-code warnings |
| cargo test | **FAIL** | 12 errors: unit tests in `main.rs` reference removed enum variants (`TracePack`, `ResultsInspect`, `ScenarioSmoke`) |
| Smoke scenarios | NOT RUN | Requires live infrastructure |
| Dead code | 4 functions unused in `smoke/scenarios.rs` (`run_readiness_probe`, `run_stages_sequential`, `skip_remaining`, `run_missing_binding`) |

---

## Summary

| Tier | Total Checks | Pass | Fail | Notes |
|------|-------------|------|------|-------|
| T1 Build | 16 | 15 | 1 | raccoon-cli test compilation fails |
| T2 Tests | 33 | 33 | 0 | All Go tests pass with race detector |
| T3 Static | 14 | 14 | 0 | No vet issues |
| T4 Compose | 7 | 6 | 1 | ClickHouse env interpolation |
| T5 Structural | — | — | — | See friction findings document |
