package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// RiskPublisherConfig holds the configuration for the risk publisher actor.
type RiskPublisherConfig struct {
	URL      string
	Source   string
	Registry adapternats.RiskRegistry
	Tracker  *healthz.Tracker
}

// RiskPublisherActor owns the NATS JetStream connection for publishing risk events.
type RiskPublisherActor struct {
	cfg       RiskPublisherConfig
	logger    *slog.Logger
	publisher *adapternats.RiskPublisher
}

func NewRiskPublisherActor(cfg RiskPublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &RiskPublisherActor{cfg: cfg}
	}
}

func (a *RiskPublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "risk-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := adapternats.NewRiskPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start risk publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("risk publisher started",
			"stream", a.cfg.Registry.PositionExposureAssessed.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close risk publisher", "error", err)
			}
		}

	case publishRiskMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishRisk(ctx, msg.Event)
		cancel()
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish risk failed",
				"error", prob.Message,
				"code", prob.Code,
				"type", msg.Event.RiskAssessment.Type,
				"source", msg.Event.RiskAssessment.Source,
				"symbol", msg.Event.RiskAssessment.Symbol,
				"timeframe", msg.Event.RiskAssessment.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + msg.Event.RiskAssessment.Symbol).Add(1)
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
