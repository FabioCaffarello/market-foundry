# Wave B After Family 05 — Gains, Trade-offs, and Open Debts

## Gains

### Confirmed Across 6 Families

1. **Full vertical analytical coverage**: L1 (Evidence) → L6 (Executions). Every layer in the trading pipeline has a read-path analytical endpoint.

2. **Zero creative decisions**: 5 family expansions, each following the identical 9-artifact template. This is the strongest evidence that the pattern is mechanically reproducible and codegen-ready.

3. **Zero write-path changes**: The writer pipeline, inserter, supervisor, and mappers have not been modified across 6 analytical expansions. The original design was correct.

4. **Linear cost growth**: Each family adds ~780 LOC (~350 impl + ~430 tests) at ~45 minutes of effort. No exponential drag, no cross-family coupling, no integration tax.

5. **JSON complexity solved**: Families 01–05 covered 1, 2, 3, and 4 JSON columns with struct, slice, map, and mixed targets. The parsing pattern generalizes trivially.

6. **New type classes absorbed**: Family 05 introduced Float64 (2 columns) and Boolean (1 column) without pattern modification. The reader/handler template handles any ClickHouse type.

7. **Dual optional filters**: Family 05 proved that 2 optional filters per method compose cleanly via additive WHERE clauses. No filter interaction needed.

8. **Struct-based DI scales**: 6 families injected into the gateway, reader, and handler composition without constructor signature changes.

9. **Observability is automatic**: Server-Timing headers, structured logging, error codes, and health checks apply identically to every family.

10. **ClickHouse optionality preserved**: The analytical layer remains entirely optional; disabling ClickHouse does not affect operational NATS/SQLite paths.

### Family 05 Specific Gains

- Highest column count (20 DDL, 16 SELECT) validated end-to-end.
- 4 JSON columns (matching Family 04 ceiling) round-tripped cleanly.
- 2 new Float64 columns used FormatFloat without modification.
- Dual-filter AND composition proved for the first time.
- 289 total tests, 47 execution-specific — most thorough coverage of any family.

## Trade-offs

### Accepted and Understood

1. **Manual artisanship at scale**: Each family is hand-crafted copy-adapt work. At 5 families this is tolerable; at 10+ it becomes engineering waste. The trade-off was accepted to validate the pattern before investing in codegen.

2. **Handler monolith**: All 6 analytical handler methods live in one file (501 lines after H-5). This concentrates risk and makes diffs noisy, but avoids premature file splitting that would scatter the pattern.

3. **Positional reader parameters**: 10 positional args in the execution reader method. Readable today, will become error-prone at 12+. Accepted to defer query-object refactoring until codegen scope.

4. **No CI integration for analytical smoke tests**: The end-to-end smoke test (651 lines) runs locally but not in CI. Accepted because ClickHouse is not in CI infrastructure yet.

5. **No pagination beyond 500 rows**: Analytical queries cap at 500 results. Sufficient for dashboard use cases; will need revisiting for bulk export.

6. **Case-sensitive, unvalidated filters**: Side/status filters accept any string without normalization. Consistent behavior across all families, low production impact.

7. **Test duplication**: Each family's tests follow the same shapes (~430 LOC). Acceptable while manual; codegen will template these.

### Trade-offs That Have Shifted Toward Debt

1. **CI smoke gap**: Flagged 5 times across 5 families. No longer a trade-off — it is a carried debt that should be addressed when ClickHouse enters CI.

2. **Manual expansion cost**: At Family 05, the 45-minute-per-family cost is a trade-off. At Family 07+, without codegen, it becomes engineering debt.

## Open Debts

### Mandatory Before Family 07

| Debt | Origin | Why Mandatory |
|------|--------|---------------|
| Codegen tranche scoping | Family 03 recommendation, deferred through 04 and 05 | 5 families with 0 creative decisions = no justification to continue manual |
| Reader query-object pattern | Family 05 hit 10-param limit | Family 07 likely needs 11+ params |
| Generic JSON parser `parseJSON[T]` | Family 04 flagged at 6 parsers, now 8 | Parser 9 benefits from generics; parser 12+ requires it |

### Tracked, Non-Blocking

| Debt | Origin | Current Status |
|------|--------|----------------|
| CI analytical smoke test | Family 01 | Flagged 5× — requires ClickHouse in CI |
| Schema coherence tooling | Wave A | 6 tables, ~95 columns — under threshold |
| Filter validation/normalization | Family 02 | Consistent behavior, no incident |
| Pagination beyond 500 | Family 01 | No production need |
| NATS consumer lag visibility | Writer hardening | Writer operational concern |
| Sticky degradation without auto-recovery | Writer supervision | Supervision model works, refinement deferred |
| Backoff jitter in writer retry | Writer hardening | Functional without jitter |
| Silent mapper fallbacks | Writer hardening | No data loss observed |

### Resolved Debts (This Gate)

| Debt | Resolution |
|------|------------|
| Handler file at ceiling (615/620) | H-5: `parseAnalyticalParams()` extraction → 501 lines |
| All handlers duplicating limit/since/until parsing | H-5: single shared helper |

## Debt Trajectory

```
Wave A:          4 debts carried
Family 01:       6 debts (2 new)
Family 02:       7 debts (1 new)
Family 03:       8 debts (1 new, hardening resolved 2)
Family 04:       9 debts (2 new, 1 resolved)
Family 05:       11 debts (3 new, 1 resolved by H-5)
                 ↑
                 3 mandatory before Family 07
                 8 tracked, non-blocking
```

Debt count grows linearly with family count but remains bounded because new debts cluster around the same pressure points (handler size, reader params, parser count). This is a sign of pattern stability, not pattern decay.

## Items That Do Not Justify Cost Now

1. **Codegen for Family 06**: The investment (~1–2 days) does not pay off for a single family. It pays off at Family 07+ where the manual pattern approaches its ceiling.

2. **Handler file split**: Splitting 501 lines into per-family files adds indirection without reducing complexity. Codegen will template this decision.

3. **ClickHouse in CI**: Requires container orchestration changes to the CI pipeline. The cost exceeds the value until analytical coverage drives production-facing decisions.

4. **Bulk export pagination**: No user or downstream system requests >500 rows. Building it now is speculative.

5. **Filter normalization**: Adding case-insensitive matching and enum validation is correct but not yet motivated by incidents or user complaints.

## Conclusion

Wave B after Family 05 is in its strongest position: full vertical coverage, proven pattern, quantified limits, and a clear path to automation. The gains are real and the debts are well-understood. The critical question is not "can we continue?" but "when does manual expansion become waste?" — and the answer is: at Family 07, without codegen.
