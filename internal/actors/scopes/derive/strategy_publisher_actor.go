package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// StrategyPublisherConfig holds the configuration for the strategy publisher actor.
type StrategyPublisherConfig struct {
	URL      string
	Source   string
	Registry natsstrategy.Registry
	Tracker  *healthz.Tracker
}

// StrategyPublisherActor owns the NATS JetStream connection for publishing strategy events.
type StrategyPublisherActor struct {
	cfg       StrategyPublisherConfig
	logger    *slog.Logger
	publisher *natsstrategy.Publisher
}

func NewStrategyPublisherActor(cfg StrategyPublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &StrategyPublisherActor{cfg: cfg}
	}
}

func (a *StrategyPublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "strategy-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := natsstrategy.NewPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start strategy publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("strategy publisher started",
			"stream", a.cfg.Registry.MeanReversionEntryResolved.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close strategy publisher", "error", err)
			}
		}

	case publishStrategyMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishStrategy(ctx, msg.Event)
		cancel()
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish strategy failed",
				"error", prob.Message,
				"code", prob.Code,
				"type", msg.Event.Strategy.Type,
				"source", msg.Event.Strategy.Source,
				"symbol", msg.Event.Strategy.VenueSymbol(),
				"timeframe", msg.Event.Strategy.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + msg.Event.Strategy.VenueSymbol()).Add(1)
			a.cfg.Tracker.Counter("strategy:" + msg.Event.Strategy.Type + ":" + string(msg.Event.Strategy.Direction)).Add(1)
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
