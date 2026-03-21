package natsdecision

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/decision"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes decision events to the DECISION_EVENTS stream.
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.RSIOversoldEvaluated.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure decision stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishDecision publishes a DecisionEvaluatedEvent to the decision stream.
// Subject: decision.events.{type}.evaluated.{source}.{symbol}.{timeframe}
func (p *Publisher) PublishDecision(ctx context.Context, event decision.DecisionEvaluatedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "decision publisher is unavailable")
	}

	spec := p.specForType(event.Decision.Type)
	if spec == nil {
		return problem.New(problem.InvalidArgument, "unknown decision type: "+event.Decision.Type)
	}

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.Decision.Source,
		event.Decision.Symbol,
		event.Decision.Timeframe,
	)

	data, prob := natskit.EncodeEvent(*spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.Decision.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish decision event")
	}

	return nil
}

func (p *Publisher) specForType(decisionType string) *natskit.EventSpec {
	switch decisionType {
	case "rsi_oversold":
		spec := p.registry.RSIOversoldEvaluated
		return &spec
	case "ema_crossover":
		spec := p.registry.EMACrossoverEvaluated
		return &spec
	case "bollinger_squeeze":
		spec := p.registry.BollingerSqueezeEvaluated
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
