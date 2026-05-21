# Stage S158: Writer–Reader–Gateway Boundary Hardening Report

## Executive Summary

S158 applied targeted boundary hardening to the analytical layer, resolving the top boundary issue identified in S157 (reader implementation misplaced in gateway) and closing the writer config validation gap flagged in S156. No functionality was added or expanded. The analytical layer is now structurally cleaner, with explicit adapter boundaries, compile-time contract enforcement, and fail-fast config validation on the write path.

## Deliverables

| # | Deliverable | Status |
|---|-------------|--------|
| 1 | Reader extracted from gateway to adapter layer | Done |
| 2 | Writer startup config validation | Done |
| 3 | Compile-time interface contract assertion | Done |
| 4 | `docs/architecture/analytical-boundary-hardening-writer-reader-gateway.md` | Done |
| 5 | `docs/architecture/analytical-contracts-and-adapter-boundaries.md` | Done |
| 6 | This report | Done |

## Boundary Hardening Applied

### H-01: Reader Extraction (primary boundary fix)

**Issue:** `analyticalCandleReader` lived in `cmd/gateway/analytical_reader.go` as an unexported `main` package type, embedding query construction, row scanning, and float formatting inside the gateway composition root.

**Fix:** Moved to `internal/adapters/clickhouse/candle_reader.go` as exported `CandleReader` struct. Gateway's `analytical_reader.go` reduced to a one-line factory delegating to the adapter.

**Files changed:**
- `internal/adapters/clickhouse/candle_reader.go` — new, 106 lines
- `internal/adapters/clickhouse/candle_reader_test.go` — new, 165 lines (migrated from gateway)
- `cmd/gateway/analytical_reader.go` — rewritten to thin delegation (15 lines)
- `cmd/gateway/analytical_reader_test.go` — rewritten to compile-time contract assertion

**Impact:** Reader is now testable outside `main`, reusable by other binaries, and follows the adapter pattern consistently.

### H-02: Writer Config Validation at Startup

**Issue:** Writer started pipelines without validating ClickHouse config structure (batch_size, flush_interval, etc.) or pipeline family names (could enable unknown families silently).

**Fix:** Added `config.ClickHouse.Validate()` and `config.Pipeline.ValidatePipeline()` calls in `cmd/writer/run.go` before opening the ClickHouse connection.

**Files changed:**
- `cmd/writer/run.go` — added validation block (10 lines)

**Impact:** Misconfigured writer now fails fast with descriptive error instead of launching with invalid parameters.

### H-03: Compile-Time Interface Assertion

**Issue:** No static check that the adapter's `CandleReader` satisfies the application-layer `analyticalclient.CandleReader` interface. Interface drift could cause runtime errors.

**Fix:** Added `var _ analyticalclient.CandleReader = (*clickhouse.CandleReader)(nil)` in `cmd/gateway/analytical_reader_test.go`.

**Impact:** Build breaks immediately if the adapter struct drifts from the interface contract.

## Files Changed

| File | Action | Lines |
|------|--------|-------|
| `internal/adapters/clickhouse/candle_reader.go` | Created | 106 |
| `internal/adapters/clickhouse/candle_reader_test.go` | Created | 165 |
| `cmd/gateway/analytical_reader.go` | Rewritten | 15 |
| `cmd/gateway/analytical_reader_test.go` | Rewritten | 10 |
| `cmd/writer/run.go` | Modified | +10 |
| `docs/architecture/analytical-boundary-hardening-writer-reader-gateway.md` | Created | — |
| `docs/architecture/analytical-contracts-and-adapter-boundaries.md` | Created | — |
| `docs/stages/stage-s158-writer-reader-gateway-boundary-hardening-report.md` | Created | — |

## Patterns Reinforced

| Pattern | Description |
|---------|-------------|
| Adapter owns translation | Storage↔domain mapping lives in the adapter layer, not in binaries |
| Composition root only composes | Gateway wires dependencies, delegates all logic to adapters and use cases |
| Accept interfaces, return structs | Application defines `CandleReader` interface; adapter provides `CandleReader` struct |
| Fail-fast validation | Writer validates config at startup before resource allocation |
| Compile-time contracts | Interface assertions catch drift at build time, not runtime |

## Limits Maintained

- No new analytical families or query types added
- No schema changes or migrations
- No changes to the write path (mappers, inserter, supervisor, consumer)
- ClickHouse optionality fully preserved (R-01 through R-10 unchanged)
- No observability changes (reader instrumentation deferred to future stage)
- Write-path mappers remain in `cmd/writer/` — asymmetry with read path is intentional and documented

## Test Results

All existing and new tests pass:
- `internal/adapters/clickhouse/...` — 8 tests (query builder + format float)
- `internal/application/analyticalclient/...` — 8 tests (use case validation)
- `internal/interfaces/http/handlers/...` — all handler tests (including 5 analytical)
- `cmd/gateway/...` — compile-time interface assertion
- `cmd/writer/...` — builds successfully with validation

## Open Debts Carried Forward

| Debt | Severity | Origin |
|------|----------|--------|
| Reader has no observability counters | Medium | S156-precondition |
| No integration test for full data path | Medium | S156-precondition |
| Schema coherence is reviewer-enforced, not compile-time | Low | S157 |
| Write-path mappers not yet in adapter layer | Low | S157 (intentional) |

## Preparation for S159

The boundary hardening in S158 creates a cleaner foundation for the next stage. Recommended S159 focus areas:

1. **End-to-end integration test** — With boundaries now explicit, an integration test can verify the full path: NATS → writer → ClickHouse → reader → HTTP. The reader adapter's testability in isolation (no `main` package) makes this feasible.

2. **Reader observability** — Add basic counters to `CandleReader.QueryCandleHistory` (query_total, query_duration_ms, query_errors). The adapter-layer location makes instrumentation straightforward.

3. **Additional analytical readers** — The expansion protocol documented in `analytical-contracts-and-adapter-boundaries.md` defines the steps for adding signal/decision/strategy/risk/execution history readers. Each follows the same adapter → use case → handler → route → compose pattern.

4. **Schema coherence tests** — Consider a test that verifies write-path mapper column order matches read-path SELECT column order, catching DDL drift without requiring ClickHouse.
