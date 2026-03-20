# Spec, Templates, Generated Outputs, and Runtime Boundaries

> **Stage:** S199
> **Purpose:** Define the ownership model, data flow, and runtime separation between spec files, templates, generated outputs, and the live monorepo

---

## 1. Overview

The codegen system has four distinct artifact categories. Each has different ownership, mutability rules, and relationships to the running system. This document makes those boundaries explicit to prevent ambiguity as generated families enter the monorepo.

```
YAML Spec ──→ Template ──→ Generated Output ──→ Runtime Code
 (human)       (human)       (machine)           (mixed)
```

---

## 2. Artifact Categories

### 2.1 Spec Files (`codegen/families/*.yaml`)

| Property | Value |
|----------|-------|
| **Owner** | Human (architect/developer) |
| **Location** | `codegen/families/{family_name}.yaml` |
| **Schema** | 14-field frozen schema (S193) |
| **Mutability** | Mutable by humans only; changes trigger regeneration |
| **Review** | Standard PR review; treated as architectural decisions |
| **CI role** | Validated by `codegen validate` (schema + invariants) |

**What a spec declares:**
- Family identity (name, layer, tier)
- NATS wiring (subject, event type, stream, durable consumer name)
- Writer wiring (table, mapper function name, pipeline key, config array)
- Domain binding (event package, event type struct name)

**What a spec does NOT declare:**
- Domain type field shapes or column types
- ClickHouse DDL or schema design
- Mapper transformation logic
- Reader query logic
- HTTP handler or routing
- Business rules or thresholds

**Invariants:**
- Every value in generated output must trace to a spec field or a deterministic derivation
- No inference, heuristics, or defaults that are not explicitly defined in the derivation rules
- Spec validation is fail-fast: schema → naming → uniqueness → referential → structural

### 2.2 Templates (`codegen/templates/*.go.tmpl`)

| Property | Value |
|----------|-------|
| **Owner** | Human (architect) |
| **Location** | `codegen/templates/{artifact_name}.go.tmpl` |
| **Current set** | `consumer_spec.go.tmpl`, `pipeline_entry.go.tmpl` |
| **Mutability** | Frozen for this phase (S198 condition 8) |
| **Review** | Requires dedicated stage for any modification |
| **CI role** | Changes trigger `codegen-golden` job (full regression) |

**What templates define:**
- The structural shape of generated Go code
- Placeholder slots filled by spec fields and derived values
- Comment patterns and formatting conventions
- Import requirements (implicitly, via generated code shape)

**What templates do NOT define:**
- Which families exist or their configuration
- Runtime behavior or business logic
- Infrastructure setup or wiring beyond the generated fragment

**Template freeze rules:**
- No template modification is permitted without a new authorization stage
- Template modifications require re-validation of all existing golden snapshots
- A template change that breaks any golden comparison is a blocking defect

### 2.3 Generated Outputs

| Property | Value |
|----------|-------|
| **Owner** | Machine (codegen engine); immutable after generation |
| **Location (golden)** | `codegen/golden-snapshots/{family}/{artifact}.go.golden` |
| **Location (runtime)** | Inserted into target files (see Section 3) |
| **Mutability** | NEVER manually edited after generation |
| **Review** | Verified by golden comparison; reviewed for correct placement |
| **CI role** | `codegen-check` validates all golden snapshots |

**Generated output lifecycle:**
1. Engine reads spec + template
2. Engine computes derived fields (deterministic naming conventions)
3. Engine renders template with spec data + derived fields
4. Output is captured as golden snapshot AND inserted into target file
5. CI verifies golden snapshot matches regenerated output on every PR

**Immutability rule:**
If a generated fragment needs modification, the change must come from:
1. The spec (if the issue is a configuration value), OR
2. The template (if the issue is structural — requires new stage), OR
3. The derivation logic (if the issue is a naming convention — requires engine fix + golden refresh)

Manual editing of generated output is a **revocation trigger** (S198).

### 2.4 Runtime Code (Target Files)

| Property | Value |
|----------|-------|
| **Owner** | Mixed — file is human-owned; generated fragments are machine-owned |
| **Mutability** | Human-owned sections freely editable; generated fragments immutable |
| **Review** | Standard PR review; reviewer must verify generated fragments are unmodified |

**Target files and their generated regions:**

| Target File | Generated Content | Manual Content |
|-------------|------------------|----------------|
| `internal/adapters/nats/{layer}_registry.go` | A1: consumer spec function | Existing functions, imports, package declaration |
| `cmd/writer/pipeline.go` | A2: pipeline entry in return slice | Function signature, other entries, imports |

---

## 3. Data Flow

```
                    ┌─────────────────────────┐
                    │   codegen/families/      │
                    │   {family}.yaml          │  ← Human authors
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   codegen validate       │  ← CI: codegen-lint
                    │   (schema + invariants)  │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   codegen generate       │
                    │   (spec + template →     │
                    │    rendered fragment)     │
                    └──────┬─────────┬────────┘
                           │         │
              ┌────────────▼──┐  ┌───▼───────────────┐
              │ Golden snapshot│  │ Fragment stdout    │
              │ (.go.golden)  │  │ (developer copies) │
              └───────┬───────┘  └────────┬──────────┘
                      │                   │
         ┌────────────▼──────┐  ┌─────────▼──────────┐
         │ CI: codegen-check │  │ Target runtime file │
         │ (regenerate +     │  │ (mixed ownership)   │
         │  compare)         │  │                     │
         └───────────────────┘  └─────────┬──────────┘
                                          │
                               ┌──────────▼──────────┐
                               │ CI: compile + test   │
                               │ + smoke-analytical   │
                               └──────────────────────┘
```

---

## 4. Ownership Boundaries

### 4.1 Three-Tier Ownership Model

| Tier | Owner | Artifacts | Edit Authority |
|------|-------|-----------|---------------|
| **Human-only** | Developer/Architect | Domain types, migrations, stream defs, infrastructure, templates, specs | Unrestricted within review |
| **Machine-only** | Codegen engine | Generated fragments (A1, A2), golden snapshots | Never manually edited |
| **Mixed** | Human file + machine fragments | Target files (registry, pipeline) | Human edits allowed outside generated regions |

### 4.2 Ownership Decision Rules

To determine whether an artifact belongs to the generated path:

1. **Is it repetitive?** Must appear 3+ times across families with identical structure
2. **Is it mechanical?** Must require zero creative decisions — pure spec-to-code transformation
3. **Is it spec-derivable?** Every value in the output must trace to a spec field or deterministic derivation

All three conditions must be true. If any is false, the artifact stays human-owned.

### 4.3 Evidence Layer Exception

The evidence layer has naming conventions that differ from all other layers:
- Consumer names omit the layer prefix: `writer-candle-consumer` (not `writer-evidence-candle-consumer`)
- Function names omit the layer: `WriterCandleConsumer` (not `WriterCandleEvidenceConsumer`)
- IsEnabled method uses generic form: `IsFamilyEnabled("candle")` (not `IsEvidenceFamilyEnabled("candle")`)

This exception is encoded in the derivation logic (`Derived()` in `spec.go`) and is automatically applied when `layer: evidence`. No manual handling required.

---

## 5. Runtime Boundaries

### 5.1 Codegen Module is Build-Time Only

The `codegen/` module:
- Has its own `go.mod` (isolated dependency graph)
- Is NOT imported by any runtime module (writer, gateway, domain, shared)
- Could be deleted entirely without affecting any running service
- Produces text output only — no runtime library, no shared types, no framework

### 5.2 Generated Code is Self-Contained

Generated fragments:
- Use only types already defined in the target file's package
- Reference only functions/types from existing imports (e.g., `adapternats.WriterXConsumer()`, `settings.PipelineConfig`)
- Do not introduce new dependencies
- Compile as part of the target file — not as separate files

### 5.3 No Codegen Runtime Dependency

```
codegen/                    │  cmd/writer/          cmd/gateway/
  families/*.yaml           │    pipeline.go         compose.go
  templates/*.tmpl          │    mappers.go          analytical_reader.go
  golden-snapshots/         │    consumer.go
  main.go                   │    ...
  spec.go                   │
  render.go                 │
  compare.go                │
────────────────────────────┼──────────────────────────────────────
  BUILD-TIME ONLY           │  RUNTIME
  (no import relationship)  │  (runs in production)
```

### 5.4 Generated Fragments in Runtime Context

When a generated A1 (consumer spec) is inserted into a registry file:
- It becomes part of the `adapternats` package
- It is called by `cmd/writer/pipeline.go` like any other consumer spec function
- The writer, gateway, and all other services are unaware that it was generated
- There is no runtime distinction between manual and generated consumer specs

This is by design: **generated code must be indistinguishable from hand-crafted code at runtime**.

---

## 6. Relationship Between Artifacts

```
Spec Field                    Template Slot              Generated Output           Runtime Usage
─────────────────────────────────────────────────────────────────────────────────────────────────
family.name: "rsi"       →   {{.Spec.Family.Name}}  →   family: "rsi"          →   pipeline config key
family.layer: "signal"   →   {{.Derived.PascalLayer}} → WriterRSISignalConsumer →  function name
nats.subject: "..."      →   {{.Spec.NATS.Subject}}  →  Subject: "..."         →   NATS subscription
nats.durable: "..."      →   {{.Spec.NATS.Durable}}  →  Durable: "..."         →   consumer identity
writer.table: "signals"  →   {{.Spec.Writer.Table}}  →  table: "signals"       →   ClickHouse target
writer.mapper: "mapX"    →   {{.Spec.Writer.Mapper}}  → mapSignalRow(event)    →   row transformation
domain.event_type: "..." →   {{.Spec.Domain.EventType}} → signal.SignalGenerated → Go type assertion
```

Every generated value is traceable: `spec field → template slot → output line → runtime usage`. No magic, no inference, no hidden defaults.

---

## 7. What This Document Does NOT Authorize

- Expanding generated artifacts beyond A1+A2
- Modifying templates
- Retroactively converting manual families
- Generating read-path artifacts (Tier 2)
- Generating cross-service code
- Implementing marker-based file integration
- Generating mapper functions (A3)

Each of these requires a separate authorization stage with its own evidence, gate conditions, and scope limits.
