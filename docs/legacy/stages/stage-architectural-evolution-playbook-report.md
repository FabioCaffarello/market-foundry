# Stage Report: Architectural Evolution Playbook

## Executive Summary

Created four canonical governance documents that crystallize Market Foundry's evolution discipline into operational, reusable artifacts. These documents translate 32+ stages of accumulated architectural decisions into a playbook, stage definition of done, anti-debt checklist, and Opus guidance rules — all grounded in the real state of the codebase, not generic best practices.

---

## Objective

Produce operational governance documentation that serves as the authoritative reference for designing stages, reviewing PRs, conducting the Opus, preventing technical debt, and evaluating readiness for new domains.

---

## Documents Created

### 1. `docs/architecture/market-foundry-evolution-playbook.md`
The primary governance artifact. Covers:
- **Identity and foundational principles** — The 8 principles ranked by precedence, derived from system-principles.md but contextualized for evolution decisions.
- **Stream mesh as central architecture** — Mesh properties, ownership matrix, data flow direction. Not restating stream-mesh-model.md but distilling it into actionable evolution rules.
- **Binary roles and boundaries** — The 5-binary ceiling with single-sentence purposes and explicit "never does" columns.
- **Mandatory design patterns** — FamilyProcessor, ProjectionPipeline, EvidenceFamilyDeps, and configctl lifecycle patterns with their rules.
- **Evolution by readiness** — The gate sequence (governance → contracts → activation → pattern → implementation) with specific criteria for each gate.
- **Anti-debt rules** — Five operational rules (no temporary implicit, no premature abstraction, structural reuse not framework, governance keeps pace, no scope creep).
- **Change evaluation criteria** — Four dimensions (canonical, robust, observable, scalable) with checklists.
- **Golden rules** — 10 inviolable rules of the Foundry, distilled from all architecture docs.
- **The triad** — How Foundry, MarketMonkey, and Market Raccoon inform evolution decisions.
- **Usage guide** — How to apply the playbook when designing stages, reviewing PRs, guiding the Opus, and conducting readiness reviews.

### 2. `docs/architecture/stage-definition-of-done.md`
Formalizes stage completion criteria. Covers:
- **Structural objective** — One sentence, one capability.
- **Explicit limits** — IN scope, OUT of scope, NOT CHANGED.
- **Minimum evidence** — Five evidence categories (code, contract, test, documentation, governance) with specific checkboxes.
- **Architectural acceptance criteria** — Boundary integrity, pattern conformance, mesh coherence, observability.
- **Rejection and recalibration criteria** — Hard rejections, soft rejections, and recalibration triggers.
- **Stage closure checklist** — Single checklist for stage completion.
- **Post-stage review questions** — 12 questions across structure, governance, debt, and readiness.
- **Relationship to readiness reviews** — How stages feed into readiness assessments.

### 3. `docs/architecture/anti-debt-checklist.md`
Practical review tool with 10 debt categories:
- **Boundary debt** — Binary and layer boundary violations.
- **Naming debt** — Convention divergence across subjects, types, actors, endpoints.
- **Ownership debt** — Single-writer and single-owner violations.
- **Stream mesh debt** — Mesh-code divergence.
- **Configuration debt** — Incomplete config-driven activation.
- **Premature abstraction debt** — Unnecessary generalization.
- **Query / read model debt** — Projection authority violations.
- **Documentation debt** — Stale architecture docs.
- **Governance debt** — raccoon-cli lag.
- **Operational debt** — Missing health tracking, smoke tests, compose configuration.

Also includes:
- **10 architectural drift signals** — Higher-level indicators that the system is diverging.
- **8 pre-approval questions** — To ask before approving any change.

### 4. `docs/architecture/opus-guidance-rules.md`
Formalizes Opus conduct. Covers:
- **Rule 1** — One structural capability per stage.
- **Rule 2** — Mandatory guard rails in every stage prompt.
- **Rule 3** — Code + Docs + Report triad of delivery.
- **Rule 4** — No "temporary implicit."
- **Rule 5** — Use the Foundry/MarketMonkey/Market Raccoon triad.
- **Rule 6** — Interruption and recalibration criteria (hard stops, soft stops, how to recalibrate).
- **Rule 7** — Robust prompt templates for implementation, documentation, and readiness review stages.
- **Rule 8** — Reviewing Opus results with structural, governance, quality, and debt checklists.
- **Common failure modes** — 10 observed patterns to watch for.

---

## Rationale for Document Structure

### Why four documents instead of one
A single monolithic governance doc would be too long to use as a practical tool. Each document serves a different workflow:
- **Playbook** — Strategic reference for "how does the Foundry evolve?"
- **Stage DoD** — Operational checklist for "is this stage done?"
- **Anti-debt checklist** — Review tool for "are we accumulating debt?"
- **Opus guidance** — Conductor manual for "how do I steer the Opus?"

### Why not abstract further
These documents intentionally reference specific Market Foundry concepts (FamilyProcessor, ProjectionPipeline, EVIDENCE_EVENTS, raccoon-cli). Generic governance docs that could apply to any project are useless. These docs are useful precisely because they are specific to this system.

### Why cross-reference existing docs instead of duplicating
Each new document references existing pattern docs (derive-pipeline-pattern.md, projection-writer-pattern.md, etc.) rather than restating their content. This prevents the stale-copy problem where two documents describe the same pattern differently.

---

## How to Use These Documents in Future Cycles

### Stage Design
1. Read the playbook's readiness evaluation criteria.
2. Formulate the stage prompt using opus-guidance-rules.md templates.
3. Verify readiness gate sequence is satisfied.

### Stage Execution
1. Follow the guard rails from opus-guidance-rules.md.
2. Deliver code + docs + report (Rule 3).
3. No temporary implicit code (Rule 4).

### Stage Closure
1. Run through the stage DoD's closure checklist.
2. Answer the 12 post-stage review questions.
3. Run through the anti-debt checklist for the specific debt categories affected by the stage.

### PR Review
1. Use the anti-debt checklist's pre-approval questions.
2. Verify against the playbook's change evaluation criteria.
3. Check governance updates per stage DoD's governance evidence section.

### Readiness Reviews
1. Use the playbook's readiness gate sequence.
2. Use the anti-debt checklist to verify no accumulated debt blocks the review.
3. Use opus-guidance-rules.md readiness review template for prompt formulation.

---

## Cross-References Updated

- AGENTS.md: Added reference to the evolution playbook and governance docs in the architecture documentation section.

---

## Gaps and Future raccoon-cli Opportunities

### Documentation coverage check
raccoon-cli could verify that every stage report exists for completed stages and that it contains mandatory sections (objective, scope, evidence, deferred, gaps).

### Anti-debt automation
Several anti-debt checklist items could become raccoon-cli checks:
- **Naming debt**: Verify NATS subject naming conventions against stream-taxonomy patterns.
- **Ownership debt**: Cross-validate stream-ownership-matrix.md against actual NATS registry code.
- **Governance debt**: Verify actor-ownership.md mentions all actors found in `internal/actors/scopes/`.
- **Operational debt**: Verify every actor scope directory has corresponding health trackers in the binary's run.go.

### Stage scope validation
raccoon-cli could parse stage reports for IN/OUT/NOT CHANGED sections and verify the actual diff matches declared scope. This is a complex feature but would be high-value.

### Playbook drift detection
raccoon-cli could verify that key invariants stated in the playbook (five-binary ceiling, single-writer per stream, gateway statelessness) are still enforced by its existing rules.

---

## Verification

- All four documents are internally consistent — cross-references verified.
- No contradictions with existing architecture docs — patterns, naming, topology, and principles are aligned with system-principles.md, stream-mesh-model.md, derive-pipeline-pattern.md, projection-writer-pattern.md, and all other canonical docs.
- Documents are specific to Market Foundry — no generic corporate text, no content that could apply to any arbitrary project.
- Documents reference the real state of the system (32 completed stages, 3 evidence types, FamilyProcessor pattern, ProjectionPipeline pattern, known governance gaps).
