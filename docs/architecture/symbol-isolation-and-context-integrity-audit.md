# Symbol Isolation and Context Integrity Audit

> S301 deliverable — audits every read-path component for cross-symbol contamination risk.
> Date: 2026-03-21

---

## 1. Scope

This audit covers all analytical read-path components that participate in multi-symbol query execution:

| Layer | Component | Files |
|-------|-----------|-------|
| Adapter | Individual readers (signal, decision, strategy, risk, execution) | `internal/adapters/clickhouse/*_reader.go` |
| Adapter | CompositeReader (5-table composition) | `internal/adapters/clickhouse/composite_reader.go` |
| Application | GetCompositeChainUseCase | `internal/application/analyticalclient/get_composite_chain.go` |
| Application | GetPipelineFunnelUseCase | `internal/application/analyticalclient/get_pipeline_funnel.go` |
| Application | GetDispositionBreakdownUseCase | `internal/application/analyticalclient/get_disposition_breakdown.go` |
| Interface | CompositeWebHandler (HTTP) | `internal/interfaces/http/handlers/composite.go` |
| Interface | Route registration | `internal/interfaces/http/routes/analytical.go` |
| Gateway | Analytical reader composition | `cmd/gateway/analytical_reader.go` |

---

## 2. Schema-Level Isolation

All five domain tables share an identical ORDER BY prefix:

```sql
ORDER BY (source, symbol, timeframe, type, timestamp)
```

- `symbol` is declared as `LowCardinality(String)` — optimized for high-cardinality filtering.
- Symbol is the **second column** in the MergeTree order key, meaning ClickHouse can prune granules efficiently when symbol is in the WHERE clause.
- `correlation_id` is **not** part of the order key — it is a secondary filter. Queries using only `correlation_id` without `symbol` perform full-table scans within the time partition.

**Assessment**: Schema supports symbol isolation. The ORDER BY design encourages symbol-scoped queries.

---

## 3. Individual Readers (Pre-Composite)

Each domain-specific reader (`signal_reader.go`, `decision_reader.go`, etc.) uses the centralized `BuildQuery()` function which enforces **mandatory** filters:

```
WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?
```

All four dimensions are required parameters — queries cannot execute without symbol.

**Assessment**: SAFE. No isolation risk in individual readers.

---

## 4. Composite Reader — Pre-S301 State

### 4.1 Batch Path (QueryChainsBatch)

The batch path starts from `queryExecutionCorrelationIDs()` which correctly filters:

```sql
WHERE source = ? AND symbol = ? AND timeframe = ?
```

**Assessment**: SAFE. Batch entry point is symbol-scoped.

### 4.2 Single-Chain Path (QueryChainByCorrelationID) — CRITICAL GAP FOUND

**Pre-S301**: The method accepted only `correlationID` and queried all 5 tables with:

```sql
WHERE correlation_id = ?
```

No symbol filter was applied. This meant:

1. A correlation_id lookup could theoretically return events from any symbol.
2. In the batch path, after the initial symbol-scoped execution query returned correlation_ids, the enrichment step (`QueryChainByCorrelationID`) lost the symbol context.
3. The HTTP endpoint `GET /analytical/composite/chain?correlation_id=...` had no symbol parameter at all.

**Risk level**: CRITICAL — provable cross-symbol contamination if correlation_ids were shared or collided across symbols.

### 4.3 Aggregation Paths (Funnel, Dispositions)

Both `QueryPipelineFunnel` and `QueryDispositionBreakdown` include symbol in every WHERE clause:

```sql
WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?
```

**Assessment**: SAFE. No isolation risk.

---

## 5. S301 Remediation Applied

### 5.1 Composite Reader

All five `queryXxxByCorrelation` methods now accept `symbol` parameter:

```go
func (r *CompositeReader) querySignalByCorrelation(ctx context.Context, correlationID, symbol string) (...)
```

SQL changed from:
```sql
WHERE correlation_id = ?
```
to:
```sql
WHERE correlation_id = ? AND symbol = ?
```

`QueryChainByCorrelationID` signature changed:
```go
// Before (S296)
QueryChainByCorrelationID(ctx context.Context, correlationID string) (...)

// After (S301)
QueryChainByCorrelationID(ctx context.Context, correlationID, symbol string) (...)
```

### 5.2 Interface and Use Case

`CompositeReader` interface updated to match. `GetCompositeChainUseCase.executeSingle()` now requires and passes `symbol`. Single-chain queries without symbol are rejected with `InvalidArgument`.

### 5.3 HTTP Handler

`GET /analytical/composite/chain` now requires `symbol` query parameter:

```
GET /analytical/composite/chain?correlation_id=...&symbol=...
```

Missing symbol returns HTTP 400.

---

## 6. Post-S301 Isolation Matrix

| Component | Symbol Filter | Pre-S301 | Post-S301 |
|-----------|--------------|----------|-----------|
| Individual readers | Mandatory (BuildQuery) | SAFE | SAFE |
| CompositeReader.QueryChainByCorrelationID | `AND symbol = ?` on all 5 sub-queries | MISSING | FIXED |
| CompositeReader.QueryChainsBatch | Symbol in execution query + chain enrichment | PARTIAL | FIXED |
| CompositeReader.QueryPipelineFunnel | Mandatory symbol WHERE | SAFE | SAFE |
| CompositeReader.QueryDispositionBreakdown | Mandatory symbol WHERE | SAFE | SAFE |
| GetCompositeChainUseCase (single) | Symbol required in query | MISSING | FIXED |
| GetCompositeChainUseCase (batch) | Symbol required (validated) | SAFE | SAFE |
| HTTP GetChain | `symbol` query param | MISSING | REQUIRED |
| HTTP GetChains | `symbol` in parseQueryKeyParams | SAFE | SAFE |
| HTTP GetFunnel | `symbol` in parseQueryKeyParams | SAFE | SAFE |
| HTTP GetDispositions | `symbol` in parseQueryKeyParams | SAFE | SAFE |

**Post-S301 status**: All read paths enforce symbol isolation. No path exists to query without symbol context.

---

## 7. Integrity Classification

**Post-S301 classification: INTACT**

- Every read-path query includes symbol in WHERE clause.
- Cross-symbol contamination is blocked at query level.
- HTTP API enforces symbol as mandatory parameter on all endpoints.
- Attribution logic derives from symbol-scoped chain data — safe by transitivity.
- Correlation_id + symbol compound filter provides unique chain identification.
