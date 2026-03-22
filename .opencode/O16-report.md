# O16 Report

## Summary

O16 hardens `.opencode` as a thin repository-native navigation layer. The work
adds explicit lightweight invariants, strengthens cheap drift checks, and
defines when this layer may evolve without turning it into a second normative
system.

## Invariants

- `.opencode` is navigational only; the codebase and canonical docs remain the
  source of truth.
- Topology stays minimal: root config files, O-reports, one root agent, two
  profiles, and the four context areas `repo`, `runtime`, `change`, and
  `intelligence`.
- Context navigation must keep explicit owner-doc anchors and route to every
  approved local leaf.
- Local links and explicit `make` references must resolve against the live
  repository.
- Removed or prohibited legacy surfaces must not reappear as active guidance.
- Reports may record change rationale, but they must not grow into owner maps,
  indexes, or durable workflow catalogs.

## Checks And Integration

- `scripts/opencode-consistency-check.sh` now validates the approved minimal
  context-file set, bounded root files, owner-doc anchors, per-area navigation
  completeness, and report-vs-owner-surface separation.
- These checks continue to run through `make repo-consistency-check`, and
  therefore through `make check`, `make verify`, and `make check-deep`.
- The guard rail stays proportional: it does not lint prose, style, markdown
  cosmetics, or broad editorial choices.

## Evolution And Maintenance Rule

- Grow `.opencode` only when canonical entry navigation becomes materially
  slower or breakable without a short local router.
- Prefer fixing `AGENTS.md`, `Makefile`, `README.md`, `DEVELOPMENT.md`, or the
  relevant `docs/*` owner surface before adding new `.opencode` material.
- If a new recurring need requires a new `.opencode` file, update the target
  tree, wire it from the area navigation file, and prove that it compresses an
  existing canonical owner surface rather than duplicating it.
- If a concern needs policy depth, lifecycle detail, or subject ownership, it
  belongs in the owner docs, not in `.opencode`.

## Validation

- Ran `../scripts/opencode-consistency-check.sh`.
- Ran `make repo-consistency-check`.

## Limitations

- The checks intentionally do not score semantic accuracy beyond explicit links,
  topology, bounded ownership signals, and known prohibited-surface drift.
- They will not detect every stale sentence if the referenced owner doc still
  exists and the navigation shape remains valid.
- The checks assume the approved minimal topology is still the right shape; a
  legitimate future expansion must update both the target tree and the
  hardening script together.

## Follow-Through Rules

- When canonical workflow or ownership entrypoints move, update `.opencode` in
  the same change.
- When `.opencode` adds or removes a leaf file, update `TARGET-TREE.md`,
  profile reachability, and the navigation file for that area in the same
  change.
- When a drift bug appears that this layer should catch, add a cheap,
  objective, repository-local check or leave it in review; do not add fuzzy
  warnings.

## Expansion Criteria

`.opencode` may grow when all are true:

- the need is recurring;
- the answer already exists in canonical owner docs;
- a short router materially reduces navigation cost; and
- the new file can stay thin and cheap to validate.

`.opencode` should not grow when any is true:

- the change would create a second owner catalog;
- the new content would explain policy better than the owner docs;
- the need is historical evidence or stage narrative;
- the change mainly adds taxonomy, ceremony, or command duplication.
