package ingest

import (
	"context"
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	"internal/adapters/exchanges/binancef"
	"internal/adapters/exchanges/binances"
	"internal/domain/observation"

	"github.com/anthdm/hollywood/actor"
)

// WebSocketAdapterConfig holds the configuration for a single WebSocket adapter.
type WebSocketAdapterConfig struct {
	Source       string // exchange source (e.g., "binancef", "binances")
	Symbol       string
	PublisherPID *actor.PID
}

// WebSocketAdapterActor connects to an exchange-specific aggTrade WebSocket stream,
// normalizes incoming trades, and forwards them to the publisher actor.
// The Source field determines which exchange adapter (Futures or Spot) is used.
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
		a.logger = slog.Default().With("actor", "ws-adapter", "source", a.cfg.Source, "symbol", a.cfg.Symbol)
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

// tradeParser abstracts the parse+normalize pipeline for an exchange source.
type tradeParser struct {
	parse     func([]byte) (interface{}, error)
	normalize func(interface{}, string) (observation.TradeReceivedEvent, error)
	streamURL string
}

func (a *WebSocketAdapterActor) start(c *actor.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	symbol := a.cfg.Symbol
	source := a.cfg.Source
	publisherPID := a.cfg.PublisherPID

	handler := a.buildHandler(c, source, symbol, publisherPID)
	if handler == nil {
		a.logger.Error("unsupported exchange source, cannot start adapter", "source", source)
		c.Engine().Poison(c.PID())
		return
	}

	var streamURL string
	switch source {
	case "binancef":
		client := binancef.NewWSClient(symbol, handler, a.logger)
		streamURL = client.StreamURL()
		a.logger.Info("starting websocket adapter", "url", streamURL)
		go func() {
			client.Run(ctx)
			if ctx.Err() == nil {
				a.logger.Error("websocket adapter exited unexpectedly")
				c.Engine().Poison(c.PID())
			}
		}()
	case "binances":
		client := binances.NewWSClient(symbol, handler, a.logger)
		streamURL = client.StreamURL()
		a.logger.Info("starting websocket adapter", "url", streamURL)
		go func() {
			client.Run(ctx)
			if ctx.Err() == nil {
				a.logger.Error("websocket adapter exited unexpectedly")
				c.Engine().Poison(c.PID())
			}
		}()
	default:
		a.logger.Error("unsupported exchange source", "source", source)
		c.Engine().Poison(c.PID())
	}
}

// buildHandler returns a MessageHandler routed to the correct exchange adapter.
// Returns nil if the source is unsupported.
func (a *WebSocketAdapterActor) buildHandler(c *actor.Context, source, symbol string, publisherPID *actor.PID) func([]byte) {
	switch source {
	case "binancef":
		return func(data []byte) {
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
		}
	case "binances":
		return func(data []byte) {
			agg, prob := binances.ParseAggTrade(data)
			if prob != nil {
				a.logger.Warn("parse aggTrade", "error", prob.Message)
				return
			}
			event, prob := binances.Normalize(agg, symbol)
			if prob != nil {
				a.logger.Error("normalize trade", "error", prob.Message)
				return
			}
			c.Send(publisherPID, publishTradeMessage{Event: event})
		}
	default:
		return nil
	}
}
