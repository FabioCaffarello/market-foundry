package store

import (
	"context"

	adapternats "internal/adapters/nats"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/problem"
)

// candleProjectionStore is the write interface used by CandleProjectionActor.
// Satisfied by *adapternats.CandleKVStore; enables unit testing without NATS.
type candleProjectionStore interface {
	Put(ctx context.Context, candle evidence.EvidenceCandle) (adapternats.PutResult, *problem.Problem)
	PutHistory(ctx context.Context, candle evidence.EvidenceCandle) *problem.Problem
}

// tradeBurstProjectionStore is the write interface used by TradeBurstProjectionActor.
type tradeBurstProjectionStore interface {
	Put(ctx context.Context, burst evidence.EvidenceTradeBurst) (adapternats.PutResult, *problem.Problem)
}

// volumeProjectionStore is the write interface used by VolumeProjectionActor.
type volumeProjectionStore interface {
	Put(ctx context.Context, vol evidence.EvidenceVolume) (adapternats.PutResult, *problem.Problem)
}

// signalProjectionStore is the write interface used by SignalProjectionActor.
type signalProjectionStore interface {
	Put(ctx context.Context, sig signal.Signal) (adapternats.PutResult, *problem.Problem)
}

// decisionProjectionStore is the write interface used by DecisionProjectionActor.
type decisionProjectionStore interface {
	Put(ctx context.Context, dec decision.Decision) (adapternats.PutResult, *problem.Problem)
}

// strategyProjectionStore is the write interface used by StrategyProjectionActor.
type strategyProjectionStore interface {
	Put(ctx context.Context, strat strategy.Strategy) (adapternats.PutResult, *problem.Problem)
}
