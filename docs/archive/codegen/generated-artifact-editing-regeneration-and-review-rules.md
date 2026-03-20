# Generated Artifact Editing, Regeneration, and Review Rules

> S201 — Operational rules for how generated artifacts are created, maintained, and reviewed.

## Golden Rule

**The spec is the source of truth. Generated output is a derived artifact. If you need to change generated code, change the spec or template — never the output directly.**

## Editing Rules by Zone

### Zone 1: Human-Owned Files

Standard development workflow. No codegen constraints apply.

### Zone 2: Machine-Owned (Golden Snapshots)

| Action | Allowed? | Procedure |
|--------|----------|-----------|
| Read | Yes | Freely readable for debugging |
| Manual edit | **Never** | Regenerate from spec instead |
| Delete | Only if family is being retired | Requires removing spec + manifest entry |
| Regenerate | Yes | `cd codegen && go run . generate families/{family}.yaml {artifact} > golden-snapshots/{family}/{artifact}.go.golden` |

### Zone 3: Machine Regions in Mixed Files

Code between `codegen:begin` and `codegen:end` markers:

| Action | Allowed? | Procedure |
|--------|----------|-----------|
| Read | Yes | Freely readable |
| Manual edit | **Never** | CI (`codegen-integrated`) will catch and fail |
| Replace with regenerated output | Yes | Regenerate, copy between markers, verify with `make codegen-integrated` |
| Add comments inside markers | **No** | Comments are stripped during comparison but create visual noise; add them outside markers |

Code **outside** markers in the same file: standard development rules apply.

## Regeneration Procedures

### Single Artifact Regeneration

When a spec or template changes:

```bash
# 1. Regenerate golden snapshot
cd codegen && go run . generate families/{family}.yaml {artifact} \
    > golden-snapshots/{family}/{artifact}.go.golden

# 2. Verify spec→golden equivalence
cd codegen && go run . check-all

# 3. Copy golden content into target file between markers
#    (manual step — no automated file patching yet)

# 4. Verify golden→target match
make codegen-integrated
```

### Full Regeneration (All Families)

When a template changes (requires authorization):

```bash
# 1. Regenerate all goldens
for spec in codegen/families/*.yaml; do
    family=$(basename "$spec" .yaml)
    for artifact in consumer_spec pipeline_entry; do
        cd codegen && go run . generate "families/${family}.yaml" "$artifact" \
            > "golden-snapshots/${family}/${artifact}.go.golden"
    done
done

# 2. Verify all
make codegen-check

# 3. Update all governed target files (only those in integrated.yaml)
# Manual copy-paste for each governed slice

# 4. Verify integrated
make codegen-integrated
```

### Cross-Spec Validation After Adding a Family

```bash
# After creating a new spec file:
cd codegen && go run . validate families/new_family.yaml
make codegen-validate-all    # checks uniqueness across all specs
```

## PR Review Checklist for Generated Artifacts

When a PR touches codegen-related files, reviewers must verify:

### If spec changed:
- [ ] Golden snapshots regenerated (not manually edited)
- [ ] `make codegen-check` passes (spec→golden equivalence)
- [ ] `make codegen-validate-all` passes (no cross-spec collisions)
- [ ] Target files updated (for governed slices only)
- [ ] `make codegen-integrated` passes (golden→target match)

### If new family added:
- [ ] YAML spec is complete (all 14 fields)
- [ ] Golden snapshots generated for both artifacts
- [ ] `make codegen-validate-all` passes (uniqueness)
- [ ] If governed: markers placed, manifest entry added
- [ ] Manual artifacts authored (mapper, tests, config, smoke)

### If template changed (requires stage authorization):
- [ ] Authorization stage reference documented
- [ ] All goldens regenerated
- [ ] All governed targets updated
- [ ] Full CI pipeline passes

### Red flags to reject:
- Golden snapshot with manual edits (diff shows non-structural changes)
- Missing `codegen:end` marker (orphaned begin marker)
- Marker text doesn't match manifest entry
- Target file edited inside markers without corresponding spec change

## Revocation Policy

Per S198, any of these conditions triggers governance revocation for the affected family:

1. Manual edit to code inside `codegen:begin/end` markers
2. Golden snapshot manually edited instead of regenerated
3. Marker removed or malformed without stage authorization
4. Spec deleted while governed slices still reference it

Revocation means: the family falls back to fully manual until a remediation stage re-establishes governance.

## Contributor Quick Reference

```
I want to...                          → Do this
─────────────────────────────────────────────────────────
Fix a bug in generated code           → Fix the spec or template, regenerate
Add a new codegen-first family        → Author spec, generate goldens, add markers + manifest
Understand what's governed            → make codegen-status
Check if my changes broke codegen     → make codegen-check && make codegen-integrated
Validate spec uniqueness              → make codegen-validate-all
See the full validation chain         → make codegen-validate-all && make codegen-check && make codegen-test && make codegen-integrated
```
