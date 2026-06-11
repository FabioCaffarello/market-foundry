package decision

import "internal/shared/events"

const (
	EventDecisionEvaluated events.Name = "decision_evaluated"
)

// DecisionEvaluatedEvent is emitted by derive when a decision is evaluated from signals.
type DecisionEvaluatedEvent struct {
	Metadata events.Metadata `json:"metadata"`
	Decision Decision        `json:"decision"`
}

func (e DecisionEvaluatedEvent) EventName() events.Name         { return EventDecisionEvaluated }
func (e DecisionEvaluatedEvent) EventMetadata() events.Metadata { return e.Metadata }
