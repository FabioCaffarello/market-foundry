# Stage S108 — Vertical Slice Architecture Definition

**Status:** Complete
**Objective:** Define the minimal vertical slice that validates the market-foundry architecture end-to-end.

## Executive Summary

S108 formalizes the first vertical slice of market-foundry: `candle-to-paper-order`. This slice exercises the complete event pipeline from observation capture through execution fill, touching all 6 runtimes, all 8 domain families in the dependency chain, 9 JetStream streams, 11 durable consumers, 8 KV projection buckets, and the full HTTP query surface. The slice is deliberately narrow (one source, one symbol, one timeframe) to maximize architectural coverage while minimizing variables.

The slice was selected because it is the only complete pipeline the codebase already implements — making this a validation exercise, not a development project.

## Slice Chosen: `candle-to-paper-order`

### Pipeline

```
observation (trade) → candle (evidence) → rsi (signal) → rsi_oversold (decision)
    → mean_reversion_entry (strategy) → position_exposure (risk)
    → paper_order (execution intent) → venue_market_order (execution fill)
```

### Binding

```
source:    binancef
symbol:    btcusdt
timeframe: 60
venue:     paper_simulator
```

### Justification

1. **Already implemented.** Every actor, publisher, consumer, projection, and query handler in this chain exists in the codebase. Zero speculative code is needed.
2. **Architecturally complete.** The chain exercises every layer of the domain model, every communication pattern (pub/sub + request/reply), every runtime, and both write and read paths.
3. **Minimally scoped.** One binding, one timeframe, one venue. The fewest possible variables that still prove the architecture works end-to-end.
4. **High discovery value.** If any wiring is broken — a mismatched subject, a missing projection, a codec error, a race condition on startup — the slice will surface it because every link must work for the final query endpoint to return data.

## Architecture and Flow

### Data Flow

```
┌──────────┐    ┌──────────┐    ┌──────────────────────────────────────────┐
│  ingest  │───▶│  NATS    │◀──▶│  derive                                  │
│          │    │ JetStream│    │  trade → candle → rsi → rsi_oversold     │
│ binancef │    │          │    │  → mean_reversion → position_exposure    │
│ btcusdt  │    │ 9 streams│    │  → paper_order                          │
└──────────┘    │ 11 durables   └──────────────────────────────────────────┘
                │          │
┌──────────┐    │          │    ┌──────────┐    ┌──────────┐
│ configctl│───▶│          │◀──▶│  store   │◀──▶│ gateway  │
│          │    │          │    │ 8 KV     │    │ HTTP API │
│ lifecycle│    │          │    │ buckets  │    │          │
└──────────┘    │          │    └──────────┘    └──────────┘
                │          │
                │          │    ┌──────────┐
                │          │◀──▶│ execute  │
                │          │    │ paper    │
                └──────────┘    │ venue    │
                                └──────────┘
```

### Runtime Responsibilities

| Runtime | Responsibility | Inputs | Outputs |
|---------|---------------|--------|---------|
| configctl | Config lifecycle, binding activation | HTTP commands via gateway | Config events, binding events |
| ingest | Market data capture | WebSocket (binancef), binding events | TradeReceivedEvent |
| derive | Full processing pipeline | TradeReceivedEvent, binding events | 6 domain event types |
| store | Read model materialization, query serving | 8 event streams | KV projections, query replies |
| gateway | HTTP API surface | HTTP requests | NATS request/reply to store/configctl |
| execute | Venue order execution | PaperOrderSubmittedEvent | VenueOrderFilledEvent |

### Event Chain (8 steps)

| # | Event | Stream | Producer | Consumer |
|---|-------|--------|----------|----------|
| 1 | TradeReceivedEvent | OBSERVATION_EVENTS | ingest | derive |
| 2 | CandleSampledEvent | EVIDENCE_EVENTS | derive | store |
| 3 | SignalGeneratedEvent | SIGNAL_EVENTS | derive | store |
| 4 | DecisionEvaluatedEvent | DECISION_EVENTS | derive | store |
| 5 | StrategyResolvedEvent | STRATEGY_EVENTS | derive | store |
| 6 | RiskAssessedEvent | RISK_EVENTS | derive | store |
| 7 | PaperOrderSubmittedEvent | EXECUTION_EVENTS | derive | store, execute |
| 8 | VenueOrderFilledEvent | EXECUTION_FILL_EVENTS | execute | store |

## Success Criteria (Summary)

10 testable criteria defined in `vertical-slice-01-success-criteria-and-out-of-scope.md`:

| ID | Criterion | Proves |
|----|-----------|--------|
| SC-1 | Config lifecycle via HTTP | Config activation path |
| SC-2 | Dynamic binding propagation | Runtime reactivity without restart |
| SC-3 | Observation capture | Ingest → NATS flow |
| SC-4 | Full derive pipeline | 6-domain event chain |
| SC-5 | Execution fill | Execute runtime processing |
| SC-6 | Read model materialization | 8 KV buckets populated |
| SC-7 | Query surface completeness | All HTTP endpoints return data |
| SC-8 | Diagnostic visibility | Health/status/diag endpoints working |
| SC-9 | Graceful lifecycle | Clean start/stop without leaks |
| SC-10 | Envelope integrity | Correlation/causation chain preserved |

## Limits and Non-Objectives

| Exclusion | Rationale |
|-----------|-----------|
| Multi-symbol / multi-source | Same architectural path, just more volume |
| Multi-timeframe | Orthogonal concern, doesn't test new wiring |
| Production venue | Paper simulator exercises the same adapter pattern |
| ClickHouse projections | NATS KV proves read/write separation |
| Horizontal scaling | Operational concern, not wiring concern |
| Schema evolution | Premature before basic interop is proven |
| Auth / multi-tenancy | Feature layer, not architecture layer |
| Automated E2E suite | Slice reveals what the suite needs to cover |
| Performance benchmarks | Correctness first; benchmarks need baseline data |
| tradeburst / volume families | Same pattern as candle, no new proof value |

## Deliverables

| Document | Purpose |
|----------|---------|
| `docs/architecture/vertical-slice-01-definition.md` | Slice identity, participating runtimes/families, config, proof points |
| `docs/architecture/vertical-slice-01-contracts-events-and-read-models.md` | Complete contract/event/stream/KV/query checklist |
| `docs/architecture/vertical-slice-01-success-criteria-and-out-of-scope.md` | 10 success criteria, 10 exclusions, risk register |
| `docs/stages/stage-s108-vertical-slice-architecture-definition-report.md` | This report |

## Preparation for S109

S109 should execute the vertical slice. Recommended preparation:

### Pre-Flight Checks (before starting S109)

1. **Verify all runtimes compile**: `go build ./cmd/...`
2. **Verify all tests pass**: `make test`
3. **Verify compose stack starts**: `docker compose -f deploy/compose/docker-compose.yaml up`
4. **Verify NATS connectivity**: All runtimes reach `/readyz` 200

### S109 Execution Sequence

1. **Start infrastructure** (NATS, ClickHouse)
2. **Start runtimes** in dependency order: configctl → store → gateway → derive → ingest → execute
3. **Create and activate config** via gateway HTTP endpoints
4. **Wait for 2 candle windows** (~120 seconds for 60s timeframe)
5. **Validate SC-1 through SC-10** against the success criteria
6. **Document findings**: What worked, what broke, what needs fixing

### Expected Friction Points

Based on codebase analysis, these areas are most likely to surface issues:

| Area | Concern |
|------|---------|
| RSI cold start | RSI requires candle history; first window may produce `insufficient` signals — verify the decision/strategy/risk chain handles this gracefully |
| Store pipeline enablement | `declarePipelines()` must include all 8 projection types with correct `IsEnabled` predicates matching the pipeline config |
| Execute consumer wiring | `execute-venue-market-order-intake` consumer must correctly filter paper_order events from EXECUTION_EVENTS |
| Gateway optional gateways | Gateway's `HasAny()` checks must not skip endpoints when the corresponding NATS gateway is available |
| KV bucket creation order | Store must create buckets before projection actors start writing — verify idempotent create-or-bind |

### What S109 Will Produce

- A running end-to-end pipeline (or a list of what's broken)
- Baseline event counts and latencies from `/statusz`
- Confidence that the 10-stage infrastructure investment actually works in practice
- A backlog of real issues discovered during integration (if any)
- Input for deciding whether to expand scope or fix integration issues next

## Structural Gains

1. **Architecture is now testable.** The slice provides a concrete, reproducible protocol for validating the platform's wiring — not just individual unit tests, but the integration surface.
2. **Scope is disciplined.** Every exclusion is explicit and justified. The temptation to add "just one more family" or "just one more source" is preempted by the out-of-scope document.
3. **Success is measurable.** 10 binary pass/fail criteria. No subjective assessments.
4. **Discovery is the goal.** The slice exists to find problems, not to ship features. Every issue it surfaces is a return on the 10 stages of infrastructure investment.
