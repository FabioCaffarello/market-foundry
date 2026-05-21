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

// RiskEvaluatorConfig holds the configuration for a risk evaluator actor.
type RiskEvaluatorConfig struct {
	Source           string
	Symbol           string
	Timeframe        time.Duration
	RiskPublisherPID *actor.PID
	ScopePID         *actor.PID // for downstream fan-out to execution evaluators
}

// PositionExposureEvaluatorActor owns a PositionExposureEvaluator and publishes risk assessments.
// It receives strategyResolvedMessage from the strategy resolver via local fan-out.
type PositionExposureEvaluatorActor struct {
	cfg       RiskEvaluatorConfig
	logger    *slog.Logger
	evaluator *apprisk.PositionExposureEvaluator
}

func NewPositionExposureEvaluatorActor(cfg RiskEvaluatorConfig) actor.Producer {
	return func() actor.Receiver {
		return &PositionExposureEvaluatorActor{cfg: cfg}
	}
}

func (a *PositionExposureEvaluatorActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "position-exposure-evaluator",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.evaluator = apprisk.NewPositionExposureEvaluator(a.cfg.Source, a.cfg.Symbol, int(a.cfg.Timeframe.Seconds()))
		a.logger.Info("position exposure evaluator started")

	case actor.Stopped:
		a.logger.Info("position exposure evaluator stopped")

	case strategyResolvedMessage:
		a.onStrategyResolved(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *PositionExposureEvaluatorActor) onStrategyResolved(c *actor.Context, msg strategyResolvedMessage) {
	assessment, ok := a.evaluator.Evaluate(msg.StrategyType, msg.StrategyDirection, msg.StrategyConfidence, msg.DecisionSeverity, msg.DecisionRationale, msg.Timeframe, msg.Timestamp)
	if !ok {
		return
	}

	// S470: enrich strategy inputs with causal event reference.
	for i := range assessment.Strategies {
		assessment.Strategies[i].EventID = msg.CausationID
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
		stratType := ""
		decSeverity := ""
		if len(assessment.Strategies) > 0 {
			stratDirection = assessment.Strategies[0].Direction
			stratConfidence = assessment.Strategies[0].Confidence
			stratType = assessment.Strategies[0].Type
			decSeverity = assessment.Strategies[0].DecisionSeverity
		}
		c.Send(a.cfg.ScopePID, riskAssessedMessage{
			Symbol:             a.cfg.Symbol,
			RiskType:           assessment.Type,
			RiskDisposition:    string(assessment.Disposition),
			RiskConfidence:     assessment.Confidence,
			MaxPositionPct:     assessment.Constraints.MaxPositionSize,
			StrategyDirection:  stratDirection,
			StrategyConfidence: stratConfidence,
			StrategyType:       stratType,
			DecisionSeverity:   decSeverity,
			Timeframe:          assessment.Timeframe,
			Timestamp:          assessment.Timestamp,
			CorrelationID:      msg.CorrelationID,
			CausationID:        event.Metadata.ID,
		})
	}

	a.logger.Info("position exposure risk assessed",
		"disposition", string(assessment.Disposition),
		"confidence", assessment.Confidence,
		"timestamp", assessment.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
		"causation_id", msg.CausationID,
	)
}
