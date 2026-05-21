package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// ExecutionEvaluatorConfig holds the configuration for an execution evaluator actor.
type ExecutionEvaluatorConfig struct {
	Source               string
	Symbol               string
	Timeframe            time.Duration
	ExecutionPublisherPID *actor.PID
}

// PaperOrderEvaluatorActor owns a PaperOrderEvaluator and publishes execution intents.
// It receives riskAssessedMessage from the risk evaluator via local fan-out.
type PaperOrderEvaluatorActor struct {
	cfg       ExecutionEvaluatorConfig
	logger    *slog.Logger
	evaluator *appexec.PaperOrderEvaluator
	simulator *appexec.PaperFillSimulator
}

func NewPaperOrderEvaluatorActor(cfg ExecutionEvaluatorConfig) actor.Producer {
	return func() actor.Receiver {
		return &PaperOrderEvaluatorActor{cfg: cfg}
	}
}

func (a *PaperOrderEvaluatorActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "paper-order-evaluator",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.evaluator = appexec.NewPaperOrderEvaluator(a.cfg.Source, a.cfg.Symbol, int(a.cfg.Timeframe.Seconds()))
		a.simulator = &appexec.PaperFillSimulator{}
		a.logger.Info("paper order evaluator started")

	case actor.Stopped:
		a.logger.Info("paper order evaluator stopped")

	case riskAssessedMessage:
		a.onRiskAssessed(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *PaperOrderEvaluatorActor) onRiskAssessed(c *actor.Context, msg riskAssessedMessage) {
	intent, ok := a.evaluator.Evaluate(
		msg.RiskType, msg.RiskDisposition, msg.RiskConfidence, msg.MaxPositionPct,
		msg.StrategyDirection, msg.StrategyConfidence,
		msg.StrategyType, msg.DecisionSeverity,
		msg.Timeframe, msg.Timestamp,
	)
	if !ok {
		return
	}

	// Persist causal trace in the intent so it survives KV materialization.
	intent.CorrelationID = msg.CorrelationID
	intent.CausationID = msg.CausationID

	// S470: enrich risk input with causal event reference.
	intent.Risk.EventID = msg.CausationID

	// Apply paper fill simulation: submitted → filled for actionable orders.
	intent, ok = a.simulator.SimulateFill(intent)
	if !ok {
		a.logger.Error("paper fill simulation failed",
			"status", string(intent.Status),
			"side", string(intent.Side),
		)
		return
	}

	if prob := intent.Validate(); prob != nil {
		a.logger.Error("execution intent validation failed", "error", prob.Message)
		return
	}

	event := domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(msg.CorrelationID).
			WithCausationID(msg.CausationID),
		ExecutionIntent: intent,
	}

	c.Send(a.cfg.ExecutionPublisherPID, publishExecutionMessage{Event: event})

	a.logger.Info("paper order execution intent produced",
		"side", string(intent.Side),
		"quantity", intent.Quantity,
		"status", string(intent.Status),
		"filled_quantity", intent.FilledQuantity,
		"fills_count", len(intent.Fills),
		"risk_disposition", msg.RiskDisposition,
		"timestamp", intent.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
		"causation_id", msg.CausationID,
	)
}
