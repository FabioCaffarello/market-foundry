package natsexecution

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/execution"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes execution events to the EXECUTION_EVENTS stream.
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.PaperOrderSubmitted.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure execution events stream: %w", err)
	}

	// Ensure fill events stream exists (used by execute binary).
	if _, err := js.CreateOrUpdateStream(ctx, p.registry.VenueMarketOrderFilled.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure execution fill stream: %w", err)
	}

	// S386: Ensure rejection events stream exists.
	if _, err := js.CreateOrUpdateStream(ctx, p.registry.VenueMarketOrderRejected.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure execution rejection stream: %w", err)
	}

	// S490: Ensure session lifecycle events stream exists.
	if _, err := js.CreateOrUpdateStream(ctx, p.registry.SessionLifecycle.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure session lifecycle stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishExecution publishes a PaperOrderSubmittedEvent to the execution stream.
// Subject: execution.events.{type}.submitted.{source}.{symbol}.{timeframe}
func (p *Publisher) PublishExecution(ctx context.Context, event execution.PaperOrderSubmittedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "execution publisher is unavailable")
	}

	spec := p.specForType(event.ExecutionIntent.Type)
	if spec == nil {
		return problem.New(problem.InvalidArgument, "unknown execution type: "+event.ExecutionIntent.Type)
	}

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.ExecutionIntent.Source,
		event.ExecutionIntent.Symbol,
		event.ExecutionIntent.Timeframe,
	)

	data, prob := natskit.EncodeEvent(*spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.ExecutionIntent.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish execution event")
	}

	return nil
}

func (p *Publisher) specForType(execType string) *natskit.EventSpec {
	switch execType {
	case "paper_order":
		spec := p.registry.PaperOrderSubmitted
		return &spec
	default:
		return nil
	}
}

// PublishFill publishes a VenueOrderFilledEvent to the EXECUTION_FILL_EVENTS stream.
// Subject: execution.fill.venue_market_order.{source}.{symbol}.{timeframe}
func (p *Publisher) PublishFill(ctx context.Context, event execution.VenueOrderFilledEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "execution publisher is unavailable")
	}

	spec := p.registry.VenueMarketOrderFilled

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.ExecutionIntent.Source,
		event.ExecutionIntent.Symbol,
		event.ExecutionIntent.Timeframe,
	)

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := fmt.Sprintf("fill:%s:%d", event.VenueOrderID, event.ExecutionIntent.Timestamp.Unix())

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish fill event")
	}

	return nil
}

// PublishRejection publishes a VenueOrderRejectedEvent to the EXECUTION_REJECTION_EVENTS stream.
// Subject: execution.rejection.venue_market_order.{source}.{symbol}.{timeframe}
// S386: closes the observability gap — venue rejections now have an auditable event trail.
func (p *Publisher) PublishRejection(ctx context.Context, event execution.VenueOrderRejectedEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "execution publisher is unavailable")
	}

	spec := p.registry.VenueMarketOrderRejected

	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.ExecutionIntent.Source,
		event.ExecutionIntent.Symbol,
		event.ExecutionIntent.Timeframe,
	)

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := fmt.Sprintf("rejection:%s:%s:%d:%d",
		event.ExecutionIntent.Source,
		event.ExecutionIntent.Symbol,
		event.ExecutionIntent.Timeframe,
		event.ExecutionIntent.Timestamp.Unix(),
	)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish rejection event")
	}

	return nil
}

// PublishSessionLifecycle publishes a SessionLifecycleEvent to the SESSION_LIFECYCLE_EVENTS stream.
// Subject: execution.session.lifecycle.{status}
// S490: Event-driven trigger for automated verification.
func (p *Publisher) PublishSessionLifecycle(ctx context.Context, event execution.SessionLifecycleEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "execution publisher is unavailable")
	}

	spec := p.registry.SessionLifecycle

	subject := fmt.Sprintf("%s.%s", spec.Subject, string(event.Status))

	data, prob := natskit.EncodeEvent(spec, p.source, event, "", "")
	if prob != nil {
		return prob
	}

	dedupKey := event.DeduplicationKey()

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish session lifecycle event")
	}

	return nil
}

func (p *Publisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
