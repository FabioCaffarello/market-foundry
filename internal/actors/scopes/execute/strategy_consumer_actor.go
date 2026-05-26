package execute

import (
	"fmt"
	"log/slog"
	"strconv"

	actorcommon "internal/actors/common"
	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/metrics"

	"github.com/anthdm/hollywood/actor"
)

// DefaultMaxPositionPct is the default position size as a fraction of portfolio.
// S360: Configurable via StrategyConsumerConfig; defaults to 1% for paper safety.
const DefaultMaxPositionPct = "0.01"

// StrategyConsumerConfig holds the configuration for the strategy consumer actor.
type StrategyConsumerConfig struct {
	// MaxPositionPct is the maximum position size as a decimal string (e.g., "0.01" = 1%).
	MaxPositionPct string
	// MinConfidence is the minimum strategy confidence required for evaluation.
	// Events with confidence below this threshold are skipped with "skipped_low_confidence".
	// A value of "" or "0" disables the threshold (all events are evaluated).
	MinConfidence string
	// Tracker for health and counter metrics.
	Tracker *healthz.Tracker
	// AdapterPID is the venue adapter actor that receives produced intents.
	AdapterPID *actor.PID
}

// StrategyConsumerActor evaluates StrategyResolvedEvent messages via PaperOrderEvaluator
// and forwards produced ExecutionIntents to the venue adapter actor.
//
// S360: This is the canonical wiring point between the strategy domain and the execution path.
// The actor:
//   - receives strategy events from the natsstrategy.Consumer handler closure
//   - evaluates via PaperOrderEvaluator with pass-through risk (INV-4)
//   - preserves correlation/causation chain (INV-3)
//   - uses strategy.Timestamp, not time.Now() (INV-5)
//   - produces no execution for flat direction (INV-7)
//   - forwards intent to venue adapter for safety gates and submission
type StrategyConsumerActor struct {
	cfg    StrategyConsumerConfig
	logger *slog.Logger
}

func NewStrategyConsumerActor(cfg StrategyConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &StrategyConsumerActor{cfg: cfg}
	}
}

func (a *StrategyConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "strategy-consumer")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.logger.Info("strategy consumer actor started",
			"max_position_pct", a.maxPositionPct(),
		)

	case actor.Stopped:
		a.logStats()

	case strategyReceivedMessage:
		a.onStrategyEvent(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *StrategyConsumerActor) onStrategyEvent(c *actor.Context, msg strategyReceivedMessage) {
	tracker := a.cfg.Tracker
	if tracker != nil {
		tracker.Counter("received").Add(1)
	}

	event := msg.Event
	strat := event.Strategy

	// INV-6: Only process mean_reversion_entry.
	if strat.Type != "mean_reversion_entry" {
		if tracker != nil {
			tracker.Counter("skipped_wrong_type").Add(1)
		}
		metrics.IncStrategyEvaluation(strat.Type, "skipped_wrong_type")
		a.logger.Warn("strategy type mismatch — skipped",
			"expected", "mean_reversion_entry",
			"got", strat.Type,
			"correlation_id", event.Metadata.CorrelationID,
		)
		return
	}

	// S361: Confidence threshold gate — skip events below configured minimum.
	if a.belowConfidenceThreshold(strat.Confidence) {
		if tracker != nil {
			tracker.Counter("skipped_low_confidence").Add(1)
		}
		metrics.IncStrategyEvaluation(strat.Type, "skipped_low_confidence")
		a.logger.Info("strategy confidence below threshold — skipped",
			"confidence", strat.Confidence,
			"min_confidence", a.cfg.MinConfidence,
			"source", strat.Source,
			"symbol", strat.VenueSymbol(),
			"timeframe", strat.Timeframe,
			"correlation_id", event.Metadata.CorrelationID,
		)
		return
	}

	// INV-7: Flat direction produces intent with side=none, quantity=0
	// but still forwards for observability through the venue adapter.

	// Extract decision severity from first decision (INV-1 contract field mapping).
	decisionSeverity := ""
	if len(strat.Decisions) > 0 {
		decisionSeverity = strat.Decisions[0].Severity
	}

	// Evaluate via PaperOrderEvaluator with pass-through risk (INV-4).
	// Use the Instrument-aware constructor so the canonical instrument carried
	// by the Strategy flows through directly — the Source label may be synthetic
	// (e.g., "execute.venue-adapter" in slice tests) and is not always a venue
	// identifier recognized by instrumentFromBinding.
	evaluator := appexec.NewPaperOrderEvaluatorForInstrument(strat.Source, strat.Instrument, strat.Timeframe)
	intent, ok := evaluator.Evaluate(
		"pass_through",          // riskType — INV-4: explicit pass-through marker
		"approved",              // riskDisposition — INV-4: auto-approved
		strat.Confidence,        // riskConfidence — from strategy
		a.maxPositionPct(),      // maxPositionPct — configurable
		string(strat.Direction), // strategyDirection
		strat.Confidence,        // strategyConfidence
		strat.Type,              // strategyType — INV-1
		decisionSeverity,        // decisionSeverity
		strat.Timeframe,         // riskTimeframe
		strat.Timestamp,         // ts — INV-5: strategy timestamp, NOT time.Now()
	)
	if !ok {
		if tracker != nil {
			tracker.RecordError()
		}
		metrics.IncStrategyEvaluation(strat.Type, "error")
		a.logger.Error("strategy evaluation failed",
			"source", strat.Source,
			"symbol", strat.VenueSymbol(),
			"timeframe", strat.Timeframe,
			"direction", string(strat.Direction),
			"correlation_id", event.Metadata.CorrelationID,
		)
		return
	}

	// INV-3: Preserve correlation/causation chain.
	intent.CorrelationID = event.Metadata.CorrelationID
	intent.CausationID = event.Metadata.ID

	// S361: Enrich intent Parameters with source-driven explainability fields.
	intent.Parameters["source_path"] = "strategy_consumer.mean_reversion_entry"
	intent.Parameters["evaluation_outcome"] = a.evaluationOutcome(strat.Direction)
	if a.cfg.MinConfidence != "" {
		intent.Parameters["confidence_threshold"] = a.cfg.MinConfidence
	}

	// Wrap intent in a PaperOrderSubmittedEvent to reuse the existing venue adapter path.
	// This preserves all safety gates (kill switch, staleness) and the composed submit pipeline.
	syntheticEvent := domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(event.Metadata.CorrelationID).
			WithCausationID(event.Metadata.ID),
		ExecutionIntent: intent,
	}

	c.Send(a.cfg.AdapterPID, intentReceivedMessage{Event: syntheticEvent})

	// Record Prometheus metrics.
	outcomeLabel := a.evaluationOutcome(strat.Direction)
	metrics.IncStrategyEvaluation(strat.Type, outcomeLabel)
	metrics.IncExecutionIntent("strategy_consumer.mean_reversion_entry", string(intent.Side))

	if tracker != nil {
		tracker.RecordEvent()
		tracker.Counter("evaluated").Add(1)
		if strat.Direction == strategy.DirectionFlat {
			tracker.Counter("evaluated_flat").Add(1)
		} else {
			tracker.Counter("evaluated_actionable").Add(1)
		}
	}

	a.logger.Info("strategy event evaluated",
		"source", strat.Source,
		"symbol", strat.VenueSymbol(),
		"timeframe", strat.Timeframe,
		"direction", string(strat.Direction),
		"side", string(intent.Side),
		"quantity", intent.Quantity,
		"risk_type", intent.Risk.Type,
		"correlation_id", event.Metadata.CorrelationID,
		"causation_id", event.Metadata.ID,
	)
}

func (a *StrategyConsumerActor) maxPositionPct() string {
	if a.cfg.MaxPositionPct != "" {
		return a.cfg.MaxPositionPct
	}
	return DefaultMaxPositionPct
}

// belowConfidenceThreshold returns true if the strategy confidence is below the
// configured minimum. Returns false (passes) if no threshold is configured.
func (a *StrategyConsumerActor) belowConfidenceThreshold(confidence string) bool {
	if a.cfg.MinConfidence == "" || a.cfg.MinConfidence == "0" {
		return false
	}
	minConf, err := strconv.ParseFloat(a.cfg.MinConfidence, 64)
	if err != nil {
		return false // invalid threshold config — fail-open
	}
	conf, err := strconv.ParseFloat(confidence, 64)
	if err != nil {
		return false // unparseable confidence — fail-open
	}
	return conf < minConf
}

// evaluationOutcome returns the Prometheus outcome label for the given direction.
func (a *StrategyConsumerActor) evaluationOutcome(direction strategy.Direction) string {
	if direction == strategy.DirectionFlat {
		return "flat"
	}
	return "actionable"
}

func (a *StrategyConsumerActor) logStats() {
	tracker := a.cfg.Tracker
	if tracker == nil {
		return
	}
	a.logger.Info("strategy consumer stats",
		"received", tracker.Counter("received").Load(),
		"evaluated", tracker.Counter("evaluated").Load(),
		"evaluated_flat", tracker.Counter("evaluated_flat").Load(),
		"evaluated_actionable", tracker.Counter("evaluated_actionable").Load(),
		"skipped_wrong_type", tracker.Counter("skipped_wrong_type").Load(),
		"skipped_low_confidence", tracker.Counter("skipped_low_confidence").Load(),
		"errors", tracker.ErrorCount(),
	)
}
