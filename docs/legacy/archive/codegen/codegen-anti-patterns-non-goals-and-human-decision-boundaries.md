# Codegen — Anti-Patterns, Non-Goals, and Human Decision Boundaries

## Purpose

This document defines what codegen must NOT do, which decisions remain exclusively human, and which anti-patterns the tranche must actively avoid. It exists to prevent the codegen investment from collapsing architectural governance into template mechanics.

## Non-Goals

### What This Codegen Tranche Is NOT

| Non-Goal | Why |
|----------|-----|
| A generic Go code generator | This is a Market Foundry analytical family generator. It knows about NATS, ClickHouse, the writer pipeline, and the analytical reader pattern. It is not general-purpose. |
| A runtime framework | Generated code has zero runtime dependency on the codegen tool. No reflection, no code loading, no plugin system. |
| A replacement for domain modeling | Domain types (`internal/domain/`) are human-authored architectural decisions. Codegen consumes domain types; it does not produce them. |
| A configuration management system | Codegen produces Go code and SQL. It does not manage deployment configs, secrets, or infrastructure. Config entries (e.g., `writer.jsonc` family arrays) are a narrow exception. |
| An abstraction layer | Codegen does not introduce new abstractions. Generated code uses the same patterns, types, and helpers as hand-crafted code. |
| A test framework | Generated tests use standard Go `testing` package. No test DSL, no assertion library, no codegen-specific test infrastructure. |
| A migration tool | Codegen can produce migration DDL files but does not run migrations. `cmd/migrate` remains the migration executor. |
| A schema evolution manager | Schema changes (adding columns, altering types, changing TTL) are architectural decisions, not template operations. |

### Features Explicitly Excluded

1. **Auto-discovery of event types**: Codegen does not scan the codebase for new domain events and auto-generate families. Family creation is an explicit, deliberate human action (creating a spec file).

2. **Dependency resolution**: Codegen does not resolve import paths, determine package boundaries, or manage Go module dependencies. Generated files use hardcoded, known import paths.

3. **Incremental generation**: Codegen regenerates all artifacts for a spec file from scratch each time. No incremental patching, no merge logic, no conflict resolution.

4. **Multi-family composition**: Codegen generates one family at a time. Cross-family queries, aggregations, and composite endpoints are out of scope.

5. **Self-modifying templates**: Templates do not contain conditional logic that changes template structure. Templates are static text with variable substitution and simple iteration.

6. **Code formatting decisions**: Generated code is `gofmt`-formatted. No custom formatting, no template-specific style rules.

## Anti-Patterns

### Anti-Pattern 1: The Abstraction Trap

**Symptom**: Codegen introduces a `GenericFamily[T]` struct or `RegisterFamily()` function that all families use at runtime.

**Why it's wrong**: This creates a runtime framework. When the framework changes, all families break simultaneously. The current pattern's strength is that each family is independent — a bug in the signal mapper does not affect the candle mapper.

**Rule**: Generated code must be copy-paste equivalent. No shared codegen runtime types.

### Anti-Pattern 2: The Config Explosion

**Symptom**: The family spec file grows to 200+ lines with nested config for every conceivable option (retry policies, batch sizes, custom filters, validation rules).

**Why it's wrong**: Spec complexity should be proportional to family complexity. A within-layer variant (Tier 1) should need ~15 lines of spec. Options that apply to all families belong in the writer/inserter infrastructure, not in per-family specs.

**Rule**: Spec files contain only what varies per family. Everything else is in shared infrastructure.

### Anti-Pattern 3: The Template Monolith

**Symptom**: A single template file generates all artifacts for a family (mapper + consumer + pipeline + tests + reader + handler).

**Why it's wrong**: Changes to the mapper template should not risk breaking the handler template. Template files should be as independent as the artifacts they generate.

**Rule**: One template per artifact type. Templates compose via the spec, not via template inheritance.

### Anti-Pattern 4: The Silent Divergence

**Symptom**: Templates evolve independently from hand-crafted code. New patterns are introduced in templates that don't exist in the 6 golden families.

**Why it's wrong**: If generated code looks structurally different from hand-crafted code, reviewers can't use their existing knowledge to evaluate it. The codegen promise is "same code, less effort" — not "different code, automated."

**Rule**: Golden test equivalence is mandatory. If a template change would break equivalence, either (a) the change is wrong, or (b) the golden families need to be updated first (manually), and then the template follows.

### Anti-Pattern 5: The Over-Generator

**Symptom**: Codegen generates domain types, NATS stream definitions, compose service entries, CI configuration, and documentation — because "why not, it's easy."

**Why it's wrong**: Each generated artifact is a maintenance commitment. The codegen tool must be maintained, its output must be validated, and drift must be detected. Generating artifacts that change rarely or require architectural judgment creates maintenance cost with near-zero benefit.

**Rule**: Only generate artifacts that repeat identically across families with zero creative decisions. If an artifact requires judgment calls, it stays manual.

### Anti-Pattern 6: The Magic Marker Section

**Symptom**: Marker sections (`// --- codegen:start ---`) proliferate across many files, making it unclear which parts of a file are human-owned and which are generated.

**Why it's wrong**: Mixed ownership within a file creates confusion about who is responsible for correctness. Developers may edit generated sections, or codegen may overwrite manual changes.

**Rule**: Minimize marker sections. Prefer dedicated generated files over marker sections in shared files. When marker sections are unavoidable (e.g., `pipeline.go` entries), they must be clearly bounded and append-only.

### Anti-Pattern 7: The Premature Tier 2

**Symptom**: S193 attempts to build Tier 2 templates (full read-path generation) before Tier 1 is validated end-to-end.

**Why it's wrong**: Tier 2 is 3x more complex than Tier 1 (15 artifacts vs 6). Building both simultaneously dilutes focus and delays the first generated family (EMA Crossover), which is the concrete proof point.

**Rule**: Tier 1 first. Tier 2 is authorized only after Tier 1 is validated in production with at least one generated family.

## Human Decision Boundaries

### Decisions That Are Always Human

| Decision | Why It Cannot Be Automated |
|----------|---------------------------|
| **Which event types become analytical families** | Business/architectural judgment about which data has analytical value |
| **Domain type design** | Event struct shape, field semantics, enum definitions are domain modeling |
| **Schema design for new layers** | Column types, partitioning strategy, TTL policy require understanding of query patterns |
| **Stream and subject naming** | NATS topology is an infrastructure architecture decision |
| **Which filters an endpoint supports** | API surface decisions affect consumers and must be intentional |
| **When to create a new table vs reuse existing** | Schema design trade-off (isolation vs complexity) |
| **Mapper field transformations** | Deciding how to map a decimal field (parseFloat vs string vs custom) requires domain knowledge |
| **Retry/backoff policy per family** | Operational judgment about acceptable data loss vs latency |
| **Whether a family warrants codegen or manual implementation** | Some families are one-offs or edge cases that don't justify template conformance |
| **Template evolution** | Changing templates is an architectural change, not a mechanical operation |

### Decisions That Codegen Makes

| Decision | Why It Can Be Automated |
|----------|------------------------|
| Consumer spec construction | Mechanical: subject + durable + stream → ConsumerSpec struct |
| Pipeline entry wiring | Mechanical: family + table + SQL + consumer → writerPipeline struct |
| Mapper column ordering | Deterministic: follows DDL column order |
| Test case generation | Deterministic: validation matrix from field types |
| Config array entry | Mechanical: append family name to array |
| Smoke test phase | Mechanical: endpoint + required fields from spec |
| Reader query building (Tier 2) | Deterministic: columns + filters from spec → SQL builder |
| Handler param extraction (Tier 2) | Deterministic: query struct fields → URL param parsing |

### The Boundary Test

When evaluating whether an artifact should be generated:

```
1. Has this artifact been implemented ≥3 times with 0 creative decisions?
   → Candidate for generation.

2. Does the artifact vary in ways that are fully captured by the spec?
   → Candidate for generation.

3. Does the artifact require understanding context beyond the spec?
   → Must remain manual.

4. Would a wrong version of this artifact cause silent data corruption?
   → Must be generated with golden test validation, or remain manual.

5. Does the artifact define an API contract visible to consumers?
   → Human review required even if generated.
```

## Scope Freeze

### What Is Frozen For This Tranche

1. Tier 1 artifact list (6 artifact types, defined in companion document).
2. Spec shape (fields defined in source-of-truth document).
3. Template granularity (one template per artifact type).
4. Validation approach (golden test equivalence against 6 families).
5. CI integration model (verification, not generation).

### What May Evolve

1. Template content (as patterns evolve, templates follow — with golden test validation).
2. Tier 2 scope (defined after Tier 1 is proven).
3. Spec shape extensions (new fields added for Tier 2; existing fields stable).
4. CI step implementation (tooling may change; principles are fixed).

### What Requires a New Tranche

1. Cross-family features (aggregations, composite queries).
2. Non-analytical codegen (operational queries, control plane).
3. Multi-service codegen (generating artifacts across multiple Go modules).
4. External consumer generation (client SDKs, API documentation).

These are explicitly **not in scope** and would require a separate scoping stage with their own gate conditions.
