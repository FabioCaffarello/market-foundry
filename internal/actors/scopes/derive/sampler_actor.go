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

// SamplerConfig holds the configuration for the candle sampler actor.
type SamplerConfig struct {
	Source       string
	Symbol       string
	Timeframe    time.Duration
	PublisherPID *actor.PID
	ScopePID     *actor.PID // SourceScopeActor PID for signal fan-out
}

// SamplerActor owns a CandleSampler and publishes finalized candles.
type SamplerActor struct {
	cfg     SamplerConfig
	logger  *slog.Logger
	sampler *appderive.CandleSampler
}

func NewSamplerActor(cfg SamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &SamplerActor{cfg: cfg}
	}
}

func (a *SamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "candle-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appderive.NewCandleSampler(a.cfg.Source, a.cfg.Symbol, a.cfg.Timeframe)
		a.logger.Info("candle sampler started")

	case actor.Stopped:
		a.logger.Info("candle sampler stopped")

	case tradeReceivedMessage:
		a.onTrade(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *SamplerActor) onTrade(c *actor.Context, msg tradeReceivedMessage) {
	candle, didFinalize := a.sampler.AddTrade(msg.Event.Trade)
	if !didFinalize {
		return
	}

	if prob := candle.Validate(); prob != nil {
		a.logger.Error("finalized candle validation failed", "error", prob.Message)
		return
	}

	event := evidence.CandleSampledEvent{
		Metadata: events.NewMetadata().WithCorrelationID(msg.Event.Metadata.CorrelationID),
		Candle:   candle,
	}

	c.Send(a.cfg.PublisherPID, publishCandleMessage{Event: event})

	// Fan-out to signal samplers via SourceScopeActor.
	if a.cfg.ScopePID != nil {
		c.Send(a.cfg.ScopePID, candleFinalizedMessage{
			Symbol:        candle.VenueSymbol(),
			Timeframe:     candle.Timeframe,
			ClosePrice:    candle.Close,
			Timestamp:     candle.CloseTime,
			CorrelationID: msg.Event.Metadata.CorrelationID,
		})
	}

	a.logger.Info("candle finalized",
		"open_time", candle.OpenTime.Format(time.RFC3339),
		"trades", candle.TradeCount,
		"open", candle.Open,
		"high", candle.High,
		"low", candle.Low,
		"close", candle.Close,
	)
}
