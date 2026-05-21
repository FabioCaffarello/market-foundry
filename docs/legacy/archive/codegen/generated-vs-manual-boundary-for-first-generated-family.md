# Generated vs Manual Boundary — First Generated Family

**Stage**: S197
**Status**: DEFINITIVE
**Date**: 2026-03-20
**Context**: Defines precisely what the codegen engine owns and what remains human-authored for the first generated family iteration.

---

## 1. Boundary Summary

The first generated family uses codegen for **2 of 10+ artifacts** required for a complete analytical family. The boundary is drawn at the line where the engine has proven 100% structural equivalence (S196). Everything beyond that line is manual.

```
┌─────────────────────────────────────────────────────────┐
│                    GENERATED (A1 + A2)                  │
│                                                         │
│  Consumer spec function ← codegen/templates/            │
│  Pipeline entry struct  ← codegen/templates/            │
│                                                         │
│  Source: YAML spec → template → code fragment            │
│  Validation: golden snapshot comparison (CI-gated)       │
├─────────────────────────────────────────────────────────┤
│                    MANUAL (everything else)              │
│                                                         │
│  Domain event type         (domain package)             │
│  Mapper function           (cmd/writer/mappers.go)      │
│  Mapper unit tests         (cmd/writer/mappers_test.go) │
│  Config entry              (deploy/configs/writer.jsonc)│
│  Smoke test phase          (scripts/smoke-*.sh)         │
│  ClickHouse migration      (deploy/migrations/)         │
│  NATS stream definition    (if new stream)              │
│  File integration          (insert fragments into src)  │
│  Reader adapter            (if read path needed)        │
│  HTTP handler + route      (if read path needed)        │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Generated Artifacts — Detail

### A1: Consumer Spec Function

**What is generated**: A Go function that returns a `ConsumerSpec` struct defining the NATS durable consumer configuration for the family.

**Template**: `codegen/templates/consumer_spec.go.tmpl`

**Output location**: Code fragment printed to stdout; manually inserted into `internal/adapters/nats/{domain}_registry.go`

**Fields derived from spec**:
| Output field | Derived from |
|---|---|
| Function name | `Writer` + PascalCase(family.name) + PascalCase(family.layer) + `Consumer` |
| Durable name | `writer-` + family.layer + `-` + HyphenCase(family.name) |
| Subject | nats.subject (verbatim from spec) |
| Event type | nats.event_type (verbatim from spec) |
| Stream | nats.stream (verbatim from spec) |
| AckWait | Hardcoded: `30 * time.Second` |
| MaxDeliver | Hardcoded: `5` |

**Evidence layer exception**: Function name and durable name omit the layer component.

**Validation**: Golden snapshot comparison after S194 normalization. CI-gated.

### A2: Pipeline Entry Struct

**What is generated**: A Go struct literal that registers the family in the writer's pipeline configuration.

**Template**: `codegen/templates/pipeline_entry.go.tmpl`

**Output location**: Code fragment printed to stdout; manually inserted into `cmd/writer/pipeline.go`

**Fields derived from spec**:
| Output field | Derived from |
|---|---|
| family | family.name (verbatim) |
| consumerName | `writer-` + family.layer + `-` + HyphenCase(family.name) + `-consumer` |
| inserterName | `writer-` + family.layer + `-` + HyphenCase(family.name) + `-inserter` |
| table | writer.table (verbatim) |
| insertSQL | `INSERT INTO ` + writer.table |
| consumerSpec | Calls the A1 function |
| isEnabled | Lambda: `p.Is{Layer}FamilyEnabled("{family}")` |
| startConsumer | Closure: creates adapter consumer, registers event handler, calls mapper, sends to inserter |

**Evidence layer exception**: `isEnabled` uses `p.IsFamilyEnabled("{family}")` without layer prefix.

**Validation**: Golden snapshot comparison after S194 normalization. CI-gated.

---

## 3. Manual Artifacts — Detail

### A3: Mapper Function (MANUAL)

**Why not generated**: Mapper functions require column-order knowledge tied to the ClickHouse DDL. The `domain.columns` spec extension required for generation has not been implemented (S195 decision). Column order is structural — ClickHouse positional binding makes it load-bearing.

**What must be written**:
- Function signature: `func map{Layer}Row(event {domain}.{EventType}) []any`
- Ordered column extraction matching DDL column order
- Type transforms (JSON marshaling, enum conversion, parseFloat, etc.)
- Return: `[]any` slice in DDL column order

**Effort**: Medium — most complex manual artifact per family.

### A4: Mapper Unit Tests (MANUAL)

**Why not generated**: Depends on A3. Test structure requires knowledge of expected column values, transform behavior, and edge cases.

**What must be written**:
- Table-driven tests with representative event instances
- Assertions on column count, order, and transformed values
- Edge case coverage (nil fields, empty strings, zero values)

### A5: Config Entry (MANUAL)

**Why not generated**: Config files use JSONC (JSON with comments). No JSONC manipulation tooling exists in the codegen engine.

**What must be written**:
- Array entry in `deploy/configs/writer.jsonc` under the appropriate `families` section
- Fields: family name, enabled flag

**Effort**: Trivial — copy existing entry, change family name.

### A6: Smoke Test Phase (MANUAL)

**Why not generated**: Shell script generation introduces different template language concerns. Not worth implementing for a single test phase.

**What must be written**:
- Phase block in `scripts/smoke-analytical-e2e.sh`
- Publish test event, wait, query ClickHouse, assert row exists

**Effort**: Low — follows established pattern from existing 6 phases.

### Domain Event Type (MANUAL — always human-owned)

**Why never generated**: Domain types are architectural decisions. The three-condition boundary test (repetitive + mechanical + spec-derivable) fails — event types require domain modeling knowledge.

**What must be written**:
- Go struct in `internal/domain/{layer}/` package
- Fields specific to the family's domain semantics
- JSON tags for NATS deserialization

### ClickHouse Migration (MANUAL — always human-owned)

**Why never generated**: DDL is a schema design decision with durability and performance implications. Codegen does not own schema design (S193 D6).

**What must be written** (only if family targets a new table):
- SQL migration file in `deploy/migrations/`
- CREATE TABLE with MergeTree engine, partition key, order key
- Column definitions matching domain semantics

### File Integration (MANUAL — until marker sections implemented)

**Why not automated**: The engine produces code fragments, not complete files. File integration requires marker section detection and append/replace logic within source files (S195 D5 — explicitly deferred).

**What must be done**:
- Copy generated A1 output into the appropriate `_registry.go` file
- Copy generated A2 output into `cmd/writer/pipeline.go`
- Follow the same insertion patterns used by existing 6 families

---

## 4. Effort Breakdown Estimate

| Artifact | Generated? | Estimated Effort |
|---|:---:|---|
| A1: Consumer spec | Yes | ~0 min (generated) |
| A2: Pipeline entry | Yes | ~0 min (generated) |
| YAML spec authoring | — | ~5 min |
| Golden snapshot creation | — | ~5 min |
| A3: Mapper function | No | ~15 min |
| A4: Mapper tests | No | ~10 min |
| A5: Config entry | No | ~2 min |
| A6: Smoke test phase | No | ~5 min |
| Domain event type | No | ~5 min (if new) |
| File integration | No | ~5 min |
| **Total** | — | **~50 min** |

Without codegen: ~65 min (A1 + A2 take ~15 min manually based on S196 estimate).

**Savings per family**: ~15 min (~23% reduction), plus elimination of naming convention errors in A1 + A2.

---

## 5. Boundary Evolution Path

The boundary defined here is for the **first iteration only**. Future stages may shift it:

| Expansion | Prerequisite | Expected Stage |
|---|---|---|
| A3: Mapper generation | `domain.columns` spec extension, column-order DDL validation | Dedicated codegen expansion stage |
| A4: Mapper test generation | A3 implemented and validated | Same stage as A3 |
| A5: Config entry generation | JSONC tooling | Low priority — trivial manual effort |
| A6: Smoke test generation | Shell template engine | Low priority — low manual effort |
| File integration | Marker section implementation | High value — eliminates manual copy step |
| Tier 2 artifacts | Tier 1 production validation | Requires new authorization stage |

The boundary shifts only when equivalence is proven at the same standard as A1 + A2 (golden comparison, CI-gated, cross-family validation).
