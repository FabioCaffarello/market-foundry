package natssignal

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the signal domain.
type Registry struct {
	RSIGenerated          natskit.EventSpec
	RSILatest             natskit.ControlSpec
	EMACrossoverGenerated natskit.EventSpec
	EMACrossoverLatest    natskit.ControlSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "SIGNAL_EVENTS",
		Subjects: []string{"signal.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		RSIGenerated: natskit.EventSpec{
			Subject: "signal.events.rsi.generated",
			Type:    "signal.events.v1.rsi_generated",
			Stream:  eventStream,
		},
		RSILatest: natskit.ControlSpec{
			Subject:     "signal.query.rsi.latest",
			RequestType: "signal.query.v1.rsi_latest_request",
			ReplyType:   "signal.query.v1.rsi_latest_reply",
			QueueGroup:  "signal.query",
		},
		EMACrossoverGenerated: natskit.EventSpec{
			Subject: "signal.events.ema_crossover.generated",
			Type:    "signal.events.v1.ema_crossover_generated",
			Stream:  eventStream,
		},
		EMACrossoverLatest: natskit.ControlSpec{
			Subject:     "signal.query.ema_crossover.latest",
			RequestType: "signal.query.v1.ema_crossover_latest_request",
			ReplyType:   "signal.query.v1.ema_crossover_latest_reply",
			QueueGroup:  "signal.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the signal type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(signalType string) (natskit.ControlSpec, bool) {
	switch signalType {
	case "rsi":
		return r.RSILatest, true
	case "ema_crossover":
		return r.EMACrossoverLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// ── Writer Consumer Specs ─────────────────────────────────────────
// RSI and EMA are codegen-governed (markers below).
// Store consumer specs remain manual:owned.

// codegen:begin consumer_spec family=rsi source=codegen/families/rsi.yaml
// WriterRSISignalConsumer defines the durable consumer spec for writer consuming RSI signal events.
func WriterRSISignalConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-signal-rsi",
		Event: natskit.EventSpec{
			Subject: "signal.events.rsi.generated.>",
			Type:    "signal.events.v1.rsi_generated",
			Stream: natskit.StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=rsi

// codegen:begin consumer_spec family=ema source=codegen/families/ema.yaml
// WriterEMASignalConsumer defines the durable consumer spec for writer consuming ema signal events.
func WriterEMASignalConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-signal-ema",
		Event: natskit.EventSpec{
			Subject: "signal.events.ema.generated.>",
			Type:    "signal.events.v1.ema_generated",
			Stream: natskit.StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=ema

// StoreRSISignalConsumer defines the durable consumer spec for store consuming RSI signal events.
func StoreRSISignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-rsi", "signal.events.rsi.generated.>", "signal.events.v1.rsi_generated", "SIGNAL_EVENTS")
}

// StoreEMACrossoverSignalConsumer defines the durable consumer spec for store consuming EMA crossover signal events.
func StoreEMACrossoverSignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-ema-crossover", "signal.events.ema_crossover.generated.>", "signal.events.v1.ema_crossover_generated", "SIGNAL_EVENTS")
}
