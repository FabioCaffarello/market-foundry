# Final Exit Criteria Closure Plan

**Stage:** S223
**Date:** 2026-03-20
**Scope:** Freeze the final short closure tranche required to convert the post-S216 / post-S222 verdict into a clean PASS
**Status:** Scope frozen for disciplined execution in S224-S227

---

## 1. Executive Summary

S223 does not open a new evolution charter. It freezes the last short tranche needed to close the previous one.

The tranche exists for one purpose only: convert the current state from "architecturally improved but formally unfinished" into a clean exit with reconciled criteria, current guard rails, current active docs, and recorded operational proof.

This tranche is therefore a **closure tranche**, not an expansion tranche:
- no new feature scope,
- no new structural ambition,
- no broad cleanup campaign,
- no medium-debt execution,
- no new charter language hidden inside "reconciliation."

---

## 2. Remaining Criteria and Their Origin

| Closure item | Why still open | Formal origin | Current evidence | Classification | Required action now |
|---|---|---|---|---|---|
| Guard-rail / tooling reconciliation | `make check` fails against pre-restructure assumptions about registry paths, per-family store consumers, and topology expectations | S222, Stage Definition of Done, Anti-Debt Checklist | `make check` fails on 2026-03-20 with stale `raccoon-cli` expectations | Tooling + governance | **Complete now** |
| Active-doc current-state reconciliation | Active docs still describe deleted paths, old marker protocol, stale counts, or pre-restructure narratives as if current | S221, S222 | Residual drift explicitly listed in S222 and confirmed by repo search | Docs | **Reconcile now** |
| XC-1 documentation-count disposition | The old target remains formally open and count discipline keeps drifting | S209, S211, S216, S217, S222 | `docs/architecture/` count was 254 before S223 outputs | Docs + evidence | **Dispose now** |
| XC-6 / EC-7 CI-on-push proof | Local checks are not formal exit evidence | S210, S211, S216, S217, S222 | No recorded green CI run on the closure baseline | Operational evidence | **Complete now** |
| XC-11 repository tag | Exit is still operationally unrecorded | S210, S211, S216, S217, S222 | No `refactoring-phase-exit` closure tag recorded for this gate state | Operational evidence | **Complete now** |
| Final gate reconciliation | The tranche improved the repo, but the prior gate is still not explicitly closed as PASS with reconciled evidence | S216, S217, S222 | Current disposition remains conditional / unfinished | Evidence + governance | **Reconcile now** |

### 2.1 What Must Be Concluded Now

The following items are execution work, not interpretation work:

1. `raccoon-cli` / `quality-gate` must become post-restructure-aware and produce a green `make check` baseline.
2. CI must run green on a real push against the closure baseline.
3. The repository must be tagged only after the validated closure baseline exists.

### 2.2 What Must Be Reconciled Now

The following items are canonicalization work and cannot be left implicit:

1. Active docs that still describe deleted paths, removed actor files, old marker protocol, or stale counts as present architecture.
2. XC-1 itself: either the doc-count target is reached by a bounded archival action, or the target is explicitly re-baselined with evidence and rationale. Leaving it "to be revisited later" is not an acceptable closure outcome.
3. The final gate narrative: the old gate must end either as PASS with evidence or as still blocked with one exact cause. Ambiguous wording is not allowed.

---

## 3. Closure Tracks

### Track A: Validation Tooling and Guard Rails

**Objective**
Bring `raccoon-cli` and `quality-gate` into alignment with the architecture already established by S218-S220.

**Bounded scope**
- analyzer path expectations,
- durable and topology inventories,
- registry discovery rules,
- store consumer model assumptions,
- module-graph assumptions where still encoded.

**Not included**
- new analyzer domains,
- stronger policy than already governed,
- runtime expansion,
- refactoring unrelated code just to satisfy tooling preferences.

### Track B: Active Docs and XC-1 Disposition

**Objective**
Reconcile only the active docs that still misstate the current architecture or exit state, and explicitly close XC-1 by one accepted path.

**Bounded scope**
- gate narrative docs,
- analytical/generated governance docs,
- migration architecture docs,
- canonical count docs,
- any active doc still treating deleted paths as live canonical references.

**Not included**
- full historical corpus rewrite,
- deep style unification,
- domain-wide content modernization unrelated to current drift,
- opportunistic archival outside the XC-1 decision path.

### Track C: Operational Exit Evidence

**Objective**
Produce the formal proof the prior gate still lacks.

**Bounded scope**
- push closure baseline,
- capture CI result,
- create the phase-exit tag on the validated commit,
- record the exact evidence identifiers.

**Not included**
- release engineering redesign,
- new pipeline work,
- CI feature expansion,
- unrelated tagging cleanup.

### Track D: Final Adjudication

**Objective**
Publish the final, short gate-close verdict after Tracks A-C complete.

**Bounded scope**
- final PASS or exact remaining blocker,
- evidence reconciliation,
- closure of conditional-pass inheritance from S216/S222,
- explicit statement that a new charter may or may not open.

**Not included**
- new roadmap,
- new capability charter,
- M-01 through M-07 reprioritization,
- broader architectural recommendations beyond the closure result.

---

## 4. Sequencing for the Last Short Tranche

### S224 — Tooling Baseline Recovery

1. Update `raccoon-cli` and related validation assumptions to the post-H-01/H-04/H-06 architecture.
2. Re-run `make check` until the closure baseline is green.
3. Record exactly which analyzer assumptions changed and why.

**Exit from S224**
- `make check` passes locally.
- No new runtime behavior was introduced beyond reflecting current architecture.

### S225 — Active-Doc Reconciliation and XC-1 Disposition

1. Reconcile the targeted active docs still describing deleted paths, stale counts, old marker protocol, or pre-restructure status as current.
2. Decide and record the XC-1 disposition:
   - either hit the count target through bounded archival,
   - or formally re-baseline the target with evidence and rationale.
3. Re-count the corpus after the doc action and record the exact numbers.

**Exit from S225**
- Targeted active docs are current.
- XC-1 is no longer ambiguous.

### S226 — Operational Proof

1. Push the closure baseline.
2. Verify CI-on-push passes.
3. Create the repository exit tag only after the green CI result is confirmed.
4. Capture commit SHA, CI run identifier, and tag name as evidence.

**Exit from S226**
- XC-6 and XC-11 are both closed in evidence, not only in intent.

### S227 — Final Gate Close

1. Reconcile the full evidence set from S224-S226.
2. Publish the short final gate-close record.
3. Mark the tranche either:
   - **PASS cleanly**, or
   - **still blocked** with one exact unresolved cause.

**Exit from S227**
- The prior tranche is no longer conditionally open.
- The repo has an explicit go / no-go basis for any subsequent charter.

---

## 5. What Counts as Sufficient Closure

The last short tranche is sufficiently closed only when all of the following are true:

1. `make check` is green against the post-restructure architecture.
2. The targeted active docs no longer depend on deleted paths, deleted actor files, stale counts, or superseded codegen markers as canonical references.
3. XC-1 is explicitly closed, either by meeting the target or by ratifying a new target with evidence and rationale.
4. CI is green on a real push for the closure baseline.
5. The repository tag exists on the validated closure baseline.
6. A final gate-close document states PASS without carrying forward conditional language.

If any one of these remains open, the tranche is not closed.

---

## 6. Out of Scope

The following remain explicitly outside this tranche:

1. New families, domains, services, streams, endpoints, or product slices.
2. M-01 through M-07 and any other medium-priority structural debt.
3. Broad module-graph redesign beyond what was already executed in S220.
4. Golden snapshot equivalence cleanup unless it is required as direct closure evidence.
5. General documentation modernization outside the targeted active-drift set.
6. New codegen scope, new governance features, or new analyzer ambitions.

---

## 7. Risks of Not Closing Now

If this tranche is not closed now, the repository carries avoidable strategic cost:

1. The CONDITIONAL PASS stays alive long enough to become the default operating state.
2. Guard rails lose credibility because they fail on architecture the repo already accepted.
3. Future charters inherit stale active docs and must spend scope re-proving old cleanup.
4. The doc-count target becomes progressively less meaningful as counts drift with every new document.
5. CI and tagging evidence stay detached from the structural tranche they are meant to close.

---

## 8. Preparation for S224

S224 should begin with a narrow execution brief:

1. Treat `make check` as the primary blocker surface.
2. Fix only validation assumptions that are stale relative to code already accepted in S218-S220.
3. Record every changed analyzer expectation in the stage report.
4. Do not begin doc reconciliation until the tooling baseline is green.
5. Do not discuss new charter options until S227 adjudicates the result.
