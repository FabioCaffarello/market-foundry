# Wave B Family-01 Validation Findings and Pattern Frictions

> Concrete findings, frictions, and observations from validating the first Wave B family (Signals/RSI) end-to-end.
> This document captures what worked, what created friction, and what the pattern needs before the second family can proceed safely.

## Findings

### F-1: The Wave B expansion pattern works as designed

The 9-artifact pattern from S163 produced a functioning family expansion with no structural surprises. The signal family was built by following the candle baseline as a template, and the result is a complete, coherent data path. This validates the pattern at the minimum useful level.

### F-2: Schema coherence is provable through tests alone

The column alignment between DDL, writer mapper, and reader adapter was verified entirely through unit tests — no runtime ClickHouse needed. The `BuildSignalQuery()` export pattern (deterministic query builder for testing) and the mapper row-length assertion together provide sufficient confidence without integration-level schema checks.

### F-3: Write path required zero changes for the signal family

The writer already consumed RSI signal events from NATS and inserted into ClickHouse via `mapSignalRow()`. The entire S164/S165 scope for the signal family was the read path (adapter, use case, handler, route, gateway composition). This confirms the writer was correctly designed as a multi-family service from inception.

### F-4: Observability parity achieved mechanically

The signal read path has identical observability to candles: wall-clock timing in the adapter, `QueryMeta` in the use case, `Server-Timing` header in the handler, structured logging at every layer. This was achieved by copying the candle pattern. No new observability infrastructure was needed.

### F-5: Error handling contracts are consistent across families

Both candle and signal endpoints return the same HTTP status codes for the same error conditions: 400 for missing/invalid parameters, 503 when ClickHouse is unavailable. The smoke test now validates both families against the same error contract, confirming behavioral parity.

### F-6: Metadata JSON adds exactly one new concern

The signal family introduces `metadata` as a JSON-encoded `map[string]string` field. This is the first field in the analytical layer that requires deserialization beyond primitive types. `ParseMetadataJSON` handles this with a silent fallback to empty map on invalid JSON — matching the write-path's `marshalJSON` fallback to `"{}"`. The trade-off (silent corruption) is documented and accepted.

### F-7: Signal type filter is mandatory and validated

Unlike candles (which have no type dimension), signals require a `type` parameter (e.g., "rsi"). This is enforced at both the handler level (400 if missing) and the use case level (validation before query). The type flows through as a WHERE clause filter, not a path parameter — this keeps the analytical URL namespace flat and consistent.

## Pattern Frictions

### PF-1: Naming residue — `parseEvidenceKeyParams()`

**Friction:** The shared function `parseEvidenceKeyParams()` in the handler extracts `source`, `symbol`, and `timeframe` from query parameters. Its name contains "evidence" but the parameters are universal across all domain types. The signal handler reuses this function.

**Impact:** Low — code reads slightly misleadingly, but the function signature and behavior are correct. Renaming would require a horizontal refactor (violates C-9 additive-only constraint).

**Recommendation:** Accept for now. If a third family reuses this function, rename to `parseAnalyticalKeyParams()` as part of that iteration's scope.

### PF-2: Constructor accumulation in AnalyticalWebHandler

**Friction:** `NewAnalyticalWebHandler()` takes N use-case arguments (currently 2: candle + signal). Each new family adds another argument. At 4+ families this becomes unwieldy.

**Impact:** Medium — the current 2-argument constructor is clean, but the third family will create pressure.

**Recommendation:** When adding the third family, consider switching to a struct-based dependency injection pattern:
```go
type AnalyticalDeps struct {
    CandleHistory  *GetCandleHistoryUseCase
    SignalHistory  *GetSignalHistoryUseCase
    DecisionHistory *GetDecisionHistoryUseCase // future
}
```

### PF-3: Mechanical duplication across families

**Friction:** The signal read path is ~80% identical to the candle read path at every layer:
- Reader adapter: same structure, different columns and table name
- Use case: same validation logic, different query/reply types
- Handler: same HTTP parsing, different response key name
- Route: same conditional registration, different path

**Impact:** Low at 2 families. Becomes a maintenance burden at 4+.

**Recommendation:** Accept explicit duplication through 3 families. At family 4, evaluate whether codegen (for readers/use cases) or a generic analytical query framework is justified. The cost of premature abstraction is higher than the cost of mechanical duplication at the current scale.

### PF-4: No signal-type validation against known families

**Friction:** The reader accepts any `type` string. Querying `?type=nonexistent` returns an empty result (200 with 0 signals) rather than a 400 error.

**Impact:** Low — ClickHouse simply returns no rows. No security or correctness risk. The alternative (validating against a registry of known signal types) would couple the read adapter to the settings layer.

**Recommendation:** Accept. Consider adding a `known_signal_types` informational field to the `/diagz` response if operators need visibility into which types are active.

### PF-5: Smoke test grows linearly with families

**Friction:** Each new family adds ~50 lines to `smoke-analytical-e2e.sh` (data verification, HTTP validation, error handling checks). At 6 families this will be a 500+ line script.

**Impact:** Medium — the script remains maintainable at 2 families but will need restructuring.

**Recommendation:** After the third family, consider extracting a `validate_analytical_family()` bash function that takes family-specific parameters (table name, endpoint path, required fields, sample query params). This reduces per-family additions to ~5 lines.

### PF-6: No automated CI integration yet

**Friction:** The Wave B checklist (S163) states "CI smoke-analytical integration is in place (required before second family; recommended before first)." This has not been implemented yet. The smoke test runs manually.

**Impact:** High — without CI, regressions in the analytical layer can ship undetected.

**Recommendation:** This is a blocking prerequisite before starting the second family. CI must run at minimum: unit tests for all analytical packages + the smoke-analytical-e2e.sh script against a compose stack.

## Limits Observed

1. **Only RSI signals tested** — the EMA crossover signal pipeline exists in the writer but depends on the EMA crossover actor being enabled. Validation covers only the RSI type.
2. **No load testing** — the signal path was tested with development-scale data only. Batch insert performance under production volumes is untested.
3. **No concurrent query testing** — the signal reader was tested with sequential queries. Behavior under concurrent analytical queries is not validated.
4. **Metadata schema not validated** — the reader deserializes metadata as `map[string]string` without checking for expected keys (e.g., `period`, `avg_gain`, `avg_loss` for RSI).
5. **No pagination** — results are hard-limited to 500 rows. Cursor-based pagination is deferred.

## What the Pattern Proves

After this validation, the Wave B expansion pattern demonstrates:

1. **A new family can be added with zero changes to the writer** (when the writer already supports the family's events).
2. **The read path follows a repeatable structure**: adapter → use case → handler → route → gateway composition.
3. **Schema coherence is testable without a running ClickHouse**.
4. **Observability parity is mechanical** — copy the pattern, get the same instrumentation.
5. **Error handling is consistent** — the same HTTP contract applies across families.
6. **Optionality is preserved** — the operational pipeline runs independently of ClickHouse.

## What the Pattern Does NOT Yet Prove

1. That the pattern scales beyond 2 families without excessive duplication.
2. That CI can catch regressions automatically.
3. That the constructor accumulation problem has a clean solution.
4. That the smoke test structure remains manageable at scale.
5. That cross-family queries (joins) are feasible or needed.
