package nats

import (
	"context"
	"fmt"
	"log/slog"

	"internal/domain/evidence"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// CandleHandler is called for each decoded evidence candle event.
type CandleHandler func(evidence.CandleSampledEvent)

// EvidenceConsumer is a durable JetStream consumer for evidence candle events.
type EvidenceConsumer struct {
	url      string
	spec     ConsumerSpec
	registry EvidenceRegistry
	handler  CandleHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewEvidenceConsumer(url string, spec ConsumerSpec, registry EvidenceRegistry, handler CandleHandler, logger *slog.Logger) *EvidenceConsumer {
	return &EvidenceConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *EvidenceConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.CandleSampled.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure evidence stream: %w", err)
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

func (c *EvidenceConsumer) onMessage(msg jetstream.Msg) {
	env, prob := decodeEvent[evidence.CandleSampledEvent](c.registry.CandleSampled, msg.Data())
	if prob != nil {
		c.logger.Error("decode evidence event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack evidence event", "error", err)
	}
}

func (c *EvidenceConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term evidence event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak evidence event", "error", err)
	}
}

func (c *EvidenceConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
