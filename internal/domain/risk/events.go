package risk

import "internal/shared/events"

const (
	EventRiskAssessed events.Name = "risk_assessed"
)

// RiskAssessedEvent is emitted by derive when a risk assessment is completed from strategies.
type RiskAssessedEvent struct {
	Metadata       events.Metadata `json:"metadata"`
	RiskAssessment RiskAssessment  `json:"risk_assessment"`
}

func (e RiskAssessedEvent) EventName() events.Name        { return EventRiskAssessed }
func (e RiskAssessedEvent) EventMetadata() events.Metadata { return e.Metadata }
