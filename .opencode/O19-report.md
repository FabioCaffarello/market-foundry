# O19 Report

## Objective

Consolidate `.opencode` as the real OpenCode layer for `market-foundry` while
keeping it bounded to navigation, semantic compression, handoff, workflow
alignment, runtime orientation, change intelligence, and `raccoon-cli`
integration.

## Legacy Baseline

No physical `opencode.legacy` directory exists in the current workspace.
For this consolidation, the legacy baseline was inferred from:

- the pre-consolidation `.opencode` tree
- the O12-O17 reports
- the editorial pattern already proven there: short files, explicit owner docs,
  bounded topology, and no parallel workflow system

The inheritance is structural and editorial, not a wholesale content import.

## Consolidated Topology

`.opencode` remains bounded to:

- `repo`
- `runtime`
- `change`
- `intelligence`

No `product` block was added.
`docs/product/` already answers that question canonically, and a local mirror
would increase surface area without a distinct operational mission.

## Conceptual Diff Vs Legacy

Before:

- `.opencode` still anchored many routes to the old `docs/operations/*` surface
- workflow, proofs, and tooling ownership reflected the previous docs topology
- the layer compressed navigation well, but not against the newest human owner
  docs

After:

- owner anchors now point to the active surfaces: `docs/development/`,
  `docs/product/`, `docs/tooling/`, and `docs/architecture/`
- `repo`, `runtime`, `change`, and `intelligence` are preserved as the stable
  operating anatomy
- `raccoon-cli` integration is tighter and clearer: strategic intelligence
  behind `make`, not a public workflow competitor
- `.opencode` now states its absorption boundary explicitly instead of only
  implying it

## Context Absorbed From `docs/`

Absorbed into `.opencode` as compressed operational context:

- owner-doc routing from `docs/development/owners.md`
- contributor workflow and proof selection from
  `docs/development/workflow.md` and
  `docs/development/commands-and-proofs.md`
- stage-entry routing from `docs/development/stages-and-governance.md`
- system-owner routing from `docs/product/owners.md`
- tooling boundary and grouped CLI usage from `docs/tooling/cli-overview.md`
- architectural guardrails from
  `docs/architecture/market-foundry-evolution-playbook.md`,
  `docs/architecture/stage-definition-of-done.md`, and
  `docs/architecture/anti-debt-checklist.md`
- runtime support anchors from
  `docs/architecture/current-baseline-runbook.md`

Not absorbed:

- long-form rationale
- full runbooks
- rule catalogs
- stage narratives
- broad architecture explanation

## Explicit Limits

`.opencode` may own only:

- navigation
- semantic compression
- entrypoint choice
- short operational context
- session and agent handoff support
- safe-change orientation

`.opencode` must not become:

- a second human documentation tree
- a framework parallel to `docs/`
- a public command catalog parallel to `Makefile`
- a product or architecture corpus
- a history or stage-evidence surface

## Tradeoffs

- Keeping four blocks preserves navigability and existing muscle memory, but it
  means some topics remain split across adjacent files rather than fully merged.
- Not adding a `product` block keeps the layer tighter, but requires explicit
  routing out to `docs/product/` when identity questions arise.
- Pulling in only compressed context avoids duplication, but places more weight
  on the human docs staying healthy and current.

## Non-Goals

- rewriting canonical human docs inside `.opencode`
- importing generic OpenCode taxonomies with no repository mission
- turning `.opencode` into workflow automation or session memory infrastructure
- replacing `make smoke*`, `make check`, or `make verify`
- replacing `raccoon-cli` documentation with local summaries

## Impacts

- OpenCode entry routing now matches the current human documentation topology
- handoff context is cheaper because owners, workflow, runtime, and change
  intelligence all point to live surfaces
- `.opencode` is more resilient to future drift because its scope is now stated
  in terms of function, not only directory shape
- the consistency check can now validate the new owner anchors instead of the
  retired `docs/operations/*` paths

## Evolution Guidance

- change owner docs first when workflow, proofs, contracts, or governance move
- change `.opencode` only when routing, compression, or handoff must change
- open a new `.opencode` block only if there is a recurring operational mission
  that cannot be covered by the current four-block model
- when a support-surface change touches `Makefile`, `scripts/`, `docs/`, and
  `raccoon-cli`, keep `.opencode` aligned in the same change
