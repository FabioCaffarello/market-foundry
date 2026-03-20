package natsdecision

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the decision domain.
type Registry struct {
	RSIOversoldEvaluated natskit.EventSpec
	RSIOversoldLatest    natskit.ControlSpec
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
	}
}

// LatestSpecByType returns the ControlSpec for the decision type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(decisionType string) (natskit.ControlSpec, bool) {
	switch decisionType {
	case "rsi_oversold":
		return r.RSIOversoldLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// WriterRSIOversoldDecisionConsumer defines the durable consumer spec for writer consuming RSI oversold decision events.
func WriterRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("writer-decision-rsi-oversold", "decision.events.rsi_oversold.evaluated.>", "decision.events.v1.rsi_oversold_evaluated", "DECISION_EVENTS")
}

// StoreRSIOversoldDecisionConsumer defines the durable consumer spec for store consuming RSI oversold decision events.
func StoreRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-decision-rsi-oversold", "decision.events.rsi_oversold.evaluated.>", "decision.events.v1.rsi_oversold_evaluated", "DECISION_EVENTS")
}
