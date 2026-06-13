package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsinsights "internal/adapters/nats/natsinsights"
	appderive "internal/application/derive"
	"internal/domain/insights"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// CrossVenueFusionConfig configures the cross-venue fusion actor
// (PROGRAM-0005 / H-8.c).
type CrossVenueFusionConfig struct {
	NATSURL    string
	Registry   natsinsights.Registry
	Timeframes []time.Duration
	Tracker    *healthz.Tracker
}

// CrossVenueFusionActor is the SINGLE supervisor-level actor that fuses
// trades across venues per canonical instrument (Decisão C1 — NOT a
// per-source FamilyProcessor; the supervisor fans every trade here). It
// owns one CrossVenueFusion per timeframe and its own insights
// publisher (single-writer at the service level, ADR-0008).
type CrossVenueFusionActor struct {
	cfg          CrossVenueFusionConfig
	logger       *slog.Logger
	publisherPID *actor.PID
	fusions      []*appderive.CrossVenueFusion
}

func NewCrossVenueFusionActor(cfg CrossVenueFusionConfig) actor.Producer {
	return func() actor.Receiver {
		return &CrossVenueFusionActor{cfg: cfg}
	}
}

func (a *CrossVenueFusionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "cross-venue-fusion")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.publisherPID = c.SpawnChild(NewInsightsPublisherActor(InsightsPublisherConfig{
			URL:      a.cfg.NATSURL,
			Source:   "derive.insights-publisher.cross-venue",
			Registry: a.cfg.Registry,
			Tracker:  a.cfg.Tracker,
		}), "insights-publisher")
		a.fusions = make([]*appderive.CrossVenueFusion, len(a.cfg.Timeframes))
		for i, tf := range a.cfg.Timeframes {
			a.fusions[i] = appderive.NewCrossVenueFusion(tf)
		}
		a.logger.Info("cross-venue fusion started", "timeframes", len(a.cfg.Timeframes))

	case actor.Stopped:
		a.logger.Info("cross-venue fusion stopped")

	case tradeReceivedMessage:
		a.onTrade(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *CrossVenueFusionActor) onTrade(c *actor.Context, msg tradeReceivedMessage) {
	for _, fusion := range a.fusions {
		snap, didFinalize := fusion.AddTrade(msg.Event.Trade)
		if !didFinalize {
			continue
		}
		if prob := snap.Validate(); prob != nil {
			a.logger.Error("finalized cross-venue snapshot validation failed", "error", prob.Message)
			continue
		}
		event := insights.CrossVenueSampledEvent{
			Metadata:           events.NewMetadata().WithCorrelationID(msg.Event.Metadata.CorrelationID),
			CrossVenueSnapshot: snap,
		}
		c.Send(a.publisherPID, publishCrossVenueMessage{Event: event})

		a.logger.Info("cross-venue snapshot finalized",
			"symbol", snap.VenueSymbol(),
			"timeframe", snap.Timeframe,
			"venues", len(snap.Venues),
			"trades", snap.TradeCount,
			"open_time", snap.OpenTime.Format(time.RFC3339),
		)
	}
}
