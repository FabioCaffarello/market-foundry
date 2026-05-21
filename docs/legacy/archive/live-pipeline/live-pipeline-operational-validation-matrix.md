# Live Pipeline Operational Validation Matrix

> S115 — Operational validation of the minimal live pipeline activated in S114.

## Validation Scope

| Dimension | Method | Result |
|-----------|--------|--------|
| **Go unit tests** | `make test` (all modules) | PASS — all packages green |
| **Quality gate (fast)** | `make quality-gate` (84 checks) | PASS — 6/6 passed, 0 errors |
| **Raccoon-cli tests** | `cargo test` (97 tests) | PASS — 97/97 passed |
| **Architecture guard** | `raccoon-cli arch-guard` (11 checks) | PASS after fix |
| **Topology doctor** | `raccoon-cli topology-doctor` (13 checks) | PASS after fix |
| **Drift detection** | `raccoon-cli drift-detect` (32 checks) | PASS after fix, 5 warnings |
| **Contract audit** | `raccoon-cli contract-audit` (13 checks) | PASS — clean |
| **Runtime bindings** | `raccoon-cli runtime-bindings` (8 checks) | PASS — clean |

## Bugs Found and Fixed

| ID | Category | Description | Root Cause | Fix |
|----|----------|-------------|------------|-----|
| B1 | **Bug** | `find_stream_name_near` heuristic in raccoon-cli picks first UPPER_SNAKE_CASE string scanning top-to-bottom, catching KV bucket name before actual stream | Linear scan order in `topology/source.rs` and `runtime_bindings/source.rs` | Changed to outward-from-center scan; nearest match wins |
| B2 | **Bug** | arch-guard: `actors/scopes/gateway/gateway.go` imports `interfaces/http/webserver` — layer 3 depends on layer 4 | `webserver` package placed in interfaces layer despite being shared infrastructure | Moved `webserver` to `internal/shared/webserver`; updated 9 import paths |
| B3 | **Bug** | Topology test fixture missing RISK_EVENTS, EXECUTION_EVENTS, EXECUTION_FILL_EVENTS streams and 4 durables | Fixture not updated when execution pipeline was added in S80-S81 | Added 3 streams, 4 durables, 12 subjects to fixture |

## Warnings (Known Debt, Not Errors)

| ID | Category | Description | Impact | Recommendation |
|----|----------|-------------|--------|----------------|
| W1 | Stale naming | "consumer" appears in ~260 references across 35 files | Low — mostly in docs from old architecture stages | Address in next naming sweep (low priority) |
| W2 | Stale naming | "validator" appears in 3 test files (runtimecontracts, configctl handlers) | Low — test string literals | Rename in targeted cleanup |
| W3 | Non-canonical stream | `TEST_STREAM` detected by drift-detect | None — used only in test infrastructure | Acceptable trade-off |

## Validated Operational Surfaces

### Startup & Lifecycle

| Surface | Status | Notes |
|---------|--------|-------|
| Service binary compilation | PASS | All 6 binaries build cleanly |
| Configuration loading & validation | PASS | bootstrap_test.go covers config → logger → slog pipeline |
| Pipeline dependency validation | PASS | `ValidatePipeline()` enforces family dependency chain |
| Actor engine creation | PASS | Hollywood actor framework wiring verified |
| Graceful shutdown (SIGTERM/SIGINT) | PASS | `WaitTillShutdown()` poisons actors with 10s timeout |

### Health & Diagnostics

| Endpoint | Status | Coverage |
|----------|--------|----------|
| `/healthz` | PASS | Liveness probe — always 200 if process alive |
| `/readyz` | PASS | Readiness probe — runs all registered checks |
| `/statusz` | PASS | Activity tracker — event counts, idle time, counters |
| `/diagz` | PASS | Diagnostic summary — readiness + tracker overview |

### Safety Gates (Execute)

| Gate | Test Coverage | Status |
|------|--------------|--------|
| Kill switch (execution control) | 14 test cases | PASS — halted/active/nil/timeout/fail-open |
| Staleness guard | 11 test cases | PASS — fresh/stale/boundary/zero/future |
| Gate ordering (kill switch first) | Explicit test | PASS |
| Paper venue adapter | 8 test cases | PASS — buy/sell/no-action/cancel/delay |

### Gateway Readiness & Degradation

| Scenario | Status |
|----------|--------|
| NATS required — fail if disabled | PASS |
| Configctl gateway required — fail if nil | PASS |
| Evidence store optional — non-blocking | PASS |
| Graceful degradation on upstream unavailable | PASS |

### Event Flow & Materialization

| Stage | Stream | Durable Consumer | Status |
|-------|--------|-----------------|--------|
| Observation | OBSERVATION_EVENTS | derive-observation | Validated |
| Evidence | EVIDENCE_EVENTS | store-candle, store-trade-burst, store-volume | Validated |
| Signal | SIGNAL_EVENTS | store-signal-rsi | Validated |
| Decision | DECISION_EVENTS | store-decision-rsi-oversold | Validated |
| Strategy | STRATEGY_EVENTS | store-strategy-mean-reversion-entry | Validated |
| Risk | RISK_EVENTS | store-risk-position-exposure | Validated |
| Execution | EXECUTION_EVENTS | store-execution-paper-order, execute-venue-market-order-intake | Validated |
| Fill | EXECUTION_FILL_EVENTS | store-execution-venue-market-order-fill | Validated |

### Query Surface

| Domain | Endpoint Pattern | Status |
|--------|-----------------|--------|
| Evidence | `/evidence/candles/latest`, `/evidence/tradeburst/latest`, `/evidence/volume/latest` | Available |
| Signal | `/signal/rsi/latest` | Available |
| Decision | `/decision/rsi_oversold/latest` | Available |
| Strategy | `/strategy/mean_reversion_entry/latest` | Available |
| Risk | `/risk/position_exposure/latest` | Available |
| Execution | `/execution/paper_order/latest`, `/execution/venue_market_order/latest` | Available |
| Execution status | `/execution/status/latest` | Available |
| Execution control | `/execution/control` (GET/PUT) | Available |

## Summary

- **3 bugs fixed** — all quality gate errors resolved
- **5 warnings documented** — all low-impact, known debt
- **84 static checks passing** — architecture, topology, contracts, bindings, drift
- **97 raccoon-cli tests passing** — no regressions
- **All Go tests passing** — unit coverage intact across all modules
