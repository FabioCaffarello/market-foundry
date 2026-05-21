# Vertical Slice 01 — Implementation Notes

## Slice Identity

**Name:** `candle-to-paper-order`
**Binding:** `binancef.btcusdt.60`
**Stage:** S109 — End-to-End Implementation

---

## Implementation Summary

The vertical slice `candle-to-paper-order` exercises the complete event pipeline across all 6 runtimes, 8 domain families, 9 JetStream streams, 11 durable consumers, 8 KV buckets, and 25+ HTTP query endpoints.

The implementation revealed that the core pipeline code — actors, publishers, consumers, projections, registries, gateways, handlers, and routes — was already fully implemented from prior stages. S109 focused on fixing operational wiring issues that would have prevented the slice from actually running end-to-end in Docker Compose.

---

## Issues Found and Fixed

### 1. Docker Compose Healthcheck Port Mismatch (Critical)

**Problem:** All service healthchecks in `docker-compose.yaml` targeted port `8080`, but only `configctl` (default) and `gateway` listen on that port. The other services listen on distinct ports configured in their JSONC files.

| Service   | Config Port | Compose Healthcheck (before) | Compose Healthcheck (after) |
|-----------|-------------|------------------------------|-----------------------------|
| configctl | :8080       | :8080                        | :8080 (unchanged)           |
| gateway   | :8080       | :8080                        | :8080 (unchanged)           |
| ingest    | :8082       | :8080 (wrong)                | :8082                       |
| derive    | :8083       | :8080 (wrong)                | :8083                       |
| store     | :8081       | :8080 (wrong)                | :8081                       |
| execute   | :8084       | :8080 (wrong)                | :8084                       |

**Impact:** Without this fix, 4 of 6 services would never report healthy in Compose, causing cascading dependency failures (store depends on derive healthy, gateway depends on store healthy).

**Root cause:** Healthchecks were written when all services used the default `:8080` port. Configs were later updated with per-service ports but healthchecks were not synchronized.

### 2. Gateway Readiness Test Stub Incomplete

**Problem:** The `readinessEvidenceGatewayStub` in `cmd/gateway/readiness_test.go` only implemented `GetLatestCandle`, but the `EvidenceGateway` interface was expanded to include `GetCandleHistory`, `GetLatestTradeBurst`, and `GetLatestVolume` in prior stages.

**Fix:** Added the three missing methods to the test stub. All gateway tests now pass.

### 3. Configctl Missing Explicit HTTP Config

**Problem:** `configctl.jsonc` had no `http` block, relying on the default `:8080` from `settings.schema.go`. While functionally correct, this was implicit and inconsistent with all other service configs which declare their port explicitly.

**Fix:** Added `"http": {"addr": ":8080"}` to `configctl.jsonc` for explicitness and config parity.

### 4. Missing `local.env` for ClickHouse

**Problem:** `docker-compose.yaml` references `../envs/local.env` for ClickHouse credentials, but only `local.env.example` existed. Compose would fail on startup with a missing env file error.

**Fix:** Created `deploy/envs/local.env` from the example template.

---

## Simplifications Adopted

1. **No E2E test suite.** The slice is validated by runtime health, manual HTTP queries, and diagnostic endpoints — not by automated integration tests. This is explicitly out of scope per S108.

2. **Single binding only.** The slice activates exactly `binancef.btcusdt.60`. Multi-symbol, multi-timeframe, and multi-source validation are deferred.

3. **Paper simulator only.** The execute runtime uses `paper_simulator` venue type. No real exchange connectivity. The venue activation gate ceremony is deferred.

4. **No ClickHouse projections.** ClickHouse is present in Compose for infrastructure readiness but no domain data flows to it yet. All read models are NATS KV only.

5. **No auth or TLS.** All HTTP endpoints are unauthenticated. NATS connections are plaintext. Acceptable for local development validation.

6. **Default staleness and timeout values.** Execute uses `staleness_max_age: 120s` and `submit_timeout: 10s` — production tuning is deferred.

---

## Architectural Observations

### What Worked Well

- **Registry-driven wiring.** Every NATS subject, stream, consumer, and KV bucket is declared in registry structs. This made it trivial to verify completeness — the registries are the single source of truth.

- **Config-driven family activation.** The pipeline section in each JSONC config cleanly controls which families are active per runtime. No code changes needed to activate/deactivate families.

- **Conditional gateway composition.** The gateway's `compose.go` uses `connectOptional` for all non-configctl gateways, meaning any subset of domain gateways can be active without code changes.

- **Actor supervision trees.** The Hollywood actor framework provides clean lifecycle management. Each runtime's supervisor tree is self-contained and starts/stops cleanly.

### What Needs Attention

- **Healthcheck port discipline.** The port mismatch was a silent failure that would only manifest at runtime. Consider adding a CI check that validates compose healthcheck ports match config file ports.

- **Test stub maintenance.** Interface changes in ports automatically break test stubs, which is good (compile-time safety), but the stubs were not updated when the interface evolved. Consider generating test stubs from interfaces.

---

## File Changes

| File | Change |
|------|--------|
| `deploy/compose/docker-compose.yaml` | Fixed healthcheck ports for ingest(:8082), derive(:8083), store(:8081), execute(:8084) |
| `cmd/gateway/readiness_test.go` | Added `GetCandleHistory`, `GetLatestTradeBurst`, `GetLatestVolume` to test stub |
| `deploy/configs/configctl.jsonc` | Added explicit `http.addr` config |
| `deploy/envs/local.env` | Created from example template |
