package ingest

import (
	"context"
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	"internal/adapters/exchanges/binancef"

	"github.com/anthdm/hollywood/actor"
)

// WebSocketAdapterConfig holds the configuration for a single WebSocket adapter.
type WebSocketAdapterConfig struct {
	Symbol       string
	PublisherPID *actor.PID
}

// WebSocketAdapterActor connects to a Binance Futures aggTrade WebSocket stream,
// normalizes incoming trades, and forwards them to the publisher actor.
type WebSocketAdapterActor struct {
	cfg    WebSocketAdapterConfig
	logger *slog.Logger
	cancel context.CancelFunc
}

func NewWebSocketAdapterActor(cfg WebSocketAdapterConfig) actor.Producer {
	return func() actor.Receiver {
		return &WebSocketAdapterActor{cfg: cfg}
	}
}

func (a *WebSocketAdapterActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "ws-adapter", "symbol", a.cfg.Symbol)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.cancel != nil {
			a.cancel()
		}
		a.logger.Info("websocket adapter stopped")

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *WebSocketAdapterActor) start(c *actor.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	symbol := a.cfg.Symbol
	publisherPID := a.cfg.PublisherPID

	client := binancef.NewWSClient(symbol, func(data []byte) {
		agg, prob := binancef.ParseAggTrade(data)
		if prob != nil {
			a.logger.Warn("parse aggTrade", "error", prob.Message)
			return
		}

		event, prob := binancef.Normalize(agg, symbol)
		if prob != nil {
			a.logger.Error("normalize trade", "error", prob.Message)
			return
		}

		c.Send(publisherPID, publishTradeMessage{Event: event})
	}, a.logger)

	a.logger.Info("starting websocket adapter", "url", client.StreamURL())

	// Run in a goroutine supervised by the actor lifecycle.
	// When the actor is stopped, cancel() terminates the read loop.
	go func() {
		client.Run(ctx)
		// If Run returns and context is not cancelled, the actor should restart.
		if ctx.Err() == nil {
			a.logger.Error("websocket adapter exited unexpectedly")
			c.Engine().Poison(c.PID())
		}
	}()
}
