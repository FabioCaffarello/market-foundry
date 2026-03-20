# First Generated Family — Scope Decision

**Stage**: S197
**Status**: DECISION ISSUED
**Date**: 2026-03-20
**Depends on**: S193 (spec freeze), S194 (equivalence baseline), S195 (minimal engine), S196 (cross-family validation)

---

## 1. Executive Summary

Based on the evidence accumulated across S193–S196, the first generated family is **authorized** under constrained conditions. The codegen engine has demonstrated 100% structural equivalence for Artifacts A1 (consumer spec) and A2 (pipeline entry) across all 6 existing hand-crafted families. These two artifacts — and only these two — may be generated for the first new family. All remaining artifacts (A3–A6) must be written manually.

This decision follows Option B from S196: prove production value with the current slice before expanding codegen coverage.

---

## 2. Decision

### 2.1 — Is There Sufficient Basis for a First Generated Family?

**YES.**

Evidence:
- **12/12 golden comparisons PASS** (6 families × 2 artifacts) — S196
- **0 structural drift** detected — S196
- **3 cosmetic drift instances**, all INFO severity, all handled by normalization — S196
- **26 unit tests PASS** covering spec parsing, derived fields, template rendering, and golden comparison — S196
- **CI gate operational**: `codegen-golden` job blocks merge on any structural mismatch — S196
- **Full complexity spectrum exercised**: evidence layer exceptions, known abbreviations (RSI, RSIOversold), multi-word families (mean_reversion_entry, position_exposure), minimal to ceiling complexity (candle → paper_order) — S196

The engine is not speculative. It has been validated against every existing family and produces structurally identical output. The risk of the first generated family is bounded by the same equivalence constraints that protect the existing families.

### 2.2 — What Can Be Generated?

| Artifact | Generated? | Rationale |
|----------|:---:|---|
| A1: Consumer spec function | **Yes** | 12/12 equivalence, zero creative decisions, fully spec-derivable |
| A2: Pipeline entry struct | **Yes** | 12/12 equivalence, zero creative decisions, fully spec-derivable |
| A3: Mapper function | **No** | Requires `domain.columns` spec extension + column-order DDL awareness |
| A4: Mapper unit tests | **No** | Depends on A3; not implementable without mapper generation |
| A5: Config entry | **No** | JSONC manipulation tooling not implemented |
| A6: Smoke test phase | **No** | Shell script template engine not implemented |

### 2.3 — What Remains Manual?

For the first generated family, the following must be hand-crafted:

1. **Domain event type** — Go struct in the appropriate domain package
2. **NATS stream and subject definitions** — if the family uses a new stream
3. **ClickHouse migration** — DDL for the target table (if new table required)
4. **Mapper function** — event-to-row transformation in `cmd/writer/mappers.go`
5. **Mapper unit tests** — in `cmd/writer/mappers_test.go`
6. **Config entry** — in `deploy/configs/writer.jsonc`
7. **Smoke test phase** — in `scripts/smoke-analytical-e2e.sh`
8. **Reader adapter** — ClickHouse read query (if read path needed)
9. **HTTP handler + route** — gateway query surface (if read path needed)
10. **Integration into source files** — A1 and A2 produce code fragments; manual insertion into `internal/adapters/nats/{domain}_registry.go` and `cmd/writer/pipeline.go` is required

### 2.4 — Scope Limits for the First Iteration

1. **Single family only** — no batch generation, no multi-family expansion
2. **Tier 1 only** — Tier 2 (read-path) generation remains unauthorized
3. **Existing layer only** — the first generated family must target a layer that already has infrastructure (table, stream, registry adapter)
4. **Existing table only** — the family must write to a ClickHouse table that already exists via a committed migration; no new DDL
5. **Named mapper only** — the family must use `mapper: "{function_name}"` (not `mapper: "generate"`); the mapper function is hand-crafted
6. **Fragment integration is manual** — generated A1 and A2 code is inserted into existing files by hand, following the same patterns as the 6 baseline families
7. **Golden snapshots required before merge** — the new family must have golden snapshots committed and passing in CI before any source integration

---

## 3. First Family Candidate Selection Criteria

The first generated family must satisfy all of the following:

| Criterion | Rationale |
|---|---|
| **Targets an existing layer** | Infrastructure (stream, registry, table) must already exist |
| **Targets an existing table** | No new DDL; writer inserts into a known schema |
| **Uses a named mapper** | Mapper generation (A3) is not authorized |
| **Is Tier 1** | Tier 2 (read-path) is out of scope |
| **Has clear domain event type** | Event struct must already exist or be trivially definable |
| **Complexity is within validated range** | Family must fall between candle (minimal) and paper_order (ceiling) in the S194 bracket |

**Note**: The specific family to be generated is NOT selected in this stage. S197 authorizes generation; the family selection and implementation belong to a subsequent stage (S198+).

---

## 4. Generation Workflow for First Family

Once a candidate is selected, the workflow is:

1. **Author YAML spec** — `codegen/families/{family}.yaml` following the S193 frozen schema
2. **Validate spec** — `codegen validate families/{family}.yaml`
3. **Generate A1 + A2** — `codegen generate families/{family}.yaml consumer_spec` and `pipeline_entry`
4. **Create golden snapshots** — save generated output to `codegen/golden-snapshots/{family}/`
5. **Run CI validation** — `make codegen-check` must pass with new family included
6. **Hand-craft A3–A6** — mapper, mapper tests, config entry, smoke phase
7. **Hand-craft domain artifacts** — event type, migration (if needed)
8. **Integrate A1 + A2 into source** — manually insert generated fragments into registry and pipeline files
9. **Run full test suite** — unit tests, golden tests, smoke tests
10. **Submit PR** — CI gates: unit-tests, codegen-golden, smoke-analytical

---

## 5. What This Decision Does NOT Authorize

- Expanding codegen to A3–A6 artifacts
- Generating multiple families simultaneously
- Tier 2 (read-path) generation
- New table DDL generation
- Automatic file integration (marker sections)
- Treating the first generated family as proof that all families should be generated
- Skipping manual review of generated output before integration

---

## 6. Preparation for S198

S198 should:
1. Select the specific first generated family candidate
2. Validate the candidate against the criteria in Section 3
3. Execute the workflow in Section 4
4. Measure: time saved vs baseline manual workflow (~45min → ~15min per S196 estimate)
5. Capture any friction or drift not anticipated by S196
6. Decide whether the generation model is ready for a second family or needs hardening first
