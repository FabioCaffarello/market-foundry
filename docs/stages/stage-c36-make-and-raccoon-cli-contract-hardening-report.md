# Stage C36 Report: Make And Raccoon CLI Contract Hardening

## Summary

Stage C36 hardened the contract between `make` and `tools/raccoon-cli` so the
CLI evolves as a strategic intelligence layer for repository development,
without drifting into a parallel runtime/operator surface.

The change stayed intentionally proportional:

- clarify ownership and overlap instead of opening new workflow families;
- promote grouped CLI taxonomy where Make wrappers already exist;
- keep `make smoke*` as the only runtime proof-of-record surface;
- tighten docs, help text, and analyzer output so contributors get one answer
  to "where do I start?" and a different, explicit answer to "how do I inspect
  or plan this change deeply?".

## Diagnosis

The repository already had the right strategic intent, but the public and
expert surfaces were still leaking into each other.

The main friction points were:

- Make targets such as `make tdd`, `make arch-guard`, `make drift-detect`,
  `make quality-gate*`, `make briefing`, and `make recommend` wrapped the CLI
  but still invoked flat compatibility aliases instead of the grouped taxonomy;
- several analyzer outputs and remediation hints still recommended legacy flat
  CLI names, which weakened the grouped UX promoted in C4;
- the operational boundary existed in multiple docs, but there was no short
  owner doc explicitly saying "make owns workflow; raccoon-cli owns strategic
  intelligence";
- this made `raccoon-cli` look closer to a second command surface than to a
  focused inspection and analysis layer.

This was primarily a boundary and discoverability problem, not a capability gap.

## Scope Boundaries

### In scope

- clarify the contract between public Make workflows and direct `raccoon-cli`
  usage;
- align promoted Make wrappers with the grouped CLI taxonomy;
- update help text, core docs, and stage/governance references accordingly;
- keep the change proportional to workflow clarity rather than feature expansion.

### Out of scope

- creating new runtime-proof or stack-orchestration entrypoints;
- rewriting the full long tail of historical tooling/reference docs;
- removing compatibility aliases from the CLI binary;
- changing smoke harness behavior or broadening the `make smoke*` family.

## Boundaries And Overlaps

### Stable boundary after C36

- `make` owns the canonical public workflow contract.
- `raccoon-cli` owns inspection, impact analysis, TDD guidance, drift
  detection, architecture safety, and machine-readable strategic output.
- `scripts/*.sh` remain execution detail behind Make targets.
- `make smoke*` remains the only runtime proof-of-record surface.

### Intentional overlap retained

- `make check`, `make tdd`, `make coverage-map`, `make briefing`, and
  `make recommend` remain public wrappers over raccoon intelligence.
- `make arch-guard`, `make drift-detect`, and `make quality-gate*` remain
  stable Make entrypoints even though they delegate to grouped CLI commands.

### Overlap removed or reduced

- Make wrappers no longer reinforce flat CLI aliases as the promoted internal
  invocation shape.
- core CLI help and analyzer output now prefer grouped command names.
- governance wording now states directly that deep/legacy CLI runtime helpers do
  not own operational proof.

## Taxonomy And UX Refinement

### Refinement 1. Keep one public workflow taxonomy

Public workflow answers continue to live in `make`.

`raccoon-cli` now reads more clearly as a second-level expert taxonomy:

- `check` for audits and guard rails;
- `inspect` for structural read-only analysis;
- `change` for blast radius and validation planning.

### Refinement 2. Treat flat CLI names as compatibility only

The grouped taxonomy remains canonical. Flat commands still work, but they are
not the names reinforced by wrappers, new examples, or remediation hints.

### Refinement 3. Make the runtime boundary explicit in help and docs

The CLI root help now states the contract with `make`, and the operations docs
now have a short owner doc dedicated to that contract.

## Changes Applied

### Code and help surface

- added `tools/raccoon-cli/src/command_refs.rs` to centralize canonical grouped
  command strings and reduce future wording drift across analyzers/help;
- updated CLI root help to describe `raccoon-cli` as the repository intelligence
  layer and to state the contract with `make`;
- updated grouped-command examples to prefer canonical grouped invocations;
- updated analyzer output and gate remediation hints to recommend grouped
  commands such as `raccoon-cli check gate`, `raccoon-cli check arch`,
  `raccoon-cli inspect coverage`, and `raccoon-cli change recommend`.

### Makefile contract hardening

- updated Make wrappers to invoke grouped CLI taxonomy directly:
  - `make tdd` -> `raccoon-cli change tdd`
  - `make coverage-map` -> `raccoon-cli inspect coverage`
  - `make briefing` -> `raccoon-cli change briefing`
  - `make recommend` -> `raccoon-cli change recommend`
  - `make arch-guard` -> `raccoon-cli check arch`
  - `make drift-detect` -> `raccoon-cli check drift`
  - `make quality-gate*` -> `raccoon-cli check gate ...`
- clarified target descriptions so these wrappers read as promoted strategic
  analysis helpers, not as a second operational platform.

### Governance and documentation

- added `docs/operations/make-and-raccoon-cli-contract.md`;
- updated `docs/operations/README.md`, `docs/tooling/README.md`,
  `docs/tooling/cli-overview.md`, and `docs/operations/raccoon-cli-command-reference.md`
  to point to the new contract and reinforce the surface split;
- updated root/workflow docs and bootstrap/consistency entrypoints so the new
  contract doc is treated as part of the canonical support surface;
- added this stage to `docs/stages/INDEX.md`.

## Validation

- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml`
- `make repo-consistency-check`
- `make stage-check STAGE_ID=C36 STAGE_SLUG=make-and-raccoon-cli-contract-hardening STAGE_REQUIRE=docs/operations/make-and-raccoon-cli-contract.md,docs/stages/stage-c36-make-and-raccoon-cli-contract-hardening-report.md`

## Risks

- many older reference docs still mention flat CLI names because they describe
  analyzer internals or historical governance; C36 did not rewrite that entire
  long tail;
- hidden compatibility aliases remain in the binary, so contributors can still
  see older usage in local scripts or historical notes;
- `quality-gate`, `arch-guard`, and related Make target names remain as public
  compatibility/convenience entrypoints, which is intentional but still a form
  of bounded overlap.

## Next Steps

1. When editing analyzer- or domain-specific tooling docs, convert examples to
   grouped CLI commands opportunistically instead of leaving flat aliases as the
   default examples.
2. If future workflow pressure appears, prefer tightening existing Make wrappers
   before adding new public targets.
3. If a capability starts needing stack orchestration or runtime proof
   ownership, route it to `make smoke*` or a Make-controlled flow instead of
   expanding `raccoon-cli`.

## Preparation For Next Stage

1. Opportunistically migrate remaining analyzer- and domain-specific docs from
   flat CLI aliases to grouped commands when those files are already being
   touched for substantive reasons.
2. If another workflow hotspot opens, use the new contract doc first to decide
   whether the answer belongs in `make`, in `raccoon-cli`, or should stay a
   lower-level script concern.
3. If compatibility aliases ever become a maintenance burden, evaluate them
   under the existing command lifecycle and deprecation rules instead of
   removing them ad hoc.
