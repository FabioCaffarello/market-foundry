package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appdecision "internal/application/decision"
	domaindecision "internal/domain/decision"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// BollingerSqueezeEvaluatorActor owns a BollingerSqueezeEvaluator and publishes finalized decisions.
// It receives signalGeneratedMessage from the signal sampler via local fan-out.
type BollingerSqueezeEvaluatorActor struct {
	cfg       DecisionEvaluatorConfig
	logger    *slog.Logger
	evaluator *appdecision.BollingerSqueezeEvaluator
}

func NewBollingerSqueezeEvaluatorActor(cfg DecisionEvaluatorConfig) actor.Producer {
	return func() actor.Receiver {
		return &BollingerSqueezeEvaluatorActor{cfg: cfg}
	}
}

func (a *BollingerSqueezeEvaluatorActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "bollinger-squeeze-evaluator",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.evaluator = appdecision.NewBollingerSqueezeEvaluator(a.cfg.Source, a.cfg.Symbol, int(a.cfg.Timeframe.Seconds()))
		a.logger.Info("bollinger squeeze evaluator started")

	case actor.Stopped:
		a.logger.Info("bollinger squeeze evaluator stopped")

	case signalGeneratedMessage:
		a.onSignalGenerated(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *BollingerSqueezeEvaluatorActor) onSignalGenerated(c *actor.Context, msg signalGeneratedMessage) {
	dec, ok := a.evaluator.Evaluate(msg.SignalType, msg.SignalValue, msg.Timeframe, msg.Timestamp, msg.SignalMetadata)
	if !ok {
		return
	}

	// S470: enrich signal inputs with causal event reference.
	for i := range dec.Signals {
		dec.Signals[i].EventID = msg.CausationID
	}

	if prob := dec.Validate(); prob != nil {
		a.logger.Error("decision validation failed", "error", prob.Message)
		return
	}

	meta := events.NewMetadata().
		WithCorrelationID(msg.CorrelationID).
		WithCausationID(msg.CausationID)
	event := domaindecision.DecisionEvaluatedEvent{
		Metadata: meta,
		Decision: dec,
	}

	c.Send(a.cfg.DecisionPublisherPID, publishDecisionMessage{Event: event})

	// Fan out to strategy resolvers via the scope actor.
	if a.cfg.ScopePID != nil {
		c.Send(a.cfg.ScopePID, decisionEvaluatedMessage{
			Symbol:             dec.Symbol,
			DecisionType:       dec.Type,
			DecisionOutcome:    string(dec.Outcome),
			DecisionConfidence: dec.Confidence,
			DecisionSeverity:   string(dec.Severity),
			DecisionRationale:  dec.Rationale,
			Timeframe:          dec.Timeframe,
			Timestamp:          dec.Timestamp,
			CorrelationID:      msg.CorrelationID,
			CausationID:        meta.ID,
		})
	}

	a.logger.Info("bollinger squeeze decision evaluated",
		"outcome", string(dec.Outcome),
		"confidence", dec.Confidence,
		"timestamp", dec.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
		"causation_id", msg.CausationID,
	)
}
