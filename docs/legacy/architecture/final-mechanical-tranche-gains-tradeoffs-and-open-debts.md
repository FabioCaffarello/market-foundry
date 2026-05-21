# Final Mechanical Tranche: Gains, Trade-offs, and Open Debts

**Tranche:** S229–S231
**Date:** 2026-03-20
**Purpose:** Close the gap between S228's honest assessment and clean-pass state

---

## 1. Tranche Scope

The S228 gate identified four concrete blockers preventing clean-pass:

1. `quality-gate-ci` failing with 40 errors (6 stale analyzer assumptions).
2. Four active-doc drift items in 3 files.
3. No fresh remote CI proof on the corrected baseline.
4. Release tag `v0.1.0-s231` blocked behind proof.

S229–S231 was chartered as a bounded mechanical correction tranche to close these four items — nothing more.

---

## 2. Gains

### 2.1 Quality-Gate Convergence (S229)

**Before:** `make quality-gate-ci` reported 40 errors. The CI profile and fast profile diverged.
**After:** Both profiles report 84 checks, 0 errors. Identical verdicts.

**Root cause:** Six raccoon-cli analyzer assumptions that predated the S218 restructuring:
- `topology.rs` — configctl control subject prefix too narrow
- `contracts.rs` — reply-type symmetry check not version-aware
- `contracts/events.rs` — hardcoded configctl-only event scanning
- `contracts.rs` — rigid suffix matching instead of domain-aware tokenization
- `drift_detect.rs` — "consumer" flagged as defunct service name (legitimate in writer)
- `runtime_bindings/source.rs` — stale doc comment and test

**Value:** The quality-gate tooling is now a reliable architectural guardian again. It will catch real regressions in future charters instead of generating false positives from stale assumptions.

### 2.2 Active Documentation Alignment (S230)

**Before:** Four specific drift items where documentation referenced pre-restructuring names.
**After:** All four corrected. Documentation matches the codebase.

**Corrections:**
- Migration catalog: "default database" → "initial bootstrap connection to system database"
- Codegen file paths: `signal_registry.go` → `natssignal/registry.go` (2 occurrences)
- Codegen markers: `BEGIN/END CODEGEN MANAGED SECTION` → `codegen:begin/end`

**Value:** Future developers reading these docs will find correct file paths and naming. No silent drift in the active corpus.

### 2.3 Remote CI Proof and Defect Discovery (S231)

**Before:** No remote CI run on the post-S229 baseline. Tag blocked.
**After:** Run `23365571775` all green. Tag `v0.1.0-s231` published.

**Bonus discovery:** The first push exposed two real defects invisible to local tooling:
1. **Codegen template misalignment** — S227's writer pipeline refactor wasn't reflected in templates. Fixed by adding `writer.columns` to 7 family YAML specs and updating template generation.
2. **Go 1.25 stdlib collision** — `cmd/migrate/migrate/` collided with Go 1.25's reserved `cmd/` prefix. Renamed to `cmd/migrate/engine/`.

**Value:** Remote CI proved its worth by catching defects that local tooling missed. The defect-then-fix cycle validates pipeline integrity.

---

## 3. Trade-offs

### 3.1 Scope Discipline vs. Deeper Cleanup

The tranche was deliberately bounded to mechanical corrections. This means:

- **265 architecture docs** were not reviewed for broader coherence. Only the 4 specific drift items from S228 were fixed.
- **224 stage reports** were not pruned or consolidated. They remain as historical artifacts.
- **raccoon-cli** received targeted fixes, not a comprehensive audit of all analyzer assumptions.

**Trade-off accepted:** Broader cleanup would have expanded scope beyond what was needed for clean-pass. The next charter can address documentation entropy if it's a priority.

### 3.2 Tag Linearity vs. Semantic Versioning

The tag `v0.1.0-s231` follows stage-based versioning rather than semantic versioning. This is appropriate for the current pre-production phase but will need to transition to semver when the project approaches release.

### 3.3 CI Pipeline Completeness

The CI pipeline validates three jobs: unit tests, codegen golden equivalence, and smoke analytical E2E. It does **not** include:

- Integration tests with embedded NATS (`make test-integration`)
- Full E2E first-slice smoke (`make smoke`)
- Load or performance testing
- Security scanning

**Trade-off accepted:** The current pipeline is sufficient for the mechanical gate. Expanding CI coverage is a valid next-charter objective.

---

## 4. Open Debts

These are known items that are **not blockers** for clean-pass but represent technical debt for future charters.

### 4.1 Documentation Entropy (Low Priority)

- 265 architecture docs accumulated across 230+ stages. Many reference superseded decisions.
- 224 stage reports provide historical value but no active governance function.
- No automated doc-staleness detection beyond the specific items raccoon-cli checks.

**Recommendation:** A future charter could introduce a doc-lifecycle policy (archive after N stages of inactivity) or a doc-health check in raccoon-cli.

### 4.2 raccoon-cli Assumption Freshness (Medium Priority)

S229 fixed 6 stale assumptions. There may be others that haven't surfaced because they happen to align with current architecture by coincidence. A systematic assumption audit would reduce future surprise.

**Recommendation:** Next charter could include a raccoon-cli assumption inventory as a precondition for new analyzer rules.

### 4.3 CI Pipeline Gaps (Medium Priority)

Integration tests and full E2E smoke are not in the remote CI pipeline. They run locally via `make test-integration` and `make smoke` but are not gated.

**Recommendation:** Adding `make test-integration` to CI would catch NATS-level regressions earlier. Full smoke requires infrastructure that may not be available in CI runners.

### 4.4 Production Readiness (Future Charter)

- No production deployment configuration.
- No load testing or capacity planning.
- Execute service uses paper venue adapter only.
- No monitoring/alerting beyond health endpoints.

These are evolutionary concerns, not mechanical debts.

---

## 5. Tranche Efficiency

| Metric | Value |
|--------|-------|
| Stages in tranche | 3 (S229, S230, S231) |
| Files modified (code) | ~20 (raccoon-cli + codegen + migrate) |
| Files modified (docs) | 3 active docs + 4 stage/architecture reports |
| Blockers closed | 4/4 from S228 |
| Defects discovered by remote CI | 2 (codegen template, Go 1.25 collision) |
| New technical debt introduced | 0 |

The tranche was efficient: bounded scope, no scope creep, zero new debt introduced.
