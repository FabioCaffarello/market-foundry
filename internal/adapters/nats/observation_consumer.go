package nats

import (
	"context"
	"fmt"
	"log/slog"

	"internal/domain/observation"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TradeHandler is called for each decoded observation trade.
type TradeHandler func(observation.TradeReceivedEvent)

// ObservationConsumer is a durable JetStream consumer for observation trade events.
type ObservationConsumer struct {
	url      string
	spec     ConsumerSpec
	obsReg   ObservationRegistry
	handler  TradeHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewObservationConsumer(url string, spec ConsumerSpec, obsReg ObservationRegistry, handler TradeHandler, logger *slog.Logger) *ObservationConsumer {
	return &ObservationConsumer{
		url:    url,
		spec:   spec,
		obsReg: obsReg,
		handler: handler,
		logger: logger,
	}
}

func (c *ObservationConsumer) Start() error {
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

func (c *ObservationConsumer) onMessage(msg jetstream.Msg) {
	env, prob := decodeEvent[observation.TradeReceivedEvent](c.obsReg.TradeReceived, msg.Data())
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

func (c *ObservationConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
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

func (c *ObservationConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
