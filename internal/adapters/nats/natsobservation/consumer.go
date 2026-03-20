package natsobservation

import (
	"context"
	"fmt"
	"log/slog"

	"internal/adapters/nats/natskit"
	"internal/domain/observation"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TradeHandler is called for each decoded observation trade.
type TradeHandler func(observation.TradeReceivedEvent)

// Consumer is a durable JetStream consumer for observation trade events.
type Consumer struct {
	url      string
	spec     natskit.ConsumerSpec
	obsReg   Registry
	handler  TradeHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewConsumer(url string, spec natskit.ConsumerSpec, obsReg Registry, handler TradeHandler, logger *slog.Logger) *Consumer {
	return &Consumer{
		url:     url,
		spec:    spec,
		obsReg:  obsReg,
		handler: handler,
		logger:  logger,
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

	// Ensure the stream exists (it should already be created by ingest).
	if _, err := js.CreateOrUpdateStream(ctx, c.obsReg.TradeReceived.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure observation stream: %w", err)
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
	env, prob := natskit.DecodeEvent[observation.TradeReceivedEvent](c.obsReg.TradeReceived, msg.Data())
	if prob != nil {
		c.logger.Error("decode observation event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack observation event", "error", err)
	}
}

func (c *Consumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	// Permanent decode errors should not be redelivered.
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term observation event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak observation event", "error", err)
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
