package main

import (
	adapterch "internal/adapters/clickhouse"
	writerpipeline "internal/adapters/clickhouse/writerpipeline"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/shared/settings"
)

// writerPipeline describes one consumer-inserter pair in the writer.
type writerPipeline struct {
	family        string
	consumerName  string
	inserterName  string
	table         string
	insertSQL     string
	consumerSpec  natskit.ConsumerSpec
	isEnabled     func(settings.PipelineConfig) bool
	startConsumer writerpipeline.ConsumerStarter
}

// writerTrackerDef describes the tracker pair for one writer pipeline.
type writerTrackerDef struct {
	consumerName string
	inserterName string
	isEnabled    func(settings.PipelineConfig) bool
}

// writerTrackerDefs returns tracker definitions for all writer pipelines.
func writerTrackerDefs() []writerTrackerDef {
	pipelines := declareWriterPipelines(nil)
	defs := make([]writerTrackerDef, len(pipelines))
	for i, p := range pipelines {
		defs[i] = writerTrackerDef{
			consumerName: p.consumerName,
			inserterName: p.inserterName,
			isEnabled:    p.isEnabled,
		}
	}
	return defs
}

// declareWriterPipelines returns all writer pipeline definitions.
// The chClient can be nil (used only for tracker def extraction).
func declareWriterPipelines(chClient *adapterch.Client) []writerPipeline {
	reg := struct {
		evidence  natsevidence.Registry
		signal    natssignal.Registry
		decision  natsdecision.Registry
		strategy  natsstrategy.Registry
		risk      natsrisk.Registry
		execution natsexecution.Registry
	}{
		evidence:  natsevidence.DefaultRegistry(),
		signal:    natssignal.DefaultRegistry(),
		decision:  natsdecision.DefaultRegistry(),
		strategy:  natsstrategy.DefaultRegistry(),
		risk:      natsrisk.DefaultRegistry(),
		execution: natsexecution.DefaultRegistry(),
	}

	return []writerPipeline{
		// codegen:begin pipeline_entry family=candle source=codegen/families/candle.yaml
		// ── Evidence: candle → evidence_candles ──
		{
			family:        "candle",
			consumerName:  "writer-candle-consumer",
			inserterName:  "writer-candle-inserter",
			table:         "evidence_candles",
			insertSQL:     "INSERT INTO evidence_candles (event_id, occurred_at, correlation_id, causation_id, source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final)",
			consumerSpec:  natsevidence.WriterCandleConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("candle") },
			startConsumer: writerpipeline.NewCandleStarter(reg.evidence),
		},
		// codegen:end pipeline_entry family=candle

		// codegen:begin pipeline_entry family=rsi source=codegen/families/rsi.yaml
		// ── Signal: rsi → signals ───────────────────────────────
		{
			family:        "rsi",
			consumerName:  "writer-signal-rsi-consumer",
			inserterName:  "writer-signal-rsi-inserter",
			table:         "signals",
			insertSQL:     "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp)",
			consumerSpec:  natssignal.WriterRSISignalConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("rsi") },
			startConsumer: writerpipeline.NewSignalStarter(reg.signal),
		},
		// codegen:end pipeline_entry family=rsi

		// codegen:begin pipeline_entry family=ema source=codegen/families/ema.yaml
		// ── Signal: ema → signals ──
		{
			family:        "ema",
			consumerName:  "writer-signal-ema-consumer",
			inserterName:  "writer-signal-ema-inserter",
			table:         "signals",
			insertSQL:     "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp)",
			consumerSpec:  natssignal.WriterEMASignalConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("ema") },
			startConsumer: writerpipeline.NewSignalStarter(reg.signal),
		},
		// codegen:end pipeline_entry family=ema

		// codegen:begin pipeline_entry family=bollinger source=codegen/families/bollinger.yaml
		// ── Signal: bollinger → signals ──
		{
			family:        "bollinger",
			consumerName:  "writer-signal-bollinger-consumer",
			inserterName:  "writer-signal-bollinger-inserter",
			table:         "signals",
			insertSQL:     "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp)",
			consumerSpec:  natssignal.WriterBollingerSignalConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("bollinger") },
			startConsumer: writerpipeline.NewSignalStarter(reg.signal),
		},
		// codegen:end pipeline_entry family=bollinger

		// codegen:begin pipeline_entry family=macd source=codegen/families/macd.yaml
		// ── Signal: macd → signals ──
		{
			family:        "macd",
			consumerName:  "writer-signal-macd-consumer",
			inserterName:  "writer-signal-macd-inserter",
			table:         "signals",
			insertSQL:     "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp)",
			consumerSpec:  natssignal.WriterMACDSignalConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("macd") },
			startConsumer: writerpipeline.NewSignalStarter(reg.signal),
		},
		// codegen:end pipeline_entry family=macd

		// codegen:begin pipeline_entry family=vwap source=codegen/families/vwap.yaml
		// ── Signal: vwap → signals ──
		{
			family:        "vwap",
			consumerName:  "writer-signal-vwap-consumer",
			inserterName:  "writer-signal-vwap-inserter",
			table:         "signals",
			insertSQL:     "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp)",
			consumerSpec:  natssignal.WriterVWAPSignalConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("vwap") },
			startConsumer: writerpipeline.NewSignalStarter(reg.signal),
		},
		// codegen:end pipeline_entry family=vwap

		// codegen:begin pipeline_entry family=atr source=codegen/families/atr.yaml
		// ── Signal: atr → signals ──
		{
			family:        "atr",
			consumerName:  "writer-signal-atr-consumer",
			inserterName:  "writer-signal-atr-inserter",
			table:         "signals",
			insertSQL:     "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp)",
			consumerSpec:  natssignal.WriterATRSignalConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("atr") },
			startConsumer: writerpipeline.NewSignalStarter(reg.signal),
		},
		// codegen:end pipeline_entry family=atr

		// codegen:begin pipeline_entry family=rsi_oversold source=codegen/families/rsi_oversold.yaml
		// ── Decision: rsi_oversold → decisions ──
		{
			family:        "rsi_oversold",
			consumerName:  "writer-decision-rsi-oversold-consumer",
			inserterName:  "writer-decision-rsi-oversold-inserter",
			table:         "decisions",
			insertSQL:     "INSERT INTO decisions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, outcome, confidence, severity, rationale, signals, metadata, final, timestamp)",
			consumerSpec:  natsdecision.WriterRSIOversoldDecisionConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsDecisionFamilyEnabled("rsi_oversold") },
			startConsumer: writerpipeline.NewDecisionStarter(reg.decision),
		},
		// codegen:end pipeline_entry family=rsi_oversold

		// codegen:begin pipeline_entry family=ema_crossover source=codegen/families/ema_crossover.yaml
		// ── Decision: ema_crossover → decisions ──
		{
			family:        "ema_crossover",
			consumerName:  "writer-decision-ema-crossover-consumer",
			inserterName:  "writer-decision-ema-crossover-inserter",
			table:         "decisions",
			insertSQL:     "INSERT INTO decisions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, outcome, confidence, severity, rationale, signals, metadata, final, timestamp)",
			consumerSpec:  natsdecision.WriterEMACrossoverDecisionConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsDecisionFamilyEnabled("ema_crossover") },
			startConsumer: writerpipeline.NewDecisionStarter(reg.decision),
		},
		// codegen:end pipeline_entry family=ema_crossover

		// codegen:begin pipeline_entry family=mean_reversion_entry source=codegen/families/mean_reversion_entry.yaml
		// ── Strategy: mean_reversion_entry → strategies ──
		{
			family:        "mean_reversion_entry",
			consumerName:  "writer-strategy-mean-reversion-entry-consumer",
			inserterName:  "writer-strategy-mean-reversion-entry-inserter",
			table:         "strategies",
			insertSQL:     "INSERT INTO strategies (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp)",
			consumerSpec:  natsstrategy.WriterMeanReversionEntryStrategyConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("mean_reversion_entry") },
			startConsumer: writerpipeline.NewStrategyStarter(reg.strategy),
		},
		// codegen:end pipeline_entry family=mean_reversion_entry

		// codegen:begin pipeline_entry family=trend_following_entry source=codegen/families/trend_following_entry.yaml
		// ── Strategy: trend_following_entry → strategies ──
		{
			family:        "trend_following_entry",
			consumerName:  "writer-strategy-trend-following-entry-consumer",
			inserterName:  "writer-strategy-trend-following-entry-inserter",
			table:         "strategies",
			insertSQL:     "INSERT INTO strategies (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp)",
			consumerSpec:  natsstrategy.WriterTrendFollowingEntryStrategyConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("trend_following_entry") },
			startConsumer: writerpipeline.NewStrategyStarter(reg.strategy),
		},
		// codegen:end pipeline_entry family=trend_following_entry

		// ── Strategy: squeeze_breakout_entry → strategies ──
		{
			family:        "squeeze_breakout_entry",
			consumerName:  "writer-strategy-squeeze-breakout-entry-consumer",
			inserterName:  "writer-strategy-squeeze-breakout-entry-inserter",
			table:         "strategies",
			insertSQL:     "INSERT INTO strategies (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp)",
			consumerSpec:  natsstrategy.WriterSqueezeBreakoutEntryStrategyConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("squeeze_breakout_entry") },
			startConsumer: writerpipeline.NewStrategyStarter(reg.strategy),
		},

		// codegen:begin pipeline_entry family=position_exposure source=codegen/families/position_exposure.yaml
		// ── Risk: position_exposure → risk_assessments ──
		{
			family:        "position_exposure",
			consumerName:  "writer-risk-position-exposure-consumer",
			inserterName:  "writer-risk-position-exposure-inserter",
			table:         "risk_assessments",
			insertSQL:     "INSERT INTO risk_assessments (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp)",
			consumerSpec:  natsrisk.WriterPositionExposureRiskConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsRiskFamilyEnabled("position_exposure") },
			startConsumer: writerpipeline.NewRiskStarter(reg.risk),
		},
		// codegen:end pipeline_entry family=position_exposure

		// codegen:begin pipeline_entry family=drawdown_limit source=codegen/families/drawdown_limit.yaml
		// ── Risk: drawdown_limit → risk_assessments ──
		{
			family:        "drawdown_limit",
			consumerName:  "writer-risk-drawdown-limit-consumer",
			inserterName:  "writer-risk-drawdown-limit-inserter",
			table:         "risk_assessments",
			insertSQL:     "INSERT INTO risk_assessments (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp)",
			consumerSpec:  natsrisk.WriterDrawdownLimitRiskConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsRiskFamilyEnabled("drawdown_limit") },
			startConsumer: writerpipeline.NewRiskStarter(reg.risk),
		},
		// codegen:end pipeline_entry family=drawdown_limit

		// codegen:begin pipeline_entry family=paper_order source=codegen/families/paper_order.yaml
		// ── Execution: paper_order → executions ──
		{
			family:        "paper_order",
			consumerName:  "writer-execution-paper-order-consumer",
			inserterName:  "writer-execution-paper-order-inserter",
			table:         "executions",
			insertSQL:     "INSERT INTO executions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp)",
			consumerSpec:  natsexecution.WriterPaperOrderExecutionConsumer(),
			isEnabled:     func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("paper_order") },
			startConsumer: writerpipeline.NewExecutionStarter(reg.execution),
		},
		// codegen:end pipeline_entry family=paper_order
	}
}
