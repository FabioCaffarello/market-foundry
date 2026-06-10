---
name: wave-prompt-skill
description: Wave-cycle pattern for the market-foundry Harvest program — producing wave/sub-wave prompts, running pré-flights, auditing wave PRs, and closing waves. Activates when the user asks to "open a wave", "produce a wave prompt", "prompt de onda", "pré-flight", "close the wave", "closure", "audit the PR", or references an onda/sub-onda identifier (H-N, H-N.x). Codifies the cross-check protocol (prescribe only against literal current-main file content), the mea culpa discipline (explicitly acknowledge corrected prescriptions), the proven wave-prompt anatomy, and the merge-gated closure sequence (P3/P4/P9).
---

# Wave prompt skill

The wave prompt is the most-produced artifact of the Harvest
program: the architect's scoped instruction set that an executor
turns into a wave's commits. This skill codifies the full cycle
around it. The general fix-prompt anatomy lives in
`fix-prompt-skill`; this skill adds what is specific to waves.

## The wave cycle

```
pré-flight (read-only, current main)
  → cross-check (literal file content, never memory)
  → wave prompt (anatomy below)
  → execution (executor; pause-and-report exchanges)
  → PR audit (architect reads the diff, not just the report)
  → closure (RESUMPTION + PRD + gates GREEN, same PR)
  → merge by human maintainer (P9)
  → only then: next wave opens (P4, P9 extension)
```

No step is skippable. The sequence is the discipline (P3).

## Cross-check protocol

**Before producing a wave prompt, obtain the literal content of
every file the prompt prescribes against, at current `main`.**
Request it from the executor session, or Read it directly. Never
prescribe from memory of how a file looked in a previous wave.

Why this is non-negotiable: between H-6.b and H-6.d the
cross-check eliminated four consecutive prescription
inconsistencies that working-from-memory had produced. The
architect's confidence in remembered file state is inversely
correlated with its accuracy after another wave has merged
(same mechanism as the Phase 4 mistake catalog —
`architect-agent.md`).

Concretely, a wave prompt's prescriptions must cite:

- File paths verified to exist on `main` at prompt-writing time.
- Current signatures/columns/policy entries quoted from the real
  file, not paraphrased.
- Counts (sites, tests, exception-list entries) recomputed, not
  carried forward from the previous wave's closure narrative.

## Pré-flight

A read-only pass over the wave's declared surface before the
prompt is finalized:

- Enumerate the migration/change sites (grep-grounded, with the
  command used recorded in the prompt so the executor can re-run
  it).
- Scan for the known blind spots: tagged-build test files
  (`//go:build requireclickhouse`, `integration`) and positional
  INSERTs/rows (H-6.d.1 lessons #1 and #2).
- Check the flake registry (RESUMPTION → G6, G7, ...) so known
  flakes are pre-declared rather than rediscovered mid-wave.

Cascade discovery at pré-flight (scope materially larger than the
PRD assumed) is a pause-and-report to the owner, possibly
splitting the wave (precedent: H-6 → a/b/b'/b''/c/d/e/f).

## Wave prompt anatomy

The proven structure (every H-wave since H-0):

1. **Contexto** — where this wave sits in the PRD; what the
   previous wave delivered; what this one closes.
2. **Decisões já tomadas** — numbered (Decisão #1, #2, ...), with
   the owner's chosen option recorded. The executor cites these
   in commit messages.
3. **Pré-condições** — previous wave merged in `main` (P4/P9),
   working tree clean, gates GREEN, flake registry consulted.
4. **Escopo IN / OUT / NÃO MUDAR** — explicit boundaries; OUT
   entries carry the destination wave (e.g., "helper deletion:
   H-6.f").
5. **Protocolo** — pause-and-report triggers specific to this
   wave, plus the standing ones (premise mismatch, cascade
   discovery, gate failure).
6. **Execução** — commit-by-commit plan. Prescribe scope and
   decision criteria; defer mechanism (exact lines, regexes,
   schema syntax) to the executor against current code.
7. **Critérios de aceitação** — independently verifiable; always
   include gates GREEN + RESUMPTION updated in the closure
   commit + no out-of-scope file modified.
8. **Como reportar** — STATUS / per-commit breakdown / validation
   / surpresas / pause-and-report log.

## Mea culpa discipline

When pré-flight evidence, cross-check, or the executor's
pause-and-report contradicts something the architect prescribed,
the next artifact (revised prompt, erratum, or report response)
**names the mistake explicitly**: what was prescribed, what
reality showed, and the corrected prescription. Real cases:
NULL vs `DEFAULT ''` column semantics, exception-list counts,
helper-deletion timing vs the TTL window.

Why: silent correction destroys the calibration signal. The
Phase 4 mistake catalog only exists because mistakes were
acknowledged when caught; that catalog is why "defer mechanism
to executor" is institutional rather than anecdotal. An
acknowledged correction also tells the executor their
pause-and-report was load-bearing — which keeps the protocol
alive (ADR-0013).

Prime convention (`a` → `a'`) names the recovery when the
correction warrants a revised sub-prompt — see
`architect-agent.md` → "Prime convention".

## Closure checklist

The closure commit (last commit of the wave PR) must leave:

- [ ] All planned commits landed (consolidations documented).
- [ ] Pre-push sequence GREEN: `make verify` always;
      `quality-gate --profile ci` if policies/analyzers changed;
      `make test-integration` if actors/adapters/execution path
      changed (CONTRIBUTING → "Pre-push validation").
- [ ] RESUMPTION: wave table row updated + "Entregas" section +
      next-wave pointer ("abre APENAS após merge").
- [ ] PRD (PROGRAM-NNNN): wave status, criteria checkboxes.
- [ ] TRUTH-MAP: anchors updated if a capability claim changed.
- [ ] ADR promotion only if the literal criteria are met in this
      PR (P7; precedent: promote in the same commit that delivers
      the last criterion).
- [ ] Next wave does NOT start — it opens after the maintainer
      merges this PR (P4, P9).

## Common pitfalls

- **Prescribing from memory** — the cross-check exists because
  this failed four times in a row. No exceptions for "small"
  files.
- **Carrying counts forward** — site counts go stale across
  waves; recompute at pré-flight.
- **Prompt without wave-specific pause triggers** — generic
  triggers miss the wave's known risk (e.g., "if the analyzer
  exception list shrinks by other than 3, pause").
- **Closure without RESUMPTION** — the post-commit drift check
  warns, but the gate is the closure commit itself.
- **Opening the next wave on local completion** — P9 extension:
  merge in `main` is the gate, not "done locally".
- **Silent correction** — see mea culpa discipline above.

## Cross-references

- `.claude/agents/architect-agent.md` — role expectations
  (cross-check + mea culpa as architect duties; prime
  convention; mistake catalog).
- `.claude/skills/fix-prompt-skill/SKILL.md` — general
  change-prompt anatomy this skill specializes.
- `.claude/skills/investigation-skill/SKILL.md` — pré-flight
  procedure when the wave needs a scoped investigation first.
- `.claude/commands/pre-push.md` — the canonical validation
  sequence required at closure.
- `docs/programs/README.md` — PRD convention; wave states.
- `CLAUDE.md` → "Fase Harvest" — P1-P9, the protocol this cycle
  implements.
