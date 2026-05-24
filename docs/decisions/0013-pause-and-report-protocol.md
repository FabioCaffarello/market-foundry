# ADR 0013: Pause-and-report protocol

## Status

Accepted. Formalization of practice instituted across Phase 1-4
and surfaced explicitly during the Phase 4.1 wave.

## Context

market-foundry is developed via a **two-agent collaboration model**:
an *architect* role that scopes prompts and an *executor* role
that applies changes. The architect cannot inspect code reality
in the same depth or freshness as the executor; the executor
cannot decide scope or trade-offs without the architect's
context. Both sides are necessary; both sides make mistakes.

Phase 1+2+3 surfaced five concrete divergences caught by an
informal "stop and ask" reflex (P2.3 toolchain-vs-project
version, P2.Y legacy refs in `bootstrap-check.sh`, P3.3 GitHub
fork tier lockdown, P3.5 false-positive shellcheck audit, P3.7
golangci-lint pin drift in CI).

Phase 4 raised the stakes: ~50 sub-prompts produced 10 cataloged
architect-prescription mistakes (DOC-1 `backups/` assumption,
P4.1.3.a contract-audit table assumption, P4.1.6.a v1/v2 GHA
services and docker network errors, E2E-1 missed inventory,
P4.1.8 framing pushback, M10 number prescription, P4.1.10
defensive-scan need, P4.2 line numbers off-by-one, P4.5 M19
framing over-stated). Pattern absolute: the architect's
confidence in mechanism details (line numbers, schema specifics,
runtime semantics, framing) is inversely correlated with their
accuracy.

Without an explicit discipline, the executor's path of least
resistance is to apply the prompt as written and propagate
errors into commits. Phase 4 evidence shows the cost of this
path: a single silent error compounds — into a wrong fix, into
a wrong test, into wrong follow-up prompts that branch from
the wrong premise.

This ADR formalizes the **pause-and-report protocol** that
caught all 10 cases.

## Decision

When the executor encounters a scope boundary, a prescription
gap, or an architectural ambiguity, the executor **MUST pause
execution and report findings** rather than silently proceed.

Each substantive prompt **MUST declare pause-and-report
triggers** explicitly in its Protocolo section. Generic
"pause if anything looks wrong" is insufficient; specific
triggers are required.

The owner respects pause signals and provides direction before
execution resumes.

### Canonical trigger types

1. **Prescription mismatch** — the executor's observed reality
   differs from the prompt's premise (line numbers, file paths,
   API signatures, schema shape).
2. **Defensive-scan finding** — a scan reveals scope expansion
   beyond the prompt's inventory (see ADR 0014).
3. **Architectural ambiguity** — a choice between viable
   approaches requires owner judgment.
4. **Time-cap exceedance** — an investigation cap is reached
   without resolution (see the time-cap convention in
   `.claude/skills/investigation-skill/SKILL.md`).
5. **Wave-depth signal** — a wave-closure decision point has
   been reached (see ADR 0015).

### Procedure (5 steps)

1. **Pause** — stop applying changes immediately.
2. **Report** — summarize expected vs found, with concrete
   evidence (paths, line numbers, exit codes, command output).
3. **Options** — provide 2-4 distinct paths forward (A/B/C/D),
   each with explicit trade-offs.
4. **Wait** — do not proceed without explicit direction.
5. **Proceed** — only after authorization. Reference the chosen
   option in the eventual commit message.

## Consequences

### Positive

- Architect mistakes are caught early, before they ship.
- The executor's substantive judgment (the *co-architect
  dynamic*, see `.claude/agents/architect-agent.md`) is
  respected when reasoning is sound.
- The owner remains the decision point for trade-offs that
  neither agent can resolve alone.
- Commit history reflects deliberate choices rather than
  silently-extended scope.

### Negative

- Per-prompt overhead: pause / discuss / resume cycles slow
  individual sub-prompts.
- Friction for trivial prescriptions where the executor's
  confidence is very high.
- Requires architect discipline to declare specific triggers
  per prompt rather than rely on generic guidance.

### Mitigation

- The "defer mechanism to executor" principle
  (`.claude/agents/architect-agent.md`) reduces the surface area
  of mechanism prescription, and therefore reduces pause
  frequency on mechanism mismatches.
- The defensive-scan discipline (ADR 0014) localizes one common
  trigger type to a predictable post-fix step rather than
  unpredictable mid-flight surprises.
- The executor exercises judgment for clearly trivial cases —
  pause-and-report is a discipline, not a contract for every
  diff line.

## Alternatives considered

- **Best-effort silent execution**. Rejected: Phase 4 evidence
  shows 10 cases where this would have shipped silent errors.
- **Architect verifies everything before prescribing**.
  Rejected: the architect lacks the executor's depth-and-freshness
  view of code reality; verification ahead-of-time is
  prohibitively expensive and still incomplete.
- **A meta-ADR bundling pause-and-report, defensive-scan, and
  wave-closure**. Rejected (owner direction, P5.5): each pattern
  has distinct trade-offs and warrants its own supersedure
  trajectory; small ADRs match repo convention.

## References

- `.claude/agents/architect-agent.md` — architect-side role
  description with the Phase 4 mistake catalog.
- `.claude/agents/execution-agent.md` — executor-side role with
  the canonical 5-step procedure.
- `.claude/skills/fix-prompt-skill/SKILL.md` — procedural
  knowledge for change-applying prompts (P5.1).
- `.claude/skills/investigation-skill/SKILL.md` — procedural
  knowledge for read-only investigations (P5.1).
- `docs/CONTRIBUTING.md` → "For AI agents" → "Pause-and-report
  protocol (5 steps)" — table of real Phase 1-3 catches.
- ADR 0014 (defensive-scan discipline).
- ADR 0015 (wave-closure discipline).
