package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsdecision "internal/adapters/nats/natsdecision"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// DecisionPublisherConfig holds the configuration for the decision publisher actor.
type DecisionPublisherConfig struct {
	URL      string
	Source   string
	Registry natsdecision.Registry
	Tracker  *healthz.Tracker
}

// DecisionPublisherActor owns the NATS JetStream connection for publishing decision events.
type DecisionPublisherActor struct {
	cfg       DecisionPublisherConfig
	logger    *slog.Logger
	publisher *natsdecision.Publisher
}

func NewDecisionPublisherActor(cfg DecisionPublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &DecisionPublisherActor{cfg: cfg}
	}
}

func (a *DecisionPublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "decision-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := natsdecision.NewPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start decision publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("decision publisher started",
			"stream", a.cfg.Registry.RSIOversoldEvaluated.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close decision publisher", "error", err)
			}
		}

	case publishDecisionMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishDecision(ctx, msg.Event)
		cancel()
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish decision failed",
				"error", prob.Message,
				"code", prob.Code,
				"type", msg.Event.Decision.Type,
				"source", msg.Event.Decision.Source,
				"symbol", msg.Event.Decision.Symbol,
				"timeframe", msg.Event.Decision.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + msg.Event.Decision.Symbol).Add(1)
			a.cfg.Tracker.Counter("decision:" + msg.Event.Decision.Type + ":" + string(msg.Event.Decision.Outcome)).Add(1)
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
