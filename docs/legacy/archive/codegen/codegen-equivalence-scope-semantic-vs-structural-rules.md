# Codegen Equivalence Scope: Semantic vs Structural Rules

> S194 — Distinguishes what counts as structural equivalence from semantic equivalence, with explicit rules for each artifact type.

## 1. Definitions

### Structural Equivalence

Two artifacts are **structurally equivalent** when their normalized AST/parse trees produce identical output. This means:

- Same tokens in the same order (after normalization)
- Same fields, same types, same nesting
- Same function signatures
- Same literal values

Structural equivalence is **mechanical** — it can be checked by a diff tool after normalization.

### Semantic Equivalence

Two artifacts are **semantically equivalent** when they produce the same observable behavior, even if their textual representation differs. This means:

- Same columns written to ClickHouse in the same order
- Same NATS subject filtering behavior
- Same event routing
- Same enable/disable logic
- Same error handling characteristics

Semantic equivalence requires **domain knowledge** — it cannot be fully checked by diff alone.

### Relationship

```
Structural equivalence ⊂ Semantic equivalence

If two artifacts are structurally equivalent, they are semantically equivalent.
If two artifacts are semantically equivalent, they may NOT be structurally equivalent.
```

For the S194 baseline, **structural equivalence is the primary gate**. Semantic equivalence rules exist to handle the small set of allowed structural differences.

## 2. Per-Artifact Equivalence Rules

### A1: Consumer Spec Function

#### Structural Rules

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Function name | Exact: `Writer{Family}{Layer}Consumer` | None |
| Return type | `ConsumerSpec` | None |
| `Durable` field value | Exact string match | None |
| `Subject` field value | Exact string match | None |
| `Type` field value | Exact string match | None |
| `Stream.Name` field value | Exact string match | None |
| `AckWait` value | `30 * time.Second` | None |
| `MaxDeliver` value | `5` | None |

#### Semantic Rules

| Rule | Check |
|------|-------|
| Durable name follows pattern | `writer-{layer}-{family}` or `writer-{family}` |
| Subject matches NATS routing | Wildcards resolve to correct event namespace |
| Event type matches domain event | `{layer}.events.v{N}.{family}_{action}` |

#### Allowed Differences

- None. Consumer specs are fully deterministic from spec fields.

---

### A2: Pipeline Entry

#### Structural Rules

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| `family` field | Exact string: family name | None |
| `consumerName` field | `writer-{layer}-{family}-consumer` | None |
| `inserterName` field | `writer-{layer}-{family}-inserter` | None |
| `table` field | Exact string: table name | None |
| `insertSQL` field | `INSERT INTO {table}` | None |
| `consumerSpec` call | References correct consumer spec function | None |
| `isEnabled` lambda | Calls correct `Is{Layer}FamilyEnabled("{family}")` | None |
| `startConsumer` lambda | Calls correct `New{Layer}Consumer` with correct mapper | None |

#### Semantic Rules

| Rule | Check |
|------|-------|
| Mapper reference resolves | `map{Layer}Row` or `map{Family}Row` function exists |
| Enable condition matches config | Config array name matches `Is{Layer}FamilyEnabled` |
| Tracker integration present | `tracker.RecordEvent()` call exists in consumer lambda |

#### Allowed Differences

- Variable names within lambdas (e.g., `event` vs `e` vs `evt`) — **semantically equivalent if same field access pattern**
- Logger variable name
- Whitespace within struct literal

---

### A3: Mapper Function

#### Structural Rules

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Function signature | `func map{Name}Row(e {EventType}) []any` | None |
| Return slice length | Matches column count in DDL | None |
| Column order | Matches DDL declaration order exactly | None |
| Field access paths | `e.{Field}` or `e.{Nested}.{Field}` | None |
| Transform functions | Correct transform per column type | None |

#### Semantic Rules

| Rule | Check |
|------|-------|
| Every DDL column has a corresponding slice element | 1:1 mapping, no gaps |
| Float columns use `parseFloat` | `Float64` DDL type → `parseFloat(...)` |
| JSON columns use `marshalJSON` | `String` DDL type with structured Go field → `marshalJSON(...)` |
| Enum columns use `string()` | `LowCardinality(String)` with Go enum → `string(...)` |
| Bool columns pass through | `Bool` DDL type → direct field access |
| DateTime columns pass through | `DateTime64` DDL type → direct field access |
| String columns pass through | `String`/`LowCardinality(String)` without enum → direct field access |
| UInt columns use cast | `UInt32` DDL type → `uint32(...)` cast |

#### Allowed Differences

- Intermediate variable names (e.g., `m := e.Metadata` vs inlining `e.Metadata.ID`)
- **But**: if golden uses intermediate variables, generated should too (structural preference)

#### Critical: Column Order Is Non-Negotiable

Column order in the `[]any` return must match the DDL `CREATE TABLE` column declaration order exactly. This is a **hard structural rule** because ClickHouse bulk insert relies on positional column binding.

---

### A4: Mapper Unit Tests

#### Structural Rules

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Test function name | `TestMap{Name}Row{Variant}` | None |
| Test input construction | Populates all event fields | Field order may vary |
| Assertion count | One assertion per output column | None |
| Assertion values | Expected values match transforms | None |

#### Semantic Rules

| Rule | Check |
|------|-------|
| Happy path test exists | Fully populated event → all columns correct |
| Zero-value test exists | Zero/empty event → no panic, correct defaults |
| Float transform test | Float columns produce correct `float64` output |
| JSON transform test | JSON columns produce valid JSON string |

#### Allowed Differences

- Test helper function usage
- Assertion library (direct comparison vs testify vs table-driven)
- Test case ordering
- Additional test cases in generated (generated ≥ golden)

---

### A5: Config Array Entry

#### Structural Rules

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Array name | Correct config array for the layer | None |
| Entry value | Exact family name string | None |
| Entry position | Appended (order within array is semantic, not structural) | Order flexible |

#### Semantic Rules

| Rule | Check |
|------|-------|
| Array exists in config | `writer.jsonc` has the target array |
| Family name is valid | Matches `family.name` in spec |
| No duplicates | Family name appears exactly once |

#### Allowed Differences

- Position within the array (first, last, alphabetical — any order is acceptable)
- JSONC comment formatting

---

### A6: Smoke Test Phase

#### Structural Rules

| Element | Must Match | Tolerance |
|---------|-----------|-----------|
| Endpoint URL | Correct analytical endpoint path | None |
| Query parameters | All required params present | Order may vary |
| Expected fields in response | All domain-relevant fields checked | None |

#### Semantic Rules

| Rule | Check |
|------|-------|
| Endpoint matches route registration | URL path resolves to correct handler |
| Query params match reader filters | Parameters align with reader's query contract |
| Field assertions cover domain fields | At minimum: type-specific fields, not just metadata |

#### Allowed Differences

- Echo/print formatting
- Variable names in shell script
- Comment text
- Assertion method (jq vs grep vs string match)

## 3. Equivalence Decision Matrix

### Quick Reference

| Question | Answer |
|----------|--------|
| Import order differs? | **Allowed** — normalize with goimports |
| Whitespace differs? | **Allowed** — normalize with gofmt |
| Comment text differs? | **Allowed** — stripped during normalization |
| Variable name differs in lambda? | **Allowed** — if same field access pattern |
| Column order differs? | **FORBIDDEN** — must match DDL order |
| Transform function differs? | **FORBIDDEN** — must match column type |
| Field missing? | **FORBIDDEN** — every DDL column must map |
| Extra field present? | **FORBIDDEN** — no phantom columns |
| Function name differs? | **FORBIDDEN** — naming is deterministic from spec |
| SQL shape differs? | **FORBIDDEN** — INSERT INTO must match table |
| Test case missing? | **FORBIDDEN** — generated ≥ golden |
| Extra test case? | **Allowed** — more coverage is acceptable |

## 4. Equivalence Tiers

### Tier 1 Equivalence (S194 Scope — Current)

Covers write-path artifacts only:
- Consumer spec (A1)
- Pipeline entry (A2)
- Mapper (A3)
- Mapper tests (A4)
- Config entry (A5)
- Smoke phase (A6)

### Tier 2 Equivalence (Future — When Authorized)

Would additionally cover:
- Reader adapter and tests
- Use case and tests
- Contracts struct
- Handler method and tests
- Route registration
- Gateway wiring
- Migration DDL

Tier 2 rules will be defined when Tier 2 is authorized. The structural vs semantic framework established here applies directly — only the artifact-specific rules need extension.

## 5. Edge Cases and Rulings

### Edge Case 1: Shared Mapper

When multiple within-layer families share the same mapper (e.g., RSI and EMA Crossover both use `mapSignalRow`):

- **Codegen does NOT generate a new mapper** — it references the existing one
- **Equivalence check**: verify the `mapper` spec field references an existing function
- **No golden snapshot comparison** for the mapper artifact itself

### Edge Case 2: Generated Mapper

When `mapper: "generate"` and `domain.columns` is present:

- **Codegen generates a new mapper function** with a unique name
- **Equivalence check**: structural rules apply to the generated function
- **Golden comparison**: against the pattern established by existing mappers (same transform rules)
- **Not in S194 baseline** — deferred until first generated-mapper family

### Edge Case 3: Marker Section Placement

Pipeline entries and config entries are appended to existing files using marker sections:

```go
// --- BEGIN CODEGEN MANAGED SECTION ---
// --- END CODEGEN MANAGED SECTION ---
```

- **Structural rule**: markers must be present and correctly placed
- **Content within markers**: subject to normal structural equivalence
- **Content outside markers**: untouched by codegen, not subject to comparison

### Edge Case 4: Template Version Mismatch

If a template evolves (e.g., v1.0 → v1.1) but golden snapshots are from v1.0:

- **Golden snapshots must be re-extracted** from hand-crafted code (which doesn't change)
- **Template must still produce equivalent output** — template evolution is tested by golden equivalence
- **If template legitimately diverges**: update golden snapshots with justification

## 6. Non-Goals

- **Behavioral equivalence testing**: proving that generated code behaves identically at runtime is out of scope for equivalence comparison. Integration and smoke tests cover that separately.
- **Performance equivalence**: generated code is not benchmarked against hand-crafted.
- **Style equivalence**: as long as `gofmt`-normalized output matches, style preferences are irrelevant.
- **Documentation equivalence**: comments and docs are stripped; only code structure matters.
