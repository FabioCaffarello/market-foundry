# Final Short Tranche Gains, Trade-offs, and Open Debts

**Date:** 2026-03-20  
**Stage:** S228  
**Purpose:** Record what the S223-S227 closure tranche actually delivered, what it did not deliver, and what still remains open before a new charter can start

---

## 1. Gains Confirmed by S228

### 1.1 Local guard rails recovered

Confirmed by S228:

1. `make check` — **PASS**
2. `make verify` — **PASS**

This is a real improvement over the S222 state, where the post-restructure baseline still failed its own fast guard rails.

### 1.2 Local analytical runtime restored to a coherent baseline

Confirmed by S228:

1. `make up` — **PASS**
2. `make seed` — **PASS**
3. `make smoke-analytical` — **PASS**

Concrete proof captured:

1. services healthy,
2. migrations present,
3. writer consuming events,
4. ClickHouse receiving candle rows,
5. gateway analytical endpoints serving `200`,
6. invalid analytical requests returning `400`,
7. no error-level compose logs during the proof run.

### 1.3 The S226 remote failure is now bounded

The short tranche did not erase the historical remote `FAIL`, but it did convert the failure class into a much narrower, mechanically explainable closure item.

This is strategically useful because the remaining work is now bounded rather than investigative.

### 1.4 The principal closure narrative is more honest

The repo no longer depends on “probably fixed” language for the local baseline.

S228 could reproduce the core S227 local claims directly instead of trusting only the stage texts.

---

## 2. Trade-offs Exposed by S228

### 2.1 Fast-profile success hid ci-profile failure

This is the main S228 trade-off.

The tranche restored a clean fast local gate, but did not fully reconcile the stricter CI profile:

1. `make check` passes,
2. `make quality-gate-ci` fails.

That means the tranche improved local confidence faster than it restored full governance symmetry.

### 2.2 Active-doc reconciliation stopped short of full current-state coherence

The tranche improved the main entry and governance docs, but S228 still found active docs with:

1. obsolete migrate path examples,
2. obsolete codegen marker examples,
3. obsolete registry target paths,
4. obsolete default-database narration.

The trade-off was scope discipline over full corpus completion.

### 2.3 XC-1 was re-baselined instead of structurally reduced

That choice was understandable inside the tranche, but it means the gate moved from “count target” to “current-state coherence.”

S228 shows that this new criterion only works if active-doc discipline remains strict. The corpus is still large, so small residues matter.

### 2.4 Remote proof was deferred behind local reconciliation

This was a reasonable sequencing choice, but it leaves the final decision still blocked by missing fresh remote evidence.

---

## 3. Open Debts After S228

### 3.1 Open debt A — `quality-gate-ci`

Still failing on the current baseline:

1. `topology-doctor`
2. `contract-audit`
3. `arch-guard`
4. `drift-detect`

Representative failure classes observed in S228:

1. missing `configctl.control.config` subject-prefix detection,
2. request/reply suffix symmetry mismatches,
3. event-registry alignment mismatches,
4. `cmd/` boundary findings,
5. stale naming-identity findings for `consumer` / `validator`.

This is a real blocker because it means governance tooling is still split between “fast says aligned” and “ci says not aligned.”

### 3.2 Open debt B — residual active-doc drift

S228 found remaining current-state drift in active docs including:

1. `docs/architecture/analytical-boundary-and-responsibility-model.md`
2. `docs/architecture/codegen-specification-and-schema.md`
3. `docs/architecture/codegen-current-usage-boundaries-and-limitations.md`
4. `docs/architecture/cmd-migrate-and-migration-catalog.md`

This is bounded debt, but it is still active-doc debt.

### 3.3 Open debt C — no fresh green remote run on the corrected baseline

The evidence log is present.

The proof itself is not.

Until a fresh push-run is green on the corrected baseline, XC-6 / EC-7 cannot be called PASS and XC-11 cannot be called justified.

---

## 4. Net S228 Judgment on the Tranche

The tranche produced real value:

1. it removed avoidable local noise,
2. it restored a coherent local analytical proof path,
3. it improved the closure narrative substantially.

The tranche did **not** finish the job:

1. full tooling reconciliation is incomplete,
2. active docs are not fully coherent,
3. final remote proof is still absent.

That is the correct S228 reading:

1. the tranche was useful,
2. the tranche was necessary,
3. the tranche was not sufficient for a clean pre-charter PASS.
