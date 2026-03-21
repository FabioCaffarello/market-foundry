# Scripts Normalization And Harness Hygiene

## Purpose

This document defines the C3 normalization posture for repository support scripts.

The goal is to keep the script surface reliable, discoverable, and low-entropy without
changing domain semantics or introducing a new internal scripting framework.

## Scope

C3 covered the repository support scripts under `scripts/` and the helper utilities
under `scripts/utils/`.

Inventory at the time of normalization:

- 10 primary shell entrypoints under `scripts/`
- 3 shared/helper shell utilities under `scripts/utils/`
- 5,486 total lines across the shell support surface

## Normalization Decisions

### 1. Prefer logical consolidation over physical mass moves

The repository already has extensive stage-history and architecture references to
existing script paths. A broad relocation would increase doc churn and review risk
without producing proportional operational value.

C3 therefore keeps the current file locations and normalizes the surface through:

- shared helper reuse
- stricter argument parsing
- explicit preflight checks
- better script self-documentation
- a cataloged canonical entrypoint model

### 2. Treat `make` as the canonical public surface when it already exists

For operator-facing flows that already have `make` targets, the target remains the
canonical entrypoint. Direct script execution remains supported for lower-level use,
extra flags, and local debugging.

### 3. Keep helper logic centralized, but minimal

`scripts/utils/lib.sh` remains the shared harness library. C3 extends it only with
small, reusable primitives:

- color output that degrades cleanly when color is disabled
- `die`
- `usage_error`
- `require_commands`
- `require_positive_integer`

This avoids a framework-shaped abstraction while removing repeated shell fragility.

## Problems Addressed

### Argument parsing was weak or inconsistent

Several scripts either:

- accepted only positional inspection of `$1`
- silently ignored unknown flags
- treated arbitrary strings as wait values
- provided no `--help`

This made failures less predictable and reduced discoverability.

### Helper reuse was incomplete

Common logging and preflight behavior was duplicated across scripts such as:

- `seed-configctl.sh`
- `diag-check.sh`
- `live-pipeline-activate.sh`
- `smoke-first-slice.sh`
- `smoke-multi-symbol.sh`

### Preflight checks were uneven

Some scripts assumed required tools or runtime readiness instead of asserting them.
That increased time-to-failure and made local troubleshooting noisier.

### Utility scripts were real but under-documented

The module helpers in `scripts/utils/` were useful, but they were not self-explanatory
from the command line and had weak failure messaging.

## Changes Applied

### Shared harness layer

Updated `scripts/utils/lib.sh` to provide:

- safer color handling
- shared hard-fail semantics via `die`
- consistent usage failure output via `usage_error`
- explicit dependency checks via `require_commands`
- numeric wait validation via `require_positive_integer`

### Bootstrap and local operation entrypoints

Updated:

- `scripts/seed-configctl.sh`
- `scripts/diag-check.sh`
- `scripts/live-pipeline-activate.sh`

Changes:

- added `--help`
- reject unknown arguments
- added command preflight checks
- reduced duplicated helper definitions
- added a readiness preflight to `seed-configctl.sh`
- normalized symbol sanitization in `seed-configctl.sh`

### Smoke and recovery harnesses

Updated:

- `scripts/smoke-first-slice.sh`
- `scripts/smoke-multi-symbol.sh`
- `scripts/smoke-os-process-operational.sh`
- `scripts/smoke-restart-recovery.sh`

Changes:

- added `--help`
- normalized `--wait <seconds>` parsing
- fail fast on invalid wait values
- added dependency preflight checks
- kept domain validation logic unchanged

### Codegen and helper surfaces

Updated:

- `scripts/codegen-integrated-check.sh`
- `scripts/codegen-equivalence-check.sh`
- `scripts/utils/for-each-module.sh`
- `scripts/utils/list-modules.sh`

Changes:

- added `--help`
- reject unknown arguments
- added minimal dependency preflight checks
- improved utility discoverability
- removed a small unnecessary pipeline in `codegen-integrated-check.sh`

## Deliberate Non-Changes

### No mass relocation of scripts

Physical regrouping was intentionally deferred. The current path layout is already
embedded in active documentation and stage evidence, so a move-only reorganization
would increase entropy before it reduces it.

### No change to domain semantics

C3 did not change:

- seed payload semantics
- smoke validation semantics
- analytical query logic
- restart/recovery semantics

### No Makefile expansion in this step

The current worktree already contains in-flight changes in `Makefile`, `README.md`,
`DEVELOPMENT.md`, and some support scripts. C3 therefore improved discoverability
through script help text plus the new operations catalog, without competing with the
active tranche in those files.

## Validation Performed

Executed during C3:

- `bash -n scripts/*.sh scripts/utils/*.sh`
- `--help` smoke test for each normalized script and helper

Not executed in C3:

- full live/smoke runtime flows against a running stack
- end-to-end codegen checks against the active dirty worktree

Those runtime validations remain environment-dependent and would risk conflating
normalization work with unrelated in-flight changes.

## Maintenance Rules Going Forward

- Keep `scripts/utils/lib.sh` small and procedural.
- Add `--help` to any new operator-facing script.
- Validate required commands before performing expensive work.
- Prefer explicit flags over positional magic for optional behavior.
- Prefer logical cataloging over moving files unless a move has clear operational gain.
- Prefer `make` as the public entrypoint when a stable target already exists.
