package nats

import (
	"context"
	"fmt"
	"log/slog"

	"internal/domain/risk"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// RiskHandler is called for each decoded risk assessed event.
type RiskHandler func(risk.RiskAssessedEvent)

// RiskConsumer is a durable JetStream consumer for risk events.
type RiskConsumer struct {
	url      string
	spec     ConsumerSpec
	registry RiskRegistry
	handler  RiskHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewRiskConsumer(url string, spec ConsumerSpec, registry RiskRegistry, handler RiskHandler, logger *slog.Logger) *RiskConsumer {
	return &RiskConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *RiskConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.PositionExposureAssessed.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure risk stream: %w", err)
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

func (c *RiskConsumer) onMessage(msg jetstream.Msg) {
	meta, _ := msg.Metadata()
	if meta != nil && meta.NumDelivered > 1 {
		c.logger.Warn("risk event redelivered",
			"subject", msg.Subject(),
			"num_delivered", meta.NumDelivered,
			"stream_seq", meta.Sequence.Stream,
		)
	}

	env, prob := decodeEvent[risk.RiskAssessedEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode risk event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack risk event", "error", err)
	}
}

func (c *RiskConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term risk event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak risk event", "error", err)
	}
}

func (c *RiskConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
