package delivery

import (
	"io"
	"log/slog"

	"internal/adapters/nats/natsdelivery"
	"internal/adapters/nats/natsinsights"

	"github.com/anthdm/hollywood/actor"
)

// Runtime is the wired delivery subsystem: a Hub the HTTP layer uses to
// admit connections, plus the NATS consumer's Closer.
type Runtime struct {
	Hub      *Hub
	consumer io.Closer
}

// Close stops the NATS consumer. The router and session actors are
// engine-managed and stop with the engine.
func (r *Runtime) Close() error {
	if r == nil || r.consumer == nil {
		return nil
	}
	return r.consumer.Close()
}

// Start spawns the delivery router on the engine, starts the durable
// insights consumer feeding it, and returns the wired Runtime. If the
// consumer fails to start (e.g. NATS unavailable) the router is poisoned
// and the error is returned so the gateway can degrade gracefully
// (no /ws endpoint) rather than fail to boot.
func Start(engine *actor.Engine, natsURL string, logger *slog.Logger) (*Runtime, error) {
	if logger == nil {
		logger = slog.Default()
	}
	router := engine.Spawn(NewRouterActor(), "delivery-router")

	consumer := natsdelivery.NewConsumer(
		natsURL,
		natsdelivery.DeliverInsightsConsumer(),
		natsinsights.DefaultRegistry(),
		func(f natsdelivery.Frame) {
			engine.Send(router, eventReceivedMessage{Subject: f.Subject, Payload: f.Payload})
		},
		logger,
	)
	if err := consumer.Start(); err != nil {
		engine.Poison(router)
		return nil, err
	}

	return &Runtime{Hub: NewHub(engine, router, logger), consumer: consumer}, nil
}
