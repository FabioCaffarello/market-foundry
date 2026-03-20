# Codegen Readiness: Gains, Tradeoffs, and Open Debts

## Gains

### G1: Proven structural equivalence across full family coverage

The codegen engine produces A1 (consumer spec) and A2 (pipeline entry) artifacts that are structurally identical to hand-crafted code across all 6 existing families, covering all 6 analytical layers. This is not a two-family sample — it is exhaustive validation against every family that exists.

**Evidence**: 12/12 golden comparisons pass (S196). Zero structural drift. Three cosmetic drift instances (comment phrasing) classified INFO and normalized away.

### G2: Deterministic naming convention enforcement

The engine eliminates naming convention errors in consumer specs and pipeline entries. Derived fields (PascalCase family/layer names, hyphenated component names, evidence-layer exceptions, known abbreviation handling) are computed mechanically from spec. Manual authoring of these fields was error-prone by nature.

**Evidence**: `TestToPascalCase` covers 12 naming edge cases. `TestDerivedFields` validates RSI, Paper Order, and Evidence layer derivation. All 6 families' naming validated in golden comparisons.

### G3: CI gate prevents regression

The `codegen-golden` CI job runs `codegen check-all` (12 golden comparisons) and `codegen-test` (26 unit tests) on every push/PR. Any structural drift blocks merge. The gate is auto-extensible: adding a new family spec + golden snapshots automatically includes it in validation.

**Evidence**: CI job defined in `.github/workflows/ci.yml`. Execution time ~3 seconds.

### G4: Specification-driven expansion model

New families are defined by authoring a 14-field YAML spec rather than copying and modifying existing code. This reduces the surface area for copy-paste errors and makes the family definition auditable independently of implementation.

**Evidence**: 6 family specs exist, all validated by `codegen validate`. Schema frozen in S193 with explicit prohibited fields and validation invariants.

### G5: Modest but real time savings

Estimated ~15 minutes saved per family (~23% reduction from ~65 min manual baseline). The savings come from eliminating A1+A2 manual authoring and reducing naming convention debugging.

**Limit**: Savings are modest because 4 of 6 Tier 1 artifacts remain manual. The primary value is correctness, not speed.

## Tradeoffs

### T1: Fragment generation, not file generation

The engine produces code fragments, not complete files. A1 (consumer spec function) and A2 (pipeline entry struct) must be manually inserted into their target files. This means the codegen does not fully automate even the two artifacts it covers.

**Accepted because**: File integration with marker sections is an engineering effort that does not improve correctness. Manual insertion into well-known locations is low-risk and takes ~5 minutes. Automating it now would add complexity without proportional value.

**Debt**: Marker-section file integration remains deferred. If family count exceeds ~10, manual insertion friction may justify the investment.

### T2: Two of six Tier 1 artifacts covered

The engine generates A1 and A2. The remaining four Tier 1 artifacts (mapper, mapper tests, config entry, smoke phase) require manual authoring. A new family is not fully generated — it is spec-driven for 2 artifacts and manual for 4.

**Accepted because**: A1+A2 are the highest-repetition, most-mechanical artifacts. A3 (mapper) requires `domain.columns` spec extension and DDL awareness — a separate engineering effort. A5 (config entry) requires JSONC tooling. A6 (smoke phase) requires shell template support. Each has its own cost-benefit analysis.

**Debt**: Mapper generation (A3) is the highest-value next artifact. It requires `domain.columns` spec extension, column-order DDL validation, and its own equivalence validation stage.

### T3: No automated scope guard

The boundary between generated and manual artifacts is enforced by documentation and review discipline, not by automated tooling. Nothing prevents someone from expanding codegen scope without a formal gate, except awareness of the rules.

**Accepted because**: With 6 families and a single-digit team, process discipline is sufficient. Automated guards add complexity.

**Debt**: If codegen scope is ever expanded without a gate, this becomes a pattern violation. Consider adding a scope-lock mechanism if team size or family count grows significantly.

### T4: Golden snapshot maintenance overhead

Each new family requires golden snapshots for A1 and A2. These must be created, reviewed, and maintained. If templates change, all golden snapshots must be regenerated and reviewed.

**Accepted because**: The overhead is proportional to family count (2 files per family). At 7 families, this is 14 golden files — manageable. The CI gate automates the comparison.

**Debt**: At ~15+ families, golden snapshot maintenance may warrant tooling (e.g., `codegen regenerate-goldens`).

### T5: Normalization pipeline may be too strict or too loose

The structural comparison normalizes via gofmt + import sort + comment strip. The normalization rules were calibrated against 6 families. New families with unusual patterns (e.g., nested structs, conditional fields) may reveal gaps in normalization.

**Accepted because**: The 6-family coverage is comprehensive (all layers, all naming patterns, all complexity levels). Risk of normalization gaps in a 7th family that targets an existing layer is low.

## Open Debts

### D1: Mapper generation (A3) — HIGH priority, DEFERRED

**What**: The mapper function is the most complex Tier 1 artifact. It maps domain event fields to ClickHouse columns with optional transform functions. Generating it requires:
- `domain.columns` spec extension (column name, Go field, CH type, optional transform)
- Column-order awareness (ClickHouse uses positional binding)
- DDL cross-validation (column order in mapper must match INSERT SQL column order)

**Why deferred**: Engineering effort is significant. Requires spec schema extension (currently frozen). Requires DDL parsing or referencing. Each of these is a separate validation gate.

**Trigger**: After first generated family validates Tier 1 A1+A2 in production.

### D2: File integration with marker sections — MEDIUM priority, DEFERRED

**What**: Generated fragments are currently copy-pasted into target files. Marker sections (e.g., `// codegen:begin:consumer_spec` / `// codegen:end:consumer_spec`) would allow the engine to insert fragments into the correct location automatically.

**Why deferred**: Manual integration takes ~5 minutes per family and is error-resistant (well-known target locations). Automation adds file-manipulation complexity without improving correctness.

**Trigger**: When manual integration friction exceeds ~10 minutes per family or error rate exceeds zero.

### D3: CI drift detection job — MEDIUM priority, DEFERRED

**What**: A `codegen-drift` CI job that regenerates all artifacts from specs and diffs against committed generated code. Currently, CI validates golden snapshots but does not detect drift between generated fragments and the actual source files they are integrated into.

**Why deferred**: With 2 artifacts and manual integration, drift between golden snapshots and source files is detectable by review. The golden comparison is the primary gate.

**Trigger**: When file integration is automated (D2), drift detection becomes critical.

### D4: Cross-spec uniqueness validation in CI — LOW priority, DEFERRED

**What**: Specs define unique values (family name, durable consumer, pipeline key). Currently uniqueness is enforced by test coverage. CI does not validate uniqueness across all specs at load time.

**Why deferred**: With 6 specs, collisions are visible by inspection. The `TestCheckAllFamilies` test implicitly validates that all specs load without error.

**Trigger**: When spec count exceeds ~10 or when spec authoring is delegated beyond the core team.

### D5: Config entry generation (A5) — LOW priority, DEFERRED

**What**: Writer config entries in `deploy/configs/writer.jsonc` are currently manual (~2 min per family). JSONC manipulation tooling is not implemented.

**Why deferred**: Effort to build JSONC tooling exceeds the ~2 min manual cost per family for the foreseeable future.

**Trigger**: Not expected to trigger unless family count exceeds ~20.

### D6: Smoke test phase generation (A6) — LOW priority, DEFERRED

**What**: Smoke test phases in `scripts/smoke-analytical-e2e.sh` are currently manual (~5 min per family). Shell script templating is not implemented.

**Why deferred**: Shell template engines add complexity. Manual smoke phases are simple and reviewed.

**Trigger**: Not expected to trigger unless smoke test structure changes significantly.

### D7: Tier 2 authorization — NOT SCHEDULED

**What**: Tier 2 covers 17 read-path artifacts (reader adapters, HTTP handlers, routes, application layer). Not authorized and not scheduled.

**Why deferred**: Tier 1 must be validated in production first. Tier 2 complexity is significantly higher (cross-service artifacts, API contracts, query logic).

**Trigger**: After Tier 1 is proven with ≥2 generated families and operational stability confirmed.

## Items That Do Not Justify Their Cost Now

| Item | Cost | Benefit | Verdict |
|------|------|---------|---------|
| Generic codegen framework (GenericFamily[T]) | High (runtime coupling) | Type safety | REJECT — anti-pattern per S193 |
| Config explosion (200+ field specs) | Medium (spec bloat) | Flexibility | REJECT — specs contain only what varies |
| Template monolith (single template, all artifacts) | Medium (coupling) | Simplicity | REJECT — one template per artifact type |
| Automated golden regeneration tooling | Low-medium | Convenience | DEFER — not needed at 7 families |
| Spec linting beyond schema validation | Medium | Catch subtle errors | DEFER — 6 families insufficient to define lint rules |
| Multi-family parallel generation | Medium | Speed | REJECT — single-family iteration is the discipline |
