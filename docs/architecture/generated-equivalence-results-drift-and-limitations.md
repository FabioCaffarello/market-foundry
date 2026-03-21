# Generated Equivalence Results: Drift and Limitations

> S261 — Explicit record of what the generated path can and cannot reproduce,
> measured against real families in the current repository.

## 1. Drift Summary

### 1.1 Structural Drift: Zero

| Artifact | Families | Drift |
|---|---|---|
| consumer_spec | 10/10 | None — byte-identical after normalization |
| pipeline_entry | 10/10 | None — byte-identical after normalization |

No structural drift was detected in any of the 20 codegen-governed artifacts.

### 1.2 Semantic Drift: Zero (within governed boundary)

The spec values are provably consistent with the surrounding manual code:
- Durable naming patterns: 10/10 match convention.
- Table references: 10/10 match spec.
- Column lists: 10/10 match spec.
- Store consumer durables: 10/10 follow mirrored pattern (`writer-` → `store-`).

### 1.3 Cross-Boundary Drift: Not Applicable

The codegen does not generate code outside its markers. Manual code outside markers
was not modified and was verified to be structurally compatible through the
cross-artifact consistency checks.

## 2. What the Codegen Already Reproduces Faithfully

| Artifact | Coverage | Confidence |
|---|---|---|
| Writer consumer specs | 10/10 families | HIGH — template-proven, CI-enforced |
| Writer pipeline entries | 10/10 families | HIGH — template-proven, CI-enforced |
| Durable naming convention | 10/10 families | HIGH — derived from spec, validated |
| INSERT SQL generation | 10/10 families | HIGH — derived from spec columns |
| Consumer spec structure | All fields | HIGH — AckWait, MaxDeliver, EventSpec |
| Pipeline struct wiring | All fields | HIGH — family, names, table, SQL, closures |

## 3. What the Codegen Does NOT Yet Reproduce

### 3.1 Tier-1 Manual Artifacts (high templateability, not yet governed)

These artifacts follow repeatable patterns and are strong candidates for future
codegen templates, but are currently not generated:

| Artifact | Count | Pattern | Templateability | Barrier |
|---|---|---|---|---|
| Store consumer specs | 10 | `NewConsumerSpec(durable, subject, type, stream)` | HIGH | No template written yet |
| Writer starters | 6 | Identical closure per layer | HIGH | Layer-level, not family-level |
| Writer mappers | 6 | `[]any` from event fields | MEDIUM-HIGH | Requires domain type introspection |
| Config `IsXFamilyEnabled` | 6 | Identical `for-range` loop | HIGH | Layer-level, not family-level |
| Config `EnabledXFamilies` | 6 | Identical `copy` pattern | HIGH | Layer-level, not family-level |

**Estimated additional coverage if templated:** ~34 artifacts → total ~54/140 (39%).

### 3.2 Tier-1 Manual Artifacts (low templateability)

These artifacts contain family-specific or domain-specific logic that resists
mechanical generation:

| Artifact | Why Not Templateable |
|---|---|
| Domain event struct fields | Field names/types vary per domain; require type introspection |
| Domain event constructors | Some layers have enrichment logic (severity, scaling) |
| Registry struct fields | Layer-specific (EventSpec vs ControlSpec layouts) |
| Registry `DefaultRegistry()` | Stream configuration, subject patterns vary per layer |
| Registry `LatestSpecByType()` | Switch statement over family names within a layer |
| NATS consumer handler logic | Domain-specific deserialization and dispatch |
| ClickHouse DDL migrations | Column types require human design decisions |
| Smoke test validation logic | Per-family HTTP response structure varies |

### 3.3 Structural Limitations of the Current Model

1. **Family-level only.** The codegen operates at the family granularity. Layer-level
   artifacts (starters, mappers, config methods) exist once per layer, not per family.
   The current spec schema has no mechanism to govern layer-level artifacts.

2. **Two artifact types only.** Only `consumer_spec` and `pipeline_entry` templates
   exist. Extending to store consumers would require a third template; extending to
   mappers would require domain type metadata in the spec.

3. **Column-opaque design.** The spec treats `writer.columns` as a free-form string.
   This is a strength (absorbs schema changes without codegen updates) but also a
   limitation: the codegen cannot validate column types, generate mappers, or
   produce DDL from the spec alone.

4. **No domain type metadata.** The spec knows the event type name
   (`DecisionEvaluatedEvent`) but not its field layout. Generating mappers would
   require adding a `fields` section to the spec schema.

5. **No store consumer spec.** The spec defines only the writer consumer durable.
   Store consumers follow a mirrored pattern (`writer-` → `store-`) but this pattern
   is not codified in the spec.

6. **Marker placement is manual.** The codegen never creates markers — they must be
   placed by hand before the first integration. This is a design choice (D5: "CI
   verifies, not generates") but limits full automation.

## 4. Risk Assessment for Codegen-First Family

| Risk | Severity | Mitigation |
|---|---|---|
| Golden snapshot and production diverge silently | LOW | CI runs `codegen-integrated-check.sh` on every push |
| Spec values drift from reality | LOW | `codegen-equivalence-check.sh` validates cross-artifact consistency |
| New family template produces wrong output | MEDIUM | Must add golden snapshots + integrated check before merging |
| Layer-level artifacts missing for new layer | HIGH | No codegen for starters/mappers/config — must write manually |
| Column list in spec doesn't match DDL | MEDIUM | Column alignment check in equivalence script |

## 5. Recommendations for S262

### 5.1 Safe to Proceed

The generated path is proven equivalent on all 10 current families across all 20
governed artifacts. Zero drift. This is sufficient evidence to open the first
codegen-first family in S262 **for the two artifact types already governed**
(consumer_spec, pipeline_entry).

### 5.2 Required for Codegen-First

For a codegen-first family to be fully operational, the following manual artifacts
must still be written by hand:

1. Family spec YAML in `codegen/families/`.
2. Golden snapshots (generated, then committed).
3. Marker placement in target files.
4. Store consumer spec in registry (manual).
5. Entry in `integrated.yaml` manifest.
6. Starter function — only if entering a NEW layer (existing layers reuse starters).
7. Mapper function — only if entering a NEW layer.
8. Config array field + method — only if entering a NEW layer.

### 5.3 Future Codegen Expansion Candidates (ordered by ROI)

1. **Store consumer spec template** — mirrors writer consumer, HIGH ROI, LOW risk.
2. **Config entry template** — boilerplate `IsXFamilyEnabled`, HIGH ROI if new layers.
3. **Mapper template** — requires spec schema extension (`fields` section), MEDIUM ROI.
4. **DDL template** — requires column type metadata, LOW ROI (migrations are rare).
