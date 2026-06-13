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

// TPOSamplerConfig holds the configuration for the TPO sampler actor
// (PROGRAM-0005 / H-8.b).
type TPOSamplerConfig struct {
	Source        string
	Symbol        string
	Timeframe     time.Duration
	BucketSize    string
	PeriodSeconds int
	MaxLevels     int
	PublisherPID  *actor.PID
}

// TPOSamplerActor owns a TPOSampler and publishes finalized TPO profiles
// to the insights publisher.
type TPOSamplerActor struct {
	cfg     TPOSamplerConfig
	logger  *slog.Logger
	sampler *appderive.TPOSampler
}

func NewTPOSamplerActor(cfg TPOSamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &TPOSamplerActor{cfg: cfg}
	}
}

func (a *TPOSamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "tpo-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appderive.NewTPOSampler(a.cfg.Source, a.cfg.Timeframe, a.cfg.BucketSize, a.cfg.PeriodSeconds, a.cfg.MaxLevels)
		a.logger.Info("tpo sampler started", "bucket_size", a.cfg.BucketSize, "period_s", a.cfg.PeriodSeconds)

	case actor.Stopped:
		a.logger.Info("tpo sampler stopped")

	case tradeReceivedMessage:
		a.onTrade(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *TPOSamplerActor) onTrade(c *actor.Context, msg tradeReceivedMessage) {
	tp, didFinalize := a.sampler.AddTrade(msg.Event.Trade)
	if !didFinalize {
		return
	}
	if prob := tp.Validate(); prob != nil {
		a.logger.Error("finalized tpo profile validation failed", "error", prob.Message)
		return
	}

	event := insights.TPOProfileSampledEvent{
		Metadata:   events.NewMetadata().WithCorrelationID(msg.Event.Metadata.CorrelationID),
		TPOProfile: tp,
	}
	c.Send(a.cfg.PublisherPID, publishTPOProfileMessage{Event: event})

	a.logger.Info("tpo profile finalized",
		"open_time", tp.OpenTime.Format(time.RFC3339),
		"periods", len(tp.Periods),
		"levels", len(tp.Levels),
		"trades", tp.TradeCount,
		"overload", int(tp.Overload),
	)
}
