# Analytical End-to-End Validation Findings

## Summary

The integration proof script (`scripts/smoke-analytical-e2e.sh`) was designed and implemented to validate the complete analytical data path. This document captures findings, gaps, and observations from the validation design process.

## Findings

### F-01: Writer not in BUILDABLE_SERVICES (fixed)

**Observation**: The Makefile's `BUILDABLE_SERVICES` list did not include `writer`, meaning `make build` and `make docker-build` silently skipped the writer binary.

**Impact**: Developer building all services via `make build` would not get a writer binary. Docker builds via compose worked because compose has an explicit writer service definition.

**Resolution**: Added `writer` to `BUILDABLE_SERVICES` in Makefile.

### F-02: lib.sh missing writer service (fixed)

**Observation**: `scripts/utils/lib.sh` did not include writer in `ALL_SERVICES` or `SVC_PORTS`. Scripts sourcing lib.sh had no standard reference for the writer port.

**Impact**: Any future script using lib.sh service iteration would skip writer.

**Resolution**: Added `writer` to `ALL_SERVICES` and `SVC_PORTS` (port 8085).

### F-03: No standalone analytical smoke target (fixed)

**Observation**: Existing `make smoke` and `make smoke-multi` validate only the operational path (NATS KV → latest candle). No target exercised the analytical path.

**Impact**: The analytical layer could silently break without any automated detection.

**Resolution**: Added `make smoke-analytical` target invoking `scripts/smoke-analytical-e2e.sh`.

### F-04: Reader has zero observability (unchanged — known debt)

**Observation**: The reader path (CandleReader → use case → HTTP handler) has no structured logging, no query timing, and no error counters. This was identified in S156 and S157 as an open debt.

**Impact**: If the reader starts failing or slowing down, there is no diagnostic signal beyond HTTP status codes.

**Status**: Intentionally deferred. This proof validates correctness, not observability. Reader instrumentation is recommended for the next wave.

### F-05: Migration application is manual (accepted)

**Observation**: Migrations must be applied before writer can insert. The proof script checks for tables but does not apply migrations itself.

**Impact**: A fresh environment requires `make migrate-up` before the analytical layer works.

**Status**: Accepted. Migrations are a deliberate, auditable step. Auto-migration at startup was explicitly rejected in the migration architecture design.

### F-06: Batch flush timing creates test non-determinism

**Observation**: The writer flushes batches either when `batch_size` (1000) rows accumulate or when `flush_interval` (5s) expires. In low-traffic environments, the first flush may take longer than expected.

**Impact**: The proof script must poll and wait rather than assert immediately. The default 120s wait accommodates the full pipeline warmup (ingest connect → derive sample → NATS publish → writer consume → batch flush).

**Mitigation**: Configurable `--wait` parameter. The script provides incremental status during the wait.

### F-07: Non-candle families not exercised by proof (accepted)

**Observation**: Only evidence_candles is fully proven (write + read + HTTP). Other tables (signals, decisions, strategies, risk_assessments, executions) are checked for row counts but not for HTTP read path.

**Reason**: Only candles have a reader and HTTP endpoint implemented. Other families have write path only. The proof script reports row counts for visibility.

**Status**: Accepted. Expanding the read path is future scope, not S159 scope.

### F-08: Gateway ClickHouse dependency is optional and not in readiness

**Observation**: Gateway readiness (`/readyz`) does not check ClickHouse connectivity (R-02 compliance). The analytical endpoint returns 503 when ClickHouse is unavailable.

**Impact**: Gateway can report "ready" while analytical endpoints are broken. The proof script explicitly checks the analytical endpoint availability (not 503) as a separate step.

**Status**: This is by design. The operational path must not depend on the analytical layer. The proof script correctly validates both independently.

## Boundary coherence observations

### Writer boundaries (confirmed sound)

- Writer has its own Go module (`cmd/writer/go.mod`)
- Writer depends only on NATS + ClickHouse, not on gateway or store
- Writer consumers use dedicated durable names (`writer-*` prefix), no conflict with store consumers
- Writer config validation at startup catches misconfigurations early (S158 fix)

### Reader boundaries (confirmed sound after S158)

- Reader implementation lives in `internal/adapters/clickhouse/candle_reader.go` (adapter layer)
- Gateway delegates to reader via thin bridge (`cmd/gateway/analytical_reader.go`, 15 lines)
- Compile-time interface assertion ensures CandleReader contract stability
- Reader never writes to ClickHouse, writer never reads

### Gateway analytical surface (confirmed minimal)

- Single endpoint: `GET /analytical/evidence/candles`
- Query params: source, symbol, timeframe (required), limit, since, until (optional)
- Response: `{ candles: [...], source: "clickhouse" }`
- Error handling: 400 for invalid params, 503 when ClickHouse unavailable

## Risk assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Reader failure invisible (no counters) | Medium | Add reader instrumentation in next wave |
| Batch flush non-determinism in tests | Low | Polling with configurable timeout |
| Non-candle families untested E2E | Low | Same machinery; candle proof is representative |
| Migration drift undetected | Low | `make migrate-validate` checks checksums |
| Writer degradation silent to gateway | Medium | Writer /statusz + /diagz expose state; gateway independent by design |
