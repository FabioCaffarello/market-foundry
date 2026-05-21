# Repository Maintenance Hotspots And Cost Reduction Principles

## Purpose

This document names the highest-value repository-maintenance hotspots visible in
the current support surface and records the principles that should guide future
reduction work.

## Hotspot Matrix

| Hotspot | Why it is expensive | Current risk | C21 treatment |
|---|---|---|---|
| Duplicated support-doc maps in root docs and helper outputs | one new support doc can force many low-value edits | medium | reduced root duplication; kept detailed catalog in `docs/operations/README.md` |
| `make docs` acting like a long secondary index | promotes maintenance of a second large catalog | medium | converted to a curated shortlist of entrypoints |
| `repository-consistency-check.sh` accumulating historical required files | every support stage adds permanent upkeep to a lightweight guard rail | high | removed historical stage-report requirements from the required-doc set |
| Stage evidence drifting into current workflow guidance | historical artifacts become accidental dependencies for current work | medium | reasserted stage reports as indexed evidence, not root support entrypoints |
| Ambiguous ownership between root docs and operations docs | contributors do not know where a support change belongs | medium | documented explicit ownership boundaries and structural-cost rules |

## Impact And Urgency

### High impact

- Keep lightweight guard rails small. When they accumulate historical artifacts,
  every future support change becomes more expensive to land and harder to trust.

### Medium impact

- Keep root entrypoints shallow. They are high-traffic files, so unnecessary
  support-doc churn there increases merge pressure and review noise.
- Keep `make docs` short. It should orient contributors quickly, not duplicate
  the full operations index.

### Lower urgency but still active

- Keep area entrypoints local and concise.
- Avoid creating a new support document when an existing canonical index can
  absorb the guidance cleanly.

## Cost Reduction Principles

1. Prefer one canonical catalog over many partial catalogs.
2. Favor entrypoint docs over narrative duplication.
3. Protect invariants, not history volume, in lightweight checks.
4. Keep high-churn files shallow so support changes do not create broad edit
   fan-out.
5. Let stage reports prove decisions, but do not make them active workflow
   dependencies.
6. Add support surface only when it reduces search time or ambiguity more than
   it increases ongoing upkeep.

## Decision Test For Future Changes

Before adding a new support artifact, ask:

1. Does this solve a repository-wide discovery or governance problem?
2. Could the existing canonical index absorb it instead?
3. Will this require recurring edits in root docs, `make docs`, and checks?
4. Is the cost justified by clearer ownership or lower operator error?

If the answer to question 3 is yes and question 4 is weak, the change is
probably accidental cost.

## Recommended Maintenance Pattern

- Add the new support guide.
- Link it from `docs/operations/README.md` if it is canonical.
- Update a root entrypoint only if contributor orientation changed materially.
- Extend lightweight checks only when a real invariant is missing.
- Add the stage report to `docs/stages/INDEX.md`, but avoid making that report a
  required runtime support artifact later.
