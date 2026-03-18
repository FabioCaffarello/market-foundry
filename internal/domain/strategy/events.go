package strategy

import "internal/shared/events"

const (
	EventStrategyResolved events.Name = "strategy_resolved"
)

// StrategyResolvedEvent is emitted by derive when a strategy is resolved from decisions.
type StrategyResolvedEvent struct {
	Metadata events.Metadata `json:"metadata"`
	Strategy Strategy        `json:"strategy"`
}

func (e StrategyResolvedEvent) EventName() events.Name        { return EventStrategyResolved }
func (e StrategyResolvedEvent) EventMetadata() events.Metadata { return e.Metadata }
