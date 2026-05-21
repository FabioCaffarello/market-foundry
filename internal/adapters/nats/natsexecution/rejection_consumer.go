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

// RejectionHandler is called for each decoded venue order rejected event.
type RejectionHandler func(execution.VenueOrderRejectedEvent)

// RejectionConsumer is a durable JetStream consumer for venue order rejection events.
// S387: Closes the projection gap from S386 — rejection events now reach the store KV.
type RejectionConsumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry Registry
	handler  RejectionHandler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext

	delivered   atomic.Int64
	redelivered atomic.Int64
	terminated  atomic.Int64
	nakked      atomic.Int64
}

func NewRejectionConsumer(url string, spec natskit.ConsumerSpec, registry Registry, handler RejectionHandler, logger *slog.Logger) *RejectionConsumer {
	return &RejectionConsumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
	}
}

func (c *RejectionConsumer) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.VenueMarketOrderRejected.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure rejection events stream: %w", err)
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
		return fmt.Errorf("create durable rejection consumer: %w", err)
	}

	consumeCtx, err := cons.Consume(c.onMessage)
	if err != nil {
		nc.Close()
		return fmt.Errorf("start rejection consume: %w", err)
	}

	c.nc = nc
	c.consumer = consumeCtx
	return nil
}

func (c *RejectionConsumer) onMessage(msg jetstream.Msg) {
	start := time.Now()
	c.delivered.Add(1)
	metrics.IncConsumerMessage(c.spec.Durable, "delivered")

	meta, _ := msg.Metadata()
	if meta != nil && meta.NumDelivered > 1 {
		c.redelivered.Add(1)
		metrics.IncConsumerMessage(c.spec.Durable, "redelivered")
		c.logger.Warn("rejection event redelivered",
			"subject", msg.Subject(),
			"num_delivered", meta.NumDelivered,
			"max_deliver", c.spec.MaxDeliver,
			"stream_seq", meta.Sequence.Stream,
		)
	}

	if meta != nil {
		metrics.SetConsumerLag(c.spec.Durable, float64(meta.NumPending))
	}

	if meta != nil && c.spec.MaxDeliver > 0 && meta.NumDelivered >= uint64(c.spec.MaxDeliver) {
		c.logger.Error("rejection event at max delivery — will be terminated by NATS after this attempt",
			"subject", msg.Subject(),
			"num_delivered", meta.NumDelivered,
			"max_deliver", c.spec.MaxDeliver,
			"stream_seq", meta.Sequence.Stream,
		)
	}

	env, prob := natskit.DecodeEvent[execution.VenueOrderRejectedEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode rejection event",
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
		c.logger.Error("ack rejection event", "error", err)
	}

	metrics.ObserveConsumerProcessing(c.spec.Durable, time.Since(start))
}

func (c *RejectionConsumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		c.terminated.Add(1)
		metrics.IncConsumerMessage(c.spec.Durable, "terminated")
		c.logger.Warn("rejection event terminated (non-recoverable)",
			"subject", msg.Subject(),
			"reason", prob.Message,
		)
		if err := msg.Term(); err != nil {
			c.logger.Error("term rejection event", "error", err)
		}
		return
	}
	c.nakked.Add(1)
	metrics.IncConsumerMessage(c.spec.Durable, "nakked")
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak rejection event", "error", err)
	}
}

// Stats returns consumer-side delivery statistics for diagnostics.
func (c *RejectionConsumer) Stats() (delivered, redelivered, terminated, nakked int64) {
	return c.delivered.Load(), c.redelivered.Load(), c.terminated.Load(), c.nakked.Load()
}

func (c *RejectionConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
