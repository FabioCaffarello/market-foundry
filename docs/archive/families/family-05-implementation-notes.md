# Family 05 — Implementation Notes

**Stage:** S187
**Family:** 05 — Executions (paper_order)
**Date:** 2026-03-20
**Pattern:** Wave B v2 (9-artifact template)

## Implementation Summary

Family 05 adds the execution analytical read path — the sixth and final layer in the trading pipeline (Evidence → Signals → Decisions → Strategies → Risk → Executions). This completes full vertical coverage of the analytical layer.

## Artifacts Delivered

### Pre-staged (no changes required)

| # | Artifact | File | Status |
|---|----------|------|--------|
| 1 | Schema migration | `deploy/migrations/006_create_executions.sql` | Pre-staged in S181 |
| 2 | Writer mapper | `cmd/writer/mappers.go` (mapExecutionRow) | Pre-staged in S148 |
| 3 | Writer pipeline entry | `cmd/writer/pipeline.go` | Pre-staged in S148 |

**Write-path changes: 0** — sixth consecutive family expansion with zero write-path modifications.

### Built in S187

| # | Artifact | File | LOC |
|---|----------|------|-----|
| 4 | ClickHouse reader | `internal/adapters/clickhouse/execution_reader.go` | 159 |
| 5 | Reader tests | `internal/adapters/clickhouse/execution_reader_test.go` | 243 |
| 6 | Use case + contracts | `internal/application/analyticalclient/get_execution_history.go` + `contracts.go` | 96 + 30 |
| 7 | Use case tests | `internal/application/analyticalclient/get_execution_history_test.go` | 188 |
| 8 | Handler method | `internal/interfaces/http/handlers/analytical.go` (GetExecutionHistory) | ~99 |
| 9 | Handler tests | `internal/interfaces/http/handlers/analytical_test.go` (execution section) | ~175 |
| 10 | Route registration | `internal/interfaces/http/routes/analytical.go` | ~15 |
| 11 | Gateway wiring | `cmd/gateway/analytical_reader.go` + `compose.go` | ~10 |

### Operational artifacts

| Artifact | File |
|----------|------|
| HTTP test queries | `tests/http/analytical.http` (entries 40–48) |
| Smoke script extension | `scripts/smoke-analytical-e2e.sh` (Family-05 section) |

## What's Novel in Family 05

| Dimension | Previous max | Family 05 | Notes |
|-----------|-------------|-----------|-------|
| DDL columns (SELECT) | 13 (risk) | 16 | Largest SELECT in read path |
| Float64 columns | 0 | 2 (quantity, filled_quantity) | First floating-point in read path — reuses FormatFloat |
| Boolean columns | 1 (final, shared) | 1 (final) | Already proven |
| JSON columns | 4 (risk) | 4 (risk, fills, parameters, metadata) | At proven ceiling |
| Optional filters per method | 1 | 2 (side, status) | First method with two optional filters |
| New parsers | 0 | 2 (ParseRiskInputJSON, ParseFillsJSON) | Total parser count: 8 |
| Handler file | 515 lines | 615 lines | Within 620-line ceiling |
| Reader method params | 8 (risk) | 10 | Largest reader signature |

## New Parsers

1. **ParseRiskInputJSON** — Deserializes JSON into `execution.RiskInput`. Pattern identical to `ParseConstraintsJSON` (struct from JSON). Falls back to zero-value on failure.

2. **ParseFillsJSON** — Deserializes JSON into `[]execution.FillRecord`. Pattern identical to `ParseStrategyInputsJSON` (slice from JSON). Falls back to empty slice on failure.

Both parsers are exported and independently tested with valid, empty, and malformed inputs.

## Creative Decisions

**Zero.** Every implementation choice followed the proven Wave B pattern mechanically.

## Simplifications

1. **No filter value validation** — Side and status filters are passed through to ClickHouse without validation. ClickHouse returns empty results for invalid values. This matches the pattern established by outcome (Family 02), direction (Family 03), and disposition (Family 04).

2. **FormatFloat reuse** — Float64 columns (quantity, filled_quantity) use the same FormatFloat helper proven for confidence fields. No new float handling needed.

3. **Reader source is "derive"** — Unlike previous families that use "binancef", execution events originate from the derive binary. HTTP test queries and smoke tests use `source=derive`.
