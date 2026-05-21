package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appstrategy "internal/application/strategy"
	domainstrategy "internal/domain/strategy"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// SqueezeBreakoutEntryResolverActor owns a SqueezeBreakoutEntryResolver and publishes finalized strategies.
// It receives decisionEvaluatedMessage from the decision evaluator via local fan-out.
type SqueezeBreakoutEntryResolverActor struct {
	cfg      StrategyResolverConfig
	logger   *slog.Logger
	resolver *appstrategy.SqueezeBreakoutEntryResolver
}

func NewSqueezeBreakoutEntryResolverActor(cfg StrategyResolverConfig) actor.Producer {
	return func() actor.Receiver {
		return &SqueezeBreakoutEntryResolverActor{cfg: cfg}
	}
}

func (a *SqueezeBreakoutEntryResolverActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "squeeze-breakout-entry-resolver",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.resolver = appstrategy.NewSqueezeBreakoutEntryResolver(a.cfg.Source, a.cfg.Symbol, int(a.cfg.Timeframe.Seconds()))
		a.logger.Info("squeeze breakout entry resolver started")

	case actor.Stopped:
		a.logger.Info("squeeze breakout entry resolver stopped")

	case decisionEvaluatedMessage:
		a.onDecisionEvaluated(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *SqueezeBreakoutEntryResolverActor) onDecisionEvaluated(c *actor.Context, msg decisionEvaluatedMessage) {
	strat, ok := a.resolver.Resolve(msg.DecisionType, msg.DecisionOutcome, msg.DecisionConfidence, msg.DecisionSeverity, msg.DecisionRationale, msg.Timeframe, msg.Timestamp)
	if !ok {
		return
	}

	// S470: enrich decision inputs with causal event reference.
	for i := range strat.Decisions {
		strat.Decisions[i].EventID = msg.CausationID
	}

	if prob := strat.Validate(); prob != nil {
		a.logger.Error("strategy validation failed", "error", prob.Message)
		return
	}

	meta := events.NewMetadata().
		WithCorrelationID(msg.CorrelationID).
		WithCausationID(msg.CausationID)
	event := domainstrategy.StrategyResolvedEvent{
		Metadata: meta,
		Strategy: strat,
	}

	c.Send(a.cfg.StrategyPublisherPID, publishStrategyMessage{Event: event})

	// Fan out to risk evaluators via scope PID.
	// Forward decision severity and rationale from the first DecisionInput for risk context.
	if a.cfg.ScopePID != nil {
		var decSeverity, decRationale string
		if len(strat.Decisions) > 0 {
			decSeverity = strat.Decisions[0].Severity
			decRationale = strat.Decisions[0].Rationale
		}
		c.Send(a.cfg.ScopePID, strategyResolvedMessage{
			Symbol:             strat.Symbol,
			StrategyType:       strat.Type,
			StrategyDirection:  string(strat.Direction),
			StrategyConfidence: strat.Confidence,
			DecisionSeverity:   decSeverity,
			DecisionRationale:  decRationale,
			Timeframe:          strat.Timeframe,
			Timestamp:          strat.Timestamp,
			CorrelationID:      meta.CorrelationID,
			CausationID:        meta.ID,
		})
	}

	a.logger.Info("squeeze breakout entry strategy resolved",
		"direction", string(strat.Direction),
		"confidence", strat.Confidence,
		"timestamp", strat.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
		"causation_id", msg.CausationID,
	)
}
