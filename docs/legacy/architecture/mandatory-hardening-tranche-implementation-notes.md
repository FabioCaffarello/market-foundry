# Mandatory Hardening Tranche — Implementation Notes

> S172 implementation record. Covers what was changed, why, and how.

## Scope

Three hardening items mandated by S171 / S167, executed in dependency order.

## H-3: Helper Renaming — `parseEvidenceKeyParams` → `parseQueryKeyParams`

**What changed:**
- `evidenceKeyParams` struct → `queryKeyParams`
- `parseEvidenceKeyParams()` function → `parseQueryKeyParams()`
- All 7 handler files updated (evidence, signal, decision, strategy, risk, execution, analytical)

**Why `parseQueryKeyParams` instead of `parseAnalyticalKeyParams`:**
The function is used by **all** handler families (operational and analytical), not just analytical endpoints. Renaming to `parseAnalyticalKeyParams` would replace one misleading prefix with another. The neutral name `parseQueryKeyParams` reflects actual scope — it parses the common `source`, `symbol`, `timeframe` query parameters shared by every handler family.

**Files modified:**
- `internal/interfaces/http/handlers/evidence.go` — definition site (struct + function)
- `internal/interfaces/http/handlers/analytical.go` — 3 call sites
- `internal/interfaces/http/handlers/signal.go` — 1 call site
- `internal/interfaces/http/handlers/decision.go` — 1 call site
- `internal/interfaces/http/handlers/strategy.go` — 1 call site
- `internal/interfaces/http/handlers/risk.go` — 1 call site
- `internal/interfaces/http/handlers/execution.go` — 2 call sites

**Verification:** `grep -r parseEvidenceKeyParams` returns zero Go matches.

## H-1: Struct DI — `NewAnalyticalWebHandler`

**What changed:**
- Added `AnalyticalHandlerDeps` struct grouping all constructor dependencies
- `NewAnalyticalWebHandler` now accepts a single `AnalyticalHandlerDeps` argument
- Route wiring in `routes/analytical.go` updated to use struct literal
- All 20 test constructor calls in `analytical_test.go` updated

**Before:**
```go
func NewAnalyticalWebHandler(
    getCandleHistory getAnalyticalCandleHistoryUseCase,
    getSignalHistory getAnalyticalSignalHistoryUseCase,
    getDecisionHistory getAnalyticalDecisionHistoryUseCase,
    logger *slog.Logger,
) *AnalyticalWebHandler
```

**After:**
```go
type AnalyticalHandlerDeps struct {
    GetCandleHistory   getAnalyticalCandleHistoryUseCase
    GetSignalHistory   getAnalyticalSignalHistoryUseCase
    GetDecisionHistory getAnalyticalDecisionHistoryUseCase
    Logger             *slog.Logger
}

func NewAnalyticalWebHandler(deps AnalyticalHandlerDeps) *AnalyticalWebHandler
```

**Why struct DI matters at this threshold:**
At 4 positional parameters, adding Family 3 would mean 5 positional args — fragile and error-prone. Struct DI eliminates positional coupling. Adding Family 4, 5, ... N requires adding one struct field and zero signature changes.

**Files modified:**
- `internal/interfaces/http/handlers/analytical.go` — struct + constructor
- `internal/interfaces/http/routes/analytical.go` — caller
- `internal/interfaces/http/handlers/analytical_test.go` — 20 test constructors

## H-2: Smoke Extraction — `validate_analytical_family()`

**What changed:**
- Extracted `validate_analytical_family()` — parameterized function covering:
  - ClickHouse row count verification
  - HTTP endpoint availability (200 check)
  - JSON response structure validation (keys, source, meta, required fields)
  - Item count verification with ClickHouse cross-check
  - Server-Timing header check
- Extracted `validate_analytical_error_handling()` — parameterized function covering:
  - Missing timeframe → 400
  - Invalid limit → 400
  - since > until → 400
- Replaced ~250 lines of per-family copy-paste with 3 function calls (~7 lines each)

**Cost of adding Family 3 to the smoke script (before vs after):**
- Before: ~80 new lines of copy-paste with manual field list adjustments
- After: ~7 lines calling `validate_analytical_family` with the right parameters

**Files modified:**
- `scripts/smoke-analytical-e2e.sh` — function extraction + call site replacement

## What was NOT changed

- No schema, migration, or writer changes
- No new families added
- No CI smoke integration (deferred per S171 out-of-scope)
- No changes to operational handler constructors (they don't need struct DI yet)
- No changes to ClickHouse adapter layer
- No changes to use case layer
