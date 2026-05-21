# Stage S97 — Registry-Driven Runtime Assembly

**Objective:** Evolve runtime assembly toward catalog-driven patterns, reducing manual list duplication and improving structural scalability as the Foundry grows by families and pipelines.

**Status:** Complete

## Executive Summary

S97 introduced registry-driven assembly patterns across all three core runtimes (store, derive, gateway) to eliminate list duplication, reduce per-family boilerplate, and make adding new families a single-entry operation. The changes are surgical — no new abstractions, no frameworks, no loss of explicitness. The key insight: catalogs were already partially in place (S96's `declarePipelines()`), but derived artifacts (trackers, log fields, scope wiring) still required separate manual lists. S97 closes that gap.

## Changes Applied

### 1. Store: Pipeline Catalog as Single Source of Truth

**Problem:** `cmd/store/run.go` maintained `allTrackerDefs` (40 lines, 9 entries) that duplicated `ProjectionName`, `ConsumerName`, and `IsEnabled` from `declarePipelines()` in `store_supervisor.go`. Adding a pipeline required editing two files with identical data.

**Solution:**
- Added `TrackerDef` struct and `PipelineTrackerDefs()` exported function to `store_supervisor.go`
- `PipelineTrackerDefs()` derives tracker definitions directly from `declarePipelines()`
- Removed `allTrackerDefs` and `trackerDef` type from `cmd/store/run.go`
- `buildTrackers()` now iterates `storeactor.PipelineTrackerDefs()`

**Result:** Adding a new pipeline requires ONE entry in `declarePipelines()`. Trackers follow automatically.

**Lines removed:** ~40 lines of duplicated definitions in `cmd/store/run.go`

### 2. Store: Streamlined Query Responder Registry Wiring

**Problem:** The store supervisor's `start()` had 15 lines of repetitive `if activeScopes[Scope*] { qrCfg.*Registry = &registries.* }` blocks that must be updated for each new scope.

**Solution:**
- Added `queryResponderConfig()` method on `pipelineRegistries`
- Replaced 15-line block with single method call

**Result:** Scope-to-registry wiring is co-located with registry definitions. Adding a new scope requires ONE line in `queryResponderConfig()`.

### 3. Derive: Generic Processor Filtering

**Problem:** `derive_supervisor.go` repeated the same declare → filter → skip-log pattern 6 times (once per scope), totaling ~100 lines of structural duplication.

**Solution:**
- Added `filterEnabled[T any]()` generic function — filters by config predicate, logs skipped families
- Added `familyNames[T any]()` generic function — extracts family names for startup logging
- Applied to all 6 processor scopes in `derive_supervisor.go`
- Applied `familyNames()` to logging blocks in both `derive_supervisor.go` and `source_scope_actor.go`

**Result:** Each scope's filtering is a single `filterEnabled()` call. The pattern is uniform and adding new scopes requires no structural changes.

**Lines removed:** ~70 lines of duplicated filter loops + ~50 lines of duplicated logging blocks

### 4. Gateway: Generic Connection Factory

**Problem:** `cmd/gateway/gateway.go` had 8 near-identical factory functions (135 lines) that differed only in the label string and gateway constructor.

**Solution:**
- Replaced all 8 functions with a single `newGatewayConn[T any]()` generic function (12 lines)
- Each call site in `compose.go` passes the domain-specific constructor as a closure

**Result:** Adding a new gateway requires ONE `newGatewayConn()` call. The factory pattern is enforced by the generic's type parameter.

**Lines removed:** ~120 lines of duplicated factory functions

## Files Changed

| File | Change | Lines Δ |
|------|--------|---------|
| `internal/actors/scopes/store/store_supervisor.go` | Added `TrackerDef`, `PipelineTrackerDefs()`, `queryResponderConfig()`; simplified `start()` | +35, −15 |
| `cmd/store/run.go` | Removed `allTrackerDefs`; `buildTrackers()` uses `PipelineTrackerDefs()` | +8, −40 |
| `internal/actors/scopes/derive/derive_supervisor.go` | Used `filterEnabled()` and `familyNames()` for all 6 scopes | +30, −100 |
| `internal/actors/scopes/derive/source_scope_actor.go` | Added `filterEnabled[T]`, `familyNames[T]`; simplified logging | +35, −40 |
| `cmd/gateway/gateway.go` | Replaced 8 factory functions with `newGatewayConn[T]` | +12, −125 |
| `cmd/gateway/compose.go` | Used `newGatewayConn()` for all connections | +30, −25 |

### Documentation Created

| Document | Purpose |
|----------|---------|
| `docs/architecture/registry-driven-runtime-assembly.md` | When and how to use registry-driven assembly; anti-patterns |
| `docs/architecture/family-runtime-registration-rules.md` | Step-by-step rules for adding families per runtime; checklists |

## Structural Scalability Gains

### Before S97: Cost of Adding a New Pipeline Family

| Step | Store | Derive | Gateway |
|------|-------|--------|---------|
| Catalog entry | 1 entry in `declarePipelines()` | 1 entry per scope | N/A |
| Tracker registration | 1 entry in `allTrackerDefs` (separate file) | N/A | N/A |
| Scope wiring | Manual if/else in `start()` | Manual filter loop | Manual factory function |
| Logging | N/A | Manual name-collection loop | N/A |
| **Total touch points** | **2 files, 2 lists** | **1 file, 2 code blocks** | **2 files, 1 function + call** |

### After S97: Cost of Adding a New Pipeline Family

| Step | Store | Derive | Gateway |
|------|-------|--------|---------|
| Catalog entry | 1 entry in `declarePipelines()` | 1 entry per scope | 1 `newGatewayConn()` call |
| Tracker registration | Automatic | N/A | N/A |
| Scope wiring | Automatic | Automatic via `filterEnabled()` | N/A |
| Logging | N/A | Automatic via `familyNames()` | N/A |
| **Total touch points** | **1 file, 1 entry** | **1 file, 1 entry** | **1 file, 1 call** |

### Net Reduction

- **Store:** 2 synchronization points → 0 (trackers derived, scope wiring encapsulated)
- **Derive:** ~170 lines of structural duplication → ~30 lines using generics
- **Gateway:** 135 lines of factory functions → 12 lines

## Limits and Trade-offs

### What Was NOT Changed

1. **Derive processor types remain distinct.** The 6 processor types (`FamilyProcessor`, `SignalFamilyProcessor`, etc.) have genuinely different `NewActor` signatures. `ExecutionFamilyProcessor` lacks `scopePID` because it's terminal. Unifying them would require `any` casts and lose type safety.

2. **Query responder remains scope-specific.** Each scope has different KV store types and handler signatures. A generic "auto-register" pattern would hide the specific store-opening logic and error handling.

3. **Source scope actor's publisher spawning stays explicit.** Each scope has a distinct publisher type (`EvidencePublisherActor`, `SignalPublisherActor`, etc.). The repetition is structural, not incidental.

4. **Route registration in gateway stays conditional.** Each `FamilyDeps` type has a `HasAny()` method and specific routes. This is inherent to the route model, not reducible by registry patterns.

5. **Venue adapter selection stays explicit.** Security-sensitive activation gates must not be auto-discovered.

### Trade-offs Accepted

| Trade-off | Rationale |
|-----------|-----------|
| Generic functions require accessor closures (`func(p T) string { return p.Family }`) | Go doesn't support structural typing for fields; the accessor is 1 line per call site and provides type safety |
| `PipelineTrackerDefs()` calls `declarePipelines()` which creates registries just to discard them | Cost is negligible (6 struct instantiations at startup); keeping tracker derivation co-located with the catalog is worth it |
| `newGatewayConn[T]` requires a closure per call site | Each closure is 2 lines; the alternative was 8 identical 15-line functions |

## Preparation for S98

The following areas are now better positioned for growth:

1. **New pipeline families** — adding evidence types, signal variants, or new scopes is a single-entry operation with no synchronization burden.
2. **MarketMonkey absorption** — when new families arrive from MarketMonkey, they slot into existing catalogs without structural changes.
3. **Query responder splitting** — when the query responder grows beyond ~5 scope types, splitting by scope will be straightforward since registries are already scope-keyed.
4. **Dynamic family discovery** — if future requirements call for config-driven family sets (beyond the current static catalogs), the catalog pattern is compatible: replace the slice literal with a config-derived builder.

### Recommended S98 Focus Areas

- **Derive pipeline chain unification** — the 6 processor types + 6 publisher types in the source scope actor represent inherent domain complexity, but the spawn-and-route pattern could benefit from a chain abstraction if more scopes are added.
- **Query responder refactoring** — as scope count approaches 5+, consider splitting into per-scope responders for independent scaling and fault isolation.
- **Configuration validation hardening** — ensure that `knownXxxFamilies` sets stay in sync with the pipeline catalogs via compile-time or startup-time assertions.
