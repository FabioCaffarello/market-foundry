# Generated vs Manual Artifact Coverage After Slice Expansion

> Stage S260 — Definitive map of what is codegen-governed vs human-maintained.

## Coverage Matrix

### Writer Pipeline Artifacts (consumer_spec + pipeline_entry)

| Family | Layer | consumer_spec | pipeline_entry | Stage |
|--------|-------|:------------:|:--------------:|-------|
| candle | evidence | GENERATED | GENERATED | S260 |
| rsi | signal | GENERATED | GENERATED | S200 |
| ema | signal | GENERATED | GENERATED | S203 |
| rsi_oversold | decision | GENERATED | GENERATED | S260 |
| ema_crossover | decision | GENERATED | GENERATED | S260 |
| mean_reversion_entry | strategy | GENERATED | GENERATED | S260 |
| trend_following_entry | strategy | GENERATED | GENERATED | S260 |
| position_exposure | risk | GENERATED | GENERATED | S260 |
| drawdown_limit | risk | GENERATED | GENERATED | S260 |
| paper_order | execution | GENERATED | GENERATED | S260 |

**Result: 100% of tier-1 writer pipeline artifacts are codegen-governed.**

### NATS Adapter Artifacts (per-layer packages)

| Artifact | Scope | Status | Reason |
|----------|-------|--------|--------|
| Writer consumer spec | Per-family function | GENERATED | Governed via codegen markers |
| Store consumer spec | Per-family function | MANUAL | May diverge per store requirements |
| Registry struct | Per-layer type | MANUAL | Contains EventSpec + ControlSpec fields |
| DefaultRegistry() | Per-layer constructor | MANUAL | Includes stream definitions |
| LatestSpecByType() | Per-layer dispatcher | MANUAL | Domain-specific switch logic |
| consumer.go | Per-layer consumer | MANUAL | Handler callback is domain-specific |
| publisher.go | Per-layer publisher | MANUAL | Event type wiring is domain-specific |
| gateway.go | Per-layer gateway | MANUAL | Port interface varies per layer |
| kv_store.go | Per-layer KV store | MANUAL | Bucket naming per family |

### Actor Layer Artifacts

| Artifact | Scope | Status | Reason |
|----------|-------|--------|--------|
| Evaluator/Resolver actors | Per-family | MANUAL | Domain-specific message handling |
| Projection actors | Per-family | MANUAL | Domain-specific KV materialization |
| Publisher actors | Per-layer | MANUAL | Tied to evaluator actor lifecycle |
| Supervisor actors | Per-scope | MANUAL | Scope-specific child management |

### Application Layer Artifacts

| Artifact | Scope | Status | Reason |
|----------|-------|--------|--------|
| Evaluators/Resolvers | Per-family | MANUAL | Core domain logic |
| Severity/Risk scaling | Per-layer | MANUAL | Behavioral domain logic |
| Client use cases | Per-family | MANUAL | Query/reply contract variations |
| Runtime contracts | Cross-layer | MANUAL | Behavioral invariant enforcement |

### ClickHouse Adapter Artifacts

| Artifact | Scope | Status | Reason |
|----------|-------|--------|--------|
| Writer pipeline starters | Per-layer | MANUAL | Event type + mapper binding |
| Row mappers (mapXxxRow) | Per-family | MANUAL | Column-specific field extraction |
| Readers | Per-family | MANUAL | Query/scan logic varies |

### Domain Layer Artifacts

| Artifact | Scope | Status | Reason |
|----------|-------|--------|--------|
| Domain structs | Per-layer | MANUAL | Core domain model |
| Event types | Per-family | MANUAL | Event payload structure |

## Summary Statistics

| Category | Generated | Manual | Total | Coverage |
|----------|-----------|--------|-------|----------|
| Writer consumer specs | 10 | 0 | 10 | 100% |
| Writer pipeline entries | 10 | 0 | 10 | 100% |
| Store consumer specs | 0 | 10+ | 10+ | 0% |
| NATS adapter files | 0 | ~30 | ~30 | 0% |
| Actor files | 0 | ~20 | ~20 | 0% |
| Application files | 0 | ~15 | ~15 | 0% |
| ClickHouse files | 0 | ~12 | ~12 | 0% |
| Domain files | 0 | ~12 | ~12 | 0% |
| **Total** | **20** | **~120** | **~140** | **~14%** |

## Codegen Boundary Principle

The current codegen boundary follows a clear principle:

> **Generate what is purely structural and repetitive. Leave manual what contains domain semantics or behavioral logic.**

The writer pipeline configuration (consumer specs + pipeline entries) is the canonical example of pure structural repetition: every family needs the same fields populated from the same spec data. There is zero domain logic in these artifacts.

The next natural expansion candidates, ordered by structural repetition and low semantic risk:

1. **Writer pipeline starters** (support.go) — ~15 LOC per layer, pure wiring
2. **Store consumer specs** — same pattern as writer, but may need independent configuration
3. **NATS consumer/publisher scaffolds** — high repetition but tighter domain coupling

## What Explicitly Remains Manual

These categories are intentionally excluded from codegen scope:

- **Domain models and events** — core business semantics
- **Evaluators, resolvers, samplers** — behavioral domain logic
- **Severity/risk scaling** — recently added behavioral enrichments
- **Actor lifecycle management** — scope-specific supervision trees
- **ClickHouse row mappers** — column-specific field extraction with type coercion
- **Runtime behavioral contracts** — cross-layer invariant enforcement
