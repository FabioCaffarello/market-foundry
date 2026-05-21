# Stage S64 — Risk First Slice: Position Exposure

| Field | Value |
|-------|-------|
| Stage | S64 |
| Title | Risk First Slice — Position Exposure |
| Status | Complete |
| Date | 2026-03-18 |

## Executive Summary

Implemented the first vertical slice of the risk domain with the Position Exposure family (RF-01). Risk is the sixth pipeline layer (observation, evidence, signal, decision, strategy, risk). The slice proves the domain with minimum scope: a single stateless, rule-based evaluator that assesses strategy outputs and emits disposition events (approved, modified, rejected).

## Prerequisites Validated

| Stage | Gate | Status |
|-------|------|--------|
| S60 | BG-1 LOW — adapter trust recovery | Pass |
| S61 | BG-2 LOW — derive actor confidence | Pass |
| S62 | Risk domain design complete | Pass |
| S63 | Risk governance active in raccoon-cli | Pass |

## Family Chosen

**Position Exposure** (`position_exposure`) — stateless, rule-based, no external dependencies. Follows the same first-family pattern used by signal (RSI), decision (RSI Oversold), and strategy (Mean Reversion Entry).

## Implementation Layers

### Domain

- `internal/domain/risk/risk.go` — PositionExposure value object, StrategyInput, dispositions, evaluation logic
- `internal/domain/risk/risk_test.go` — 12 tests covering all disposition paths and edge cases
- `internal/domain/risk/events.go` — PositionExposureAssessed event type

### Application

- `internal/application/risk/position_exposure_evaluator.go` — use case orchestrating domain evaluation
- `internal/application/risk/position_exposure_evaluator_test.go` — 10 tests
- `internal/application/riskclient/contracts.go` — client port interface
- `internal/application/riskclient/get_latest_risk.go` — query use case
- `internal/application/riskclient/get_latest_risk_test.go` — 5 tests
- `internal/application/ports/risk.go` — risk adapter port definitions

### Adapters (NATS)

- `internal/adapters/nats/risk_registry.go` — stream, consumer, KV bucket, query subject registration
- `internal/adapters/nats/risk_registry_test.go` — 6 tests
- `internal/adapters/nats/risk_publisher.go` — publishes to RISK_EVENTS stream
- `internal/adapters/nats/risk_consumer.go` — durable consumer for store
- `internal/adapters/nats/risk_gateway.go` — query request/reply gateway
- `internal/adapters/nats/risk_kv_store.go` — KV latest projection store
- `internal/adapters/nats/risk_kv_store_test.go` — roundtrip tests

### Actors — Derive

- `internal/actors/scopes/derive/risk_evaluator_actor.go` — PositionExposureEvaluatorActor
- `internal/actors/scopes/derive/risk_evaluator_actor_test.go` — 4 tests
- `internal/actors/scopes/derive/risk_publisher_actor.go` — RiskPublisherActor

### Actors — Store

- `internal/actors/scopes/store/risk_consumer_actor.go` — RiskConsumerActor
- `internal/actors/scopes/store/risk_projection_actor.go` — RiskProjectionActor
- `internal/actors/scopes/store/risk_projection_actor_test.go` — 11 tests

### HTTP Interface

- `internal/interfaces/http/handlers/risk.go` — handler for risk queries
- `internal/interfaces/http/handlers/risk_test.go` — 4 tests
- `internal/interfaces/http/routes/risk.go` — route registration
- `internal/interfaces/http/routes/risk_test.go` — 3 tests

### Test Fixtures

- `tests/http/risk.http` — manual HTTP test file

### Architecture Documentation

- `docs/architecture/risk-first-slice.md` — this slice architecture
- `docs/architecture/risk-family-01-contracts.md` — Position Exposure contracts

## Modified Files

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/messages.go` | Added `strategyResolvedMessage`, `publishRiskMessage` |
| `internal/actors/scopes/derive/source_scope_actor.go` | Added RiskFamilyProcessor, risk evaluator maps, spawning, routing |
| `internal/actors/scopes/derive/strategy_resolver_actor.go` | Added `ScopePID`, fan-out to risk evaluators |
| `internal/actors/scopes/derive/derive_supervisor.go` | Added risk registry, processors, config passing |
| `internal/actors/scopes/store/messages.go` | Added `riskReceivedMessage` |
| `internal/actors/scopes/store/projection_store.go` | Added `riskProjectionStore` interface |
| `internal/actors/scopes/store/store_supervisor.go` | Added RiskPipeline, risk pipeline spawning |
| `internal/actors/scopes/store/query_responder_actor.go` | Added risk KV store, query route, handler |
| `internal/interfaces/http/routes/core.go` | Added `RiskFamilyDeps`, risk route registration |
| `internal/shared/settings/schema.go` | Added risk families, validation, dependency rules |
| `cmd/gateway/gateway.go` | Added `newRiskGateway` |
| `cmd/gateway/run.go` | Added risk gateway wiring |
| `cmd/store/run.go` | Added risk trackers |
| `deploy/configs/derive.jsonc` | Added `risk_families` |
| `deploy/configs/store.jsonc` | Added `risk_families` |

## Test Results

All tests pass.

| Layer | Tests |
|-------|-------|
| Domain | 12 |
| Application | 10 |
| Risk client | 5 |
| Adapters | 6 |
| Derive actors | 4 |
| Store projection | 11 |
| HTTP handlers | 4 |
| Routes | 3 |
| Settings | Existing tests pass |

## Limits Encountered

- Strategy resolver actor needed `ScopePID` field for risk fan-out — minimal, backward-compatible change
- `StrategyFamilyProcessor` signature changed from 4 to 5 parameters (`scopePID` added)

## Deferred to S65+

1. Risk history projections (`RISK_POSITION_EXPOSURE_HISTORY` bucket)
2. Multi-strategy risk evaluation (single `StrategyInput` only in S64)
3. Drawdown Guard family (RF-02) — requires execution/portfolio state
4. Correlation Limit family (RF-03) — requires multi-symbol portfolio
5. Volatility Scaler family (RF-04) — requires volatility evidence
6. ClickHouse risk analytics
7. Separate risk binary extraction
8. Risk evaluator purity enforcement in raccoon-cli
9. Activate prepared drift checks in raccoon-cli (requires approximately 15 min in S65)
10. Portfolio-level exposure aggregation

## Confidence Assessment

| Dimension | Score |
|-----------|-------|
| Domain boundary clarity | 10/10 |
| Pattern consistency | 10/10 |
| First family feasibility | 10/10 (proven) |
| Test coverage | 9/10 |
| Activation model | 10/10 |
| **Overall** | **9.5/10** |
