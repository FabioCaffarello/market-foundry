# Restructure Wave Gains, Trade-offs, and Open Debts

**Date:** 2026-03-20
**Stage:** S222
**Scope:** Gains, limits, and residual debt after S217–S221
**Purpose:** Separate structural value from residual gate debt before any new charter opens

---

## 1. Executive Summary

The restructure wave delivered the promised **structural** gains: cleaner NATS boundaries, real actor-layer duplication removal, a smaller and more justifiable module graph, and clearer analytical/generated ownership. It did **not** deliver a fully self-consistent operating baseline. The code moved further than the docs, guard rails, and formal gate closure.

The correct reading is:
- **Gains are real**
- **Trade-offs were accepted knowingly**
- **Open debts remain visible and must not be disguised as “readiness”**

---

## 2. Structural Gains

### 2.1 H-01: NATS adapter became structurally navigable

**Before**
- One large flat adapter area.
- Domain contracts and transport helpers shared the same namespace.
- Refactoring and review cost were higher than necessary.

**After**
- Domain adapters are separated into explicit packages.
- Shared transport mechanics are isolated in `natskit`.
- Current layout aligns better with the repo’s bounded contexts.

**Structural value**
- Lower cognitive load when working in a single domain.
- Cleaner package-level dependency reasoning.
- Better base for future bounded changes in signal/decision/strategy/risk/execution without reopening flat-namespace sprawl.

### 2.2 H-04: generic store consumer infrastructure now pays for itself

**Before**
- Store consumer actors repeated the same lifecycle and transport logic across families.
- The generic actor abstraction existed but did not reduce any real duplication.

**After**
- Store consumer actor creation is centralized via closures in `declarePipelines()`.
- The generic consumer path is the operative path, not dead scaffolding.
- The old consumer actor layer is deleted.

**Structural value**
- Lower blast radius when adding or adjusting a store pipeline.
- Clearer store runtime topology.
- Less glue code between registries, actors, and projection routing.

### 2.3 H-06: module graph is more honest

**Before**
- Some `go.mod` boundaries existed without corresponding isolation value.

**After**
- `internal/migrate` and `internal/adapters/repositories` were absorbed into more natural homes.
- External dependency isolation remained intact where it still mattered.

**Structural value**
- Fewer artificial module boundaries.
- Lower workspace maintenance cost.
- Better match between logical ownership and module ownership.

### 2.4 S214 remains a legitimate gain

Even though it predates the S218–S220 execution tranche, the analytical/generated consolidation remains part of the net gain:
- ownership markers are still present in live code,
- the mixed-file boundary is clearer than before,
- generated-vs-manual responsibility is easier to audit.

---

## 3. Trade-offs Accepted by the Wave

### 3.1 Code cleanliness moved faster than the guard rails

The tranche improved code structure without simultaneously updating the quality tooling. That is now visible in `make check`, which fails because `raccoon-cli` still expects:
- flat NATS registry files,
- deleted per-family store consumer actors,
- old contract locations.

**Assessment:** acceptable as a temporary tranche trade-off, unacceptable as a starting point for a new charter.

### 3.2 Documentation preservation beat documentation minimization

The repo preserved traceability by adding tranche and reconciliation docs instead of aggressively collapsing active docs. That preserved evidence, but it also kept the active corpus large and allowed stale references to persist in secondary active docs.

**Assessment:** acceptable during restructure, not acceptable as a permanent exit condition.

### 3.3 The wave improved architecture more than operational proof

The tranche delivered real code-level cleanup, but the remaining formal proof items were deferred:
- CI-on-push,
- repository tagging,
- guard-rail alignment,
- active-doc corpus cleanup.

**Assessment:** this is the single biggest reason the post-restructure gate should not be treated as “done”.

### 3.4 H-04 stopped at the right abstraction boundary

Projection actors were intentionally not generalized.

**Assessment:** correct trade-off. The wave removed mechanical consumer duplication without forcing false symmetry where domain logic still diverges.

---

## 4. Open Debts

### 4.1 Gate-closing debts

| Debt | Why it matters now | Current disposition |
|------|--------------------|---------------------|
| Active-doc count still materially above target | The prior gate remains formally open | Must close or re-baseline in consolidation tranche |
| CI not verified on push | Local confidence is not formal gate evidence | Must close in consolidation tranche |
| Repository tag missing | Exit remains undocumented operationally | Must close in consolidation tranche |

### 4.2 Tooling and guard-rail debts

| Debt | Evidence | Current disposition |
|------|----------|---------------------|
| `raccoon-cli` path assumptions are stale | `make check` fails against old NATS paths and deleted actors | Must close in consolidation tranche |
| Contract/topology analyzers are not restructure-aware | Failures point to files removed by H-01/H-04 | Must close in consolidation tranche |

### 4.3 Documentation debts

| Debt | Evidence | Current disposition |
|------|----------|---------------------|
| Canonical counts already drift again | Active docs/stage-file counts in active docs do not match current workspace | Must reconcile in consolidation tranche |
| Active docs still describe deleted paths | analytical/generated and migration docs reference removed layouts | Must reconcile in consolidation tranche |
| Gate narrative docs still contain pre-tranche claims | S216 exit-gate doc still says H-01/H-06 were “not addressed” in narrative sections | Must reconcile in consolidation tranche |

### 4.4 Medium structural debts that should stay deferred

M-01 through M-07 remain open, but they should **not** be bundled into the consolidation tranche. The tranche should stay focused on proof, reconciliation, and guard rails.

---

## 5. Net Assessment

### Gains
- Real reduction in structural sprawl.
- Real duplication removed.
- Better package and module boundaries.
- Better analytical/generated ownership visibility.

### Limits
- The wave did not finish its own proof surface.
- Tooling and active docs lag behind the code.
- Formal exit items remain open.

### S222 conclusion

The restructure wave should be judged as **architecturally successful but operationally unfinished**.

That is the correct strategic reading:
- not a failure,
- not a clean exit,
- not a green light for automatic expansion.

The residual work is now narrow enough that it should be handled in one short consolidation tranche rather than diluted into a new expansion charter.
