package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appderive "internal/application/derive"
	"internal/domain/evidence"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// VolumeSamplerConfig holds the configuration for the volume sampler actor.
type VolumeSamplerConfig struct {
	Source       string
	Symbol       string
	Timeframe    time.Duration
	PublisherPID *actor.PID
}

// VolumeSamplerActor owns a VolumeSampler and publishes finalized volume profiles.
type VolumeSamplerActor struct {
	cfg     VolumeSamplerConfig
	logger  *slog.Logger
	sampler *appderive.VolumeSampler
}

func NewVolumeSamplerActor(cfg VolumeSamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &VolumeSamplerActor{cfg: cfg}
	}
}

func (a *VolumeSamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "volume-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appderive.NewVolumeSampler(a.cfg.Source, a.cfg.Symbol, a.cfg.Timeframe)
		a.logger.Info("volume sampler started")

	case actor.Stopped:
		a.logger.Info("volume sampler stopped")

	case tradeReceivedMessage:
		a.onTrade(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *VolumeSamplerActor) onTrade(c *actor.Context, msg tradeReceivedMessage) {
	vol, didFinalize := a.sampler.AddTrade(msg.Event.Trade)
	if !didFinalize {
		return
	}

	if prob := vol.Validate(); prob != nil {
		a.logger.Error("finalized volume validation failed", "error", prob.Message)
		return
	}

	event := evidence.VolumeSampledEvent{
		Metadata: events.NewMetadata().WithCorrelationID(msg.Event.Metadata.CorrelationID),
		Volume:   vol,
	}

	c.Send(a.cfg.PublisherPID, publishVolumeMessage{Event: event})

	a.logger.Info("volume finalized",
		"open_time", vol.OpenTime.Format(time.RFC3339),
		"trades", vol.TradeCount,
		"total_volume", vol.TotalVolume,
		"vwap", vol.VWAP,
	)
}
