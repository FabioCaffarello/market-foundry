---
name: audit
description: Skeleton for investigation prompt — read-only audit of an area.
arguments:
  - name: area
    required: true
    description: The area to audit (e.g., 'CI workflow', 'docs/', 'scripts/').
---

Produce a structured investigation report of `$1`. **Read-only** —
make no changes during this audit.

## Precedent

This is the skeleton used for areas like:

- P1A.4a (runtime inventory before `docs/RUNTIME.md`).
- P1A.6a (domain inventory before `docs/domain/*`).
- P1D.3 (G6/drift_detect investigation).
- P2.X (scripts hygiene investigation).
- P3.0 (environment audit).
- P4.3 (30 min cap, used ~4): context propagation distribution
  across 88 sites, 4-axis categorization.
- P4.4 (45 min cap, used ~4.5): ControlGate fail-open framing +
  design discussion; outcome was ADR-0012.
- P4.5 (30 min cap, used ~4): Dependabot triage of 17 PRs;
  S/M/J/B × G/R/P categorization → 3 archetype waves.
- P5.0 (60 min cap, used ~7.4): Phase 5 environment audit; 12
  findings categorized P0/P1/P2/P3.

## Time cap

Every investigation declares an explicit wall-clock cap —
20/30/45/60 min by scope. The canonical table lives in
`.claude/skills/investigation-skill/SKILL.md` → "Time cap".

Cap exceedance is itself a finding — produce a report with what
was collected and surface gaps. Phase 4-5 investigations
consistently used 10-30% of the cap when scope was cleanly
bounded.

## Steps

1. **Setup**: write to `/tmp/audit-<sanitized-area>.md`.
2. **Inventory**: what is present (files, sizes, dates) — reuse the
   `/inventory <area>` command.
3. **State analysis**: how things currently work (config, deps,
   callers, invariants).
4. **Issue scan**: categorize findings by severity:
   - **P0**: critical (broken, exposure, blocking real work).
   - **P1**: solid improvement (high value, low risk).
   - **P2**: nice-to-have.
   - **P3**: out of scope or speculative.
5. **Options**: list 2–4 distinct paths forward (A/B/C/D), each with
   tradeoffs.
6. **Recommendation**: pick one with justification.
7. **Summary**: top findings + suggested next step.

## Critical

**Stop at recommendations.** Do not execute changes. Pause and report
to owner for direction.

If during the audit something looks like a critical issue
(security leak, broken-but-needed infrastructure, factual error in
canonical docs), surface it **immediately** with a HIGH severity tag
at the top of the report.

## See also

- `.claude/skills/investigation-skill/SKILL.md` — procedural
  knowledge for the investigation pattern (auto-loaded by Claude
  Code on semantic relevance; this command is the explicit
  invocation surface).
- `.claude/agents/architect-agent.md` — investigate-before-prescribe
  discipline (architect-side role).
- `.claude/agents/execution-agent.md` — pause-and-report protocol
  + defensive-scan discipline.
