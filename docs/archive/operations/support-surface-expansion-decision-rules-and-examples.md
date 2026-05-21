# Support-Surface Expansion Decision Rules And Examples

## Purpose

This document applies the C31 decision model to real repository patterns so
contributors can judge future expansion requests with concrete examples rather
than abstract governance language.

Use it with:

- [`criteria-for-opening-containing-or-rejecting-new-support-surfaces.md`](criteria-for-opening-containing-or-rejecting-new-support-surfaces.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
- [`support-surface-lifecycle-signals-and-consolidation-criteria.md`](support-surface-lifecycle-signals-and-consolidation-criteria.md)

## Decision Rules

### Rule 1. Open a new surface only when a recurring repository job lacks a credible owner

Good signal:
contributors repeatedly need to do something important and current surfaces do
not offer a coherent place for it.

Repository example:

- `make stage-status` was worth opening because continuity between
  `stage-scaffold` and `stage-check` was a real recurring gap.
- The result stayed narrow, advisory, and attached to the existing `stage-*`
  family rather than becoming a workflow engine.

Decision:
open.

### Rule 2. If the owner already exists, extend that owner instead of opening a sibling

Good signal:
the need is real, but it clearly belongs inside an existing family.

Repository examples:

- `make smoke-restart-recovery` and `make codegen-equivalence` were promoted
  into already-existing public families instead of creating new top-level
  categories.
- `lint` remained an alias to `check`, improving discovery without challenging
  canonical ownership.

Decision:
contain inside the existing owner.

### Rule 3. If the problem is ambiguity, solve it with labeling and docs before adding execution surface

Good signal:
contributors are confused about which surface is canonical, but the capability
already exists.

Repository examples:

- the CLI lifecycle work in C23 contained `runtime-smoke` as legacy instead of
  promoting it as another current proof path;
- repeated docs clarified that `scripts/*.sh` are harness/debug surfaces behind
  `make`, not an alternative public API;
- stage-governance docs repeatedly moved active rules out of stage reports and
  into `docs/operations/`.

Decision:
docs/convention only, not a new surface.

### Rule 4. If two surfaces already answer the same question, consolidate before expanding

Good signal:
the request appears to ask for something new, but the real issue is overlap.

Repository examples:

- operations docs were created to stop active workflow rules from being spread
  across architecture history and stage evidence;
- grouped CLI taxonomy replaced the risk of a flat command catalog growing as a
  parallel surface;
- curated root docs and `make docs` were used to point into canonical owners
  instead of letting every support document compete as a starting point.

Decision:
consolidate.

### Rule 5. Reject additions that exist mainly to preserve convenience at creation time

Bad signal:
the proposal sounds easy because it pushes cost into future maintenance.

Repository examples:

- adding a new script only because an existing harness is large is not enough;
- adding a new lightweight invariant for a stage-local concern would make
  `repo-consistency-check` noisier without protecting a stable canonical asset;
- adding a second public entrypoint for a current workflow would reduce
  canonical clarity.

Decision:
reject.

## Worked Examples By Surface

## CLI

### Example A. New CLI surface that added value

Case:
grouped CLI taxonomy and lifecycle clarification.

Why it worked:

- the CLI already owned structural analysis and governance;
- the change improved trust and discoverability inside that owner;
- it contained compatibility helpers instead of widening operational scope.

Correct interpretation:
contain and govern inside the CLI owner.

### Example B. CLI surface that should be contained or rejected

Case:
operational runtime helpers such as `runtime-smoke`.

Why containment won:

- runtime proof ownership already belongs to `make smoke*` plus harness scripts;
- promoting runtime orchestration in the CLI would create a competing workflow
  surface;
- the value was compatibility, not canonical ownership.

Correct interpretation:
legacy containment, not expansion.

## Make targets

### Example A. New Make target that was justified

Case:
`make stage-status`.

Why it worked:

- it filled a recurring workflow gap;
- it belonged naturally in an existing family;
- it reduced memory burden without automating judgment.

Correct interpretation:
open inside an existing family.

### Example B. Make additions that were intentionally lightweight

Case:
`lint`, `test-unit`, `stack-up`, `stack-down`, `stack-restart`, `stack-logs`.

Why they were acceptable:

- they improved familiarity and discoverability;
- they did not create new ownership or claim new canonical flows;
- the canonical targets remained unchanged.

Correct interpretation:
contain as aliases, not as new primary surfaces.

### Example C. Make expansion that should usually be rejected

Case:
a new top-level family for a one-off proof or a narrow debug task.

Why rejection is usually right:

- it would expand the public contract for a non-recurring need;
- the behavior belongs in scripts or direct expert usage;
- a new family increases help, docs, and indexing burden immediately.

Correct interpretation:
reject or keep as direct debug support.

## Scripts and wrappers

### Example A. Hidden but real flows that deserved promotion

Case:
restart/recovery and codegen equivalence wrappers.

Why promotion worked:

- the flows were already real and useful;
- discoverability was the missing piece;
- `make` could expose them without inventing a second model.

Correct interpretation:
promote through the existing public owner.

### Example B. Script growth that should be treated as a consolidation signal

Case:
multiple smoke or live harnesses getting larger and closer in shape.

Why caution is required:

- nearby scripts differing mainly by waits, symbols, or narrow variants raise
  maintenance fan-out;
- large harnesses are a reason to parameterize or refactor, not to spawn new
  sibling scripts automatically.

Correct interpretation:
consolidate or parameterize before adding another script.

## Docs

### Example A. New docs that were justified

Case:
creation of `docs/operations/` and later strategic governance docs from C25 to
C30.

Why they worked:

- active operational rules needed a canonical home outside stage evidence;
- the docs established durable owners for health, lifecycle, checkpoints, and
  readiness;
- they improved orientation across the repository platform.

Correct interpretation:
open only when a lasting rule genuinely lacks a home.

### Example B. Doc requests that should be absorbed or rejected

Case:
creating another summary doc that mostly repeats a current canonical guide.

Why expansion would be wrong:

- it would split the practical start point for the same topic;
- it adds index, link, and drift burden immediately;
- the same value is usually achievable by updating the current owner doc.

Correct interpretation:
contain or consolidate.

## Checks

### Example A. Lightweight checks that added value

Case:
`make repo-consistency-check` protecting required docs, live links, stage
indexing, script wrappers, and canonical cross-links.

Why it worked:

- the invariants are objective;
- drift would otherwise be silent;
- fixes are cheap and local;
- the check protects canonical support surfaces, not subjective style.

Correct interpretation:
open only for cheap, durable, objective invariants.

### Example B. Checks that should be rejected

Case:
adding a permanent check for a one-stage preference or for a rule that still
needs human interpretation.

Why rejection is right:

- such checks turn the guard rail into a noisy policy engine;
- they reduce trust in a surface that only works when it stays small and
  objective.

Correct interpretation:
reject.

## Quick Classification Table

| Situation | Correct answer |
|---|---|
| recurring gap with no clear owner | open a small new surface |
| recurring need already inside one owner | contain in that owner |
| confusion about canonical path | fix docs, help, or labeling |
| overlap between current siblings | consolidate |
| one-off or debug-only convenience | reject as durable surface |
| proposed invariant is objective and low-cost | maybe guard |
| proposed invariant is subjective or temporary | reject as check |

## Repository-Specific Anti-Patterns

Treat these as immediate caution signals:

- "let's add one more wrapper" without naming what current owner failed;
- "let's add a doc for this" when the current canonical doc is merely stale;
- "let's add a script because the existing one is large";
- "let's expose this expert flow publicly" when it is really for debugging;
- "let's enforce this in the lightweight check" before the rule is stable;
- "let's keep both entrypoints for convenience" when both answer the same first
  question.

## Final Rule Of Thumb

When the repository feels friction, ask this in order:

1. does capability actually not exist?
2. or is the current owner unclear, stale, or overloaded?

If capability exists, do not open a new surface until containment and
consolidation have been ruled out with concrete reasons.
