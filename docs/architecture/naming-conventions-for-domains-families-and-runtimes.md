# Naming Conventions for Domains, Families, and Runtimes

> Canonical reference for how to name entities across the Market Foundry monorepo.

## Core Concepts

### Domain (Bounded Context)

A **domain** is a bounded business context owning a set of types, events, and behaviors.

| Convention | Format | Examples |
|-----------|--------|----------|
| Package path | `internal/domain/{domain}/` | `internal/domain/configctl/`, `internal/domain/observation/` |
| Application logic | `internal/application/{domain}/` | `internal/application/configctl/` |
| Client package | `internal/application/{domain}client/` | `internal/application/evidenceclient/` |
| Port interface | `{Domain}Gateway` (PascalCase) | `ConfigctlGateway`, `EvidenceGateway` |
| Pipeline classifier | `Domain{Domain}` constant | `DomainEvidence`, `DomainSignal` |
| NATS registry | `{Domain}Registry` | `EvidenceRegistry`, `SignalRegistry` |

**Naming rule:** Domain names are lowercase, singular nouns: `evidence`, `signal`, `decision`, `strategy`, `risk`, `execution`, `observation`, `configctl`.

### Family (Stream Type)

A **family** is a specific type within a domain — the canonical name for a kind of event or projection.

| Convention | Format | Examples |
|-----------|--------|----------|
| Family name | `snake_case` | `candle`, `tradeburst`, `volume`, `rsi`, `rsi_oversold`, `paper_order` |
| Stream | `{DOMAIN}_EVENTS` (SCREAMING_CASE) | `EVIDENCE_EVENTS`, `OBSERVATION_EVENTS` |
| KV bucket | `{Family}{Qualifier}Bucket` (PascalCase) | `CandleLatestBucket`, `CandleHistoryBucket` |
| Consumer durable | `store-{family}` (kebab-case) | `store-candle`, `store-volume` |
| Projection actor | `{family}-projection` (kebab-case) | `candle-projection`, `volume-projection` |
| Consumer actor | `{family}-consumer` (kebab-case) | `candle-consumer`, `volume-consumer` |
| Processor actor | `{domain}-{family}` (kebab-case) | `evidence-candle`, `signal-rsi` |
| Pipeline entry | `Family: "{family}"` in `declarePipelines()` | `Family: "candle"`, `Family: "rsi"` |

**Naming rule:** Family names are lowercase `snake_case`. Never plural. Never prefixed with the domain name (the pipeline's `Domain` field provides that context).

### Runtime (Binary)

A **runtime** is a deployed process — one of the `cmd/{binary}/` entry points.

| Convention | Format | Examples |
|-----------|--------|----------|
| Binary name | lowercase, no separators | `gateway`, `store`, `derive`, `ingest`, `execute`, `configctl` |
| Root supervisor | `{Binary}Supervisor` (PascalCase) | `StoreSupervisor`, `DeriveSupervisor` |
| Package alias | `{binary}actor` | `storeactor`, `deriveactor`, `ingestactor` |
| Actor child name | `{binary}-supervisor` (kebab-case) | `store-supervisor`, `derive-supervisor` |
| NATS source ID | `{binary}.{role}` | `store.query-responder`, `gateway.http` |

**Naming rule:** Binary names are lowercase nouns describing the runtime's primary responsibility.

### Scope (Actor Supervision Boundary)

A **scope** is a runtime partition providing failure isolation for a group of related actors.

| Convention | Format | Examples |
|-----------|--------|----------|
| Scope actor | `{Kind}ScopeActor` | `SourceScopeActor`, `ExchangeScopeActor` |
| Scope config | `{Kind}ScopeConfig` | `SourceScopeConfig` |
| Scope naming in logs | `source={id}` | `source=binancef`, `exchange=binancef` |

**Naming rule:** Scope names describe the partitioning dimension (source, exchange). Never use "scope" for domain classification — use `PipelineDomain` instead.

## Cross-Layer Naming Map

| Layer | Entity | Naming Convention |
|-------|--------|------------------|
| Domain | Types, value objects | `PascalCase`, domain-specific nouns (`Candle`, `Signal`, `Decision`) |
| Application | Use cases | `{Verb}{Entity}UseCase` (`GetLatestCandleUseCase`, `CreateDraftUseCase`) |
| Application | Client packages | `{domain}client` (`evidenceclient`, `signalclient`) |
| Application | Ports | `{Domain}Gateway` interface (`EvidenceGateway`) |
| Adapters | NATS registry | `{Domain}Registry` (`EvidenceRegistry`) |
| Adapters | NATS gateway impl | `{Domain}Gateway` struct (implements port) |
| Adapters | Consumer spec | `Store{Family}Consumer()` function |
| Actors | Root | `{Binary}Supervisor` |
| Actors | Scope | `{Kind}ScopeActor` |
| Actors | Processor | `{Family}SamplerActor`, `{Family}EvaluatorActor`, `{Family}ResolverActor` |
| Actors | Projection | `{Family}ProjectionActor` |
| Actors | Consumer | `{Family}ConsumerActor` (for store) / `{Domain}ConsumerActor` (for evidence) |
| Actors | Publisher | `{Domain}PublisherActor` |
| Actors | Responder | `QueryResponderActor`, `ControlResponderActor` |
| Interfaces | HTTP handler | `{Domain}WebHandler` |
| Interfaces | Route deps | `{Domain}FamilyDeps` |
| Cmd | Composition root | `run.go` (+ `compose.go` when >80 lines) |

## Adding a New Family (Naming Checklist)

When adding a new family to an existing domain:

1. **Settings/schema.go**: Add to `known{Domain}Families` slice
2. **Store supervisor**: Add `Pipeline` entry in `declarePipelines()` with correct `PipelineDomain` constant
3. **Derive supervisor**: Add entry to scope-specific processor slice
4. **NATS adapter**: Add consumer spec function (`Store{Family}Consumer()`) and bucket constant
5. **Query responder**: Add KV handler for the new bucket
6. **Gateway**: Add use case, route dependency, and HTTP handler method (if queryable)

All names must follow the conventions above. The raccoon-cli `arch-guard` will reject violations of layered architecture rules.

## Adding a New Domain (Naming Checklist)

All of the above, plus:

1. **PipelineDomain**: Add `Domain{Name}` constant in `store_supervisor.go`
2. **Derive**: Add new processor type (`{Domain}FamilyProcessor`), publisher actor, routing in `SourceScopeActor`
3. **NATS adapter**: Add `{Domain}Registry` type with `Default{Domain}Registry()` constructor
4. **Application ports**: Add `{Domain}Gateway` interface
5. **Settings**: Add `Is{Domain}FamilyEnabled()` method on `PipelineConfig`
6. **Store supervisor**: Add registry field to `pipelineRegistries` and injection in `queryResponderConfig()`
