package natsobservation

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/observation"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes normalized trade events to the OBSERVATION_EVENTS stream.
type Publisher struct {
	url      string
	source   string
	registry Registry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewPublisher(url, source string, registry Registry) *Publisher {
	return &Publisher{
		url:      url,
		source:   source,
		registry: registry,
	}
}

func (p *Publisher) Start() error {
	nc, err := natskit.Connect(p.url)
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.TradeReceived.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure observation stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishTrade publishes a TradeReceivedEvent to the observation stream.
// The subject is extended with the trade source for partition ordering.
func (p *Publisher) PublishTrade(ctx context.Context, event observation.TradeReceivedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "observation publisher is unavailable")
	}

	spec := p.registry.TradeReceived
	subject := spec.Subject + "." + event.Trade.Source

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.Trade.DeduplicationKey()
	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish observation trade")
	}

	return nil
}

func (p *Publisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
