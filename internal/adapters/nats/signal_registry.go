package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// SignalRegistry defines the NATS subject and stream contracts for the signal domain.
type SignalRegistry struct {
	RSIGenerated          EventSpec
	RSILatest             ControlSpec
	EMACrossoverGenerated EventSpec
	EMACrossoverLatest    ControlSpec
}

func DefaultSignalRegistry() SignalRegistry {
	eventStream := StreamSpec{
		Name:     "SIGNAL_EVENTS",
		Subjects: []string{"signal.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	return SignalRegistry{
		RSIGenerated: EventSpec{
			Subject: "signal.events.rsi.generated",
			Type:    "signal.events.v1.rsi_generated",
			Stream:  eventStream,
		},
		RSILatest: ControlSpec{
			Subject:     "signal.query.rsi.latest",
			RequestType: "signal.query.v1.rsi_latest_request",
			ReplyType:   "signal.query.v1.rsi_latest_reply",
			QueueGroup:  "signal.query",
		},
		EMACrossoverGenerated: EventSpec{
			Subject: "signal.events.ema_crossover.generated",
			Type:    "signal.events.v1.ema_crossover_generated",
			Stream:  eventStream,
		},
		EMACrossoverLatest: ControlSpec{
			Subject:     "signal.query.ema_crossover.latest",
			RequestType: "signal.query.v1.ema_crossover_latest_request",
			ReplyType:   "signal.query.v1.ema_crossover_latest_reply",
			QueueGroup:  "signal.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the signal type's latest query.
// Returns false if the type is not registered.
func (r SignalRegistry) LatestSpecByType(signalType string) (ControlSpec, bool) {
	switch signalType {
	case "rsi":
		return r.RSILatest, true
	case "ema_crossover":
		return r.EMACrossoverLatest, true
	default:
		return ControlSpec{}, false
	}
}

// StoreRSISignalConsumer defines the durable consumer spec for store consuming RSI signal events.
func StoreRSISignalConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-signal-rsi",
		Event: EventSpec{
			Subject: "signal.events.rsi.generated.>",
			Type:    "signal.events.v1.rsi_generated",
			Stream: StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// StoreEMACrossoverSignalConsumer defines the durable consumer spec for store consuming EMA crossover signal events.
func StoreEMACrossoverSignalConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-signal-ema-crossover",
		Event: EventSpec{
			Subject: "signal.events.ema_crossover.generated.>",
			Type:    "signal.events.v1.ema_crossover_generated",
			Stream: StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
