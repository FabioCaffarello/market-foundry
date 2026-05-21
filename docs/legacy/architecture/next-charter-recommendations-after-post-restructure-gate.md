# Next-Charter Recommendations After the Post-Restructure Gate

**Date:** 2026-03-20
**Stage:** S222
**Status:** Recommendation document
**Primary recommendation:** Complete one last short consolidation tranche before opening a new charter

---

## 1. Executive Summary

After S217–S221, the Foundry is **closer** to a clean next charter, but it is **not there yet**. The deciding evidence is not the code cleanup itself; it is the mismatch between the cleaned-up code and the still-stale proof surfaces around it.

The next charter should therefore **not** be an expansion charter yet. It should be a short consolidation charter whose sole purpose is to finish the post-restructure closure work.

---

## 2. Option Review

### Option 1: Open a new evolution/expansion charter now

**Decision:** Not recommended.

**Why this is premature**
- The previous gate is still formally open.
- `make check` is not green against the current architecture.
- Active docs still contain live-path drift.
- CI-on-push and phase-exit tagging are still unresolved.

**Risk if chosen now**
- The next charter would inherit unresolved proof drift from the last one.
- New work would mix real product evolution with guard-rail repair, obscuring whether the base is actually ready.

### Option 2: Execute one last short consolidation tranche

**Decision:** Recommended.

**Why this is the right move**
- The blockers are known, bounded, and directly traceable to the restructure tranche.
- The code already contains the structural gains; the missing work is reconciliation and proof.
- This keeps strategic discipline: the expansion restart depends on evidence, not sentiment.

### Option 3: Pause until blockers are closed

**Decision:** Not the default recommendation, but a fallback if the consolidation tranche cannot restore a green proof surface quickly.

**When to escalate to pause**
- If guard-rail/tooling updates reveal deeper structural contradictions.
- If CI-on-push exposes regressions not visible locally.
- If the documentation corpus cannot be reconciled without reopening architectural scope.

---

## 3. Recommended Consolidation Tranche

### 3.1 Scope

The tranche should stay narrow and non-expansive.

1. **Guard-rail reconciliation**
   Update `raccoon-cli` analyzers and any related validation rules so they reflect:
   - domain-scoped NATS packages,
   - generic store consumer actors,
   - the 17-module workspace.

2. **Active-doc reconciliation**
   Update active docs that still describe deleted paths, old marker protocols, or stale counts.

3. **Formal gate closure**
   Run CI on push and create the formal phase-exit tag.

4. **Documentation entropy disposition**
   Either:
   - reduce the active-doc count materially toward the target, or
   - explicitly reset the target with evidence and rationale.

### 3.2 Explicit non-goals

- No new families
- No new services
- No new domains
- No new codegen scope
- No medium-debt execution
- No product expansion work

### 3.3 Exit criteria for the consolidation tranche

The next expansion charter becomes acceptable only when all of the following are true:

1. `make check` passes against the post-restructure codebase.
2. The core active docs no longer depend on deleted paths as canonical references.
3. CI is green on a real push.
4. The phase-exit tag is created.
5. The documentation target is either met or redefined explicitly with an accepted rationale.

---

## 4. What the Next Real Expansion Charter Should Look Like

Once the consolidation tranche closes, the next charter may legitimately reopen evolution work. When that moment comes, it should:

1. Start from a clean proof surface.
2. Declare a fresh entry/exit gate instead of inheriting S211/S216 leftovers.
3. Use the updated guard rails as the baseline.
4. Pick one bounded growth direction rather than reopening multiple fronts at once.

Good candidate directions after consolidation:
- next family/domain expansion,
- analytical/generated scope expansion,
- another bounded product-slice charter.

Bad candidate direction:
- mixing consolidation leftovers with new functional expansion in the same charter.

---

## 5. Formal Recommendation

| Option | Disposition | Reason |
|--------|-------------|--------|
| 1. Open new evolution/expansion charter now | **Reject** | Base is cleaner, but proof surface is still stale |
| 2. Execute one last short consolidation tranche | **Recommend** | Bounded closure work remains and should be finished first |
| 3. Pause until blockers close | **Fallback only** | Use only if consolidation reveals deeper contradictions |

**S222 recommendation:** choose **Option 2** now, and only open the next expansion charter after that tranche exits cleanly.
