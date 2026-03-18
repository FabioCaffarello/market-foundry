package ingest

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// ExchangeScopeConfig holds the configuration for an exchange scope actor.
type ExchangeScopeConfig struct {
	Source           string
	NATSURL          string
	Registry         adapternats.ObservationRegistry
	PublisherTracker *healthz.Tracker
}

// ExchangeScopeActor supervises all actors for a single exchange/source.
// It owns the NATS publisher and all WebSocket adapters for that source.
// Lifecycle of all symbols from this exchange is managed here.
type ExchangeScopeActor struct {
	cfg          ExchangeScopeConfig
	logger       *slog.Logger
	publisherPID *actor.PID
	adapters     map[string]*actor.PID // key: symbol → adapter PID
}

func NewExchangeScopeActor(cfg ExchangeScopeConfig) actor.Producer {
	return func() actor.Receiver {
		return &ExchangeScopeActor{
			cfg:      cfg,
			logger:   slog.Default().With("actor", "exchange-scope", "source", cfg.Source),
			adapters: make(map[string]*actor.PID),
		}
	}
}

func (a *ExchangeScopeActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.logger.Info("exchange scope stopped",
			"active_adapters", len(a.adapters),
		)

	case activateBindingMessage:
		a.onActivate(c, msg)

	case clearBindingMessage:
		a.onClear(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *ExchangeScopeActor) start(c *actor.Context) {
	a.publisherPID = c.SpawnChild(NewPublisherActor(PublisherConfig{
		URL:      a.cfg.NATSURL,
		Source:   "ingest.observation-publisher." + a.cfg.Source,
		Registry: a.cfg.Registry,
		Tracker:  a.cfg.PublisherTracker,
	}), "publisher")

	a.logger.Info("exchange scope started")
}

func (a *ExchangeScopeActor) onActivate(c *actor.Context, msg activateBindingMessage) {
	symbol := msg.Target.Symbol
	if _, exists := a.adapters[symbol]; exists {
		a.logger.Info("symbol already active, skipping", "symbol", symbol)
		return
	}

	childName := "ws-" + symbol
	pid := c.SpawnChild(NewWebSocketAdapterActor(WebSocketAdapterConfig{
		Symbol:       symbol,
		PublisherPID: a.publisherPID,
	}), childName)

	a.adapters[symbol] = pid
	a.logger.Info("adapter spawned",
		"symbol", symbol,
		"active_adapters", len(a.adapters),
	)
}

func (a *ExchangeScopeActor) onClear(c *actor.Context, msg clearBindingMessage) {
	symbol := msg.Target.Symbol
	pid, exists := a.adapters[symbol]
	if !exists {
		a.logger.Info("symbol not active, nothing to clear", "symbol", symbol)
		return
	}

	c.Engine().Poison(pid)
	delete(a.adapters, symbol)
	a.logger.Info("adapter stopped",
		"symbol", symbol,
		"active_adapters", len(a.adapters),
	)
}
