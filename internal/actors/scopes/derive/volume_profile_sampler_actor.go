package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appderive "internal/application/derive"
	"internal/domain/insights"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// VolumeProfileSamplerConfig holds the configuration for the volume
// profile sampler actor (PROGRAM-0005 / H-8.a).
type VolumeProfileSamplerConfig struct {
	Source       string
	Symbol       string
	Timeframe    time.Duration
	BucketSize   string
	MaxBuckets   int
	PublisherPID *actor.PID
}

// VolumeProfileSamplerActor owns a VolumeProfileSampler and publishes
// finalized volume profiles to the insights publisher.
type VolumeProfileSamplerActor struct {
	cfg     VolumeProfileSamplerConfig
	logger  *slog.Logger
	sampler *appderive.VolumeProfileSampler
}

func NewVolumeProfileSamplerActor(cfg VolumeProfileSamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &VolumeProfileSamplerActor{cfg: cfg}
	}
}

func (a *VolumeProfileSamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "volume-profile-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appderive.NewVolumeProfileSampler(a.cfg.Source, a.cfg.Timeframe, a.cfg.BucketSize, a.cfg.MaxBuckets)
		a.logger.Info("volume profile sampler started", "bucket_size", a.cfg.BucketSize)

	case actor.Stopped:
		a.logger.Info("volume profile sampler stopped")

	case tradeReceivedMessage:
		a.onTrade(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *VolumeProfileSamplerActor) onTrade(c *actor.Context, msg tradeReceivedMessage) {
	vp, didFinalize := a.sampler.AddTrade(msg.Event.Trade)
	if !didFinalize {
		return
	}
	if prob := vp.Validate(); prob != nil {
		a.logger.Error("finalized volume profile validation failed", "error", prob.Message)
		return
	}

	event := insights.VolumeProfileSampledEvent{
		Metadata:      events.NewMetadata().WithCorrelationID(msg.Event.Metadata.CorrelationID),
		VolumeProfile: vp,
	}
	c.Send(a.cfg.PublisherPID, publishVolumeProfileMessage{Event: event})

	a.logger.Info("volume profile finalized",
		"open_time", vp.OpenTime.Format(time.RFC3339),
		"buckets", len(vp.Buckets),
		"trades", vp.TradeCount,
		"overload", int(vp.Overload),
	)
}
