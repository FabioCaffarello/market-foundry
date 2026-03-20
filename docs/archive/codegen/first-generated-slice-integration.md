# First Generated Slice Integration

## Purpose

This document records the integration of the first codegen-governed slice into the real monorepo flow. It establishes the pattern, boundaries, and governance rules for generated code participating in runtime.

## Slice Selected: RSI Signal Family (A1 + A2)

### What was integrated

| Artifact | Type | Target File | Golden Snapshot |
|----------|------|-------------|-----------------|
| A1 | consumer_spec | `internal/adapters/nats/signal_registry.go` | `codegen/golden-snapshots/rsi/consumer_spec.go.golden` |
| A2 | pipeline_entry | `cmd/writer/pipeline.go` | `codegen/golden-snapshots/rsi/pipeline_entry.go.golden` |

### Selection rationale

The RSI signal family was chosen because:

1. **Non-evidence layer**: Avoids the evidence-layer naming exception, making it representative of 5 out of 6 layers.
2. **Known abbreviation**: RSI exercises the PascalCase derivation logic with abbreviation handling, proving the spec-to-code path handles non-trivial naming.
3. **Mid-complexity**: Signal layer sits between evidence (simplest) and execution (most complex), providing a representative test point.
4. **Pre-validated**: S196 confirmed the golden snapshot matches the manual code structurally. Zero drift. The integration is a governance change, not a code change.
5. **Minimal blast radius**: The signal layer has no special infrastructure requirements beyond what already exists.

### What was NOT selected

- **Candle (evidence)**: Layer exception would add complexity without representative value.
- **Paper order (execution)**: Ceiling complexity — better reserved for validating after the first integration proves stable.
- **Multiple families**: Guard rail violation — S200 scope is one family only.

## Integration Mechanism

### Marker comments

Generated regions are demarcated with structured marker comments:

```go
// codegen:begin <artifact> family=<name> source=<spec_path>
// ... generated code ...
// codegen:end <artifact> family=<name>
```

These markers serve three purposes:

1. **Ownership boundary**: Code between markers is codegen-governed. Changes must go through the codegen pipeline (spec → generate → compare → insert).
2. **CI extraction**: The `codegen-integrated-check` script extracts marked regions and compares them against golden snapshots.
3. **Audit trail**: The `source=` attribute traces each region back to its YAML spec.

### Governance rules

1. **No manual edits**: Code inside `codegen:begin`/`codegen:end` markers must not be edited manually. If the code needs to change, the spec or template must change first.
2. **Change flow**: Modify spec → regenerate → compare with golden → update golden → update target.
3. **CI enforcement**: `make codegen-integrated` verifies marker regions match golden snapshots. This runs in CI alongside `make codegen-check`.

### Integration manifest

The file `codegen/integrated.yaml` tracks all governed slices, including:
- Family and artifact identifiers
- Spec and golden file paths
- Target file and marker string
- Integration date and stage

This manifest is the single source of truth for what is codegen-governed in the monorepo.

## Verification

### Automated checks

| Check | Command | Scope |
|-------|---------|-------|
| Golden equivalence (all families) | `make codegen-check` | Spec → golden match for all 6 families × 2 artifacts |
| Codegen unit tests | `make codegen-test` | Template rendering, comparison logic, spec validation |
| Integrated slice verification | `make codegen-integrated` | Golden → target match for governed regions |
| Writer compilation | `go build ./cmd/writer/` | Integrated code compiles |
| Writer unit tests | `cd cmd/writer && go test ./...` | Runtime behavior unchanged |

### Results at integration time

- `codegen check-all`: 12/12 PASS
- `codegen-integrated`: 2/2 PASS
- Writer compilation: clean
- Writer unit tests: PASS

## What remains manual

The following artifacts for the RSI family are NOT codegen-governed:

| Artifact | File | Reason |
|----------|------|--------|
| Mapper function (A3) | `cmd/writer/mappers.go` | Requires domain column knowledge; `domain.columns` spec extension not yet implemented |
| Mapper unit tests (A4) | `cmd/writer/mappers_test.go` | Depends on A3 |
| Config entry (A5) | `deploy/configs/writer.jsonc` | JSONC tooling not yet implemented |
| Smoke test phase (A6) | `scripts/smoke-analytical-e2e.sh` | Shell template engine deferred |
| Domain event type | `internal/domain/signal/` | Permanently manual — creative design decision |
| NATS stream/registry structure | `internal/adapters/nats/signal_registry.go` | Only the writer consumer spec is governed; registry structure stays manual |
| ClickHouse schema | `deploy/migrations/` | Permanently manual — DDL requires human judgment |
| Reader/handler/route | `internal/adapters/clickhouse/`, `internal/interfaces/http/` | Read-path artifacts are Tier 2, not yet authorized |

## Risks and mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Marker comments accidentally deleted | Medium | CI gate (`codegen-integrated`) fails if markers missing |
| Manual edit inside governed region | Medium | CI gate detects drift; PR review checklist |
| Marker format inconsistency | Low | Fixed format in manifest; script validates exact match |
| False confidence from narrow slice | Medium | Document explicitly limits scope to A1+A2 of one family |

## Relationship to existing families

The other 5 families (candle, rsi_oversold, mean_reversion_entry, position_exposure, paper_order) remain fully manual. Their golden snapshots exist and pass `codegen check-all`, but they have no governance markers in target files.

The path to governing additional families is:
1. Add markers to target files
2. Add entries to `codegen/integrated.yaml`
3. Add checks to `scripts/codegen-integrated-check.sh`
4. Verify with `make codegen-integrated`

This is a deliberate, per-family opt-in — not a bulk migration.
