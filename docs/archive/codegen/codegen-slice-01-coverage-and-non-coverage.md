# Codegen Slice 01: Coverage and Non-Coverage

## Slice Definition

Slice 01 covers the **consumer spec function** and **pipeline entry struct** — the two most mechanical Tier 1 artifacts — for two baseline families (RSI and Paper Order).

## What Slice 01 Covers

### Artifacts Generated

| # | Artifact | Template | Target Location |
|---|----------|----------|-----------------|
| A1 | Consumer spec function | `consumer_spec.go.tmpl` | `internal/adapters/nats/{layer}_registry.go` |
| A2 | Pipeline entry struct | `pipeline_entry.go.tmpl` | `cmd/writer/pipeline.go` |

### Baseline Families

| Family | Layer | Complexity | Role |
|--------|-------|-----------|------|
| RSI | Signal (L2) | Minimal — 12 columns, 1 JSON, 0 enums | Lower bound |
| Paper Order | Execution (L6) | Ceiling — 20 columns, 4 JSON, 2 enums | Upper bound |

### Capabilities Proven

1. **YAML spec parsing** — Reads and validates family specs per S193 frozen schema.
2. **Naming derivation** — Computes all Go naming conventions (PascalCase with abbreviation awareness, hyphenation, layer-specific exceptions).
3. **Template rendering** — Produces Go code fragments via `text/template`.
4. **Golden comparison** — Structural normalization and line-by-line diff against hand-crafted baselines.
5. **CLI interface** — validate, generate, compare, check-all commands.

### Spec Coverage

All 14 required S193 fields are parsed and validated:

- `family.name`, `family.layer`, `family.tier`
- `nats.subject`, `nats.event_type`, `nats.stream`, `nats.durable`
- `writer.table`, `writer.mapper`, `writer.pipeline_family_key`, `writer.config_array`
- `domain.event_package`, `domain.event_type`

### Test Coverage

- 17 unit tests
- Derived field tests for 3 families (RSI, Paper Order, Candle/Evidence)
- PascalCase conversion for all 12 known names
- Golden comparison for 4 artifact instances (2 families × 2 artifacts)
- Normalization logic tests
- Spec loading and validation tests

## What Slice 01 Does NOT Cover

### Deferred Tier 1 Artifacts

| # | Artifact | Reason Deferred | Prerequisite |
|---|----------|----------------|--------------|
| A3 | Mapper function | Requires column-order knowledge, type transforms, DDL-aware code generation | Column spec extension in YAML |
| A4 | Mapper unit tests | Depends on mapper structure; test fixtures need domain type knowledge | A3 first |
| A5 | Config entry (writer.jsonc) | JSONC manipulation tooling; trivial value for effort | JSON/JSONC template support |
| A6 | Smoke test phase | Shell script generation; different template language | Shell template support |

### Deferred Capabilities

| Capability | Reason Deferred |
|-----------|----------------|
| File integration (write into source files) | Requires marker section detection and insertion logic |
| Tier 2 artifacts (read-path) | S192 D2: Tier 2 deferred until Tier 1 proven in production |
| New family generation | S194: First new-family generation is S196+, after golden equivalence confirmed |
| CI drift detection | Requires `codegen-golden`, `codegen-drift`, `codegen-lint` CI jobs |
| Generated file headers | Requires file-write mode with header injection |
| Spec uniqueness validation (cross-file) | Requires loading all specs and checking uniqueness invariants |

### Never Generated (Per S192 D6)

These artifact types are always human-authored:

1. Domain event types
2. NATS stream definitions
3. Writer core logic (inserter, supervisor, consumer actor)
4. ClickHouse client
5. Health framework
6. Gateway composition root
7. HTTP server setup
8. Shared helpers
9. CI configuration
10. Template files themselves

### Existing Families: Immutable

Per S192 D6, the 6 hand-crafted families remain hand-crafted and serve as golden references:

- candle (evidence)
- rsi (signal)
- rsi_oversold (decision)
- mean_reversion_entry (strategy)
- position_exposure (risk)
- paper_order (execution)

## Expansion Path

### Slice 02 (Candidate)

Add mapper function generation (`A3`) with `domain.columns` spec extension. This requires:
- Column spec YAML schema (name, go_field, ch_type, transform)
- Transform-aware template (parseFloat, marshalJSON, string cast, uint32 cast)
- Column order validation against DDL

### Slice 03 (Candidate)

Add mapper test generation (`A4`). Requires:
- Test fixture template
- Assertion generation per column type
- JSON validation for marshal columns

### From Validation to Production

Once all 6 Tier 1 artifacts are covered and golden equivalence is confirmed for both baseline families:
1. Generate specs for a new family (S196+)
2. Run codegen to produce all artifacts
3. Compare against manual implementation for correctness
4. Integrate into CI as `codegen-drift` check

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Premature expansion to Tier 2 | Gate: Tier 1 must be fully validated first |
| Template complexity creep | Each template covers exactly one artifact type |
| Naming convention drift | Derived fields are tested against all known families |
| Golden snapshot staleness | Snapshots are frozen extracts; hand-crafted code changes require snapshot update |
| Over-reliance on structural comparison | Semantic rules (column order, transform correctness) deferred to mapper slice |
