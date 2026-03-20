# Stage S203 — First Generated Family Implementation and Validation Report

**Status:** Complete
**Date:** 2026-03-20
**Predecessor:** S202 (First Generated Family Definition)
**Successor:** S204 (Post-Generated-Family Gate Review)

---

## Objective

Implement and validate the first codegen-first family (EMA), proving that the generated path can participate in the real analytical pipeline expansion with explicit governance, auditability, and zero infrastructure overhead.

## Executive Summary

The EMA signal family was successfully implemented as the first codegen-first family. Both generated artifacts (consumer spec and pipeline entry) were produced by the codegen engine, matched golden snapshots, compiled into the writer binary, passed all governance checks, and caused zero test regressions across the entire codebase. The generated path is proven to work for same-layer, same-infrastructure families.

## Family Implemented

| Property | Value |
|----------|-------|
| Family | EMA (Exponential Moving Average) |
| Layer | Signal (Tier 1) |
| Spec | `codegen/families/ema.yaml` |
| Generated Artifacts | A1 (consumer spec) + A2 (pipeline entry) |
| Reused Artifacts | `mapSignalRow`, existing mapper tests |
| Manual Artifacts | Config entry, settings registration |
| New Infrastructure | None — full reuse |

## Success Criteria Results

| # | Criterion | Result | Evidence |
|---|-----------|--------|----------|
| SC-1 | Spec validates clean | **PASS** | `codegen validate` + `validate-all` (7 families, cross-spec OK) |
| SC-2 | Golden snapshots match | **PASS** | `codegen check-all`: 14/14 PASS |
| SC-3 | Code compiles | **PASS** | `go build ./cmd/writer/...` + `go build ./internal/adapters/nats/...` clean |
| SC-4 | Existing tests pass | **PASS** | All packages pass (writer, settings, nats, clickhouse, http, application, codegen) |
| SC-5 | Integrated check passes | **PASS** | `codegen-integrated-check.sh`: 4/4 PASS |
| SC-6 | Pipeline activates with config | **PARTIAL** | Writer config enables EMA; structural activation confirmed. Live traffic proof deferred (no EMA signal producer) |
| SC-7 | No manual edits to generated code | **PASS** | Confirmed by SC-5 golden-to-target match |
| SC-8 | Manifest updated | **PASS** | `integrated.yaml`: 4 entries (2 RSI + 2 EMA) |

**Overall:** 7/8 fully satisfied, 1/8 partial (SC-6 — structural only, no live events).

## Files Changed

### New Files (Generated Path)

| File | Purpose |
|------|---------|
| `codegen/families/ema.yaml` | EMA family spec (frozen, 14 fields) |
| `codegen/golden-snapshots/ema/consumer_spec.go.golden` | Golden snapshot for A1 |
| `codegen/golden-snapshots/ema/pipeline_entry.go.golden` | Golden snapshot for A2 |
| `docs/architecture/first-generated-family-implementation-and-validation.md` | Implementation and validation evidence |
| `docs/architecture/first-generated-family-findings-generated-path-frictions-and-limits.md` | Frictions, limits, and recommendations |

### Modified Files

| File | Change |
|------|--------|
| `internal/adapters/nats/signal_registry.go` | Added `WriterEMASignalConsumer()` with codegen markers |
| `cmd/writer/pipeline.go` | Added EMA pipeline entry with codegen markers |
| `codegen/integrated.yaml` | Added 2 EMA manifest entries |
| `internal/shared/settings/schema.go` | Added `"ema"` to `knownSignalFamilies` and `signalDependsOnEvidence` |
| `internal/shared/settings/settings_test.go` | Updated signal family count assertion (2 → 3) |
| `deploy/configs/writer.jsonc` | Added `"ema"` to `signal_families` |

## Frictions Observed

| # | Friction | Severity |
|---|---------|----------|
| F-1 | Manual fragment insertion (copy-paste from codegen output to target file) | Medium |
| F-2 | Config registration not generated (knownSignalFamilies, signalDependsOnEvidence) | Low |
| F-3 | No live activation proof without EMA signal producer | Medium |
| F-4 | Test count assertion fragility (hardcoded count broke) | Low |
| F-5 | CODEGEN_ROOT env var required for `go run` | Low |
| F-6 | Golden snapshot duplication (accepted by design) | Low |

See `docs/architecture/first-generated-family-findings-generated-path-frictions-and-limits.md` for full analysis.

## Risk Assessment (Post-Implementation)

| Risk | S202 Severity | Outcome |
|------|--------------|---------|
| R-1: Fragment insertion error | Medium | **Not triggered** — insertion was clean, markers correct |
| R-2: Cross-family interference | Low | **Not triggered** — zero test regressions |
| R-3: Config omission | Low | **Not triggered** — config entry added, validation passes |
| R-4: Overconfidence extrapolation | Medium | **Mitigated** — limits explicitly documented |
| R-5: NATS durable collision | Low | **Not triggered** — cross-spec validation confirms no collision |

## Generated vs Manual Boundary Audit

| Artifact | Owner | Generated? | Verified |
|----------|-------|------------|----------|
| Consumer spec function (`WriterEMASignalConsumer`) | Machine | Yes | Golden match confirmed |
| Pipeline entry struct | Machine | Yes | Golden match confirmed |
| Row mapper (`mapSignalRow`) | Human | No — reused | Existing tests pass |
| Config entry (`writer.jsonc`) | Human | No — manual | Applied, validation passes |
| Settings registration (`schema.go`) | Human | No — manual | Applied, tests pass |
| Test adjustment (`settings_test.go`) | Human | No — manual | Updated count assertion |

Boundaries are clear: generated code lives inside markers; manual code lives outside.

## What Was Proven

1. The codegen engine produces correct, compilable Go code for a real signal-layer family
2. The governance chain (spec → golden → markers → CI check) is operational
3. Same-layer infrastructure reuse eliminates all new infrastructure cost
4. Naming derivation handles abbreviations (`ema` → `EMA`) correctly
5. Cross-spec validation prevents collisions across 7 families
6. Existing families and tests are completely unaffected by the addition

## What Was NOT Proven

1. Live event flow (no EMA signal producer exists)
2. Cross-layer generation (EMA is signal-only)
3. New-infrastructure families (EMA reuses everything)
4. Mapper generation (reused, not generated)
5. Multi-family batch efficiency (one family only)

## Preparation for S204 (Gate Review)

The following evidence is available for the S204 gate:

- **Codegen-first family exists and compiles:** Yes
- **Governance chain operational:** 4 integrated slices, all passing
- **Zero regressions:** All existing tests pass
- **Boundaries preserved:** Generated vs manual clearly documented
- **Frictions documented:** 6 frictions catalogued with severity and mitigation paths
- **Limits documented:** 10 current limits of the generated path

**Recommended S204 gate questions:**
1. Is structural activation sufficient, or must live event flow be demonstrated?
2. Should a second codegen-first family be authorized to test repeatability?
3. Should automated fragment insertion be prioritized before multi-family expansion?
4. Should config registration be added to the spec schema?
