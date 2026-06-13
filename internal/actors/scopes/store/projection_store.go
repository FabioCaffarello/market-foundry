package store

import (
	"context"

	"internal/adapters/nats/natskit"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/insights"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/problem"
)

// candleProjectionStore is the write interface used by CandleProjectionActor.
// Satisfied by *natsevidence.CandleKVStore; enables unit testing without NATS.
type candleProjectionStore interface {
	Put(ctx context.Context, candle evidence.EvidenceCandle) (natskit.PutResult, *problem.Problem)
	PutHistory(ctx context.Context, candle evidence.EvidenceCandle) *problem.Problem
}

// tradeBurstProjectionStore is the write interface used by TradeBurstProjectionActor.
type tradeBurstProjectionStore interface {
	Put(ctx context.Context, burst evidence.EvidenceTradeBurst) (natskit.PutResult, *problem.Problem)
}

// volumeProjectionStore is the write interface used by VolumeProjectionActor.
type volumeProjectionStore interface {
	Put(ctx context.Context, vol evidence.EvidenceVolume) (natskit.PutResult, *problem.Problem)
}

// volumeProfileProjectionStore is the write interface used by
// VolumeProfileProjectionActor (PROGRAM-0005 / H-8.a).
type volumeProfileProjectionStore interface {
	Put(ctx context.Context, vp insights.VolumeProfile) (natskit.PutResult, *problem.Problem)
}

// tpoProjectionStore is the write interface used by
// TPOProjectionActor (PROGRAM-0005 / H-8.b).
type tpoProjectionStore interface {
	Put(ctx context.Context, tp insights.TPOProfile) (natskit.PutResult, *problem.Problem)
}

// crossVenueProjectionStore is the write interface used by
// CrossVenueProjectionActor (PROGRAM-0005 / H-8.c).
type crossVenueProjectionStore interface {
	Put(ctx context.Context, cv insights.CrossVenueSnapshot) (natskit.PutResult, *problem.Problem)
}

// signalProjectionStore is the write interface used by SignalProjectionActor.
type signalProjectionStore interface {
	Put(ctx context.Context, sig signal.Signal) (natskit.PutResult, *problem.Problem)
}

// decisionProjectionStore is the write interface used by DecisionProjectionActor.
type decisionProjectionStore interface {
	Put(ctx context.Context, dec decision.Decision) (natskit.PutResult, *problem.Problem)
}

// strategyProjectionStore is the write interface used by StrategyProjectionActor.
type strategyProjectionStore interface {
	Put(ctx context.Context, strat strategy.Strategy) (natskit.PutResult, *problem.Problem)
}

// riskProjectionStore is the write interface used by RiskProjectionActor.
type riskProjectionStore interface {
	Put(ctx context.Context, assessment risk.RiskAssessment) (natskit.PutResult, *problem.Problem)
}
