package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// EvidenceRegistry defines the NATS subject and stream contracts for the evidence domain.
type EvidenceRegistry struct {
	CandleSampled     EventSpec
	CandleLatest      ControlSpec
	CandleHistory     ControlSpec
	TradeBurstSampled EventSpec
	TradeBurstLatest  ControlSpec
	VolumeSampled     EventSpec
	VolumeLatest      ControlSpec
}

// StoreCandleConsumer defines the durable consumer spec for store consuming candle events.
// Note: durable name "store-candle" is the canonical name for this consumer.
func StoreCandleConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-candle",
		Event: EventSpec{
			Subject: "evidence.events.candle.sampled.>",
			Type:    "evidence.events.v1.candle_sampled",
			Stream: StreamSpec{
				Name: "EVIDENCE_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// StoreTradeBurstConsumer defines the durable consumer spec for store consuming trade burst events.
func StoreTradeBurstConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-trade-burst",
		Event: EventSpec{
			Subject: "evidence.events.tradeburst.sampled.>",
			Type:    "evidence.events.v1.trade_burst_sampled",
			Stream: StreamSpec{
				Name: "EVIDENCE_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

func DefaultEvidenceRegistry() EvidenceRegistry {
	eventStream := StreamSpec{
		Name:     "EVIDENCE_EVENTS",
		Subjects: []string{"evidence.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	return EvidenceRegistry{
		CandleSampled: EventSpec{
			Subject: "evidence.events.candle.sampled",
			Type:    "evidence.events.v1.candle_sampled",
			Stream:  eventStream,
		},
		CandleLatest: ControlSpec{
			Subject:     "evidence.query.candle.latest",
			RequestType: "evidence.query.v1.candle_latest_request",
			ReplyType:   "evidence.query.v1.candle_latest_reply",
			QueueGroup:  "evidence.query",
		},
		CandleHistory: ControlSpec{
			Subject:     "evidence.query.candle.history",
			RequestType: "evidence.query.v1.candle_history_request",
			ReplyType:   "evidence.query.v1.candle_history_reply",
			QueueGroup:  "evidence.query",
		},
		TradeBurstSampled: EventSpec{
			Subject: "evidence.events.tradeburst.sampled",
			Type:    "evidence.events.v1.trade_burst_sampled",
			Stream:  eventStream,
		},
		TradeBurstLatest: ControlSpec{
			Subject:     "evidence.query.tradeburst.latest",
			RequestType: "evidence.query.v1.trade_burst_latest_request",
			ReplyType:   "evidence.query.v1.trade_burst_latest_reply",
			QueueGroup:  "evidence.query",
		},
		VolumeSampled: EventSpec{
			Subject: "evidence.events.volume.sampled",
			Type:    "evidence.events.v1.volume_sampled",
			Stream:  eventStream,
		},
		VolumeLatest: ControlSpec{
			Subject:     "evidence.query.volume.latest",
			RequestType: "evidence.query.v1.volume_latest_request",
			ReplyType:   "evidence.query.v1.volume_latest_reply",
			QueueGroup:  "evidence.query",
		},
	}
}

// StoreVolumeConsumer defines the durable consumer spec for store consuming volume events.
func StoreVolumeConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-volume",
		Event: EventSpec{
			Subject: "evidence.events.volume.sampled.>",
			Type:    "evidence.events.v1.volume_sampled",
			Stream: StreamSpec{
				Name: "EVIDENCE_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
