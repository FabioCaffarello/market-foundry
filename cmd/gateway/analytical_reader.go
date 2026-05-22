package main

import (
	"log/slog"

	"internal/adapters/clickhouse"
	"internal/application/analyticalclient"
)

// newAnalyticalCandleReader creates the analytical candle reader from the adapter layer.
// The gateway composition root delegates to the ClickHouse adapter's CandleReader,
// which owns the storage↔domain translation. This function exists only to satisfy
// the analyticalclient.CandleReader interface contract at the composition boundary.
func newAnalyticalCandleReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.CandleReader {
	return clickhouse.NewCandleReader(client, logger)
}

// newAnalyticalSignalReader creates the analytical signal reader from the adapter layer.
// Same pattern as candle reader — the adapter owns storage↔domain translation,
// the gateway satisfies the analyticalclient.SignalReader interface at the composition boundary.
func newAnalyticalSignalReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.SignalReader {
	return clickhouse.NewSignalReader(client, logger)
}

// newAnalyticalDecisionReader creates the analytical decision reader from the adapter layer.
// Same pattern as candle/signal readers — the adapter owns storage↔domain translation,
// the gateway satisfies the analyticalclient.DecisionReader interface at the composition boundary.
func newAnalyticalDecisionReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.DecisionReader {
	return clickhouse.NewDecisionReader(client, logger)
}

// newAnalyticalStrategyReader creates the analytical strategy reader from the adapter layer.
// Same pattern as candle/signal/decision readers — the adapter owns storage↔domain translation,
// the gateway satisfies the analyticalclient.StrategyReader interface at the composition boundary.
func newAnalyticalStrategyReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.StrategyReader {
	return clickhouse.NewStrategyReader(client, logger)
}

// newAnalyticalRiskReader creates the analytical risk reader from the adapter layer.
// Same pattern as candle/signal/decision/strategy readers — the adapter owns storage↔domain translation,
// the gateway satisfies the analyticalclient.RiskReader interface at the composition boundary.
func newAnalyticalRiskReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.RiskReader {
	return clickhouse.NewRiskReader(client, logger)
}

// newAnalyticalLifecycleReader creates the analytical lifecycle history reader from the adapter layer.
// S453A: Unlike the execution reader (per-type queries), the lifecycle reader queries across
// all execution event types for unified timeline reconstruction.
func newAnalyticalLifecycleReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.LifecycleHistoryReader {
	return clickhouse.NewExecutionReader(client, logger)
}

// newAnalyticalCompositeReader creates the composite execution chain reader from the adapter layer.
// The CompositeReader queries all five domain tables by correlation_id and assembles
// a unified causal chain. It satisfies both CompositeReader (chain queries) and
// AggregationReader (funnel/disposition queries) interfaces.
func newAnalyticalCompositeReader(client *clickhouse.Client, logger *slog.Logger) *clickhouse.CompositeReader {
	return clickhouse.NewCompositeReader(client, logger)
}
