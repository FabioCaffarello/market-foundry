package natsstrategy

import (
	"context"
	"fmt"
	"log/slog"

	"internal/adapters/nats/natskit"
	"internal/domain/strategy"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Handler is called for each decoded strategy resolved event.
type Handler func(strategy.StrategyResolvedEvent)

// Consumer is a durable JetStream consumer for strategy events.
type Consumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry Registry
	handler  Handler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewConsumer(url string, spec natskit.ConsumerSpec, registry Registry, handler Handler, logger *slog.Logger) *Consumer {
	return &Consumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *Consumer) Start() error {
	nc, err := natskit.Connect(c.url)
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), natskit.DefaultSetupTimeout)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.MeanReversionEntryResolved.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure strategy stream: %w", err)
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, c.spec.Event.Stream.Name, jetstream.ConsumerConfig{
		Durable:       c.spec.Durable,
		FilterSubject: c.spec.Event.Subject,
		AckWait:       c.spec.AckWait,
		MaxDeliver:    c.spec.MaxDeliver,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("create durable consumer: %w", err)
	}

	consumeCtx, err := cons.Consume(c.onMessage)
	if err != nil {
		nc.Close()
		return fmt.Errorf("start consume: %w", err)
	}

	c.nc = nc
	c.consumer = consumeCtx
	return nil
}

func (c *Consumer) onMessage(msg jetstream.Msg) {
	env, prob := natskit.DecodeEvent[strategy.StrategyResolvedEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode strategy event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack strategy event", "error", err)
	}
}

func (c *Consumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	// Permanent decode errors should not be redelivered.
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term strategy event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak strategy event", "error", err)
	}
}

func (c *Consumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
