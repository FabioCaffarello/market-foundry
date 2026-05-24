---
name: investigation-skill
description: Read-only investigation pattern for diagnosing market-foundry issues, scoping work before fixes, or building decision artifacts. Activates when the user asks to "audit", "investigate", "diagnose", "scope out", or "categorize" an area of the codebase, when a fix shape needs verification before commitment, or when accumulated context requires bucketing for prioritization. Codifies the Phase 4 institutional discipline of "investigate before prescribe" — produces a structured artifact at `/tmp/<id>-<topic>.md` with explicit categorization, open questions, and an A/B/C/D recommended-next-step framework, and is bounded by an explicit wall-clock time cap (20/30/45/60 min by scope).
---

# Investigation skill

Read-only diagnostic and decision-support work. Distinguished
from the fix-prompt skill (which applies changes) by the
boundary "no files modified, working tree clean at end".

## When this applies

- **Scope unclear** — "how many sites does this affect?", "is
  this a bug or a design decision?", "what's the categorization
  shape?".
- **Fix shape needs verification** — before committing to a fix
  approach, verify reality matches the prompt's premise. This
  is exactly where Phase 4 caught ~10 architect-prescription
  errata.
- **Categorization needed** — multiple items need bucketing for
  prioritization or treatment.
- **Owner-decision support** — produce a factual report so the
  owner can direct the next step.

## Structure

Every investigation has these sections, in this order:

### 1. Objetivo

State what this investigation will produce. The outcome is
bounded — "audit file with categorization", "list of sites
with severity", "recommendation framework". Never "I will fix
X" — investigations do not fix.

### 2. Time cap

Wall-clock budget. Phase 4 evolved these conventions:

| Scope | Cap |
|---|---|
| Abbreviated (binary categorization, one axis) | 20 min |
| Standard (broader survey, multiple axes) | 30 min |
| Wide (multi-axis with design framework) | 45 min |
| Comprehensive (environment-level audit) | 60 min |

If the budget is exceeded, **that itself is a finding** —
produce a report with what was collected and surface gaps for
follow-up. Do not silently keep digging.

### 3. Audit file

Always at `/tmp/<id>-<topic>.md`. Persistent artifact for
discussion. Concise but complete — typically 100-300 lines.
Header includes the time cap and a UTC timestamp. Re-readable
in a future session without conversation context.

### 4. Categorization framework

State explicit categories before findings. Examples from
Phase 4:

- **A/B/C/D severity** (P4.3 context propagation: real bug /
  defensible / architectural / unclear).
- **S/M/J/B + G/R/P** (P4.5 Dependabot triage:
  security/minor/major/breaking × green/red/pending).
- **ENV-HIGH/MEDIUM/LOW** (P5.0 audit: environment relevance).
- **P0/P1/P2/P3** (P5.0 audit severity: blocking / high-value /
  nice-to-have / speculative).

Frameworks force clarity. Avoid free-form findings.

### 5. "No files modified" emphasis

State the read-only invariant explicitly in both the Protocolo
and Escopo sections. Verify a clean working tree at the end.

### 6. Pause-and-report markers

Identify scenarios that should pause execution before completion:

- Unexpected scope (>10 items where 3 were expected).
- Architectural concern surfaces (not just categorization).
- Time cap exceeded.
- Categorization framework no longer fits the findings.
- Premise of the investigation itself is wrong.

When triggered, follow the canonical 5-step pause-and-report
protocol (see `docs/CONTRIBUTING.md` → "Pause-and-report
protocol").

### 7. Open questions for owner

Investigations produce decision artifacts. The owner needs to
direct the next step. The "Open questions" section enumerates
the trade-offs explicitly — typically 3-7 questions that fall
out of the findings.

### 8. Recommended next step

Multi-option framework:

- **A** — proceed with the dominant recommendation.
- **B** — defer pending data.
- **C** — different direction (smaller scope, or pivot).
- **D** — more investigation needed.

Recommend one with rationale. Architect or owner chooses.
The investigator recommends; it does not prescribe.

## Phase 4 examples (institutional precedent)

- **P4.1.5** (45 min): CI residual investigation. Found NATS
  infrastructure + Smoke Analytical issues. Time cap was
  unspoken at this point; the explicit-cap convention evolved
  shortly after.
- **P4.1.11** (20 min): Abbreviated writerpipeline investigation.
  First explicit hard cap. Decisive binary outcome (Category A
  vs B).
- **P4.3** (30 min, used ~4): Context propagation distribution.
  Categorized 88 sites across 4 axes. Reframed P0-3 from
  "propagate caller ctx" to "bound fresh Background".
- **P4.4** (45 min, used ~4.5): ControlGate fail-open
  discussion. Investigation + design framework hybrid; outcome
  was ADR-0012.
- **P4.5** (30 min, used ~4): Dependabot triage. 17 PRs
  categorized S/M/J/B + G/R/P → resolved as three archetype
  waves.
- **P5.0** (60 min, used ~7.4): Phase 5 environment audit.
  Categorized 12 findings as P0/P1/P2/P3; produced 8-slot
  P5.x candidate roadmap.

Pattern observed: investigations typically use **10-30%** of
the cap when scope is cleanly bounded. Cap exceedance signals
substantial complexity worth surfacing before continuing.

## Common pitfalls

- **Premature prescription** — investigation drifts into
  recommending a specific fix mechanism. Stop. Investigations
  produce categorization + open questions, not fix
  prescriptions.
- **Scope creep** — "while I'm here, let me also check X".
  Each expansion costs time budget. Pause-and-report instead.
- **Free-form findings** — no categorization framework, just
  narrative. Forces the owner to do categorization mental work.
  Always be explicit.
- **Missing audit file** — investigation reported inline, not
  persisted. Lost when the conversation moves on. Always
  produce a `/tmp/` artifact.
- **Convenient categorization** — reaching for a tidy
  explanation when evidence is thin. Investigate before
  adopting; see Phase 4 errata in `docs/CONTRIBUTING.md`.

## Anti-pattern: when this skill does NOT apply

- **Trivial questions** with a clear single answer — use a
  direct tool call, not an investigation prompt.
- **Already-decided fixes** where the shape is clear — use the
  fix-prompt skill instead.
- **Tight time pressure with high confidence** — skip
  investigation if >85% confident in fix shape; only
  investigate when uncertainty justifies the cost.

## Cross-references

- `.claude/commands/audit.md` — slash-command skeleton for
  invoking the same pattern interactively.
- `.claude/agents/investigation-agent.md` — agent role
  definition (descriptive companion).
- `docs/CONTRIBUTING.md` → "Audit and investigation patterns"
  — five Phase 4.1-derived patterns (deletions-in-disguise,
  analyzer logic vs output, static-enforcement vs intent,
  CI red status, process debt vs regression).
