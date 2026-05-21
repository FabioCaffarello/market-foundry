# Runtime Simplification Wave: Capabilities, Questions, and Non-Goals

## Identity

| Field | Value |
|---|---|
| Wave | Runtime Simplification and Futures Proof Prep |
| Charter | S421 |
| Stages | S421-S426 |
| Date | 2026-03-23 |

## Capabilities

These are the capabilities this wave must deliver. Each maps to a block in the charter.

| ID | Capability | Block | Acceptance criterion |
|---|---|---|---|
| RS-C1 | Single canonical execute config with segment/mode parameterization | S422 | `execute.jsonc` serves as template; transitional variants retired or justified |
| RS-C2 | Config selection documented in a single reference table | S422 | Developers can determine which config to use without reading multiple files |
| RS-C3 | Compose surface reduced to base + 2 live overlays | S423 | `docker-compose.spot.yaml`, `docker-compose.futures.yaml`, `docker-compose.unified.yaml` retired |
| RS-C4 | No script, CI, or Makefile references retired compose/config files | S423 | `grep` across repo returns zero hits for retired filenames |
| RS-C5 | Smoke scripts consolidated by capability, not by stage | S424 | Each remaining script has a clear capability label; no two scripts test the same capability |
| RS-C6 | Stage test files consolidated where Spot/Futures are structurally identical | S424 | Parameterized test suites replace duplicated stage-prefixed files |
| RS-C7 | All untracked architecture docs and stage reports committed | S424 | `git status` shows zero untracked files under `docs/` |
| RS-C8 | Full regression suite passes on simplified surface | S425 | All 7 test packages pass; zero test failures |
| RS-C9 | Futures dry-run and venue-live paths exercisable from simplified config/compose | S425 | Consolidated smoke scripts prove Futures accessibility |
| RS-C10 | Entropy reduction measured and classified | S426 | Evidence gate quantifies reduction across all 6 categories |

## Governing Questions

These questions must be answerable at the evidence gate (S426). Each question drives investigation during execution.

| ID | Question | Expected evidence |
|---|---|---|
| RS-Q1 | Can the execute binary be configured for any segment/mode combination from a single config template? | Config consolidation proof showing Spot dry-run, Spot venue-live, Futures dry-run, Futures venue-live, Unified dry-run all derivable from `execute.jsonc` |
| RS-Q2 | Are all transitional compose overlays removable without breaking any operational path? | Dependency scan showing zero references to retired files in Makefile, scripts, CI, and other compose files |
| RS-Q3 | Which smoke scripts are subsumed by later scripts and can be safely retired? | Coverage matrix mapping each smoke script to the capability it exercises, with subsumption relationships |
| RS-Q4 | Can Spot and Futures stage tests be parameterized without losing assertion specificity? | Side-by-side diff of S405/S416, S406/S417, S407/S418, S408/S419 pairs showing structural identity |
| RS-Q5 | Does the simplified surface introduce any regression in the execution layer? | Full test suite pass across all 7 packages with zero failures |
| RS-Q6 | Is the Futures execution path still accessible from the consolidated config and compose surface? | Smoke script execution showing Futures dry-run and venue-live both operational |
| RS-Q7 | What entropy remains after consolidation and why is it justified? | Residual entropy inventory with retention rationale for each item |
| RS-Q8 | Are all 97 untracked documentation files suitable for commit as-is, or do any require corrections? | Documentation audit log showing each file checked and committed or corrected |

## Non-Goals

These items are explicitly excluded from the wave scope. Each is numbered for traceability.

### Execution and Runtime

| ID | Non-goal | Rationale |
|---|---|---|
| NG-41 | Real Futures venue execution proof beyond S420 | S420 closed with PASS. No additional Futures proof is authorized in this wave |
| NG-42 | Production code changes to execution runtime | This is an operational consolidation wave, not an implementation wave |
| NG-43 | Settings schema structural refactor | Schema complexity (1,382 lines) is justified by segment/mode matrix. Refactor requires its own wave |
| NG-44 | New execution modes (paper_trading expansion, backtesting) | Out of scope. Requires OMS expansion wave |
| NG-45 | Segment routing logic changes | Unified runtime routing (S400-S403) is validated and not touched |

### Infrastructure and Deployment

| ID | Non-goal | Rationale |
|---|---|---|
| NG-46 | Separate compose files per segment (spot-only, futures-only deployments) | Contradicts unified runtime architecture. Segment isolation is config-driven, not compose-driven |
| NG-47 | Separate config files per segment as the canonical model | Consolidation eliminates this pattern, not reinforces it |
| NG-48 | CI/CD pipeline changes | No CI exists yet; consolidation prepares for future CI but does not implement it |
| NG-49 | Docker image changes or Dockerfile modifications | Out of scope for operational surface consolidation |
| NG-50 | Kubernetes, Helm, or production deployment artifacts | Not part of the current runtime surface |

### Multi-Exchange and Market Expansion

| ID | Non-goal | Rationale |
|---|---|---|
| NG-51 | Second exchange adapter (Bybit, OKX) | Requires its own wave after runtime simplification |
| NG-52 | Mainnet execution or real money paths | Requires safety controls wave |
| NG-53 | New market segments beyond Spot and USD-M Futures | Out of scope |

### OMS and Domain

| ID | Non-goal | Rationale |
|---|---|---|
| NG-54 | Limit orders, cancel API, position awareness | OMS expansion wave |
| NG-55 | Fee normalization or ClickHouse analytical views | Analytics consolidation wave |
| NG-56 | Dashboard or monitoring expansion | Out of scope |

### Documentation

| ID | Non-goal | Rationale |
|---|---|---|
| NG-57 | Documentation content rewrite or restructuring | This wave commits existing docs and consolidates operational artifacts. Content quality is a separate concern |
| NG-58 | Archive or delete architecture documents | Architecture docs represent delivered evidence. They are committed, not pruned |
| NG-59 | Stage report format changes | Reports follow the established format. No format changes in this wave |

### Code Quality

| ID | Non-goal | Rationale |
|---|---|---|
| NG-60 | Broad refactoring of execution, domain, or adapter packages | Not authorized. Only test consolidation and config simplification are in scope |
| NG-61 | Dead code removal beyond provably unreachable config branches | Requires broader analysis outside this wave |
| NG-62 | Linting, formatting, or style enforcement changes | Out of scope |

## Cumulative Non-Goal Count

| Source | Range | Count |
|---|---|---|
| Prior waves (S370-S420) | NG-1 through NG-40 | 40 |
| This wave (S421-S426) | NG-41 through NG-62 | 22 |
| **Total** | | **62** |

All 62 non-goals remain in force through the evidence gate (S426).
