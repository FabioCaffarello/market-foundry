package nats

import (
	"context"
	"fmt"

	"internal/domain/strategy"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StrategyPublisher publishes strategy events to the STRATEGY_EVENTS stream.
type StrategyPublisher struct {
	url      string
	source   string
	registry StrategyRegistry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewStrategyPublisher(url, source string, registry StrategyRegistry) *StrategyPublisher {
	return &StrategyPublisher{
		url:      url,
		source:   source,
		registry: registry,
	}
}

func (p *StrategyPublisher) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.MeanReversionEntryResolved.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure strategy stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishStrategy publishes a StrategyResolvedEvent to the strategy stream.
// Subject: strategy.events.{type}.resolved.{source}.{symbol}.{timeframe}
func (p *StrategyPublisher) PublishStrategy(ctx context.Context, event strategy.StrategyResolvedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "strategy publisher is unavailable")
	}

	spec := p.specForType(event.Strategy.Type)
	if spec == nil {
		return problem.New(problem.InvalidArgument, "unknown strategy type: "+event.Strategy.Type)
	}

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.Strategy.Source,
		event.Strategy.Symbol,
		event.Strategy.Timeframe,
	)

	data, prob := encodeEvent(*spec, p.source, event, event.Metadata.CorrelationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.Strategy.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish strategy event")
	}

	return nil
}

func (p *StrategyPublisher) specForType(strategyType string) *EventSpec {
	switch strategyType {
	case "mean_reversion_entry":
		spec := p.registry.MeanReversionEntryResolved
		return &spec
	default:
		return nil
	}
}

func (p *StrategyPublisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
