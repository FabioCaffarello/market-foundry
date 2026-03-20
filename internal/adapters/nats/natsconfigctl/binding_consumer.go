package natsconfigctl

import (
	"context"
	"fmt"
	"log/slog"

	"internal/adapters/nats/natskit"
	configdomain "internal/domain/configctl"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type BindingHandler func(configdomain.IngestionRuntimeChangedEvent)

type BindingConsumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry Registry
	handler  BindingHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewBindingConsumer(url string, spec natskit.ConsumerSpec, registry Registry, handler BindingHandler, logger *slog.Logger) *BindingConsumer {
	return &BindingConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *BindingConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.Activated.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure configctl stream: %w", err)
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, c.spec.Event.Stream.Name, jetstream.ConsumerConfig{
		Durable:       c.spec.Durable,
		FilterSubject: c.spec.Event.Subject,
		AckWait:       c.spec.AckWait,
		MaxDeliver:    c.spec.MaxDeliver,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverLastPerSubjectPolicy,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("create binding consumer: %w", err)
	}

	consumeCtx, err := cons.Consume(c.onMessage)
	if err != nil {
		nc.Close()
		return fmt.Errorf("start binding consume: %w", err)
	}

	c.nc = nc
	c.consumer = consumeCtx
	return nil
}

func (c *BindingConsumer) onMessage(msg jetstream.Msg) {
	env, prob := natskit.DecodeEvent[configdomain.IngestionRuntimeChangedEvent](c.registry.IngestionRuntimeChanged, msg.Data())
	if prob != nil {
		c.logger.Error("decode binding event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack binding event", "error", err)
	}
}

func (c *BindingConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term binding event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak binding event", "error", err)
	}
}

func (c *BindingConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
