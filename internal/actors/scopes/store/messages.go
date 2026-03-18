package store

import (
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// candleReceivedMessage is sent from the evidence consumer actor to the projection actor.
type candleReceivedMessage struct {
	Event evidence.CandleSampledEvent
}

// tradeBurstReceivedMessage is sent from the trade burst consumer actor to the projection actor.
type tradeBurstReceivedMessage struct {
	Event evidence.TradeBurstSampledEvent
}

// volumeReceivedMessage is sent from the volume consumer actor to the projection actor.
type volumeReceivedMessage struct {
	Event evidence.VolumeSampledEvent
}

// signalReceivedMessage is sent from the signal consumer actor to the signal projection actor.
type signalReceivedMessage struct {
	Event signal.SignalGeneratedEvent
}

// decisionReceivedMessage is sent from the decision consumer actor to the decision projection actor.
type decisionReceivedMessage struct {
	Event decision.DecisionEvaluatedEvent
}

// strategyReceivedMessage is sent from the strategy consumer actor to the strategy projection actor.
type strategyReceivedMessage struct {
	Event strategy.StrategyResolvedEvent
}

// riskReceivedMessage is sent from the risk consumer actor to the risk projection actor.
type riskReceivedMessage struct {
	Event risk.RiskAssessedEvent
}
