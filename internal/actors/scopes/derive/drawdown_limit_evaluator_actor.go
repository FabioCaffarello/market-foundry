package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	apprisk "internal/application/risk"
	domainrisk "internal/domain/risk"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// DrawdownLimitEvaluatorActor owns a DrawdownLimitEvaluator and publishes risk assessments.
// It receives strategyResolvedMessage from the strategy resolver via local fan-out.
type DrawdownLimitEvaluatorActor struct {
	cfg       RiskEvaluatorConfig
	logger    *slog.Logger
	evaluator *apprisk.DrawdownLimitEvaluator
}

func NewDrawdownLimitEvaluatorActor(cfg RiskEvaluatorConfig) actor.Producer {
	return func() actor.Receiver {
		return &DrawdownLimitEvaluatorActor{cfg: cfg}
	}
}

func (a *DrawdownLimitEvaluatorActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "drawdown-limit-evaluator",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.evaluator = apprisk.NewDrawdownLimitEvaluator(a.cfg.Source, a.cfg.Symbol, int(a.cfg.Timeframe.Seconds()))
		a.logger.Info("drawdown limit evaluator started")

	case actor.Stopped:
		a.logger.Info("drawdown limit evaluator stopped")

	case strategyResolvedMessage:
		a.onStrategyResolved(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *DrawdownLimitEvaluatorActor) onStrategyResolved(c *actor.Context, msg strategyResolvedMessage) {
	assessment, ok := a.evaluator.Evaluate(msg.StrategyType, msg.StrategyDirection, msg.StrategyConfidence, msg.DecisionSeverity, msg.DecisionRationale, msg.Timeframe, msg.Timestamp)
	if !ok {
		return
	}

	if prob := assessment.Validate(); prob != nil {
		a.logger.Error("risk assessment validation failed", "error", prob.Message)
		return
	}

	event := domainrisk.RiskAssessedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(msg.CorrelationID).
			WithCausationID(msg.CausationID),
		RiskAssessment: assessment,
	}

	c.Send(a.cfg.RiskPublisherPID, publishRiskMessage{Event: event})

	// Fan out to execution evaluators via scope.
	if a.cfg.ScopePID != nil {
		stratDirection := ""
		stratConfidence := ""
		decSeverity := ""
		if len(assessment.Strategies) > 0 {
			stratDirection = assessment.Strategies[0].Direction
			stratConfidence = assessment.Strategies[0].Confidence
			decSeverity = assessment.Strategies[0].DecisionSeverity
		}
		c.Send(a.cfg.ScopePID, riskAssessedMessage{
			Symbol:             a.cfg.Symbol,
			RiskType:           assessment.Type,
			RiskDisposition:    string(assessment.Disposition),
			RiskConfidence:     assessment.Confidence,
			MaxPositionPct:     assessment.Constraints.StopDistance,
			StrategyDirection:  stratDirection,
			StrategyConfidence: stratConfidence,
			DecisionSeverity:   decSeverity,
			Timeframe:          assessment.Timeframe,
			Timestamp:          assessment.Timestamp,
			CorrelationID:      msg.CorrelationID,
			CausationID:        event.Metadata.ID,
		})
	}

	a.logger.Info("drawdown limit risk assessed",
		"disposition", string(assessment.Disposition),
		"confidence", assessment.Confidence,
		"timestamp", assessment.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
		"causation_id", msg.CausationID,
	)
}
