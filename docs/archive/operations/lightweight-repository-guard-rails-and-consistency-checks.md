# Lightweight Repository Guard Rails And Consistency Checks

## Purpose

This document describes the lightweight repository consistency guard rail added in
C6 and extended in C12.

The goal is narrow: catch real repository drift in naming, documentation support
surfaces, stage inventory, and Makefile wrappers before that drift turns into
operator confusion or broken workflow guidance.

This is not a general policy platform and not a replacement for `raccoon-cli`.

## Entry Point

Run the guard rail directly with:

```bash
make repo-consistency-check
```

It is also part of the default workflow:

```bash
make check
make verify
make check-deep
```

## What The Check Validates

| Check | Severity | Why it matters |
|---|---|---|
| Required repository documents exist | error | Missing entrypoints or policy docs make the support surface incomplete and harder to navigate |
| Stage report filenames follow `stage-*-report.md` | error | Stable naming keeps stage evidence searchable and predictable |
| Stage reports have minimum document shape | error | A stage report without a title or sections is not useful as durable evidence |
| `docs/stages/INDEX.md` matches the real stage inventory | error | New stage reports should not silently disappear from the historical index |
| Local links in primary support docs resolve | error | Broken links in workflow docs create immediate operational friction |
| Canonical workflow docs reference real Makefile targets | error | Repository entrypoints must not tell contributors to run commands that do not exist |
| Makefile script wrappers point to real executable scripts | error | Broken wrappers create false confidence in the supported command surface |
| `.opencode` core wiring and local references resolve when that surface exists | error | The thin OpenCode layer should fail fast if it points to dead entrypoints |
| `.opencode` context files stay reachable and thin | error | The navigation layer should not accumulate orphan context or duplicate owner docs |
| `.opencode` keeps the approved minimal native topology | error | The Foundry OpenCode layer should stay bounded to one root agent and the four concern areas: `repo`, `runtime`, `change`, and `intelligence` |
| `.opencode` navigation keeps canonical owner anchors and complete per-area routing | error | The layer should keep pointing back to the real owners instead of inventing new local authorities |
| `.opencode` O-reports remain reports instead of owner indexes | error | Change reports should record rationale, not accumulate a second active governance surface |

All automated checks are intentionally high-signal and blocking. Lower-value
style concerns were left out on purpose.

## Scope Of The Link Check

The local-link validation is intentionally limited to the primary support
surfaces:

- `README.md`
- `DEVELOPMENT.md`
- `docs/README.md`
- `docs/operations/*.md`
- `docs/tooling/*.md`
- `docs/architecture/README.md`
- `docs/archive/README.md`
- `docs/stages/INDEX.md`

The check does not scan the full architecture or archive corpus. That would turn
this guard rail into a noisy documentation migration project instead of a fast
consistency pass.

The `.opencode` validation is equally narrow. It checks only the thin
navigation-layer invariants: wiring, reachable local references, explicit
`make` target mentions, prohibited carryovers, the approved minimal topology,
owner-doc anchors, per-area navigation completeness, report-vs-owner-surface
separation, orphaned context, and basic anti-duplication size limits. It does
not attempt to score prose quality or turn `.opencode` into another governance
system.

## Integration With Existing Tooling

The consistency guard rail complements, rather than replaces, the existing
validation stack:

- `make repo-consistency-check` handles lightweight repository invariants.
- `make quality-gate` keeps `raccoon-cli` as the structural and architecture
  enforcement layer.
- `make check` now runs both in sequence.
- `make verify` keeps Go tests in front, then runs the same lightweight
  consistency pass and the fast quality gate.

This split keeps the new checks cheap and repository-focused while preserving the
current architecture-governance model.

## C12 Extension

Stage C12 extends the original guard rail with second-generation checks for:

- minimum docs-area entrypoints;
- public script self-description (`--help` + standard bash shebang);
- bootstrap governed-entrypoint alignment;
- Makefile/script-catalog alignment;
- preservation of the `raccoon-cli` support-only public taxonomy across source
  and canonical docs.

The intent remains the same: cheap, high-signal enforcement only.

## Why These Checks Were Chosen

These invariants were selected because they are:

- cheap to run
- stable over time
- difficult to police reliably in review alone
- directly tied to contributor and operator experience

Checks were not added for subjective prose quality, broad markdown linting,
historical archive cleanup, or domain-level policy expansion.

## Limits

- The guard rail does not validate the full architecture corpus.
- It does not enforce a rigid editorial template for all historical stage
  reports.
- It does not inspect business-domain code or runtime behavior.
- It does not duplicate `raccoon-cli` topology, contract, or arch-layer checks.
- It does not attempt broad policy-as-code expansion.

## Related Documents

- [`repository-policy-and-lightweight-enforcement-2.md`](repository-policy-and-lightweight-enforcement-2.md)
- [`repository-invariants-check-matrix-and-enforcement-policy.md`](repository-invariants-check-matrix-and-enforcement-policy.md)
- [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`../stages/stage-c6-lightweight-repository-guard-rails-and-consistency-checks-report.md`](../stages/stage-c6-lightweight-repository-guard-rails-and-consistency-checks-report.md)
