# Final Closure Success, Failure, and Stop Conditions

**Stage:** S223
**Date:** 2026-03-20
**Scope:** Success criteria, failure criteria, and stop rules for the final closure tranche

---

## 1. Success Conditions

The tranche succeeds only if all closure conditions below are satisfied together.

### 1.1 Success Condition Set

1. `make check` passes against the post-restructure codebase.
2. The active-doc drift identified in S222 is reconciled in the targeted canonical documents.
3. XC-1 is explicitly closed by one accepted path:
   - the target is achieved by bounded archival,
   - or the target is formally re-baselined with evidence and rationale.
4. CI is green on a real push for the closure baseline.
5. The repository tag is created on that validated baseline.
6. The final gate-close record states PASS cleanly and does not carry forward conditional language.

### 1.2 What Success Does Not Require

The tranche does **not** require:

1. M-01 through M-07 execution.
2. New product capability.
3. Broad documentation minimization beyond the targeted active drift and XC-1 decision path.
4. Deeper module consolidation than S220 already executed.
5. Re-opening analytical/generated scope design.

---

## 2. Failure Conditions

The tranche fails if any of the following occurs:

1. Guard-rail alignment cannot be restored without reopening architectural scope or reworking accepted runtime behavior.
2. Active-doc reconciliation expands into a broad corpus rewrite instead of a bounded closure action.
3. XC-1 remains ambiguous at the end of the tranche.
4. CI-on-push fails and exposes real regressions that require a new corrective charter rather than closure-only work.
5. The tag is created before the validated baseline exists.
6. The final report still needs caveats such as "mostly closed," "ready enough," or "close enough to PASS."

### 2.1 Failure Interpretation

If the tranche fails, the correct conclusion is not "open the next charter anyway."

The correct conclusion is:

1. the prior gate remains open,
2. the closure tranche was insufficient,
3. a narrower corrective scope or explicit pause is required before expansion.

---

## 3. Stop Conditions

Stop the tranche immediately and re-scope if any stop condition below is triggered.

### 3.1 Hard Stop Conditions

1. The work starts introducing new features, new routes, new services, new families, or new streams.
2. Tooling repair requires broad runtime refactoring rather than governance/tooling updates.
3. XC-1 can only be addressed by a repository-wide doc campaign instead of a bounded archival or formal re-baseline.
4. CI exposes a runtime regression outside the closure surface that cannot be fixed without reopening code scope.
5. The tranche starts absorbing M-01 through M-07 or any other medium-priority debt.

### 3.2 Soft Stop Conditions

Pause and reassess if any of the following occurs:

1. More active docs than the targeted set require substantial rewrites.
2. Additional analyzer families outside the known stale assumptions need redesign.
3. Tagging or CI workflow assumptions turn out to be undocumented or blocked by external process.
4. Evidence sources disagree again on counts, current paths, or closure status.

---

## 4. Go / No-Go Decision Rules

| Situation | Decision |
|---|---|
| `make check` is green and targeted docs are reconciled | Continue to operational proof |
| `make check` is green but XC-1 is still ambiguous | Do not push/tag; finish the doc-count disposition first |
| CI is green but local guard rails are still stale | Do not close the gate; local proof surface is still invalid |
| CI fails because of unrelated regression | Pause closure tranche and open corrective decision path |
| Tag exists without validated baseline | Treat as invalid closure evidence and correct before adjudication |

---

## 5. Risks of Deferring Closure Again

1. Conditional PASS becomes the de facto steady state.
2. Guard rails become optional in practice because they are known to be stale.
3. New charters inherit unresolved closure work and dilute accountability.
4. The evidence surface drifts further away from the already-completed restructure tranche.
5. Every added document worsens XC-1 without any explicit operating rule for the target.

---

## 6. Stop Rule for New Charter Discussion

No new expansion charter should be drafted, accepted, or implied until one of the following is true:

1. S227 closes the tranche as PASS, or
2. the tranche is formally declared failed and replaced by a new corrective gate decision.

Anything in between recreates the same ambiguity S223 was created to remove.
