---
name: fix-prompt-skill
description: Pattern for market-foundry prompts that apply changes — code edits, configuration updates, documentation refresh, PR merges, dependency bumps. Activates when the user asks to "fix", "apply", "implement", "patch", "bump", "migrate", "refresh", "merge", or "land" a specific change with defined scope and verification approach. Codifies Phase 4 institutional discipline of atomic-per-concern commits, defensive scanning after primary fix, explicit bundle-vs-split decisions, pause-and-report at scope boundaries, structured commit messages, and post-push CI monitoring.
---

# Fix prompt skill

Applies changes to the codebase, configuration, or
documentation. Distinguished from the investigation skill by
the boundary "files will be modified, a commit will be made,
and the change will be pushed".

## When this applies

- A specific change has defined scope (often from a prior
  investigation).
- The verification approach is known.
- The outcome is a deliverable (commit + push), not an
  artifact (audit file).

## Structure

Every fix prompt has these sections, in this order:

### 1. Contexto

Why this change. Reference the investigation if the scope came
from one (e.g., "P5.0 audit identified F2 as top P1 finding").
Document the bundle decision explicitly (same-concern bundle
vs split-by-concern).

### 2. Bundle vs split decision

Two or more concerns may be **bundled** if:

- They share the same architectural concern semantically.
- The same fix recipe applies to each.
- Atomic-per-concern is preserved (one commit = one
  responsibility, even if the responsibility spans multiple
  files).

**Split** when:

- Concerns are different architecturally.
- Asymmetric complexity (one large, one trivial).
- Different decision criteria apply.

Always document the bundle decision explicitly. Authorize a
mid-flight split if execution reveals unexpected complexity.

### 3. Pré-condições

- Working tree clean (`git status --porcelain` empty).
- Up-to-date with `origin/main`.
- Baseline test (`make verify` or scoped equivalent).
- Predecessor work completed (if applicable).
- CI status known (do not start fix work on red).

### 4. Escopo — IN / OUT / NÃO MUDAR

Explicit boundaries. Easier to enforce later.

- **IN**: what this fix addresses.
- **OUT**: what is deliberately excluded (with rationale —
  often "scope of P5.X").
- **NÃO MUDAR**: what stays untouched (defensive guardrail).

### 5. Protocolo — pause-and-report triggers

Scenarios that should pause execution mid-fix:

- Unexpected complexity in the fix area.
- Defensive scan reveals broader scope than prompted.
- Verification step fails non-trivially.
- Architectural decision surfaces mid-execution.
- Prompt's premise is wrong (line numbers, schema, runtime
  semantics).

Phase 4 caught ~10 architect-prescription mistakes via these
triggers. Generous use is encouraged — the cost of pausing is
much less than the cost of unwinding silent expansion.

### 6. Execução

Step-by-step. Phase 4 evolved this canonical pattern:

1. **Inspect** — read current state before changing. Verify
   the prompt's premise (file paths, line numbers, surrounding
   context).
2. **Defensive scan** — verify the inventory; find missed
   sites adjacent to the prompted ones.
3. **Apply fix** — make changes using `Edit` (prefer) or
   `Write` (for new files).
4. **Verify** — run `make verify` (or scoped subset), build,
   lint.
5. **Commit** — structured commit message (see §7).
6. **Pre-push validation** — follow the canonical sequence in
   `docs/CONTRIBUTING.md` → "Pre-push validation":
   `make verify` always; `raccoon-cli quality-gate --profile ci`
   when analyzer policy files or analyzer source changed;
   `make test-integration` when actors, adapters, or the
   execution path changed. `make verify` GREEN alone does not
   imply CI green (H-6.c.1 lesson: PR #30 failed CI with 17+2
   errors despite local verify GREEN on every commit).
7. **Push** — to the dedicated feature branch
   (`feat/h-N-<slug>` or equivalent), **never** directly to
   `main`. Merge into `main` happens only via PR reviewed and
   merged by the human maintainer — P9, see `CLAUDE.md` →
   "Fase Harvest". Agents do not self-merge.
8. **Monitor** — watch CI for green; rerun or annotate
   intermittent flakes per documented posture.

### 7. Commit message conventions

Phase 4 evolved structured commit messages with:

- `<type>(<phase-id>): <one-line summary>` header
  (Conventional Commits style — `feat`, `fix`, `docs`, `chore`,
  `ci`, `test`, `refactor`).
- Context paragraph (why — often references the investigation
  that scoped the work).
- Changes section (what, file:line where precise).
- Validation section (proof — `make verify` exit, CI status,
  scope-specific tests).
- M-list updates if any new design-meta surfaced.
- Phase progress closure narrative.

Length scales with change significance. Substantive changes
get substantive messages.

### 8. Defensive scan discipline

After applying the primary fix, scan defensively for missed
sites. The canonical Phase 4 evidence table (defensive scan
caught additional sites in every inventory-grounded fix) lives in
`docs/decisions/0014-defensive-scan-discipline.md`.

**Default expectation**: defensive scan finds 1-3 additional
items. Surprise if scan finds 0 (may indicate the scan was too
narrow). The prime convention naming pattern (`a` → `a'`,
canonical notation in `.claude/agents/architect-agent.md` →
"Prime convention") signals that an architect-prescription error
was caught and a recovery sub-prompt followed.

### 9. Critérios de aceitação

Explicit checklist. Each item independently verifiable. No
vague success criteria — define what "works" means concretely.
Always include: working tree clean, `make verify` PASS, CI
green, no out-of-scope file modified.

### 10. Como reportar

Structured report. Phase 4 evolved these canonical sections:

- **STATUS** (SUCCESS / PARTIAL / FAILURE).
- Per-change breakdown.
- Validation results.
- CI run details (commit SHA, run ID, job-by-job status).
- **Surpresas** — unexpected findings, defensive-scan
  additions, etc.
- Working tree state.
- Phase status ASCII block (✓ closed, ◐ partial, ⏳ pending,
  ❌ failed).

## Phase 4 examples (instructive cases)

- **P4.2** (rate_limiter): 91 LOC + 10 tests + Close-lifecycle
  wiring. Single commit. Defensive scan caught all production
  wire sites.
- **P4.3.a** (context bounding): 14 sites bounded +
  `contextcheck` lint enabled. Defensive scan found 4
  additional sites (3 `//nolint` with rationale, 1 genuine
  improvement in `retry_submitter.isHalted`).
- **P4.4.a** (ControlGate ADR): ADR-0012 + Prometheus counter
  + 6 tests. Bundle decision: same concern (formalize +
  monitor).
- **P4.5.b** (Dependabot minor batch): 8 PRs merged
  sequentially. Mirror-pair handling (#10/#14, #13/#15) to
  avoid `go.sum` ping-pong.

## Common pitfalls

- **Silent scope creep** — "while I'm here, fix X too". Either
  scope expansion is authorized by prompt protocol, OR
  pause-and-report.
- **Missing defensive scan** — applies fix, doesn't verify
  completeness. Phase 4 found additional sites in nearly every
  fix.
- **Vague commit message** — "fix bug" with no context. The
  future archaeologist needs the *why*.
- **No CI monitoring** — push and walk away. CI flakes happen;
  rerun or annotate as needed.
- **Bundle without rationale** — three unrelated concerns in
  one commit. Splits commit history; makes reverting one
  concern harder.
- **Reframing reality to fit the premise** — if a prompt's
  premise doesn't match reality, do not reframe reality. Pause
  and report. See `docs/CONTRIBUTING.md` → anti-patterns.

## Anti-pattern: when this skill does NOT apply

- **Investigations** — use the investigation skill.
- **Discussions / design documents** — use the investigation
  skill or a free-form prompt.
- **Trivial chores** with no decision content — just do it;
  full ceremony is overkill (single-line typo fixes, version
  bumps under Dependabot).

## Cross-references

- `.claude/agents/execution-agent.md` — executor role definition
  (scope discipline, validation, anti-patterns).
- `docs/CONTRIBUTING.md` → "Pause-and-report protocol" —
  canonical 5-step procedure with Phase 1-3 catches table
  (institutional commitment: ADR-0013).
- `docs/CONTRIBUTING.md` → "Authorized expansion protocol" —
  procedure for legitimate mid-flight scope growth.
- `.claude/commands/check-clean.md` — pre-action working-tree
  verification shortcut.
- `.claude/commands/check-refs.md` — defensive cross-reference
  scan shortcut (use before deletions/renames).
