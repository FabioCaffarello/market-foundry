package nats

import "time"

// newConsumerSpec creates a ConsumerSpec with standard defaults (30s AckWait, 5 MaxDeliver).
// This factory eliminates duplication across the 20+ consumer spec functions while
// preserving their exact durable names, subjects, and stream bindings.
func newConsumerSpec(durable, subject, eventType, stream string) ConsumerSpec {
	return ConsumerSpec{
		Durable: durable,
		Event: EventSpec{
			Subject: subject,
			Type:    eventType,
			Stream: StreamSpec{
				Name: stream,
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
