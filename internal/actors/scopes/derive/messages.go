package derive

import (
	"time"

	"internal/application/ingest"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/insights"
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

// publishVolumeProfileMessage is sent from the volume profile sampler
// actor to the insights publisher actor (PROGRAM-0005 / H-8.a).
type publishVolumeProfileMessage struct {
	Event insights.VolumeProfileSampledEvent
}

// publishTPOProfileMessage is sent from the TPO sampler actor to the
// insights publisher actor (PROGRAM-0005 / H-8.b).
type publishTPOProfileMessage struct {
	Event insights.TPOProfileSampledEvent
}

// publishCrossVenueMessage is sent from the cross-venue fusion actor to
// the insights publisher actor (PROGRAM-0005 / H-8.c).
type publishCrossVenueMessage struct {
	Event insights.CrossVenueSampledEvent
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
	Symbol         string
	SignalType     string
	SignalValue    string
	SignalMetadata map[string]string // signal-specific metadata (e.g., bandwidth, sma for bollinger)
	Timeframe      int
	Timestamp      time.Time
	CorrelationID  string
	CausationID    string // event ID of the signal that caused this fan-out
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
	DecisionSeverity   string
	DecisionRationale  string
	Timeframe          int
	Timestamp          time.Time
	CorrelationID      string
	CausationID        string // event ID of the decision that caused this fan-out
}

// publishStrategyMessage is sent from strategy resolver actors to the strategy publisher actor.
type publishStrategyMessage struct {
	Event strategy.StrategyResolvedEvent
}

// strategyResolvedMessage is sent from the SourceScopeActor to risk evaluator actors
// when a strategy is resolved. Contains primitive data per domain isolation (no strategy.Strategy struct).
// DecisionSeverity and DecisionRationale carry the originating decision's semantic depth
// forward so risk can incorporate decision context into rationale and consistency checks.
type strategyResolvedMessage struct {
	Symbol             string
	StrategyType       string
	StrategyDirection  string
	StrategyConfidence string
	DecisionSeverity   string
	DecisionRationale  string
	Timeframe          int
	Timestamp          time.Time
	CorrelationID      string
	CausationID        string // event ID of the strategy that caused this fan-out
}

// publishRiskMessage is sent from risk evaluator actors to the risk publisher actor.
type publishRiskMessage struct {
	Event risk.RiskAssessedEvent
}

// riskAssessedMessage is sent from risk evaluator actors to the SourceScopeActor,
// which fans it out to execution evaluator actors for the matching symbol.
// Contains primitive data per domain isolation (no risk.RiskAssessment struct).
// S265: StrategyType added to preserve strategy family identity across the boundary.
// DecisionSeverity carries the originating decision's severity for downstream traceability.
type riskAssessedMessage struct {
	Symbol             string
	RiskType           string
	RiskDisposition    string
	RiskConfidence     string
	MaxPositionPct     string
	StrategyDirection  string
	StrategyConfidence string
	StrategyType       string
	DecisionSeverity   string
	Timeframe          int
	Timestamp          time.Time
	CorrelationID      string
	CausationID        string // event ID of the risk assessment that caused this fan-out
}

// publishExecutionMessage is sent from execution evaluator actors to the execution publisher actor.
type publishExecutionMessage struct {
	Event execution.PaperOrderSubmittedEvent
}
