# Repository Policy And Lightweight Enforcement 2

## Purpose

This document defines the second-generation repository guard rail policy for
`market-foundry`.

Stage C12 closes the support-surface cleanup wave by protecting a small set of
durable repository invariants that are now important enough to enforce
automatically.

The design goal is narrow and practical:

- protect the cleaned repository support surface from obvious structural drift;
- keep checks cheap enough for `make check` and `make verify`;
- avoid turning repository governance into a heavy policy framework.

## Executive Summary

The repository now has a much cleaner public shape than it had before the
C1-C11 sequence:

- `make` is the canonical public workflow surface;
- `scripts/*.sh` are better normalized and intentionally sit behind Make;
- `raccoon-cli` has a clearer support-only taxonomy;
- the documentation system has stable entrypoints and taxonomy rules.

That cleanup creates a new need: not broad policy-as-code, but lightweight
protection against regressions in the support surface itself.

C12 therefore extends the lightweight repository consistency guard rail with a
small second-generation policy set focused on:

- canonical documentation entrypoints;
- minimum docs/index presence;
- consistency between Makefile wrappers and script cataloging;
- basic script-entrypoint hygiene;
- alignment between bootstrap expectations and the governed repository surface;
- preservation of the CLI's public support-only taxonomy.

## Enforcement Philosophy

Only invariants with all of the following properties should be enforced here:

1. they protect a real support-surface contract;
2. they can be checked directly from the worktree;
3. the check is cheap enough for routine local use;
4. the failure implies real contributor or operator friction;
5. the rule does not duplicate architecture or runtime enforcement already
   owned elsewhere.

If a rule becomes noisy, interpretive, or expensive, it does not belong in this
guard rail.

## What C12 Adds

### 1. Documentation entrypoint protection

The guard rail now protects the minimum entrypoint set more explicitly:

- required governance docs must exist;
- each first-level documentation area must keep its expected entrypoint file;
- the new C12 policy documents and stage report are part of the required set.

This protects navigation and avoids slow entropy in the doc system.

### 2. Bootstrap-to-repository alignment

`make bootstrap` is now an established public workflow contract.

That means `scripts/bootstrap-check.sh` should not silently drift away from the
repository's current canonical entrypoints. C12 adds an explicit alignment check
for the governed `required_paths` set.

### 3. Script-entrypoint hygiene

The public shell surface now has enough shape to justify a minimal hygiene bar:

- public shell entrypoints must use the standard bash shebang;
- public shell entrypoints must respond successfully to `--help`.

This is a lightweight discoverability and maintainability rule, not a shell
style framework.

### 4. Makefile/script/doc alignment

The Makefile remains the public workflow surface, but its lower-level shell
wrappers are still part of the support contract. C12 adds alignment checks so
that:

- Makefile script wrappers still resolve to executable files;
- those wrapper scripts remain cataloged in the canonical scripts guide.

This keeps workflow docs honest when new wrappers are added.

### 5. CLI public-surface protection

The repository now depends on the `raccoon-cli` governance model remaining
stable:

- it must remain a repository support CLI;
- grouped taxonomy (`check`, `inspect`, `change`, `legacy`) must remain visible;
- `runtime-smoke` must remain explicitly legacy and must not become a promoted
  public runtime path again;
- canonical CLI docs must stay aligned with that support-only model.

This does not validate CLI business behavior. It protects the public contract.

## Integration Points

The policy is enforced through the existing lightweight path:

```bash
make repo-consistency-check
```

and therefore also through:

```bash
make check
make verify
make check-deep
```

The policy is also reflected in:

- `make docs`
- `make bootstrap`
- `docs/operations/README.md`

## What Remains Recommendation-Only

The following remain deliberate review or documentation concerns, not automated
blocking checks:

- prose quality;
- broad markdown linting;
- full-corpus link validation across the architecture and archive trees;
- subjective CLI wording improvements;
- shell implementation style beyond the basic shebang/help contract;
- whether a document is well argued or complete from a business perspective.

## Explicit Limits

C12 does not:

- introduce a generic policy engine;
- add warning-only noise checks;
- scan the full repository for editorial consistency;
- add runtime behavior checks to the repository consistency layer;
- duplicate `raccoon-cli` architecture, topology, or contract analysis.

## Enforcement Boundary

The repository guard-rail stack now divides responsibilities like this:

- `make repo-consistency-check`: support-surface and repository-policy
  invariants;
- `make quality-gate`: architecture, topology, contract, and structural
  governance through `raccoon-cli`;
- `make test` and `make smoke*`: executable correctness and runtime proof.

That separation is intentional. Repository policy should stay small and highly
credible.

## Related Documents

- [`repository-invariants-check-matrix-and-enforcement-policy.md`](repository-invariants-check-matrix-and-enforcement-policy.md)
- [`lightweight-repository-guard-rails-and-consistency-checks.md`](lightweight-repository-guard-rails-and-consistency-checks.md)
- [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
- [`raccoon-cli-ux-taxonomy-and-guard-rails.md`](raccoon-cli-ux-taxonomy-and-guard-rails.md)
- [`../stages/stage-c12-repository-policy-and-lightweight-enforcement-2-report.md`](../stages/stage-c12-repository-policy-and-lightweight-enforcement-2-report.md)
