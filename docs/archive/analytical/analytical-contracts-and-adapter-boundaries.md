# Analytical Contracts and Adapter Boundaries

## Purpose

This document defines the contract surfaces and adapter boundaries for the analytical layer. It specifies who owns each contract, how adapters connect, and the rules that prevent boundary erosion.

## Contract Inventory

### 1. Application-Layer Contracts (`internal/application/analyticalclient/`)

These are the contracts consumed by HTTP handlers and composed by the gateway.

| Contract | Type | Owner | Consumers |
|----------|------|-------|-----------|
| `CandleHistoryQuery` | Request struct | analyticalclient | handlers, routes |
| `CandleHistoryReply` | Response struct | analyticalclient | handlers, routes |
| `CandleReader` | Interface | analyticalclient | use case, adapter |

**`CandleReader` interface:**
```go
type CandleReader interface {
    QueryCandleHistory(ctx context.Context, source, symbol string,
        timeframe int, since, until int64, limit int,
    ) ([]evidence.EvidenceCandle, error)
}
```

**Ownership rule:** The interface is defined in the application layer (consumer side), following Go's "accept interfaces, return structs" convention. The adapter implements this interface but does not import or reference it.

### 2. Adapter-Layer Contracts (`internal/adapters/clickhouse/`)

These are the storage-level contracts provided by the ClickHouse adapter.

| Contract | Type | Owner | Consumers |
|----------|------|-------|-----------|
| `Client` | Struct | clickhouse adapter | writer, gateway |
| `Config` | Struct | clickhouse adapter | writer, gateway |
| `Rows` | Interface | clickhouse adapter | CandleReader |
| `CandleReader` | Struct | clickhouse adapter | gateway composition |
| `BuildCandleQuery` | Function | clickhouse adapter | CandleReader, tests |
| `FormatFloat` | Function | clickhouse adapter | CandleReader, tests |

**Separation rule:** The adapter provides storage-level capabilities. It may import domain types (e.g., `evidence.EvidenceCandle`) for row↔domain mapping, but it must not import application-layer packages or HTTP/transport packages.

### 3. Domain Contracts (consumed, not owned)

The analytical layer uses domain types as the shared vocabulary between write and read paths.

| Contract | Package | Used By |
|----------|---------|---------|
| `evidence.EvidenceCandle` | internal/domain/evidence | writer mappers, reader, use case, handler |
| `evidence.CandleSampledEvent` | internal/domain/evidence | writer consumer |
| `signal.SignalGeneratedEvent` | internal/domain/signal | writer consumer |
| `decision.DecisionEvaluatedEvent` | internal/domain/decision | writer consumer |
| `strategy.StrategyResolvedEvent` | internal/domain/strategy | writer consumer |
| `risk.RiskAssessedEvent` | internal/domain/risk | writer consumer |
| `execution.PaperOrderSubmittedEvent` | internal/domain/execution | writer consumer |

**Rule:** Domain types are the lingua franca. Both write-path mappers and read-path readers translate between domain types and ClickHouse storage. Neither side references the other's translation code.

### 4. Schema Contracts (implicit, DDL-defined)

Column schemas are defined in migration DDL files and implicitly consumed by mappers and readers.

| Table | DDL Source | Write Consumer | Read Consumer |
|-------|-----------|----------------|---------------|
| evidence_candles | 001_create_evidence_candles.sql | cmd/writer/mappers.go `mapCandleRow` | internal/adapters/clickhouse/candle_reader.go |
| signals | 002_create_signals.sql | cmd/writer/mappers.go `mapSignalRow` | (not yet) |
| decisions | 003_create_decisions.sql | cmd/writer/mappers.go `mapDecisionRow` | (not yet) |
| strategies | 004_create_strategies.sql | cmd/writer/mappers.go `mapStrategyRow` | (not yet) |
| risk_assessments | 005_create_risk_assessments.sql | cmd/writer/mappers.go `mapRiskRow` | (not yet) |
| executions | 006_create_executions.sql | cmd/writer/mappers.go `mapExecutionRow` | (not yet) |

**Schema coherence rule:** Any DDL change to column names, types, or ordering must be reflected in both the write mapper and the read adapter. There is no compile-time enforcement today — coherence is validated by integration tests and reviewer discipline.

## Adapter Boundary Rules

### Rule AB-01: Adapter imports domain, never application
The ClickHouse adapter may import `internal/domain/*` for row↔domain type mapping. It must never import `internal/application/*` or `internal/interfaces/*`.

### Rule AB-02: Application defines interfaces, adapter provides structs
The `CandleReader` interface lives in `internal/application/analyticalclient/`. The `clickhouse.CandleReader` struct satisfies it. The adapter does not reference the interface — Go's structural typing handles the contract.

### Rule AB-03: Gateway composes, adapter translates
The gateway composition root (`cmd/gateway/`) connects adapters to use cases. It does not contain translation logic (query building, row scanning, float formatting). All translation lives in the adapter or domain layer.

### Rule AB-04: Writer mappers stay in the binary
Write-path mappers (`cmd/writer/mappers.go`) live in the writer binary because they are composition-specific: they wire NATS event decoding to ClickHouse row tuples. This is the write-side equivalent of the adapter-layer reader. The asymmetry is intentional — the writer's mapper is tightly coupled to the consumer/inserter actor model, while the reader's adapter is consumable by any binary.

### Rule AB-05: No cross-path imports
Write-path code (`cmd/writer/`) must not import read-path code (`candle_reader.go`), and vice versa. Both paths independently translate between domain types and ClickHouse storage. This prevents accidental coupling between read and write lifecycles.

### Rule AB-06: ClickHouse client is shared, readers are not
The `clickhouse.Client` struct is used by both writer (via `InsertBatch`) and reader (via `Query`). But `CandleReader` is a read-path-only type — it must not be used by the writer.

## Compile-Time Boundary Enforcement

| Check | Location | Mechanism |
|-------|----------|-----------|
| `CandleReader` satisfies `analyticalclient.CandleReader` | `cmd/gateway/analytical_reader_test.go` | `var _ analyticalclient.CandleReader = (*clickhouse.CandleReader)(nil)` |
| Writer pipeline families are known | `cmd/writer/run.go` | `config.Pipeline.ValidatePipeline()` at startup |
| ClickHouse config is structurally valid | `cmd/writer/run.go` | `config.ClickHouse.Validate()` at startup |

## Expansion Protocol

When adding a new analytical reader (e.g., signal history):

1. Create `signal_reader.go` in `internal/adapters/clickhouse/` with query builder + row scanner
2. Define `SignalReader` interface in `internal/application/analyticalclient/`
3. Create `GetSignalHistoryUseCase` in `internal/application/analyticalclient/`
4. Add handler method in `internal/interfaces/http/handlers/analytical.go`
5. Register route in `internal/interfaces/http/routes/analytical.go`
6. Wire in `cmd/gateway/compose.go` via `AnalyticalFamilyDeps`
7. Add compile-time interface assertion in `cmd/gateway/analytical_reader_test.go`

No step touches the writer. No step modifies existing read-path code for other families.
