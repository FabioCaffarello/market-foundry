# S421: Runtime Simplification and Futures Proof Prep Wave Charter Report

## Stage Identity

| Field | Value |
|---|---|
| Stage | S421 |
| Type | Charter and scope freeze |
| Wave | Runtime Simplification and Futures Proof Prep |
| Scope | S421-S426 |
| Date | 2026-03-23 |
| Predecessor | S420 (Futures Venue Execution Proof Evidence Gate, PASS) |

## Objective

Open a short consolidation tranche to reduce the operational entropy accumulated across eight consecutive delivery waves (S370-S420). The execution architecture is validated — this wave targets the operational surface (configs, compose, smoke scripts, test taxonomy, documentation) without modifying production code.

## Pre-Wave Entropy Assessment

### Quantified Entropy

| Category | Count | Canonical | Transitional | Entropy % |
|---|---|---|---|---|
| Compose files | 6 | 1 | 5 | 83% |
| Execute config variants | 6 | 0 (all ad-hoc) | 6 | 100% |
| Smoke scripts | 35 (10 canonical + 25 stage-specific) | 10 | 25 | 71% |
| Stage-prefixed test files | 39 / 215 total | 176 | 39 | 18% |
| Untracked docs | 97 (65 architecture + 32 reports) | 0 | 97 | 100% |
| Settings schema | 1,382 lines | N/A | N/A | Complex but justified |

### Root Cause

Each wave (S370-S420) added 1-3 smoke scripts, 2-4 stage tests, 1-2 config variants, and 4-5 architecture documents. No consolidation ceremony existed between waves. The artifacts served their purpose as stage evidence but were never folded into canonical suites.

### Impact

- **Onboarding friction**: A new contributor sees 35 scripts, 12 configs, and 6 compose files with no clear entry point.
- **Maintenance cost**: Changes to the compose or config model require updating multiple files.
- **Cognitive load**: 97 untracked docs obscure the actual repository state in `git status`.

## Charter Summary

The wave is organized in 5 execution blocks plus this charter:

| Stage | Block | Description |
|---|---|---|
| S421 | Charter | Scope freeze and entropy baseline (this document) |
| S422 | Execute config consolidation | 6 execute configs -> 1-2 canonical with parameterization |
| S423 | Compose surface consolidation | 6 compose files -> 3 (base + 2 live overlays) |
| S424 | Artifact removal and taxonomy cleanup | Smoke 25->10-14, test consolidation, docs commit |
| S425 | Unified runtime smoke and Futures preflight | Regression proof on simplified surface |
| S426 | Evidence gate | Entropy reduction measurement and wave closure |

## Governing Questions

| ID | Question |
|---|---|
| RS-Q1 | Can the execute binary be configured for any segment/mode from a single config template? |
| RS-Q2 | Are all transitional compose overlays removable without breaking any operational path? |
| RS-Q3 | Which smoke scripts are subsumed by later scripts and can be safely retired? |
| RS-Q4 | Can Spot and Futures stage tests be parameterized without losing assertion specificity? |
| RS-Q5 | Does the simplified surface introduce any regression? |
| RS-Q6 | Is the Futures execution path still accessible from the consolidated surface? |
| RS-Q7 | What entropy remains after consolidation and why? |
| RS-Q8 | Are all 97 untracked docs suitable for commit as-is? |

## Non-Goals (22 new, 62 cumulative)

Key exclusions:

- **NG-41**: No additional Futures venue execution proof.
- **NG-42**: No production code changes.
- **NG-43**: No settings schema structural refactor.
- **NG-46**: No separate compose per segment.
- **NG-47**: No separate config per segment as canonical model.
- **NG-51**: No multi-exchange expansion.
- **NG-52**: No mainnet execution.
- **NG-60**: No broad code refactoring.

Full list in `docs/architecture/runtime-simplification-capabilities-questions-and-non-goals.md`.

## Invariants

1. Zero production code changes to execution runtime, domain, or adapter layers.
2. Zero regressions against all prior wave test suites (S370-S420).
3. All 62 non-goals respected.
4. Unified runtime architecture not modified.
5. Segment routing logic not modified.

## Deliverables

| Artifact | Path |
|---|---|
| Charter and scope freeze | `docs/architecture/runtime-simplification-and-futures-proof-prep-wave-charter-and-scope-freeze.md` |
| Capabilities, questions, non-goals | `docs/architecture/runtime-simplification-capabilities-questions-and-non-goals.md` |
| Stage report | `docs/stages/stage-s421-runtime-simplification-charter-report.md` |

## Preparation for S422

Before starting execute config consolidation:

1. Read all 6 execute config files to catalog field-level differences.
2. Read `internal/shared/settings/schema.go` to map config fields to runtime behavior.
3. Identify all Makefile targets, scripts, and compose files referencing specific execute config filenames.
4. Design the consolidated config surface before any file operations.
5. Ensure all 97 untracked docs are not blocking `git status` clarity for the consolidation work.

## Cumulative Wave History

| Wave | Gate | Verdict |
|---|---|---|
| Multi-binary orchestration (S370-S375) | S375 | PASS |
| Exchange listening + dry-run (S376-S381) | S381 | PASS |
| OMS foundation (S382-S388) | S388 | PASS |
| Binance segmentation (S389-S395) | S395 | PASS |
| Testnet venue execution, Spot-first (S396-S403) | S403 | PASS |
| Testnet venue execution, unified runtime (S404-S409) | S409 | PASS |
| Production readiness hardening (S410-S414) | S414 | PASS |
| Futures venue execution proof (S415-S420) | S420 | PASS |
| **Runtime simplification (S421-S426)** | **S426** | **OPEN** |
