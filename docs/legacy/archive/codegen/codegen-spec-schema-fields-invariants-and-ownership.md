# Codegen Spec Schema, Fields, Invariants, and Ownership

> Companion to [codegen-specification-freeze.md](codegen-specification-freeze.md).
> Defines the exact shape of a family spec, every field's status, validation invariants, and ownership rules.

## Canonical Spec Schema

Every family spec file (`codegen/families/{family_name}.yaml`) must conform exactly to this shape:

```yaml
# ── Identity ──────────────────────────────────────────────
family:
  name: "<string>"          # REQUIRED — unique family identifier (snake_case)
  layer: "<string>"         # REQUIRED — L1-L6 layer name (evidence|signal|decision|strategy|risk|execution)
  tier: <integer>           # REQUIRED — 1 (within-layer) or 2 (new-layer)

# ── NATS Binding ──────────────────────────────────────────
nats:
  subject: "<string>"       # REQUIRED — NATS subject filter
  event_type: "<string>"    # REQUIRED — fully qualified event type identifier
  stream: "<string>"        # REQUIRED — NATS JetStream stream name
  durable: "<string>"       # REQUIRED — durable consumer name

# ── Writer Binding ────────────────────────────────────────
writer:
  table: "<string>"         # REQUIRED — target ClickHouse table name
  mapper: "<string>"        # REQUIRED — existing mapper function name OR "generate"
  pipeline_family_key: "<string>"  # REQUIRED — family key for pipeline registration
  config_array: "<string>"  # REQUIRED — writer config array to append to

# ── Domain Binding ────────────────────────────────────────
domain:
  event_package: "<string>" # REQUIRED — Go package containing event type
  event_type: "<string>"    # REQUIRED — Go type name of the event struct
  columns:                  # CONDITIONAL — required only when mapper: "generate"
    - name: "<string>"      #   REQUIRED per entry — ClickHouse column name
      go_field: "<string>"  #   REQUIRED per entry — Go struct field name
      ch_type: "<string>"   #   REQUIRED per entry — ClickHouse column type
      transform: "<string>" #   OPTIONAL per entry — transformation function name

# ── Tier 2 Only ───────────────────────────────────────────
schema:                     # CONDITIONAL — required only when tier: 2
  migration_number: <integer>  # REQUIRED — migration sequence number
  ddl: "<string>"           # REQUIRED — CREATE TABLE IF NOT EXISTS statement
  reverse_ddl: "<string>"   # REQUIRED — DROP TABLE IF EXISTS statement
```

## Field Reference

### Required Fields (always mandatory)

| Section | Field | Type | Constraints | Purpose |
|---------|-------|------|-------------|---------|
| `family` | `name` | string | Unique across all specs; snake_case; `[a-z][a-z0-9_]*` | Primary identifier for the family |
| `family` | `layer` | string | One of: `evidence`, `signal`, `decision`, `strategy`, `risk`, `execution` | Maps to L1–L6 analytical layer |
| `family` | `tier` | integer | `1` or `2` | Determines artifact scope |
| `nats` | `subject` | string | Valid NATS subject with `>` wildcard | NATS subscription filter |
| `nats` | `event_type` | string | Dot-separated, versioned: `{layer}.events.v{N}.{name}` | Event type discriminator |
| `nats` | `stream` | string | UPPER_SNAKE_CASE; must reference existing stream | JetStream stream name |
| `nats` | `durable` | string | Pattern: `writer-{layer}-{family}` or `writer-{family}` | Durable consumer name |
| `writer` | `table` | string | Must reference table created by existing migration | Target ClickHouse table |
| `writer` | `mapper` | string | Existing Go function name or literal `"generate"` | Mapper function reference |
| `writer` | `pipeline_family_key` | string | snake_case; unique across all specs | Pipeline registration key |
| `writer` | `config_array` | string | Must reference existing config array in `writer.jsonc` | Config injection target |
| `domain` | `event_package` | string | Valid Go package name; must exist in codebase | Package containing event struct |
| `domain` | `event_type` | string | Valid Go exported type name; must exist in codebase | Event struct type |

### Conditional Fields

| Section | Field | Condition | Constraints |
|---------|-------|-----------|-------------|
| `domain` | `columns` | Required when `writer.mapper` = `"generate"` | Non-empty array; each entry has `name`, `go_field`, `ch_type` |
| `domain.columns[]` | `transform` | Optional per column entry | Must reference known transform function (`parseFloat`, `marshalJSON`, etc.) |
| `schema` | (entire section) | Required when `family.tier` = `2` | Must contain `migration_number`, `ddl`, `reverse_ddl` |

### Invalid Fields

Any field not listed above is **invalid**. Spec validation must reject specs containing unknown fields. This prevents:

- Implicit feature creep via undocumented spec extensions
- Ambiguity about what the codegen engine reads
- Silent ignored fields that create false confidence

Specifically prohibited:

| Prohibited Field | Reason |
|------------------|--------|
| `reader.*` | Reader artifacts are Tier 2 scope; not present in Tier 1 specs |
| `handler.*` | Handler artifacts are Tier 2 scope |
| `route.*` | Route artifacts are Tier 2 scope |
| `retry.*` | Retry/backoff policy is infrastructure concern, not per-family |
| `observability.*` | Observability is provided by shared infrastructure |
| `ttl` | TTL is schema-level decision, defined in DDL migration |
| `partitioning` | Partitioning is schema-level decision, defined in DDL migration |
| `version` | Spec files are versioned via git, not internal field |
| `description` | Specs are declarative, not documentary |
| `tags` | No tagging system; layers provide categorization |
| `depends_on` | No dependency resolution; families are independent |

## Validation Invariants

Every spec file must pass all invariants before codegen execution. Violation of any invariant is a hard error.

### Uniqueness Invariants

| Invariant | Scope | Violation Response |
|-----------|-------|--------------------|
| `family.name` is unique | Across all `codegen/families/*.yaml` | Reject: duplicate family |
| `nats.durable` is unique | Across all specs | Reject: durable collision |
| `writer.pipeline_family_key` is unique | Across all specs | Reject: pipeline key collision |

### Referential Integrity Invariants

| Invariant | Verification |
|-----------|-------------|
| `writer.table` references an existing ClickHouse migration | Scan `deploy/migrations/*.sql` for matching `CREATE TABLE` |
| `domain.event_type` exists as a Go type in `domain.event_package` | Parse Go source in `internal/domain/{event_package}/` |
| `nats.stream` references a defined JetStream stream | Cross-reference with stream definitions in NATS adapter |
| `writer.mapper` (when not `"generate"`) references existing Go function | Scan `cmd/writer/mappers.go` for function name |
| `writer.config_array` references existing array in `writer.jsonc` | Parse `deploy/configs/writer.jsonc` |

### Structural Invariants

| Invariant | Rule |
|-----------|------|
| Tier 1 specs must NOT contain `schema` section | `tier: 1` + `schema:` present = invalid |
| Tier 2 specs MUST contain `schema` section | `tier: 2` + `schema:` absent = invalid |
| `mapper: "generate"` requires `domain.columns` | `mapper: "generate"` + no `columns` = invalid |
| `mapper: "<function_name>"` must NOT have `domain.columns` | Named mapper + `columns` present = invalid |
| Column names must match DDL column names | When `mapper: "generate"`, column names validated against migration DDL |
| Column order must match DDL column order | `domain.columns` order = DDL column order |

### Naming Pattern Invariants

| Field | Pattern | Example |
|-------|---------|---------|
| `family.name` | `[a-z][a-z0-9_]*` | `ema_crossover` |
| `nats.durable` | `writer-{layer}-{family}` or `writer-{family}` | `writer-signal-ema_crossover` |
| `nats.event_type` | `{layer}.events.v{N}.{name}` | `signal.events.v1.ema_crossover_generated` |
| `nats.stream` | `[A-Z][A-Z0-9_]*` | `SIGNAL_EVENTS` |
| `writer.pipeline_family_key` | matches `family.name` | `ema_crossover` |

## Ownership Rules

### Rule 1 — Spec Is Authoritative

If a generated file and its spec disagree on any value (subject, table, mapper, columns), the spec is correct. The generated file must be regenerated.

### Rule 2 — Generated Files Carry Headers

Every generated file must include:

```go
// Code generated by mf-codegen from codegen/families/{family_name}.yaml. DO NOT EDIT.
// Template version: v{major}.{minor}.{patch}
// Generated at: {ISO-8601 timestamp}
```

Files missing this header are not codegen-owned and must not be overwritten.

### Rule 3 — No Manual Edits to Generated Files

If a generated file requires a fix:

1. Identify whether the fix belongs in the **template** or the **spec**.
2. Apply the fix to the template or spec.
3. Regenerate the file.
4. Commit the updated template/spec AND regenerated file together.

Direct edits to generated files are treated as drift and flagged by CI.

### Rule 4 — Human-Owned Files Are Never Overwritten

Codegen never overwrites or replaces content in human-owned files. Where integration requires modifying human-owned files (e.g., `pipeline.go`, `mappers.go`), codegen uses **append-only sections** delimited by markers:

```go
// --- BEGIN CODEGEN MANAGED SECTION ---
// Do not edit between these markers. Managed by mf-codegen.

// ... generated entries ...

// --- END CODEGEN MANAGED SECTION ---
```

Human code above or below these markers is never touched.

### Rule 5 — Templates Are Human-Owned

Template files (`codegen/templates/*`) are authored, reviewed, and modified exclusively by humans. Codegen reads templates; it never writes to them.

### Rule 6 — Spec Files Are Human-Authored

Spec files (`codegen/families/*.yaml`) are created and maintained by humans making explicit architectural decisions. No tool auto-generates spec files from code, events, or other sources.

### Rule 7 — Golden Specs Are Frozen

Golden spec files (`codegen/golden/*.yaml`) represent the 6 existing hand-crafted families. They are modified only when:

- A hand-crafted family's implementation changes (extremely rare)
- A spec schema field is added or modified (requires freeze update)

Golden specs are never regenerated from code. They are manually authored to describe what the hand-crafted families declare.

## Spec Validation Execution Order

When validating a spec, checks run in this order (fail-fast):

1. **Schema conformance** — all required fields present, no unknown fields, correct types
2. **Naming pattern compliance** — all naming invariants satisfied
3. **Uniqueness checks** — no collisions with other specs
4. **Referential integrity** — all references resolve to existing artifacts
5. **Structural invariants** — tier/mapper/column consistency
6. **Column alignment** — (when applicable) columns match DDL

A spec that fails any step is rejected with a specific, actionable error message identifying the violated invariant.

## Spec Evolution Rules

This schema is frozen. To add, remove, or change any field:

1. A new architectural stage must be opened.
2. The change must be justified by a concrete expansion need (not speculative).
3. All existing specs and golden specs must be updated atomically.
4. All validation invariants must be updated to cover the new field.
5. The freeze document must be re-versioned.

No "experimental" or "optional but undocumented" fields are permitted.
