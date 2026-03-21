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
	BollingerGenerated    natskit.EventSpec
	BollingerLatest       natskit.ControlSpec
	MACDGenerated         natskit.EventSpec
	MACDLatest            natskit.ControlSpec
	VWAPGenerated         natskit.EventSpec
	VWAPLatest            natskit.ControlSpec
	ATRGenerated          natskit.EventSpec
	ATRLatest             natskit.ControlSpec
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
		BollingerGenerated: natskit.EventSpec{
			Subject: "signal.events.bollinger.generated",
			Type:    "signal.events.v1.bollinger_generated",
			Stream:  eventStream,
		},
		BollingerLatest: natskit.ControlSpec{
			Subject:     "signal.query.bollinger.latest",
			RequestType: "signal.query.v1.bollinger_latest_request",
			ReplyType:   "signal.query.v1.bollinger_latest_reply",
			QueueGroup:  "signal.query",
		},
		MACDGenerated: natskit.EventSpec{
			Subject: "signal.events.macd.generated",
			Type:    "signal.events.v1.macd_generated",
			Stream:  eventStream,
		},
		MACDLatest: natskit.ControlSpec{
			Subject:     "signal.query.macd.latest",
			RequestType: "signal.query.v1.macd_latest_request",
			ReplyType:   "signal.query.v1.macd_latest_reply",
			QueueGroup:  "signal.query",
		},
		VWAPGenerated: natskit.EventSpec{
			Subject: "signal.events.vwap.generated",
			Type:    "signal.events.v1.vwap_generated",
			Stream:  eventStream,
		},
		VWAPLatest: natskit.ControlSpec{
			Subject:     "signal.query.vwap.latest",
			RequestType: "signal.query.v1.vwap_latest_request",
			ReplyType:   "signal.query.v1.vwap_latest_reply",
			QueueGroup:  "signal.query",
		},
		ATRGenerated: natskit.EventSpec{
			Subject: "signal.events.atr.generated",
			Type:    "signal.events.v1.atr_generated",
			Stream:  eventStream,
		},
		ATRLatest: natskit.ControlSpec{
			Subject:     "signal.query.atr.latest",
			RequestType: "signal.query.v1.atr_latest_request",
			ReplyType:   "signal.query.v1.atr_latest_reply",
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
	case "bollinger":
		return r.BollingerLatest, true
	case "macd":
		return r.MACDLatest, true
	case "vwap":
		return r.VWAPLatest, true
	case "atr":
		return r.ATRLatest, true
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

// codegen:begin consumer_spec family=bollinger source=codegen/families/bollinger.yaml
// WriterBollingerSignalConsumer defines the durable consumer spec for writer consuming
// bollinger signal events.
func WriterBollingerSignalConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-signal-bollinger",
		Event: natskit.EventSpec{
			Subject: "signal.events.bollinger.generated.>",
			Type:    "signal.events.v1.bollinger_generated",
			Stream: natskit.StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=bollinger

// codegen:begin consumer_spec family=macd source=codegen/families/macd.yaml
// WriterMACDSignalConsumer defines the durable consumer spec for writer consuming
// macd signal events.
func WriterMACDSignalConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-signal-macd",
		Event: natskit.EventSpec{
			Subject: "signal.events.macd.generated.>",
			Type:    "signal.events.v1.macd_generated",
			Stream: natskit.StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=macd

// codegen:begin consumer_spec family=vwap source=codegen/families/vwap.yaml
// WriterVWAPSignalConsumer defines the durable consumer spec for writer consuming
// vwap signal events.
func WriterVWAPSignalConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-signal-vwap",
		Event: natskit.EventSpec{
			Subject: "signal.events.vwap.generated.>",
			Type:    "signal.events.v1.vwap_generated",
			Stream: natskit.StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=vwap

// codegen:begin consumer_spec family=atr source=codegen/families/atr.yaml
// WriterATRSignalConsumer defines the durable consumer spec for writer consuming
// atr signal events.
func WriterATRSignalConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-signal-atr",
		Event: natskit.EventSpec{
			Subject: "signal.events.atr.generated.>",
			Type:    "signal.events.v1.atr_generated",
			Stream: natskit.StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=atr

// StoreATRSignalConsumer defines the durable consumer spec for store consuming ATR signal events.
func StoreATRSignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-atr", "signal.events.atr.generated.>", "signal.events.v1.atr_generated", "SIGNAL_EVENTS")
}

// StoreVWAPSignalConsumer defines the durable consumer spec for store consuming VWAP signal events.
func StoreVWAPSignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-vwap", "signal.events.vwap.generated.>", "signal.events.v1.vwap_generated", "SIGNAL_EVENTS")
}

// StoreMACDSignalConsumer defines the durable consumer spec for store consuming MACD signal events.
func StoreMACDSignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-macd", "signal.events.macd.generated.>", "signal.events.v1.macd_generated", "SIGNAL_EVENTS")
}

// StoreRSISignalConsumer defines the durable consumer spec for store consuming RSI signal events.
func StoreRSISignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-rsi", "signal.events.rsi.generated.>", "signal.events.v1.rsi_generated", "SIGNAL_EVENTS")
}

// StoreEMACrossoverSignalConsumer defines the durable consumer spec for store consuming EMA crossover signal events.
func StoreEMACrossoverSignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-ema-crossover", "signal.events.ema_crossover.generated.>", "signal.events.v1.ema_crossover_generated", "SIGNAL_EVENTS")
}

// StoreBollingerSignalConsumer defines the durable consumer spec for store consuming Bollinger signal events.
func StoreBollingerSignalConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-signal-bollinger", "signal.events.bollinger.generated.>", "signal.events.v1.bollinger_generated", "SIGNAL_EVENTS")
}
