# Pattern Hardening After Wave B Family-02

> Captures the state of the analytical expansion pattern after the S172 mandatory hardening tranche, and defines what is now true for Family 3 entry.

## Pattern State After Hardening

### Constructor pattern
- `NewAnalyticalWebHandler` accepts `AnalyticalHandlerDeps` struct
- Adding a new family requires: one struct field, one use case, one route block
- Zero positional coupling; zero signature churn on expansion

### Query parameter parsing
- `parseQueryKeyParams()` is the canonical helper for extracting `source`, `symbol`, `timeframe`
- Name is family-neutral; used by all 7 handler families without semantic friction
- Defined in `evidence.go` (historical home); could be moved to a shared file if handler count grows further — not needed now

### Smoke validation
- `validate_analytical_family()` is the canonical function for per-family E2E validation
- Parameters: family label, CH table, WHERE clause, HTTP URL, JSON key, required fields
- `validate_analytical_error_handling()` covers standard 400-response validation
- Adding a new family to the smoke script costs ~7 lines

## What the Pattern Now Supports

| Capability | Before S172 | After S172 |
|---|---|---|
| Add analytical family: constructor | Edit positional args (N params) | Add 1 struct field |
| Add analytical family: smoke | Copy ~80 lines, adjust manually | Call function with 6 args |
| Helper naming | `parseEvidenceKeyParams` (misleading) | `parseQueryKeyParams` (neutral) |
| Test DI wiring | Match positional order with nils | Named struct fields |

## What Remains Simple (By Design)

- **Operational handler constructors** (Evidence, Signal, Decision, etc.) stay positional. They have 1-4 args and no expansion pressure. Struct DI would be premature.
- **ClickHouse adapter layer** — one reader per table, no shared abstraction. Each reader owns its query builder and row mapper. This is correct at the current scale.
- **Use case layer** — one use case per family per operation. No generic use case wrapper. Each use case validates independently.
- **Route registration** — conditional `if deps.X != nil` blocks. Clean enough at 3 families.

## Open Debts (Not Blocking Family 3)

1. **`parseQueryKeyParams` lives in `evidence.go`** — semantically it should be in a shared handler utility file. Not worth moving now (Go package scoping makes it visible to all handlers regardless).
2. **Candle family has slightly different structure validation** — the smoke function uses a generic Python validator that works for all families but doesn't print OHLCV-specific details for candles. Acceptable trade-off.
3. **No codegen** — still manual. Evaluation deferred to Family 4 per wave-b-family-expansion-pattern-v2.md.

## Gate Criteria for Family 3

All three blockers from `family-03-blockers-and-hardening-success-criteria.md` are now resolved:

| Blocker | Status | Verification |
|---|---|---|
| H-1: Struct DI constructor | Done | `grep AnalyticalHandlerDeps` in analytical.go |
| H-2: Smoke function extraction | Done | `grep validate_analytical_family` in smoke script |
| H-3: Helper rename | Done | `grep -r parseEvidenceKeyParams *.go` returns 0 |

Family 3 can proceed once a formal gate review (S173) confirms these results and scopes the next family definition.
