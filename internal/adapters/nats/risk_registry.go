package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// RiskRegistry defines the NATS subject and stream contracts for the risk domain.
type RiskRegistry struct {
	PositionExposureAssessed EventSpec
	PositionExposureLatest   ControlSpec
}

func DefaultRiskRegistry() RiskRegistry {
	eventStream := StreamSpec{
		Name:     "RISK_EVENTS",
		Subjects: []string{"risk.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	return RiskRegistry{
		PositionExposureAssessed: EventSpec{
			Subject: "risk.events.position_exposure.assessed",
			Type:    "risk.events.v1.position_exposure_assessed",
			Stream:  eventStream,
		},
		PositionExposureLatest: ControlSpec{
			Subject:     "risk.query.position_exposure.latest",
			RequestType: "risk.query.v1.position_exposure_latest_request",
			ReplyType:   "risk.query.v1.position_exposure_latest_reply",
			QueueGroup:  "risk.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the risk type's latest query.
// Returns false if the type is not registered.
func (r RiskRegistry) LatestSpecByType(riskType string) (ControlSpec, bool) {
	switch riskType {
	case "position_exposure":
		return r.PositionExposureLatest, true
	default:
		return ControlSpec{}, false
	}
}

// StorePositionExposureRiskConsumer defines the durable consumer spec for store consuming position exposure risk events.
func StorePositionExposureRiskConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-risk-position-exposure",
		Event: EventSpec{
			Subject: "risk.events.position_exposure.assessed.>",
			Type:    "risk.events.v1.position_exposure_assessed",
			Stream: StreamSpec{
				Name: "RISK_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
