# Codegen Tranche Scoping — S192 Principal Document

## Purpose

This document formally defines the codegen tranche that replaces manual analytical family expansion in Market Foundry. It transforms the structural ceiling identified in S191 (no Family 06 candidate satisfies the "no write-path changes" gate condition) into an architectural plan for automated, governed family generation.

The scope is **decision and definition only** — no implementation occurs in this stage.

## Context and Rationale

### Why Codegen, Why Now

Wave B delivered 6 analytical families covering all vertical layers (L1–L6) with:
- 0 creative decisions across 5 expansions
- 0 write-path modifications across 6 expansions
- ~780 LOC per family at ~45 min manual effort
- 289 unit tests following identical shapes

The S191 trigger assessment proved that every remaining candidate requires write-path changes. The analytical read-path is already type-parameterized and generic — within-layer variants (e.g., EMA Crossover in L2 Signals) work at the HTTP/reader/use-case level today, returning empty results only because no writer pipeline entry persists data to ClickHouse.

This means:
1. The manual 9-artifact expansion pattern has no remaining pure read-path work.
2. Future family expansion requires write-path + read-path artifacts generated together.
3. The 6 existing families provide a complete, zero-ambiguity specification for generation templates.

### What Changed Since S190

| S190 Gate Expectation | Actual Outcome (S191) |
|---|---|
| Family 06 may proceed under conditions | No candidate satisfies C1 (no write-path changes) |
| Codegen mandatory before Family 07 | Codegen mandatory before **any** next family |
| Manual pattern "approaching ceiling" | Manual pattern **at** ceiling — structurally complete |

The timeline accelerated: codegen is not a "nice to have before Family 07" — it is the **only path** to the next family.

## Tranche Definition

### What Is a "Codegen Tranche"

A codegen tranche is a bounded, scoped set of generation capabilities that:
- Covers a specific set of artifacts (not "everything")
- Has a single source of truth (not scattered config)
- Produces output that is structurally equivalent to hand-crafted code
- Is validated against existing families (not trusted on spec alone)
- Enters CI as a verification step (not a black-box build step)

### Tranche Scope: Analytical Family Generation

This tranche generates the artifacts required to add a new analytical event type to the Market Foundry pipeline. It covers **within-layer expansion** (new event types within existing L1–L6 layers), which is the structural bottleneck identified in S191.

**In scope**: artifacts that repeat identically across families with zero creative decisions.
**Out of scope**: new layers, cross-family features, infrastructure changes.

## Artifact Inventory

### Complete Artifact Map Per Family

Based on evidence from 6 hand-crafted families, each analytical family consists of these artifacts:

| # | Artifact | Layer | Path Pattern | Varies By |
|---|----------|-------|-------------|-----------|
| 1 | Writer consumer spec | Write | `internal/adapters/nats/{domain}_registry.go` | Subject, durable name, event type |
| 2 | Writer pipeline entry | Write | `cmd/writer/pipeline.go` | Family name, consumer spec, table, insert SQL, mapper |
| 3 | Writer mapper function | Write | `cmd/writer/mappers.go` | Column count, field extraction, JSON/float/enum handling |
| 4 | Writer mapper tests | Write | `cmd/writer/mappers_test.go` | Field assertions, edge cases per column type |
| 5 | Reader adapter method | Read | `internal/adapters/clickhouse/{family}_reader.go` | Query columns, optional filters, row scanning |
| 6 | Reader adapter tests | Read | `internal/adapters/clickhouse/{family}_reader_test.go` | Query building, filter combinations |
| 7 | Use case | Read | `internal/application/analyticalclient/get_{family}_history.go` | Query/Reply types, validation rules |
| 8 | Use case tests | Read | `internal/application/analyticalclient/get_{family}_history_test.go` | Validation matrix |
| 9 | Handler method | Read | `internal/interfaces/http/handlers/analytical.go` | Query param extraction, use case dispatch |
| 10 | Handler tests | Read | `internal/interfaces/http/handlers/analytical_test.go` | HTTP request/response assertions |
| 11 | Route registration | Read | `internal/interfaces/http/routes/analytical.go` | Path, method, handler binding |
| 12 | Contracts (query/reply) | Read | `internal/application/analyticalclient/contracts.go` | Query struct fields, reply result type |
| 13 | Gateway composition | Read | `cmd/gateway/analytical_reader.go` | Reader factory function |
| 14 | Gateway wiring | Read | `cmd/gateway/compose.go` | AnalyticalFamilyDeps field |
| 15 | Migration DDL | Schema | `deploy/migrations/{NNN}_{table}.sql` | Column definitions, TTL, partitioning |
| 16 | Writer config entry | Config | `deploy/configs/writer.jsonc` | Family name in pipeline array |
| 17 | Smoke test phase | Test | `scripts/smoke-analytical-e2e.sh` | Endpoint, required fields, filter tests |
| 18 | HTTP test file | Test | `tests/http/analytical.http` | Sample requests |

### Within-Layer vs New-Layer Artifacts

For **within-layer expansion** (e.g., EMA Crossover in Signals), many artifacts are shared:

| Artifact | Within-Layer (shared table) | New Layer (new table) |
|----------|:---------------------------:|:---------------------:|
| Migration DDL | ❌ Not needed (table exists) | ✅ Required |
| Writer mapper | ⚠️ Depends on event struct | ✅ Required |
| Writer pipeline entry | ✅ Required | ✅ Required |
| Writer consumer spec | ✅ Required | ✅ Required |
| Reader adapter | ❌ Already generic | ✅ Required |
| Use case | ❌ Already generic | ✅ Required |
| Handler | ❌ Already generic | ✅ Required |
| Route | ❌ Already registered | ✅ Required |
| Contracts | ❌ Already defined | ✅ Required |
| Gateway composition | ❌ Already wired | ✅ Required |

**Key insight**: within-layer expansion is primarily a write-path generation problem. The read-path is already generic for existing layers.

## Generation Approach

### Source of Truth

See companion document: `codegen-source-of-truth-artifact-coverage-and-ownership.md`.

### Generation Model: Template Expansion, Not Framework

The codegen tranche uses **template expansion** (static code generation from a declarative spec) rather than a runtime framework or reflection-based system. Rationale:

1. Generated code must be readable, reviewable, and editable.
2. Generated code must be structurally identical to hand-crafted code.
3. No runtime dependency on the codegen tool — generated files are standalone Go.
4. Templates are versioned alongside the codebase, not external tools.

### Template Categories

| Category | Templates | Input |
|----------|-----------|-------|
| Write-path | Consumer spec, pipeline entry, mapper, mapper tests | Family spec (event type, columns, field mappings) |
| Read-path (new layer only) | Reader, reader tests, use case, use case tests, handler, handler tests, route, contracts, gateway factory, gateway wiring | Family spec + layer definition |
| Schema (new layer only) | Migration DDL | Column definitions, partitioning strategy |
| Config | Writer config entry | Family name |
| Test | Smoke test phase, HTTP test requests | Endpoint, required fields, filters |

### Two-Tier Generation

**Tier 1 — Within-Layer Expansion (primary scope)**:
- Input: family spec (event type, NATS subject, durable name)
- Output: writer consumer spec + pipeline entry + config entry
- If event struct differs from existing mapper: also generates mapper + mapper tests
- Read-path artifacts: **none generated** (already generic)
- Validation: writer pipeline activates, data flows to ClickHouse, existing reader returns it

**Tier 2 — New-Layer Expansion (secondary scope, future)**:
- Input: full family spec (event type, columns, field mappings, filters)
- Output: all 18 artifacts
- Validation: full end-to-end equivalence with hand-crafted families
- Not required for EMA Crossover or other within-layer candidates

## Success Criteria for S193

The codegen implementation (S193) is considered successful when:

1. **Template coverage**: Templates exist for all Tier 1 artifacts (consumer spec, pipeline entry, mapper, mapper tests, config entry).
2. **Golden test equivalence**: Running templates against existing family specs produces output that is structurally equivalent to the hand-crafted code for those families.
3. **First generated family**: EMA Crossover (Tier 1) is generated and passes:
   - Compilation
   - Unit tests
   - Integration with existing smoke test infrastructure
4. **No framework**: Generated code has zero runtime dependency on the codegen tool.
5. **Reviewability**: Every generated file is human-readable and diff-reviewable.

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Templates diverge from hand-crafted patterns over time | Medium | High | Golden test equivalence against 6 families; CI drift detection |
| Codegen becomes a framework (runtime dependency) | Low | Critical | Hard rule: generated code must compile and run without codegen tool |
| Over-generation (generating artifacts that should stay manual) | Medium | Medium | Explicit ownership boundary document; generation scope frozen per tranche |
| Template complexity exceeds manual effort for rare edge cases | Low | Low | Edge cases remain manual; templates cover the 90% case |
| Codegen delays next family indefinitely | Medium | Medium | S193 targets EMA Crossover (Tier 1, minimal scope); bounded to write-path artifacts |

## Recommended Sequence

```
S192: Codegen Tranche Scoping       ← THIS STAGE
  │
  ├── S193: Codegen Implementation
  │   ├── Build Tier 1 templates (write-path)
  │   ├── Golden test: regenerate 6 existing families, compare output
  │   ├── Generate EMA Crossover writer artifacts
  │   └── Validate: compilation + unit tests
  │
  ├── S194: First Generated Family End-to-End Validation
  │   ├── EMA Crossover deployed in local compose
  │   ├── Writer persists data to ClickHouse
  │   ├── Existing SignalReader returns EMA crossover results
  │   ├── Smoke test extended with EMA crossover phase
  │   └── Ceiling metrics measured
  │
  └── S195: Codegen Gate Review
      ├── Template coverage assessment
      ├── Drift detection results
      ├── Cost comparison: manual vs generated
      └── Decision: Tier 2 scope authorization
```

## Companion Documents

| Document | Purpose |
|----------|---------|
| `codegen-source-of-truth-artifact-coverage-and-ownership.md` | Source of truth definition, artifact-level generation scope, ownership boundaries |
| `codegen-validation-drift-and-ci-strategy.md` | Equivalence validation, drift detection, CI integration approach |
| `codegen-anti-patterns-non-goals-and-human-decision-boundaries.md` | What codegen must NOT do, human decision boundaries, anti-patterns |
