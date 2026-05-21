# Repository Invariants Check Matrix And Enforcement Policy

## Purpose

This matrix records which repository invariants are currently protected by the
lightweight repository consistency guard rail, how they are enforced, and what
is intentionally left outside blocking automation.

Use it when deciding whether to add, change, or remove a repository-policy
check.

## Selection Policy

Add an invariant here only when all of the following are true:

1. the invariant protects a stable repository support surface;
2. it can be checked cheaply from the worktree;
3. the expected fix is concrete and local;
4. it materially reduces contributor or operator confusion;
5. it does not substantially overlap with `raccoon-cli` or runtime validation.

## Enforced Matrix

| Invariant | Check name | Severity | Why it is enforced |
|---|---|---|---|
| Required repository docs exist | `required-documents` | error | Missing support-surface docs break navigation and governance directly |
| First-level docs areas keep their entrypoints | `docs-area-entrypoints` | error | `docs/*` navigation should not silently lose `README.md` or `INDEX.md` |
| Stage reports keep `stage-*-report.md` naming | `stage-report-naming` | error | Stable historical naming supports traceability and indexing |
| Stage reports keep minimum structure | `stage-report-shape` | error | Historical evidence should remain minimally readable and durable |
| `docs/stages/INDEX.md` matches the stage inventory | `stage-index-alignment` | error | Stage evidence should not fall out of the canonical index |
| Local links resolve in primary support docs | `support-doc-links` | error | Broken links in entrypoint docs create immediate operational friction |
| Canonical docs reference real Makefile targets | `primary-doc-make-targets` | error | Public workflow docs must not publish dead commands |
| Makefile script wrappers resolve to executable scripts | `makefile-script-wrappers` | error | Broken wrappers are real operator failures |
| Public scripts use the bash shebang and support `--help` | `public-scripts-self-describing` | error | Script entrypoints should stay discoverable and predictable |
| Bootstrap required paths include governed repository entrypoints | `bootstrap-entrypoints-alignment` | error | `make bootstrap` must reflect the canonical support surface |
| Makefile-backed scripts remain cataloged in the scripts guide | `makefile-script-catalog-alignment` | error | Public wrappers should remain documented when the surface evolves |
| CLI governance surface remains aligned across source and canonical docs | `cli-governance-surface` | error | The CLI must remain a support tool with the grouped public taxonomy intact |

## Severity Policy

All enforced checks currently use `error` severity.

This is acceptable because the check set remains intentionally small and each
failure indicates a concrete support-surface regression rather than a style
preference.

Warning-level checks are still intentionally excluded.

## Recommended But Not Enforced

The repository currently recommends, but does not block on:

- concise documents with clear purpose sections;
- keeping new docs near existing canonical docs before creating new files;
- preferring grouped `raccoon-cli` commands in new examples;
- keeping script comments and usage text tight and current;
- avoiding public proliferation of new aliases when existing canonical
  entrypoints already fit.

These are real governance expectations, but they still require review
judgment rather than cheap objective automation.

## Explicit Non-Invariants

The following remain outside lightweight enforcement on purpose:

- broad markdown style rules;
- full-archive and full-architecture broken-link scanning;
- deep shell linting;
- whether a script is internally elegant;
- editorial duplication that does not change the support contract;
- domain architecture correctness already enforced by `raccoon-cli`;
- runtime proof, service health, or business behavior.

## Change Policy

### Add a new invariant only when

1. drift has happened more than once or is clearly likely to recur;
2. the invariant can be explained in one sentence;
3. the check can fail with a concrete repair path;
4. the check stays fast enough for `make check`.

### Remove or demote an invariant when

1. it becomes noisy during ordinary work;
2. it duplicates a better enforcement layer;
3. the repository contract changes and the invariant no longer protects the
   right thing.

## Ownership Boundary

| Layer | Primary responsibility |
|---|---|
| `scripts/repository-consistency-check.sh` | Lightweight repository-policy enforcement |
| `scripts/bootstrap-check.sh` | Setup readiness and canonical entrypoint presence |
| `Makefile` | Public workflow surface and discoverability |
| `docs/operations/*` | Canonical policy, usage rules, and support-surface documentation |
| `tools/raccoon-cli/*` | Structural governance and expert tooling behavior |

## Related Documents

- [`repository-policy-and-lightweight-enforcement-2.md`](repository-policy-and-lightweight-enforcement-2.md)
- [`lightweight-repository-guard-rails-and-consistency-checks.md`](lightweight-repository-guard-rails-and-consistency-checks.md)
- [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md)
