package natsexecution

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/execution"
	"internal/shared/metrics"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SessionLifecycleHandler is called for each decoded session lifecycle event.
type SessionLifecycleHandler func(execution.SessionLifecycleEvent)

// SessionLifecycleConsumer is a durable JetStream consumer for session lifecycle events.
// S490: Used by the gateway binary to trigger verification on session close/halt.
type SessionLifecycleConsumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry Registry
	handler  SessionLifecycleHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext

	delivered   atomic.Int64
	redelivered atomic.Int64
	terminated  atomic.Int64
	nakked      atomic.Int64
}

func NewSessionLifecycleConsumer(url string, spec natskit.ConsumerSpec, registry Registry, handler SessionLifecycleHandler, logger *slog.Logger) *SessionLifecycleConsumer {
	return &SessionLifecycleConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *SessionLifecycleConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.SessionLifecycle.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure session lifecycle stream: %w", err)
	}

	consumerCfg := jetstream.ConsumerConfig{
		Durable:       c.spec.Durable,
		FilterSubject: c.spec.Event.Subject,
		AckWait:       c.spec.AckWait,
		MaxDeliver:    c.spec.MaxDeliver,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, c.spec.Event.Stream.Name, consumerCfg)
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

func (c *SessionLifecycleConsumer) onMessage(msg jetstream.Msg) {
	start := time.Now()
	c.delivered.Add(1)
	metrics.IncConsumerMessage(c.spec.Durable, "delivered")

	meta, _ := msg.Metadata()
	if meta != nil && meta.NumDelivered > 1 {
		c.redelivered.Add(1)
		metrics.IncConsumerMessage(c.spec.Durable, "redelivered")
		c.logger.Warn("session lifecycle event redelivered",
			"subject", msg.Subject(),
			"num_delivered", meta.NumDelivered,
			"stream_seq", meta.Sequence.Stream,
		)
	}

	if meta != nil {
		metrics.SetConsumerLag(c.spec.Durable, float64(meta.NumPending))
	}

	env, prob := natskit.DecodeEvent[execution.SessionLifecycleEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode session lifecycle event",
			"error", prob.Message,
			"code", prob.Code,
			"subject", msg.Subject(),
		)
		c.terminateOrNak(msg, prob)
		metrics.ObserveConsumerProcessing(c.spec.Durable, time.Since(start))
		return
	}

	c.handler(env.Payload)

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack session lifecycle event", "error", err)
	}

	metrics.ObserveConsumerProcessing(c.spec.Durable, time.Since(start))
}

func (c *SessionLifecycleConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		c.terminated.Add(1)
		metrics.IncConsumerMessage(c.spec.Durable, "terminated")
		c.logger.Warn("session lifecycle event terminated (non-recoverable)",
			"subject", msg.Subject(),
			"reason", prob.Message,
		)
		if err := msg.Term(); err != nil {
			c.logger.Error("term session lifecycle event", "error", err)
		}
		return
	}
	c.nakked.Add(1)
	metrics.IncConsumerMessage(c.spec.Durable, "nakked")
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak session lifecycle event", "error", err)
	}
}

func (c *SessionLifecycleConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
