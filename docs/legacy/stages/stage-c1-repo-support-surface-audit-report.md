# Stage C1 Repo Support Surface Audit Report

## Summary

C1 audited the repository support surfaces of `market-foundry` with the explicit constraint that the active functional tranche must not be disturbed.

The audit examined:

- `Makefile`
- `.github/workflows/ci.yml`
- executable support scripts in `scripts/`
- shared shell utilities in `scripts/utils/`
- `tools/raccoon-cli/`
- `README.md`
- `DEVELOPMENT.md`
- `docs/tooling/`
- operationally relevant documents in `docs/architecture/`

Deliverables produced:

- `docs/operations/repo-support-surface-audit.md`
- `docs/operations/repo-support-prioritized-improvement-matrix.md`
- `docs/stages/stage-c1-repo-support-surface-audit-report.md`

## Acceptance Criteria Check

### Clear and Prioritized View of Support Surfaces

Met.

The audit documents inventory the concrete support surfaces and classify them by:

- ergonomics
- robustness
- discoverability
- documentation organization
- guard rails

### Diagnosis Distinguishes What Is Safe To Change Now

Met.

The audit explicitly separates:

- safe parallel work
- post-wave work
- items outside current scope
- current worktree conflict zones

### Action Plan Reduces Conflict Risk With the Main Wave

Met.

The recommended sequence keeps C2 and C3 on docs and `raccoon-cli` guidance surfaces, while deferring Makefile and CI changes until the active wave stabilizes.

### Analysis Anchored in the Real Codebase

Met.

The audit used the live worktree and command outputs, including:

- `make help`
- `make check`
- `git status --short`
- `raccoon-cli --help`
- `raccoon-cli quality-gate --help`
- `raccoon-cli tdd`
- `raccoon-cli --json coverage-map`

## Key Findings

### 1. Operations discoverability is the largest immediate gap

The repo had no `docs/operations/` namespace even though current operational guidance already exists in architecture docs such as:

- `current-baseline-runbook.md`
- `current-baseline-operational-diagnostics.md`
- `operational-smoke-ci-and-runbook-closure.md`

This is the highest-value docs-only improvement and the safest parallel lane.

### 2. The published guard rail story is stronger than the CI reality

The support surfaces advertise `make check`, `quality-gate`, and `quality-gate-ci` as first-class workflow steps, but the current `ci.yml` worktree does not invoke them.

This is a meaningful guard rail gap, but not a safe parallel change right now because `.github/workflows/ci.yml` is already in the active change set.

### 3. `raccoon-cli` currently contradicts itself in several places

Observed contradictions include:

- deprecated-but-still-executed `runtime-smoke`
- guidance to run nonexistent `make up-dataplane`
- `coverage-map` recommending nonexistent `raccoon-cli smoke-e2e`

This is a high-value, low-risk support fix lane because it is mostly wording and emitted guidance, not functional domain behavior.

### 4. Support entrypoints are split across Make, direct CLI, and hidden scripts

The current repo requires contributors to inspect:

- `make help`
- `raccoon-cli --help`
- architecture docs
- raw scripts

to understand the complete support surface. This is manageable for maintainers and poor for everyone else.

### 5. The support docs are not equally trustworthy

`README.md` is broadly aligned, but `DEVELOPMENT.md` still contains stale tree paths. That makes the “read this first” workflow documentation less reliable than it should be.

## Worktree-Safe Parallel Lanes

Recommended lanes that can run in parallel to the main tranche with low conflict risk:

1. docs-only support consolidation under `docs/operations/`
2. root-doc and tooling-doc cross-link cleanup
3. `raccoon-cli` help/remediation wording fixes
4. `raccoon-cli` TDD UX improvement for dirty worktrees

## Current Conflict Zones

These surfaces should be treated as active-wave territory for now:

- `Makefile`
- `.github/workflows/ci.yml`
- `scripts/codegen-integrated-check.sh`
- `scripts/smoke-analytical-e2e.sh`
- wave-coupled newly added scripts

## Out of Scope

This stage intentionally did not:

- modify bounded contexts
- change runtime contracts
- refactor smoke flows
- rewrite architecture history
- fold stage evidence into new canonical docs

## Validation

`make check` was executed during the audit.

Result:

- passed
- 84 checks
- static guard rails green
- runtime-smoke skipped in fast profile

This confirms the audit is diagnosing support hygiene, not reporting an already broken baseline.

## Recommended Next Prompts

### C2

Implement the docs-only support consolidation layer:

- establish `docs/operations/`
- add an operations landing page
- repair stale support path references
- add minimal cross-links from root docs

### C3

Implement `raccoon-cli` support hygiene fixes only:

- replace `make up-dataplane`
- align runtime-smoke/deep-profile language
- fix `coverage-map` command naming
- improve `tdd` output for dirty worktrees

### C4

After current active work on `Makefile` and CI lands, align guard rails end to end:

- add CI quality-gate enforcement
- evaluate `raccoon-test` in CI
- expose missing support entrypoints in Make/help

### C5

Run a surgical script hygiene pass:

- consolidate duplicated shell helpers
- keep scenario semantics unchanged
- avoid large-scale script rewrites
