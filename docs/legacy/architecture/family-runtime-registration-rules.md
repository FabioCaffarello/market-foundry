# Family Runtime Registration Rules

> Rules for registering new families, pipelines, and processors in Market Foundry runtimes.

## Core Rule

**One catalog entry = one family.** Adding a new family to any runtime means adding exactly one entry to the appropriate declarative catalog. All derived artifacts (trackers, filters, log fields, query routes) follow from that entry.

## Registration by Runtime

### Store — Projection Pipeline Registration

**Catalog:** `declarePipelines()` in `internal/actors/scopes/store/store_supervisor.go`

**To add a new pipeline:**

1. Add one `Pipeline` struct entry in `declarePipelines()`.
2. Implement the projection actor (materializes events → KV bucket).
3. Implement the consumer actor (reads from JetStream, forwards to projection).
4. Add the NATS registry entries (event spec, control spec) if not already present.
5. Add the KV bucket constant in the NATS adapter package.

**What you do NOT need to do:**
- ~~Add a tracker definition~~ — derived from `PipelineTrackerDefs()`
- ~~Wire the query responder~~ — scope-based registry injection handles it
- ~~Update the composition root~~ — `cmd/store/run.go` iterates the catalog

**Pipeline entry requirements:**

| Field | Required | Notes |
|-------|:--------:|-------|
| `Scope` | Yes | Must be one of `DomainEvidence`, `DomainSignal`, `DomainDecision`, `DomainStrategy`, `DomainRisk`, `DomainExecution` |
| `Family` | Yes | Canonical name matching `knownXxxFamilies` in settings/schema.go |
| `ProjectionName` | Yes | Convention: `{scope}-{family}-projection` (or `{family}-projection` for evidence) |
| `ConsumerName` | Yes | Convention: `{scope}-{family}-consumer` (or `{family}-consumer` for evidence) |
| `Buckets` | Yes | KV bucket names owned by this projection |
| `ConsumerSpec` | Yes | Durable consumer definition from NATS adapter |
| `IsEnabled` | Yes | Must use the scope-appropriate `PipelineConfig` method |
| `NewProjection` | Yes | Factory returning `actor.Producer` |
| `NewConsumer` | Yes | Factory capturing registry via closure |

**Activation rules:**
- Evidence families: enabled by default when no `pipeline.families` configured (backward compatible)
- All other scopes: opt-in only via `pipeline.{scope}_families`
- The `IsEnabled` predicate must call the correct scope method (`IsFamilyEnabled`, `IsSignalFamilyEnabled`, etc.)

### Store — Query Responder Registration

When adding a pipeline to an **existing scope**, the query responder's registry is already injected for that scope. You only need to:

1. Open the KV store in `query_responder_actor.go:start()`.
2. Add the control route.
3. Add the handler method.
4. Add the close logic in `actor.Stopped`.

When adding a pipeline for a **new scope** (rare — requires new domain):

1. Create the scope's NATS registry type.
2. Add the registry to `pipelineRegistries`.
3. Add the scope constant.
4. Add the registry injection in `queryResponderConfig()`.

### Derive — Family Processor Registration

**Catalog:** Scope-specific slices in `derive_supervisor.go:start()`

**To add a new processor:**

1. Add one entry to the appropriate processor slice (evidence: `[]FamilyProcessor`, signal: `[]SignalFamilyProcessor`, etc.).
2. Implement the processor actor.
3. If adding to a new scope: add the publisher spawn logic in `source_scope_actor.go:start()` and the routing method.

**Processor types and their signatures:**

| Type | NewActor Params | Scope |
|------|----------------|-------|
| `FamilyProcessor` | `(source, symbol, timeframe, publisherPID, scopePID)` | evidence |
| `SignalFamilyProcessor` | `(source, symbol, timeframe, signalPublisherPID, scopePID)` | signal |
| `DecisionFamilyProcessor` | `(source, symbol, timeframe, decisionPublisherPID, scopePID)` | decision |
| `StrategyFamilyProcessor` | `(source, symbol, timeframe, strategyPublisherPID, scopePID)` | strategy |
| `RiskFamilyProcessor` | `(source, symbol, timeframe, riskPublisherPID, scopePID)` | risk |
| `ExecutionFamilyProcessor` | `(source, symbol, timeframe, executionPublisherPID)` | execution (no scopePID — terminal) |

These types are intentionally distinct. Execution processors don't receive a `scopePID` because they're terminal in the pipeline chain. Attempting to unify them into a single type would lose this type-level guarantee.

### Gateway — Connection Registration

**Catalog:** `buildGatewayConns()` in `cmd/gateway/compose.go`

**To add a new query gateway:**

1. Add one field to `gatewayConns`.
2. Add one `newGatewayConn()` call in `buildGatewayConns()`.
3. Add the use case wiring in `buildRouteDependencies()`.
4. Add the family deps type and routes in `internal/interfaces/http/routes/`.

### Settings — Family Configuration

**Catalog:** `internal/shared/settings/schema.go`

Every new family must be registered in:

1. `knownXxxFamilies` set — for config validation.
2. A `PipelineConfig.IsXxxFamilyEnabled()` method — for activation predicates.

Cross-layer dependency validation ensures families can only be activated when their upstream dependencies are also enabled.

## Naming Conventions

| Entity | Convention | Example |
|--------|-----------|---------|
| Family name | `snake_case` | `mean_reversion_entry` |
| Projection actor name | `{scope}-{family}-projection` | `strategy-mean-reversion-entry-projection` |
| Consumer actor name | `{scope}-{family}-consumer` | `strategy-mean-reversion-entry-consumer` |
| KV bucket constant | `{Scope}{Family}LatestBucket` | `StrategyMeanReversionEntryLatestBucket` |
| Consumer spec function | `Store{Family}{Scope}Consumer()` | `StoreMeanReversionEntryStrategyConsumer()` |
| Processor actor prefix | `{scope}-{family}` | `strategy-mean-reversion-entry` |

## Checklist for Adding a New Family

### Same-scope family (e.g., new evidence type)

- [ ] Add `Pipeline` entry in `store_supervisor.go:declarePipelines()`
- [ ] Add processor entry in `derive_supervisor.go:start()`
- [ ] Implement projection + consumer actors in store
- [ ] Implement processor actor in derive
- [ ] Add NATS registry entries (EventSpec, ControlSpec)
- [ ] Add KV bucket constant
- [ ] Add consumer spec function
- [ ] Add query handler in `query_responder_actor.go`
- [ ] Add family to `knownXxxFamilies` in settings
- [ ] Add gateway connection + use case + route in gateway (if queryable)

### New scope (rare — new domain layer)

All of the above, plus:
- [ ] Add `PipelineDomain` constant
- [ ] Add new processor type in derive
- [ ] Add publisher actor in derive
- [ ] Add routing method in `source_scope_actor.go`
- [ ] Add registry type in NATS adapter
- [ ] Add `IsXxxFamilyEnabled()` method on `PipelineConfig`
- [ ] Add cross-layer dependency validation in settings
- [ ] Add scope to `queryResponderConfig()` in store supervisor
