# 0026 — Claude Code hooks as enforcement layer for P2 and P9

## Status

Accepted. Delivered together with the implementing configuration
(`.claude/hooks/` + `.claude/settings.json`) in the same PR —
P7-compliant: no capability is declared before the code that
delivers it.

## Context

The 2026-06-09 harness audit (FASE 1) found that the two most
operationally dangerous Harvest principles had **zero mechanical
enforcement**:

- **P2** (raccoon read-only): `$RACCOON_REFERENCE_PATH` could be
  modified by any Write/Edit/Bash call; the invariant lived only in
  CLAUDE.md prose (audit finding P0-2).
- **P9** (no agent push to `main`, no self-merge, no hook bypass):
  branch protection guards the remote, but nothing on the agent
  side blocked `git push origin main`, `git commit --no-verify`, or
  `gh pr merge` before they hit the remote (finding P0-4). Worse,
  `fix-prompt-skill` actively *taught* "Push — to `origin/main`"
  (finding P0-1, fixed in the B1 PR).

This repository already formalized (ADR-0004, harvest principle P5)
that an invariant without static enforcement is an intention, not
an invariant. P2 and P9 were exactly that.

`.claude/hooks/` had been intentionally empty since P3.8 ("Claude
Code hooks remain exploratory; populated only when concrete needs
surface"). The audit is the concrete need surfacing.

## Decision

Populate `.claude/hooks/` with three hooks, wired via
`.claude/settings.json`:

1. **`raccoon-readonly-guard.sh`** (PreToolUse —
   Write/Edit/NotebookEdit/Bash): **deny** any file modification
   under `$RACCOON_REFERENCE_PATH`, and any Bash command that
   references the raccoon checkout (by path, by repo name, or by
   the env-var name) together with a write-capable token. `cp` out
   of the raccoon is also denied — P1 forbids copying raccoon files
   into the foundry. Read-only access (grep/cat/diff) passes.
2. **`p9-branch-guard.sh`** (PreToolUse — Bash): **deny** `git push`
   targeting `main` (any refspec form), **deny** `--no-verify` /
   `LEFTHOOK=0` hook bypass, **ask** on `gh pr merge` (owner
   decision 2026-06-09: "misto" — deny is the default fidelity to
   P9; ask on merge preserves the owner's in-the-moment agency).
3. **`session-start-reminder.sh`** (SessionStart): emits a compact
   orientation line pointing at RESUMPTION → "Fase Harvest" and
   naming the two enforced invariants.

Additionally, `settings.json` gains `permissions.deny` rules for
Write/Edit/NotebookEdit under the raccoon path — a second,
independent layer in front of hook execution.

### Posture

Conservative by design, mirroring the P5.3 precedent (post-commit
drift check shipped warn-only before any hardening was considered):

- Denials are **scoped to the agent's tool surface**. The human
  maintainer retains full agency: commands typed with the `!`
  prefix (or in a regular terminal) are not gated by these hooks.
- The text remains canonical: hooks enforce CLAUDE.md → Fase
  Harvest P1/P2/P9; they do not define them.
- False-positive escape: a denied-but-legitimate command is run by
  the human directly; the guard's reason string says so.

## Consequences

**Positive:**

- P2 and P9 move from "intention in prose" to "enforced by
  construction" — the failure mode ADR-0004 exists to prevent.
- A skill or prompt that drifts back toward the P0-1 anti-pattern
  ("push to origin/main") now fails mechanically instead of
  depending on the model noticing the contradiction.
- SessionStart reminder shrinks the cold-start window where a
  session acts before reading RESUMPTION.

**Negative / accepted costs:**

- Pattern-based Bash matching has false positives (e.g., a branch
  name containing `main`) and false negatives (novel write
  spellings). Accepted: hooks are defense-in-depth above branch
  protection and below the documented protocol, not a sandbox.
- Hook scripts are repo-tracked config that must evolve with
  Claude Code's hook API; `.claude/README.md` documents them and
  this ADR records intent if the API shifts.

## Alternatives considered

- **`permissions.deny` only, no hooks** — covers Write/Edit but
  cannot inspect Bash command strings (the riskiest surface for
  both P2 and P9). Rejected as insufficient alone; kept as layer 1.
- **Hard deny on `gh pr merge`** — most faithful to P9 but removes
  the owner's ability to delegate a merge interactively. Owner
  chose ask (decisão "misto").
- **Keep `.claude/hooks/` empty, rely on text + branch protection**
  — the status quo the audit scored P0. Rejected.

## References

- CLAUDE.md → "Fase Harvest" — P1, P2, P9 (the invariants enforced).
- [ADR-0004](0004-raccoon-cli-static-enforcement.md) — the
  "invariant without enforcement is intention" principle.
- [ADR-0016](0016-harvest-from-market-raccoon.md) — raccoon
  consultative read-only posture.
- `.claude/hooks/raccoon-readonly-guard.sh`,
  `.claude/hooks/p9-branch-guard.sh`,
  `.claude/hooks/session-start-reminder.sh`,
  `.claude/settings.json` — the implementation.
- `.claude/hooks/test-guards.sh` — the 13-scenario decision matrix
  (11 regression + 2 heredoc false-positive shapes), committed so
  the validation is reproducible; run it after any guard change.
- `lefthook.yml` — git-side hooks (pre-push `make verify` enabled in
  the same PR, absorbing the P5.8 minimal posture).
- Harness audit FASE 1 (2026-06-09) — findings P0-1, P0-2, P0-4.

## Errata

### 2026-06-13 — Owner delegation of self-merge for the PROGRAM-0005 loop

The owner (Fabio Caffarello) explicitly authorized the agent to run an
**autonomous wave loop** for PROGRAM-0005 (Fase Insights): implement →
open PR → monitor the required checks → **squash-merge** → open the
next wave (H-8.a.1 → H-8.b → onward). This is a **scoped, explicit
override of P9's "no agent self-merge"** for this loop's PRs — *not* a
blanket rescission of P9.

Crucially, **the hook posture does not change**: `p9-branch-guard.sh`
already **asks** (not denies) on `gh pr merge` — the very "misto"
decision recorded above (2026-06-09) that "preserves the owner's
in-the-moment agency". The delegation simply means the agent *attempts*
the squash-merge and the owner *answers that ask with allow* for this
loop's PRs. `git push origin main` and `--no-verify`/`LEFTHOOK=0`
bypass remain **denied** — the agent pushes only feature branches and
merges via `gh pr merge --squash` (linear history is required on
`main`). Branch protection still gates the merge on the three required
checks (Unit Tests, Repository Consistency & Quality Gate, Go Lint).

Agent merge discipline for the loop (self-imposed, beyond green
checks): (1) confirm the required checks pass, (2) **diff self-audit**
— read the diff, verify scope and no out-of-scope files, (3)
`gh pr merge --squash`, (4) sync `main` + clean baseline, (5) open the
next wave. See also CLAUDE.md → "Fase Harvest" P9 (note appended).

### 2026-06-13 — Re-confirmation: delegation extended to PROGRAM-0006 (Delivery)

PROGRAM-0005 (Insights) closed; the owner re-confirmed the **same
autonomous self-merge loop** for the new Fase **PROGRAM-0006 (Delivery
WS)**. Each scoped grant is per-Fase, not standing — this errata records
the extension to the Delivery Fase. Same posture and discipline as the
PROGRAM-0005 entry above (hook "ask"-on-merge unchanged; `git push
origin main` + bypass remain denied; the five-step merge discipline
applies). The next Fase after Delivery again requires re-confirmation.

> **Note (2026-06-13 — H-11.c merged by owner; H-11.d re-confirmed):**
> the owner merged PR #57 (H-11.c) themselves, which closed PROGRAM-0006.
> They then chose a **hardening increment H-11.d** (the `check delivery`
> analyzer) which **reopens PROGRAM-0006**, and re-confirmed the same
> autonomous self-merge loop for it. The per-Fase grant therefore covers
> H-11.d under the (reopened) PROGRAM-0006; a genuinely new Fase still
> requires fresh re-confirmation.

### 2026-06-15 — Re-confirmation: H-11.e (Delivery hardening, reopens PROGRAM-0006)

After PROGRAM-0007 H-9 closed, the owner chose another Delivery hardening
increment **H-11.e (max-sessions cap)**, which **reopens PROGRAM-0006**,
and re-confirmed the same autonomous self-merge loop. Per-Fase grant;
snapshot-then-delta is deferred to H-11.f. Same posture/discipline as
prior entries; the next genuinely new Fase requires fresh re-confirmation.

### 2026-06-14 — Re-confirmation: delegation extended to PROGRAM-0007 (Storage Tier)

PROGRAM-0006 (Delivery) closed (H-11.a–d). The owner chose the next Fase
**PROGRAM-0007 (Storage Tier)** and re-confirmed the **same autonomous
self-merge loop**. Scope is the ADR-0023-compliant **Stage 1 closure +
trigger instrumentation** (no TimescaleDB — the agent paused-and-reported
the trigger-gate conflict per P6 and the owner chose this path). Same
posture and discipline as the prior entries (hook "ask"-on-merge
unchanged; `git push origin main` + bypass remain denied; five-step merge
discipline). The next Fase again requires re-confirmation.

## Changelog

- 2026-06-09 — Created and Accepted (harness FASE 2 B2 PR, with
  implementation).
- 2026-06-13 — Errata: owner delegated self-merge to the agent for the
  PROGRAM-0005 autonomous loop (scoped P9 override; hook "ask"-on-merge
  posture unchanged). Recorded with the H-8.a.1 docs commit.
- 2026-06-13 — Errata: delegation re-confirmed + extended to PROGRAM-0006
  (Delivery WS) after PROGRAM-0005 closed. Per-Fase grant; not standing.
- 2026-06-13 — Note: owner merged H-11.c (PR #57, closed PROGRAM-0006);
  H-11.d hardening increment reopens PROGRAM-0006 with the same delegation
  re-confirmed.
- 2026-06-14 — Errata: delegation re-confirmed + extended to PROGRAM-0007
  (Storage Tier, Stage-1 closure + trigger instrumentation). Per-Fase.
- 2026-06-15 — Errata: H-11.e (Delivery hardening, max-sessions cap)
  reopens PROGRAM-0006; same delegation re-confirmed. Per-Fase.
