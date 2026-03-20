package natsevidence

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

type Registry struct {
	CandleSampled     natskit.EventSpec
	CandleLatest      natskit.ControlSpec
	CandleHistory     natskit.ControlSpec
	TradeBurstSampled natskit.EventSpec
	TradeBurstLatest  natskit.ControlSpec
	VolumeSampled     natskit.EventSpec
	VolumeLatest      natskit.ControlSpec
}

func StoreCandleConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-candle", "evidence.events.candle.sampled.>", "evidence.events.v1.candle_sampled", "EVIDENCE_EVENTS")
}

func StoreTradeBurstConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-trade-burst", "evidence.events.tradeburst.sampled.>", "evidence.events.v1.trade_burst_sampled", "EVIDENCE_EVENTS")
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "EVIDENCE_EVENTS",
		Subjects: []string{"evidence.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		CandleSampled: natskit.EventSpec{
			Subject: "evidence.events.candle.sampled",
			Type:    "evidence.events.v1.candle_sampled",
			Stream:  eventStream,
		},
		CandleLatest: natskit.ControlSpec{
			Subject:     "evidence.query.candle.latest",
			RequestType: "evidence.query.v1.candle_latest_request",
			ReplyType:   "evidence.query.v1.candle_latest_reply",
			QueueGroup:  "evidence.query",
		},
		CandleHistory: natskit.ControlSpec{
			Subject:     "evidence.query.candle.history",
			RequestType: "evidence.query.v1.candle_history_request",
			ReplyType:   "evidence.query.v1.candle_history_reply",
			QueueGroup:  "evidence.query",
		},
		TradeBurstSampled: natskit.EventSpec{
			Subject: "evidence.events.tradeburst.sampled",
			Type:    "evidence.events.v1.trade_burst_sampled",
			Stream:  eventStream,
		},
		TradeBurstLatest: natskit.ControlSpec{
			Subject:     "evidence.query.tradeburst.latest",
			RequestType: "evidence.query.v1.trade_burst_latest_request",
			ReplyType:   "evidence.query.v1.trade_burst_latest_reply",
			QueueGroup:  "evidence.query",
		},
		VolumeSampled: natskit.EventSpec{
			Subject: "evidence.events.volume.sampled",
			Type:    "evidence.events.v1.volume_sampled",
			Stream:  eventStream,
		},
		VolumeLatest: natskit.ControlSpec{
			Subject:     "evidence.query.volume.latest",
			RequestType: "evidence.query.v1.volume_latest_request",
			ReplyType:   "evidence.query.v1.volume_latest_reply",
			QueueGroup:  "evidence.query",
		},
	}
}

func WriterCandleConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("writer-candle", "evidence.events.candle.sampled.>", "evidence.events.v1.candle_sampled", "EVIDENCE_EVENTS")
}

func StoreVolumeConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-volume", "evidence.events.volume.sampled.>", "evidence.events.v1.volume_sampled", "EVIDENCE_EVENTS")
}
