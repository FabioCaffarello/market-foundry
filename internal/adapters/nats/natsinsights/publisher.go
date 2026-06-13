package natsinsights

import (
	"context"
	"fmt"
	"strconv"

	"internal/adapters/nats/natskit"
	"internal/domain/insights"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes insights events to the INSIGHTS_EVENTS stream.
// Single-writer (ADR-0008): derive owns this publisher.
type Publisher struct {
	url      string
	source   string
	registry Registry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewPublisher(url, source string, registry Registry) *Publisher {
	return &Publisher{url: url, source: source, registry: registry}
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
	if _, err := js.CreateOrUpdateStream(ctx, p.registry.VolumeProfileSampled.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure insights stream: %w", err)
	}
	p.nc = nc
	p.js = js
	return nil
}

// PublishVolumeProfile publishes a VolumeProfileSampledEvent. The
// subject's {symbol} token is the canonical SubjectToken() (ADR-0009);
// the dedup key keys on the window open time so interim updates for a
// window coalesce while distinct windows stay distinct.
func (p *Publisher) PublishVolumeProfile(ctx context.Context, event insights.VolumeProfileSampledEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "insights publisher is unavailable")
	}

	vp := event.VolumeProfile
	spec := p.registry.VolumeProfileSampled
	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		vp.Source,
		vp.Instrument.SubjectToken(),
		vp.Timeframe,
	)

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	// Canonical SubjectToken() in the dedup key (H-8.a; enforced by
	// check subjects [dedup]).
	dedupKey := "volprofile:" + vp.Source + ":" +
		vp.Instrument.SubjectToken() + ":" +
		strconv.Itoa(vp.Timeframe) + ":" +
		strconv.FormatInt(vp.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish volume profile")
	}
	return nil
}

// PublishTPOProfile publishes a TPOProfileSampledEvent. Same subject /
// dedup scheme as PublishVolumeProfile: the {symbol} token is the
// canonical SubjectToken() (ADR-0009); the dedup key keys on the window
// open time so interim updates for a window coalesce.
func (p *Publisher) PublishTPOProfile(ctx context.Context, event insights.TPOProfileSampledEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "insights publisher is unavailable")
	}

	tp := event.TPOProfile
	spec := p.registry.TPOProfileSampled
	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		tp.Source,
		tp.Instrument.SubjectToken(),
		tp.Timeframe,
	)

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := "tpoprofile:" + tp.Source + ":" +
		tp.Instrument.SubjectToken() + ":" +
		strconv.Itoa(tp.Timeframe) + ":" +
		strconv.FormatInt(tp.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish tpo profile")
	}
	return nil
}

func (p *Publisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
