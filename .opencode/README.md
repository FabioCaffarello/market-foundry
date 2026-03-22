# market-foundry `.opencode`

`.opencode` is a short navigation layer for real engineering work in this repo.
It compresses entrypoints; it does not replace the owner docs.

Canonical owners:

- `AGENTS.md` and `Makefile` for workflow and command surface
- `README.md` and `DEVELOPMENT.md` for orientation and daily loop
- `docs/operations/`, `docs/tooling/`, and `docs/architecture/` for support and governance

Use this layer only to answer quickly:

- where to start the local stack and prove behavior
- how to change safely and validate the touched path
- when to use `make`, direct scripts, or `raccoon-cli`

Entry navigation:

- `context/navigation.md`
- `context/runtime/navigation.md`
- `context/change/navigation.md`
- `context/intelligence/navigation.md`

Guard rails:

- keep files short and task-shaped
- point to real entrypoints and owner docs
- avoid copying long command catalogs or architecture prose

## Lightweight Invariants

- `.opencode` is a navigation layer, not a second owner-doc tree.
- Canonical ownership remains in `AGENTS.md`, `Makefile`, `README.md`,
  `DEVELOPMENT.md`, `docs/operations/`, `docs/tooling/`, and
  `docs/architecture/`.
- The approved topology stays bounded to one root agent, two profiles, and the
  four context areas `repo`, `runtime`, `change`, and `intelligence`.
- Every context area must keep explicit links to canonical owner docs and local
  navigation to all of its approved leaf files.
- Every local path and `make` reference must resolve against the live codebase.
- `.opencode` must not revive removed surfaces, publish stale workflow
  commands, or grow durable owner catalogs, stage history, or parallel command
  taxonomies.

## Evolution Rule

Change `.opencode` only when one of these is true:

- a canonical entrypoint moved and navigation would otherwise break;
- a recurring repository task needs a shorter route that the owner docs already
  define clearly;
- a real drift pattern escaped review more than once and a cheap objective
  check can catch it.

Do not grow `.opencode` when the real need is:

- updating canonical policy, runbooks, or architecture rationale;
- adding another owner catalog or subject map;
- documenting historical evidence better in `docs/stages/`;
- compensating for weak owner docs instead of fixing those owner docs.
