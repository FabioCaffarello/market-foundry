# Boundary Naming and Interface Hygiene

> Canonical reference for naming clarity across bounded contexts, layers, and runtime interfaces.

## Principles

1. **Names must reflect current responsibilities** — not historical origins or aspirational future scope.
2. **The same term must not mean different things in different layers** — if it does, one usage must be renamed.
3. **Error messages are part of the interface** — operator-visible strings must use architecture-correct terminology.
4. **Test data is documentation** — labels and identifiers in test fixtures must align with the current project identity.

## Terminology Disambiguation

### Scope vs Domain

| Term | Meaning | Where used | Examples |
|------|---------|-----------|----------|
| **Scope** | Runtime supervision boundary — a partition of actors sharing a lifecycle and failure domain | Actor layer (`SourceScopeActor`, `ExchangeScopeActor`) | One scope per exchange in ingest, one scope per data source in derive |
| **Domain** | Bounded business context — a group of related types, events, and projections | Pipeline catalog, application layer, domain layer | Evidence, signal, decision, strategy, risk, execution |

**Rule:** Use `Scope` only for actor supervision hierarchy. Use `Domain` for business-context classification. Never mix them.

The store supervisor's `PipelineDomain` type classifies pipelines by bounded context (evidence, signal, decision, etc.). This was previously named `PipelineScope`, which conflicted with the actor-layer meaning of "scope."

### Gateway vs Registry

| Term | Meaning | Where used | Examples |
|------|---------|-----------|----------|
| **Gateway** | Port interface for cross-binary communication — request/reply abstraction | `application/ports/` (interfaces), `adapters/nats/` (implementations) | `EvidenceGateway`, `ConfigctlGateway` |
| **Registry** | Declarative mapping of domain concepts to NATS infrastructure (subjects, streams, buckets) | `adapters/nats/` (value objects) | `EvidenceRegistry`, `SignalRegistry` |

**Rule:** A gateway is a port (behavioral contract). A registry is a value object (data mapping). A gateway *uses* a registry but they serve different purposes. Both are legitimate names in the adapters layer.

### Handler vs Responder

| Term | Meaning | Layer |
|------|---------|-------|
| **WebHandler** | HTTP request handler (struct with methods per endpoint) | `interfaces/http/handlers/` |
| **ResponderActor** | Actor that serves NATS request/reply queries | `actors/scopes/` |
| **Responder** | NATS adapter utility for request/reply mechanics | `adapters/nats/` |

**Rule:** `WebHandler` is for HTTP. `ResponderActor` is for actors serving NATS queries. Plain `Responder` is for adapter-layer utilities. The prefix/suffix indicates the layer.

### Processing Verbs (Derive Pipeline)

| Verb | Domain | Actor | Meaning |
|------|--------|-------|---------|
| **Sample** | Evidence, Signal | `SamplerActor`, `SignalSamplerActor` | Collect and structure raw observations into typed data |
| **Evaluate** | Decision, Risk | `DecisionEvaluatorActor`, `RiskEvaluatorActor` | Apply rules to signals to produce judgments |
| **Resolve** | Strategy | `StrategyResolverActor` | Combine evaluations into actionable strategy parameters |

**Rule:** These verbs are intentionally distinct — they reflect genuinely different processing semantics. Do not unify them. The verb encodes domain meaning: samplers aggregate, evaluators judge, resolvers synthesize.

## Error Message Conventions

Error messages visible to operators (logs, API responses) must use the correct architectural term:

| Incorrect | Correct | Why |
|-----------|---------|-----|
| `"X service is unavailable"` | `"X gateway is unavailable"` | Market Foundry uses gateways (NATS request/reply), not services |
| `"config service"` | `"configctl gateway"` | The binary is `configctl`, the abstraction is a gateway |

## Test Data Identity

Test fixtures must reflect the current project identity:

| Legacy | Current |
|--------|---------|
| `"Core Quality Config"` | `"Core Market Config"` |
| `"team": "quality"` | `"team": "foundry"` |

## Hygiene Checklist for New Code

- [ ] Types in `application/ports/` end with `Gateway` (behavioral port)
- [ ] Types in `adapters/nats/` use `Registry` for subject/bucket mappings
- [ ] HTTP handlers use `WebHandler` suffix
- [ ] Actor-layer query handlers use `ResponderActor` suffix
- [ ] `PipelineDomain` constants classify pipelines (not `PipelineScope`)
- [ ] Error messages reference "gateway" not "service"
- [ ] Test data uses `market-foundry` / `foundry` identity, not legacy names
- [ ] The raccoon-cli module prefix matches actual Go module paths (`internal/`)
