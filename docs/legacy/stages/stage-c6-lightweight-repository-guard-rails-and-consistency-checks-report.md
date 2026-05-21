# Stage C6 - Lightweight Repository Guard Rails And Consistency Checks Report

## Objective

Add a small, robust repository-level consistency guard rail that catches drift in
naming, stage documentation inventory, support-document links, expected docs, and
Makefile script wrappers without expanding into a broad policy platform.

## Scope

C6 was intentionally limited to repository support surfaces:

- naming and inventory conventions for stage reports
- expected documentation entrypoints
- broken local links in primary support docs
- canonical workflow-doc references to Makefile targets
- Makefile wrappers around shell scripts

Non-goals:

- domain-functional changes
- new architecture analyzers inside `raccoon-cli`
- broad markdown linting
- full-corpus documentation cleanup
- heavy policy-as-code expansion

## Invariants Chosen

The stage selected only invariants that are cheap, objective, and directly tied
to operational drift:

1. required repository docs must exist
2. stage reports must keep the `stage-*-report.md` naming convention
3. stage reports must keep minimal shape: title plus at least two sections
4. `docs/stages/INDEX.md` must match the real stage inventory
5. local links in primary support docs must resolve
6. canonical workflow docs must reference real Makefile targets
7. Makefile shell-script wrappers must resolve to executable files

## Changes Made

### New guard rail script

Added:

- `scripts/repository-consistency-check.sh`

This script runs the lightweight repository consistency checks in one fast pass.

### Workflow integration

Updated:

- `Makefile`
- `README.md`
- `DEVELOPMENT.md`
- `docs/operations/README.md`
- `docs/operations/makefile-targets-reference-and-conventions.md`

The new standalone entrypoint is:

```bash
make repo-consistency-check
```

The default workflow now composes it into:

```bash
make check
make verify
make check-deep
```

### Governance and operator docs

Added:

- `docs/operations/lightweight-repository-guard-rails-and-consistency-checks.md`
- `docs/operations/repository-consistency-invariants-and-check-policy.md`

These docs define what is checked, why it is checked, and the explicit limits of
the guard rail.

### Stage navigation

Updated:

- `docs/stages/INDEX.md`

The C-series index now includes C6.

## Validation

Executed:

- `make repo-consistency-check`
- `make check`
- `make verify`

Result:

- passed after the C6 updates landed

## Outcome Against Acceptance Criteria

### Lightweight and useful checks exist

Met.

The implemented checks are inventory- and workflow-oriented, not a broad policy
framework.

### The repository gains protection against real drift

Met.

The new guard rail covers common support-surface drift modes: missing entrypoint
docs, broken support-doc links, stale stage index coverage, naming drift, and
dead Makefile wrappers.

### The solution avoids heavy bureaucracy

Met.

No general policy engine, no broad doc linting, and no new functional-domain
barriers were introduced.

### Governance improves without touching the functional tranche

Met.

All changes stay in support scripts, workflow entrypoints, and repository
documentation.

## Limits

- C6 does not validate the entire architecture corpus for broken links.
- C6 does not enforce a single editorial template across all historical stage
  reports.
- C6 does not replace `raccoon-cli` architecture and topology enforcement.
- C6 does not add review-hostile low-value checks.

## Optional Follow-Ups

1. Add a non-blocking CI job that runs `make repo-consistency-check` alongside
   the existing published guard rails.
2. Extend the primary-doc target validation if more canonical workflow docs are
   introduced.
3. Revisit the support-doc link scope only after the architecture corpus itself
   is intentionally normalized.
