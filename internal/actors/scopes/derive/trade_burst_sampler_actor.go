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

// TradeBurstSamplerConfig holds the configuration for the trade burst sampler actor.
type TradeBurstSamplerConfig struct {
	Source       string
	Symbol       string
	Timeframe    time.Duration
	PublisherPID *actor.PID
}

// TradeBurstSamplerActor owns a TradeBurstSampler and publishes finalized trade bursts.
type TradeBurstSamplerActor struct {
	cfg     TradeBurstSamplerConfig
	logger  *slog.Logger
	sampler *appderive.TradeBurstSampler
}

func NewTradeBurstSamplerActor(cfg TradeBurstSamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &TradeBurstSamplerActor{cfg: cfg}
	}
}

func (a *TradeBurstSamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "trade-burst-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appderive.NewTradeBurstSampler(a.cfg.Source, a.cfg.Symbol, a.cfg.Timeframe)
		a.logger.Info("trade burst sampler started")

	case actor.Stopped:
		a.logger.Info("trade burst sampler stopped")

	case tradeReceivedMessage:
		a.onTrade(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *TradeBurstSamplerActor) onTrade(c *actor.Context, msg tradeReceivedMessage) {
	burst, didFinalize := a.sampler.AddTrade(msg.Event.Trade)
	if !didFinalize {
		return
	}

	if prob := burst.Validate(); prob != nil {
		a.logger.Error("finalized trade burst validation failed", "error", prob.Message)
		return
	}

	event := evidence.TradeBurstSampledEvent{
		Metadata:   events.NewMetadata().WithCorrelationID(msg.Event.Metadata.CorrelationID),
		TradeBurst: burst,
	}

	c.Send(a.cfg.PublisherPID, publishTradeBurstMessage{Event: event})

	a.logger.Info("trade burst finalized",
		"open_time", burst.OpenTime.Format(time.RFC3339),
		"trades", burst.TradeCount,
		"buy_volume", burst.BuyVolume,
		"sell_volume", burst.SellVolume,
		"burst", burst.Burst,
	)
}
