package natsevidence

import (
	"context"
	"fmt"
	"strconv"

	"internal/adapters/nats/natskit"
	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

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

	if _, err := js.CreateOrUpdateStream(ctx, p.registry.CandleSampled.Stream.Config()); err != nil {
		nc.Close()
		return fmt.Errorf("ensure evidence stream: %w", err)
	}

	p.nc = nc
	p.js = js
	return nil
}

func (p *Publisher) PublishCandle(ctx context.Context, event evidence.CandleSampledEvent) *problem.Problem {
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

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := event.Candle.Source + ":" +
		event.Candle.Symbol + ":" +
		strconv.Itoa(event.Candle.Timeframe) + ":" +
		strconv.FormatInt(event.Candle.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish evidence candle")
	}

	return nil
}

func (p *Publisher) PublishTradeBurst(ctx context.Context, event evidence.TradeBurstSampledEvent) *problem.Problem {
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

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		return prob
	}

	dedupKey := "burst:" + event.TradeBurst.Source + ":" +
		event.TradeBurst.Symbol + ":" +
		strconv.Itoa(event.TradeBurst.Timeframe) + ":" +
		strconv.FormatInt(event.TradeBurst.OpenTime.Unix(), 10)

	if _, err := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(dedupKey)); err != nil {
		return problem.Wrap(err, problem.Unavailable, "publish evidence trade burst")
	}

	return nil
}

func (p *Publisher) PublishVolume(ctx context.Context, event evidence.VolumeSampledEvent) *problem.Problem {
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

	data, prob := natskit.EncodeEvent(spec, p.source, event, event.Metadata.CorrelationID, event.Metadata.CausationID)
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

func (p *Publisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}
