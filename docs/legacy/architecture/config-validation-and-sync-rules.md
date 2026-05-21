# Config Validation and Sync Rules

## Overview

Market Foundry enforces configuration correctness through layered validation at multiple boundaries. This document specifies what is validated, where, and what sync rules apply between config surfaces.

## Validation Layers

### Layer 1 — Static Config (`config.jsonc`)

**When:** Process startup, before any actor spawns.
**Where:** `bootstrap.LoadAndValidate()` → `settings.Load()` + `AppConfig.Validate()`

| Check | Enforced By | Failure Mode |
|-------|------------|--------------|
| JSON/JSONC syntax | `json.Decoder` with `DisallowUnknownFields()` | Process fails to start |
| Unknown top-level fields | `DisallowUnknownFields()` | Process fails to start |
| Log level/format enums | `LogConfig.Validate()` | Process fails to start |
| HTTP timeout durations | `HTTPConfig.Validate()` | Process fails to start |
| NATS URL required when enabled | `NATSConfig.Validate()` | Process fails to start |
| Venue type in whitelist | `VenueConfig.Validate()` | Process fails to start |
| Venue duration bounds | `VenueConfig.Validate()` | Process fails to start |
| Pipeline: unknown family names | `PipelineConfig.ValidatePipeline()` | Process fails to start |
| Pipeline: duplicate family names | `PipelineConfig.ValidatePipeline()` | Process fails to start |
| Pipeline: cross-layer dependency rules | `PipelineConfig.ValidatePipeline()` | Process fails to start |

### Layer 2 — Config Document (Configctl Domain)

**When:** Draft creation and validation via configctl API.
**Where:** `ConfigDocument.Validate()` called by `InspectDocument()`

| Check | Enforced By | Failure Mode |
|-------|------------|--------------|
| Source format (json/yaml) | `ConfigSource.ValidateForDraft()` | Validation diagnostic |
| Source content non-empty | `ConfigSource.ValidateForDraft()` | Validation diagnostic |
| Metadata name non-empty | `ConfigDocument.Validate()` | Validation diagnostic |
| At least one binding | `ConfigDocument.Validate()` | Validation diagnostic |
| At least one field | `ConfigDocument.Validate()` | Validation diagnostic |
| At least one rule | `ConfigDocument.Validate()` | Validation diagnostic |
| Binding names unique | `ConfigDocument.Validate()` | Validation diagnostic |
| Binding topics unique | `ConfigDocument.Validate()` | Validation diagnostic |
| Binding topic format (`source.symbol`) | `ConfigDocument.Validate()` | Validation diagnostic |
| Field names unique | `ConfigDocument.Validate()` | Validation diagnostic |
| Field types in enum | `ConfigDocument.Validate()` | Validation diagnostic |
| Rule names unique | `ConfigDocument.Validate()` | Validation diagnostic |
| Rule field references existing field | `ConfigDocument.Validate()` | Validation diagnostic |
| Rule operator in enum | `ConfigDocument.Validate()` | Validation diagnostic |
| Rule severity in enum (or empty) | `ConfigDocument.Validate()` | Validation diagnostic |
| Equals operator requires expected_value | `ConfigDocument.Validate()` | Validation diagnostic |

### Layer 3 — Compilation Artifact

**When:** `CompileVersion` use case.
**Where:** `NewCompilationArtifact()` in `domain/configctl/runtime.go`

| Check | Enforced By | Failure Mode |
|-------|------------|--------------|
| Artifact ID non-empty | `NewCompilationArtifact()` | Validation problem |
| Schema version known | `NewCompilationArtifact()` | Validation problem |
| Checksum non-empty | `NewCompilationArtifact()` | Validation problem |
| Storage ref non-empty | `NewCompilationArtifact()` | Validation problem |
| Runtime loader known | `NewCompilationArtifact()` | Validation problem |
| Created-at non-zero | `NewCompilationArtifact()` | Validation problem |

### Layer 4 — Activation Scope

**When:** `ActivateVersion` use case.
**Where:** `ActivationScope.Validate()`

| Check | Enforced By | Failure Mode |
|-------|------------|--------------|
| Scope kind non-empty | `ActivationScope.Validate()` | Validation problem |
| Scope key non-empty | `ActivationScope.Validate()` | Validation problem |
| Activation matches version | `ConfigSet.ActivateVersion()` | Conflict problem |

### Layer 5 — Binding Topic Parsing (Ingest/Derive)

**When:** Binding activation at runtime.
**Where:** `ingest.ParseBindingTopic()`

| Check | Enforced By | Failure Mode |
|-------|------------|--------------|
| Topic is `source.symbol` format | `ParseBindingTopic()` | Problem returned, binding skipped |

This is a **redundant safety net** — the same format is now validated in Layer 2 when the config document is created. The ingest-layer check prevents activation of malformed topics that may exist in legacy data.

## Sync Rules

### Rule 1 — Family name consistency

**Invariant:** Every family name used in derive processor registration or store pipeline declaration must exist in the canonical `knownXxxFamilies` registry in `settings/schema.go`.

**Enforcement:** Compile-time (type safety via `IsEnabled` predicate closures) + config validation at startup. If a family name doesn't match the canonical registry, the corresponding processor/pipeline simply won't activate.

**Manual check required:** When adding a new family, verify that derive and store both have corresponding entries if materialization is desired.

### Rule 2 — Cross-layer dependency completeness

**Invariant:** Enabling a downstream family implicitly requires all upstream families in the dependency chain to also be enabled.

**Enforcement:** `PipelineConfig.ValidatePipeline()` — traverses the dependency maps and rejects configs where upstream families are missing.

**Example:** Enabling `paper_order` (execution) without `position_exposure` (risk) produces:
```
execution family "paper_order" requires risk family "position_exposure" to be enabled
```

### Rule 3 — No duplicate family entries

**Invariant:** Each family list in `PipelineConfig` must not contain duplicate entries.

**Enforcement:** `PipelineConfig.ValidatePipeline()` — calls `rejectDuplicates()` for each family list before any other validation.

### Rule 4 — Binding topic format bridge

**Invariant:** Binding topics in configctl documents must follow the `source.symbol` convention expected by ingest and derive.

**Enforcement:** `ConfigDocument.Validate()` checks format; `ParseBindingTopic()` provides runtime safety net.

### Rule 5 — Artifact metadata conventions

**Invariant:** Compilation artifacts must use known schema versions and runtime loaders.

**Enforcement:** `NewCompilationArtifact()` validates against `knownSchemaVersions` and `knownRuntimeLoaders` whitelists.

**Adding new versions:** Register in `domain/configctl/runtime.go` maps before use.

### Rule 6 — Configctl activation scope isolation

**Invariant:** Each activation scope can have exactly one active config version at a time. Activating a new version in an occupied scope automatically deactivates the previous.

**Enforcement:** `ActivateConfigUseCase` handles deactivation as part of the activation transaction.

## What Remains Manual

| Aspect | Why Manual | Risk Level |
|--------|-----------|------------|
| Derive ↔ Store family coherence | Not all families need materialization; forcing 1:1 would be over-constraining | Low — startup logs clearly show enabled families/pipelines |
| Binding topic semantic validity | Configctl can't know which exchange sources exist at config time | Low — ingest will fail to connect and health tracker will report |
| Config document schema evolution | Document structure is stable; no migration framework yet | Low — current schema (v1) has no known evolution pressure |
| Cross-runtime config sync | Pipeline families are static per-process; no hot-reload | Medium — requires coordinated restarts when changing pipeline families |

## Validation Coverage Summary

```
config.jsonc
  ├── [P] JSON syntax + unknown fields
  ├── [P] Log/HTTP/NATS/Venue field validation
  └── [P] Pipeline family validation
        ├── [P] Unknown family names
        ├── [P] Duplicate family names     ← S104
        └── [P] Cross-layer dependencies

ConfigDocument (configctl)
  ├── [P] Structure (metadata, bindings, fields, rules)
  ├── [P] Binding topic format             ← S104
  ├── [P] Uniqueness (names, topics)
  └── [P] Rule-field reference integrity

CompilationArtifact
  ├── [P] Required fields
  ├── [P] Known schema versions            ← S104
  └── [P] Known runtime loaders            ← S104

[P] = Programmatic enforcement (fail-fast)
```
