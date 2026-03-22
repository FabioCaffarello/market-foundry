# Tooling Evolution Patterns And Repository Extension Discipline

## Purpose

This document defines how the support tooling of `market-foundry` should evolve
now that the repository already has enough baseline capability.

The current need is no longer "add more tools". The need is disciplined
extension: deciding which surface should absorb a new need, how far that
surface should expand, and when a request should be rejected or folded into an
existing path.

## Current Growth Pattern

The support surface grew in three visible waves:

1. public workflow consolidation in the root `Makefile`
2. direct harness expansion in `scripts/` for smoke, bring-up, diagnostics, and
   governed checks
3. support-surface convergence in `docs/operations/`, `docs/tooling/`, area
   entrypoints, and the lightweight repository consistency guard rail

This sequence was appropriate:

- the repository first needed working operational entrypoints;
- then it needed wrappers and harnesses for real runtime proof;
- then it needed documentation, navigation, and guard rails to keep those
  surfaces coherent.

The repository is now at the point where undisciplined extension would create
more maintenance cost than user value.

## Good Patterns Already Present

### One public workflow surface

- `make` is the canonical public entrypoint for day-to-day repository usage.
- Prefix families such as `smoke-*`, `live-*`, `codegen-*`, `migrate-*`, and
  `stage-*` make the surface scannable.
- Aliases exist mainly for discoverability and do not replace the canonical
  targets.

### Lower-level harnesses behind the public surface

- `scripts/*.sh` mostly implement harness behavior behind `make`.
- direct script usage is tolerated for debugging, extra flags, and harness
  maintenance rather than being promoted as a second public API.

### Tooling kept in the tooling lane

- `raccoon-cli` owns structural analysis, drift detection, guard rails, and
  change-planning support.
- CLI taxonomy already protects the repository from turning the tool into a
  second operational runtime platform.

### Canonical documentation ownership

- `README.md` stays orientation-focused.
- `DEVELOPMENT.md` owns the daily workflow.
- `docs/operations/README.md` acts as the canonical detailed support index.
- `docs/tooling/` documents tooling internals instead of user workflow.
- area `README.md` files reduce repository-shape search cost.

### Lightweight objective enforcement

- `make repo-consistency-check` protects high-signal support-surface invariants
  without becoming a general policy engine.
- `make stage-status` and `make stage-check` automate continuity and closure
  mechanics while leaving judgment manual.

## Problem Patterns And Accumulation Risks

### Surface inflation through additive convenience

The repository can now add new targets, aliases, scripts, and docs quickly.
That makes opportunistic growth easy:

- one more wrapper;
- one more helper script;
- one more operations guide;
- one more alias for discoverability.

Each addition is cheap once. The maintenance fan-out is not.

### Monolithic harness growth

Some smoke and bring-up scripts already carry multiple concerns. That is
acceptable while they remain canonical proof harnesses, but it is a warning:
new needs should prefer consolidation and parameterization over spawning nearby
scripts that differ only slightly.

### Documentation fragmentation risk

`docs/operations/` now has real value, but it can drift into topic splitting
faster than the repository can absorb. A new support document should exist only
when it creates a durable canonical home for a real rule or model.

### Check accretion risk

The repository consistency pass is useful because it stays small and objective.
If every support-stage concern becomes a permanent required invariant, the
check will become noisy, costly, and less trustworthy.

### CLI creep risk

The CLI is powerful enough that new support needs may look like "another CLI
subcommand". That is often the wrong move when the need is:

- a normal contributor workflow better served by `make`;
- a harness implementation detail better served by a script;
- a one-time policy clarification better served by documentation only.

## Canonical Extension Model

Use the smallest durable surface that solves the real recurring problem.

### Make target

Add or extend a `make` target when all of the following are true:

- the workflow is part of normal repository usage;
- contributors benefit from one short, memorable entrypoint;
- the flow composes existing scripts, tools, or substrate commands into a
  stable user-facing contract;
- the target fits an existing family or clearly deserves a new family.

Do not add a Make target when the need is one-off, debug-only, or still too
volatile to become a public workflow contract.

### Script

Add or extend a script when the need is harness-level:

- lower-level orchestration behind `make`;
- custom flags or narrower debugging behavior;
- shell-heavy operational glue that should not live inline in the `Makefile`;
- reusable implementation support for other scripts.

Do not add a script as a parallel public API when the real user need is already
served by a Make target.

### CLI command

Extend `raccoon-cli` when the need is tooling-specific:

- structural inspection or guard-rail enforcement;
- machine-readable output;
- change-impact analysis or validation guidance;
- reusable repository analysis that is not a runtime/operator flow.

Do not add CLI commands for stack bring-up, smoke orchestration, service
operations, or stage workflow control when those concerns already belong to
`make` and `scripts/`.

### Documentation

Add or extend documentation when the need is:

- a lasting rule, taxonomy, ownership map, or lifecycle model;
- a canonical explanation for why a support surface exists;
- guidance that helps contributors choose between existing surfaces.

Prefer updating an existing canonical document when the rule already has a real
home.

### Lightweight check

Add or extend a lightweight check only when the rule is:

- objective;
- cheap to run locally and routinely;
- directly tied to a canonical support surface;
- likely to catch real drift before review.

Examples:

- missing entrypoints
- dead local links
- missing Make wrapper for a public script
- required canonical docs missing

Do not promote subjective quality rules, one-off stage preferences, or
historical volume into the lightweight guard rail.

### Nothing new

The correct outcome is often to add nothing.

Do not create a new surface when:

- an existing target or doc can absorb the need with a small extension;
- the problem is stage-local and not yet repository-wide;
- the workflow is rare enough that a short note in the current canonical doc is
  sufficient;
- the new entrypoint would exist only as a synonym for an existing one.

## Inclusion And Naming Discipline

### Public workflow naming

- Prefer existing families first: `smoke-*`, `live-*`, `codegen-*`,
  `migrate-*`, `stage-*`, `stack-*`.
- Create a new family only when the repository is clearly adding a new stable
  class of workflows, not a one-off command.
- Aliases must improve discovery, not create ambiguity about which target is
  canonical.

### Script naming

- Name scripts for the workflow or harness they implement, not for a temporary
  stage.
- Keep utility helpers under `scripts/utils/`.
- Avoid near-duplicate scripts that differ only by one flag set or one runtime
  variant.

### Documentation naming

- Prefer descriptive, repository-specific names over generic policy titles.
- Put current support rules in `docs/operations/`.
- Put tooling-internal behavior and rule catalogs in `docs/tooling/`.
- Keep stage reports in `docs/stages/` as evidence, not as the canonical owner
  of ongoing support policy.

## Ownership And Lifecycle Rules

| Surface | Owns | Does not own |
|---|---|---|
| `Makefile` | public workflow contract | low-level harness detail, tooling internals |
| `scripts/` | harness implementation and debug flags | repository-wide policy authority |
| `docs/operations/` | current support rules and workflow selection | tooling internals, immutable evidence |
| `docs/tooling/` | CLI internals, analyzer rules, tooling architecture | normal contributor workflow |
| `docs/stages/` | historical evidence | current canonical support rules |

Every new lasting surface should have:

- one canonical owner;
- one clear audience;
- one reason it exists that is stronger than "convenience".

## Consolidation And Retirement Model

Prefer consolidation before extension.

Consolidate when you see:

- two entrypoints answering the same user question;
- a script and a Make target competing as public entrypoints;
- a guide that mostly restates another canonical guide;
- an alias promoted as if it were a second canonical command family;
- a check protecting historical artifacts rather than present invariants.

Retire or de-emphasize a surface when:

- the replacement is already canonical and documented;
- the surface no longer reduces search time or error rate;
- the maintenance cost is higher than its current value;
- the surface exists only for compatibility and can be clearly labeled as such.

Compatibility support is allowed, but it should be visibly non-canonical.

## Decision Questions Before Extending Tooling

Answer these questions before adding a new tooling surface:

1. What recurring friction is being removed?
2. Why can the existing surface not absorb this need?
3. Which single surface should own the result?
4. Will the change reduce search time, error rate, or manual repetition enough
   to justify its ongoing upkeep?
5. What should stay manual after this change?

If the answers are weak, the repository should prefer documentation-only
clarification or no new surface at all.

## Repository-Specific Discipline For The Next Stages

- Prefer extending existing `smoke-*`, `live-*`, `stage-*`, and
  `repo-consistency-check` paths over creating adjacent helper families.
- Treat large operational scripts as consolidation candidates, not as a reason
  to spawn more sibling scripts.
- Keep `make docs` curated; do not turn it back into a second large index.
- Keep `docs/operations/README.md` as the detailed support catalog.
- Keep `raccoon-cli` in the inspection/governance lane.
- Keep the lightweight consistency pass small, objective, and locally fixable.

## Related Documents

- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
- [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md)
- [`repository-automation-boundaries-high-value-routines-and-sustainability-rules.md`](repository-automation-boundaries-high-value-routines-and-sustainability-rules.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
- [`../tooling/README.md`](../tooling/README.md)
