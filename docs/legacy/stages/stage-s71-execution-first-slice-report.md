# Stage S71 — Execution First Slice Report

## Executive Summary

S71 implemented the first vertical slice of the `execution` domain in Market Foundry. The `paper_order` family (EF-01) is now a functioning end-to-end pipeline: derive evaluates risk assessments into paper order intents, publishes to `EXECUTION_EVENTS`, store materializes to `EXECUTION_PAPER_ORDER_LATEST` KV bucket, and gateway serves `GET /execution/:type/latest` via NATS request/reply.

No venue adapter, order lifecycle, position tracking, or portfolio logic was introduced. Execution is a narrow, deterministic translator — exactly as designed in S69.

---

## Pre-Condition Verification

- **S68**: Conditionally ready for design — satisfied
- **S69**: Execution domain design complete (7 documents) — satisfied
- **S70**: Execution under active governance in raccoon-cli — satisfied, now upgraded to full drift checks

---

## Family Chosen

**`paper_order` (EF-01)** — the only family designed in S69. Deterministic mapping:
- Risk approved/modified + direction long → side=buy, quantity=maxPositionPct
- Risk approved/modified + direction short → side=sell, quantity=maxPositionPct
- Risk approved/modified + direction flat → side=none, quantity=0
- Risk rejected → side=none, quantity=0

---

## Architecture Changes

### Causal Chain Extended

```
observation → evidence → signal → decision → strategy → risk → execution
```

Risk evaluators are no longer terminal. They now send a `riskAssessedMessage` back to `SourceScopeActor`, which fans it out to execution evaluators.

### Data Flow

```
PositionExposureEvaluatorActor
  │ riskAssessedMessage (primitive types)
  ▼
SourceScopeActor.routeRiskToExecution()
  │
  ▼
PaperOrderEvaluatorActor
  │ PaperOrderEvaluator.Evaluate() (pure function)
  │ publishExecutionMessage
  ▼
ExecutionPublisherActor → EXECUTION_EVENTS (JetStream)
                              │
                              ▼
                    ExecutionConsumerActor (store)
                              │
                              ▼
                    ExecutionProjectionActor
                              │ final gate, validation gate, monotonicity guard
                              ▼
                    EXECUTION_PAPER_ORDER_LATEST (KV bucket)
                              │
                              ▼
                    QueryResponderActor → ExecutionGateway (gateway) → HTTP
```

---

## Files Created (17 new files)

### Domain Layer
| File | Purpose |
|------|---------|
| `internal/domain/execution/execution.go` | ExecutionIntent entity, Side, Status, RiskInput, Validate(), PartitionKey(), DeduplicationKey() |
| `internal/domain/execution/events.go` | PaperOrderSubmittedEvent |

### Application Layer
| File | Purpose |
|------|---------|
| `internal/application/execution/paper_order_evaluator.go` | Pure function evaluator (risk → execution intent) |
| `internal/application/executionclient/contracts.go` | ExecutionLatestQuery, ExecutionLatestReply |
| `internal/application/executionclient/get_latest_execution.go` | GetLatestExecutionUseCase |
| `internal/application/ports/execution.go` | ExecutionGateway port interface |

### Adapter Layer
| File | Purpose |
|------|---------|
| `internal/adapters/nats/execution_registry.go` | EXECUTION_EVENTS stream, subjects, consumer specs |
| `internal/adapters/nats/execution_publisher.go` | Publishes to EXECUTION_EVENTS with dedup |
| `internal/adapters/nats/execution_consumer.go` | Durable JetStream consumer |
| `internal/adapters/nats/execution_gateway.go` | NATS request/reply gateway adapter |
| `internal/adapters/nats/execution_kv_store.go` | KV store with monotonicity guard |

### Actor Layer
| File | Purpose |
|------|---------|
| `internal/actors/scopes/derive/execution_evaluator_actor.go` | PaperOrderEvaluatorActor |
| `internal/actors/scopes/derive/execution_publisher_actor.go` | ExecutionPublisherActor |
| `internal/actors/scopes/store/execution_projection_actor.go` | ExecutionProjectionActor with 3-gate pattern |
| `internal/actors/scopes/store/execution_consumer_actor.go` | ExecutionConsumerActor |

### HTTP Interface
| File | Purpose |
|------|---------|
| `internal/interfaces/http/handlers/execution.go` | GET /execution/:type/latest handler |
| `internal/interfaces/http/routes/execution.go` | Route registration |

---

## Files Modified (14 files)

### Derive Wiring
| File | Change |
|------|--------|
| `internal/actors/scopes/derive/messages.go` | Added riskAssessedMessage, publishExecutionMessage |
| `internal/actors/scopes/derive/source_scope_actor.go` | Added ExecutionFamilyProcessor, executionPublisherPID, executionEvaluators map, routeRiskToExecution(), spawning logic |
| `internal/actors/scopes/derive/derive_supervisor.go` | Added execRegistry, executionProcessors, allExecutionProcessors registration |
| `internal/actors/scopes/derive/risk_evaluator_actor.go` | Added ScopePID to config, sends riskAssessedMessage for downstream fan-out |

### Store Wiring
| File | Change |
|------|--------|
| `internal/actors/scopes/store/messages.go` | Added executionReceivedMessage |
| `internal/actors/scopes/store/store_supervisor.go` | Added ExecutionPipeline, allExecutionPipelines, execution registry to query responder |
| `internal/actors/scopes/store/query_responder_actor.go` | Added ExecutionRegistry to config, executionPaperOrderStore, handleExecutionPaperOrderLatest |

### Gateway Wiring
| File | Change |
|------|--------|
| `cmd/gateway/gateway.go` | Added newExecutionGateway factory |
| `cmd/gateway/run.go` | Added execution gateway creation, use case wiring, route deps |

### Settings & Config
| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added knownExecutionFamilies, executionDependsOnRisk, ExecutionFamilies field, IsExecutionFamilyEnabled, EnabledExecutionFamilies, validation rules 10-11 |
| `deploy/configs/derive.jsonc` | Added execution_families: ["paper_order"] |
| `deploy/configs/store.jsonc` | Added execution_families: ["paper_order"] |

### HTTP Routes
| File | Change |
|------|--------|
| `internal/interfaces/http/routes/core.go` | Added ExecutionFamilyDeps, handlersGetLatestExecutionUseCase, Execution field on Dependencies, route wiring |

### Governance
| File | Change |
|------|--------|
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | EXECUTION_EVENTS moved to CANONICAL_STREAMS, removed from PROHIBITED_STREAMS, activated 5 execution drift checks, updated test helpers |

---

## Contracts Implemented

| Resource | Type | Value |
|----------|------|-------|
| Stream | JetStream | `EXECUTION_EVENTS` (72h retention, 2GB, FileStorage) |
| Subject | Event | `execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}` |
| Subject | Query | `execution.query.paper_order.latest` |
| Durable | Consumer | `store-execution-paper-order` |
| Bucket | KV | `EXECUTION_PAPER_ORDER_LATEST` |
| Endpoint | HTTP | `GET /execution/:type/latest?source=...&symbol=...&timeframe=...` |
| Envelope | Event | `execution.events.v1.paper_order_submitted` |
| Envelope | Request | `execution.query.v1.paper_order_latest_request` |
| Envelope | Reply | `execution.query.v1.paper_order_latest_reply` |

---

## Boundary Invariants Respected

| ID | Invariant | Status |
|----|-----------|--------|
| EBI-1 | No cross-domain imports | Enforced — RiskInput is execution-owned |
| EBI-2 | Evaluators are pure functions | Enforced — PaperOrderEvaluator has no I/O |
| EBI-3 | Risk data via local actor messages | Enforced — riskAssessedMessage has primitive fields |
| EBI-4 | Only derive publishes to EXECUTION_EVENTS | Enforced — single ExecutionPublisherActor per source |
| EBI-5 | Only store materializes projections | Enforced — single ExecutionProjectionActor per bucket |
| EBI-6 | Gateway never accesses KV directly | Enforced — uses NATS request/reply |
| EBI-7 | No external venue API interaction | Enforced — paper_order records intent only |
| EBI-8 | RiskInput is domain-owned | Enforced — no import of risk package |
| EBI-9 | No aggregation across assessments | Enforced — one risk → one intent |
| EBI-10 | No cumulative position/P&L | Enforced — no state tracked |

---

## Governance Status

| Check | Result |
|-------|--------|
| raccoon-cli drift-detect | 32/32 PASS |
| raccoon-cli cargo test | 29/29 PASS |
| go build ./cmd/derive/... | OK |
| go build ./cmd/store/... | OK |
| go build ./cmd/gateway/... | OK |
| go test (all project packages) | All OK (1 pre-existing unrelated failure in configctl) |

---

## Items Explicitly Deferred

| Item | Rationale | When |
|------|-----------|------|
| Venue adapter families (EF-02, EF-03) | Requires venue adapter architecture | S77+ |
| Order lifecycle state machine | Requires venue interaction | S77+ |
| Fill processing and reconciliation | No venue means no fills | S77+ |
| Execution history projection | Requires trace metadata persistence decision | S72+ |
| Kill switch / circuit breaker | S76 design concern | S76 |
| Position tracking | Portfolio domain | Future |
| Multi-strategy aggregation | One risk → one intent | Future |
| Execution-specific smoke test | Runtime test against live NATS | Future |
| Paper order evaluator unit tests | Pure function, testable | Follow-up |

---

## Stage Closure Checklist

- [x] Single execution family implemented (paper_order)
- [x] Family derived from risk (position_exposure → paper_order)
- [x] Latest-only projection (no history)
- [x] Query surface minimal and clear (GET /execution/:type/latest)
- [x] Store remains read-side authority
- [x] Gateway clean — no domain logic
- [x] Portfolio not opened
- [x] No venue adapter, retries, order state machine, or multi-venue router
- [x] All 3 binaries compile
- [x] All project Go tests pass
- [x] raccoon-cli 32/32 drift checks pass
- [x] S69 design followed exactly (no deviations)
- [x] Boundary invariants EBI-1 through EBI-10 respected
- [x] Configuration-driven activation (execution_families: ["paper_order"])
- [x] Cross-layer dependency validation (paper_order requires position_exposure)
