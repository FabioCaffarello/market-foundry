# Legacy documentation

> **This directory is historical material, preserved for archaeology.**
> It is **not** authoritative for the current system. For current
> documentation, go back one level to [`../README.md`](../README.md).

## What is here

The pre-2026-05 documentation set, moved here intact during the
documentation reset (Phase 1A). Subdirectories:

| Directory | What it held |
|---|---|
| `architecture/` | 822 architecture docs accumulated across 500+ stages |
| `stages/` | 539 stage reports — the audit trail of the previous evolution model |
| `archive/` | 316 docs already considered archived in the old taxonomy |
| `operations/` | 1 doc — operational entrypoint of the old structure |
| `tooling/` | 24 docs about `raccoon-cli` internals |
| `product/` | 3 docs about product/runtime overview |
| `development/` | 6 docs about contributor workflow |
| `README-original.md` | The original `docs/README.md` from the previous structure |

Total: ~1712 markdown files, every one with its git history preserved.

## Why this was retired

The previous documentation model grew alongside a stage-based governance
process (each capability added through a charter → execution → evidence
gate → closure cycle). After 500+ stages it accumulated:

- Three competing entry points (`docs/operations/`, `docs/development/`,
  `docs/README.md`).
- Stage-shaped material (charters, gates, closure narratives) mixed with
  durable architecture rules.
- One doc per question replaced by N docs per question with routing
  indexes between them.

The new `docs/` ([`../README.md`](../README.md)) collapses this into
one document per recurring question and uses ADRs ([`../decisions/`](../decisions/README.md))
for durable design decisions instead of stage reports.

## How to use this directory

You should only need to come here when:

- you're reading a git blame and want to understand the historical
  context of a code change;
- you're researching a specific stage report referenced in a commit
  message (e.g., `S474`, `S502`);
- you're looking for the rationale behind a structural decision that
  predates the new `decisions/` directory.

If you find yourself looking here for *current* answers, that is a
signal that the current `docs/` is missing something. File the gap
rather than treating legacy as authoritative.

## What is **not** here

- Code. All code lives in `internal/`, `cmd/`, `tools/`, `deploy/`.
- Stage reports for work done after 2026-05. The new model uses git
  history + PR descriptions, not stage reports.
- Anti-debt and opus-guidance documents are not enforced anymore in
  their original form; their rules survive in
  [`../CONTRIBUTING.md`](../CONTRIBUTING.md) (PR-based equivalent).

## Eventual removal

This directory is preserved through the current cycle of work and may
be removed later once the new `docs/` is mature and all references to
legacy paths have been pruned from the codebase. Until then, treat it
as a read-only museum.
