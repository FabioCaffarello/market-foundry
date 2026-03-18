package derive

import (
	"time"

	"internal/application/ingest"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/observation"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// tradeReceivedMessage is sent from the consumer actor to the supervisor for routing.
type tradeReceivedMessage struct {
	Event observation.TradeReceivedEvent
}

// publishCandleMessage is sent from the sampler actor to the publisher actor.
type publishCandleMessage struct {
	Event evidence.CandleSampledEvent
}

// publishTradeBurstMessage is sent from the trade burst sampler actor to the publisher actor.
type publishTradeBurstMessage struct {
	Event evidence.TradeBurstSampledEvent
}

// publishVolumeMessage is sent from the volume sampler actor to the publisher actor.
type publishVolumeMessage struct {
	Event evidence.VolumeSampledEvent
}

// activateSamplerMessage is sent from the binding watcher to the supervisor
// when a new ingestion binding becomes active (at startup or via runtime event).
type activateSamplerMessage struct {
	Target ingest.BindingTarget
}

// candleFinalizedMessage is sent from evidence sampler actors to the SourceScopeActor
// when a candle is finalized. The SourceScopeActor fans it out to signal samplers
// matching the same symbol.
type candleFinalizedMessage struct {
	Symbol        string
	Timeframe     int
	ClosePrice    string
	Timestamp     time.Time
	CorrelationID string
}

// publishSignalMessage is sent from signal sampler actors to the signal publisher actor.
type publishSignalMessage struct {
	Event signal.SignalGeneratedEvent
}

// signalGeneratedMessage is sent from signal sampler actors to the SourceScopeActor,
// which fans it out to decision evaluator actors for the matching symbol.
// Contains primitive data per DBI-9 (no signal.Signal struct).
type signalGeneratedMessage struct {
	Symbol        string
	SignalType    string
	SignalValue   string
	Timeframe     int
	Timestamp     time.Time
	CorrelationID string
}

// publishDecisionMessage is sent from decision evaluator actors to the decision publisher actor.
type publishDecisionMessage struct {
	Event decision.DecisionEvaluatedEvent
}

// decisionEvaluatedMessage is sent from the SourceScopeActor to strategy resolver actors
// when a decision is evaluated. Contains primitive data per DBI-9 (no decision.Decision struct).
type decisionEvaluatedMessage struct {
	Symbol             string
	DecisionType       string
	DecisionOutcome    string
	DecisionConfidence string
	Timeframe          int
	Timestamp          time.Time
	CorrelationID      string
}

// publishStrategyMessage is sent from strategy resolver actors to the strategy publisher actor.
type publishStrategyMessage struct {
	Event strategy.StrategyResolvedEvent
}

// strategyResolvedMessage is sent from the SourceScopeActor to risk evaluator actors
// when a strategy is resolved. Contains primitive data per domain isolation (no strategy.Strategy struct).
type strategyResolvedMessage struct {
	Symbol             string
	StrategyType       string
	StrategyDirection  string
	StrategyConfidence string
	Timeframe          int
	Timestamp          time.Time
	CorrelationID      string
}

// publishRiskMessage is sent from risk evaluator actors to the risk publisher actor.
type publishRiskMessage struct {
	Event risk.RiskAssessedEvent
}
