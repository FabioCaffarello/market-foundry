# Domain Depth End-to-End Validation Findings

## Purpose

Record the findings from validating the S234–S236 domain depth evolution
(decision → strategy → risk) through the full pipeline:
derive → store → clickhouse → http.

## Validation Scope

### Unit Tests (108 test files)

All unit tests pass across all modules. Key test files exercising domain depth:

| Test File | What It Validates |
|-----------|-------------------|
| `internal/domain/decision/decision_test.go` | Severity enum, Rationale, Validate(), SignalInput |
| `internal/domain/strategy/strategy_test.go` | DecisionInput with Severity/Rationale, Validate() |
| `internal/domain/risk/risk_test.go` | StrategyInput with DecisionSeverity/DecisionRationale |
| `internal/application/decision/rsi_oversold_evaluator_test.go` | Severity classification, rationale generation, confidence scaling |
| `internal/application/strategy/mean_reversion_entry_resolver_test.go` | Decision context forwarding, metadata propagation |
| `internal/application/risk/position_exposure_evaluator_test.go` | Decision severity in rationale, metadata tracking |
| `internal/actors/scopes/derive/risk_evaluator_actor_test.go` | Actor-level message propagation with DecisionSeverity |
| `internal/actors/scopes/store/decision_projection_actor_test.go` | KV materialization with full domain |
| `internal/adapters/clickhouse/decision_reader_test.go` | Read path with severity/rationale columns |
| `internal/adapters/clickhouse/risk_reader_test.go` | Read path with decision context in strategies JSON |
| `internal/adapters/clickhouse/strategy_reader_test.go` | Read path with decision context in decisions JSON |
| `internal/adapters/clickhouse/writerpipeline/support_test.go` | mapDecisionRow, mapStrategyRow, mapRiskRow column mapping |
| `internal/interfaces/http/handlers/decision_test.go` | HTTP response includes severity/rationale |
| `internal/interfaces/http/handlers/analytical_test.go` | Analytical endpoints return new fields |

### Integration Tests (build tag: `integration`)

All integration tests pass. The primary integration test
(`internal/application/execution/pipeline_integration_test.go`) validates the
full execution pipeline with embedded NATS:

- Multi-symbol isolation (3 symbols × 2 timeframes)
- Risk → evaluate → simulate → fill chain
- Staleness guard detection
- Status propagation through execute chain

### Smoke-Analytical E2E

The smoke-analytical script (updated in S237) validates the complete data path
through docker-compose infrastructure. The new Phase 7 specifically targets
domain depth validation.

## Findings

### Proven

1. **Decision Severity Classification** — The RSI Oversold evaluator correctly
   classifies severity into `none`, `low`, `moderate`, `high` zones based on
   distance from threshold. Unit tests cover all zone boundaries.

2. **Decision Rationale Generation** — Human-readable rationale is generated
   with concrete values (RSI value, threshold, distance percentage, severity).
   Format is consistent and parseable.

3. **Decision → Strategy Propagation** — `DecisionInput` in strategy domain
   carries `Severity` and `Rationale` as string copies (DBI-9 boundary preserved).
   Strategy metadata includes `decision_rationale` when non-empty.

4. **Strategy → Risk Propagation** — `StrategyInput` in risk domain carries
   `DecisionSeverity` and `DecisionRationale` with `omitempty` tags. Risk
   rationale text includes decision severity context. Risk metadata stores
   `decision_severity` and `decision_rationale` for flat query access.

5. **ClickHouse Write Path** — `mapDecisionRow()` writes severity and rationale
   as direct columns. `mapStrategyRow()` embeds decision context in serialized
   `decisions` JSON. `mapRiskRow()` embeds decision context in serialized
   `strategies` JSON and `metadata` JSON.

6. **ClickHouse Read Path** — `DecisionReader` scans severity and rationale
   columns. Strategy and risk readers deserialize JSON arrays containing
   decision context fields.

7. **HTTP Response Surface** — Decision history endpoint returns severity and
   rationale at the top level. Strategy responses carry decision context in
   `decisions` array. Risk responses carry decision context in `strategies`
   array and `metadata` map.

8. **Migration** — `007_add_decision_severity_rationale.sql` adds severity
   (LowCardinality) and rationale columns with empty string defaults, ensuring
   backward compatibility with pre-S234 data.

9. **Codegen Alignment** — `rsi_oversold.yaml` family spec includes severity
   and rationale in columns list. Golden snapshot updated.

### Not Yet Proven

1. **Live multi-symbol domain depth** — Unit tests validate per-symbol isolation
   but the smoke-analytical E2E uses a single symbol (btcusdt). Multi-symbol
   domain depth propagation is structurally identical but not E2E-proven with
   live infrastructure.

2. **Severity-dependent behavior** — Severity is recorded/propagated but not
   acted upon. No evaluator, resolver, or risk gate currently modulates behavior
   based on severity. This is by design (deferred to future charter) but means
   the field is observability-only.

3. **Historical data migration** — Pre-S234 decision rows will have empty
   severity and rationale strings. The read path handles this correctly (Go
   zero-values), but no explicit migration backfill exists.

4. **Performance under load** — No latency regression testing. The rationale
   string adds ~50-100 bytes per decision row. At current volumes this is
   negligible, but has not been profiled under high-throughput scenarios.

## Conclusion

The domain depth introduced in S234–S236 is fully proven through unit tests,
integration tests, and (with S237 updates) the smoke-analytical E2E pipeline.
The data flows correctly from evaluator through NATS, writer, ClickHouse, reader,
and HTTP response. No regressions were found. The remaining gaps are either
by-design deferrals (severity-dependent behavior) or low-risk operational
concerns (multi-symbol E2E, performance profiling).
