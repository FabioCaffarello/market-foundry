package natssignal

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/signal"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes signal events to the SIGNAL_EVENTS stream.
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.RSIGenerated.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure signal stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishSignal publishes a SignalGeneratedEvent to the signal stream.
// Subject: signal.events.{type}.generated.{source}.{symbol}.{timeframe}
func (p *Publisher) PublishSignal(ctx context.Context, event signal.SignalGeneratedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "signal publisher is unavailable")
	}

	spec := p.specForType(event.Signal.Type)
	if spec == nil {
		return problem.New(problem.InvalidArgument, "unknown signal type: "+event.Signal.Type)
	}

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.Signal.Source,
		event.Signal.Instrument.SubjectToken(),
		event.Signal.Timeframe,
	)

	data, prob := natskit.EncodeEvent(*spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.Signal.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish signal event")
	}

	return nil
}

func (p *Publisher) specForType(signalType string) *natskit.EventSpec {
	switch signalType {
	case "rsi":
		spec := p.registry.RSIGenerated
		return &spec
	case "ema_crossover":
		spec := p.registry.EMACrossoverGenerated
		return &spec
	default:
		return nil
	}
}

func (p *Publisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
