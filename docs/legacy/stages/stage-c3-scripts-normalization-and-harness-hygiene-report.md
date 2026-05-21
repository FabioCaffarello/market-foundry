# Stage C3: Scripts Normalization And Harness Hygiene Report

## 1. Executive Summary

C3 normalized the repository script surface as support infrastructure, not as a
domain-work vehicle.

The work focused on:

- inventorying the real shell surface
- classifying scripts by operational purpose
- removing repeated shell fragility
- improving fail-fast behavior
- making direct script usage more discoverable
- documenting the final support catalog

The result is a clearer and more predictable script layer without a big-bang move
or semantic changes to domain flows.

## 2. Inventory And Classification

### 2.1 Current Shell Inventory

At C3 execution time the repository contained:

- 10 primary entry scripts under `scripts/`
- 3 shared/helper scripts under `scripts/utils/`
- 5,486 total lines across the shell support surface

### 2.2 Classification

| Script | Category | Canonical Entry | Notes |
|---|---|---|---|
| `scripts/seed-configctl.sh` | bootstrap/setup | `make seed`, `make seed-multi` | config lifecycle seeding |
| `scripts/live-pipeline-activate.sh` | bootstrap/setup | `make live*` | full activation/validation harness |
| `scripts/smoke-first-slice.sh` | smoke/integration | `make smoke` | smallest operational proof |
| `scripts/smoke-multi-symbol.sh` | smoke/integration | `make smoke-multi` | multi-symbol operational proof |
| `scripts/smoke-analytical-e2e.sh` | smoke/integration | `make smoke-analytical` | analytical writer/reader proof |
| `scripts/smoke-os-process-operational.sh` | smoke/integration | `make smoke-operational` | isolated-process operational proof |
| `scripts/smoke-restart-recovery.sh` | smoke/integration | `make smoke-restart-recovery` | restart/recovery proof |
| `scripts/diag-check.sh` | local dev | `make diag` | lightweight diagnostics snapshot |
| `scripts/codegen-integrated-check.sh` | docs/tooling | `make codegen-integrated` | governed slice verification |
| `scripts/codegen-equivalence-check.sh` | docs/tooling / stage support | `make codegen-equivalence` | cross-artifact equivalence harness |
| `scripts/utils/for-each-module.sh` | local dev helper | direct only | module iterator |
| `scripts/utils/list-modules.sh` | local dev helper | direct only | module inventory helper |
| `scripts/utils/lib.sh` | shared harness support | source-only | shared procedural helpers |

## 3. Problems Found

### 3.1 Helper duplication

Logging, fail behavior, and shell preflight logic were duplicated across multiple
scripts instead of consistently reusing `scripts/utils/lib.sh`.

### 3.2 Fragile argument handling

Several scripts:

- only peeked at `$1`
- silently accepted unknown flags
- treated arbitrary strings as wait values
- lacked `--help`

This lowered predictability and slowed operator troubleshooting.

### 3.3 Uneven dependency and readiness preflight

Command dependencies such as `curl`, `python3`, `docker`, `go`, and `make` were
often assumed rather than asserted. `seed-configctl.sh` also lacked an explicit
gateway readiness preflight.

### 3.4 Utility discoverability was weak

The module helpers in `scripts/utils/` were useful but under-documented and
non-obvious when invoked directly.

### 3.5 Physical reorganization had poor cost/benefit

Mass moves would have created broad doc churn because existing stage evidence and
architecture docs reference current script paths heavily. The entropy reduction from
path moves alone was not strong enough to justify that disruption.

## 4. Changes Performed

### 4.1 Shared harness hardening

Updated `scripts/utils/lib.sh` with:

- color handling that degrades cleanly without terminal color support
- `die`
- `usage_error`
- `require_commands`
- `require_positive_integer`

### 4.2 Entry script normalization

Added `--help`, explicit argument validation, and dependency preflight to:

- `scripts/seed-configctl.sh`
- `scripts/diag-check.sh`
- `scripts/live-pipeline-activate.sh`
- `scripts/smoke-first-slice.sh`
- `scripts/smoke-multi-symbol.sh`
- `scripts/smoke-os-process-operational.sh`
- `scripts/smoke-restart-recovery.sh`
- `scripts/codegen-integrated-check.sh`
- `scripts/codegen-equivalence-check.sh`

### 4.3 Bootstrap/setup predictability

`scripts/seed-configctl.sh` now additionally:

- sanitizes symbol input
- fails early if no symbols resolve
- checks gateway readiness before mutating config state

### 4.4 Helper discoverability

Added direct usage/help surfaces to:

- `scripts/utils/for-each-module.sh`
- `scripts/utils/list-modules.sh`

### 4.5 Documentation surface added

Created:

- `docs/operations/scripts-normalization-and-harness-hygiene.md`
- `docs/operations/scripts-catalog-and-usage-guide.md`

## 5. Final Catalog

The normalized operational contract is:

- use `make` for public workflows already exposed by the repository
- use direct script invocation for lower-level debugging or flag overrides
- use `scripts/utils/*` as support utilities rather than primary workflows
- keep shared shell behavior small and procedural in `scripts/utils/lib.sh`

High-value canonical flows after C3:

- bootstrap/setup: `make live`, `make live-multi`, `make seed`, `make seed-multi`
- smoke/integration: `make smoke`, `make smoke-multi`, `make smoke-analytical`, `make smoke-operational`, `make smoke-restart-recovery`
- diagnostics: `make diag`
- codegen/stage support: `make codegen-integrated`, `make codegen-equivalence`

## 6. Validation

Validated in C3:

- `bash -n scripts/*.sh scripts/utils/*.sh`
- `--help` execution for each normalized script/helper

Not run in C3:

- full runtime smoke flows against a live stack
- full codegen/runtime proof commands against the current dirty worktree

Those checks were intentionally left out because the repository already has unrelated
in-flight work in `Makefile`, `README.md`, `DEVELOPMENT.md`, `scripts/smoke-analytical-e2e.sh`,
and newly added support/domain artifacts.

## 7. Follow-Ups Recommended

### 7.1 Decompose the two largest harnesses carefully

The biggest remaining shell concentration risks are:

- `scripts/smoke-analytical-e2e.sh`
- `scripts/smoke-multi-symbol.sh`

They should be decomposed only when there is a clear operational need, not as a
cosmetic refactor.

### 7.2 Reconcile catalog discoverability with active Makefile work

After the in-flight `Makefile`/doc edits settle, consider adding the new operations
docs to the canonical doc surface so the catalog is reachable from existing entrypoints.

### 7.3 Keep new scripts inside the same contract

Any new support script should ship with:

- `set -euo pipefail`
- `--help`
- explicit dependency preflight
- explicit optional-flag parsing
- reuse of `scripts/utils/lib.sh` when shared behavior is needed
