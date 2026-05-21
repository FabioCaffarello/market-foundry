# Repository Development Workflow

Use this file for the shortest safe change loop.

Canonical owner docs:

- `../../../DEVELOPMENT.md`
- `../../../docs/development/workflow.md`
- `../../../docs/development/commands-and-proofs.md`

Core loop:

- `make check` before coding
- `make tdd` for impact-driven validation guidance
- implement the smallest correct change
- `make verify` after coding
- `make check-deep` only for significant changes

When runtime behavior changed:

- use `make smoke-help` to pick the proof
- run the narrowest relevant `make smoke*`
- diagnose with `make diag`, `make ps`, and `make logs SERVICE=gateway`

Do not promote direct `scripts/*.sh`, raw `docker compose`, or direct
`raccoon-cli` commands as the first answer when `make` already owns the flow.
