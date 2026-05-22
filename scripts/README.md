# Script Entry Surface

`scripts/` contains the executable harnesses behind the public `make` surface.

This directory is for debugging and support work, not for replacing the
repository workflow contract.

## Operating Rule

- Prefer `make` first.
- Use scripts directly when you need narrower debugging, flags, or to maintain
  the harness itself.

## Script Map

| Script | Primary owner surface | Use it for |
|---|---|---|
| `bootstrap-check.sh` | `make bootstrap` | local environment validation |
| `repository-consistency-check.sh` | `make repo-consistency-check` | support-doc and repository guard rails |
| `lint-go.sh` | `make lint-go` | golangci-lint across all workspace modules |
| `validate-commit-msg.sh` | lefthook `commit-msg` hook | conventional commit format validation (see `lefthook.yml`) |
| `live-pipeline-activate.sh` | `make live`, `make live-check` | ergonomic stack bring-up and validation |
| `seed-configctl.sh` | `make seed`, `make seed-multi` | config seeding helpers |
| `smoke-*.sh` | `make smoke*` | operational proofs and targeted runtime validation |
| `smoke-activation.sh` | `make smoke-activation` | activation acceptance proof against the control surface |
| `diag-check.sh` | `make diag` | first-line diagnosis summary |
| `codegen-*.sh` | `make codegen-*` | codegen integrity and equivalence checks |
| `utils/` | internal helper layer | shared shell helpers used by other scripts |

## Maintenance Rules

- Every user-facing script should remain reachable from a documented Makefile target.
- Add a new script only when the behavior is harness-level, debug-oriented, or an implementation detail behind a canonical public surface.
- If a script becomes routine for normal contributors, promote or fold it into an existing `make` family instead of teaching the script path as a second public API.
- If two scripts differ mainly by waits, narrow flags, or adjacent variants of
  the same proof, treat that as a consolidation signal before adding another
  sibling harness.
- If a script remains compatibility-only, debugging-only, or wrapper-only,
  label that status clearly in docs rather than letting it look canonical by
  accident.
- Keep this file short; canonical contributor guidance belongs in
  [`../docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md).
- If a script becomes a public workflow entrypoint, update this file, the
  Makefile help text, and the relevant operations doc in the same change.
