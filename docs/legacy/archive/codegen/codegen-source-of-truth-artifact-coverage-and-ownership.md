# Codegen — Source of Truth, Artifact Coverage, and Ownership

## Purpose

This document defines the single source of truth for analytical family code generation, enumerates exactly which artifacts are generated vs manual, and establishes ownership boundaries that prevent boundary collapse.

## Source of Truth

### Family Specification File

The source of truth for codegen is a **family specification file** — a declarative YAML (or equivalent) file that describes a single analytical family. One file per family. No inheritance, no composition, no indirection.

```
codegen/families/{family_name}.yaml
```

### Specification Shape

A family spec contains the following fields, derived from evidence across 6 hand-crafted families:

```yaml
family:
  name: "rsi"                              # unique identifier
  layer: "signal"                          # L1-L6 layer name
  tier: 1                                  # 1 = within-layer, 2 = new layer

# NATS consumer identity
nats:
  subject: "signal.events.rsi.generated.>"
  event_type: "signal.events.v1.rsi_generated"
  stream: "SIGNAL_EVENTS"
  durable: "writer-signal-rsi"

# Write-path
writer:
  table: "signals"                         # existing ClickHouse table
  mapper: "mapSignalRow"                   # existing mapper (shared) or "generate"
  pipeline_family_key: "rsi"               # key in writer config families array
  config_array: "signal_families"          # config field in writer.jsonc

# Domain type reference (for mapper generation)
domain:
  event_package: "signal"
  event_type: "SignalGeneratedEvent"
  columns:                                 # only needed if mapper = "generate"
    - { name: "event_id", go_field: "EventID", ch_type: "UUID" }
    - { name: "value", go_field: "Value", ch_type: "Float64", transform: "parseFloat" }
    # ... (full column list per DDL)
```

### Why YAML, Not Go Structs

- The spec must be readable by non-Go tooling (CI scripts, documentation generators, drift checkers).
- The spec must be diffable and reviewable in PR context.
- Go structs would create a runtime dependency between the spec and generated code.
- YAML is the lingua franca of declarative config in the Go ecosystem.

### Spec Validation Rules

1. `family.name` must be unique across all spec files.
2. `nats.durable` must follow the pattern `writer-{layer}-{family}` or `writer-{family}`.
3. `writer.table` must reference an existing migration.
4. `writer.mapper` must be either a known existing function name or `"generate"`.
5. `domain.event_type` must exist in the codebase.
6. If `tier: 2`, the spec must include `schema` section (migration DDL definition).

## Artifact Coverage

### Tier 1 Artifacts (Within-Layer Expansion)

| # | Artifact | Generated? | Location | Rationale |
|---|----------|:----------:|----------|-----------|
| 1 | Writer consumer spec function | **YES** | `internal/adapters/nats/{domain}_registry.go` | Mechanical: subject + durable + stream → ConsumerSpec |
| 2 | Writer pipeline entry | **YES** | `cmd/writer/pipeline.go` | Mechanical: family + table + SQL + consumer + mapper binding |
| 3 | Writer mapper function | **CONDITIONAL** | `cmd/writer/mappers.go` | Only if event struct differs from existing mapper; otherwise reuse |
| 4 | Writer mapper tests | **CONDITIONAL** | `cmd/writer/mappers_test.go` | Only if mapper is generated |
| 5 | Writer config entry | **YES** | `deploy/configs/writer.jsonc` | Append family name to config array |
| 6 | Smoke test phase | **YES** | `scripts/smoke-analytical-e2e.sh` | Endpoint + required fields + filter test assertions |

**Not generated (Tier 1)**:
- Reader adapter: already generic (type-parameterized)
- Use case: already generic
- Handler: already generic
- Route: already registered
- Contracts: already defined
- Gateway composition: already wired
- Migration DDL: table already exists

### Tier 2 Artifacts (New-Layer Expansion)

| # | Artifact | Generated? | Location | Rationale |
|---|----------|:----------:|----------|-----------|
| 1–6 | All Tier 1 artifacts | **YES** | (same as above) | Write-path always needed |
| 7 | Migration DDL | **YES** | `deploy/migrations/{NNN}_{table}.sql` | Column definitions from spec |
| 8 | Reader adapter | **YES** | `internal/adapters/clickhouse/{family}_reader.go` | Build query, scan rows, return domain types |
| 9 | Reader adapter tests | **YES** | `internal/adapters/clickhouse/{family}_reader_test.go` | Query building + filter tests |
| 10 | Use case | **YES** | `internal/application/analyticalclient/get_{family}_history.go` | Validation + reader dispatch |
| 11 | Use case tests | **YES** | `internal/application/analyticalclient/get_{family}_history_test.go` | Validation matrix |
| 12 | Contracts | **YES** | `internal/application/analyticalclient/contracts.go` | Query/Reply struct additions |
| 13 | Handler method | **YES** | `internal/interfaces/http/handlers/analytical.go` | Param extraction + use case call |
| 14 | Handler tests | **YES** | `internal/interfaces/http/handlers/analytical_test.go` | HTTP assertions |
| 15 | Route registration | **YES** | `internal/interfaces/http/routes/analytical.go` | Path + handler binding |
| 16 | Gateway reader factory | **YES** | `cmd/gateway/analytical_reader.go` | NewXReader wrapper |
| 17 | Gateway wiring | **YES** | `cmd/gateway/compose.go` | AnalyticalFamilyDeps field |

### Artifacts That Are NEVER Generated

| Artifact | Reason |
|----------|--------|
| Domain event types (`internal/domain/`) | Domain types are architectural decisions, not template output |
| NATS stream definitions | Stream creation is infrastructure, not family-specific |
| Writer core logic (consumer.go, inserter.go, supervisor.go) | Framework code, not per-family |
| ClickHouse client | Infrastructure adapter |
| Health/observability framework | Shared infrastructure |
| Gateway `compose.go` core logic | Composition root is architectural |
| HTTP server/router setup | Infrastructure |
| Shared helpers (`parseFloat`, `marshalJSON`, `parseAnalyticalParams`) | Shared utilities, not per-family |

## Ownership Boundaries

### Boundary Model

```
┌─────────────────────────────────────────────────────┐
│  HUMAN-OWNED (never generated)                       │
│                                                      │
│  Domain types, stream definitions, infrastructure,   │
│  composition root logic, shared helpers, CI config   │
├─────────────────────────────────────────────────────┤
│  CODEGEN-OWNED (generated, validated, replaceable)   │
│                                                      │
│  Consumer specs, pipeline entries, mappers (when     │
│  new), mapper tests, config entries, smoke phases    │
│  Reader/handler/route/use-case (Tier 2 only)         │
├─────────────────────────────────────────────────────┤
│  SPEC-OWNED (source of truth)                        │
│                                                      │
│  codegen/families/*.yaml                             │
│  Templates in codegen/templates/                     │
└─────────────────────────────────────────────────────┘
```

### Ownership Rules

1. **Spec files are the single source of truth.** If a generated file and a spec file disagree, the spec wins and the generated file is regenerated.

2. **Generated files carry a header comment** marking them as codegen output:
   ```go
   // Code generated by mf-codegen from codegen/families/ema_crossover.yaml. DO NOT EDIT.
   ```

3. **Generated files must not be manually edited.** If a fix is needed, the fix goes into the template or the spec, and the file is regenerated.

4. **Human-owned files are never overwritten by codegen.** Codegen may append to human-owned files (e.g., adding a pipeline entry to `pipeline.go`) but must do so via clearly marked, append-only sections.

5. **Templates are human-owned.** Templates are reviewed, versioned, and modified by architects. They are not self-modifying.

### Append-Only Integration Points

Some generated artifacts must be integrated into existing human-owned files. These integration points use **append-only marker sections**:

```go
// --- codegen:pipeline-entries:start ---
// (generated pipeline entries appear here)
// --- codegen:pipeline-entries:end ---
```

Integration points:
- `cmd/writer/pipeline.go` → pipeline entry declarations
- `cmd/writer/mappers.go` → new mapper functions (if needed)
- `deploy/configs/writer.jsonc` → family array entries
- `internal/adapters/nats/{domain}_registry.go` → consumer spec functions

### What Happens to the 6 Existing Families

The 6 hand-crafted families remain **as-is**. They are not retroactively generated. Rationale:

1. They are proven, tested, and deployed.
2. Regenerating them risks introducing regressions with zero benefit.
3. They serve as the **golden reference** for template validation.

The codegen tool must be able to produce output that is structurally equivalent to these 6 families when given equivalent spec files. This is the primary validation mechanism (see companion document on validation).

### Specification Evidence From 6 Families

| Family | Layer | Spec Evidence Provided |
|--------|-------|----------------------|
| Candle (baseline) | L1 Evidence | 16 columns, OHLCV floats, no JSON, no optional filters |
| RSI | L2 Signal | 12 columns, 1 JSON (metadata), no optional filters |
| RSI Oversold | L3 Decision | 14 columns, 2 JSON (signals, metadata), 1 enum (outcome) |
| Mean Reversion Entry | L4 Strategy | 15 columns, 3 JSON (decisions, parameters, metadata), 1 enum (direction) |
| Position Exposure | L5 Risk | 17 columns, 4 JSON (strategies, constraints, parameters, metadata), 1 enum (disposition), 1 text (rationale) |
| Paper Order | L6 Execution | 20 columns, 4 JSON (risk, fills, parameters, metadata), 2 enums (side, status), 2 correlation IDs |

This covers all column types (UUID, String, LowCardinality, Float64, UInt32, DateTime64, Bool), all JSON complexities (0–4 JSON columns, map/slice/struct targets), all enum patterns, and all filter types used in the platform.
