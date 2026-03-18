package nats

import (
	"context"
	"fmt"
	"log/slog"

	"internal/domain/decision"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// DecisionHandler is called for each decoded decision evaluated event.
type DecisionHandler func(decision.DecisionEvaluatedEvent)

// DecisionConsumer is a durable JetStream consumer for decision events.
type DecisionConsumer struct {
	url      string
	spec     ConsumerSpec
	registry DecisionRegistry
	handler  DecisionHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

func NewDecisionConsumer(url string, spec ConsumerSpec, registry DecisionRegistry, handler DecisionHandler, logger *slog.Logger) *DecisionConsumer {
	return &DecisionConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *DecisionConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.RSIOversoldEvaluated.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure decision stream: %w", err)
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

func (c *DecisionConsumer) onMessage(msg jetstream.Msg) {
	env, prob := decodeEvent[decision.DecisionEvaluatedEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode decision event",
			"error", prob.Message,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack decision event", "error", err)
	}
}

func (c *DecisionConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term decision event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak decision event", "error", err)
	}
}

func (c *DecisionConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
