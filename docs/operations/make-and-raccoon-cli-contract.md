# Make And Raccoon CLI Contract

## Purpose

This document defines the boundary between the repository's public workflow
surface and its strategic tooling-intelligence surface.

Use it when the question is not "what command exists?" but rather:

- which surface owns this workflow?
- should a need become a Make target or stay a direct `raccoon-cli` command?
- is a change clarifying the contract or opening a parallel entrypoint?

## Core Contract

- `make` owns the canonical public workflow contract.
- `raccoon-cli` owns strategic repository intelligence.
- `scripts/*.sh` own execution detail behind Makefile targets.
- `make smoke*` owns runtime proof-of-record.
- `raccoon-cli` must not become a parallel runtime orchestrator.

## Ownership Matrix

| Need | Canonical surface | Why |
|---|---|---|
| bootstrap, daily loop, verification, runtime bring-up, runtime proof | `make` | stable public workflow, short remembered entrypoints, curated discovery |
| repository inspection, impact analysis, TDD guidance, drift detection, architecture safety, machine-readable output | `raccoon-cli` | narrower expert tooling, read-heavy analysis, auditable structured output |
| smoke harness flags, wait tuning, low-level wrapper behavior | `scripts/*.sh` behind `make` | implementation detail, debugging, environment-specific execution details |
| raw compose/go/cargo behavior | substrate tools | below the repository workflow contract |

## Boundary Rules

### What belongs in `make`

Use or add a Make target when the need is:

- part of the recurring public workflow;
- something contributors should discover from `make help`;
- a stable wrapper over one or more deeper steps;
- a runtime, smoke, lifecycle, or setup operation.

### What belongs in `raccoon-cli`

Keep the capability in `raccoon-cli` when the need is:

- read-only or analysis-heavy;
- best expressed as structural facts, inferred impact, or recommendation output;
- useful in JSON or narrower expert scope;
- supporting the change loop rather than replacing it.

### What should not happen

Do not use `raccoon-cli` to:

- introduce new runtime-proof entrypoints that compete with `make smoke*`;
- orchestrate the live stack as a first-class operator surface;
- duplicate Makefile workflow naming with a second promoted taxonomy;
- expand the CLI just because a script exists or because a command is technically possible.

## Canonical Flows

### Public workflow

```bash
make check
make tdd
# implement the smallest correct change
make verify
make smoke
```

### Expert tooling flow

```bash
raccoon-cli check gate --profile ci --json
raccoon-cli inspect symbol ConfigSet --lsp
raccoon-cli change impact
raccoon-cli change recommend
```

The second flow deepens the first. It does not replace it.

## Overlap Rules

Intentional overlap is allowed only when it clarifies the workflow:

- `make check`, `make tdd`, `make coverage-map`, `make briefing`, and `make recommend`
  are promoted wrappers over `raccoon-cli`;
- `make arch-guard`, `make drift-detect`, and `make quality-gate*`
  exist as public aliases into the CLI's strategic check families;
- direct CLI usage remains canonical for expert narrowing, JSON, and tooling work.

Overlap becomes drift when:

- docs present flat CLI aliases as first-choice commands;
- Make targets invoke legacy flat CLI commands instead of the grouped taxonomy;
- runtime proof starts being described through `quality-gate --profile deep` or
  `legacy runtime-smoke`;
- the same workflow question gets two public first answers.

## UX Taxonomy Guidance

- `make` answers: "what is the official workflow step?"
- `raccoon-cli check` answers: "what repository invariant should I audit?"
- `raccoon-cli inspect` answers: "what structure or contract relationship do I need to inspect?"
- `raccoon-cli change` answers: "what is the blast radius and what should I validate?"

That division keeps the CLI strategic and keeps `make` legible.

## Governance Implications

When evolving either surface:

1. prefer tightening an existing Make wrapper before adding a new public target;
2. prefer grouped `raccoon-cli` commands over flat compatibility aliases;
3. document runtime proof only through `make smoke*`;
4. treat CLI deep/runtime helpers as compatibility paths, not ownership shifts.

## Protected Convergence Contract

The repository now protects one narrow convergence contract across the workflow
owner docs, the `Makefile`, the scripts catalog, and the CLI taxonomy.

Protected invariants:

- the minimal public loop stays the same across owner docs:
  `make bootstrap`, `make check`, `make tdd`, `make verify`, then the relevant
  `make smoke*`;
- `make` wrappers that delegate to `raccoon-cli` keep using the grouped
  taxonomy (`check`, `inspect`, `change`), not flat compatibility aliases;
- `docs/operations/scripts-catalog-and-usage-guide.md` must describe the real
  `make <target>` to `scripts/*.sh` mapping for script-backed public wrappers;
- the CLI public taxonomy remains `check`, `inspect`, `change`, and `legacy`,
  while runtime proof remains owned by `make smoke*`.

This contract is intentionally small. It exists to catch silent drift between
central support surfaces, not to lint wording or freeze every secondary
workflow note.

## Related Documents

- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md)
- [`developer-workflow-unification.md`](developer-workflow-unification.md)
- [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md)
- [`raccoon-cli-command-reference.md`](raccoon-cli-command-reference.md)
- [`../tooling/cli-overview.md`](../tooling/cli-overview.md)
