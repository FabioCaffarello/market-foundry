# Codegen Golden Outputs and Comparison Strategy

> S194 — Defines how golden outputs are structured, how comparison works mechanically, and how drift is detected.

## 1. Golden Output Architecture

### Directory Structure

```
codegen/
├── golden/                    # Golden specs — manually authored, never regenerated
│   ├── rsi.yaml               # Describes existing RSI family
│   └── paper_order.yaml       # Describes existing Paper Order family
├── golden-snapshots/          # Frozen snapshots of hand-crafted artifacts
│   ├── rsi/
│   │   ├── consumer_spec.go.golden
│   │   ├── pipeline_entry.go.golden
│   │   ├── mapper.go.golden
│   │   ├── mapper_test.go.golden
│   │   ├── config_entry.jsonc.golden
│   │   └── smoke_phase.sh.golden
│   └── paper_order/
│       ├── consumer_spec.go.golden
│       ├── pipeline_entry.go.golden
│       ├── mapper.go.golden
│       ├── mapper_test.go.golden
│       ├── config_entry.jsonc.golden
│       └── smoke_phase.sh.golden
└── templates/                 # Templates — human-owned, used for generation
    ├── consumer_spec.go.tmpl
    ├── pipeline_entry.go.tmpl
    ├── mapper.go.tmpl
    ├── mapper_test.go.tmpl
    ├── config_entry.jsonc.tmpl
    └── smoke_phase.sh.tmpl
```

### Golden Snapshot Extraction

Golden snapshots are **extracted once** from the current hand-crafted codebase and frozen. They represent the "known good" output that codegen must reproduce.

#### Extraction Rules

1. **Isolate the artifact**: extract only the relevant function/block, not the entire file.
2. **Normalize**: run `gofmt` on Go snippets; normalize whitespace in shell/JSON.
3. **Strip comments**: remove non-functional comments (but preserve `// Code generated` headers if present).
4. **Freeze**: commit to `codegen/golden-snapshots/` with a clear commit message.
5. **Never modify**: once committed, golden snapshots are immutable. If the hand-crafted code changes, create a new snapshot version.

#### Extraction Boundaries Per Artifact

| Artifact | Extraction Boundary |
|----------|-------------------|
| Consumer spec | Full function body: `func Writer{Name}Consumer() ConsumerSpec { ... }` |
| Pipeline entry | Single struct literal in the `familyPipelines` slice |
| Mapper | Full function body: `func map{Layer}Row(e ...) []any { ... }` |
| Mapper test | All `Test` functions for the mapper |
| Config entry | The specific array value in the JSONC config |
| Smoke phase | The curl + assertion block for the family |

## 2. Comparison Strategy

### Two-Phase Comparison

```
Phase 1: Structural Comparison (automated, fast)
  ├── Token-level AST diff for Go artifacts
  ├── JSON structural diff for config artifacts
  └── Line-level diff for shell artifacts

Phase 2: Semantic Comparison (semi-automated, targeted)
  ├── Column order matches DDL declaration order
  ├── Transform functions match column types
  ├── NATS subject pattern matches event routing
  └── Enable condition matches config array semantics
```

### Comparison Tool Selection

| Artifact Type | Comparison Method | Tool |
|--------------|-------------------|------|
| Go functions | AST-normalized diff | `go/ast` parse → canonical print → diff |
| Go tests | AST-normalized diff | Same as above |
| JSONC config | JSON parse → structural diff | Strip comments → `encoding/json` → deep equal |
| Shell scripts | Line-normalized diff | Strip blank lines → strip comments → diff |

### Normalization Pipeline

Before comparison, both golden snapshot and generated output pass through:

```
1. gofmt (Go files only)
2. Import sorting (Go files only — goimports canonical order)
3. Whitespace normalization (trim trailing, normalize newlines)
4. Comment stripping (remove // comments except codegen headers)
5. Blank line collapsing (max 1 consecutive blank line)
```

After normalization, byte-level equality is the comparison operator.

## 3. Drift Detection Model

### What Is Drift

Drift occurs when:
- A generated artifact diverges from what the spec + template would produce
- A hand-crafted artifact changes but its golden snapshot is not updated
- A template changes but affected generated artifacts are not regenerated

### Drift Detection Points

| Detection Point | Trigger | Action |
|-----------------|---------|--------|
| **CI: codegen-golden** | Template or golden spec change | Regenerate from golden specs → compare to golden snapshots |
| **CI: codegen-drift** | Generated file change | Regenerate from family specs → compare to committed files |
| **CI: codegen-lint** | Any PR | Validate spec schema + verify headers |
| **Manual: baseline refresh** | Hand-crafted code change | Re-extract golden snapshot → update frozen snapshot |

### Drift Severity Levels

| Level | Definition | CI Behavior | Example |
|-------|-----------|-------------|---------|
| **CRITICAL** | Missing field, wrong SQL shape, wrong validation | Hard fail, block merge | Generated mapper missing a column |
| **WARNING** | Comment difference, import order, formatting | Soft fail, log warning | Different comment text |
| **INFO** | Cosmetic difference within normalization tolerance | Pass, log info | Extra blank line |

### Drift Response Protocol

```
CRITICAL drift detected:
  1. Identify source: template bug vs spec error vs extraction error
  2. Fix at source (template or spec, never patch generated file)
  3. Regenerate
  4. Re-run comparison
  5. Commit fix + regenerated output together

WARNING drift detected:
  1. Log for review
  2. No blocking action
  3. Consider tightening normalization rules if recurring

INFO drift detected:
  1. Log only
  2. No action required
```

## 4. Golden Test Workflow

### End-to-End Golden Test Sequence

```
┌──────────────────────────────────────────────────────────────┐
│  1. Load golden spec:  codegen/golden/rsi.yaml               │
│                                                              │
│  2. Run template engine:                                     │
│     for each template in codegen/templates/*.tmpl:            │
│       render(template, spec) → candidate artifact            │
│                                                              │
│  3. Load golden snapshot:                                    │
│     codegen/golden-snapshots/rsi/{artifact}.golden           │
│                                                              │
│  4. Normalize both:                                          │
│     normalize(candidate) → normalized_candidate              │
│     normalize(snapshot)  → normalized_snapshot                │
│                                                              │
│  5. Compare:                                                 │
│     if normalized_candidate == normalized_snapshot:           │
│       → PASS                                                 │
│     else:                                                    │
│       → FAIL + emit diff                                     │
│                                                              │
│  6. Repeat for all artifacts × all golden families           │
└──────────────────────────────────────────────────────────────┘
```

### Golden Test Matrix

| Golden Family | A1 Consumer | A2 Pipeline | A3 Mapper | A4 Tests | A5 Config | A6 Smoke |
|--------------|:-----------:|:-----------:|:---------:|:--------:|:---------:|:--------:|
| RSI          | ✅          | ✅          | ✅        | ✅       | ✅        | ✅       |
| Paper Order  | ✅          | ✅          | ✅        | ✅       | ✅        | ✅       |

Total comparisons: **12** (2 families × 6 artifacts).

## 5. Snapshot Versioning

### Version Scheme

Golden snapshots carry a version tag in their filename when the hand-crafted source changes:

```
codegen/golden-snapshots/rsi/consumer_spec.go.golden      # Current
codegen/golden-snapshots/rsi/consumer_spec.go.golden.v1    # If archived
```

### Version Rules

1. Only the **current** `.golden` file is used for comparison.
2. Archived versions (`.v1`, `.v2`) are kept for audit trail only.
3. A snapshot update requires a commit message explaining what changed in the hand-crafted source.
4. Snapshot updates are rare — they only happen when the hand-crafted families themselves change (which should be near-never post-freeze).

## 6. Comparison Outputs

### Success Report Format

```
=== Golden Equivalence Report ===
Family: rsi
  A1 consumer_spec:  PASS (0 diffs after normalization)
  A2 pipeline_entry: PASS (0 diffs after normalization)
  A3 mapper:         PASS (0 diffs after normalization)
  A4 mapper_test:    PASS (0 diffs after normalization)
  A5 config_entry:   PASS (0 diffs after normalization)
  A6 smoke_phase:    PASS (0 diffs after normalization)

Family: paper_order
  A1 consumer_spec:  PASS (0 diffs after normalization)
  A2 pipeline_entry: PASS (0 diffs after normalization)
  A3 mapper:         PASS (0 diffs after normalization)
  A4 mapper_test:    PASS (0 diffs after normalization)
  A5 config_entry:   PASS (0 diffs after normalization)
  A6 smoke_phase:    PASS (0 diffs after normalization)

Result: 12/12 PASS — baseline equivalence confirmed
```

### Failure Report Format

```
=== Golden Equivalence Report ===
Family: rsi
  A1 consumer_spec:  PASS
  A2 pipeline_entry: PASS
  A3 mapper:         FAIL (CRITICAL)
    - Line 5: expected "parseFloat(s.Value)" got "s.Value"
    - Missing transform for column "value"
  ...

Result: 11/12 PASS, 1/12 FAIL — baseline equivalence BROKEN
Action: Fix template codegen/templates/mapper.go.tmpl line 12
```

## 7. Limits of This Strategy

### What Golden Tests Prove

- Template output matches hand-crafted code for known families
- No structural regression when templates change
- Normalization rules are sufficient to absorb cosmetic differences

### What Golden Tests Do NOT Prove

- Correctness of a **new** family spec (only format validation covers that)
- Runtime behavior (compilation and integration tests cover that)
- Performance equivalence
- Operational equivalence under failure conditions
