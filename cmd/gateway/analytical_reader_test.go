package main

import (
	"internal/adapters/clickhouse"
	"internal/application/analyticalclient"
)

// Compile-time assertions: each ClickHouse adapter reader satisfies the
// corresponding analyticalclient interface used by the gateway composition root.
var _ analyticalclient.CandleReader = (*clickhouse.CandleReader)(nil)
var _ analyticalclient.SignalReader = (*clickhouse.SignalReader)(nil)
var _ analyticalclient.DecisionReader = (*clickhouse.DecisionReader)(nil)
var _ analyticalclient.StrategyReader = (*clickhouse.StrategyReader)(nil)
var _ analyticalclient.RiskReader = (*clickhouse.RiskReader)(nil)
var _ analyticalclient.ExecutionReader = (*clickhouse.ExecutionReader)(nil)
