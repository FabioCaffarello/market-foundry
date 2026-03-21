# Repository Consistency Invariants And Check Policy

## Purpose

This document defines which lightweight repository invariants are protected by
the C6 consistency check and its C12 extension, and why they are enforced at
blocking severity.

The policy is intentionally conservative. If an invariant is not cheap,
objective, and broadly useful, it should not be automated here.

## Selection Policy

An invariant belongs in the lightweight repository consistency check only when
all of the following are true:

1. It protects a real repository support surface.
2. It can be evaluated cheaply from the worktree.
3. It has low interpretive ambiguity.
4. A failure would create concrete contributor or operator friction.
5. It does not overlap heavily with existing `raccoon-cli` enforcement.

If any of these conditions is false, the invariant should stay in review,
architecture governance, or future specialized tooling.

## Enforced Invariants

| Invariant | Severity | Policy rationale |
|---|---|---|
| Required repository entrypoint docs must exist | error | Missing entrypoints break the repository support map immediately |
| Stage report files must remain lowercase kebab-case under `stage-*-report.md` | error | Stable historical naming reduces drift and supports traceability |
| Stage reports must contain a top-level title and at least two level-2 sections | error | This is the minimum useful completeness bar without imposing a rigid template |
| Every stage report must be represented in `docs/stages/INDEX.md` | error | The index is the canonical navigation surface for stage evidence |
| Local links in primary support docs must resolve | error | Broken local references directly damage usability of operational documentation |
| Canonical workflow docs must reference real Makefile targets | error | Workflow docs are entrypoints and should never publish dead commands |
| Every Makefile wrapper to `./scripts/*.sh` must resolve to an executable file | error | Wrapper drift is a real operational failure, not a style issue |

## Severity Policy

Current automated checks are all `error` severity.

This is deliberate:

- the check set is small
- each invariant is high-value
- each failure indicates real drift, not preference

Warning-level checks were intentionally avoided because they tend to accumulate
noise faster than value in a support-surface guard rail.

## Explicit Non-Invariants

The following concerns are out of scope for this check:

- prose quality
- broad markdown style linting
- full-doc corpus broken-link scanning
- archive cleanup
- historical terminology normalization
- domain or architecture policy enforcement already covered by `raccoon-cli`
- review-only judgments such as whether a document is persuasive or complete in
  a business sense

## Change Policy

Add a new invariant only when:

1. there is repeated evidence of real drift;
2. the rule can be defined in one sentence;
3. a failure message can point to a concrete fix action; and
4. the check stays fast enough to remain part of `make check`.

Remove or demote an invariant when it becomes noisy, redundant, or too coupled
to ongoing repository churn.

## Relationship To Other Guard Rails

- `make repo-consistency-check` protects lightweight repository consistency.
- `make quality-gate` protects architecture, topology, and contract governance.
- `make test` protects executable behavior.
- Stage reports and architecture docs remain the place for rationale, not for
  replacing automated checks.

## Related Documents

- [`repository-policy-and-lightweight-enforcement-2.md`](repository-policy-and-lightweight-enforcement-2.md)
- [`repository-invariants-check-matrix-and-enforcement-policy.md`](repository-invariants-check-matrix-and-enforcement-policy.md)
- [`lightweight-repository-guard-rails-and-consistency-checks.md`](lightweight-repository-guard-rails-and-consistency-checks.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`../architecture/monorepo-documentation-and-stage-governance.md`](../architecture/monorepo-documentation-and-stage-governance.md)
