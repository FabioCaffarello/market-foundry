# Stage S194 — Manual-to-Generated Equivalence Baseline Report

> Defines and materializes the baseline for proving codegen equivalence against hand-crafted families.

## Executive Summary

S194 establishes how market-foundry will prove that code generation produces artifacts equivalent to the 6 hand-crafted analytical families. Two families — **RSI (L2 Signal)** and **Paper Order (L6 Execution)** — were selected as the equivalence baseline, bracketing the full complexity spectrum from simplest (12 columns, 1 JSON) to most complex (20 columns, 4 JSON, 2 enums, 2 correlation IDs). The stage defines structural vs semantic equivalence rules for all 6 Tier 1 artifact types, a golden snapshot comparison strategy, and a drift detection model. The baseline is ready for a minimal codegen engine in S195.

## Baseline Selection

### Two-Family Bracket Strategy

| Family | Layer | Columns | JSON Fields | Enums | Why Selected |
|--------|-------|---------|-------------|-------|-------------|
| **RSI** | L2 Signal | 12 | 1 | 0 | Minimal — fewest transforms, shared table, simplest case |
| **Paper Order** | L6 Execution | 20 | 4 | 2 | Ceiling — every transform type exercised, widest column spread |

**Rationale**: Any within-layer family's complexity falls between these two. If codegen reproduces both correctly, intermediate families (RSI Oversold, Mean Reversion, Position Exposure) are covered by interpolation. Candle (L1) is deferred because it's structurally identical despite being a unique layer.

### Deferred: Candle, RSI Oversold, Mean Reversion Entry, Position Exposure

These 4 families remain available as expansion evidence if the RSI–Paper Order bracket proves insufficient during S195.

## Equivalence Strategy

### Three Layers of Equivalence

```
Layer 1: Structural (primary gate)
  Normalized AST/token equality after gofmt + import sort + comment strip
  → Mechanical, automated, fast

Layer 2: Semantic (secondary validation)
  Domain-aware checks: column order matches DDL, transforms match types,
  NATS subjects resolve correctly
  → Requires domain rules, semi-automated

Layer 3: Behavioral (out of scope for equivalence)
  Runtime correctness: compilation, tests, integration, smoke
  → Covered by existing CI and smoke tests, not by equivalence comparison
```

### Structural vs Semantic Distinction

| Structural = tokens match | Semantic = behavior matches |
|--------------------------|---------------------------|
| After normalization, byte equality | Even with different variable names or ordering |
| Primary gate for all artifacts | Secondary gate for pipeline entries and smoke tests |
| Automated via diff | Requires rule-based validation |
| No false positives | May accept cosmetic differences |

**Key ruling**: Column order in mapper `[]any` return is **structural, not semantic**. ClickHouse bulk insert uses positional binding, so column order is load-bearing.

## Artifacts Under Comparison

### 6 Artifact Types × 2 Families = 12 Comparison Points

| # | Artifact | Equivalence Type | Comparison Method |
|---|----------|-----------------|-------------------|
| A1 | Consumer spec function | Structural | AST-normalized diff |
| A2 | Pipeline entry | Structural + Semantic | AST diff + domain rule check |
| A3 | Mapper function | Structural + Semantic | AST diff + column order + transform verification |
| A4 | Mapper unit tests | Structural | AST diff (generated ≥ golden) |
| A5 | Config array entry | Structural | JSON parse + deep equal |
| A6 | Smoke test phase | Structural | Line-normalized diff |

### Allowed vs Forbidden Differences (Summary)

| Allowed | Forbidden |
|---------|-----------|
| Import order | Missing/extra columns |
| Whitespace | Wrong transform function |
| Comments | Different SQL shape |
| Lambda variable names | Missing function |
| Extra test cases | Wrong function name |
| Config array order | Missing test cases |

## Golden Outputs Strategy

### Architecture

```
codegen/golden/*.yaml              → Specs describing existing families (manually authored)
codegen/golden-snapshots/{family}/ → Frozen extracts of hand-crafted artifacts
codegen/templates/*.tmpl           → Templates producing generated artifacts
```

### Comparison Workflow

```
1. Load golden spec
2. Render templates → candidate artifacts
3. Normalize candidate + golden snapshot
4. Byte-compare normalized outputs
5. Report PASS/FAIL per artifact
```

### Drift Detection

| Point | Trigger | CI Job |
|-------|---------|--------|
| Template change | Regenerate golden specs → compare to snapshots | `codegen-golden` |
| Generated file change | Regenerate from specs → compare to committed | `codegen-drift` |
| Any PR | Validate spec schemas + headers | `codegen-lint` |

### Drift Severity

- **CRITICAL**: Missing field, wrong SQL, wrong validation → blocks merge
- **WARNING**: Comment difference, formatting → logged, no block
- **INFO**: Cosmetic within tolerance → logged only

## Decisions Made

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Two-family bracket, not all six | Sufficient coverage with minimal maintenance cost |
| D2 | Structural equivalence as primary gate | Eliminates ambiguity; semantic rules only for known exceptions |
| D3 | Golden snapshots extracted once, frozen | Hand-crafted families are post-freeze; snapshots won't drift |
| D4 | Column order is structural, not semantic | ClickHouse positional binding makes order load-bearing |
| D5 | Generated tests must be ≥ golden, not exactly equal | More coverage is always acceptable |
| D6 | Normalization strips comments | Comments carry no behavioral weight; comparing them creates false failures |

## Limits and Deferred Items

### What S194 Covers

- Tier 1 (write-path) artifact equivalence rules
- Named mapper reuse pattern (mapper already exists)
- Within-layer expansion only (shared table)
- 2 baseline families (RSI + Paper Order)
- Structural + semantic rule definitions
- Golden snapshot strategy and drift model

### What S194 Does NOT Cover

| Deferred Item | Why | When |
|---------------|-----|------|
| Tier 2 artifact equivalence (read-path) | Not yet authorized | When Tier 2 is triggered |
| Generated mapper equivalence (`mapper: "generate"`) | No generated-mapper family in baseline | First generated-mapper family |
| New-layer DDL equivalence | No new-layer expansion yet | First Tier 2 family |
| Performance benchmarking | Not an equivalence concern | Operational validation phase |
| Cross-family interaction testing | Families are independent by design | Not expected to be needed |
| Golden snapshot extraction tooling | Manual extraction sufficient for 2 families | S195 if automation needed |

### Open Risks

| Risk | Mitigation |
|------|-----------|
| Normalization rules may be too strict or too loose | Calibrate during S195 when first real comparison runs |
| Golden snapshots may not capture all relevant structure | Extraction boundaries defined per artifact type |
| Two families may not cover all edge cases | Expansion families available; bracket strategy provides safety margin |

## Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Clear equivalence baseline exists | ✅ | RSI + Paper Order selected with rationale |
| Generated vs manual comparison defined | ✅ | Golden snapshot workflow with normalization pipeline |
| Structural vs semantic distinguishable | ✅ | Per-artifact rules with allowed/forbidden matrices |
| Drift is no longer vague | ✅ | Three detection points, three severity levels, response protocol |
| Ready for minimal engine in S195 | ✅ | Templates can target golden specs; comparison workflow is mechanical |

## Artifacts Produced

| Artifact | Path |
|----------|------|
| Equivalence baseline | `docs/architecture/manual-to-generated-equivalence-baseline.md` |
| Golden outputs strategy | `docs/architecture/codegen-golden-outputs-and-comparison-strategy.md` |
| Semantic vs structural rules | `docs/architecture/codegen-equivalence-scope-semantic-vs-structural-rules.md` |
| This report | `docs/stages/stage-s194-manual-to-generated-equivalence-baseline-report.md` |

## S195 Preparation Recommendations

### Minimum Viable Engine Scope

S195 should implement the **smallest possible codegen engine** that:

1. Parses a golden spec YAML (`codegen/golden/rsi.yaml`)
2. Renders one template (`consumer_spec.go.tmpl`) against it
3. Compares output to golden snapshot (`codegen/golden-snapshots/rsi/consumer_spec.go.golden`)
4. Reports PASS/FAIL

### Recommended S195 Sequence

```
Step 1: Extract golden snapshots for RSI and Paper Order (manual, one-time)
Step 2: Author golden specs for both families (manual, using S194 schema)
Step 3: Implement spec parser (YAML → Go struct)
Step 4: Implement simplest template (consumer_spec.go.tmpl)
Step 5: Implement normalization pipeline (gofmt + imports + comment strip)
Step 6: Implement comparison (normalized byte equality)
Step 7: Run against RSI golden → calibrate normalization rules
Step 8: Run against Paper Order golden → validate bracket coverage
Step 9: Extend to remaining 5 templates if Step 7-8 pass
```

### S195 Entry Gate

S195 can begin when:
- This S194 report is reviewed and accepted
- No open questions about equivalence rules remain
- The 12 comparison points (2 families × 6 artifacts) are understood and agreed

### Critical S195 Constraint

**Do NOT generate new families in S195**. S195 proves the engine against existing families. First new-family generation is S196 or later, only after golden equivalence is confirmed for both baseline families across all 6 artifact types.
