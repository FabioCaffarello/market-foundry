package nats

import (
	"context"
	"fmt"
	"log/slog"

	"internal/domain/signal"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SignalHandler is called for each decoded signal generated event.
type SignalHandler func(signal.SignalGeneratedEvent)

// SignalConsumer is a durable JetStream consumer for signal events.
type SignalConsumer struct {
	url      string
	spec     ConsumerSpec
	registry SignalRegistry
	handler  SignalHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewSignalConsumer(url string, spec ConsumerSpec, registry SignalRegistry, handler SignalHandler, logger *slog.Logger) *SignalConsumer {
	return &SignalConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *SignalConsumer) Start() error {
	nc, err := connect(c.url)
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultSetupTimeout)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.RSIGenerated.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure signal stream: %w", err)
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

func (c *SignalConsumer) onMessage(msg jetstream.Msg) {
	env, prob := decodeEvent[signal.SignalGeneratedEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode signal event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack signal event", "error", err)
	}
}

func (c *SignalConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term signal event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak signal event", "error", err)
	}
}

func (c *SignalConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
