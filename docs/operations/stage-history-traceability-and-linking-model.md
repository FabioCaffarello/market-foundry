# Stage History Traceability And Linking Model

## Purpose

This document defines the practical linking model for reading and maintaining
stage history across charter, execution, promoted docs, and gate decisions.

Use it when stage history feels readable as isolated files but unclear as an
evolution trail.

## Reading Model

Read a governed stage chain through these questions:

| Question | Primary artifact |
|---|---|
| Why was this wave opened? | Charter or scope-freeze stage and its promoted architecture docs |
| What was allowed or frozen? | Charter report plus governing architecture docs |
| What actually changed? | Execution, hardening, proof, or reconciliation stage reports |
| Was the wave accepted? | Gate or closure stage |
| What remains current after the wave? | Promoted operations or architecture docs |

## Standard Linking Pattern

For future stages, prefer this light chain:

1. `docs/stages/INDEX.md` points to the opening stage and the closing gate for the wave.
2. The opening stage report points to the charter, freeze, or criteria docs promoted into `docs/architecture/`.
3. Execution stages point back to the opening charter when wave scope matters.
4. Gate or closure stages point to the execution tranche they evaluate.
5. Promoted docs point back to stage history only when historical rationale materially helps the reader.

## Artifact Responsibilities

| Artifact | Should answer |
|---|---|
| Charter stage report | Why this wave exists, what is authorized, what is frozen |
| Execution stage report | What changed, what stayed bounded, what evidence was produced |
| Proof or validation stage report | What was exercised and what confidence was gained |
| Gate stage report | Whether the charter succeeded, failed, or needs correction |
| Promoted operations doc | How contributors should work now |
| Promoted architecture doc | What rule, boundary, or decision is now canonical |

## Index Responsibilities

`docs/stages/INDEX.md` should make these things visible without becoming a
registry:

- phase ordering
- repository-support stages
- recent wave start/end landmarks
- the distinction between historical evidence and current canonical docs

## Recent Wave Examples

### Refactor and documentation consolidation

| Role | Artifact |
|---|---|
| Charter open | [S211](../stages/stage-s211-refactor-wave-charter-and-entry-freeze-report.md) |
| Execution tranche | [S212](../stages/stage-s212-repository-architecture-census-and-refactor-map-report.md) through [S215](../stages/stage-s215-documentation-consolidation-and-noise-removal-report.md) |
| Exit gate | [S216](../stages/stage-s216-post-refactor-and-documentation-exit-gate-report.md) |
| Evidence reconciliation | [S217](../stages/stage-s217-exit-gate-closure-and-evidence-reconciliation-report.md) |

### Domain evolution depth wave

| Role | Artifact |
|---|---|
| Charter open | [S233](../stages/stage-s233-domain-evolution-charter-and-scope-freeze-report.md) |
| Execution tranche | [S234](../stages/stage-s234-decision-domain-deepening-report.md) through [S237](../stages/stage-s237-integration-and-ci-hardening-for-the-new-domain-depth-report.md) |
| Gate | [S238](../stages/stage-s238-post-domain-evolution-gate-report.md) |
| Governance correction | [S239](../stages/stage-s239-charter-correction-and-hardening-closure-report.md) |

### Breadth wave

| Role | Artifact |
|---|---|
| Charter open | [S240](../stages/stage-s240-breadth-charter-and-scope-freeze-report.md) |
| Execution tranche | [S241](../stages/stage-s241-decision-breadth-expansion-report.md) through [S243](../stages/stage-s243-risk-breadth-expansion-report.md) |
| Gate | [S244](../stages/stage-s244-breadth-integration-and-gate-report.md) |
| Hardening follow-up | [S245](../stages/stage-s245-remote-ci-closure-for-breadth-wave-report.md) through [S248](../stages/stage-s248-post-breadth-hardening-gate-report.md) |

### Behavioral wave

| Role | Artifact |
|---|---|
| Charter open | [S249](../stages/stage-s249-behavioral-feature-charter-and-scope-freeze-report.md) |
| Execution tranche | [S250](../stages/stage-s250-decision-to-strategy-behavior-activation-report.md) through [S253](../stages/stage-s253-integration-and-ci-hardening-for-behavioral-scenarios-report.md) |
| Gate | [S254](../stages/stage-s254-post-behavioral-wave-gate-report.md) |

## Lightweight Conventions For Future Stages

- In a charter stage, include predecessor and successor context when the wave is explicit.
- In a gate stage, name the charter or reviewed tranche directly in the summary.
- In a support stage, link the promoted operations docs and the stage index entrypoint.
- When a wave needs correction after a gate, create a clearly named reconciliation or closure stage instead of silently rewriting earlier verdicts.

## Anti-Patterns

- A gate report that does not name the charter it evaluates
- A charter stage with no obvious closing gate or closure path
- A support stage that leaves its durable rules only inside the report
- A new index section that repeats whole report summaries instead of surfacing landmarks
- Retrofitting dozens of old reports when an index or entrypoint change would solve the problem

## Related Documents

- [`stage-documentation-governance-and-narrative-coherence.md`](stage-documentation-governance-and-narrative-coherence.md)
- [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md)
- [`../architecture/monorepo-documentation-and-stage-governance.md`](../architecture/monorepo-documentation-and-stage-governance.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
