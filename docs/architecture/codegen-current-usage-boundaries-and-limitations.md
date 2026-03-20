# Codegen: Current Usage Boundaries and Limitations

**Stage**: S207
**Date**: 2026-03-20
**Status**: Stabilized — controlled use within boundaries below

---

## 1. What Can Be Used Today

### Stable and CI-Gated

| Capability | Status | Confidence |
|------------|--------|------------|
| `go run codegen validate <spec>` | Stable | High — validates all 14 required fields |
| `go run codegen validate-all` | Stable | High — per-spec + cross-spec uniqueness |
| `go run codegen generate <spec> <artifact>` | Stable | High — deterministic output for A1, A2 |
| `go run codegen compare <spec> <artifact>` | Stable | High — structural normalization + line diff |
| `go run codegen check-all` | Stable | High — 14/14 golden comparisons pass |
| `make codegen-validate-all` | Stable | CI-blocking gate |
| `make codegen-check` | Stable | CI-blocking gate |
| `make codegen-test` | Stable | CI-blocking gate |
| `make codegen-integrated` | Stable | CI-blocking gate |
| Golden snapshot comparison | Stable | Structural normalization handles comment/whitespace variance |
| Spec cross-validation | Stable | Prevents durable/subject/name collisions |

### Usable for New Families (with conditions)

Adding a new family via codegen is permitted if **all** conditions are met:

1. The family fits the frozen S193 spec schema (14 required fields, 4 sections)
2. The family is Tier 1 write-path only
3. The family uses one of the 6 known layers (evidence, signal, decision, strategy, risk, execution)
4. A golden snapshot is generated and committed for both A1 and A2
5. The golden snapshot passes `check-all` before integration
6. Integration uses codegen:begin/end markers in the target file
7. An entry is added to `integrated.yaml`
8. The integrated check passes before merge

---

## 2. What Cannot Be Used

### Not Implemented

| Capability | Status | Blocker |
|------------|--------|---------|
| Mapper generation (A3) | Not available | Requires `domain.columns` spec extension |
| Mapper test generation (A4) | Not available | Depends on A3 |
| Config entry generation (A5) | Not available | JSONC tooling not implemented |
| Smoke test generation (A6) | Not available | Shell template not implemented |
| Tier 2 read-path generation | Not authorized | Requires Tier 1 production proof with ≥2 families |
| Automated file insertion | Not available | Manual marker placement required |
| Batch integration | Not available | Per-family validation required |

### Not Proven

| Capability | Status | Gap |
|------------|--------|-----|
| Cross-layer integration | Unproven | Both integrated families (RSI, EMA) are signal layer |
| Live event flow for generated families | Unproven | No production activation evidence |
| Scale beyond 7 families | Untested | Template and naming may have edge cases |
| Concurrent spec evolution | Untested | Single-author assumption |

---

## 3. Current Integration Map

### Governed by Codegen (4 slices)

| Family | Artifact | Target File | Stage |
|--------|----------|-------------|-------|
| rsi | consumer_spec | internal/adapters/nats/natssignal/registry.go | S200 |
| rsi | pipeline_entry | cmd/writer/pipeline.go | S200 |
| ema | consumer_spec | internal/adapters/nats/natssignal/registry.go | S203 |
| ema | pipeline_entry | cmd/writer/pipeline.go | S203 |

### Golden Snapshots Only (10 slices — not integrated)

| Family | Layer | Golden Exists | Integrated | Reason |
|--------|-------|---------------|------------|--------|
| candle | evidence | Yes (2 files) | No | Awaiting per-family integration stage |
| mean_reversion_entry | strategy | Yes (2 files) | No | Awaiting per-family integration stage |
| paper_order | execution | Yes (2 files) | No | Awaiting per-family integration stage |
| position_exposure | risk | Yes (2 files) | No | Awaiting per-family integration stage |
| rsi_oversold | decision | Yes (2 files) | No | Awaiting per-family integration stage |

These 5 families have validated golden snapshots (all pass `check-all`) but their code is manually maintained in the target files. Integration would replace manual code with governed markers.

---

## 4. Limitations

### Structural Limitations

1. **Two-artifact ceiling**: Only consumer_spec and pipeline_entry are generated. The remaining 4 Tier 1 artifacts (mapper, mapper test, config entry, smoke test) are manual.

2. **Snippet insertion, not file generation**: Codegen produces code snippets that are inserted into existing files using markers. It does not generate complete files.

3. **Manual integration step**: Inserting generated code into target files and placing codegen:begin/end markers is a manual operation. There is no `codegen integrate` command.

4. **Signal-layer bias**: Both integrated families are signal layer. Evidence, decision, strategy, risk, and execution layers have golden snapshots but no integration proof.

5. **No rollback mechanism**: If a generated snippet causes issues, the fix is manual editing within markers (which then must be reconciled with golden snapshots).

### Governance Limitations

1. **Marker discipline**: Governance depends on codegen:begin/end markers being correctly placed. If markers are missing or malformed, the integrated check cannot verify the region.

2. **Manifest completeness**: `integrated.yaml` must be manually updated when new slices are integrated. There is no auto-discovery.

3. **Single-template assumption**: Each artifact type has exactly one template. Layer-specific template variants are not supported.

4. **Comment-blind comparison**: Structural normalization strips comments before comparison. Comment-only changes in golden snapshots are invisible to drift detection.

### Operational Limitations

1. **No codegen status dashboard**: `make codegen-status` exists but is informational only — not integrated into monitoring.

2. **No automated regeneration**: When a template is fixed, all 14 golden snapshots must be manually regenerated and committed.

3. **No dependency tracking**: Codegen does not track which domain types or NATS streams must exist for generated code to compile.

---

## 5. Safe Operating Procedures

### Adding a New Family

```
1. Create codegen/families/<name>.yaml following S193 schema
2. Run: go run codegen validate codegen/families/<name>.yaml
3. Run: go run codegen validate-all (cross-spec uniqueness)
4. Run: go run codegen generate codegen/families/<name>.yaml consumer_spec
5. Run: go run codegen generate codegen/families/<name>.yaml pipeline_entry
6. Save outputs as golden snapshots:
   - codegen/golden-snapshots/<name>/consumer_spec.go.golden
   - codegen/golden-snapshots/<name>/pipeline_entry.go.golden
7. Run: go run codegen check-all (all 16 goldens must pass)
8. Commit spec + golden snapshots
```

### Integrating a Family

```
1. Ensure golden snapshots pass check-all
2. Insert generated code into target file with markers:
   // codegen:begin consumer_spec family=<name>
   <generated code>
   // codegen:end consumer_spec family=<name>
3. Add entry to codegen/integrated.yaml
4. Run: make codegen-integrated (must pass)
5. Commit changes
```

### Modifying a Template

```
1. Edit codegen/templates/<artifact>.go.tmpl
2. Regenerate ALL golden snapshots for affected artifact
3. Run: go run codegen check-all (all must pass)
4. Run: make codegen-integrated (all integrated slices must pass)
5. Commit template + all affected goldens
```
