# Analytical vs Generated: Ownership and Boundaries

> **Stage:** S214
> **Purpose:** Definitive reference for ownership zones, boundary rules, and integration contracts
> **Supersedes:** Fragmented ownership descriptions across S193–S204 docs

---

## 1. Three-Zone Ownership Model

Every artifact in the codebase belongs to exactly one of three zones:

### Zone 1: Human-Owned

Content authored and maintained exclusively by humans. Codegen never reads, writes, or validates these artifacts.

| Category | Examples |
|----------|----------|
| Domain event types | `internal/domain/{layer}/*.go` |
| ClickHouse migrations | `deploy/migrations/*.sql` |
| NATS stream/registry definitions | `internal/adapters/nats/*_registry.go` (struct + DefaultRegistry) |
| Writer core logic | `cmd/writer/consumer.go`, `inserter.go`, `supervisor.go`, `mappers.go` |
| Reader adapters | `internal/adapters/clickhouse/*_reader.go` |
| Use cases | `internal/application/analyticalclient/get_*_history.go` |
| HTTP handlers + routes | `internal/interfaces/http/handlers/*.go`, `routes/*.go` |
| Gateway composition | `cmd/gateway/compose.go`, `run.go`, `analytical_reader.go` |
| Migrate service | `cmd/migrate/main.go`, `internal/migrate/*.go` |
| Codegen templates | `codegen/templates/*.go.tmpl` (frozen — changes require authorization) |
| Codegen specs | `codegen/families/*.yaml` (human-authored source of truth) |
| Config files | `deploy/configs/*.jsonc` |
| Settings schema | `internal/shared/settings/schema.go` |
| CI workflows | `.github/workflows/*.yml` |
| Scripts | `scripts/*.sh` (including `codegen-integrated-check.sh`) |
| Store pipelines | `internal/actors/scopes/store/*.go` |

### Zone 2: Machine-Owned

Content produced by the codegen pipeline. Humans do not edit these artifacts directly.

| Category | Location |
|----------|----------|
| Golden snapshots | `codegen/golden-snapshots/{family}/{artifact}.go.golden` |
| Governed code fragments | Between `codegen:begin` / `codegen:end` markers in target files |

**Governance rule:** Content in Zone 2 must structurally match the golden snapshot. Drift triggers CI failure via `codegen-integrated-check.sh`.

**Caveat on golden snapshots for non-integrated families:** The 5 families (candle, rsi_oversold, mean_reversion_entry, position_exposure, paper_order) have golden snapshots that were hand-crafted as reference baselines. These are technically Zone 1 in origin but serve as Zone 2 validation targets for the codegen engine's `check-all` command. They are never regenerated.

### Zone 3: Mixed Files

Files containing both human-owned regions and machine-owned fragments.

| File | Human Regions | Machine Fragments |
|------|---------------|-------------------|
| `internal/adapters/nats/signal_registry.go` | Registry struct, DefaultSignalRegistry(), LatestSpecByType(), Store* functions | WriterRSISignalConsumer(), WriterEMASignalConsumer() (between codegen markers) |
| `cmd/writer/pipeline.go` | writerPipeline type, writerTrackerDef, declareWriterPipelines scaffold, evidence/decision/strategy/risk/execution entries | RSI and EMA pipeline entries (between codegen markers) |

**Rule:** Humans may freely edit content outside markers. Content inside markers is overwritten during regeneration.

---

## 2. Boundary Rules

### 2.1 The Three-Condition Test

An artifact is a candidate for codegen governance only if ALL three conditions hold:

1. **Repetitive** — same structure implemented 3+ times across families
2. **Mechanical** — zero creative or architectural decisions required
3. **Spec-derivable** — every value comes directly from the YAML spec, no inference

If any condition fails, the artifact stays manual. This is a hard gate.

### 2.2 Current Codegen Scope (Tier 1 Write-Path Only)

| Artifact ID | Description | Template | Target |
|-------------|-------------|----------|--------|
| A1 | Consumer spec function | `consumer_spec.go.tmpl` | `internal/adapters/nats/{layer}_registry.go` |
| A2 | Pipeline entry struct | `pipeline_entry.go.tmpl` | `cmd/writer/pipeline.go` |

**Not in scope (explicitly deferred):**

| Artifact ID | Description | Blocker |
|-------------|-------------|---------|
| A3 | Mapper function | Requires `domain.columns` spec extension |
| A4 | Mapper tests | Depends on A3 |
| A5 | Config entries | JSONC tooling gap |
| A6 | Smoke test phases | Shell template complexity |
| Tier 2 | Readers, handlers, routes, use cases | Requires dedicated stage |
| Store path | Projection + consumer actor pairs | Different actor pattern |

### 2.3 Integration Protocol

1. **Markers placed manually** — codegen never creates new marker pairs
2. **Content between markers is fully codegen-owned** — regeneration replaces all
3. **Content outside markers is fully human-owned** — codegen never touches
4. **One marker pair per family×artifact** — no nesting
5. **Manifest is the authority** — `codegen/integrated.yaml` lists every governed slice

### 2.4 Marker Format (Canonical)

```go
// codegen:begin <artifact_type> family=<family_name> source=<spec_path>
... governed code ...
// codegen:end <artifact_type> family=<family_name>
```

Manual sections use:
```go
// ── Writer Consumer Specs (manual:owned) ─────────────────────────
// Ownership: human-maintained. Not codegen-governed.
```

---

## 3. Analytical Path Boundaries

The analytical path is entirely Zone 1 (human-owned). It has no codegen representation.

### 3.1 Write Path (Writer Service)

| Component | Owner | Codegen Status |
|-----------|-------|----------------|
| Pipeline declarations | Mixed (Zone 3) | RSI + EMA governed; rest manual |
| Consumer actor logic | Human | Never generated |
| Inserter actor logic | Human | Never generated |
| Supervisor + recovery | Human | Never generated |
| Row mappers | Human | Deferred (A3 blocker) |
| Mapper tests | Human | Deferred (A4 blocker) |

### 3.2 Schema Path (Migrations)

| Component | Owner |
|-----------|-------|
| Migration SQL files | Human |
| Migration runner | Human |
| Catalog discovery | Human |
| Checksum validation | Human |

No codegen interaction. Schema changes require architectural decisions (partitioning, TTL, column order).

### 3.3 Read Path (Gateway)

| Component | Owner |
|-----------|-------|
| ClickHouse readers | Human |
| Query builder | Human |
| Use cases | Human |
| HTTP handlers | Human |
| Routes | Human |
| Gateway composition | Human |

Entirely manual. The read path is structurally different from the write path (query builder pattern vs. consumer-inserter pattern), making it unsuitable for current templates.

### 3.4 Operational Path (Store)

| Component | Owner |
|-----------|-------|
| Store supervisor | Human |
| All 13 pipelines | Human |
| Projection actors | Human |
| Consumer actors | Human |
| KV bucket definitions | Human |

Entirely manual. The store actor pattern (projection + consumer pairs with KV bucket wiring) is structurally different from the writer pipeline pattern.

---

## 4. Cross-Boundary Interaction Points

The analytical path and generated path interact at exactly two files:

1. **`internal/adapters/nats/{layer}_registry.go`** — Writer consumer spec functions
   - Generated: RSI, EMA (signal layer only)
   - Manual: all other layers + all store consumer specs

2. **`cmd/writer/pipeline.go`** — Pipeline entry structs
   - Generated: RSI, EMA entries
   - Manual: candle, rsi_oversold, mean_reversion_entry, position_exposure, paper_order entries

No other files are shared between the two paths. The analytical read path (ClickHouse readers, use cases, handlers) has zero overlap with codegen.

---

## 5. CI Validation Chain

```
codegen validate-all     → cross-spec uniqueness (names, durables, subjects)
codegen check-all        → spec → golden structural equivalence (all 7 families × 2 artifacts)
codegen test             → engine unit tests
codegen-integrated-check → golden → target match (4 governed slices only)
unit-tests               → Go compilation + logic
smoke-analytical         → runtime end-to-end
```

**Key distinction:** `check-all` validates ALL 7 families (including non-integrated ones). `codegen-integrated-check` validates only the 4 slices listed in `integrated.yaml`.

---

## 6. Spec Schema (Frozen)

14-field schema per `codegen/spec.go`:

```yaml
family:  { name, layer, tier }
nats:    { subject, event_type, stream, durable }
writer:  { table, mapper, pipeline_family_key, config_array }
domain:  { event_package, event_type }
```

Schema changes require a new architectural stage. Value changes within the frozen schema do not.

---

## 7. Decision Record

| Decision | Rationale |
|----------|-----------|
| Only A1+A2 are generated | Three-condition test; mappers require domain knowledge |
| Store path stays manual | Different actor pattern; projection closures not spec-derivable |
| Read path stays manual | Structurally different; Tier 2 not authorized |
| Evidence layer has naming exception | Historical convention; encoded in `spec.Derived()` |
| Golden snapshots for 5 non-integrated families are reference artifacts | Prove template correctness; not deployment targets |
| `manual:owned` markers added in S214 | Visual clarity for ownership in mixed files |
