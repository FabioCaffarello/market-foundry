package main

import (
	"io"
	"log/slog"

	adapterch "internal/adapters/clickhouse"
	adapternats "internal/adapters/nats"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// writerPipeline describes one consumer-inserter pair in the writer.
type writerPipeline struct {
	family       string
	consumerName string
	inserterName string
	table        string
	insertSQL    string
	consumerSpec adapternats.ConsumerSpec
	isEnabled    func(settings.PipelineConfig) bool
	// startConsumer creates the NATS consumer, wiring it to send rows to the inserter.
	// The actor.Context is the consumer actor's context for sending messages.
	startConsumer func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error)
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
		evidence  adapternats.EvidenceRegistry
		signal    adapternats.SignalRegistry
		decision  adapternats.DecisionRegistry
		strategy  adapternats.StrategyRegistry
		risk      adapternats.RiskRegistry
		execution adapternats.ExecutionRegistry
	}{
		evidence:  adapternats.DefaultEvidenceRegistry(),
		signal:    adapternats.DefaultSignalRegistry(),
		decision:  adapternats.DefaultDecisionRegistry(),
		strategy:  adapternats.DefaultStrategyRegistry(),
		risk:      adapternats.DefaultRiskRegistry(),
		execution: adapternats.DefaultExecutionRegistry(),
	}

	return []writerPipeline{
		// ── manual:owned — Evidence families ─────────────────────
		// Ownership: human-maintained. Not codegen-governed.
		// Reason: evidence layer has unique naming conventions and
		// event type variations that require architectural decisions.

		// ── Evidence: candle → evidence_candles ──────────────────
		{
			family:       "candle",
			consumerName: "writer-candle-consumer",
			inserterName: "writer-candle-inserter",
			table:        "evidence_candles",
			insertSQL:    "INSERT INTO evidence_candles",
			consumerSpec: adapternats.WriterCandleConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("candle") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewEvidenceConsumer(natsURL, spec, reg.evidence,
					func(event evidence.CandleSampledEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapCandleRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},

		// codegen:begin pipeline_entry family=rsi source=codegen/families/rsi.yaml
		// ── Signal: rsi → signals ───────────────────────────────
		{
			family:       "rsi",
			consumerName: "writer-signal-rsi-consumer",
			inserterName: "writer-signal-rsi-inserter",
			table:        "signals",
			insertSQL:    "INSERT INTO signals",
			consumerSpec: adapternats.WriterRSISignalConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("rsi") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewSignalConsumer(natsURL, spec, reg.signal,
					func(event signal.SignalGeneratedEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapSignalRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},
		// codegen:end pipeline_entry family=rsi

		// codegen:begin pipeline_entry family=ema source=codegen/families/ema.yaml
		// ── Signal: ema → signals ──
		{
			family:       "ema",
			consumerName: "writer-signal-ema-consumer",
			inserterName: "writer-signal-ema-inserter",
			table:        "signals",
			insertSQL:    "INSERT INTO signals",
			consumerSpec: adapternats.WriterEMASignalConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("ema") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewSignalConsumer(natsURL, spec, reg.signal,
					func(event signal.SignalGeneratedEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapSignalRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},
		// codegen:end pipeline_entry family=ema

		// ── manual:owned — Decision, Strategy, Risk, Execution families ──
		// Ownership: human-maintained. Not codegen-governed.
		// Reason: these families have codegen specs and golden snapshots
		// but integration has not been authorized. They remain manual
		// until a future stage explicitly migrates them.

		// ── Decision: rsi_oversold → decisions ──────────────────
		{
			family:       "rsi_oversold",
			consumerName: "writer-decision-rsi-oversold-consumer",
			inserterName: "writer-decision-rsi-oversold-inserter",
			table:        "decisions",
			insertSQL:    "INSERT INTO decisions",
			consumerSpec: adapternats.WriterRSIOversoldDecisionConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsDecisionFamilyEnabled("rsi_oversold") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewDecisionConsumer(natsURL, spec, reg.decision,
					func(event decision.DecisionEvaluatedEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapDecisionRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},

		// ── Strategy: mean_reversion_entry → strategies ─────────
		{
			family:       "mean_reversion_entry",
			consumerName: "writer-strategy-mean-reversion-entry-consumer",
			inserterName: "writer-strategy-mean-reversion-entry-inserter",
			table:        "strategies",
			insertSQL:    "INSERT INTO strategies",
			consumerSpec: adapternats.WriterMeanReversionEntryStrategyConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("mean_reversion_entry") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewStrategyConsumer(natsURL, spec, reg.strategy,
					func(event strategy.StrategyResolvedEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapStrategyRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},

		// ── Risk: position_exposure → risk_assessments ──────────
		{
			family:       "position_exposure",
			consumerName: "writer-risk-position-exposure-consumer",
			inserterName: "writer-risk-position-exposure-inserter",
			table:        "risk_assessments",
			insertSQL:    "INSERT INTO risk_assessments",
			consumerSpec: adapternats.WriterPositionExposureRiskConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsRiskFamilyEnabled("position_exposure") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewRiskConsumer(natsURL, spec, reg.risk,
					func(event risk.RiskAssessedEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapRiskRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},

		// ── Execution: paper_order → executions ─────────────────
		{
			family:       "paper_order",
			consumerName: "writer-execution-paper-order-consumer",
			inserterName: "writer-execution-paper-order-inserter",
			table:        "executions",
			insertSQL:    "INSERT INTO executions",
			consumerSpec: adapternats.WriterPaperOrderExecutionConsumer(),
			isEnabled:    func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("paper_order") },
			startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
				consumer := adapternats.NewExecutionConsumer(natsURL, spec, reg.execution,
					func(event execution.PaperOrderSubmittedEvent) {
						if tracker != nil {
							tracker.RecordEvent()
							tracker.Counter("events_received").Add(1)
						}
						actorCtx.Send(inserterPID, insertRowMsg{row: mapExecutionRow(event)})
					},
					logger,
				)
				return consumer, consumer.Start()
			},
		},
	}
}
