# Stage S454A: Operational List Queries and Retrieval Ergonomics

**Status:** Complete
**Depends on:** S453A (Historical Read Model), S413 (Lifecycle List)
**Enables:** S455A (Explainability Surface)

## Objective

Make the execution history and lifecycle surfaces practically usable for
human operation: post-session audit, troubleshooting, and lifecycle navigation.
Before S454A, all query endpoints required full partition key foreknowledge
(source + symbol + timeframe + type), making ad-hoc operational queries
impractical.

## What Changed

### New ClickHouse Query Surfaces

1. **Execution List** (`GET /analytical/execution/list`)
   - Relaxed-filter query: at least one filter required, but none individually mandatory
   - Enables "show all rejected orders" or "all fills in the last hour"
   - Uses `WHERE 1=1` with optional AND clauses instead of rigid mandatory filters

2. **Execution Summary** (`GET /analytical/execution/summary`)
   - GROUP BY (type, status) with counts and latest timestamp
   - Enables "how many rejected vs filled?" without fetching individual rows
   - At least one scope filter required

### HTTP-Exposed Lifecycle List

3. **Lifecycle List** (`GET /execution/lifecycle/list`)
   - Exposes the S413 KV-backed lifecycle list through the gateway HTTP surface
   - Previously reachable only via NATS request/reply
   - Full wiring: port interface, NATS gateway method, use case, handler, route

## Files Changed

### Adapter Layer (ClickHouse)
- `internal/adapters/clickhouse/execution_reader.go` — Added `QueryExecutionList`, `BuildExecutionListQuery`, `QueryExecutionSummary`, `BuildExecutionSummaryQuery`

### Adapter Layer (NATS)
- `internal/adapters/nats/natsexecution/gateway.go` — Added `GetLifecycleList` method

### Application Layer (Contracts)
- `internal/application/analyticalclient/contracts.go` — Added `ExecutionListQuery`, `ExecutionListReply`, `ExecutionSummaryQuery`, `ExecutionSummaryEntry`, `ExecutionSummaryReply`
- `internal/application/analyticalclient/get_execution_list.go` — New use case + `ExecutionListReader` interface
- `internal/application/analyticalclient/get_execution_summary.go` — New use case + `ExecutionSummaryReader` interface + `ExecutionSummaryRawRow`
- `internal/application/executionclient/get_lifecycle_list.go` — New use case for lifecycle list
- `internal/application/ports/execution.go` — Added `GetLifecycleList` to `ExecutionGateway` interface

### Interface Layer (HTTP)
- `internal/interfaces/http/handlers/analytical.go` — Added `GetExecutionList`, `GetExecutionSummary` handlers
- `internal/interfaces/http/handlers/execution.go` — Added `GetLifecycleList` handler, updated constructor
- `internal/interfaces/http/routes/analytical.go` — Added routes for `/analytical/execution/list` and `/analytical/execution/summary`
- `internal/interfaces/http/routes/execution.go` — Added route for `/execution/lifecycle/list`
- `internal/interfaces/http/routes/core.go` — Added `GetLifecycleList` to `ExecutionFamilyDeps`

### Composition
- `cmd/gateway/compose.go` — Wired all three new use cases at the composition root

### Tests
- `internal/adapters/clickhouse/s454a_operational_list_queries_test.go` — 11 tests for query builders
- `internal/application/analyticalclient/s454a_operational_list_queries_test.go` — 12 tests for use cases

### Documentation
- `docs/architecture/operational-list-queries-and-retrieval-ergonomics.md`
- `docs/architecture/listing-filters-query-semantics-operator-usage-and-limitations.md`

## Test Results

All 23 new tests pass. All existing tests in affected packages remain green:

- `internal/adapters/clickhouse` — PASS
- `internal/application/analyticalclient` — PASS
- `internal/interfaces/http/handlers` — PASS
- `internal/interfaces/http/routes` — PASS

Full binary compilation verified for all three binaries (gateway, execute, writer).

## Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Operational query is materially more practical | Met | Three new endpoints eliminate partition-key-foreknowledge requirement |
| Improves audit and troubleshooting | Met | Status-based listing + summary + lifecycle list discovery |
| Historical surface is actually usable | Met | Relaxed filters enable ad-hoc queries without knowing exact keys |
| Ready for explainability surface (S455A) | Met | List queries provide the data discovery layer S455A needs |

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No BI/reporting inflation | Clean — summary is a single GROUP BY, not a reporting framework |
| No broad dashboards | Clean — endpoints are query APIs, not UI |
| No complex query DSL | Clean — fixed filter set with AND semantics only |
| No masked retrieval gaps | Clean — limitations documented explicitly in companion doc |

## Limitations

- No wildcard or pattern matching on filters
- No cursor-based pagination (offset/limit only)
- Summary groups by (type, status) only — no multi-dimensional breakdown
- KV lifecycle list shows latest-only, not historical
- No cross-table join queries

## What Follows

- **S455A:** Explainability surface — "why was this order rejected?" narratives
  built on top of the discovery queries introduced here.
