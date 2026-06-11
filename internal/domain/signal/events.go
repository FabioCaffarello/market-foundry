package signal

import "internal/shared/events"

const (
	EventSignalGenerated events.Name = "signal_generated"
)

// SignalGeneratedEvent is emitted by derive when a signal is computed from evidence.
type SignalGeneratedEvent struct {
	Metadata events.Metadata `json:"metadata"`
	Signal   Signal          `json:"signal"`
}

func (e SignalGeneratedEvent) EventName() events.Name         { return EventSignalGenerated }
func (e SignalGeneratedEvent) EventMetadata() events.Metadata { return e.Metadata }
