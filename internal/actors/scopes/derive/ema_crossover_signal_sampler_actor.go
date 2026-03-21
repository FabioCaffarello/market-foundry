package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appsignal "internal/application/signal"
	domainsignal "internal/domain/signal"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// EMACrossoverSignalSamplerActor owns an EMACrossoverSampler and publishes finalized signals.
// It receives candleFinalizedMessage from the candle sampler via local fan-out.
type EMACrossoverSignalSamplerActor struct {
	cfg     SignalSamplerConfig
	logger  *slog.Logger
	sampler *appsignal.EMACrossoverSampler
}

func NewEMACrossoverSignalSamplerActor(cfg SignalSamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &EMACrossoverSignalSamplerActor{cfg: cfg}
	}
}

func (a *EMACrossoverSignalSamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "ema-crossover-signal-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appsignal.NewEMACrossoverSampler(a.cfg.Source, a.cfg.Symbol, int(a.cfg.Timeframe.Seconds()))
		a.logger.Info("ema crossover signal sampler started")

	case actor.Stopped:
		a.logger.Info("ema crossover signal sampler stopped")

	case candleFinalizedMessage:
		a.onCandleFinalized(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *EMACrossoverSignalSamplerActor) onCandleFinalized(c *actor.Context, msg candleFinalizedMessage) {
	sig, ok := a.sampler.AddClose(msg.ClosePrice, msg.Timestamp)
	if !ok {
		return
	}

	if prob := sig.Validate(); prob != nil {
		a.logger.Error("signal validation failed", "error", prob.Message)
		return
	}

	meta := events.NewMetadata().WithCorrelationID(msg.CorrelationID)
	event := domainsignal.SignalGeneratedEvent{
		Metadata: meta,
		Signal:   sig,
	}

	c.Send(a.cfg.SignalPublisherPID, publishSignalMessage{Event: event})

	// Notify scope for decision fan-out (same pattern as candle→signal).
	if a.cfg.ScopePID != nil {
		c.Send(a.cfg.ScopePID, signalGeneratedMessage{
			Symbol:         sig.Symbol,
			SignalType:     sig.Type,
			SignalValue:    sig.Value,
			SignalMetadata: sig.Metadata,
			Timeframe:      sig.Timeframe,
			Timestamp:      sig.Timestamp,
			CorrelationID:  msg.CorrelationID,
			CausationID:    meta.ID,
		})
	}

	a.logger.Info("ema crossover signal generated",
		"value", sig.Value,
		"timestamp", sig.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
	)
}
