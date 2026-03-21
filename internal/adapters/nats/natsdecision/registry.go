package natsdecision

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the decision domain.
type Registry struct {
	RSIOversoldEvaluated      natskit.EventSpec
	RSIOversoldLatest         natskit.ControlSpec
	EMACrossoverEvaluated     natskit.EventSpec
	EMACrossoverLatest        natskit.ControlSpec
	BollingerSqueezeEvaluated natskit.EventSpec
	BollingerSqueezeLatest    natskit.ControlSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "DECISION_EVENTS",
		Subjects: []string{"decision.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		RSIOversoldEvaluated: natskit.EventSpec{
			Subject: "decision.events.rsi_oversold.evaluated",
			Type:    "decision.events.v1.rsi_oversold_evaluated",
			Stream:  eventStream,
		},
		RSIOversoldLatest: natskit.ControlSpec{
			Subject:     "decision.query.rsi_oversold.latest",
			RequestType: "decision.query.v1.rsi_oversold_latest_request",
			ReplyType:   "decision.query.v1.rsi_oversold_latest_reply",
			QueueGroup:  "decision.query",
		},
		EMACrossoverEvaluated: natskit.EventSpec{
			Subject: "decision.events.ema_crossover.evaluated",
			Type:    "decision.events.v1.ema_crossover_evaluated",
			Stream:  eventStream,
		},
		EMACrossoverLatest: natskit.ControlSpec{
			Subject:     "decision.query.ema_crossover.latest",
			RequestType: "decision.query.v1.ema_crossover_latest_request",
			ReplyType:   "decision.query.v1.ema_crossover_latest_reply",
			QueueGroup:  "decision.query",
		},
		BollingerSqueezeEvaluated: natskit.EventSpec{
			Subject: "decision.events.bollinger_squeeze.evaluated",
			Type:    "decision.events.v1.bollinger_squeeze_evaluated",
			Stream:  eventStream,
		},
		BollingerSqueezeLatest: natskit.ControlSpec{
			Subject:     "decision.query.bollinger_squeeze.latest",
			RequestType: "decision.query.v1.bollinger_squeeze_latest_request",
			ReplyType:   "decision.query.v1.bollinger_squeeze_latest_reply",
			QueueGroup:  "decision.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the decision type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(decisionType string) (natskit.ControlSpec, bool) {
	switch decisionType {
	case "rsi_oversold":
		return r.RSIOversoldLatest, true
	case "ema_crossover":
		return r.EMACrossoverLatest, true
	case "bollinger_squeeze":
		return r.BollingerSqueezeLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// ── Writer Consumer Specs ─────────────────────────────────────────
// RSI oversold and EMA crossover are codegen-governed (markers below).
// Store consumer specs remain manual:owned.

// codegen:begin consumer_spec family=rsi_oversold source=codegen/families/rsi_oversold.yaml
// WriterRSIOversoldDecisionConsumer defines the durable consumer spec for writer consuming RSI oversold decision events.
func WriterRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-decision-rsi-oversold",
		Event: natskit.EventSpec{
			Subject: "decision.events.rsi_oversold.evaluated.>",
			Type:    "decision.events.v1.rsi_oversold_evaluated",
			Stream: natskit.StreamSpec{
				Name: "DECISION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=rsi_oversold

// codegen:begin consumer_spec family=ema_crossover source=codegen/families/ema_crossover.yaml
// WriterEMACrossoverDecisionConsumer defines the durable consumer spec for writer consuming
// ema_crossover decision events.
func WriterEMACrossoverDecisionConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-decision-ema-crossover",
		Event: natskit.EventSpec{
			Subject: "decision.events.ema_crossover.evaluated.>",
			Type:    "decision.events.v1.ema_crossover_evaluated",
			Stream: natskit.StreamSpec{
				Name: "DECISION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=ema_crossover

// StoreRSIOversoldDecisionConsumer defines the durable consumer spec for store consuming RSI oversold decision events.
func StoreRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-decision-rsi-oversold", "decision.events.rsi_oversold.evaluated.>", "decision.events.v1.rsi_oversold_evaluated", "DECISION_EVENTS")
}

// StoreEMACrossoverDecisionConsumer defines the durable consumer spec for store consuming EMA crossover decision events.
func StoreEMACrossoverDecisionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-decision-ema-crossover", "decision.events.ema_crossover.evaluated.>", "decision.events.v1.ema_crossover_evaluated", "DECISION_EVENTS")
}

// WriterBollingerSqueezeDecisionConsumer defines the durable consumer spec for writer consuming
// bollinger_squeeze decision events.
func WriterBollingerSqueezeDecisionConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-decision-bollinger-squeeze",
		Event: natskit.EventSpec{
			Subject: "decision.events.bollinger_squeeze.evaluated.>",
			Type:    "decision.events.v1.bollinger_squeeze_evaluated",
			Stream: natskit.StreamSpec{
				Name: "DECISION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// StoreBollingerSqueezeDecisionConsumer defines the durable consumer spec for store consuming Bollinger Squeeze decision events.
func StoreBollingerSqueezeDecisionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-decision-bollinger-squeeze", "decision.events.bollinger_squeeze.evaluated.>", "decision.events.v1.bollinger_squeeze_evaluated", "DECISION_EVENTS")
}
