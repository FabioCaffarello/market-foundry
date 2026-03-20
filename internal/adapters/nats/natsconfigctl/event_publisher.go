package natsconfigctl

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	configdomain "internal/domain/configctl"
	"internal/shared/events"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type EventPublisher struct {
	url      string
	source   string
	registry Registry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewEventPublisher(url, source string, registry Registry) *EventPublisher {
	return &EventPublisher{
		url:      url,
		source:   source,
		registry: registry,
	}
}

func (p *EventPublisher) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.Activated.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure event stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

func (p *EventPublisher) Publish(ctx context.Context, event events.Event) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "domain event publisher is unavailable")
	}

	spec, prob := p.specFor(event)
	if prob != nil {
		return prob
	}
	data, prob := natskit.EncodeEvent(spec, p.source, event, event.EventMetadata().CorrelationID, event.EventMetadata().CausationID)
	if prob != nil {
		return prob
	}

	if _, err := p.js.Publish(ctx, spec.Subject, data, jetstream.WithMsgID(event.EventMetadata().ID)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish domain event")
	}

	return nil
}

func (p *EventPublisher) specFor(event events.Event) (natskit.EventSpec, *problem.Problem) {
	switch event.(type) {
	case configdomain.DraftCreatedEvent:
		return p.registry.DraftCreated, nil
	case configdomain.ConfigValidatedEvent:
		return p.registry.Validated, nil
	case configdomain.ConfigCompiledEvent:
		return p.registry.Compiled, nil
	case configdomain.ConfigActivatedEvent:
		return p.registry.Activated, nil
	case configdomain.ConfigDeactivatedEvent:
		return p.registry.Deactivated, nil
	case configdomain.IngestionRuntimeChangedEvent:
		return p.registry.IngestionRuntimeChanged, nil
	case configdomain.ConfigArchivedEvent:
		return p.registry.Archived, nil
	case configdomain.ConfigRejectedEvent:
		return p.registry.Rejected, nil
	default:
		return natskit.EventSpec{}, problem.New(problem.InvalidArgument, "domain event type is unsupported")
	}
}

func (p *EventPublisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
