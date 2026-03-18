package nats

import (
	"context"
	"fmt"

	"internal/domain/risk"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// RiskPublisher publishes risk events to the RISK_EVENTS stream.
type RiskPublisher struct {
	url      string
	source   string
	registry RiskRegistry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewRiskPublisher(url, source string, registry RiskRegistry) *RiskPublisher {
	return &RiskPublisher{
		url:      url,
		source:   source,
		registry: registry,
	}
}

func (p *RiskPublisher) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.PositionExposureAssessed.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure risk stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishRisk publishes a RiskAssessedEvent to the risk stream.
// Subject: risk.events.{type}.assessed.{source}.{symbol}.{timeframe}
func (p *RiskPublisher) PublishRisk(ctx context.Context, event risk.RiskAssessedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "risk publisher is unavailable")
	}

	spec := p.specForType(event.RiskAssessment.Type)
	if spec == nil {
		return problem.New(problem.InvalidArgument, "unknown risk type: "+event.RiskAssessment.Type)
	}

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.RiskAssessment.Source,
		event.RiskAssessment.Symbol,
		event.RiskAssessment.Timeframe,
	)

	data, prob := encodeEvent(*spec, p.source, event, event.Metadata.CorrelationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.RiskAssessment.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish risk event")
	}

	return nil
}

func (p *RiskPublisher) specForType(riskType string) *EventSpec {
	switch riskType {
	case "position_exposure":
		spec := p.registry.PositionExposureAssessed
		return &spec
	default:
		return nil
	}
}

func (p *RiskPublisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
