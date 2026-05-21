# Stage S104 — Config Activation and Dependency Map Hardening

**Status:** Complete
**Date:** 2026-03-19

## 1. Executive Summary

S104 hardens the configuration activation, validation, and dependency map mechanisms in Market Foundry, closing the "config validation sync" debt identified in S100. The focus is on reducing fragility in the bridge between static pipeline config, dynamic configctl bindings, and runtime family registration — without introducing excessive automation.

**Key outcomes:**
- Added duplicate family detection in pipeline config validation, preventing silent misconfiguration.
- Added binding topic format validation in configctl, bridging the gap between config documents and the `source.symbol` convention expected by ingest/derive.
- Added artifact metadata convention enforcement (schema version and runtime loader whitelists), preventing incompatible artifacts from being activated.
- Exported the canonical family catalog and dependency graph as queryable APIs for tooling, tests, and coherence checks.
- Updated all test fixtures from legacy `"validator:v1"` to the canonical `"configctl-sync/v1"` runtime loader.
- Documented the config activation model, dependency map, validation layers, and sync rules.

## 2. Gap Analysis (Pre-S104)

### What S100 identified:
> "config validation sync" listed as open debt.

### Concrete fragilities found:

| Category | Impact |
|----------|--------|
| Pipeline config accepted duplicate family entries (e.g., `["candle", "candle"]`) | Silent misconfiguration; processors/pipelines spawned correctly but config misleading |
| Configctl binding topics not format-validated | Malformed topics (uppercase, missing separator, extra segments) could be activated and fail silently at runtime in ingest/derive |
| Artifact `SchemaVersion` and `RuntimeLoader` accepted free-form strings | Typos or version drift could produce artifacts that no runtime knows how to load |
| Known family registries were opaque (unexported maps) | Tooling and coherence tests couldn't programmatically query the canonical catalog |
| Test fixtures used `"validator:v1"` as runtime loader | Inconsistent with the canonical `"configctl-sync/v1"` convention |

## 3. Changes Made

### 3.1 Code — Pipeline Config Validation (`settings/schema.go`)

| Change | Detail |
|--------|--------|
| Duplicate family detection | Added `rejectDuplicates()` helper called for all 6 family lists before any other validation in `ValidatePipeline()` |
| Exported `PipelineDomain` type | `DomainEvidence`, `DomainSignal`, `DomainDecision`, `DomainStrategy`, `DomainRisk`, `DomainExecution` |
| Exported `KnownFamilies(domain)` | Returns the canonical family names for a given domain |
| Exported `IsKnownFamily(domain, family)` | Reports whether a family is in the canonical catalog |
| Exported `DependencyGraph()` | Returns the full cross-layer dependency map as `[]FamilyDependency` |

### 3.2 Code — Configctl Document Validation (`domain/configctl/document.go`)

| Change | Detail |
|--------|--------|
| Binding topic format validation | Added `isValidBindingTopic()` check: topic must be `source.symbol` where both segments are lowercase alphanumeric with underscores |

### 3.3 Code — Compilation Artifact Validation (`domain/configctl/runtime.go`)

| Change | Detail |
|--------|--------|
| Known schema version whitelist | `knownSchemaVersions` map; `NewCompilationArtifact()` rejects unknown versions |
| Known runtime loader whitelist | `knownRuntimeLoaders` map; `NewCompilationArtifact()` rejects unknown loaders |

### 3.4 Test Fixtures Updated

| File | Change |
|------|--------|
| `domain/configctl/config_set_test.go` | `"validator:v1"` → `"configctl-sync/v1"` (2 locations) |
| `application/configctl/usecases_test.go` | `"validator:v1"` → `"configctl-sync/v1"` (4 locations) |
| `application/runtimecontracts/runtime_test.go` | `"validator:v1"` → `"configctl-sync/v1"` (1 location) |
| `adapters/repositories/memory/configctl/repository_test.go` | `"validator:v1"` → `"configctl-sync/v1"` (3 locations) |

### 3.5 New Tests

| Test | What it verifies |
|------|-----------------|
| `TestValidatePipelineRejectsDuplicateEvidenceFamily` | Duplicate `candle` in evidence families produces validation error |
| `TestValidatePipelineRejectsDuplicateSignalFamily` | Duplicate `rsi` in signal families produces validation error |
| `TestValidatePipelineRejectsDuplicateExecutionFamily` | Duplicate `paper_order` in execution families produces validation error |
| `TestKnownFamiliesReturnsRegisteredNames` | Exported catalog returns correct count and content |
| `TestIsKnownFamilyReturnsFalseForUnknown` | Lookup rejects unknown families |
| `TestDependencyGraphCoversAllNonEvidenceFamilies` | All non-evidence families appear in the dependency graph |
| `TestDocumentValidationRejectsInvalidBindingTopicFormat` | 9 test cases covering valid/invalid topic formats |
| `TestNewCompilationArtifactRejectsUnknownSchemaVersion` | Unknown schema version rejected |
| `TestNewCompilationArtifactRejectsUnknownRuntimeLoader` | Unknown runtime loader rejected |

### 3.6 Architecture Documentation

| Document | Purpose |
|----------|---------|
| `docs/architecture/config-activation-and-dependency-map-model.md` | Canonical reference for activation flow, dependency maps, runtime dependencies, and "how to add a new family" guide |
| `docs/architecture/config-validation-and-sync-rules.md` | Complete validation matrix across all 5 layers, sync rules, and what remains manual |

## 4. Files Changed

| File | Type |
|------|------|
| `internal/shared/settings/schema.go` | Modified — duplicate detection, exported catalog API |
| `internal/shared/settings/settings_test.go` | Modified — 6 new tests |
| `internal/domain/configctl/document.go` | Modified — binding topic format validation |
| `internal/domain/configctl/runtime.go` | Modified — artifact metadata whitelists |
| `internal/domain/configctl/config_set_test.go` | Modified — 3 new tests + fixture update |
| `internal/application/configctl/usecases_test.go` | Modified — fixture update |
| `internal/application/runtimecontracts/runtime_test.go` | Modified — fixture update |
| `internal/adapters/repositories/memory/configctl/repository_test.go` | Modified — fixture update |
| `docs/architecture/config-activation-and-dependency-map-model.md` | New |
| `docs/architecture/config-validation-and-sync-rules.md` | New |
| `docs/stages/stage-s104-config-activation-and-dependency-map-hardening-report.md` | New |

## 5. Fragilities Reduced

| Before | After |
|--------|-------|
| Duplicate family entries silently accepted | Rejected at config validation with specific error message |
| Binding topics accepted any non-empty string | Must follow `source.symbol` convention (lowercase alphanumeric + underscore) |
| Artifact schema version / runtime loader accepted any string | Must be registered in whitelists; unknown values rejected |
| Known family catalog was unexported, opaque to tooling | Exported `KnownFamilies()`, `IsKnownFamily()`, `DependencyGraph()` APIs |
| Test fixtures used legacy `"validator:v1"` | All tests use canonical `"configctl-sync/v1"` |
| No documented validation matrix | Complete 5-layer validation matrix with sync rules |
| No documented "how to add a family" guide | Step-by-step guide with canonical vs. derived artifact classification |

## 6. Limits Maintained

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| No auto-sync between derive and store | Both declare families independently | Not all families need materialization; forcing 1:1 would be over-constraining |
| No hot-reload of pipeline families | Static per-process startup | Hot-reload adds significant complexity for a low-frequency operation |
| No semantic binding topic validation | Format-only, not exchange-aware | Configctl can't know which exchange sources exist at config time |
| No automated config document migration | Schema v1 is stable | No known evolution pressure; premature framework would add cost |
| No cross-config dependency modeling | Individual configs validated independently | Multi-config dependencies are not currently needed |

## 7. Recommended Preparation for S105

Based on remaining gaps and natural next steps:

1. **Cross-registration coherence test** — A test-time check that verifies derive processor families and store pipeline families align with the canonical catalog. This was scoped for S104 but deferred because it crosses package boundaries and would benefit from a shared test harness.

2. **Config document schema versioning** — When the document model needs to evolve, a migration path will be needed. The `SchemaVersion` field on artifacts is already whitelisted; extending it requires explicit registration.

3. **Pipeline family hot-reload** — If operational pressure grows for zero-downtime family changes, this would be the natural next investment. The exported catalog API provides the foundation.

4. **Binding topic semantic validation** — Optional: validate that binding topics reference known exchange sources. This would require a source registry that doesn't exist yet.
