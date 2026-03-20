# Stage S222 — Post-Restructure Gate and Next-Charter Decision Report

**Date:** 2026-03-20
**Type:** Formal gate review and decision stage
**Scope:** Evaluate the state of the Foundry after S217–S221 and decide the acceptable next charter
**Status:** COMPLETE
**Verdict:** One last short consolidation tranche is required before any new expansion charter opens

---

## 1. Executive Summary

S222 reviewed the restructure tranche honestly against code, documentation, and validation signals. The outcome is mixed but clear:

- The tranche delivered real structural value.
- The previous gate ambiguity introduced by S216 was removed by S217.
- The remaining HIGH structural items were actually executed in S218–S220.
- Core gate documentation was reconciled in S221.

But:

- the previous gate is still not formally closed,
- `make check` currently fails because the validation tooling still encodes pre-restructure assumptions,
- active documentation still contains meaningful current-state drift,
- the formal exit mechanics (CI-on-push, tag, doc-count disposition) remain open.

S222 therefore does **not** authorize automatic expansion restart.

---

## 2. Questions S222 Had to Answer

### 2.1 Was the previous gate really closed?

**Answer:** No, not formally.

S217 closed the evidence ambiguity. S218–S220 closed the outstanding HIGH structural work. S221 reconciled the core tranche documents. But XC-1, XC-6, and XC-11 remain open in substance, and the local quality gate is now out of sync with the restructured codebase.

### 2.2 Did H-01, H-04, and H-06 generate real structural value?

**Answer:** Yes.

- **H-01:** NATS is now domain-organized instead of flat.
- **H-04:** `GenericConsumerActor` is now the actual store consumer path.
- **H-06:** the workspace dropped from 19 to 17 modules with justified absorptions.

### 2.3 Is the analytical/generated path more sustainable?

**Answer:** Yes, but not fully reconciled.

The code-level ownership model remains clearer than before, but active docs and governance docs still lag behind the code layout and marker conventions.

### 2.4 Is the main documentation coherent with the code?

**Answer:** Partially.

The core gate and restructure docs are coherent enough to explain what happened. The broader active corpus is not yet coherent enough to serve as a clean canonical base for a new charter.

---

## 3. Evidence Collected

### 3.1 Direct code evidence

- `go.work` lists **17** modules.
- `internal/adapters/nats/` is organized into domain packages plus `natskit`.
- `internal/actors/scopes/store/store_supervisor.go` routes store consumers through closures into `GenericConsumerActor`.
- `internal/interfaces/http/handlers/analytical.go` still contains `parseAnalyticalParams()`, confirming the S217 reconciliation basis.

### 3.2 Documentation evidence

Documents that matched the current direction:
- `post-restructure-documentation-reconciliation.md`
- `h04-actor-migration-completion.md`
- `h06-module-graph-simplification.md`
- `exit-gate-closure-and-evidence-reconciliation.md`

Documents showing residual active drift:
- `documentation-canonical-map-after-consolidation.md`
- `analytical-generated-path-consolidation.md`
- `analytical-vs-generated-ownership-and-boundaries.md`
- `codegen-boundaries-and-governance.md`
- `cmd-migrate-and-migration-catalog.md`
- `migrations-infrastructure-architecture.md`

Count evidence collected during S222:
- Before S222 outputs: **251** active architecture docs, **218** stage files
- After publishing S222 outputs: **254** active architecture docs, **219** stage files

This confirms that XC-1 is still materially open and that even the stage-report corpus requires active counting discipline.

### 3.3 Validation evidence

`make tdd`
- **PASS**
- Confirmed the changed areas and recommended proofs.

`make check`
- **FAIL**
- The failure is not random; it is dominated by stale `raccoon-cli` assumptions that still look for:
  - flat NATS registry files such as `internal/adapters/nats/signal_registry.go`,
  - deleted per-family store consumer actor files,
  - pre-restructure contract locations.

This is direct evidence that the restructure tranche cleaned the code faster than it cleaned the guard rails.

`make verify`
- **FAIL**
- All Go test suites passed across the workspace modules, but the command still failed because the same `quality-gate` drift remains unresolved.
- This is the strongest local proof that the repo is structurally healthier than before, but not yet operationally closed for the next charter.

---

## 4. Gains and Trade-offs

### Gains

1. **Cleaner transport boundaries**
   The NATS adapter is substantially easier to navigate and reason about by domain.

2. **Real actor-layer duplication removal**
   The store consumer path is materially simpler and cheaper to extend.

3. **More honest workspace structure**
   Artificial module boundaries were removed without creating new dependency sprawl.

4. **Clearer ownership on the analytical/generated path**
   The code still shows the ownership model the wave intended to establish.

### Trade-offs

1. **Guard rails are stale**
   The local quality gate has not been updated to the new architecture.

2. **Documentation remains heavy and partially stale**
   Traceability was preserved, but active-corpus cleanup was not finished.

3. **Formal closure lagged structural execution**
   The tranche improved the architecture without finishing its own exit proof.

---

## 5. Open Debts and Deferred Items

### Must close before a new charter

1. Update `raccoon-cli` and related validation assumptions for the post-H-01/H-04/H-06 architecture.
2. Reconcile active docs that still describe deleted paths, old marker protocols, or stale counts.
3. Close CI-on-push and repository-tag items.
4. Explicitly dispose of the documentation-count target.

### Still deferred beyond that tranche

1. M-01 through M-07 remain medium-priority structural items.
2. Golden snapshot equivalence debt remains documented.
3. Any deeper module consolidation still requires new evidence.

---

## 6. Decision

### Accepted next step

**Option 2 — execute one last short consolidation tranche.**

### Rejected next step

**Option 1 — open a new expansion charter now.**

**Reason:** the base is structurally improved but not yet operationally or canonically closed.

### Not chosen now

**Option 3 — pause until blockers close.**

**Reason:** the blockers are specific and bounded enough to justify a short closure tranche first.

---

## 7. Deliverables Produced

S222 produced:

1. `docs/architecture/post-restructure-gate-and-next-charter-decision.md`
2. `docs/architecture/restructure-wave-gains-tradeoffs-and-open-debts.md`
3. `docs/architecture/next-charter-recommendations-after-post-restructure-gate.md`
4. `docs/stages/stage-s222-post-restructure-gate-and-next-charter-decision-report.md`

---

## 8. Final Disposition

The post-restructure tranche should be considered **architecturally successful, formally unfinished**.

That is the disciplined S222 outcome:
- do not celebrate the restructure as automatic readiness,
- do not reopen expansion on intuition,
- do not hide the remaining blockers,
- close the proof surface first,
- then open the next charter from a genuinely cleaner base.
