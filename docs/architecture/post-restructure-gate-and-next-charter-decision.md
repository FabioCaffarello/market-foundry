# Post-Restructure Gate and Next-Charter Decision

**Date:** 2026-03-20
**Stage:** S222
**Scope:** Formal gate review after the S217–S221 restructure tranche
**Verdict:** Expansion charter should **not** open yet; execute one last short consolidation tranche first

---

## 1. Executive Summary

The restructure tranche delivered real structural value, but it did **not** fully close the post-S216 gate in formal terms.

What is true:
- S217 removed the ambiguity from the S216 review.
- S218–S220 completed the three deferred HIGH structural items: H-01, H-04, H-06.
- S221 reconciled the core gate documents to the post-restructure code state.
- The analytical/generated boundary is clearer and more defensible than before the tranche.

What is also true:
- The prior gate is still open on formal closure items: doc-count target, CI-on-push, and repository tag.
- The active documentation corpus is still drifting behind code in multiple active documents.
- `make check` currently fails because `raccoon-cli` still encodes pre-restructure assumptions about NATS file paths and store consumer actors.

**Decision:** the acceptable next step is **Option 2 — one last short consolidation tranche**. Opening a new evolution/expansion charter now would treat structural cleanliness as a feeling rather than a verified condition.

---

## 2. Formal Assessment of the Post-S216 Tranche

### 2.1 Was the previous gate really closed?

**Answer: No, not formally. Yes, structurally in part.**

The tranche closed the *structural* part of the remaining HIGH debt, but it did not close the *formal gate*.

| Gate item | State at S217 | State after S221/S222 review | Assessment |
|-----------|---------------|-------------------------------|------------|
| XC-1: active architecture docs ≤150 | FAIL at 243 | Still FAIL; workspace currently has 251 active architecture docs before S222 outputs | Not closed |
| XC-6: CI verified on real push | PENDING | Still PENDING | Not closed |
| XC-11: repository tag | PENDING | Still PENDING | Not closed |
| Structural HIGH items H-01/H-04/H-06 | Open after S217 | Completed in S218–S220 | Closed |
| Documentation reconciliation for tranche docs | Open after S220 | Completed in S221 for core gate docs | Closed for core docs only |
| Local guard rails aligned to new structure | Assumed | **Not aligned**: `make check` fails against stale paths/types | Not closed |

**Conclusion:** the prior gate is **closed as an evidence ambiguity problem**, but **not closed as an exit gate**. The Foundry is cleaner; it is not yet formally clear to re-open expansion.

### 2.2 Evidence reviewed

The S222 decision was based on direct repository evidence:
- `go.work` shows **17** workspace modules, confirming the S220 simplification.
- `internal/adapters/nats/` is now organized into domain packages (`natsconfigctl`, `natsdecision`, `natsevidence`, `natsexecution`, `natskit`, `natsobservation`, `natsrisk`, `natssignal`, `natsstrategy`), confirming H-01.
- `internal/actors/scopes/store/store_supervisor.go` and `generic_consumer_actor.go` confirm store consumers now flow through `GenericConsumerActor`, confirming H-04 adoption.
- `cmd/writer/pipeline.go` and `internal/adapters/nats/natssignal/registry.go` still show the ownership markers and governed slices that S214 intended.
- `make check` fails because `tools/raccoon-cli/src/analyzers/*.rs` still hardcode pre-restructure paths such as `internal/adapters/nats/signal_registry.go` and deleted per-family consumer actor files.

---

## 3. Did H-01, H-04, and H-06 Produce Real Structural Value?

### 3.1 H-01 — NATS adapter sub-packaging

**Answer: Yes.**

**Real gain**
- The largest adapter area stopped being a flat namespace and is now navigable by domain.
- Shared transport concerns were separated into `natskit`, which is a cleaner inward dependency than repeating low-level helpers across domains.
- The package layout now matches the bounded-context split more honestly.

**Evidence**
- `internal/adapters/nats/natskit/`
- `internal/adapters/nats/natsevidence/`
- `internal/adapters/nats/natssignal/`
- `internal/adapters/nats/natsdecision/`
- `internal/adapters/nats/natsstrategy/`
- `internal/adapters/nats/natsrisk/`
- `internal/adapters/nats/natsexecution/`
- `internal/adapters/nats/natsconfigctl/`
- `internal/adapters/nats/natsobservation/`

**Limit**
- Tooling and docs were not fully updated to the new layout. The code got cleaner faster than the guard rails did.

### 3.2 H-04 — store actor migration

**Answer: Yes.**

**Real gain**
- The generic actor infrastructure now captures value instead of existing as unused scaffolding.
- Consumer-actor creation is centralized in one place (`declarePipelines()`), which materially lowers the blast radius of adding or modifying a pipeline.
- The old duplicated consumer-actor layer is gone.

**Evidence**
- `internal/actors/scopes/store/generic_consumer_actor.go`
- `internal/actors/scopes/store/store_supervisor.go`
- Deleted per-family consumer actor files in `internal/actors/scopes/store/`

**Limit**
- Projection actors remain intentionally domain-specific, so the actor layer is cleaner but not generalized end-to-end.
- The guard rails still expect deleted per-family actor files.

### 3.3 H-06 — module graph simplification

**Answer: Yes, but with bounded magnitude.**

**Real gain**
- Two low-value module boundaries were removed with no new dependency edges.
- `internal/migrate` and `internal/adapters/repositories` were absorbed into more natural homes, making the workspace easier to reason about.

**Evidence**
- `go.work` now lists 17 modules.
- `cmd/migrate/migrate/` exists.
- `internal/application/configctl/memoryrepo/` exists.

**Limit**
- This was a pragmatic trim, not a deep monorepo redesign.
- Several large modules are still correctly isolated and should not be merged without new evidence.

### 3.4 Overall judgment on the three items

**They generated real structural value.** None of the three looks cosmetic. The problem is not that the tranche failed; the problem is that the tranche was not carried through the final layer of operational proof and canonicalization.

---

## 4. Analytical/Generated Path Sustainability

### 4.1 Did the path become more sustainable?

**Answer: Yes, but only moderately more sustainable.**

Why it improved:
- Ownership markers remain explicit in code.
- Mixed files are clearer about what codegen governs and what humans own.
- The path boundaries are still recognizable in `cmd/writer/pipeline.go` and `internal/adapters/nats/natssignal/registry.go`.

Why it is not fully “settled”:
- Active architecture docs still describe pre-H-01 paths such as `internal/adapters/nats/signal_registry.go`.
- `codegen-boundaries-and-governance.md` still documents the old `BEGIN/END CODEGEN MANAGED SECTION` marker format as if it were the operative integration protocol.
- `analytical-vs-generated-ownership-and-boundaries.md` and `analytical-generated-path-consolidation.md` still cite deleted or moved files.
- `cmd-migrate-and-migration-catalog.md` and `migrations-infrastructure-architecture.md` still describe `internal/migrate/` as an active module.

### 4.2 S222 assessment

The analytical/generated path is **conceptually cleaner** than before the tranche, but the documentation and tooling around it are **not yet synchronized enough** to call it a fully sustainable foundation for a new charter.

---

## 5. Documentation Coherence with Code

### 5.1 Did the main documentation become coherent with code?

**Answer: Core gate docs mostly yes; active corpus overall no.**

### 5.2 What is coherent

These documents match the current architectural direction and were useful during the review:
- `post-restructure-documentation-reconciliation.md`
- `h04-actor-migration-completion.md`
- `h06-module-graph-simplification.md`
- `module-graph-before-and-after.md`
- `exit-gate-closure-and-evidence-reconciliation.md`

### 5.3 What still drifts

Examples of active-document drift that materially matter:

| Area | Drift |
|------|-------|
| Canonical counts | `documentation-canonical-map-after-consolidation.md` says 249 architecture docs / 219 stage reports, but the workspace currently has 251 architecture docs and 218 stage files before S222 outputs |
| Analytical/generated docs | Several docs still point to pre-H-01 registry paths and pre-H-06 migrate layout |
| Codegen governance docs | Old marker protocol and old target paths are still documented in active docs |
| Migrate architecture docs | Active docs still describe `internal/migrate/` as live architecture |
| Gate narrative docs | `post-refactor-and-documentation-exit-gate.md` still contains pre-S221 narrative sections saying NATS flattening and module simplification were “not addressed” |

### 5.4 Documentation judgment

The documentation is **good enough to understand what happened**, but **not clean enough to claim the tranche is fully reconciled**. The remaining drift is not historical trivia; it affects current navigation, tooling expectations, and gate confidence.

---

## 6. Open Debts and Deferred Items

### 6.1 Must be closed before opening a new charter

1. **Guard-rail drift after restructure**
   `make check` fails because `raccoon-cli` still expects deleted file paths and actor types.

2. **Gate mechanics still open**
   CI-on-push and repository tagging remain unclosed.

3. **Documentation corpus still not canonically reconciled**
   Counts drift, active docs still reference deleted paths, and the doc-count target remains materially open.

### 6.2 Open but deferrable beyond the consolidation tranche

1. **M-01 through M-07**
   These remain medium-priority structural items, but they should not be mixed into the consolidation tranche.

2. **Golden snapshot equivalence debt**
   Still documented; not the right blocker for the post-restructure gate.

3. **Potential deeper module consolidation**
   Not justified yet.

### 6.3 Explicitly not hidden

- The tranche increased clarity but did not reduce documentation count enough.
- The tranche improved structure but left guard rails stale.
- The tranche removed duplication but did not prove the new structure through a clean quality-gate baseline.

---

## 7. Objective Recommendation for the Next Charter

### Recommended next step

**Option 2 — execute one last short consolidation tranche before opening any new evolution/expansion charter.**

### Why Option 1 is premature

Opening a new charter now would accept the following as “good enough”:
- failing local architectural guard rails,
- stale active architecture docs,
- unresolved formal exit items from the previous gate.

That would undermine the entire strategic purpose of the restructure tranche.

### Why Option 3 is too strong

The blockers are real, but they are also specific and bounded. This is not a case for indefinite pause. It is a case for disciplined closure work.

### Required scope of the consolidation tranche

1. **Update `raccoon-cli` for the post-H-01/H-04/H-06 architecture**
   It must stop expecting deleted flat NATS files and deleted per-family consumer actors.

2. **Reconcile active docs that describe current architecture**
   Especially analytical/generated governance, migration architecture, canonical counts, and gate narrative docs.

3. **Close the formal gate mechanics**
   Run and verify CI on push, then create the phase-exit tag.

4. **Re-measure and explicitly disposition documentation entropy**
   Either hit the doc-count target or formally redefine the target with evidence and rationale.

### Exit condition for opening the next charter

A new charter is acceptable only when:
- `make check` is green against the post-restructure layout,
- current active docs no longer depend on deleted paths as canonical references,
- CI-on-push and tagging are complete,
- the prior gate is explicitly marked closed without caveats.

---

## 8. Formal Disposition

| Question | S222 answer |
|----------|-------------|
| Was the previous gate really closed? | **No, not formally. Structurally it advanced; formally it remains open.** |
| Did H-01, H-04, and H-06 create real structural value? | **Yes.** |
| Is the analytical/generated path more sustainable? | **Yes, but not yet fully reconciled operationally/documentally.** |
| Did the main documentation become coherent with code? | **Partially. Core gate docs yes; active corpus overall no.** |
| What is the acceptable next step? | **Option 2: one last short consolidation tranche.** |
