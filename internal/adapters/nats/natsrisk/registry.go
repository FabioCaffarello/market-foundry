package natsrisk

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the risk domain.
type Registry struct {
	PositionExposureAssessed natskit.EventSpec
	PositionExposureLatest   natskit.ControlSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "RISK_EVENTS",
		Subjects: []string{"risk.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		PositionExposureAssessed: natskit.EventSpec{
			Subject: "risk.events.position_exposure.assessed",
			Type:    "risk.events.v1.position_exposure_assessed",
			Stream:  eventStream,
		},
		PositionExposureLatest: natskit.ControlSpec{
			Subject:     "risk.query.position_exposure.latest",
			RequestType: "risk.query.v1.position_exposure_latest_request",
			ReplyType:   "risk.query.v1.position_exposure_latest_reply",
			QueueGroup:  "risk.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the risk type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(riskType string) (natskit.ControlSpec, bool) {
	switch riskType {
	case "position_exposure":
		return r.PositionExposureLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// ── Writer Consumer Specs (manual:owned) ─────────────────────────
// Ownership: human-maintained. Not codegen-governed.

// WriterPositionExposureRiskConsumer defines the durable consumer spec for writer consuming position exposure risk events.
func WriterPositionExposureRiskConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("writer-risk-position-exposure", "risk.events.position_exposure.assessed.>", "risk.events.v1.position_exposure_assessed", "RISK_EVENTS")
}

// StorePositionExposureRiskConsumer defines the durable consumer spec for store consuming position exposure risk events.
func StorePositionExposureRiskConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-risk-position-exposure", "risk.events.position_exposure.assessed.>", "risk.events.v1.position_exposure_assessed", "RISK_EVENTS")
}
