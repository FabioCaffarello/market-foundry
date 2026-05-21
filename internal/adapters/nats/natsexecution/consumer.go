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

// Handler is called for each decoded execution event.
type Handler func(execution.PaperOrderSubmittedEvent)

// Consumer is a durable JetStream consumer for execution events.
type Consumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry Registry
	handler  Handler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext

	delivered   atomic.Int64
	redelivered atomic.Int64
	terminated  atomic.Int64
	nakked      atomic.Int64
}

func NewConsumer(url string, spec natskit.ConsumerSpec, registry Registry, handler Handler, logger *slog.Logger) *Consumer {
	return &Consumer{
		url:      url,
		spec:     spec,
		registry: registry,
		handler:  handler,
		logger:   logger,
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.PaperOrderSubmitted.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure execution stream: %w", err)
	}

	// S401: When FilterSubjects is set, use multi-subject filtering for segment-scoped
	// consumers. Otherwise fall back to single FilterSubject from Event.Subject.
	consumerCfg := jetstream.ConsumerConfig{
		Durable:    c.spec.Durable,
		AckWait:    c.spec.AckWait,
		MaxDeliver: c.spec.MaxDeliver,
		AckPolicy:  jetstream.AckExplicitPolicy,
	}
	if len(c.spec.FilterSubjects) > 0 {
		consumerCfg.FilterSubjects = c.spec.FilterSubjects
	} else {
		consumerCfg.FilterSubject = c.spec.Event.Subject
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

func (c *Consumer) onMessage(msg jetstream.Msg) {
	start := time.Now()
	c.delivered.Add(1)
	metrics.IncConsumerMessage(c.spec.Durable, "delivered")

	meta, _ := msg.Metadata()
	if meta != nil && meta.NumDelivered > 1 {
		c.redelivered.Add(1)
		metrics.IncConsumerMessage(c.spec.Durable, "redelivered")
		c.logger.Warn("execution event redelivered",
			"subject", msg.Subject(),
			"num_delivered", meta.NumDelivered,
			"max_deliver", c.spec.MaxDeliver,
			"stream_seq", meta.Sequence.Stream,
		)
	}

	// Update consumer lag from message metadata.
	if meta != nil {
		metrics.SetConsumerLag(c.spec.Durable, float64(meta.NumPending))
	}

	// Detect near-exhaustion: warn when approaching MaxDeliver limit.
	if meta != nil && c.spec.MaxDeliver > 0 && meta.NumDelivered >= uint64(c.spec.MaxDeliver) {
		c.logger.Error("execution event at max delivery — will be terminated by NATS after this attempt",
			"subject", msg.Subject(),
			"num_delivered", meta.NumDelivered,
			"max_deliver", c.spec.MaxDeliver,
			"stream_seq", meta.Sequence.Stream,
		)
	}

	env, prob := natskit.DecodeEvent[execution.PaperOrderSubmittedEvent](c.spec.Event, msg.Data())
	if prob != nil {
		c.logger.Error("decode execution event",
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
		c.logger.Error("ack execution event", "error", err)
	}

	metrics.ObserveConsumerProcessing(c.spec.Durable, time.Since(start))
}

func (c *Consumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		c.terminated.Add(1)
		metrics.IncConsumerMessage(c.spec.Durable, "terminated")
		c.logger.Warn("execution event terminated (non-recoverable)",
			"subject", msg.Subject(),
			"reason", prob.Message,
		)
		if err := msg.Term(); err != nil {
			c.logger.Error("term execution event", "error", err)
		}
		return
	}
	c.nakked.Add(1)
	metrics.IncConsumerMessage(c.spec.Durable, "nakked")
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak execution event", "error", err)
	}
}

// Stats returns consumer-side delivery statistics for diagnostics.
func (c *Consumer) Stats() (delivered, redelivered, terminated, nakked int64) {
	return c.delivered.Load(), c.redelivered.Load(), c.terminated.Load(), c.nakked.Load()
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
