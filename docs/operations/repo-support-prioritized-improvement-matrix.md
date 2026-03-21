# Repo Support Prioritized Improvement Matrix

## Prioritization Model

This matrix prioritizes support-surface work using four lenses:

- operational return
- implementation risk
- conflict risk with the active wave
- ability to run in parallel without touching domain flow

Legend:

- Severity: `high`, `medium`, `low`
- Fix risk: `low`, `medium`, `high`
- Parallel-safe now: `yes`, `partial`, `no`

## Quick Wins

| ID | Improvement | Problem Type | Severity | Fix Risk | Parallel-Safe Now | Why It Matters | Recommended Mini-Stage |
|---|---|---|---|---|---|---|---|
| Q1 | Create a real `docs/operations/` home and move or index current runbooks from architecture docs | discoverability, organization | high | low | yes | gives contributors one current-state landing zone for operational support docs | C2 |
| Q2 | Repair `DEVELOPMENT.md` path drift (`cmd/migrate/engine`, `internal/shared/webserver`) | organization | medium | low | yes | fixes a read-first document that currently points at stale layout | C2 |
| Q3 | Fix `raccoon-cli` text drift: remove `make up-dataplane` guidance and align deep-profile/runtime-smoke wording | robustness | high | low | yes | remediation output should never point to nonexistent commands | C3 |
| Q4 | Fix `coverage-map` runtime command label from nonexistent `raccoon-cli smoke-e2e` to the supported surface | discoverability | high | low | yes | removes a direct tooling contradiction | C3 |
| Q5 | Add a brief “core workflow vs expert commands” section linking Make targets to direct `raccoon-cli` commands | discoverability | medium | low | yes | reduces command-surface ambiguity without changing behavior | C2 or C3 |

## Light Structural Improvements

| ID | Improvement | Problem Type | Severity | Fix Risk | Parallel-Safe Now | Why It Matters | Recommended Mini-Stage |
|---|---|---|---|---|---|---|---|
| S1 | Add a `raccoon-cli tdd` guard for very large dirty worktrees, with a suggestion to pass explicit targets | ergonomics | medium | low | yes | preserves signal during active waves instead of dumping hundreds of files | C3 |
| S2 | Add promoted wrappers or doc aliases for hidden support flows: restart-recovery smoke and codegen equivalence | discoverability | medium | low | no | good improvement, but these scripts are part of the active worktree and should wait | C4 |
| S3 | Consolidate duplicated Makefile module-iteration logic behind shared helpers | organization, robustness | low | low | no | reduces repetitive shell logic, but `Makefile` is currently in conflict zone | C4 |
| S4 | Link root docs to `docs/tooling/cli-overview.md` and the future `docs/operations/` index | discoverability | medium | low | yes | reduces dependency on repo folklore | C2 |
| S5 | Normalize shell helper reuse so `diag-check.sh` and `live-pipeline-activate.sh` source `scripts/utils/lib.sh` | robustness | medium | medium | partial | worthwhile, but touches operational scripts and should remain surgical | C5 |

## Guard Rail Gaps To Address After the Current Wave

| ID | Improvement | Problem Type | Severity | Fix Risk | Parallel-Safe Now | Why It Matters | Recommended Mini-Stage |
|---|---|---|---|---|---|---|---|
| G1 | Add an explicit CI quality-gate job using `make quality-gate-ci` or `make check` | guard rails | high | medium | no | published guard rails are not currently enforced by CI | C4 |
| G2 | Decide whether CI should also run `make raccoon-test` | guard rails | medium | medium | no | protects the governance tool itself, not just the Go code | C4 |
| G3 | Align CI documentation with the actual enforced pipeline after G1/G2 land | organization | high | low | no | prevents workflow docs from overpromising coverage | C4 |

## Items To Delay

| ID | Deferred Item | Why Defer | Trigger To Revisit |
|---|---|---|---|
| D1 | Break up `smoke-multi-symbol.sh` and `smoke-analytical-e2e.sh` into smaller libraries | high coordination cost and elevated regression risk while the functional tranche is active | after current wave stabilizes and support scripts stop moving |
| D2 | Large-scale archival or re-homing of architecture and stage history | this is documentation surgery, not a support quick win | after an agreed documentation governance stage |
| D3 | Replacing shell-based smoke flows with a new runner or framework | changes execution semantics and likely collides with operational proof work | only with explicit charter |
| D4 | Broad `Makefile` redesign | low-value if done mid-wave; conflict risk is high because `Makefile` is already dirty | after C4 guard rail alignment |

## Proposed Safe Sequence

### C2. Docs Consolidation Without Behavioral Changes

Scope:

- create `docs/operations/index` style guidance
- repair stale support doc paths
- link root docs to tooling and operations surfaces

Why first:

- highest return
- lowest conflict
- zero impact on runtime behavior

### C3. Raccoon CLI Guidance Hygiene

Scope:

- fix `make up-dataplane` references
- resolve `runtime-smoke` and deep-profile wording
- fix `coverage-map` command naming
- add dirty-worktree TDD signal protection

Why second:

- support surface only
- mostly text and presentation changes
- no need to touch current domain tranche

### C4. Post-Wave Makefile and CI Alignment

Scope:

- add CI quality-gate enforcement
- decide on `raccoon-test` in CI
- expose any missing support entrypoints in Make/help

Why third:

- very valuable
- but current worktree already modifies `Makefile` and `.github/workflows/ci.yml`

### C5. Shell Helper Extraction and Script Hardening

Scope:

- reuse `scripts/utils/lib.sh`
- eliminate duplicated compose/logging helpers
- keep edits surgical, not architectural

Why fourth:

- useful hygiene improvement
- touches large operational scripts, so should come only after C4 stabilizes workflow surfaces

## Explicit Do-Not-Touch List

These should remain outside the next mini-stages unless the charter changes:

- domain logic under `internal/domain/`, `internal/application/`, `internal/adapters/`, `internal/actors/`
- service contracts and runtime topology in `deploy/`
- active wave artifacts already modified in the current worktree
- broad archive surgery in `docs/architecture/` and `docs/stages/`

## Recommended C2+ Prompts

### C2 Prompt

Audit accepted. Implement only the safe docs consolidation layer for repository support surfaces:

- create an operations landing page under `docs/operations/`
- add minimal cross-links from `README.md` and `DEVELOPMENT.md`
- repair stale support-surface paths
- do not modify domain/runtime code

### C3 Prompt

Implement only `raccoon-cli` support hygiene fixes:

- remove or replace `make up-dataplane` guidance
- align runtime-smoke/deep-profile wording across help, README, and emitted output
- fix `coverage-map` to reference the real runtime command surface
- improve `tdd` behavior for very large dirty worktrees

### C4 Prompt

After the active wave lands, align Makefile and CI guard rails:

- ensure CI runs the published quality-gate path
- decide whether `raccoon-test` belongs in CI
- expose missing support entrypoints in `make help`
- preserve current functional tranche behavior

### C5 Prompt

Perform a surgical shell hygiene pass only on support scripts:

- consolidate duplicated helpers onto `scripts/utils/lib.sh`
- avoid changing scenario semantics
- keep smoke behavior identical
