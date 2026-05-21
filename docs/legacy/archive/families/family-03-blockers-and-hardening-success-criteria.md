# Family 03 Blockers and Hardening Success Criteria

> Explicit gate conditions that must be satisfied before Family 03 expansion begins.

## Binding Commitments

The following blockers originate from S167 (Wave B Iteration Gate) and were reconfirmed in S170 (Decisions Family E2E Validation). They are **not suggestions**; they are pre-committed gate conditions.

## Blockers

### Blocker 1: Struct-Based DI (H-1)

**Status:** Blocking
**Origin:** S167 D-2, S170 PF-1, Wave B Pattern Hardening A-5
**Severity:** Medium

**Condition to clear:**
- `NewAnalyticalWebHandler` accepts a single `AnalyticalHandlerDeps` struct argument.
- All existing handler behavior is preserved (no functional changes).
- All handler tests pass with the new constructor signature.
- Route wiring (`internal/interfaces/http/routes/analytical.go`) uses the struct.
- Gateway composition (`cmd/gateway/compose.go`) constructs and passes the struct.
- No positional argument constructor remains in the codebase.

**Verification:**
```bash
# Must return zero results:
grep -r 'NewAnalyticalWebHandler(' --include='*.go' | grep -v 'AnalyticalHandlerDeps'
# Must compile and pass:
go test ./internal/interfaces/http/handlers/... ./internal/interfaces/http/routes/...
```

---

### Blocker 2: Smoke Test Function Extraction (H-2)

**Status:** Blocking
**Origin:** S167 D-3, S170 PF-3, Wave B Pattern Hardening A-4
**Severity:** Medium

**Condition to clear:**
- A `validate_analytical_family()` bash function exists in the smoke script.
- All three current families (candles, signals, decisions) are validated through this function.
- The smoke script produces **identical** pass/fail results as the current version.
- Family-specific parameters (endpoint, fields, required params) are passed as arguments.
- Adding a fourth family requires only a single function call, not 80+ lines of copy.

**Verification:**
```bash
# Function must exist:
grep -c 'validate_analytical_family' scripts/smoke-analytical-e2e.sh
# Script must still pass against running infrastructure:
bash scripts/smoke-analytical-e2e.sh --wait 60
```

---

### Blocker 3: Helper Renaming (H-3)

**Status:** Blocking
**Origin:** S167 D-1, S170 PF-2, Wave B Pattern Hardening A-1
**Severity:** Low-Medium

**Condition to clear:**
- `parseEvidenceKeyParams` is renamed to `parseAnalyticalKeyParams`.
- All call sites updated (3 handlers).
- All tests pass with the new name.
- Zero instances of `parseEvidenceKeyParams` remain in the codebase.

**Verification:**
```bash
# Must return zero results:
grep -r 'parseEvidenceKeyParams' --include='*.go'
# Must return 3+ results:
grep -r 'parseAnalyticalKeyParams' --include='*.go'
# Must compile and pass:
go test ./internal/interfaces/http/handlers/...
```

---

## Composite Gate Criteria

All three blockers must be cleared simultaneously. The gate passes when:

| Criterion | Measurement |
|---|---|
| All three H-items implemented | Code changes merged, no legacy signatures remain |
| All Go tests pass | `go test ./...` across affected modules |
| Smoke test behavioral parity | Smoke script output unchanged (same phases, same validations, same pass/fail) |
| CI green | `.github/workflows/ci.yml` passes on the branch |
| No new frictions introduced | Hardening did not create new scaling constraints |
| Documentation updated | Handler signatures, smoke structure reflected in docs |

## Stop Conditions

The hardening tranche must halt and escalate if:

- A hardening change introduces a behavioral regression (functional output differs).
- The struct DI refactor requires changes to the writer or reader adapters (scope creep signal).
- The smoke extraction changes the set of validations performed (must be behavior-preserving).
- Any hardening item requires schema, migration, or ClickHouse changes (wrong layer).

## What This Gate Unlocks

Once this gate passes, Family 03 may proceed following the Wave B Family Expansion Pattern v2. The expansion will:

- Add a 5th field to `AnalyticalHandlerDeps` (struct scales cleanly).
- Add a single `validate_analytical_family()` call to the smoke script (~5 lines, not ~80).
- Use `parseAnalyticalKeyParams()` without naming confusion.

The hardening directly reduces the marginal cost of each subsequent family expansion.

## What This Gate Does NOT Unlock

- CI smoke integration (PF-5) — remains a separate initiative.
- Codegen evaluation (D-4) — deferred to family 4 gate.
- Consumer/inserter naming review (H-4) — deferred, insufficient severity.
- Pagination (D-9, PF-6) — functional improvement, not structural.
