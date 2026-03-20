# Stage S195: Minimal Codegen Engine for a Narrow Slice — Report

## Executive Summary

S195 delivers the first functional codegen engine for market-foundry. The engine covers a deliberately narrow slice — consumer spec function and pipeline entry struct — for two baseline families (RSI and Paper Order). All 4 golden comparisons pass, proving structural equivalence between generated and hand-crafted artifacts. The engine is minimal (6 Go source files, 2 templates, ~350 lines of engine code), explicit (every naming derivation is tested), and auditable (YAML spec → template → generated code, no hidden logic).

## Narrow Slice Chosen

**Artifacts**: Consumer Spec Function + Pipeline Entry Struct

**Rationale**:
- Most mechanical artifacts in the writer service (zero creative decisions)
- 100% spec-derivable (all values come directly from YAML)
- Structurally identical across all 6 existing families
- Sufficient to validate the entire generation pipeline (spec → template → compare)
- Low risk: does not touch DDL-aware code, type transforms, or shell scripts

**Baseline Families**: RSI (signal, minimal complexity) + Paper Order (execution, ceiling complexity) — per S194 two-family bracket strategy.

## Decisions

| ID | Decision | Rationale |
|----|----------|-----------|
| D1 | Consumer spec + pipeline entry as narrow slice | Most mechanical, zero-creative-decision artifacts; exercises full generation pipeline |
| D2 | Standalone Go module at `codegen/` | Isolated from runtime code; no dependency on application modules |
| D3 | `text/template` only, no codegen framework | S192 D3: no runtime dependency, no reflection, no plugin system |
| D4 | Derived fields computed from spec, not stored in YAML | Keeps specs minimal per S193; naming rules are testable engine logic |
| D5 | Known-abbreviation map for PascalCase (RSI, EMA, ID) | Go naming conventions require abbreviation awareness; small, explicit list |
| D6 | Structural normalization for comparison (strip comments, normalize whitespace) | Per S194 Layer 1 equivalence rules; allows cosmetic differences |

## Implementation

### Files Created

| File | Purpose | Lines |
|------|---------|-------|
| `codegen/go.mod` | Module definition | 5 |
| `codegen/main.go` | CLI entrypoint (validate, generate, compare, check-all) | 130 |
| `codegen/spec.go` | YAML parsing, validation, derived field computation | 170 |
| `codegen/spec_test.go` | Derived fields + parsing tests | 110 |
| `codegen/render.go` | Template rendering | 50 |
| `codegen/render_test.go` | Render + golden comparison tests | 200 |
| `codegen/compare.go` | Structural normalization and diff | 90 |
| `codegen/compare_test.go` | Comparison logic tests | 50 |
| `codegen/families/rsi.yaml` | RSI baseline spec | 20 |
| `codegen/families/paper_order.yaml` | Paper Order baseline spec | 20 |
| `codegen/templates/consumer_spec.go.tmpl` | Consumer spec template | 17 |
| `codegen/templates/pipeline_entry.go.tmpl` | Pipeline entry template | 23 |
| `codegen/golden-snapshots/rsi/consumer_spec.go.golden` | Hand-crafted RSI consumer spec extract | 16 |
| `codegen/golden-snapshots/rsi/pipeline_entry.go.golden` | Hand-crafted RSI pipeline entry extract | 23 |
| `codegen/golden-snapshots/paper_order/consumer_spec.go.golden` | Hand-crafted Paper Order consumer spec extract | 17 |
| `codegen/golden-snapshots/paper_order/pipeline_entry.go.golden` | Hand-crafted Paper Order pipeline entry extract | 23 |

### Files Modified

| File | Change |
|------|--------|
| `go.work` | Added `./codegen` to workspace |

### Architecture Documents Created

| Document | Purpose |
|----------|---------|
| `docs/architecture/minimal-codegen-engine-for-narrow-slice.md` | Engine design, directory structure, CLI, validation status |
| `docs/architecture/codegen-slice-01-coverage-and-non-coverage.md` | What is covered, what is deferred, expansion path |

## Validation Results

### Golden Comparisons (4/4 PASS)

```
PASS  rsi/consumer_spec
PASS  rsi/pipeline_entry
PASS  paper_order/consumer_spec
PASS  paper_order/pipeline_entry
```

### Unit Tests (17/17 PASS)

```
TestToPascalCase               — 12 snake_case → PascalCase conversions
TestDerivedFields_RSI          — 10 derived field assertions
TestDerivedFields_PaperOrder   — 8 derived field assertions
TestDerivedFields_Evidence     — 4 derived field assertions (evidence exceptions)
TestLoadSpec                   — YAML parsing + validation
TestValidate_MissingFields     — Empty spec rejection
TestNormalizeForComparison     — Comment stripping + whitespace normalization
TestCompareWithGolden_Pass     — Matching content detection
TestCompareWithGolden_Fail     — Divergence detection
TestRenderConsumerSpec_RSI     — 5 content assertions
TestRenderPipelineEntry_RSI    — 10 content assertions
TestRenderConsumerSpec_PaperOrder — 4 content assertions
TestRenderPipelineEntry_PaperOrder — 8 content assertions
TestGoldenComparison_RSI_ConsumerSpec — Full golden pass
TestGoldenComparison_RSI_PipelineEntry — Full golden pass
TestGoldenComparison_PaperOrder_ConsumerSpec — Full golden pass
TestGoldenComparison_PaperOrder_PipelineEntry — Full golden pass
```

## Gains

1. **Generation model validated** — The YAML → template → generated code pipeline works and produces structurally equivalent output.
2. **Naming derivation proven** — PascalCase, hyphenation, layer-specific exceptions all produce correct results.
3. **Comparison baseline established** — Golden snapshots extracted from hand-crafted code; structural normalization handles allowed differences.
4. **Ready for S196 comparison** — The engine can generate artifacts for comparison against any manual implementation.

## Tradeoffs

1. **Fragments, not files** — The engine produces code fragments, not complete Go files. Integration into source files remains manual until marker-section support is added.
2. **Two artifacts only** — Mapper, tests, config, and smoke test generation are deferred. This limits immediate automation value but reduces risk.
3. **No CI integration yet** — `codegen-drift` and `codegen-golden` CI jobs are not yet created. Tests run locally only.
4. **Evidence layer tested via derivation only** — Candle family derived fields are tested but no golden comparison exists (candle is not a baseline family per S194).

## Open Debts

| Debt | Priority | Trigger |
|------|----------|---------|
| Mapper function generation (A3) | High | When column spec YAML schema is designed |
| Mapper test generation (A4) | Medium | After A3 is validated |
| Config entry generation (A5) | Low | When JSONC manipulation is justified |
| Smoke test phase generation (A6) | Low | When shell template support is added |
| File integration with marker sections | High | When codegen needs to write into source files |
| CI drift detection jobs | Medium | When codegen is used for new families |
| Cross-spec uniqueness validation | Medium | When more than 2 spec files exist |
| Generated file header comments | Low | When codegen writes complete files |

## S196 Preparation

S196 should validate that the engine can be used to compare a new family implementation against generated output. Recommended approach:

1. Write a spec YAML for the next candidate family.
2. Run `codegen generate` for both artifacts.
3. Implement the family manually.
4. Compare manual vs generated for structural equivalence.
5. Document any friction or template gaps.

The engine is ready for this comparison workflow. No changes to the engine are required for S196 unless mapper generation (A3) is in scope.

## Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No broad codegen | PASS — 2 artifacts only |
| No write+read+gateway+smoke all at once | PASS — write-path only, 2 artifacts |
| No hidden architectural decisions in templates | PASS — templates are trivial substitutions |
| No generic framework | PASS — `text/template` only, no abstraction layers |
| Clear documentation of manual vs generated | PASS — coverage doc lists all deferred artifacts |
