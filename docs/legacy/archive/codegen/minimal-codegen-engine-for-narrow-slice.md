# Minimal Codegen Engine for Narrow Slice

## Purpose

This document describes the first codegen engine implemented for market-foundry. The engine is intentionally narrow: it covers exactly two artifact types for two baseline families, proving the generation model before any expansion.

## Narrow Slice: Consumer Spec + Pipeline Entry

### Why This Slice

The consumer spec function and pipeline entry struct are the two most mechanical artifacts in the writer service. They satisfy all three codegen boundary conditions from S193:

1. **Repetitive** — 6 identical instances exist across all families, differing only in naming and spec values.
2. **Mechanical** — Zero creative decisions. Every value is directly derivable from the family spec YAML.
3. **Spec-derivable** — No inference, no domain knowledge, no DDL awareness required.

### Why Not Other Artifacts

| Artifact | Reason Excluded |
|----------|----------------|
| Mapper function | Requires column-order knowledge, type transforms, DDL awareness. Creative decisions about field accessors. |
| Mapper tests | Depends on mapper structure. Test fixtures require domain type knowledge. |
| Config entry | Trivial (one string in an array) but involves JSONC manipulation — tooling complexity for low value. |
| Smoke test phase | Shell script generation introduces a different template language and escape concerns. |

These are candidates for future slices once the engine model is validated.

## Engine Architecture

### Directory Structure

```
codegen/
├── go.mod                              # Standalone Go module
├── main.go                             # CLI: validate, generate, compare, check-all
├── spec.go                             # YAML parsing, validation, derived fields
├── spec_test.go                        # Derived field computation tests
├── render.go                           # Template rendering
├── render_test.go                      # Render + golden comparison tests
├── compare.go                          # Structural normalization and diff
├── compare_test.go                     # Comparison logic tests
├── families/
│   ├── rsi.yaml                        # Baseline family: minimal complexity
│   └── paper_order.yaml                # Baseline family: ceiling complexity
├── templates/
│   ├── consumer_spec.go.tmpl           # Consumer spec function template
│   └── pipeline_entry.go.tmpl          # Pipeline entry struct template
└── golden-snapshots/
    ├── rsi/
    │   ├── consumer_spec.go.golden     # Extracted from signal_registry.go
    │   └── pipeline_entry.go.golden    # Extracted from pipeline.go
    └── paper_order/
        ├── consumer_spec.go.golden     # Extracted from execution_registry.go
        └── pipeline_entry.go.golden    # Extracted from pipeline.go
```

### Design Principles

1. **No runtime dependency** — Generated code is standalone Go. The codegen tool is a development-time CLI only.
2. **No framework** — Standard `text/template`. No plugin system, no reflection, no code generation library.
3. **No magic** — All naming conventions are documented, tested, and deterministic. The `Derived()` method on `FamilySpec` computes every derived field explicitly.
4. **Single source of truth** — One YAML file per family contains all spec values. Templates + spec = generated code.

### Spec Schema

The YAML spec follows the S193 frozen specification with 14 required fields:

```yaml
family:
  name: string          # snake_case, unique
  layer: string         # evidence|signal|decision|strategy|risk|execution
  tier: integer         # 1 or 2

nats:
  subject: string       # NATS subscription filter
  event_type: string    # Versioned event identifier
  stream: string        # JetStream stream name
  durable: string       # Durable consumer name

writer:
  table: string         # ClickHouse target table
  mapper: string        # Existing mapper function name
  pipeline_family_key: string
  config_array: string  # Config injection target

domain:
  event_package: string # Go package name
  event_type: string    # Go struct type name
```

### Derived Fields

The engine computes 10 derived fields from spec values:

| Derived Field | Example (RSI) | Example (Paper Order) |
|--------------|---------------|----------------------|
| ConsumerSpecFunc | WriterRSISignalConsumer | WriterPaperOrderExecutionConsumer |
| ConsumerName | writer-signal-rsi-consumer | writer-execution-paper-order-consumer |
| InserterName | writer-signal-rsi-inserter | writer-execution-paper-order-inserter |
| IsEnabledMethod | IsSignalFamilyEnabled | IsExecutionFamilyEnabled |
| RegistryField | signal | execution |
| NewConsumerFunc | NewSignalConsumer | NewExecutionConsumer |
| PascalFamily | RSI | PaperOrder |
| PascalLayer | Signal | Execution |
| InsertSQL | INSERT INTO signals | INSERT INTO executions |
| HyphenFamily | rsi | paper-order |

The evidence layer has documented exceptions: no layer prefix in consumer names, function names, or isEnabled methods.

### Comparison Model

Golden comparison uses structural normalization per S194:

1. Strip single-line comments
2. Normalize tabs to spaces
3. Trim whitespace per line
4. Remove empty lines
5. Compare line-by-line

This allows comment and formatting differences (S194 "allowed differences") while catching structural divergence (S194 "forbidden differences").

## CLI Interface

```bash
# Validate a spec file
codegen validate families/rsi.yaml

# Generate an artifact to stdout
codegen generate families/rsi.yaml consumer_spec

# Compare generated vs golden
codegen compare families/rsi.yaml consumer_spec

# Check all families × all artifacts
codegen check-all
```

## Validation Status

The engine passes all golden comparisons for both baseline families across both artifact types:

- `rsi/consumer_spec` — PASS
- `rsi/pipeline_entry` — PASS
- `paper_order/consumer_spec` — PASS
- `paper_order/pipeline_entry` — PASS

17 unit tests cover spec parsing, derived field computation, template rendering, and golden comparison.

## Constraints

- The engine does NOT generate new families. It proves equivalence against existing hand-crafted families.
- Templates produce code fragments, not complete files. Integration into source files remains manual.
- The evidence layer's naming exceptions are handled but only tested via the candle family derived fields (not golden comparison, since candle is not a baseline family).
