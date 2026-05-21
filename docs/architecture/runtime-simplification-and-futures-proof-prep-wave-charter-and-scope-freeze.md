# Runtime Simplification and Futures Proof Prep Wave: Charter and Scope Freeze

## Identity

| Field | Value |
|---|---|
| Wave | Runtime Simplification and Futures Proof Prep |
| Charter stage | S421 |
| Predecessor gate | S420 (Futures Venue Execution Proof, PASS) |
| Date opened | 2026-03-23 |
| Scope frozen | 2026-03-23 |

## Strategic Context

Eight consecutive passing waves (S370-S420) have proven the execution layer for dual-segment testnet operation. The unified runtime supports both Spot and Futures on Binance testnet with full lifecycle evidence. However, this rapid delivery accumulated significant operational entropy:

- **6 compose overlays** where 3 suffice (83% transitional).
- **6 execute config variants** that encode segment x mode combinations without template inheritance.
- **25 stage-specific smoke scripts** layered without consolidation across 8 waves.
- **39 stage-prefixed test files** with near-identical Spot/Futures test suites.
- **97 untracked documentation files** (65 architecture + 32 stage reports) spanning S388-S420.

This entropy is not architectural debt — the runtime architecture is validated. It is operational surface sprawl that slows onboarding, increases maintenance friction, and obscures the canonical paths.

## Wave Objective

Consolidate the runtime, config, compose, smoke, and documentation surfaces to reduce entropy by 40-50%, preserving all proven capabilities and producing zero regressions. Prepare a clean base for future macro-directions (OMS expansion, mainnet readiness, or multi-exchange).

## Wave Blocks

### Block 1: Execute Config Consolidation (S422)

**Problem**: 6 execute config files encode the cross-product of {spot, futures, unified} x {dry_run, venue_live}. Each is a minimal stub but the naming convention is ad-hoc and the relationship between them is undocumented.

**Scope**:
- Establish `execute.jsonc` as the single canonical template with segment and mode as runtime-switchable parameters.
- Retire `execute-spot.jsonc`, `execute-futures.jsonc`, `execute-unified.jsonc` by folding their differences into the canonical template with environment variable or CLI flag overrides.
- Retain `execute-venue-live-spot.jsonc` and `execute-venue-live-futures.jsonc` only if their differences are non-trivial; otherwise fold into the canonical template.
- Document the config selection model in a single reference table.

**Target**: 6 execute configs -> 1-2 canonical + 0-2 mode-specific overlays.

### Block 2: Compose Surface Consolidation (S423)

**Problem**: 5 transitional compose overlays (spot, futures, unified, unified-spot-live, unified-futures-live) accumulated linearly from S394-S419. The intermediate variants (spot-only, futures-only, unified-without-live) are superseded by the unified-live variants.

**Scope**:
- Establish `docker-compose.yaml` as the canonical base.
- Retain `docker-compose.unified-spot-live.yaml` and `docker-compose.unified-futures-live.yaml` as the only operational overlays.
- Retire `docker-compose.spot.yaml`, `docker-compose.futures.yaml`, `docker-compose.unified.yaml` as transitional artifacts.
- Verify that no script, CI, or Makefile target references the retired files before removal.

**Target**: 6 compose files -> 3 (base + 2 live overlays).

### Block 3: Transitional Artifact Removal and Taxonomy Cleanup (S424)

**Problem**: 25 stage-specific smoke scripts and 39 stage-prefixed test files represent evidence from completed waves. The evidence value is preserved in stage reports; the runtime value should be consolidated into canonical suites.

**Scope**:
- **Smoke scripts**: Categorize all 25 by capability (exchange-listening, dry-run, venue-live, segment-isolation, endurance, compose-wiring, multi-binary). Consolidate overlapping scripts into canonical capability-based suites. Retire scripts whose coverage is subsumed. Target: 25 -> 10-14.
- **Stage test files**: Identify Spot/Futures test pairs with near-identical structure (S405/S416, S406/S417, S407/S418, S408/S419). Consolidate into parameterized test suites that exercise both segments. Target: reduce duplication by 30-40%.
- **Documentation commit**: Commit all 97 untracked files (65 architecture + 32 stage reports) with a single audited batch. No content changes — this is a commit-only action to establish the baseline.

**Target**: Smoke scripts 25 -> 10-14. Stage tests consolidated by parameterization. Docs committed.

### Block 4: Unified Runtime Smoke and Futures Preflight Proof (S425)

**Problem**: After consolidation, the runtime must prove zero regressions. Additionally, the Futures execution path must remain exercisable from the simplified surface.

**Scope**:
- Run full regression suite across all 7 test packages.
- Exercise the consolidated smoke scripts against the unified runtime.
- Prove that Futures dry-run and venue-live paths remain accessible from the simplified config/compose surface.
- Document any consolidation that revealed dead code or unreachable paths.

**Target**: Zero regressions. Futures preflight passes on simplified surface.

### Block 5: Evidence Gate (S426)

**Problem**: Formal closure with entropy metrics.

**Scope**:
- Measure entropy reduction across all 6 categories (compose, config, smoke, tests, docs, settings schema).
- Classify each consolidation action as FULL, SUBSTANTIAL, or PARTIAL.
- Identify any residual entropy that was deliberately preserved and justify retention.
- Recommend next macro-direction from the simplified base.

**Target**: Wave closure with measurable entropy reduction.

## Execution Order

```
S421  Charter and scope freeze (this document)
S422  Execute config consolidation
S423  Compose surface consolidation
S424  Transitional artifact removal and taxonomy cleanup
S425  Unified runtime smoke and Futures preflight proof
S426  Evidence gate
```

All blocks are sequential. Each block requires the previous to be complete before execution.

## Entropy Baseline (Frozen at S421)

| Category | Pre-wave count | Target post-wave | Reduction target |
|---|---|---|---|
| Compose files | 6 | 3 | 50% |
| Execute config variants | 6 | 1-2 | 67-83% |
| Smoke scripts | 25 stage-specific | 10-14 canonical | 44-60% |
| Stage test files | 39 stage-prefixed | ~25 consolidated | ~36% |
| Untracked docs | 97 | 0 (all committed) | 100% |
| Settings schema lines | 1,382 | ~1,382 (no refactor) | 0% |

**Overall target: 40-50% entropy reduction across operational surfaces.**

## Invariants

1. Zero production code changes to execution runtime, domain, or adapter layers.
2. Zero regressions against all prior wave test suites (S370-S420).
3. All 40 non-goals from prior waves remain respected.
4. Unified runtime architecture (S400-S403) is not modified.
5. Segment routing logic is not modified.
6. Settings schema is not structurally changed (validation simplification only if branches are provably dead).

## Dependencies

- All prior waves (S370-S420) closed with PASS verdicts.
- No open medium or high severity gaps from S420.
- 15 low-severity residual gaps (RG-2 through RG-15) are carried forward unchanged.

## Preparation for S422

Before starting S422 (execute config consolidation):

1. Read all 6 execute config files to catalog actual differences.
2. Read `internal/shared/settings/schema.go` to understand how config files map to runtime behavior.
3. Identify which config fields are segment-specific vs mode-specific vs common.
4. Identify all Makefile targets, scripts, and compose files that reference specific execute config filenames.
5. Design the consolidated config surface before any file operations.
