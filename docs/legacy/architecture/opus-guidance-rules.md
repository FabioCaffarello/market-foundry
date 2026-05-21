# Opus Guidance Rules

## Purpose

This document formalizes how Claude (Opus) must be guided when conducting architectural stages in Market Foundry. The Opus is a powerful executor, but without disciplined guidance it will introduce drift, scope creep, premature abstraction, and governance debt.

These rules exist because Market Foundry's quality comes from disciplined evolution, not raw output volume.

---

## Rule 1: One Structural Capability Per Stage

Every prompt that initiates a stage must declare exactly one structural capability.

**Correct:**
> "Implement volume evidence type end-to-end following FamilyProcessor and ProjectionPipeline patterns."

**Incorrect:**
> "Implement volume evidence type and also fix config-driven activation and update raccoon-cli."

If the Opus proposes expanding scope during execution, interrupt. The expansion becomes a separate stage.

### How to enforce
- State the single objective at the top of every stage prompt.
- Include an explicit "OUT OF SCOPE" section listing related but excluded work.
- After execution, verify the stage report's scope matches the original prompt.

---

## Rule 2: Mandatory Guard Rails in Every Stage Prompt

Every stage prompt must include:

```
Guard rails:
- Do not introduce new binaries without architectural justification.
- Do not modify spawning loops if the pattern supports declarative registration.
- Do not abstract prematurely — prefer explicit duplication.
- Do not introduce interfaces with single implementations.
- Do not skip governance updates (docs, raccoon-cli rules).
- Do not leave "temporary" code without documented resolution path.
- Verify layer sovereignty: no outward or sideways imports.
- Verify single-writer invariant for all streams and KV buckets.
- Follow established naming conventions (stream-taxonomy, evidence-read-model-guidelines).
```

Tailor additional guard rails to the specific stage, but never remove the base set above.

---

## Rule 3: Code + Docs + Report (The Triad of Delivery)

Every stage must produce all three:

1. **Code** — Compilable, tested, passing `make verify`.
2. **Docs** — Updated architecture docs reflecting any structural changes (actor-ownership, stream-family-catalog, pattern docs).
3. **Report** — Stage report at `docs/stages/stage-{id}-{slug}-report.md` with: objective, scope, evidence, outcomes, deferred items, gaps.

If the Opus delivers code without docs and report, the stage is incomplete. Reject and require completion.

### Report structure (mandatory sections)
- Executive summary (1-3 sentences)
- Objective and scope (IN / OUT / NOT CHANGED)
- What was implemented (with file-level detail)
- What was deferred (with justification)
- Known gaps (with severity and resolution path)
- Architectural impact (what changed in topology, ownership, patterns)
- Verification evidence (tests, smoke, quality gate results)

---

## Rule 4: No "Temporary Implicit"

The Opus must never introduce code described as "temporary" without:

1. An explicit comment in code: `// TEMPORARY: [description]. Resolution: [stage or action].`
2. A gap entry in the stage report with severity rating.
3. A concrete resolution path (which future stage will address it).

If the Opus says "we can fix this later" without all three items, reject the change.

### Common "temporary implicit" patterns to watch for
- Hardcoded values that should be config-driven.
- TODO comments without associated stage reference.
- Stubbed functions that return nil.
- Error handling that silently swallows failures.
- Test skips without justification.

---

## Rule 5: Use the Triad (Foundry / MarketMonkey / Market Raccoon)

When the Opus makes architectural decisions, it must ground them in the triad:

### Market Foundry (this repository)
- **Use for:** Pattern conformance, naming conventions, established topology, governance rules.
- **The Opus must:** Read existing pattern docs before implementing. Follow FamilyProcessor, ProjectionPipeline, EvidenceFamilyDeps patterns. Respect stream-taxonomy naming.

### MarketMonkey (reference runtime)
- **Use for:** Validating that patterns can support real-world market data processing. Understanding actor-per-stream supervision. Understanding observation → evidence → signal data flow.
- **The Opus must:** Reference MarketMonkey patterns when designing new pipelines. Never copy code — re-implement natively.

### Market Raccoon (domain reference)
- **Use for:** Boundary definitions, domain maturity assessment, invariant validation. Preventing premature coupling.
- **The Opus must:** Check domain readiness before implementing new domains. Respect domain boundaries even when shortcuts seem faster.

### How to reference in prompts
Include relevant context:
```
Reference context:
- Foundry patterns: [list relevant pattern docs]
- MarketMonkey influence: [specific patterns to follow]
- Market Raccoon boundaries: [domain readiness status]
```

---

## Rule 6: Interruption and Recalibration Criteria

Stop the Opus and recalibrate if any of the following occur:

### Hard stops (immediately interrupt)
- The Opus proposes a sixth binary.
- The Opus introduces Kafka, gRPC, or a second message broker.
- The Opus creates an interface with a single implementation where a struct suffices.
- The Opus modifies SourceScopeActor spawning loop when FamilyProcessor should be used.
- The Opus modifies StoreSupervisor spawning loop when ProjectionPipeline should be used.
- The Opus produces domain events from store.
- The Opus adds domain logic to gateway.
- The Opus skips governance updates ("we can update docs later").

### Soft stops (pause and verify direction)
- The Opus adds more than 15 new files in a single stage (potential scope creep).
- The Opus introduces a new pattern instead of following an existing one.
- The Opus creates "utility" or "helper" packages.
- The Opus adds error handling for scenarios that cannot occur.
- The Opus adds feature flags or backward-compatibility shims.
- The Opus proposes changes to files explicitly listed as "NOT CHANGED" in scope.

### How to recalibrate
1. State what went wrong: "You expanded scope beyond X" or "You introduced pattern Y instead of using established pattern Z."
2. Reference the specific rule: "Per Rule 1, one capability per stage" or "Per the evolution playbook, governance before expansion."
3. Provide corrected direction: "Revert the addition of X and instead follow the FamilyProcessor pattern as documented in derive-family-processor-pattern.md."
4. If the divergence is large, restart the stage from scratch rather than patching.

---

## Rule 7: Robust Prompt Formulation

### Prompt structure for implementation stages

```markdown
# Stage S{XX}: {Title}

## Objective
{Single sentence describing the one structural capability.}

## Context
- Current state: {What exists today relevant to this stage.}
- Prerequisites: {Readiness gates that have been met.}
- Reference docs: {List of architecture docs the Opus must read before implementing.}

## Scope
### IN scope
- {Specific files, patterns, actors, adapters, routes, contracts.}

### OUT of scope
- {Explicitly excluded work, even if closely related.}

### NOT CHANGED
- {Files and patterns that must remain untouched.}

## Patterns to follow
- {Reference specific pattern docs with file paths.}

## Guard rails
- {Base set from Rule 2 plus stage-specific additions.}

## Deliverables
1. Code: {What must compile and pass tests.}
2. Docs: {Which architecture docs must be updated.}
3. Report: docs/stages/stage-s{XX}-{slug}-report.md

## Verification
- make verify must pass
- make smoke must cover the new path
- {Stage-specific verification steps}
```

### Prompt structure for documentation stages

```markdown
# Stage S{XX}: {Title}

## Objective
{Single sentence describing the documentation deliverable.}

## Context
- {What triggered this documentation need.}
- {Current state of relevant docs.}

## Deliverables
- {Exact file paths and content expectations for each document.}

## Consistency requirements
- {Which existing docs must remain consistent with the new ones.}
- {Cross-references that must be updated.}

## Guard rails
- Do not introduce concepts that contradict existing architecture docs.
- Do not invent new domains or patterns.
- Reflect the real system state, not an idealized version.

## Report
docs/stages/stage-s{XX}-{slug}-report.md
```

### Prompt structure for readiness reviews

```markdown
# Readiness Review: {Domain or Capability}

## Question
Is Market Foundry ready to begin implementing {domain/capability}?

## Assessment scope
- {Subsystems to evaluate.}
- {Prerequisites from the evolution playbook to verify.}

## Expected output
1. Verdict: READY / CONDITIONALLY READY / NOT READY
2. Per-subsystem assessment with evidence
3. Blocking gaps (if any) with severity and resolution path
4. Recommendation for next stage

## Reference docs
- {List of docs the Opus must read for this assessment.}
```

---

## Rule 8: Reviewing Opus Results

After every stage, verify:

### Structural review
1. **Scope compliance** — Did the Opus stay within declared scope? Check the diff against IN/OUT/NOT CHANGED sections.
2. **Pattern conformance** — Did it follow existing patterns or introduce new ones? If new, is it justified?
3. **Naming compliance** — Do all new identifiers follow naming conventions?
4. **Layer sovereignty** — Run `raccoon-cli arch-guard`. No violations.

### Governance review
5. **Docs current** — Are actor-ownership.md, stream-family-catalog.md, and relevant pattern docs updated?
6. **raccoon-cli rules current** — If topology changed, are CLI rules updated?
7. **Stage report complete** — Does it have all mandatory sections? Are gaps listed with severity?

### Quality review
8. **Tests pass** — `make verify` succeeds.
9. **Smoke test covers new path** — The specific E2E path introduced is exercised.
10. **No new warnings** — `make check` does not show new governance warnings.

### Debt review
11. **No implicit temporariness** — Every "temporary" item has comment + gap entry + resolution path.
12. **No premature abstraction** — Every interface has multiple implementations. Every helper is used more than once.
13. **No scope creep** — Changes are limited to declared scope.

If any review item fails, the stage is not complete. Require correction before accepting.

---

## Common Opus Failure Modes

These are patterns observed in prior stages. Watch for them:

1. **Over-documentation** — Creating docs for patterns that don't need standalone documentation. If the pattern is already documented, don't create a second doc.
2. **Premature generalization** — Introducing `type Sampler interface` when concrete structs suffice.
3. **Scope expansion** — "While we're here, let's also..." — always reject this.
4. **Governance deferral** — "We can update actor-ownership.md in the next stage" — always reject this.
5. **Framework emergence** — Simple list of processors becoming a registry with dynamic loading.
6. **Naming creativity** — Inventing new naming conventions instead of following established ones.
7. **Backward-compatibility shims** — Adding re-exports or aliases for renamed types.
8. **Defensive over-engineering** — Error handling for impossible states, fallbacks for scenarios that never occur.
9. **Summary repetition** — Restating what was done at the end of every response. Not needed — the diff speaks.
10. **Idealization** — Describing the system as it should be rather than as it is. Stage reports must be honest.
