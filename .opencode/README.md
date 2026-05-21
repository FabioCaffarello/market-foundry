# market-foundry `.opencode`

`.opencode` is the real OpenCode configuration layer for `market-foundry`.
Its job is operational compression: route work quickly, compress recurring
context, and shorten safe handoff between sessions and agents.

It does not replace human owner docs.

Canonical owners:

- `AGENTS.md` and `Makefile` for workflow and command surface
- `README.md` and `DEVELOPMENT.md` for root orientation
- `docs/development/` for contributor workflow, proofs, and repo navigation
- `docs/product/` for system and product identity
- `docs/tooling/` and `tools/raccoon-cli/README.md` for tooling contracts
- `docs/architecture/` for boundaries, governance, and rationale
- `docs/architecture/information-system-governance-and-classification.md` for
  classification and evolution rules across `.opencode` and `docs/`

Use `.opencode` only for:

- navigation and entrypoint choice
- semantic compression of recurring repo context
- short operational handoff between sessions
- safe-change orientation tied to the real workflow
- `raccoon-cli` usage in support of `make`, not instead of it

Approved blocks:

- `repo`
- `runtime`
- `change`
- `intelligence`

No `product` block is opened inside `.opencode` for now.
`docs/product/` already owns that question well enough, and adding a local
mirror here would widen the surface without a distinct operational mission.

Entry navigation:

- `context/navigation.md`
- `context/repo/navigation.md`
- `context/runtime/navigation.md`
- `context/change/navigation.md`
- `context/intelligence/navigation.md`

## Invariants

- `.opencode` is the OpenCode layer for this repo, but not a parallel
  documentation framework.
- The layer stays bounded to one root agent, two profiles, and the four context
  areas `repo`, `runtime`, `change`, and `intelligence`.
- Every context file must stay short, task-shaped, and anchored to real owner
  docs or live repository entrypoints.
- `.opencode` may absorb only navigation, compression, entrypoint choice,
  operational short context, session support, and safe-change guidance.
- Long policy, full runbooks, stage evidence, and architecture rationale remain
  in canonical docs.
- `make` stays the public workflow surface; `raccoon-cli` stays the strategic
  intelligence layer behind that workflow.

## Evolution Rule

Change `.opencode` only when one of these is true:

- a canonical owner doc or repository entrypoint moved
- a recurring task needs a shorter route into already-existing owner content
- safe handoff or safe-change context is repeatedly too expensive to rebuild
- a cheap repository-local check can prevent real `.opencode` drift

Do not grow `.opencode` when the real need is:

- better human explanation in `docs/`
- deeper governance or architecture rationale
- another taxonomy, registry, catalog, or framework layer
- historical narrative that belongs in `docs/stages/` or `docs/archive/`
