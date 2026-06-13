package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsinsights "internal/adapters/nats/natsinsights"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// InsightsPublisherConfig holds the configuration for the insights
// publisher actor (PROGRAM-0005 / H-8.a). Single-writer to
// INSIGHTS_EVENTS (ADR-0008).
type InsightsPublisherConfig struct {
	URL      string
	Source   string
	Registry natsinsights.Registry
	Tracker  *healthz.Tracker
}

// InsightsPublisherActor owns the NATS JetStream connection for
// publishing insights events.
type InsightsPublisherActor struct {
	cfg       InsightsPublisherConfig
	logger    *slog.Logger
	publisher *natsinsights.Publisher
}

func NewInsightsPublisherActor(cfg InsightsPublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &InsightsPublisherActor{cfg: cfg}
	}
}

func (a *InsightsPublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "insights-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := natsinsights.NewPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start insights publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("insights publisher started",
			"stream", a.cfg.Registry.VolumeProfileSampled.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close insights publisher", "error", err)
			}
		}

	case publishVolumeProfileMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishVolumeProfile(ctx, msg.Event)
		cancel()
		vp := msg.Event.VolumeProfile
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish volume profile failed",
				"error", prob.Message,
				"code", prob.Code,
				"source", vp.Source,
				"symbol", vp.VenueSymbol(),
				"timeframe", vp.Timeframe,
				"open_time", vp.OpenTime.Format(time.RFC3339),
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + vp.VenueSymbol()).Add(1)
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
