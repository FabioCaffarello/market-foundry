# Stage Definition of Done

## Purpose

This document defines what it means for a stage to be **complete** in Market Foundry. A stage is the fundamental unit of architectural evolution — each stage adds exactly one structural capability to the system.

Stages are not sprints. They have no time limit. They are done when they meet every criterion below, and not before.

---

## Structural Objective

Every stage must have a **single, declarable structural objective** that can be stated in one sentence.

Examples of valid objectives:
- "Prove observation → evidence pipeline end-to-end with candle sampling."
- "Introduce ProjectionPipeline pattern for declarative store family registration."
- "Add third evidence type (volume) to validate family adoption at scale."

Examples of invalid objectives:
- "Improve the system." (no specificity)
- "Add volume evidence type and also fix config-driven activation." (two capabilities)
- "Refactor derive and update docs." (mixed concerns — if refactor changes structure, docs update is part of it; if separate, they are separate stages)

---

## Explicit Limits

Every stage report must declare:

1. **What is IN scope** — the specific files, patterns, actors, adapters, routes, and contracts that will be created or modified.
2. **What is OUT of scope** — capabilities that are explicitly deferred, even if closely related.
3. **What is NOT being changed** — systems and patterns that must remain untouched (to verify no unintended side effects).

If a stage report does not have explicit limits, the stage is not ready to begin.

---

## Minimum Evidence Required

A stage is not complete without all of the following:

### Code Evidence
- [ ] All new code compiles and passes `make verify` (tests + quality gate).
- [ ] All new actors follow established patterns (FamilyProcessor for derive, ProjectionPipeline for store, EvidenceFamilyDeps for gateway — where applicable).
- [ ] Layer sovereignty verified — no outward or sideways imports introduced.
- [ ] No new binaries introduced without architectural justification.

### Contract Evidence
- [ ] All new NATS subjects follow stream taxonomy naming (`{domain}.{plane}.{aggregate}.{verb}[.{key}]`).
- [ ] All new message types follow envelope convention (`{domain}.{plane}.{version}.{name}`).
- [ ] All new KV buckets have documented key format, retention, and max size.
- [ ] All new query subjects have documented request/reply types.

### Test Evidence
- [ ] Pure logic (samplers, domain types) has table-driven unit tests.
- [ ] Adapter code has integration tests where applicable.
- [ ] Smoke test covers the end-to-end path introduced by this stage (or extends existing smoke test).
- [ ] No tests disabled, skipped, or marked as TODO without explicit justification in the stage report.

### Documentation Evidence
- [ ] Stage report created at `docs/stages/stage-{id}-{slug}-report.md`.
- [ ] Architecture docs updated if the stage changes topology, ownership, patterns, or contracts.
- [ ] actor-ownership.md reflects any new actors, streams, or KV buckets.
- [ ] stream-family-catalog.md reflects any new families or family changes.

### Governance Evidence
- [ ] `raccoon-cli` rules updated if the stage introduces new subjects, consumers, or durables.
- [ ] `make check` passes with no new warnings related to the stage's changes.
- [ ] No governance debt introduced (or if introduced, explicitly documented as a gap with resolution plan).

---

## Architectural Acceptance Criteria

Beyond evidence, the stage must satisfy these architectural criteria:

### Boundary Integrity
- [ ] No binary acquired responsibilities outside its declared purpose.
- [ ] Gateway remains stateless — no KV access, no domain logic, no event publishing.
- [ ] Store remains read-only — no domain event production.
- [ ] Derive remains write-only to EVIDENCE_EVENTS — no query serving (unless explicitly justified as in early slices before store existed).
- [ ] Single-writer invariant preserved for all streams and KV buckets.

### Pattern Conformance
- [ ] New evidence types follow the 13-step onboarding checklist (evidence-read-model-guidelines.md).
- [ ] New derive families use FamilyProcessor registration — no SourceScopeActor modifications.
- [ ] New store families use ProjectionPipeline registration — no StoreSupervisor modifications beyond the pipeline list.
- [ ] New gateway routes use EvidenceFamilyDeps grouping.

### Mesh Coherence
- [ ] Stream ownership matrix remains accurate after the stage.
- [ ] Data flow remains acyclic across the governed paths (`configctl → ingest → derive`, `derive → store → gateway`, `derive → execute`, `derive/execute → writer → ClickHouse → gateway`).
- [ ] No new streams created without architectural justification and catalog entry.
- [ ] Consumer durables follow naming convention (`{service}-{family}` or `{service}-binding-watcher`).

### Observability
- [ ] New consumers have health trackers.
- [ ] New projections have health trackers.
- [ ] Projection actors emit stats (materialized, skipped, rejected).
- [ ] New actors use structured logging with context.

---

## Rejection and Recalibration Criteria

A stage must be **rejected or recalibrated** if any of the following are true:

### Hard Rejections (stage cannot proceed)
- Introduces a second structural capability beyond the declared objective.
- Breaks layer sovereignty (detected by `raccoon-cli arch-guard`).
- Violates single-writer invariant on any stream or KV bucket.
- Introduces a new binary beyond the current governed set (`configctl`, `gateway`, `ingest`, `derive`, `store`, `execute`, `writer`, `migrate`) without prior architectural justification.
- Reintroduces quality-service patterns, naming, or identity.
- Leaves governance debt without explicit documentation and resolution plan.

### Soft Rejections (stage must be revised)
- Stage report lacks explicit scope limits (IN/OUT/NOT CHANGED).
- Tests pass but smoke test does not cover the new path.
- Architecture docs not updated to reflect structural changes.
- raccoon-cli rules not updated when new subjects or consumers are introduced.
- Naming does not follow established conventions.

### Recalibration Triggers
- Scope expanded during implementation beyond declared limits → split into two stages.
- Blocking gap discovered that requires prerequisite work → pause, create prerequisite stage, resume after.
- Pattern does not fit and requires structural innovation → document the tension, propose pattern evolution as separate stage.

---

## Stage Closure Checklist

Before declaring a stage complete:

```
[ ] Single structural objective achieved and demonstrable
[ ] Stage report written with: objective, scope, evidence, outcomes, deferred items, gaps
[ ] Code compiles: make build passes
[ ] Tests pass: make test passes
[ ] Quality gate passes: make verify passes
[ ] Smoke test covers the new path
[ ] Architecture docs updated (actor-ownership, stream-family-catalog, relevant pattern docs)
[ ] raccoon-cli rules updated if topology changed
[ ] No untracked governance debt
[ ] Stage report lists all known gaps with severity and resolution path
[ ] Cross-references updated in related docs
```

---

## Post-Stage Review Questions

After closing a stage, answer these questions honestly:

### Structure
1. Did this stage make the system more canonical, or did it introduce a deviation?
2. Is every new pattern consistent with existing patterns, or was a new pattern necessary?
3. If a new pattern was introduced, is it documented and ready for reuse?

### Governance
4. Are actor-ownership.md and stream-family-catalog.md current after this stage?
5. Can raccoon-cli detect violations of the patterns introduced in this stage?
6. If not, is a raccoon-cli update staged as follow-up?

### Debt
7. Did this stage introduce any gaps? If yes, are they documented with severity and resolution path?
8. Did this stage resolve any gaps from previous stages?
9. Is the system in a better governance state after this stage than before?

### Readiness
10. Does this stage's completion unlock any readiness gate for future work?
11. What is the next logical stage after this one?
12. Are there any prerequisites that must be met before the next stage can begin?

---

## Relationship to Readiness Reviews

Stages and readiness reviews serve different purposes:

- **Stage**: Adds one structural capability. Evaluated against this Definition of Done.
- **Readiness Review**: Evaluates whether the system is ready to enter a new domain or capability area. Evaluates multiple completed stages collectively.

A readiness review may be triggered when:
- A sequence of stages completes a capability area (e.g., all evidence types implemented).
- A new domain is being considered (e.g., signal).
- Governance concerns need formal assessment.

The readiness gate sequence from the [evolution playbook](market-foundry-evolution-playbook.md) applies:

```
governance current → contracts defined → activation verified → pattern proven → implementation
```

No readiness review can approve expansion if governance is not current. No stage can begin in a new domain without a passing readiness review.
