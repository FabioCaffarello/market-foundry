# Repository Support Surface Canonical Model

## Purpose

This document defines the canonical support-surface model for `market-foundry`.

Its job is to make one thing explicit: when multiple support surfaces coexist,
which one is authoritative for a given kind of work, which ones are auxiliary,
and where ownership sits.

It does not redefine domain architecture, service contracts, or runtime
behavior. It only governs the repository support layer.

## Canonical Surface Model

| Surface | Canonical role | Canonical status | Primary owner | Typical user | Notes |
|---|---|---|---|---|---|
| `README.md` | Repository orientation | canonical | repository/tooling owner | any contributor | First entrypoint for overall shape |
| `DEVELOPMENT.md` | Daily workflow contract | canonical | repository/tooling owner | contributors | Defines the normal validation loop |
| `Makefile` | Public command surface for repository workflows | canonical | repository/tooling owner | contributors and operators | First choice for validation, stack lifecycle, smoke flows, codegen, migrations |
| `scripts/*.sh` | Harness implementation behind `make` | auxiliary | operational tooling owner | advanced contributors | Direct use is for debugging, extra flags, or harness work |
| `scripts/utils/*` | Shared script internals | private auxiliary | operational tooling owner | script maintainers | Never a user-facing workflow surface |
| `tools/raccoon-cli/` implementation | Support-tool engine | canonical within tooling scope | governance/tooling owner | tooling maintainers | Owns structural analysis and guard-rail logic |
| Direct `raccoon-cli` commands | Expert inspection and governance surface | canonical for expert tooling use, auxiliary for daily workflow | governance/tooling owner | advanced contributors | Prefer direct use for JSON output, narrow analysis, or CLI development |
| `deploy/compose/`, `deploy/envs/`, `deploy/configs/` | Runtime substrate definitions | substrate, not public workflow | deploy/runtime owner | maintainers | Backing artifacts for `make up`, `make compose-config`, and smoke flows |
| Direct `docker compose`, `go`, `cargo` commands | Low-level implementation interfaces | substrate, not canonical workflow | layer-specific owners | maintainers | Use when debugging or evolving those layers directly |
| `docs/operations/` | Canonical support-architecture documentation | canonical | documentation/operations owner | contributors and operators | Explains how repository support surfaces coexist |
| `docs/tooling/` | Tooling-internal reference | canonical within tooling scope | governance/tooling owner | tooling maintainers | Explains what `raccoon-cli` enforces |
| `docs/architecture/` | Binding system architecture and governance | canonical outside support-surface usage | architecture owner | architects and maintainers | Not the default home for support workflow guidance |
| `docs/stages/` | Delivery evidence | historical, non-canonical for current workflow | stage/reporting owner | auditors and maintainers | Historical trace only |

## Entry Point Rules

### Rule 1. Public repository workflows start in `make`

If a workflow is part of normal repository usage, the canonical entrypoint must
be a Make target.

Examples:

- `make check`
- `make tdd`
- `make verify`
- `make up`
- `make smoke`
- `make migrate-up`

### Rule 2. Scripts implement workflows; they do not compete with them

`scripts/*.sh` may expose more flags or lower-level diagnostics than the public
Make target. That does not make the script a competing public API.

Direct script invocation is appropriate when:

- debugging a harness;
- using a flag intentionally omitted from `make`;
- developing the harness itself.

### Rule 3. Direct `raccoon-cli` usage is for expert support work

The CLI is the right direct interface when the task is about:

- structural inspection;
- change analysis;
- machine-readable output;
- evolving or debugging the tooling layer.

It is not the preferred direct entrypoint for runtime/operator flows that
already exist in `make`.

### Rule 4. Raw substrate commands stay below the workflow contract

Direct `docker compose`, `go`, and `cargo` usage is legitimate, but those
surfaces sit below the repository workflow contract.

They should not be documented as first-choice workflows when a stable Make
target already exists.

## Ownership Model

| Concern | Owning surface | Supporting surfaces |
|---|---|---|
| Daily workflow contract | `DEVELOPMENT.md`, `Makefile` | `docs/operations/README.md`, `README.md` |
| Command taxonomy and public entrypoints | `Makefile`, `docs/operations/` | scripts, `raccoon-cli` wrappers |
| Structural governance and analysis | `tools/raccoon-cli/`, `docs/tooling/` | `Makefile` wrappers such as `make check` and `make tdd` |
| Harness behavior and runtime support scripts | `scripts/*.sh` | `Makefile`, `deploy/*`, `docs/operations/scripts-catalog-and-usage-guide.md` |
| Runtime substrate definitions | `deploy/*` | `Makefile`, scripts |
| Historical evidence | `docs/stages/` | stage indexes and reports |

## Decision Table

| If you need to... | Start here | Escalate to... |
|---|---|---|
| run the normal validation loop | `make check`, `make tdd`, `make verify` | direct `raccoon-cli` only if you need narrower tooling analysis |
| bring up or inspect the local stack | `make up`, `make logs`, `make ps`, `make smoke*` | direct scripts for debug flags; raw `docker compose` for substrate debugging |
| inspect architecture or drift | `make check`, `make coverage-map`, `make recommend` | direct `raccoon-cli` grouped commands |
| debug a smoke harness | the corresponding `scripts/*.sh` | raw compose or service logs when the harness is not enough |
| modify `raccoon-cli` behavior | direct `cargo` and direct `raccoon-cli` | `make raccoon-test`, `make check` |
| understand current support-surface policy | this document | [`repository-architecture-convergence.md`](repository-architecture-convergence.md) |

## Non-Goals

- turning `raccoon-cli` into a parallel runtime control plane;
- making scripts the public API when a Make target already exists;
- moving binding runtime or domain rules out of `docs/architecture/`;
- treating stage reports as the current source of truth for support workflows.
