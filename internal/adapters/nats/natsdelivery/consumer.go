// Package natsdelivery is the NATS adapter for the delivery subsystem:
// a durable reader of INSIGHTS_EVENTS that decodes events and forwards
// them as JSON frames to the delivery router. Delivery is a free reader
// (ADR-0008 single-writer: derive remains the only publisher to
// INSIGHTS_EVENTS); this package never writes to the stream.
package natsdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"internal/adapters/nats/natsinsights"
	"internal/adapters/nats/natskit"
	"internal/domain/insights"
	"internal/shared/problem"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Frame is a decoded insights event ready to forward to subscribed
// delivery clients: the concrete NATS subject it was published on, plus
// the event re-encoded as JSON for the WebSocket wire.
type Frame struct {
	Subject string
	Payload []byte
}

// Handler receives each decoded delivery frame. It runs on the NATS
// consume goroutine, so it must be cheap and non-blocking — the actor
// it feeds owns any buffering/backpressure (ADR-0028 I4).
type Handler func(Frame)

// DeliverInsightsConsumer is the durable consumer spec the gateway uses
// to feed the delivery router. H-11.a scopes it to volume-profile
// events; H-11.b widens the filter subject to all insights events.
func DeliverInsightsConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec(
		"deliver-insights",
		"insights.events.volumeprofile.sampled.>",
		"insights.events.v1.volume_profile_sampled",
		"INSIGHTS_EVENTS",
	)
}

// Consumer is a durable JetStream consumer that decodes insights events
// and forwards them as JSON frames via its handler.
type Consumer struct {
	url      string
	spec     natskit.ConsumerSpec
	registry natsinsights.Registry
	handler  Handler
	logger   *slog.Logger
	nc       *natsclient.Conn
	consumer jetstream.ConsumeContext
}

// NewConsumer builds a delivery consumer. registry supplies the canonical
// insights EventSpec (subject/type/stream) so the wire contract has a
// single source of truth (natsinsights), not a duplicated literal.
func NewConsumer(url string, spec natskit.ConsumerSpec, registry natsinsights.Registry, handler Handler, logger *slog.Logger) *Consumer {
	return &Consumer{url: url, spec: spec, registry: registry, handler: handler, logger: logger}
}

// Start connects, ensures the durable consumer exists, and begins
// consuming. The stream itself is owned by the publisher (derive); a
// CreateOrUpdateStream here is idempotent and keeps the gateway able to
// start before derive has run.
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

	if _, err := js.CreateOrUpdateStream(ctx, c.registry.VolumeProfileSampled.Stream.Config()); err != nil {
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

func (c *Consumer) onMessage(msg jetstream.Msg) {
	env, prob := natskit.DecodeEvent[insights.VolumeProfileSampledEvent](c.registry.VolumeProfileSampled, msg.Data())
	if prob != nil {
		c.logger.Error("decode delivery event", "error", prob.Message, "subject", msg.Subject())
		c.terminateOrNak(msg, prob)
		return
	}

	// Re-encode to JSON for the client wire (the stream carries CBOR).
	payload, err := json.Marshal(env.Payload)
	if err != nil {
		c.logger.Error("marshal delivery frame", "error", err, "subject", msg.Subject())
		if termErr := msg.Term(); termErr != nil { // un-encodable: do not redeliver
			c.logger.Error("term delivery event", "error", termErr)
		}
		return
	}

	c.handler(Frame{Subject: msg.Subject(), Payload: payload})

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack delivery event", "error", err)
	}
}

func (c *Consumer) terminateOrNak(msg jetstream.Msg, prob *problem.Problem) {
	if prob.Code == problem.InvalidArgument {
		if err := msg.Term(); err != nil {
			c.logger.Error("term delivery event", "error", err)
		}
		return
	}
	if err := msg.Nak(); err != nil {
		c.logger.Error("nak delivery event", "error", err)
	}
}

// Close stops consuming and closes the NATS connection. Idempotent.
func (c *Consumer) Close() error {
	if c.consumer != nil {
		c.consumer.Stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	return nil
}
