package natskit

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ControlSpec defines a request/reply control specification.
type ControlSpec struct {
	Subject     string
	RequestType string
	ReplyType   string
	QueueGroup  string
}

// StreamSpec defines a JetStream stream configuration.
type StreamSpec struct {
	Name     string
	Subjects []string
	Storage  jetstream.StorageType
	MaxAge   time.Duration
	MaxBytes int64
}

// Config returns the JetStream stream configuration.
func (s StreamSpec) Config() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:       s.Name,
		Subjects:   append([]string(nil), s.Subjects...),
		Storage:    s.Storage,
		MaxAge:     s.MaxAge,
		MaxBytes:   s.MaxBytes,
		MaxMsgSize: 10 * 1024 * 1024,
	}
}

// EventSpec defines an event type specification.
type EventSpec struct {
	Subject string
	Type    string
	Stream  StreamSpec
}

// ConsumerSpec defines a consumer specification.
type ConsumerSpec struct {
	Durable    string
	Event      EventSpec
	AckWait    time.Duration
	MaxDeliver int
	// FilterSubjects enables multi-subject filtering on the consumer.
	// S401: When set, these subjects replace the single Event.Subject in the
	// NATS ConsumerConfig. Used for segment-scoped consumers that subscribe
	// to multiple source-specific subjects (e.g., binancef + binances).
	// When empty, the consumer falls back to Event.Subject (single filter).
	FilterSubjects []string
}

// PutResult describes the outcome of a projection write to the latest bucket.
type PutResult int

const (
	// PutWritten means the value was materialized (new or newer than existing).
	PutWritten PutResult = iota
	// PutSkippedStale means an existing value has a strictly newer timestamp.
	PutSkippedStale
	// PutSkippedDuplicate means the existing value has the same timestamp.
	PutSkippedDuplicate
)

func (r PutResult) String() string {
	switch r {
	case PutWritten:
		return "written"
	case PutSkippedStale:
		return "skipped_stale"
	case PutSkippedDuplicate:
		return "skipped_duplicate"
	default:
		return "unknown"
	}
}
