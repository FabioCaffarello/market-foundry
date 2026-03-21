# Makefile Targets Reference And Conventions

## Canonical Workflow

For normal development, the repository standard remains:

```bash
make bootstrap
make check
make tdd
# implement the smallest correct change
make verify
```

`make check` and `make verify` both start with `make repo-consistency-check`, then
delegate to the existing raccoon quality-gate flow. Use `make check-deep` only
for broader or riskier changes.
Use `make live` for the fastest official bring-up path and `make up` +
`make seed*` when you need controlled manual bring-up.

## Surface Hierarchy

- `make` owns the stable public workflow contract for repository support.
- `scripts/*.sh` are lower-level harnesses that sit behind `make`; invoke them directly only for debugging, custom waits, or harness work.
- Direct `raccoon-cli` usage is canonical for expert inspection and tooling governance, but it is not the primary runtime/operator surface.
- Raw `docker compose`, `go`, and `cargo` commands remain substrate interfaces and should not be documented as competing first-choice repository workflows when a Make target already exists.
- Module-aware Go targets now fail the Make invocation on the first failing module instead of allowing a later successful module to mask the failure.

## Naming Conventions

The root `Makefile` now follows these conventions:

- Existing established targets remain canonical when already widely referenced in docs or stages.
- Aliases are added only for discoverability, not to replace the canonical target.
- The `smoke*` family is the proof-of-record operational surface.
- The `live*` and `stack-*` families are ergonomic wrappers and aliases, not competing proof surfaces.
- Operational families stay prefix-based:
  - `smoke-*`
  - `live-*`
  - `codegen-*`
  - `migrate-*`
  - `quality-gate*`
- Stack-oriented aliases use the `stack-*` prefix and map to existing compose-facing targets.

## Discoverability Aliases

| Alias | Canonical Target | Intent |
|---|---|---|
| `make lint` | `make check` | Match common contributor expectation for a fast static gate |
| `make test-unit` | `make test` | Make unit-oriented intent more obvious |
| `make stack-up` | `make up` | Make runtime stack lifecycle easier to scan |
| `make stack-down` | `make down` | Make runtime stack lifecycle easier to scan |
| `make stack-restart` | `make restart` | Make runtime stack lifecycle easier to scan |
| `make stack-logs` | `make logs` | Make runtime stack lifecycle easier to scan |

These aliases are additive. Existing docs and scripts may continue using the canonical names.
They must not be documented as replacing the canonical `up`/`down`/`logs` or `smoke*` targets.

## Primary Targets

### Help And Docs

| Target | Purpose |
|---|---|
| `make help` | Show grouped targets and common variables |
| `make docs` | Print the primary workflow and tooling docs |
| `make bootstrap` | Validate local prerequisites and canonical repository entrypoints |

### Core Workflow

| Target | Purpose |
|---|---|
| `make check` | Fast pre-change guard rail: repo consistency + quality gate |
| `make lint` | Alias for `make check` |
| `make repo-consistency-check` | Lightweight repository consistency checks |
| `make stage-help` | Show the stage helper usage and supported inputs |
| `make stage-scaffold` | Scaffold a stage report for a governed stage |
| `make stage-check` | Validate one active stage and optional required artifacts |
| `make tdd` | Impact-driven validation guide |
| `make verify` | Post-change Go tests plus repo consistency and fast quality gate |
| `make check-deep` | Repo consistency plus deep validation profile |

`make check-deep` is a deeper tooling gate, not a replacement for `make smoke*`.

### Go And Test

| Target | Purpose |
|---|---|
| `make tidy` | Run `go mod tidy` in all workspace modules |
| `make test` | Run `go test ./...` in all workspace modules |
| `make test-unit` | Alias for `make test` |
| `make test-integration` | Run tests tagged `integration` |
| `make test-clickhouse` | Run tests tagged `requireclickhouse` when `CLICKHOUSE_DSN` is set |
| `make test-behavioral` | Run charter-protected behavioral tests |
| `make test-behavioral-roundtrip` | Run behavioral round-trip writer pipeline tests |
| `make build` | Build local binaries |
| `make docker-build` | Build compose-backed service images |
| `make clean` | Remove `bin/` and Go caches |

### Runtime Stack

| Target | Purpose |
|---|---|
| `make compose-config` | Validate the compose file |
| `make up` | Start the full stack and apply migrations |
| `make down` | Stop the stack |
| `make restart` | Restart stack or one runtime service |
| `make logs` | Stream stack or service logs |
| `make ps` | Show compose status |
| `make live` | Full single-symbol activation |
| `make live-check` | Validate a running single-symbol stack |
| `make live-multi` | Full multi-symbol activation |
| `make live-multi-check` | Validate a running multi-symbol stack |
| `make smoke-help` | Show proof selection, prerequisites, and common diagnosis commands |
| `make seed` | Seed single-symbol config |
| `make seed-multi` | Seed multi-symbol config |
| `make smoke` | First-slice smoke |
| `make smoke-multi` | Multi-symbol smoke |
| `make smoke-analytical` | Analytical path proof |
| `make smoke-round-trip` | Full persistence round-trip proof |
| `make smoke-live-stack` | Live stack and gateway verification proof |
| `make smoke-operational` | OS-process operational smoke |
| `make smoke-restart-recovery` | Compose-level restart and recovery smoke |
| `make diag` | Diagnostic snapshot |

Operational proof rule:
`make smoke*` owns runtime proof. `make live*` may orchestrate startup and then
delegate into those proofs, but it is not the canonical proof-of-record surface.

### Architecture And Analysis

| Target | Purpose |
|---|---|
| `make arch-guard` | Enforce layer boundaries |
| `make drift-detect` | Detect structural drift |
| `make coverage-map` | Show coverage and gap map |
| `make snapshot` | Generate code-intelligence snapshot |
| `make snapshot-diff` | Compare two snapshots |
| `make baseline-drift` | Detect drift from baseline |
| `make briefing` | Generate raccoon briefing |
| `make recommend` | Generate recommendations from current diff or supplied targets |

### Raccoon CLI

| Target | Purpose |
|---|---|
| `make raccoon-build` | Build the Rust CLI |
| `make raccoon-test` | Run Rust CLI tests |
| `make quality-gate` | Run fast quality gate profile |
| `make quality-gate-ci` | Run CI profile with JSON output |
| `make quality-gate-deep` | Run deep profile |

### Codegen

| Target | Purpose |
|---|---|
| `make codegen-check` | Golden snapshot validation |
| `make codegen-test` | Codegen unit tests |
| `make codegen-integrated` | Integrated slice validation |
| `make codegen-equivalence` | Cross-artifact equivalence wrapper |
| `make codegen-validate-all` | Per-spec and cross-spec validation |
| `make codegen-status` | Governance status report |

### Migrations

| Target | Purpose |
|---|---|
| `make migrate-up` | Apply pending ClickHouse migrations |
| `make migrate-status` | Show migration status |
| `make migrate-validate` | Verify migration checksums |

## Common Variables

| Variable | Applies To | Meaning |
|---|---|---|
| `MODULE=./path` | `tidy`, `test`, `test-integration`, `test-clickhouse` | Restrict Go module-aware commands to one module |
| `SERVICE=name` | `build`, `docker-build`, `logs`, `restart` | Restrict service-aware commands to one supported service |
| `TARGETS=a,b` | `briefing`, `recommend` | Pass explicit paths or targets to raccoon |
| `SNAP1=file SNAP2=file` | `snapshot-diff` | Snapshot diff inputs |
| `BASELINE=file` | `baseline-drift` | Baseline snapshot input |
| `STAGE_ID=C15` | `stage-scaffold`, `stage-check` | Stage identifier |
| `STAGE_SLUG=name` | `stage-scaffold`, `stage-check` | Kebab-case report slug |
| `STAGE_TITLE=Title` | `stage-scaffold` | Report title for scaffolded stage report |
| `STAGE_REQUIRE=path1,path2` | `stage-check` | Extra artifacts that must exist for the stage |
| `BASE_URL=http://...` | `smoke-help`, `smoke*` | Override the gateway base URL used by smoke scripts |
| `SMOKE_WAIT=180` | `smoke*` | Override smoke wait/flush timeout without direct script invocation |
| `FLUSH_WAIT=180` | `smoke-restart-recovery` | Override post-restart flush wait |

## Service Scope Rules

To reduce ambiguity, `SERVICE=...` is validated against the target's actual support surface:

- `make build` accepts buildable binaries, including `migrate`
- `make docker-build` accepts only compose-backed image services
- `make logs` and `make restart` accept only runtime services present in compose

This is intentional hardening. The goal is to fail fast on invalid combinations instead of delegating a typo or unsupported target to Docker Compose.
