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

## Phase 1+2 precedent

This is the skeleton used for areas like:

- P1A.4a (runtime inventory before `docs/RUNTIME.md`).
- P1A.6a (domain inventory before `docs/domain/*`).
- P1D.3 (G6/drift_detect investigation).
- P2.X (scripts hygiene investigation).
- P3.0 (environment audit).

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
