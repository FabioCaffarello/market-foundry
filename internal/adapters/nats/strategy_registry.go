package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// StrategyRegistry defines the NATS subject and stream contracts for the strategy domain.
type StrategyRegistry struct {
	MeanReversionEntryResolved EventSpec
	MeanReversionEntryLatest   ControlSpec
}

func DefaultStrategyRegistry() StrategyRegistry {
	eventStream := StreamSpec{
		Name:     "STRATEGY_EVENTS",
		Subjects: []string{"strategy.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	return StrategyRegistry{
		MeanReversionEntryResolved: EventSpec{
			Subject: "strategy.events.mean_reversion_entry.resolved",
			Type:    "strategy.events.v1.mean_reversion_entry_resolved",
			Stream:  eventStream,
		},
		MeanReversionEntryLatest: ControlSpec{
			Subject:     "strategy.query.mean_reversion_entry.latest",
			RequestType: "strategy.query.v1.mean_reversion_entry_latest_request",
			ReplyType:   "strategy.query.v1.mean_reversion_entry_latest_reply",
			QueueGroup:  "strategy.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the strategy type's latest query.
// Returns false if the type is not registered.
func (r StrategyRegistry) LatestSpecByType(strategyType string) (ControlSpec, bool) {
	switch strategyType {
	case "mean_reversion_entry":
		return r.MeanReversionEntryLatest, true
	default:
		return ControlSpec{}, false
	}
}

// ── Writer Consumer Specs (manual:owned) ─────────────────────────
// Ownership: human-maintained. Not codegen-governed.

// WriterMeanReversionEntryStrategyConsumer defines the durable consumer spec for writer consuming mean reversion entry strategy events.
func WriterMeanReversionEntryStrategyConsumer() ConsumerSpec {
	return newConsumerSpec("writer-strategy-mean-reversion-entry", "strategy.events.mean_reversion_entry.resolved.>", "strategy.events.v1.mean_reversion_entry_resolved", "STRATEGY_EVENTS")
}

// StoreMeanReversionEntryStrategyConsumer defines the durable consumer spec for store consuming mean reversion entry strategy events.
func StoreMeanReversionEntryStrategyConsumer() ConsumerSpec {
	return newConsumerSpec("store-strategy-mean-reversion-entry", "strategy.events.mean_reversion_entry.resolved.>", "strategy.events.v1.mean_reversion_entry_resolved", "STRATEGY_EVENTS")
}
