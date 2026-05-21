# Codegen Slice 01 CI Validation Strategy

> S196 deliverable. Defines how codegen validation enters CI and what it gates.

## Design Principle

CI **verifies** that committed code matches codegen output. CI never **generates** code. All generated artifacts are committed to the repository and reviewed in PRs as readable Go code.

## CI Job: `codegen-golden`

### Trigger
- Every push to `main`
- Every pull request targeting `main`

### Steps
1. `make codegen-check` — Runs `codegen check-all` which:
   - Loads every YAML spec in `codegen/families/`
   - Renders all supported artifact templates against each spec
   - Compares generated output to golden snapshots using structural normalization
   - Fails on any structural mismatch
2. `make codegen-test` — Runs `go test ./...` in the `codegen/` module which:
   - Validates spec parsing and derived field computation
   - Validates template rendering content assertions
   - Runs per-family golden comparison tests
   - Runs `TestCheckAllFamilies` cross-validation gate

### Duration
- `codegen-check`: ~2s (6 families × 2 artifacts = 12 comparisons)
- `codegen-test`: ~1s (26 unit tests)
- Total: ~3s

### Failure Mode
Hard fail — if any golden comparison fails, the CI job blocks merge. This prevents:
- Template changes that break existing family equivalence
- Spec changes that produce incorrect naming
- Normalization changes that mask real drift

## What This Job Gates

| Change Type | Detected By | Example |
|------------|------------|---------|
| Template regression | Golden comparison | Template edit breaks RSI consumer spec |
| Spec error | Spec validation | Typo in NATS subject |
| Naming bug | Derived field tests | PascalCase abbreviation missing |
| New family without golden | Check-all failure | Spec added but no golden snapshot |
| Golden drift from template | Golden comparison | Golden snapshot edited manually |

## What This Job Does NOT Gate

| Concern | Why Not | When |
|---------|---------|------|
| Live-code drift | Generated code is fragments, not integrated files | Future: when file integration is implemented |
| Mapper correctness | A3 not yet in codegen scope | Slice 02+ |
| Runtime behavior | Codegen is dev-time only | Covered by unit tests and smoke tests |
| Config correctness | A5 not yet in scope | Slice 02+ |

## CI Position

```
ci.yml jobs:
  unit-tests          → runs all Go module tests (including codegen/)
  codegen-golden      → runs codegen-specific check-all + tests  [NEW]
  smoke-analytical    → E2E integration proof (needs: unit-tests)
```

The `codegen-golden` job runs independently of `unit-tests` since it has no dependency on build artifacts. Both can run in parallel.

## Local Developer Workflow

```bash
# Quick check: does my template change break anything?
make codegen-check

# Full test suite including content assertions
make codegen-test

# Generate output for inspection
cd codegen && CODEGEN_ROOT=. go run . generate families/rsi.yaml consumer_spec
```

## Adding a New Family (Workflow)

1. Author `codegen/families/{family}.yaml` following S193 schema
2. Extract golden snapshots from existing code into `codegen/golden-snapshots/{family}/`
3. Add fixture function to `codegen/render_test.go`
4. Add per-artifact golden tests
5. Run `make codegen-check` — should show the new family passing
6. Commit spec + golden snapshots + tests together
7. CI validates on PR

## Evolving the CI Strategy

### Near-term (Slice 02+)
When additional artifacts (A3: mapper, A4: mapper tests) are added to the engine:
- Golden snapshots extend to `mapper.go.golden` and `mapper_test.go.golden`
- `SupportedArtifacts()` adds `"mapper"` and `"mapper_test"`
- `check-all` automatically covers the new artifact types
- No CI config changes needed — the check-all loop is artifact-agnostic

### Medium-term (File Integration)
When the engine gains the ability to write into live source files:
- Add a `codegen-drift` CI job that regenerates and diffs against committed files
- This catches cases where someone edits a generated section manually
- Requires marker section implementation first

### Long-term (First Generated Family)
When the first new family is generated (not retrofitted):
- The family's spec is committed
- Generated files are committed
- CI validates both golden equivalence AND that the generated files compile and pass tests
- The smoke-analytical job validates runtime behavior end-to-end

## File Changes

| File | Change |
|------|--------|
| `.github/workflows/ci.yml` | Added `codegen-golden` job |
| `Makefile` | Added `codegen-check` and `codegen-test` targets |
