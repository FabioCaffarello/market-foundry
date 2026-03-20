package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// DecisionRegistry defines the NATS subject and stream contracts for the decision domain.
type DecisionRegistry struct {
	RSIOversoldEvaluated EventSpec
	RSIOversoldLatest    ControlSpec
}

func DefaultDecisionRegistry() DecisionRegistry {
	eventStream := StreamSpec{
		Name:     "DECISION_EVENTS",
		Subjects: []string{"decision.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	return DecisionRegistry{
		RSIOversoldEvaluated: EventSpec{
			Subject: "decision.events.rsi_oversold.evaluated",
			Type:    "decision.events.v1.rsi_oversold_evaluated",
			Stream:  eventStream,
		},
		RSIOversoldLatest: ControlSpec{
			Subject:     "decision.query.rsi_oversold.latest",
			RequestType: "decision.query.v1.rsi_oversold_latest_request",
			ReplyType:   "decision.query.v1.rsi_oversold_latest_reply",
			QueueGroup:  "decision.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the decision type's latest query.
// Returns false if the type is not registered.
func (r DecisionRegistry) LatestSpecByType(decisionType string) (ControlSpec, bool) {
	switch decisionType {
	case "rsi_oversold":
		return r.RSIOversoldLatest, true
	default:
		return ControlSpec{}, false
	}
}

// ── Writer Consumer Specs (manual:owned) ─────────────────────────
// Ownership: human-maintained. Not codegen-governed.

// WriterRSIOversoldDecisionConsumer defines the durable consumer spec for writer consuming RSI oversold decision events.
func WriterRSIOversoldDecisionConsumer() ConsumerSpec {
	return newConsumerSpec("writer-decision-rsi-oversold", "decision.events.rsi_oversold.evaluated.>", "decision.events.v1.rsi_oversold_evaluated", "DECISION_EVENTS")
}

// StoreRSIOversoldDecisionConsumer defines the durable consumer spec for store consuming RSI oversold decision events.
func StoreRSIOversoldDecisionConsumer() ConsumerSpec {
	return newConsumerSpec("store-decision-rsi-oversold", "decision.events.rsi_oversold.evaluated.>", "decision.events.v1.rsi_oversold_evaluated", "DECISION_EVENTS")
}
