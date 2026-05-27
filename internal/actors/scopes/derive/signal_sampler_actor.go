package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appsignal "internal/application/signal"
	"internal/domain/instrument"
	domainsignal "internal/domain/signal"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

// SignalSamplerConfig holds the configuration for a signal sampler actor.
//
// H-6.c.1 commit 6: the canonical Instrument field is the
// pass-through identity computed at the binding boundary via
// BindingTarget.Instrument(). The legacy (Source, Symbol) string
// pair remains in the struct for back-compat with the legacy
// NewXxxSampler constructors during the migration window; commits
// 7a–7d remove both the legacy fields and the wrappers once tests
// migrate.
type SignalSamplerConfig struct {
	Source             string
	Symbol             string
	Instrument         instrument.CanonicalInstrument
	Timeframe          time.Duration
	SignalPublisherPID *actor.PID
	ScopePID           *actor.PID // For fan-out to decision evaluators
}

// RSISignalSamplerActor owns an RSISampler and publishes finalized signals.
// It receives candleFinalizedMessage from the candle sampler via local fan-out.
type RSISignalSamplerActor struct {
	cfg     SignalSamplerConfig
	logger  *slog.Logger
	sampler *appsignal.RSISampler
}

func NewRSISignalSamplerActor(cfg SignalSamplerConfig) actor.Producer {
	return func() actor.Receiver {
		return &RSISignalSamplerActor{cfg: cfg}
	}
}

func (a *RSISignalSamplerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With(
			"actor", "rsi-signal-sampler",
			"source", a.cfg.Source,
			"symbol", a.cfg.Symbol,
			"timeframe_s", int(a.cfg.Timeframe.Seconds()),
		)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.sampler = appsignal.NewRSISamplerForInstrument(a.cfg.Source, a.cfg.Instrument, int(a.cfg.Timeframe.Seconds()))
		a.logger.Info("rsi signal sampler started")

	case actor.Stopped:
		a.logger.Info("rsi signal sampler stopped")

	case candleFinalizedMessage:
		a.onCandleFinalized(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *RSISignalSamplerActor) onCandleFinalized(c *actor.Context, msg candleFinalizedMessage) {
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
			Symbol:         sig.VenueSymbol(),
			SignalType:     sig.Type,
			SignalValue:    sig.Value,
			SignalMetadata: sig.Metadata,
			Timeframe:      sig.Timeframe,
			Timestamp:      sig.Timestamp,
			CorrelationID:  msg.CorrelationID,
			CausationID:    meta.ID,
		})
	}

	a.logger.Info("rsi signal generated",
		"value", sig.Value,
		"timestamp", sig.Timestamp.Format(time.RFC3339),
		"correlation_id", msg.CorrelationID,
	)
}
