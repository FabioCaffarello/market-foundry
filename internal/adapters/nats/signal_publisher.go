package nats

import (
	"context"
	"fmt"

	"internal/domain/signal"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SignalPublisher publishes signal events to the SIGNAL_EVENTS stream.
type SignalPublisher struct {
	url      string
	source   string
	registry SignalRegistry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewSignalPublisher(url, source string, registry SignalRegistry) *SignalPublisher {
	return &SignalPublisher{
		url:      url,
		source:   source,
		registry: registry,
	}
}

func (p *SignalPublisher) Start() error {
	nc, err := connect(p.url)
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
func (p *SignalPublisher) PublishSignal(ctx context.Context, event signal.SignalGeneratedEvent) *problem.Problem {
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
		event.Signal.Symbol,
		event.Signal.Timeframe,
	)

	data, prob := encodeEvent(*spec, p.source, event, event.Metadata.CorrelationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.Signal.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish signal event")
	}

	return nil
}

func (p *SignalPublisher) specForType(signalType string) *EventSpec {
	switch signalType {
	case "rsi":
		spec := p.registry.RSIGenerated
		return &spec
	default:
		return nil
	}
}

func (p *SignalPublisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
