package nats

import (
	"context"
	"fmt"
	"strconv"

	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// EvidencePublisher publishes candle sampled events to the EVIDENCE_EVENTS stream.
type EvidencePublisher struct {
	url      string
	source   string
	registry EvidenceRegistry
	nc       *nats.Conn
	js       jetstream.JetStream
}

func NewEvidencePublisher(url, source string, registry EvidenceRegistry) *EvidencePublisher {
	return &EvidencePublisher{
		url:      url,
		source:   source,
		registry: registry,
	}
}

func (p *EvidencePublisher) Start() error {
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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.CandleSampled.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure evidence stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

// PublishCandle publishes a CandleSampledEvent to the evidence stream.
// Subject: evidence.events.candle.sampled.{source}.{symbol}.{timeframe}
func (p *EvidencePublisher) PublishCandle(ctx context.Context, event evidence.CandleSampledEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "evidence publisher is unavailable")
	}

	spec := p.registry.CandleSampled
	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.Candle.Source,
		event.Candle.Symbol,
		event.Candle.Timeframe,
	)

	data, prob := encodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	// Dedup key: source:symbol:timeframe:open_time_unix — one candle per window.
	dedupKey := event.Candle.Source + ":" +
		event.Candle.Symbol + ":" +
		strconv.Itoa(event.Candle.Timeframe) + ":" +
		strconv.FormatInt(event.Candle.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish evidence candle")
	}

	return nil
}

// PublishTradeBurst publishes a TradeBurstSampledEvent to the evidence stream.
// Subject: evidence.events.tradeburst.sampled.{source}.{symbol}.{timeframe}
func (p *EvidencePublisher) PublishTradeBurst(ctx context.Context, event evidence.TradeBurstSampledEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "evidence publisher is unavailable")
	}

	spec := p.registry.TradeBurstSampled
	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.TradeBurst.Source,
		event.TradeBurst.Symbol,
		event.TradeBurst.Timeframe,
	)

	data, prob := encodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	// Dedup key: burst:source:symbol:timeframe:open_time_unix — one burst per window.
	dedupKey := "burst:" + event.TradeBurst.Source + ":" +
		event.TradeBurst.Symbol + ":" +
		strconv.Itoa(event.TradeBurst.Timeframe) + ":" +
		strconv.FormatInt(event.TradeBurst.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish evidence trade burst")
	}

	return nil
}

// PublishVolume publishes a VolumeSampledEvent to the evidence stream.
// Subject: evidence.events.volume.sampled.{source}.{symbol}.{timeframe}
func (p *EvidencePublisher) PublishVolume(ctx context.Context, event evidence.VolumeSampledEvent) *problem.Problem {
	if p == nil || p.js == nil {
		return problem.New(problem.Unavailable, "evidence publisher is unavailable")
	}

	spec := p.registry.VolumeSampled
	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		event.Volume.Source,
		event.Volume.Symbol,
		event.Volume.Timeframe,
	)

	data, prob := encodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := "vol:" + event.Volume.Source + ":" +
		event.Volume.Symbol + ":" +
		strconv.Itoa(event.Volume.Timeframe) + ":" +
		strconv.FormatInt(event.Volume.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish evidence volume")
	}

	return nil
}

func (p *EvidencePublisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
