# Analytical / Generated Path Consolidation

> **Stage:** S214
> **Status:** Consolidated
> **Scope:** Ownership clarity, boundary hardening, noise reduction
> **Non-goal:** No functional expansion, no new families, no codegen amplification

---

## 1. Problem Statement

The analytical path (writer → ClickHouse → reader → gateway) and the generated path (codegen spec → template → golden → governed fragment) coexist in the same codebase. Both touch overlapping files — particularly `cmd/writer/pipeline.go` and `internal/adapters/nats/<domain>/registry.go`. Prior to this consolidation:

- It was not visually obvious which pipeline entries were codegen-governed and which were manually maintained.
- Registry files lacked explicit ownership markers for their writer consumer spec sections.
- Multiple architecture documents described overlapping boundary rules with inconsistent terminology and occasionally contradictory details (e.g., marker format variations between S201 and S202).
- The relationship between "has a codegen spec + golden snapshot" and "is actually governed by codegen in the codebase" was ambiguous — 5 families have specs/goldens but remain manually coded.

## 2. Consolidation Applied

### 2.1 Code-Level Ownership Markers

Added explicit `manual:owned` annotations to all human-maintained writer consumer spec sections and pipeline entries. This creates a clear visual contract:

- `codegen:begin / codegen:end` → machine-governed, CI-validated against golden snapshots
- `manual:owned` → human-maintained, not subject to codegen drift detection

**Files annotated:**
- `cmd/writer/pipeline.go` — evidence section and decision/strategy/risk/execution section
- `internal/adapters/nats/natsevidence/registry.go` — writer consumer specs
- `internal/adapters/nats/natsdecision/registry.go` — writer consumer specs
- `internal/adapters/nats/natsstrategy/registry.go` — writer consumer specs
- `internal/adapters/nats/natsrisk/registry.go` — writer consumer specs
- `internal/adapters/nats/natsexecution/registry.go` — writer consumer specs
- `internal/adapters/nats/natssignal/registry.go` — clarified that store specs remain manual:owned

### 2.2 Canonical Marker Standard

**Canonical format (S201, active):**
```go
// codegen:begin <artifact_type> family=<family_name> source=<spec_path>
... governed code ...
// codegen:end <artifact_type> family=<family_name>
```

**Deprecated format (S202, superseded):**
```
// --- BEGIN CODEGEN MANAGED SECTION ---
```

The S201 format is the only format recognized by `codegen/integrated.yaml` and `scripts/codegen-integrated-check.sh`. References to the S202 format in earlier docs are historical artifacts.

### 2.3 Ownership Consolidation

The definitive ownership model is documented in [analytical-vs-generated-ownership-and-boundaries.md](analytical-vs-generated-ownership-and-boundaries.md).

The artifact classification model is documented in [manual-generated-derived-operational-artifact-model.md](manual-generated-derived-operational-artifact-model.md).

## 3. Current Integration State

### 3.1 Codegen-Governed Slices (4 total)

| Family | Artifact | Target File | Stage |
|--------|----------|-------------|-------|
| rsi | consumer_spec | `internal/adapters/nats/natssignal/registry.go` | S200 |
| rsi | pipeline_entry | `cmd/writer/pipeline.go` | S200 |
| ema | consumer_spec | `internal/adapters/nats/natssignal/registry.go` | S203 |
| ema | pipeline_entry | `cmd/writer/pipeline.go` | S203 |

### 3.2 Families with Specs + Goldens but No Governance (5 total)

| Family | Layer | Status |
|--------|-------|--------|
| candle | evidence | Manual. Evidence naming conventions require architectural decisions. |
| rsi_oversold | decision | Manual. Spec exists but integration not authorized. |
| mean_reversion_entry | strategy | Manual. Spec exists but integration not authorized. |
| position_exposure | risk | Manual. Spec exists but integration not authorized. |
| paper_order | execution | Manual. Spec exists but integration not authorized. |

These families have `codegen/families/*.yaml` specs and `codegen/golden-snapshots/` outputs that pass `codegen check-all`. Their golden snapshots serve as **reference artifacts** proving the codegen engine can produce structurally equivalent code. They are NOT regenerated — they validate template correctness.

### 3.3 Families Without Codegen Specs (operational-only)

| Family | Layer | Notes |
|--------|-------|-------|
| tradeburst | evidence | Store-only, no writer pipeline |
| volume | evidence | Store-only, no writer pipeline |
| ema_crossover | signal | Store-only, no writer pipeline |
| venue_market_order | execution | Separate owner (execute binary), different stream |

These families exist in the operational path (NATS registries, store pipelines, config) but have no codegen representation and no analytical path coverage.

## 4. Documentation Noise Reduced

### 4.1 Superseded Concepts

| Concept | Source | Status |
|---------|--------|--------|
| `--- BEGIN/END CODEGEN MANAGED SECTION ---` markers | S202 | Superseded by S201 `codegen:begin/end` format |
| "File integration is manual (copy-paste)" | S197, S199 | Superseded by S201 marker-based integration + manifest |
| "Golden-to-target drift detection deferred" | S199 | Implemented in S201 via `codegen-integrated-check.sh` |

### 4.2 Confirmed Deferrals

| Item | Status | Blocker |
|------|--------|---------|
| Mapper generation (A3) | Not authorized | Requires `domain.columns` spec extension + equivalence proof |
| Tier 2 read-path generation | Not authorized | Requires dedicated stage |
| Store pipeline codegen | Not authorized | Different actor pattern, projection closures |
| File integration automation (auto-inject) | Deferred | Manual marker placement remains required |

## 5. Boundary Invariants

1. **Codegen never creates markers.** Markers are placed manually during initial integration. Codegen only owns content between existing markers.
2. **Codegen never touches files without markers.** The manifest (`integrated.yaml`) is the sole authority for what is governed.
3. **Golden snapshots for non-integrated families are reference artifacts**, not deployment targets. They prove template correctness, nothing more.
4. **The analytical path (ClickHouse) is entirely separate from the generated path (codegen).** They share the writer service as an execution boundary but have independent ownership models.
5. **Store path is entirely manual.** No codegen markers, no governance. The store actor pattern (projection + consumer pairs) is structurally different from the writer pattern.

## 6. Known Pre-Existing Drift

The `codegen-integrated-check.sh` reports drift for `rsi/consumer_spec` and `ema/consumer_spec`. This is a **pre-existing condition** caused by the pragmatic adaptation in S200/S203: the golden snapshots contain the expanded `ConsumerSpec{}` struct literal (as produced by the template), but the actual target code uses the `newConsumerSpec()` factory function (introduced to reduce duplication).

Both forms are **functionally equivalent** — they produce identical `ConsumerSpec` values at runtime. The drift exists because the structural comparison cannot recognize this equivalence.

**Resolution options (for a future stage, not S214):**
1. Update golden snapshots to use `newConsumerSpec()` factory calls (requires template change)
2. Update the comparison normalization to recognize factory-based equivalence
3. Accept the drift as a documented exception and annotate the manifest

This drift does NOT affect `pipeline_entry` artifacts (both golden and target match structurally).

---

## 7. Cross-Reference

- Ownership model: [analytical-vs-generated-ownership-and-boundaries.md](analytical-vs-generated-ownership-and-boundaries.md)
- Artifact classification: [manual-generated-derived-operational-artifact-model.md](manual-generated-derived-operational-artifact-model.md)
- Integration manifest: `codegen/integrated.yaml`
- CI validation: `scripts/codegen-integrated-check.sh`
- Codegen engine: `codegen/` (spec.go, render.go, compare.go, main.go)
