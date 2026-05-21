# Repository Sustainability Review Routines And Entropy Control

## Purpose

This document defines the short review routines that keep the `market-foundry`
support surface healthy without turning documentation and tooling governance
into a heavyweight process.

Use it when a change touches active docs, public entrypoints, scripts, or
tooling support surfaces.

## Design Principle

Attach the smallest useful review to the change that already happened.

The repository should resist entropy through brief, local routines that fit the
existing engineering loop:

- before adding a new support surface, decide whether an existing one should
  absorb it;
- when adding an active doc, index it immediately;
- when changing a public flow, review the entrypoints that teach that flow;
- when closing a stage, promote lasting rules and avoid leaving stage-only
  guidance behind.

## Routine Set

### 1. Surface-selection review

Trigger:
a proposed new Make target, script, CLI command, lightweight check, or support
document.

Ask:

1. Which current surface should own this?
2. Can an existing surface absorb it with a small extension?
3. Is this recurring enough to deserve a durable home?
4. Would this create a parallel entrypoint for the same question?

Use:

- `docs/operations/tooling-evolution-patterns-and-repository-extension-discipline.md`
- `docs/operations/tooling-inclusion-deprecation-and-consolidation-rules.md`
- `docs/tooling/raccoon-cli-command-lifecycle-and-deprecation-strategy.md`

Outcome:
extend an existing surface, create one justified new surface, or do nothing.

### 2. Active-doc indexing review

Trigger:
a new active document under `docs/operations/` or `docs/tooling/`, or a stage
promotion into one of those directories.

Check:

- is the doc linked from the owning README?
- does the title describe a durable concern rather than a stage-local activity?
- is a root-doc update actually needed, or is the owning index enough?

Outcome:
no active doc should be left discoverable only by filename or search.

### 3. Entrypoint coherence review

Trigger:
changes to `README.md`, `DEVELOPMENT.md`, `docs/README.md`, `Makefile`, or area
README files.

Check:

- did the change alter contributor orientation materially?
- are root docs still shallow and curated?
- is `make` still the canonical public workflow entrypoint?
- do area READMEs still explain local ownership without duplicating broader
  policy?

Outcome:
entrypoints remain useful navigation aids instead of broad secondary catalogs.

### 4. Script and wrapper review

Trigger:
new script, new public wrapper, or a script that becomes routine in practice.

Check:

- does the script have a clear owning public surface?
- is there already a Make target for the user-facing workflow?
- is the script documented in `scripts/README.md` and the operations catalog if
  it is an active harness?
- could the new behavior be a flag or mode on an existing script?

Outcome:
script growth stays harness-oriented rather than becoming a parallel workflow
surface.

### 5. Lightweight-check admission review

Trigger:
proposal to extend `make repo-consistency-check` or another routine guard rail.

Check:

- is the invariant binary and objective?
- will silent drift happen without the check?
- is the failure locally understandable and cheap to fix?
- is the invariant attached to an active canonical surface rather than
  historical completeness?

Outcome:
lightweight checks stay trusted because they remain small and high-signal.

### 6. Stage-closure promotion review

Trigger:
closing a governed support or documentation stage.

Check:

- what lasting rule was established?
- which canonical doc now owns that rule?
- was the owning README updated?
- did the stage report remain evidence rather than current policy?

Outcome:
future contributors read the current rule from an active document, not from
historical execution evidence.

## Review Cadence

This repository does not need a separate recurring governance meeting.

Use this cadence instead:

- per change: run the relevant local review from this document;
- per governed stage: run the stage-closure promotion review;
- per support-heavy wave: do one hotspot pass before declaring closure, focused
  on docs, tooling, entrypoints, and low-frequency helpers added during that
  wave.

That cadence is enough because the repository already has frequent touch points
through `make check`, `make verify`, stage reports, and canonical docs.

This cadence remains the local and continuous review layer.
Use
[`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md)
when a broader periodic strategic pass is needed across several support
surfaces, and use
[`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
to decide whether recurring signals justify that broader pass.

## Entropy Watchlist

Treat these as recurring warning signals:

- a new active doc exists but is not in `docs/operations/README.md` or
  `docs/tooling/README.md`;
- a root doc grows because it is trying to mirror the operations index;
- a script path is taught as a normal workflow even though a Make target exists;
- more than one public command/doc is now answering the same repository
  question;
- a check proposal is justified mainly by "we touched it in this stage";
- a stage report is the only place where a lasting operating rule was written.

When one of these appears, fix the owning surface immediately instead of
deferring it to a later cleanup wave.

## Cheap Control Actions

Use the smallest corrective move that restores clarity:

1. add the missing doc to the owning README;
2. remove duplicate guidance from a root or bridge doc;
3. route users back to the canonical Make target or canonical doc;
4. merge or narrow overlapping docs;
5. reject a new check and document the rule instead;
6. move a lasting rule from a stage report into the right active document.

## Verification Expectations

When a C24-style sustainability change lands, the normal proof is usually:

- `make repo-consistency-check`
- `make stage-check STAGE_ID=... STAGE_SLUG=...`

Escalate to broader verification only when the change also touched runtime or
tooling behavior beyond documentation and lightweight guard rails.

## Limits

These routines are intentionally light.

They do not require:

- ownership rosters for every doc;
- mandatory reviewers by topic;
- a formal ticket for small documentation maintenance;
- separate sustainability dashboards;
- exhaustive audits of the whole repository on every change.

If a future routine cannot fit inside normal repository work, it should be
reduced or rejected.

## Related Documents

- [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md)
- [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
- [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
