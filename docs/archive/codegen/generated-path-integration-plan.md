# Generated Path Integration Plan

> **Stage:** S199
> **Status:** ACTIVE
> **Scope:** Formal definition of how the codegen-generated path enters the monorepo as a governed, auditable process
> **Prerequisite:** S198 conditional pass (A1+A2 authorized, single family, 8 conditions, 5 revocation triggers)

---

## 1. Problem Statement

The codegen engine exists, is validated (12/12 golden comparisons, 0 structural drift), and has conditional authorization to generate A1 (consumer spec) + A2 (pipeline entry) for a single new family. The problem is no longer "how to design codegen" — it is **how to integrate the generated path into the real monorepo workflow** without creating ambiguity between manual and generated code, without blurring ownership, and without bypassing existing quality gates.

This document defines the generated path as a formal process with explicit entry points, ownership rules, validation requirements, and scope limits.

---

## 2. Generated Path Definition

The **generated path** is the sequence of steps by which a new analytical family's mechanical artifacts are produced from a YAML spec rather than hand-authored. It is not a replacement for the manual path — it is a parallel, constrained process that covers a subset of artifacts under strict governance.

### 2.1 Path Stages

```
[1] Author spec     → codegen/families/{family}.yaml
[2] Validate spec   → codegen validate {spec}
[3] Generate A1+A2  → codegen generate {spec} consumer_spec
                     → codegen generate {spec} pipeline_entry
[4] Golden snapshot  → codegen/golden-snapshots/{family}/
[5] Manual insert    → copy fragments into target files
[6] Manual artifacts → mapper, tests, config, smoke, domain, migration
[7] CI validation    → codegen-golden + codegen-lint + unit-tests + smoke
[8] Review + merge   → PR review with generated-path checklist
```

### 2.2 What the Generated Path Covers (This Phase)

| Artifact | ID | Generated? | Target File |
|----------|----|-----------|-------------|
| Consumer spec function | A1 | YES | `internal/adapters/nats/{layer}_registry.go` |
| Pipeline entry struct | A2 | YES | `cmd/writer/pipeline.go` |
| Mapper function | A3 | NO | `cmd/writer/mappers.go` |
| Mapper tests | A4 | NO | `cmd/writer/mappers_test.go` |
| Config entry | A5 | NO | `deploy/configs/writer.jsonc` |
| Smoke test phase | A6 | NO | `scripts/smoke-analytical-e2e.sh` |
| Domain type | — | NEVER | `internal/domain/` |
| Migration DDL | — | NEVER | `deploy/migrations/` |
| Reader adapter | — | NO | `internal/adapters/clickhouse/` |
| HTTP handler/route | — | NO | `internal/interfaces/http/` |

### 2.3 What the Generated Path Does NOT Cover (Permanently)

These artifacts require architectural decisions that cannot be derived from a spec:

- Domain event types and their field shapes
- ClickHouse schema design (column types, codecs, TTL, partition keys)
- NATS stream definitions (subjects, retention, replicas)
- Shared infrastructure (client pools, actor system, health checks)
- CI/CD pipeline structure
- Templates and spec schema themselves
- Gateway composition root
- Operational (non-analytical) endpoints

---

## 3. Integration Points

### 3.1 Spec → Engine

- YAML spec is authored by a human in `codegen/families/{family}.yaml`
- Spec is the single source of truth for all derived values
- Spec validation runs locally (`codegen validate`) and in CI (`codegen-lint`)
- Spec files are committed and reviewed like any other source file

### 3.2 Engine → Fragments

- Engine renders A1 and A2 as standalone Go code fragments
- Fragments are written to stdout; the developer copies them into target files
- Fragments have no runtime dependency on the codegen module
- Fragments must compile without modification — any manual edit is a revocation trigger (S198 condition)

### 3.3 Fragments → Target Files

- **Consumer spec (A1)**: inserted into the appropriate `internal/adapters/nats/{layer}_registry.go` file as a new exported function
- **Pipeline entry (A2)**: inserted into `cmd/writer/pipeline.go` within the `declareWriterPipelines()` return slice
- Insertion is manual in this phase (marker-based automation deferred)
- The developer is responsible for correct placement

### 3.4 Golden Snapshots → CI

- For every generated family, both A1 and A2 golden snapshots must be committed in `codegen/golden-snapshots/{family}/`
- CI runs `make codegen-check` which validates all families (existing + new) against golden snapshots
- A golden comparison failure blocks merge

### 3.5 Generated Files → PR Review

Every PR that adds a codegen-first family must include:

1. The new YAML spec file
2. Both golden snapshot files (consumer_spec.go.golden, pipeline_entry.go.golden)
3. The generated fragments inserted in their target files
4. All manual artifacts (A3–A6, domain type, migration, etc.)
5. CI passing: codegen-golden + codegen-lint + unit-tests + smoke-analytical

---

## 4. Governance Rules

### 4.1 Spec Authorship

- Only humans author YAML specs
- Specs are reviewed in PRs like any architecture decision
- Spec changes trigger regeneration of affected golden snapshots
- No spec may reference infrastructure that does not yet exist (S198 condition 4)

### 4.2 Template Freeze

- Templates are frozen for this phase (S198 condition 8)
- Any template modification requires a new authorization stage
- Template authorship is a human-only activity

### 4.3 Generated Output Immutability

- Generated fragments must not be manually edited after insertion
- If a generated fragment does not compile or is incorrect, the fix must be in the spec or template — not in the output
- Manual edits to generated output constitute a revocation trigger

### 4.4 Single-Family Iteration

- Only one new family may be generated per authorization cycle
- After the first generated family validates (S197 success criteria), a new gate review determines whether to continue
- Batch generation is explicitly not authorized

### 4.5 Layer Constraint

- The generated family must target an existing layer (evidence, signal, decision, strategy, risk, or execution)
- New layers require infrastructure that is outside codegen scope

---

## 5. Process Flow for Adding a Codegen-First Family

### Step 1: Spec Authorship
```
codegen/families/{new_family}.yaml
```
Author must supply all 14 fields. The `writer.mapper` field must reference a named function (not `"generate"`). The `writer.table` must reference an existing ClickHouse table (or a new migration must be included in the same PR).

### Step 2: Local Validation
```bash
cd codegen && go run . validate families/{new_family}.yaml
```
Must output `VALID`.

### Step 3: Generate Fragments
```bash
cd codegen && go run . generate families/{new_family}.yaml consumer_spec
cd codegen && go run . generate families/{new_family}.yaml pipeline_entry
```
Capture stdout for insertion.

### Step 4: Create Golden Snapshots
```bash
mkdir -p codegen/golden-snapshots/{new_family}/
# Save generated output as golden files
cd codegen && go run . generate families/{new_family}.yaml consumer_spec > golden-snapshots/{new_family}/consumer_spec.go.golden
cd codegen && go run . generate families/{new_family}.yaml pipeline_entry > golden-snapshots/{new_family}/pipeline_entry.go.golden
```

### Step 5: Verify Golden Match
```bash
cd codegen && go run . check-all
```
Must report all families PASS (existing + new).

### Step 6: Insert Fragments
- Copy A1 into the appropriate NATS registry file
- Copy A2 into `cmd/writer/pipeline.go`

### Step 7: Author Manual Artifacts
- Mapper function (A3) in `cmd/writer/mappers.go`
- Mapper tests (A4) in `cmd/writer/mappers_test.go`
- Config entry (A5) in `deploy/configs/writer.jsonc`
- Smoke test phase (A6) in `scripts/smoke-analytical-e2e.sh`
- Domain type (if new) in `internal/domain/`
- Migration DDL (if new table) in `deploy/migrations/`
- Reader adapter, use case, handler, route (if read-path needed)

### Step 8: CI Validation
```bash
make codegen-check   # golden equivalence
make codegen-test    # engine unit tests
make test            # all unit tests
make smoke-analytical # end-to-end
```

### Step 9: PR Review
- Reviewer verifies spec correctness
- Reviewer verifies generated fragments match golden snapshots
- Reviewer verifies no manual edits to generated output
- Reviewer verifies manual artifacts are complete
- CI must be green

---

## 6. Scope Limits (This Phase)

| Dimension | Limit |
|-----------|-------|
| Artifacts generated | A1 + A2 only |
| Families per iteration | 1 |
| Tiers | Tier 1 only |
| Layers | Existing 6 only |
| File integration | Manual copy (no markers) |
| Template evolution | Frozen |
| Spec schema evolution | Frozen (14 fields) |
| Mapper generation | Not authorized |
| Read-path generation | Not authorized (Tier 2) |
| Cross-service generation | Not in scope |

### 6.1 What Unlocks the Next Phase

The following must be true before expanding the generated path:

1. First codegen-first family passes all 7 S197 success criteria
2. No revocation triggers activated during first family
3. Gate review (S200+) explicitly authorizes expansion
4. If mapper generation desired: `domain.columns` spec extension must be designed and frozen in a dedicated stage

---

## 7. Relationship to Existing Manual Families

The 6 existing families (candle, rsi, rsi_oversold, mean_reversion_entry, position_exposure, paper_order) remain **manually authored and manually maintained**. They are not retroactively converted to codegen-first.

Their role in the generated path:
- They serve as the **golden reference baseline** for template equivalence
- Their YAML specs in `codegen/families/` exist for validation purposes, not for regeneration
- Changes to existing families follow the manual path exclusively
- The codegen engine must produce output structurally equivalent to these families — that is the correctness proof

### 7.1 Future Retroactive Conversion

Retroactive conversion of manual families to codegen-managed is explicitly **not in scope** for this phase. If considered in the future, it would require:
- A dedicated authorization stage
- Proof that regenerated output matches current hand-crafted code
- A migration plan for any manual customizations
- CI validation of the conversion

---

## 8. Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Generated output diverges from expected | Golden snapshot comparison in CI blocks merge |
| Fragment inserted incorrectly | Compilation + unit tests + smoke catch wiring errors |
| Spec references non-existent infrastructure | Spec validation + S198 condition 4 |
| Boundary between manual and generated blurs | Ownership table (Section 2.2) + PR review checklist |
| Template modification sneaks in | S198 condition 8 + PR review |
| Scope creep beyond A1+A2 | S198 condition 1 + single-family gate |
| Over-reliance on codegen correctness | Manual artifacts (A3–A6) still require human judgment |

---

## 9. Success Criteria for S199

| # | Criterion | Evidence |
|---|-----------|----------|
| SC1 | Generated path formally defined as a governed process | This document |
| SC2 | Ownership of spec/templates/outputs is unambiguous | Section 2.2 + companion doc |
| SC3 | CI integration and drift policy are explicit | Companion CI/drift policy doc |
| SC4 | Manual vs generated coexistence is safe | Section 7 + governance rules |
| SC5 | Foundation ready for S200 (first codegen-first family) | Process flow (Section 5) + scope limits (Section 6) |
