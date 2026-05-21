# Generated/Manual Coexistence Hardening

> S201 — Hardening the boundary between hand-written and codegen-governed artifacts.

## Context

After S200 integrated the first codegen slice (RSI consumer_spec + pipeline_entry), the project entered a hybrid state: 6 families have YAML specs and golden snapshots, but only 1 (RSI) has governance markers in target files. The remaining 5 families are fully manual. This document defines the hardened coexistence model that makes ownership, drift, and regeneration operationally safe before the first codegen-first family is introduced.

## Three-Zone Ownership Model

Every file in the repository falls into exactly one of three zones:

### Zone 1: Human-Owned (never generated)

| Category | Examples | Rule |
|----------|----------|------|
| Domain event types | `internal/domain/{layer}/*.go` | Permanently manual |
| ClickHouse schema | `deploy/migrations/*.sql` | Permanently manual |
| NATS stream definitions | `internal/adapters/nats/{layer}_registry.go` (stream/registry structs) | Permanently manual |
| Writer core logic | `cmd/writer/{consumer,inserter,supervisor}.go` | Permanently manual |
| Templates | `codegen/templates/*.go.tmpl` | Frozen; change requires authorization stage |
| Specs | `codegen/families/*.yaml` | Human-authored; source of truth for generated output |
| Config | `deploy/configs/*.jsonc` | Permanently manual |
| CI workflows | `.github/workflows/*.yml` | Permanently manual |
| Operational scripts | `scripts/*.sh` (except `codegen-integrated-check.sh`) | Permanently manual |

### Zone 2: Machine-Owned (generated, immutable)

| Category | Location | Rule |
|----------|----------|------|
| Golden snapshots | `codegen/golden-snapshots/{family}/{artifact}.go.golden` | Regenerated only via `codegen generate`; never manually edited |
| Governed fragments | Code between `codegen:begin`/`codegen:end` markers in target files | Immutable; bugs fixed in spec or template, never in output |

### Zone 3: Mixed (human file containing machine fragments)

| File | Human Region | Machine Region |
|------|-------------|----------------|
| `internal/adapters/nats/{layer}_registry.go` | Registry structs, stream specs, store consumers | Consumer spec functions within `codegen:begin/end` markers |
| `cmd/writer/pipeline.go` | Pipeline struct definition, tracker defs | Pipeline entries within `codegen:begin/end` markers |

## Marker Format Standard

All governance markers follow this exact format:

```go
// codegen:begin <artifact_type> family=<family_name> source=<spec_path>
... generated code ...
// codegen:end <artifact_type> family=<family_name>
```

Where:
- `artifact_type` is one of: `consumer_spec`, `pipeline_entry`
- `family_name` matches `family.name` in the YAML spec
- `source` points to the authoritative spec file

Markers are comments with zero runtime impact. They exist solely for CI extraction and human readability.

## Integration Manifest

`codegen/integrated.yaml` is the single source of truth for which slices are governed:

```yaml
slices:
  - family: rsi
    artifact: consumer_spec
    spec: codegen/families/rsi.yaml
    golden: codegen/golden-snapshots/rsi/consumer_spec.go.golden
    target: internal/adapters/nats/signal_registry.go
    marker: "codegen:begin consumer_spec family=rsi"
    integrated_at: "2026-03-20"
    stage: S200
```

Adding a new governed slice requires adding an entry here. The `codegen-integrated-check.sh` script reads this manifest automatically — no script edits needed.

## Manual Families: Deliberate Choice

The 5 non-governed families (candle, rsi_oversold, mean_reversion_entry, position_exposure, paper_order) remain fully manual. This is deliberate:

1. They were written before the codegen engine existed.
2. Retroactive conversion adds risk with no functional benefit.
3. Their golden snapshots serve as the engine's test baseline.
4. They will **never** be retroactively governed — new families are the codegen path forward.

## Cross-Spec Uniqueness

S201 added cross-spec validation (`codegen validate-all`) that enforces:

- No two specs share the same `family.name`
- No two specs share the same `nats.durable`
- No two specs share the same `nats.subject`

This prevents silent collisions when new families are added. The check runs in CI before golden comparison.

## What Cannot Be Edited Manually

1. Code inside `codegen:begin`/`codegen:end` markers → CI will reject
2. Golden snapshot files → regenerate from spec instead
3. Derived naming conventions → change the spec, not the output

## What Remains Manual and Why

| Artifact | Why Manual |
|----------|-----------|
| Mapper functions (A3) | Require domain knowledge; field mapping logic isn't derivable from A1/A2 spec fields |
| Mapper tests (A4) | Test logic depends on domain semantics |
| Config entries (A5) | Operational decision, not a code generation concern |
| Smoke test phases (A6) | Assertion logic depends on runtime behavior |
| Domain event types | Core domain modeling; must be human-designed |
| ClickHouse DDL | Schema design requires human judgment |

These remain manual until a future stage authorizes their generation (requires spec extensions not yet designed).
