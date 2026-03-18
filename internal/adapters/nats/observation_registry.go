package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ObservationRegistry defines the NATS subject and stream contracts for the observation domain.
type ObservationRegistry struct {
	TradeReceived EventSpec
}

func DefaultObservationRegistry() ObservationRegistry {
	eventStream := StreamSpec{
		Name:     "OBSERVATION_EVENTS",
		Subjects: []string{"observation.events.market.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   6 * time.Hour,
		MaxBytes: 1 * 1024 * 1024 * 1024, // 1 GB
	}

	return ObservationRegistry{
		TradeReceived: EventSpec{
			Subject: "observation.events.market.trade",
			Type:    "observation.events.v1.trade_received",
			Stream:  eventStream,
		},
	}
}

// DeriveObservationConsumer defines the durable consumer spec for derive consuming observation trades.
func DeriveObservationConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "derive-observation",
		Event: EventSpec{
			Subject: "observation.events.market.trade.>",
			Type:    "observation.events.v1.trade_received",
			Stream: StreamSpec{
				Name: "OBSERVATION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
