# Codegen Specification and Schema

> **Consolidated document.** Merges content from:
> - `codegen-specification-freeze.md` (S193)
> - `codegen-spec-schema-fields-invariants-and-ownership.md`
> - `codegen-source-of-truth-artifact-coverage-and-ownership.md`
> - `codegen-equivalence-scope-semantic-vs-structural-rules.md` (S194)
>
> Originals archived to `docs/archive/codegen/`.

---

## 1. Freeze Status

**Status: FROZEN** -- this specification is the authoritative contract for all codegen behavior in Market Foundry. Effective from S193. Changes require a new stage with explicit architectural review.

---

## 2. Frozen Decisions

### D1 -- Single Source of Truth

The **sole** source of truth for any generated analytical family is a YAML specification file at:

```
codegen/families/{family_name}.yaml
```

One file per family. No family may be generated without a corresponding spec file. No spec file may exist without producing at least one artifact.

**Why YAML**: readable by non-Go tooling (CI, docs, drift checkers), diffable in PRs, no runtime dependency between spec and generated code.

### D2 -- Two-Tier Generation Model

| Tier | Scope | Artifacts | Activation |
|------|-------|-----------|------------|
| **Tier 1** | Within-layer write-path | 6 | Immediate (S193+) |
| **Tier 2** | Full new-layer (read + write) | 17 | Deferred until Tier 1 proven in production |

Tier 2 is not authorized until Tier 1 has been validated through at least one generated family passing end-to-end smoke tests and gate review.

### D3 -- Template Expansion, Not Runtime Framework

Generated code is standalone Go source. Zero runtime dependency on codegen tooling. Any generated file must compile and function identically if the codegen tool were deleted from the repository.

### D4 -- Golden Test Equivalence

Codegen correctness is validated by regenerating specs for the 6 existing hand-crafted families and comparing structural equivalence. This is the primary validation mechanism.

### D5 -- CI Verifies, Does Not Generate

Generated files are committed to the repository. CI checks that committed files match codegen output. CI never produces build artifacts via codegen.

### D6 -- Existing Families Are Immutable Golden References

The 6 hand-crafted families (candle, rsi, rsi_oversold, mean_reversion, position_exposure, paper_order) are never retroactively regenerated. They serve as golden references for template validation.

---

## 3. Canonical Spec Schema

Every family spec file (`codegen/families/{family_name}.yaml`) must conform exactly to this shape:

```yaml
# -- Identity -----------------------------------------------
family:
  name: "<string>"          # REQUIRED -- unique family identifier (snake_case)
  layer: "<string>"         # REQUIRED -- L1-L6 layer name (evidence|signal|decision|strategy|risk|execution)
  tier: <integer>           # REQUIRED -- 1 (within-layer) or 2 (new-layer)

# -- NATS Binding -------------------------------------------
nats:
  subject: "<string>"       # REQUIRED -- NATS subject filter
  event_type: "<string>"    # REQUIRED -- fully qualified event type identifier
  stream: "<string>"        # REQUIRED -- NATS JetStream stream name
  durable: "<string>"       # REQUIRED -- durable consumer name

# -- Writer Binding -----------------------------------------
writer:
  table: "<string>"         # REQUIRED -- target ClickHouse table name
  mapper: "<string>"        # REQUIRED -- existing mapper function name OR "generate"
  pipeline_family_key: "<string>"  # REQUIRED -- family key for pipeline registration
  config_array: "<string>"  # REQUIRED -- writer config array to append to

# -- Domain Binding -----------------------------------------
domain:
  event_package: "<string>" # REQUIRED -- Go package containing event type
  event_type: "<string>"    # REQUIRED -- Go type name of the event struct
  columns:                  # CONDITIONAL -- required only when mapper: "generate"
    - name: "<string>"      #   REQUIRED per entry -- ClickHouse column name
      go_field: "<string>"  #   REQUIRED per entry -- Go struct field name
      ch_type: "<string>"   #   REQUIRED per entry -- ClickHouse column type
      transform: "<string>" #   OPTIONAL per entry -- transformation function name

# -- Tier 2 Only --------------------------------------------
schema:                     # CONDITIONAL -- required only when tier: 2
  migration_number: <integer>  # REQUIRED -- migration sequence number
  ddl: "<string>"           # REQUIRED -- CREATE TABLE IF NOT EXISTS statement
  reverse_ddl: "<string>"   # REQUIRED -- DROP TABLE IF EXISTS statement
```

### Field Reference -- Required Fields

| Section | Field | Type | Constraints | Purpose |
|---------|-------|------|-------------|---------|
| `family` | `name` | string | Unique across all specs; snake_case; `[a-z][a-z0-9_]*` | Primary identifier |
| `family` | `layer` | string | One of: `evidence`, `signal`, `decision`, `strategy`, `risk`, `execution` | L1-L6 layer |
| `family` | `tier` | integer | `1` or `2` | Determines artifact scope |
| `nats` | `subject` | string | Valid NATS subject with `>` wildcard | Subscription filter |
| `nats` | `event_type` | string | Dot-separated, versioned: `{layer}.events.v{N}.{name}` | Event discriminator |
| `nats` | `stream` | string | UPPER_SNAKE_CASE; must reference existing stream | JetStream stream |
| `nats` | `durable` | string | Pattern: `writer-{layer}-{family}` or `writer-{family}` | Durable consumer name |
| `writer` | `table` | string | Must reference table created by existing migration | Target ClickHouse table |
| `writer` | `mapper` | string | Existing Go function name or literal `"generate"` | Mapper reference |
| `writer` | `pipeline_family_key` | string | snake_case; unique across all specs | Pipeline registration key |
| `writer` | `config_array` | string | Must reference existing config array in `writer.jsonc` | Config injection target |
| `domain` | `event_package` | string | Valid Go package name; must exist in codebase | Event struct package |
| `domain` | `event_type` | string | Valid Go exported type name; must exist in codebase | Event struct type |

### Conditional Fields

| Section | Field | Condition | Constraints |
|---------|-------|-----------|-------------|
| `domain` | `columns` | Required when `writer.mapper` = `"generate"` | Non-empty array; each entry has `name`, `go_field`, `ch_type` |
| `domain.columns[]` | `transform` | Optional per column entry | Must reference known transform function |
| `schema` | (entire section) | Required when `family.tier` = `2` | Must contain `migration_number`, `ddl`, `reverse_ddl` |

### Invalid Fields (Rejected by Validation)

| Prohibited Field | Reason |
|------------------|--------|
| `reader.*` | Tier 2 scope only |
| `handler.*`, `route.*` | Tier 2 scope only |
| `retry.*` | Infrastructure concern, not per-family |
| `observability.*` | Shared infrastructure |
| `ttl`, `partitioning` | Schema-level, defined in DDL migration |
| `version` | Versioned via git |
| `description`, `tags` | Specs are declarative, not documentary |
| `depends_on` | No dependency resolution; families are independent |

---

## 4. Validation Invariants

Every spec file must pass all invariants before codegen execution. Violation is a hard error.

### Uniqueness Invariants

| Invariant | Scope |
|-----------|-------|
| `family.name` is unique | Across all `codegen/families/*.yaml` |
| `nats.durable` is unique | Across all specs |
| `writer.pipeline_family_key` is unique | Across all specs |

### Referential Integrity Invariants

| Invariant | Verification |
|-----------|-------------|
| `writer.table` references existing migration | Scan `deploy/migrations/*.sql` for matching `CREATE TABLE` |
| `domain.event_type` exists in `domain.event_package` | Parse Go source in `internal/domain/{event_package}/` |
| `nats.stream` references defined JetStream stream | Cross-reference NATS adapter |
| `writer.mapper` (when not `"generate"`) references existing function | Scan `cmd/writer/mappers.go` |
| `writer.config_array` references existing array | Parse `deploy/configs/writer.jsonc` |

### Structural Invariants

| Invariant | Rule |
|-----------|------|
| Tier 1 must NOT contain `schema` section | `tier: 1` + `schema:` present = invalid |
| Tier 2 MUST contain `schema` section | `tier: 2` + `schema:` absent = invalid |
| `mapper: "generate"` requires `domain.columns` | Missing = invalid |
| Named mapper must NOT have `domain.columns` | Present = invalid |
| Column names must match DDL column names | Validated against migration DDL |
| Column order must match DDL column order | Positional binding requirement |

### Naming Pattern Invariants

| Field | Pattern | Example |
|-------|---------|---------|
| `family.name` | `[a-z][a-z0-9_]*` | `ema_crossover` |
| `nats.durable` | `writer-{layer}-{family}` or `writer-{family}` | `writer-signal-ema_crossover` |
| `nats.event_type` | `{layer}.events.v{N}.{name}` | `signal.events.v1.ema_crossover_generated` |
| `nats.stream` | `[A-Z][A-Z0-9_]*` | `SIGNAL_EVENTS` |

### Validation Execution Order (fail-fast)

1. Schema conformance -- all required fields present, no unknown fields, correct types
2. Naming pattern compliance
3. Uniqueness checks -- no collisions with other specs
4. Referential integrity -- all references resolve
5. Structural invariants -- tier/mapper/column consistency
6. Column alignment -- columns match DDL (when applicable)

---

## 5. Artifact Coverage

### Tier 1 -- Generated Artifacts (6)

| # | Artifact | Condition | Target File |
|---|----------|-----------|-------------|
| 1 | Writer consumer spec | Always | `internal/adapters/nats/nats{domain}/registry.go` |
| 2 | Writer pipeline entry | Always | `cmd/writer/pipeline.go` |
| 3 | Writer mapper | When `mapper: "generate"` | `cmd/writer/mappers.go` |
| 4 | Writer mapper tests | When mapper generated | `cmd/writer/mappers_test.go` |
| 5 | Writer config entry | Always | `deploy/configs/writer.jsonc` |
| 6 | Smoke test phase | Always | `scripts/smoke-analytical-e2e.sh` |

### Tier 2 -- Generated Artifacts (17, not yet authorized)

| # | Artifact | Location |
|---|----------|----------|
| 1-6 | All Tier 1 artifacts | (same as above) |
| 7 | Migration DDL | `deploy/migrations/{NNN}_{table}.sql` |
| 8 | Reader adapter | `internal/adapters/clickhouse/{family}_reader.go` |
| 9 | Reader adapter tests | `internal/adapters/clickhouse/{family}_reader_test.go` |
| 10 | Use case | `internal/application/analyticalclient/get_{family}_history.go` |
| 11 | Use case tests | `internal/application/analyticalclient/get_{family}_history_test.go` |
| 12 | Contracts | `internal/application/analyticalclient/contracts.go` |
| 13 | Handler method | `internal/interfaces/http/handlers/analytical.go` |
| 14 | Handler tests | `internal/interfaces/http/handlers/analytical_test.go` |
| 15 | Route registration | `internal/interfaces/http/routes/analytical.go` |
| 16 | Gateway reader factory | `cmd/gateway/analytical_reader.go` |
| 17 | Gateway wiring | `cmd/gateway/compose.go` |

### Never Generated (Frozen Exclusion List)

| Artifact | Reason |
|----------|--------|
| Domain event types (`internal/domain/`) | Architectural decisions |
| NATS stream definitions | Infrastructure architecture |
| Writer core logic (consumer.go, inserter.go, supervisor.go) | Framework code |
| ClickHouse client | Infrastructure adapter |
| Health/observability framework | Shared infrastructure |
| Gateway `compose.go` core logic | Composition root is architectural |
| HTTP server/router setup | Infrastructure |
| Shared helpers (`parseFloat`, `marshalJSON`, `parseAnalyticalParams`) | Shared utilities |
| CI configuration | Repository-wide concern |
| Template files themselves | Meta-level human ownership |

### Specification Evidence From 6 Families

| Family | Layer | Coverage Provided |
|--------|-------|-------------------|
| Candle | L1 Evidence | 16 columns, OHLCV floats, no JSON, no optional filters |
| RSI | L2 Signal | 12 columns, 1 JSON, no optional filters |
| RSI Oversold | L3 Decision | 14 columns, 2 JSON, 1 enum |
| Mean Reversion Entry | L4 Strategy | 15 columns, 3 JSON, 1 enum |
| Position Exposure | L5 Risk | 17 columns, 4 JSON, 1 enum, 1 text |
| Paper Order | L6 Execution | 20 columns, 4 JSON, 2 enums, 2 correlation IDs |

---

## 6. Ownership Model

```
+-----------------------------------------------------+
|  HUMAN-OWNED (never generated, never overwritten)    |
|  Domain types, stream definitions, infrastructure,   |
|  composition root, shared helpers, CI config,        |
|  template design, schema design, API surface         |
+-----------------------------------------------------+
|  CODEGEN-OWNED (generated, validated, replaceable)   |
|  Consumer specs, pipeline entries, mappers (cond.),  |
|  mapper tests, config entries, smoke phases          |
|  Reader/handler/route/use-case (Tier 2 only)         |
+-----------------------------------------------------+
|  SPEC-OWNED (source of truth, human-authored)        |
|  codegen/families/*.yaml, codegen/templates/*        |
+-----------------------------------------------------+
```

### Ownership Rules

**Rule 1 -- Spec Is Authoritative**: If a generated file and its spec disagree, the spec is correct. Regenerate the file.

**Rule 2 -- Generated Files Carry Headers**:
```go
// Code generated by mf-codegen from codegen/families/{family_name}.yaml. DO NOT EDIT.
// Template version: v{major}.{minor}.{patch}
// Generated at: {ISO-8601 timestamp}
```
Files missing this header are not codegen-owned and must not be overwritten.

**Rule 3 -- No Manual Edits to Generated Files**: Fixes go to the template or spec; the file is regenerated. Direct edits are treated as drift and flagged by CI.

**Rule 4 -- Human-Owned Files Are Never Overwritten**: Where integration requires modifying human-owned files, codegen uses append-only sections delimited by markers:
```go
// codegen:begin <artifact_type> family=<family_name> source=<spec_path>
// ... generated entries ...
// codegen:end <artifact_type> family=<family_name>
```

**Rule 5 -- Templates Are Human-Owned**: Reviewed, versioned, modified exclusively by humans.

**Rule 6 -- Spec Files Are Human-Authored**: No tool auto-generates spec files from code.

**Rule 7 -- Golden Specs Are Frozen**: Modified only when hand-crafted family implementation changes or spec schema is updated.

---

## 7. Equivalence Rules

### Definitions

- **Structural equivalence**: Normalized AST/parse trees produce identical output. Same tokens, fields, types, signatures, values. Mechanical -- checkable by diff after normalization.
- **Semantic equivalence**: Same observable behavior even if textual representation differs.
- Structural equivalence is a subset of semantic equivalence. For baseline validation, **structural equivalence is the primary gate**.

### Per-Artifact Equivalence Rules

#### A1: Consumer Spec Function

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Function name | Exact: `Writer{Family}{Layer}Consumer` | None |
| Return type | `ConsumerSpec` | None |
| All field values (Durable, Subject, Type, Stream.Name) | Exact string match | None |
| AckWait | `30 * time.Second` | None |
| MaxDeliver | `5` | None |

#### A2: Pipeline Entry

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| All struct fields (family, consumerName, inserterName, table, insertSQL) | Exact | None |
| consumerSpec call, isEnabled lambda, startConsumer lambda | Correct references | None |
| Variable names within lambdas | Flexible if same access pattern | Allowed |

#### A3: Mapper Function

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Function signature | `func map{Name}Row(e {EventType}) []any` | None |
| Return slice length | Matches column count in DDL | None |
| Column order | Matches DDL declaration order exactly | **Non-negotiable** |
| Transform functions per column type | Correct transform | None |

Transform rules: Float64 -> `parseFloat(...)`, JSON struct -> `marshalJSON(...)`, Enum -> `string(...)`, Bool/DateTime -> direct access, UInt32 -> `uint32(...)` cast.

#### A4: Mapper Unit Tests

Must include happy-path and zero-value tests. Generated test coverage must be >= golden.

#### A5: Config Array Entry

Exact family name string appended to correct config array. Position within array is flexible.

#### A6: Smoke Test Phase

Correct endpoint URL, required query params, domain-relevant field assertions.

### Equivalence Decision Matrix

| Question | Answer |
|----------|--------|
| Import order differs? | **Allowed** -- normalize with goimports |
| Whitespace differs? | **Allowed** -- normalize with gofmt |
| Comment text differs? | **Allowed** -- stripped during normalization |
| Variable name differs in lambda? | **Allowed** -- if same field access pattern |
| Column order differs? | **FORBIDDEN** -- must match DDL order |
| Transform function differs? | **FORBIDDEN** -- must match column type |
| Field missing/extra? | **FORBIDDEN** |
| Function name differs? | **FORBIDDEN** -- deterministic from spec |

---

## 8. Spec Evolution Rules

This schema is frozen. To add, remove, or change any field:

1. A new architectural stage must be opened.
2. The change must be justified by a concrete expansion need (not speculative).
3. All existing specs and golden specs must be updated atomically.
4. All validation invariants must be updated to cover the new field.
5. The freeze document must be re-versioned.

No "experimental" or "optional but undocumented" fields are permitted.

---

## 9. What This Freeze Means

1. **No new artifact types** may be added to Tier 1 without a new stage.
2. **No spec fields** may be added/removed/changed without updating this document.
3. **No ownership boundary** may shift without explicit architectural review.
4. **No validation mechanism** may be weakened or removed.
5. **Tier 2 activation** requires a separate authorization stage.

### Freeze Exceptions

The following may evolve without a new freeze:
- Template content (as long as golden test equivalence is maintained)
- CI step implementation details
- Spec field documentation clarifications that do not change semantics

---

## Related Documents

- [codegen-tranche-scoping.md](codegen-tranche-scoping.md) -- original scoping (S192, active reference)
- [codegen-validation-and-ci-strategy.md](codegen-validation-and-ci-strategy.md) -- validation and CI details
- [codegen-boundaries-and-governance.md](codegen-boundaries-and-governance.md) -- anti-patterns, boundaries, governance
- [codegen-path-stabilization-or-freeze-decision.md](codegen-path-stabilization-or-freeze-decision.md) -- active decision record
- [codegen-current-usage-boundaries-and-limitations.md](codegen-current-usage-boundaries-and-limitations.md) -- active reference
