# Codegen Validation and CI Strategy

> **Consolidated document.** Merges content from:
> - `codegen-validation-drift-and-ci-strategy.md`
> - `codegen-drift-findings-and-equivalence-results.md` (S196)
> - `codegen-golden-outputs-and-comparison-strategy.md` (S194)
> - `codegen-slice-01-ci-validation-strategy.md` (S196)
> - `codegen-slice-01-coverage-and-non-coverage.md`
>
> Originals archived to `docs/archive/codegen/`.

---

## 1. Validation Strategy

### Golden Test Equivalence

The primary validation mechanism is **golden test equivalence**: the codegen tool, given a family spec describing an existing hand-crafted family, must produce output that is structurally equivalent to the hand-crafted code.

Equivalence is tested at the **semantic structure** level, not byte-for-byte identity.

| Dimension | Allowed Difference |
|-----------|-------------------|
| Import ordering | Go-standard reordering |
| Whitespace/formatting | `gofmt`-normalized differences |
| Comment phrasing | Template comments vs hand-written |
| Variable naming in lambdas | Allowed if same field access pattern |
| Test case ordering | Allowed to differ |
| Test case coverage | Generated must be >= hand-crafted |

Unacceptable differences: missing fields, different SQL shape, different validation rules, different error handling, extra/missing behavioral imports.

### Compilation Gate

Every generated Go file must: compile (`go build ./...`), pass `go vet ./...`, and not introduce new lint warnings.

### Unit Test Gate

Generated test files must pass against their implementation, cover the same cases as golden references, and have no codegen runtime dependency.

### Integration Validation

For the first generated family (EMA Crossover):
1. Writer pipeline entry activates and connects to NATS
2. Events flow through consumer -> mapper -> inserter -> ClickHouse
3. Existing `SignalReader` returns results via `type=ema_crossover`
4. Smoke test validates HTTP endpoint end-to-end
5. Health trackers report metrics for the new pipeline

---

## 2. Golden Output Architecture

### Directory Structure

```
codegen/
+-- golden/                    # Golden specs -- manually authored, never regenerated
+-- golden-snapshots/          # Frozen snapshots of hand-crafted artifacts
|   +-- {family}/
|       +-- consumer_spec.go.golden
|       +-- pipeline_entry.go.golden
|       +-- mapper.go.golden
|       +-- mapper_test.go.golden
|       +-- config_entry.jsonc.golden
|       +-- smoke_phase.sh.golden
+-- templates/                 # Templates -- human-owned, used for generation
    +-- consumer_spec.go.tmpl
    +-- pipeline_entry.go.tmpl
    +-- ...
```

### Golden Snapshot Extraction Rules

1. **Isolate the artifact**: extract only the relevant function/block, not the entire file.
2. **Normalize**: run `gofmt` on Go snippets; normalize whitespace in shell/JSON.
3. **Strip comments**: remove non-functional comments (preserve codegen headers if present).
4. **Freeze**: commit to `codegen/golden-snapshots/`. Once committed, golden snapshots are immutable.
5. **Never modify**: if hand-crafted code changes, create a new snapshot version.

### Extraction Boundaries Per Artifact

| Artifact | Extraction Boundary |
|----------|-------------------|
| Consumer spec | Full function body: `func Writer{Name}Consumer() ConsumerSpec { ... }` |
| Pipeline entry | Single struct literal in `familyPipelines` slice |
| Mapper | Full function body: `func map{Layer}Row(e ...) []any { ... }` |
| Mapper test | All `Test` functions for the mapper |
| Config entry | Specific array value in JSONC config |
| Smoke phase | curl + assertion block for the family |

### Golden Test Procedure

```
1. Load golden spec (e.g., codegen/golden/rsi.yaml)
2. Run codegen tool -> produces output files in temp directory
3. Both outputs through normalization pipeline:
   a. gofmt (Go files only)
   b. Import sorting (goimports canonical order)
   c. Whitespace normalization
   d. Comment stripping (except codegen headers)
   e. Blank line collapsing
4. Byte-level equality comparison after normalization
5. PASS if all output files are within equivalence bounds
6. FAIL with diff if any structural deviation detected
```

### Snapshot Versioning

Only the current `.golden` file is used for comparison. Archived versions (`.v1`, `.v2`) are kept for audit trail only. Snapshot updates require a commit message explaining what changed. Updates are rare -- only when hand-crafted families change.

---

## 3. Equivalence Results (S196)

### Summary

| Metric | Value |
|--------|-------|
| Families validated | 6 / 6 |
| Artifacts per family | 2 (consumer_spec, pipeline_entry) |
| Total comparison points | 12 |
| Structural passes | 12 / 12 (100%) |
| Cosmetic drift instances | 3 (all INFO severity) |
| Dangerous drift instances | 0 |

### Equivalence Matrix

| Family | consumer_spec | pipeline_entry |
|--------|:------------:|:--------------:|
| candle (evidence) | PASS | PASS |
| rsi (signal) | PASS | PASS |
| rsi_oversold (decision) | PASS | PASS |
| mean_reversion_entry (strategy) | PASS | PASS |
| position_exposure (risk) | PASS | PASS |
| paper_order (execution) | PASS | PASS |

### Cosmetic Drift Instances (All INFO, No Action Required)

- **D1**: Comment phrasing variation in consumer spec doc comments (template wraps to 2 lines vs single-line in live code; template uses raw family_name vs human-readable form)
- **D2**: Section comment dash decoration length in pipeline entries (fixed short suffix vs variable-length trailing dashes)
- **D3**: Evidence layer comment omits "evidence" in `WriterCandleConsumer` doc comment

All handled correctly by normalization pipeline -- zero false positives or false negatives.

### Current Artifact Coverage Ceiling

| Artifact | Can Generate? | Blocking Reason |
|----------|:------------:|-----------------|
| A1: Consumer spec | Yes | -- |
| A2: Pipeline entry | Yes | -- |
| A3: Mapper function | No | Requires column-order knowledge, type transforms, DDL awareness |
| A4: Mapper unit tests | No | Depends on A3 |
| A5: Config entry | No | JSONC manipulation tooling |
| A6: Smoke test phase | No | Shell script generation |

---

## 4. Drift Detection

### What Is Drift

Drift occurs when:
1. A template is modified but existing generated files are not regenerated.
2. A generated file is manually edited, breaking the spec -> output contract.
3. A dependency of generated code changes in a way that invalidates the output.

### Drift Detection Mechanisms

**Header Comment Check**: CI verifies all files with codegen headers have matching spec files and matching template versions.

**Regeneration Comparison**: Authoritative drift check -- regenerate all family specs and compare against committed files.

**Spec-to-Code Consistency**: Every spec has corresponding artifacts; every artifact references a valid spec; no orphaned files.

### Drift Severity Levels

| Level | Definition | CI Behavior |
|-------|-----------|-------------|
| CRITICAL | Missing field, wrong SQL, wrong validation | Hard fail, block merge |
| WARNING | Comment difference, import order, formatting | Soft fail, log warning |
| INFO | Cosmetic within normalization tolerance | Pass, log info |

### Drift Response Protocol

| Drift Type | Severity | Response |
|------------|----------|----------|
| Template version mismatch | Warning | Regenerate and review diff |
| Manual edit to generated file | Error | Revert; apply fix to template or spec |
| Missing generated file | Error | Regenerate from spec |
| Orphaned generated file | Warning | Delete or create spec |
| Domain type change affecting mapper | Error | Update spec columns; regenerate |

---

## 5. CI Strategy

### Principle: Transparent, Not Opaque

Codegen in CI is a **verification step, not a build step**. Generated files are committed. CI verifies committed files match codegen output. CI never generates files during build.

### CI Pipeline Integration

```
ci.yml jobs:
  unit-tests           -> runs all Go module tests (including codegen/)
  codegen-golden       -> runs codegen-specific check-all + tests  [NEW]
  smoke-analytical     -> E2E integration proof (needs: unit-tests)
```

The `codegen-golden` job runs independently of `unit-tests` (no dependency on build artifacts; both run in parallel).

### CI Job: codegen-golden

**Trigger**: Every push to `main` and every PR targeting `main`.

**Steps**:
1. `make codegen-check` -- loads every YAML spec, renders all supported artifact templates, compares to golden snapshots using structural normalization. Fails on structural mismatch.
2. `make codegen-test` -- runs `go test ./...` in `codegen/` module (spec parsing, template rendering, per-family golden comparison, `TestCheckAllFamilies` cross-validation gate).

**Duration**: ~3s total (check ~2s + test ~1s).

**Failure Mode**: Hard fail -- blocks merge on any golden comparison failure.

### What CI Gates

| Change Type | Detected By |
|------------|------------|
| Template regression | Golden comparison |
| Spec error | Spec validation |
| Naming bug | Derived field tests |
| New family without golden | Check-all failure |
| Golden drift from template | Golden comparison |

### What CI Does NOT Do

1. Does not run codegen to produce build artifacts.
2. Does not auto-fix drift.
3. Does not validate ClickHouse integration (smoke test scope).
4. Does not enforce codegen for manual families.

---

## 6. Slice 01 Coverage

### Artifacts Generated

| # | Artifact | Template | Target |
|---|----------|----------|--------|
| A1 | Consumer spec function | `consumer_spec.go.tmpl` | `internal/adapters/nats/{layer}_registry.go` |
| A2 | Pipeline entry struct | `pipeline_entry.go.tmpl` | `cmd/writer/pipeline.go` |

### Baseline Families

| Family | Layer | Complexity | Role |
|--------|-------|-----------|------|
| RSI | Signal (L2) | Minimal -- 12 columns, 1 JSON | Lower bound |
| Paper Order | Execution (L6) | Ceiling -- 20 columns, 4 JSON, 2 enums | Upper bound |

### Capabilities Proven

1. YAML spec parsing per S193 frozen schema
2. Naming derivation (PascalCase with abbreviation awareness, layer-specific exceptions)
3. Template rendering via `text/template`
4. Golden comparison with structural normalization
5. CLI interface: validate, generate, compare, check-all

### Deferred Tier 1 Artifacts

| # | Artifact | Reason Deferred | Prerequisite |
|---|----------|----------------|--------------|
| A3 | Mapper function | Requires column-order knowledge, DDL-aware generation | Column spec extension |
| A4 | Mapper unit tests | Depends on A3 | A3 first |
| A5 | Config entry | JSONC manipulation tooling | JSON/JSONC template support |
| A6 | Smoke test phase | Shell script generation | Shell template support |

---

## 7. Validation Lifecycle

### New Within-Layer Family (Tier 1)

```
1. Author creates spec: codegen/families/{family}.yaml
2. Run codegen locally -> generates write-path artifacts
3. Developer reviews generated output
4. Commit spec + generated files
5. CI: codegen-lint (valid), codegen-drift (match), go build (compiles), go test (pass)
6. PR review (generated code is committed, readable)
7. Merge -> deploy -> smoke test validates end-to-end
```

### Template Modification

```
1. Modify template in codegen/templates/
2. Run golden tests locally -> verify equivalence with 6 families
3. Regenerate all family specs -> update generated files
4. Commit template + regenerated files
5. CI: codegen-golden (equivalence), codegen-drift (up to date), go build/test (no regressions)
6. PR review: template diff AND regenerated diffs
7. Merge
```

### Adding a New Family

```
1. Author codegen/families/{family}.yaml following S193 schema
2. Extract golden snapshots into codegen/golden-snapshots/{family}/
3. Add fixture function to codegen/render_test.go
4. Add per-artifact golden tests
5. Run make codegen-check
6. Commit spec + golden snapshots + tests
7. CI validates on PR
```

---

## 8. Maturity Model

| Phase | Scope |
|-------|-------|
| Phase 1 (S193) | Tier 1 templates, golden tests against 6 families, first generated family (EMA Crossover) |
| Phase 2 (post-S194) | codegen-lint and codegen-drift in CI, automated golden test on template changes, header tracking |
| Phase 3 (future) | Tier 2 templates, full read-path generation, ~90% family expansion cost coverage |

---

## Related Documents

- [codegen-specification-and-schema.md](codegen-specification-and-schema.md) -- frozen spec, schema, ownership
- [codegen-boundaries-and-governance.md](codegen-boundaries-and-governance.md) -- anti-patterns, boundaries, governance
- [codegen-path-stabilization-or-freeze-decision.md](codegen-path-stabilization-or-freeze-decision.md) -- active decision record
