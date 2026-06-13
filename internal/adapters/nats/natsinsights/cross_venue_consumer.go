package natsinsights

import (
	"context"
	"fmt"
	"log/slog"

	"internal/adapters/nats/natskit"
	"internal/domain/insights"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type CrossVenueHandler func(insights.CrossVenueSampledEvent)

// CrossVenueConsumer is a durable consumer of cross-venue snapshot
// events from INSIGHTS_EVENTS. Used by the store binary to project
// snapshots into the KV latest bucket (PROGRAM-0005 / H-8.c).
type CrossVenueConsumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry Registry
	handler  CrossVenueHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewCrossVenueConsumer(url string, spec natskit.ConsumerSpec, registry Registry, handler CrossVenueHandler, logger *slog.Logger) *CrossVenueConsumer {
	return &CrossVenueConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *CrossVenueConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.CrossVenueSampled.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure insights stream: %w", err)
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

func (c *CrossVenueConsumer) onMessage(msg jetstream.Msg) {
	env, prob := natskit.DecodeEvent[insights.CrossVenueSampledEvent](c.registry.CrossVenueSampled, msg.Data())
	if prob != nil {
		c.logger.Error("decode cross venue event", "error", prob.Message, "subject", msg.Subject())
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack cross venue event", "error", err)
	}
}

func (c *CrossVenueConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term cross venue event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak cross venue event", "error", err)
	}
}

func (c *CrossVenueConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
