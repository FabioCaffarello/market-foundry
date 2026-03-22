# Hotspot Candidates Prioritization And Selection Rationale

## Purpose

This document records the explicit C32-model comparison used in C33 to rank the
current repository-platform hotspot candidates.

The comparison is qualitative by design. It uses the C32 `high` / `medium` /
`low` method and avoids numeric scoring.

## Candidate Set

| Candidate | Problem shape | Expected bucket |
|---|---|---|
| A. Canonical operational-proof taxonomy drift | Structural misalignment across `Makefile`, workflow docs, and harness governance | structural improvement |
| B. Bring-up and proof entrypoint overlap | Repeated explanation required to keep `live*`, `up` + `seed*`, and `smoke*` legible | structural improvement |
| C. Workflow-document fan-out | Same command story repeated across many active docs | structural improvement |
| D. Lightweight guard-rail coverage gap | Alignment drift can survive because current checks do not verify proof-taxonomy completeness | quick win after structural alignment |

## C32 Criteria Matrix

| Candidate | Impact on daily development | Friction reduction | Entropy-risk reduction | Maintenance cost | Predictability improvement | Discoverability gain | Environment reliability | Implementation cost |
|---|---|---|---|---|---|---|---|---|
| A. Canonical operational-proof taxonomy drift | high | high | high | medium | high | high | high | medium |
| B. Bring-up and proof entrypoint overlap | high | medium | medium | medium | high | medium | medium | medium |
| C. Workflow-document fan-out | medium | medium | high | medium | medium | medium | medium | medium |
| D. Lightweight guard-rail coverage gap | medium | medium | medium | low | medium | low | high | low |

## Candidate-By-Candidate Rationale

### Candidate A. Canonical operational-proof taxonomy drift

Dominant value:
predictability and environment reliability.

Why the ratings land where they do:

- `high` daily impact because proof selection is part of the normal validation
  loop for any runtime-facing work;
- `high` friction reduction because contributors should not need to reconcile
  `Makefile` with multiple workflow documents to know which proof surface is
  current;
- `high` entropy reduction because the mismatch is already cross-surface, not
  local;
- `medium` maintenance cost because the right solution is consolidation and
  re-alignment, not permanent expansion;
- `high` discoverability because specialized proofs become easier to find and
  trust when the canonical taxonomy is current;
- `medium` implementation cost because several active docs need coordinated
  tightening, but the work is still bounded.

Decision:
`do next`, and treat it as the primary hotspot for the next short wave.

### Candidate B. Bring-up and proof entrypoint overlap

Dominant value:
predictability.

Why it is not first:

- the distinction between orchestration and proof already exists and is stated
  repeatedly;
- the current weakness is less that the model is absent and more that the proof
  taxonomy underneath it has drifted.

Decision:
defer behind Candidate A unless work on A exposes a tighter consolidation
opportunity.

### Candidate C. Workflow-document fan-out

Dominant value:
entropy control.

Why it is not first:

- this is a real structural cost, but much of the fan-out is purposeful layering
  for onboarding, navigation, lifecycle, and reference use cases;
- reducing fan-out safely requires knowing which workflow story is actually
  unstable first;
- Candidate A is the more operationally urgent subset of this broader problem.

Decision:
contain for now; allow only changes that reduce fan-out as part of the primary
hotspot correction.

### Candidate D. Lightweight guard-rail coverage gap

Dominant value:
environment reliability.

Why it is not first:

- the check gap is real, but it protects the same surface that Candidate A
  identifies as currently misaligned;
- C32 explicitly prefers the smallest durable move, which means aligning the
  canonical owner surface before adding more enforcement.

Decision:
hold as the secondary reserve for the same short-wave area, not as a separate
independent wave.

## Final Ranking

1. Candidate A. Canonical operational-proof taxonomy drift
2. Candidate D. Lightweight guard-rail coverage gap
3. Candidate B. Bring-up and proof entrypoint overlap
4. Candidate C. Workflow-document fan-out

## Selection Logic

Candidate A wins because it has the strongest combined effect on:

- trust in the canonical workflow;
- reduction of recurring support-surface ambiguity;
- entropy control across several active operational surfaces;
- and readiness for another expansion wave that will likely add or evolve proof
  flows again.

Candidate D is kept as reserve because it is the natural follow-through after
the taxonomy is corrected, not before.

## Recommended C34 Scope Boundary

Authorize only a short applied wave that:

1. tightens the canonical operational-proof taxonomy;
2. reduces overlap between proof-selection docs and support catalogs;
3. adds the minimum consistency check needed to keep that taxonomy aligned.

Reject for C34:

- opening new abstract governance models;
- broad workflow-doc rewrites unrelated to proof-surface clarity;
- runtime or domain-architecture refactors disguised as workflow cleanup.

## Related Documents

- [`continuous-prioritization-model-for-the-development-platform.md`](continuous-prioritization-model-for-the-development-platform.md)
- [`canonical-workflow-hotspot-assessment-and-selection.md`](canonical-workflow-hotspot-assessment-and-selection.md)
