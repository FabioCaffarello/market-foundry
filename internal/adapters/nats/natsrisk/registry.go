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
	DrawdownLimitAssessed    natskit.EventSpec
	DrawdownLimitLatest      natskit.ControlSpec
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
		DrawdownLimitAssessed: natskit.EventSpec{
			Subject: "risk.events.drawdown_limit.assessed",
			Type:    "risk.events.v1.drawdown_limit_assessed",
			Stream:  eventStream,
		},
		DrawdownLimitLatest: natskit.ControlSpec{
			Subject:     "risk.query.drawdown_limit.latest",
			RequestType: "risk.query.v1.drawdown_limit_latest_request",
			ReplyType:   "risk.query.v1.drawdown_limit_latest_reply",
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
	case "drawdown_limit":
		return r.DrawdownLimitLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// ── Writer Consumer Specs ─────────────────────────────────────────
// Position exposure and drawdown limit are codegen-governed (markers below).
// Store consumer specs remain manual:owned.

// codegen:begin consumer_spec family=position_exposure source=codegen/families/position_exposure.yaml
// WriterPositionExposureRiskConsumer defines the durable consumer spec for writer consuming position exposure risk events.
func WriterPositionExposureRiskConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-risk-position-exposure",
		Event: natskit.EventSpec{
			Subject: "risk.events.position_exposure.assessed.>",
			Type:    "risk.events.v1.position_exposure_assessed",
			Stream: natskit.StreamSpec{
				Name: "RISK_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=position_exposure

// codegen:begin consumer_spec family=drawdown_limit source=codegen/families/drawdown_limit.yaml
// WriterDrawdownLimitRiskConsumer defines the durable consumer spec for writer consuming
// drawdown_limit risk events.
func WriterDrawdownLimitRiskConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-risk-drawdown-limit",
		Event: natskit.EventSpec{
			Subject: "risk.events.drawdown_limit.assessed.>",
			Type:    "risk.events.v1.drawdown_limit_assessed",
			Stream: natskit.StreamSpec{
				Name: "RISK_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=drawdown_limit

// StorePositionExposureRiskConsumer defines the durable consumer spec for store consuming position exposure risk events.
func StorePositionExposureRiskConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-risk-position-exposure", "risk.events.position_exposure.assessed.>", "risk.events.v1.position_exposure_assessed", "RISK_EVENTS")
}

// StoreDrawdownLimitRiskConsumer defines the durable consumer spec for store consuming drawdown limit risk events.
func StoreDrawdownLimitRiskConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-risk-drawdown-limit", "risk.events.drawdown_limit.assessed.>", "risk.events.v1.drawdown_limit_assessed", "RISK_EVENTS")
}
