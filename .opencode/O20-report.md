# O20 Report

## Objective

Harden the joint information-system boundary between `.opencode/` and `docs/`
so the post-O17/O18/O19 topology resists recontamination, drift, duplication,
and opportunistic support-surface growth.

## Explicit Invariants

- `.opencode/` remains limited to navigation, compression, handoff, and
  safe-change support.
- `docs/` remains the canonical human owner surface.
- `docs/stages/` and `docs/archive/` hold history and superseded material, not
  current recurring answers.
- recurring human ownership lives in `docs/`, never in `.opencode/`.
- new active surfaces must earn their place through classification, not habit.

## Checks Added Or Tightened

- joint repo consistency now checks the active docs topology, owner-map
  uniqueness, orphaned active docs, navigation alignment, historical leakage,
  and `.opencode/` integration.
- `.opencode` consistency now also guards against human-owner filename
  duplication and direct mirroring of active doc stems.
- active surface shape is bounded so generic governance/taxonomy essays do not
  quietly re-enter the primary navigation model.

## Validation

- `../scripts/opencode-consistency-check.sh`
- `../scripts/repository-consistency-check.sh`
- `make repo-consistency-check`

## Limitations

- duplication detection stays heuristic and structural; it targets real owner
  competition, not semantic similarity at paragraph level.
- architecture-wide history detection is intentionally not exhaustive because
  many architecture docs legitimately carry findings, limits, or audits.
- future active-surface expansion still requires updating the checks in the same
  change; that is deliberate friction, not accidental rigidity.

## Follow-Through Rules

- update owner docs before updating `.opencode/`.
- when adding an active doc, update its README/owner map/checks together.
- when retiring an active doc, move it to `docs/archive/` or preserve it in
  `docs/stages/` if the value is historical evidence.
- if `.opencode/` needs a new leaf, update `TARGET-TREE.md`, navigation, and
  checks in the same patch.

## Future Expansion Criteria

- add a new active surface only for a recurring current question with stable
  ownership;
- reject expansion when the need is historical, transitional, or already
  answerable by an existing owner;
- keep checks proportional: block real drift, not editorial noise.
