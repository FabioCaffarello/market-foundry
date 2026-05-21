# Commands And Proofs

## Purpose

This file owns the human-facing command surface: which commands are canonical,
which are wrappers, and which proofs to use.

## Canonical Entrypoints

| Need | Command |
|---|---|
| Setup and validate the machine | `make bootstrap` |
| Fast bring-up | `make live` |
| Controlled bring-up | `make up`, `make seed*` |
| Baseline operational proof | `make smoke` |
| Narrower proof surfaces | `make smoke-*` |
| Pre-change guard rail | `make check` |
| Impact-driven guidance | `make tdd` |
| Post-change validation | `make verify` |
| Deep structural validation | `make check-deep` |

## Proof Rule

Choose the narrowest `make smoke*` target that proves the behavior you changed.
`make live*` helps bring the stack up; it does not replace the proof-of-record
surface.

## Tooling Boundary

- Prefer `make` for repository workflows.
- Use direct `raccoon-cli` commands when you need narrower expert inspection,
  JSON output, or you are evolving the CLI itself.
- Use [`../tooling/README.md`](../tooling/README.md) for CLI rule catalogs and
  internals.

## References

- Root workflow: [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- Make target inventory: [`../../Makefile`](../../Makefile)
- Tooling area: [`../tooling/README.md`](../tooling/README.md)
