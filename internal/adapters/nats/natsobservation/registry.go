package natsobservation

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the observation domain.
type Registry struct {
	TradeReceived natskit.EventSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "OBSERVATION_EVENTS",
		Subjects: []string{"observation.events.market.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   6 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		TradeReceived: natskit.EventSpec{
			Subject: "observation.events.market.trade",
			Type:    "observation.events.v1.trade_received",
			Stream:  eventStream,
		},
	}
}

// DeriveObservationConsumer defines the durable consumer spec for derive consuming observation trades.
func DeriveObservationConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "derive-observation",
		Event: natskit.EventSpec{
			Subject: "observation.events.market.trade.>",
			Type:    "observation.events.v1.trade_received",
			Stream: natskit.StreamSpec{
				Name: "OBSERVATION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
