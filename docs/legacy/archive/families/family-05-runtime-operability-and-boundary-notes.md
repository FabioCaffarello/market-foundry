# Family 05 — Runtime Operability and Boundary Notes

**Stage:** S187
**Family:** 05 — Executions (paper_order)
**Date:** 2026-03-20

## Endpoint

```
GET /analytical/execution/history
```

### Required parameters

| Param | Type | Example |
|-------|------|---------|
| type | string | paper_order |
| source | string | derive |
| symbol | string | btcusdt |
| timeframe | int | 60 |

### Optional parameters

| Param | Type | Default | Example |
|-------|------|---------|---------|
| side | string | (empty = all) | buy, sell, none |
| status | string | (empty = all) | submitted, filled, rejected |
| since | int64 | 0 (unset) | 1710000000 |
| until | int64 | 0 (unset) | 1710003600 |
| limit | int | 50 | 10 (max 500) |

### Response codes

| Code | Condition |
|------|-----------|
| 200 | Success |
| 400 | Missing required param, invalid limit, invalid timestamp, since > until |
| 503 | ClickHouse unavailable or execution reader not configured |

### Response shape

```json
{
  "executions": [...],
  "source": "analytical/clickhouse",
  "meta": { "query_ms": N, "row_count": M }
}
```

### Response headers

- `Server-Timing: total;dur=N, query;dur=M`

## ClickHouse Optionality

Family 05 preserves full ClickHouse optionality:

- **No ClickHouse configured** → endpoint not registered; gateway starts normally
- **ClickHouse configured but unreachable** → endpoint registered; returns 503 on query failure
- **ClickHouse configured and healthy** → endpoint returns 200 with data

This matches the behavior of all 5 previous analytical families.

## Responsibility Boundaries

| Layer | Responsibility | File |
|-------|---------------|------|
| Adapter | SQL query, row scanning, JSON parsing | `execution_reader.go` |
| Use case | Validation, defaults, timing, error wrapping | `get_execution_history.go` |
| Handler | HTTP parsing, param validation, response formatting | `analytical.go` |
| Route | Conditional registration | `routes/analytical.go` |
| Gateway | Wiring adapter → use case → handler | `compose.go` + `analytical_reader.go` |

No layer crosses its responsibility boundary. The adapter does not validate; the handler does not query.

## Degradation Behavior

- **Reader nil** → handler returns 503 immediately (no query attempted)
- **ClickHouse timeout** → reader returns error → use case wraps as Unavailable → handler returns 503 with Server-Timing
- **Invalid filter values** → ClickHouse returns 0 rows (no server-side validation)
- **Empty result** → returns 200 with empty `executions` array

## Handler File Size

Post-Family-05: **615 lines** (hard ceiling: 620).

This is the terminal state for the manual pattern. Family 06 would exceed the ceiling without either:
1. Extracting a `parseAnalyticalParams()` helper (~30 lines saved)
2. Splitting the handler file

Both are deferred — they are codegen-tranche prerequisites, not Family 05 scope.

## Observed Frictions

### Friction 1: Two optional filters increases query builder complexity

Family 05 is the first family with two optional filters (side + status). The query builder handles this correctly with additive WHERE clauses, but the arg count reaches 9 in the worst case (4 base + 2 filters + 2 time + 1 limit).

**Impact:** Low. The pattern handles it without structural change.
**Codegen relevance:** Query builder template should parameterize optional filter count.

### Friction 2: Reader method signature at 10 parameters

`QueryExecutionHistory` has 10 parameters — the largest reader signature in the codebase. This is acceptable for the terminal family but confirms that a query-object pattern would reduce signature complexity for codegen.

**Impact:** Low. Go handles positional parameters adequately at this count.
**Codegen relevance:** Consider query-object pattern in generated readers.

### Friction 3: Handler file at ceiling

At 615 lines, the handler is 5 lines below the hard ceiling. Adding one more family method (~99 lines) would exceed 620 lines.

**Impact:** None for Family 05 — it's within bounds.
**Codegen relevance:** Handler split or helper extraction is mandatory before Family 06.

## Limits Maintained

- One family added (executions)
- One endpoint added (`GET /analytical/execution/history`)
- Zero write-path changes
- Zero new abstractions
- Zero changes to existing families
- Wave B pattern applied with discipline
- ClickHouse optionality preserved
